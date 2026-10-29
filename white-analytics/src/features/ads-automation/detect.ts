/**
 * Daily Meta Ads anomaly detection (called from features/sync/daily.ts after the
 * ads step, and on demand from the "Periksa sekarang" action).
 *
 * Loads ad-level AdDailyInsight rows per campaign and day, runs the pure rules in
 * `@/lib/metrics/ads-anomaly`, upserts AdAlert rows idempotently (unique per
 * client + rule + campaign + day) and notifies the client audience about NEW
 * WARNING/CRITICAL alerts on the latest day only. One SyncJob ADS_ANOMALY per run.
 */
import "server-only";
import { db } from "@/lib/db";
import { addDays, latestCompleteDay, parseISODate, toISODate } from "@/lib/dates";
import { inDemoMode } from "@/lib/demo-mode";
import { clientAudience, notify } from "@/lib/notify";
import {
  BASELINE_DAYS,
  accountSubject,
  detectAnomalies,
  isNotifiable,
  type AdDay,
  type Anomaly,
  type AnomalySubject,
} from "@/lib/metrics/ads-anomaly";
import { NON_CONVERSION_RESULT_TYPES } from "@/features/overview/queries";
import { aa, ruleLabel } from "./strings";

export type DetectResult = { anchor: string | null; days: number; found: number; created: number; notified: number };

/** How many anchor days a first run (or a manual check) scans so the panel has history. */
export const BACKFILL_DAYS = 7;

export async function detectAdAnomalies(opts: { clientId: string; now?: Date; lookbackDays?: number }): Promise<DetectResult> {
  const now = opts.now ?? new Date();
  const client = await db.client.findUnique({ where: { id: opts.clientId }, select: { id: true, slug: true, currency: true } });
  const empty: DetectResult = { anchor: null, days: 0, found: 0, created: 0, notified: 0 };
  if (!client) return empty;
  const accounts = await db.adAccount.findMany({ where: { clientId: client.id }, select: { id: true, connectionId: true, currency: true } });
  if (accounts.length === 0) return empty;
  const accountIds = accounts.map((a) => a.id);

  // Anchor = latest complete day. Accounts without a live connection (demo seed,
  // CSV imports) anchor on their latest data day instead, so stale demo data
  // never reads as "spend stopped".
  const yesterday = latestCompleteDay(now);
  const live = !inDemoMode() && accounts.some((a) => a.connectionId);
  const latest = await db.adDailyInsight.aggregate({ where: { adAccountId: { in: accountIds }, adId: { not: null } }, _max: { date: true } });
  const maxDate = latest._max.date;
  if (!maxDate) return empty;
  const anchorDate = live || maxDate > yesterday ? yesterday : maxDate;

  const lookback = opts.lookbackDays ?? ((await db.adAlert.count({ where: { clientId: client.id } })) === 0 ? BACKFILL_DAYS : 1);
  const anchors = Array.from({ length: Math.max(1, lookback) }, (_, i) => addDays(anchorDate, -i));
  const from = addDays(anchors[anchors.length - 1]!, -BASELINE_DAYS);

  const job = await db.syncJob.create({ data: { clientId: client.id, kind: "ADS_ANOMALY", status: "RUNNING", message: toISODate(anchorDate) } });
  try {
    const [campaigns, rows] = await Promise.all([
      db.adCampaign.findMany({ where: { adAccountId: { in: accountIds } }, select: { id: true, name: true, status: true } }),
      db.adDailyInsight.groupBy({
        by: ["campaignId", "date", "resultType"],
        where: { adAccountId: { in: accountIds }, adId: { not: null }, date: { gte: from, lte: anchorDate } },
        _sum: { spend: true, impressions: true, reach: true, linkClicks: true, results: true, purchaseValue: true },
      }),
    ]);

    const subjects = new Map<string, AnomalySubject>();
    for (const c of campaigns) subjects.set(c.id, { campaignId: c.id, name: c.name, active: c.status === "ACTIVE", isConversion: false, days: [] });
    for (const r of rows) {
      const s = r.campaignId ? subjects.get(r.campaignId) : undefined;
      if (!s) continue;
      const conversion = !NON_CONVERSION_RESULT_TYPES.has(r.resultType ?? "");
      if (conversion) s.isConversion = true;
      const day: AdDay = {
        date: toISODate(r.date),
        spend: r._sum.spend ?? 0,
        impressions: r._sum.impressions ?? 0,
        reach: r._sum.reach ?? 0,
        linkClicks: r._sum.linkClicks ?? 0,
        results: conversion ? (r._sum.results ?? 0) : 0,
        purchaseValue: r._sum.purchaseValue ?? null,
      };
      s.days.push(day);
    }
    const list = [...subjects.values()];
    const all = [...list, accountSubject(list)];
    const currency = accounts[0]?.currency ?? client.currency;

    const found: Anomaly[] = anchors.flatMap((a) => detectAnomalies(all, toISODate(a), currency));
    const { created, fresh } = await persistAlerts(client.id, found);

    // Notify only about NEW, notifiable alerts on the latest day (backfill stays quiet).
    const latestKey = toISODate(anchorDate);
    const toNotify = fresh.filter((a) => a.date === latestKey && isNotifiable(a.severity));
    let notified = 0;
    if (toNotify.length > 0) {
      const names = new Map(campaigns.map((c) => [c.id, c.name]));
      const sample = toNotify
        .slice(0, 2)
        .map((a) => `${ruleLabel(a.rule)}: ${a.campaignId ? (names.get(a.campaignId) ?? "") : aa.alerts.account}`)
        .join(", ");
      const critical = toNotify.filter((a) => a.severity === "CRITICAL").length;
      notified = await notify(await clientAudience(client.id), {
        type: "ADS_ANOMALY",
        title: aa.notif.anomalyTitle(toNotify.length, critical),
        body: aa.notif.anomalyBody(sample, toNotify.length - 2),
        href: `/clients/${client.slug}/ads`,
        clientId: client.id,
      });
    }

    await db.syncJob.update({
      where: { id: job.id },
      data: { status: "SUCCESS", finishedAt: new Date(), message: `${toISODate(anchorDate)} · ${found.length} anomali, ${created} baru` },
    });
    return { anchor: latestKey, days: anchors.length, found: found.length, created, notified };
  } catch (e) {
    const message = e instanceof Error ? e.message : String(e);
    await db.syncJob.update({ where: { id: job.id }, data: { status: "FAILED", finishedAt: new Date(), error: message.slice(0, 900) } });
    throw e;
  }
}

