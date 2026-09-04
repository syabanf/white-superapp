/**
 * Real Meta Ads sync (Marketing API). Auth-free write path shared by the
 * "Sinkronkan" action and the daily cron. Runs only for ad accounts whose Meta
 * connection has a usable token; otherwise `mode: "demo"` and nothing is written.
 */
import "server-only";
import { db } from "@/lib/db";
import { addDays, latestCompleteDay, parseISODate, toISODate } from "@/lib/dates";
import { getConnectionToken } from "@/lib/connections";
import { getMetaMarketing, type MetaMarketingAdapter } from "@/lib/providers/meta-marketing";
import { aggregateDailyInsights, aggregateDemographics } from "./sync-map";

export type AdsSyncResult = { mode: "real" | "demo"; campaigns: number; adSets: number; ads: number; days: number; demographics: number; error?: string };

type AccountRow = { id: string; clientId: string; externalId: string; connectionId: string | null };

export async function syncAdAccount(account: AccountRow, opts: { days?: number; adapter?: MetaMarketingAdapter } = {}): Promise<AdsSyncResult> {
  let adapter = opts.adapter;
  if (!adapter) {
    const resolved = await getConnectionToken(account.connectionId);
    if (!resolved) return { mode: "demo", campaigns: 0, adSets: 0, ads: 0, days: 0, demographics: 0 };
    adapter = getMetaMarketing(resolved.token);
  }
  const until = latestCompleteDay();
  const since = addDays(until, -((opts.days ?? 30) - 1));
  const window = { since: toISODate(since), until: toISODate(until) };
  const actId = account.externalId;

  // ── hierarchy ──
  const [campaigns, adSets, ads] = await Promise.all([adapter.listCampaigns(actId), adapter.listAdSets(actId), adapter.listAds(actId)]);
  const campaignIds = new Map<string, string>();
  for (const c of campaigns) {
    const row = await db.adCampaign.upsert({
      where: { adAccountId_externalId: { adAccountId: account.id, externalId: c.id } },
      create: { adAccountId: account.id, externalId: c.id, name: c.name, objective: c.objective, status: c.status, dailyBudget: c.dailyBudget, startTime: c.startTime ? new Date(c.startTime) : null },
      update: { name: c.name, objective: c.objective, status: c.status, dailyBudget: c.dailyBudget, startTime: c.startTime ? new Date(c.startTime) : null },
    });
    campaignIds.set(c.id, row.id);
  }
  const adSetIds = new Map<string, string>();
  for (const s of adSets) {
    const campaignId = campaignIds.get(s.campaignId);
    if (!campaignId) continue;
    const row = await db.adSet.upsert({
      where: { campaignId_externalId: { campaignId, externalId: s.id } },
      create: { campaignId, externalId: s.id, name: s.name, status: s.status, dailyBudget: s.dailyBudget },
      update: { name: s.name, status: s.status, dailyBudget: s.dailyBudget },
    });
    adSetIds.set(s.id, row.id);
  }
  const adIds = new Map<string, string>();
  for (const a of ads) {
    const adSetId = adSetIds.get(a.adSetId);
    if (!adSetId) continue;
    const row = await db.ad.upsert({
      where: { adSetId_externalId: { adSetId, externalId: a.id } },
      create: { adSetId, externalId: a.id, name: a.name, status: a.status, thumbnailUrl: a.thumbnailUrl },
      update: { name: a.name, status: a.status, thumbnailUrl: a.thumbnailUrl },
    });
    adIds.set(a.id, row.id);
  }

  // ── daily insights (ad level) ──
  const daily = aggregateDailyInsights(await adapter.getInsights(actId, { ...window, level: "ad", timeIncrement: 1 }));
  const dates = [...new Set(daily.map((d) => d.date))].map((d) => parseISODate(d)!).filter(Boolean);
  const existing = dates.length ? await db.adDailyInsight.findMany({ where: { adAccountId: account.id, date: { in: dates } }, select: { id: true, date: true, campaignId: true, adSetId: true, adId: true } }) : [];
  const existingByKey = new Map(existing.map((e) => [`${toISODate(e.date)}|${e.campaignId}|${e.adSetId}|${e.adId}`, e.id]));
  let days = 0;
  const seenDays = new Set<string>();
  for (const part of chunk(daily, 100)) {
    await db.$transaction(
      part.flatMap((d) => {
        const campaignId = campaignIds.get(d.campaignExternalId) ?? null;
        const adSetId = adSetIds.get(d.adSetExternalId) ?? null;
        const adId = adIds.get(d.adExternalId) ?? null;
        if (!adId) return [];
        seenDays.add(d.date);
        const data = { spend: d.spend, impressions: d.impressions, reach: d.reach, clicks: d.clicks, linkClicks: d.linkClicks, results: d.results, resultType: d.resultType, purchaseValue: d.purchaseValue };
        const id = existingByKey.get(`${d.date}|${campaignId}|${adSetId}|${adId}`);
        return [id ? db.adDailyInsight.update({ where: { id }, data }) : db.adDailyInsight.create({ data: { adAccountId: account.id, date: parseISODate(d.date)!, campaignId, adSetId, adId, ...data } })];
      }),
    );
  }
  days = seenDays.size;

  // ── demographics (campaign level, whole window) ──
  const demo = aggregateDemographics(await adapter.getInsights(actId, { ...window, level: "campaign", breakdowns: ["age", "gender"] }));
  let demographics = 0;
  for (const d of demo) {
    const campaignId = campaignIds.get(d.campaignExternalId) ?? null;
    const found = await db.adDemographic.findFirst({ where: { adAccountId: account.id, dateFrom: since, dateTo: until, campaignId, age: d.age, gender: d.gender }, select: { id: true } });
    const data = { spend: d.spend, impressions: d.impressions, reach: d.reach, results: d.results };
    if (found) await db.adDemographic.update({ where: { id: found.id }, data });
    else await db.adDemographic.create({ data: { adAccountId: account.id, dateFrom: since, dateTo: until, campaignId, age: d.age, gender: d.gender, ...data } });
    demographics++;
  }

  await db.adAccount.update({ where: { id: account.id }, data: { lastSyncedAt: new Date() } });
  return { mode: "real", campaigns: campaigns.length, adSets: adSets.length, ads: ads.length, days, demographics };
}

