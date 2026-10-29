/**
 * Weekly auto ADS insight (called from the daily job). Reuses the report insight
 * path (`generateInsightCore`), so the result is cached in AiInsight for the
 * last 7 complete days and shows up in Laporan → Analisa AI with preset 7d.
 * At most one ADS insight per client per 7 days (manual ones count too).
 */
import "server-only";
import { db } from "@/lib/db";
import { formatDateRange } from "@/lib/format";
import { rangeForPreset } from "@/lib/dates";
import { clientAudience, notify } from "@/lib/notify";
import { generateInsightCore } from "@/features/reports/create-core";
import { isInsightStale } from "./lib";
import { aa } from "./strings";

export type AutoInsightResult = { generated: boolean; reason?: "no-ads" | "fresh"; model?: string };

export async function ensureWeeklyAdsInsight(opts: { clientId: string; now?: Date }): Promise<AutoInsightResult> {
  const now = opts.now ?? new Date();
  const client = await db.client.findUnique({ where: { id: opts.clientId }, select: { id: true, slug: true, name: true } });
  if (!client) return { generated: false, reason: "no-ads" };

  const range = rangeForPreset("7d", now);
  const hasAds = await db.adDailyInsight.count({
    where: { adAccount: { clientId: client.id }, adId: { not: null }, date: { gte: range.from, lte: range.to }, spend: { gt: 0 } },
  });
  if (hasAds === 0) return { generated: false, reason: "no-ads" };

  const last = await db.aiInsight.findFirst({
    where: { clientId: client.id, module: "ADS" },
    orderBy: { createdAt: "desc" },
    select: { createdAt: true },
  });
  if (!isInsightStale(last?.createdAt ?? null, now)) return { generated: false, reason: "fresh" };

  const job = await db.syncJob.create({ data: { clientId: client.id, kind: "ADS_AUTO_INSIGHT", status: "RUNNING" } });
  try {
    const res = await generateInsightCore({ clientId: client.id, clientName: client.name, module: "ADS", range, language: "id" });
    const period = formatDateRange(range.from, range.to);
    await notify(await clientAudience(client.id), {
      type: "ADS_INSIGHT",
      title: aa.notif.insightTitle,
      body: aa.notif.insightBody(period),
      href: `/clients/${client.slug}/reports?tab=insight&preset=7d`,
      clientId: client.id,
    });
    await db.syncJob.update({
      where: { id: job.id },
      data: { status: "SUCCESS", finishedAt: new Date(), message: res.model === "mock" ? "mock" : period },
    });
    return { generated: true, model: res.model };
  } catch (e) {
    const message = e instanceof Error ? e.message : String(e);
    await db.syncJob.update({ where: { id: job.id }, data: { status: "FAILED", finishedAt: new Date(), error: message.slice(0, 900) } });
    throw e;
  }
}