/** Insert new alerts, refresh existing ones (keeping resolvedAt). Returns the new ones. */
async function persistAlerts(clientId: string, found: Anomaly[]): Promise<{ created: number; fresh: Anomaly[] }> {
  if (found.length === 0) return { created: 0, fresh: [] };
  const key = (a: { rule: string; campaignId: string; date: string }) => `${a.rule}|${a.campaignId}|${a.date}`;
  const dates = [...new Set(found.map((a) => a.date))].map((d) => parseISODate(d)!);
  const existing = await db.adAlert.findMany({
    where: { clientId, date: { in: dates } },
    select: { id: true, rule: true, campaignId: true, date: true },
  });
  const byKey = new Map(existing.map((e) => [key({ ...e, date: toISODate(e.date) }), e.id]));
  const fresh = found.filter((a) => !byKey.has(key(a)));
  const toRow = (a: Anomaly) => ({
    severity: a.severity,
    metric: a.metric,
    current: a.current,
    baseline: a.baseline,
    message: a.message,
  });
  const res = await db.adAlert.createMany({
    data: fresh.map((a) => ({ clientId, rule: a.rule, campaignId: a.campaignId, date: parseISODate(a.date)!, ...toRow(a) })),
    skipDuplicates: true,
  });
  const updates = found.filter((a) => byKey.has(key(a)));
  for (const a of updates) await db.adAlert.update({ where: { id: byKey.get(key(a))! }, data: toRow(a) });
  return { created: res.count, fresh };
}
