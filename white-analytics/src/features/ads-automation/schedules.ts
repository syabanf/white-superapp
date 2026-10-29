/**
 * Recurring reports. The daily cron calls `runDueReportSchedules(now, clientId)`;
 * each due schedule is claimed atomically (compare-and-set on nextRunAt, so
 * overlapping runs never double-create), turned into a Report over the last full
 * week/month (optionally with an AI summary), and the client audience gets a
 * REPORT_READY notification. A failed run puts nextRunAt back so the next cron
 * retries. Missed periods are not back-filled: one report for the latest period.
 */
import "server-only";
import { db } from "@/lib/db";
import { isProviderError } from "@/lib/action-result";
import { toISODate } from "@/lib/dates";
import { runInDemoMode } from "@/lib/demo-mode";
import { clientAudience, notify } from "@/lib/notify";
import { createReportCore } from "@/features/reports/create-core";
import { computeNextRun, periodLabel, reportPeriod, scheduledReportTitle, type Cadence } from "./lib";
import { aa } from "./strings";

type ScheduleRow = {
  id: string;
  clientId: string;
  name: string;
  cadence: Cadence;
  modules: ("SOCIAL" | "SEO" | "ADS")[];
  withInsight: boolean;
  createdById: string | null;
  client: { slug: string; name: string; isDemo: boolean };
};

const SELECT = {
  id: true,
  clientId: true,
  name: true,
  cadence: true,
  modules: true,
  withInsight: true,
  createdById: true,
  nextRunAt: true,
  lastRunAt: true,
  client: { select: { slug: true, name: true, isDemo: true } },
} as const;

export type ScheduledRun = { scheduleId: string; reportId: string; title: string; withAi: boolean };

/** Build the Report for one schedule at `runAt` and notify. Does not touch nextRunAt. */
export async function generateScheduledReport(s: ScheduleRow, runAt: Date): Promise<ScheduledRun> {
  return runInDemoMode(s.client.isDemo, async () => {
    const period = reportPeriod(s.cadence, runAt);
    const title = scheduledReportTitle(s.name, s.cadence, period);
    const job = await db.syncJob.create({ data: { clientId: s.clientId, kind: "REPORT_SCHEDULE", status: "RUNNING", message: title } });
    try {
      const base = {
        clientId: s.clientId,
        clientName: s.client.name,
        title,
        range: period,
        modules: s.modules.length ? s.modules : (["SOCIAL", "SEO", "ADS"] as const).slice(),
        compare: true,
        language: "id" as const,
        createdById: s.createdById,
      };
      let report: { id: string; aiSummary: string | null };
      let note = "";
      try {
        report = await createReportCore({ ...base, includeAi: s.withInsight });
      } catch (e) {
        // The AI summary is a bonus: an AI provider failure still ships the report.
        if (!s.withInsight || !isProviderError(e)) throw e;
        note = ` (tanpa AI: ${e.code})`;
        report = await createReportCore({ ...base, includeAi: false });
      }
      const href = `/clients/${s.client.slug}/reports?tab=history&created=${report.id}&from=${toISODate(period.from)}&to=${toISODate(period.to)}`;
      await notify(await clientAudience(s.clientId), {
        type: "REPORT_READY",
        title: aa.notif.reportTitle(title),
        body: aa.notif.reportBody(periodLabel(s.cadence, period)),
        href,
        clientId: s.clientId,
      });
      await db.syncJob.update({ where: { id: job.id }, data: { status: "SUCCESS", finishedAt: new Date(), message: `${title}${note}` } });
      return { scheduleId: s.id, reportId: report.id, title, withAi: report.aiSummary != null };
    } catch (e) {
      const message = e instanceof Error ? e.message : String(e);
      await db.syncJob.update({ where: { id: job.id }, data: { status: "FAILED", finishedAt: new Date(), error: message.slice(0, 900) } });
      throw e;
    }
  });
}

export type DueRunResult = { due: number; created: number; failed: number; runs: ScheduledRun[]; errors: string[] };

export async function runDueReportSchedules(now: Date = new Date(), clientId?: string): Promise<DueRunResult> {
  const due = await db.reportSchedule.findMany({
    where: { enabled: true, nextRunAt: { lte: now }, ...(clientId ? { clientId } : {}) },
    select: SELECT,
    orderBy: { nextRunAt: "asc" },
  });
  const out: DueRunResult = { due: due.length, created: 0, failed: 0, runs: [], errors: [] };
  for (const s of due) {
    const claimed = await db.reportSchedule.updateMany({
      where: { id: s.id, enabled: true, nextRunAt: s.nextRunAt },
      data: { nextRunAt: computeNextRun(s.cadence, now), lastRunAt: now },
    });
    if (claimed.count === 0) continue; // another run took it
    try {
      out.runs.push(await generateScheduledReport(s, now));
      out.created++;
    } catch (e) {
      out.failed++;
      out.errors.push(`${s.name}: ${e instanceof Error ? e.message : String(e)}`);
      // put it back so tomorrow's cron retries
      await db.reportSchedule.update({ where: { id: s.id }, data: { nextRunAt: s.nextRunAt, lastRunAt: s.lastRunAt } });
    }
  }
  return out;
}

/** "Jalankan sekarang": build the report immediately; nextRunAt stays as scheduled. */
export async function runScheduleNow(scheduleId: string, now: Date = new Date()): Promise<ScheduledRun | null> {
  const s = await db.reportSchedule.findUnique({ where: { id: scheduleId }, select: SELECT });
  if (!s) return null;
  const run = await generateScheduledReport(s, now);
  await db.reportSchedule.update({ where: { id: s.id }, data: { lastRunAt: now } });
  return run;
}
