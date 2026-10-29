import "server-only";
import { cache } from "react";
import { db } from "@/lib/db";
import { addDays, todayUtc, toISODate } from "@/lib/dates";
import { severityRank, type AnomalySeverity } from "@/lib/metrics/ads-anomaly";

/** Unresolved alerts shown in the panel: the last 14 days. */
export const ALERT_WINDOW_DAYS = 14;

export type AlertItem = {
  id: string;
  rule: string;
  severity: AnomalySeverity;
  campaignId: string;
  campaignName: string | null;
  /** latest day this rule fired for this campaign (YYYY-MM-DD) */
  date: string;
  /** how many days in the window it fired */
  days: number;
  message: string;
};

export type AlertsPanelData = {
  alerts: AlertItem[];
  lastCheckedAt: string | null;
  weeklyInsight: { createdAt: string; isMock: boolean } | null;
};

/**
 * Unresolved alerts collapsed per rule + campaign (latest day wins, `days` counts
 * repeats), worst first. Plus when detection last ran and the latest ADS insight.
 */
export const getAlertsPanel = cache(async (clientId: string): Promise<AlertsPanelData> => {
  const since = addDays(todayUtc(), -ALERT_WINDOW_DAYS);
  const [rows, job, insight] = await Promise.all([
    db.adAlert.findMany({
      where: { clientId, resolvedAt: null, date: { gte: since } },
      orderBy: [{ date: "desc" }, { createdAt: "desc" }],
    }),
    db.syncJob.findFirst({
      where: { clientId, kind: "ADS_ANOMALY", status: "SUCCESS" },
      orderBy: { startedAt: "desc" },
      select: { finishedAt: true, startedAt: true },
    }),
    db.aiInsight.findFirst({ where: { clientId, module: "ADS" }, orderBy: { createdAt: "desc" }, select: { createdAt: true, model: true } }),
  ]);

  const campaignIds = [...new Set(rows.map((r) => r.campaignId).filter(Boolean))];
  const campaigns = campaignIds.length
    ? await db.adCampaign.findMany({ where: { id: { in: campaignIds } }, select: { id: true, name: true } })
    : [];
  const names = new Map(campaigns.map((c) => [c.id, c.name]));

  const grouped = new Map<string, AlertItem>();
  for (const r of rows) {
    const key = `${r.rule}|${r.campaignId}`;
    const hit = grouped.get(key);
    if (hit) {
      hit.days++;
      continue;
    }
    grouped.set(key, {
      id: r.id,
      rule: r.rule,
      severity: r.severity,
      campaignId: r.campaignId,
      campaignName: r.campaignId ? (names.get(r.campaignId) ?? null) : null,
      date: toISODate(r.date),
      days: 1,
      message: r.message,
    });
  }
  const alerts = [...grouped.values()].sort(
    (a, b) => severityRank(b.severity) - severityRank(a.severity) || b.date.localeCompare(a.date),
  );

  return {
    alerts,
    lastCheckedAt: (job?.finishedAt ?? job?.startedAt)?.toISOString() ?? null,
    weeklyInsight: insight ? { createdAt: insight.createdAt.toISOString(), isMock: insight.model === "mock" } : null,
  };
});

export type ScheduleItem = {
  id: string;
  name: string;
  cadence: "WEEKLY" | "MONTHLY";
  modules: ("SOCIAL" | "SEO" | "ADS")[];
  withInsight: boolean;
  enabled: boolean;
  lastRunAt: string | null;
  nextRunAt: string;
};

export const listSchedules = cache(async (clientId: string): Promise<ScheduleItem[]> => {
  const rows = await db.reportSchedule.findMany({ where: { clientId }, orderBy: { createdAt: "asc" } });
  return rows.map((r) => ({
    id: r.id,
    name: r.name,
    cadence: r.cadence,
    modules: r.modules,
    withInsight: r.withInsight,
    enabled: r.enabled,
    lastRunAt: r.lastRunAt?.toISOString() ?? null,
    nextRunAt: r.nextRunAt.toISOString(),
  }));
});