export async function syncAdsForClient(clientId: string, opts: { days?: number } = {}): Promise<{ accounts: number; real: number; results: AdsSyncResult[] }> {
  const accounts = await db.adAccount.findMany({ where: { clientId }, select: { id: true, clientId: true, externalId: true, connectionId: true } });
  const results: AdsSyncResult[] = [];
  let real = 0;
  for (const a of accounts) {
    let r: AdsSyncResult;
    try {
      r = await syncAdAccount(a, opts);
    } catch (e) {
      r = { mode: "real", campaigns: 0, adSets: 0, ads: 0, days: 0, demographics: 0, error: e instanceof Error ? e.message : String(e) };
    }
    results.push(r);
    if (r.mode === "real" && !r.error) real++;
    await db.syncJob.create({
      data: {
        clientId,
        kind: "ADS_INSIGHTS",
        status: r.error ? "FAILED" : "SUCCESS",
        finishedAt: new Date(),
        message: r.mode === "demo" ? "mock" : `${r.campaigns} kampanye, ${r.ads} iklan, ${r.days} hari · ${a.externalId}`,
        error: r.error ?? null,
      },
    });
  }
  return { accounts: accounts.length, real, results };
}

function chunk<T>(arr: T[], size: number): T[][] {
  const out: T[][] = [];
  for (let i = 0; i < arr.length; i += size) out.push(arr.slice(i, i + size));
  return out;
}
