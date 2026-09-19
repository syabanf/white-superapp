import "server-only";
import { cache } from "react";
import { db } from "@/lib/db";
import type { DateRange } from "@/lib/dates";
import {
  aggregateAds,
  alignPrevious,
  computeDelta,
  fillDaily,
  sparkline,
  type AdKpis,
  type Delta,
} from "@/lib/metrics";
import {
  buildBreakdown,
  buildCampaignShare,
  buildTopAds,
  groupInsights,
  type AdsCampaignNode,
  type AdsCampaignShare,
  type AdsTopAd,
} from "./queries-breakdown";
import { buildDemographics, type AdsDemographics } from "./queries-demographics";
import { dominantResultType, isConversion, kpisWithConversion, type InsightRow } from "./queries-insights";

export type AdsAccountSummary = {
  id: string;
  externalId: string;
  name: string;
  currency: string;
  hasConnection: boolean;
  lastSyncedAt: string | null; // ISO
};

/** Akun iklan milik klien (untuk pengaturan/halaman ads). */
export const getAdAccounts = cache(async (clientId: string): Promise<AdsAccountSummary[]> => {
  const accounts = await db.adAccount.findMany({
    where: { clientId },
    orderBy: { createdAt: "asc" },
    select: { id: true, externalId: true, name: true, currency: true, connectionId: true, lastSyncedAt: true },
  });
  return accounts.map((a) => ({
    id: a.id,
    externalId: a.externalId,
    name: a.name,
    currency: a.currency,
    hasConnection: a.connectionId != null,
    lastSyncedAt: a.lastSyncedAt ? a.lastSyncedAt.toISOString() : null,
  }));
});

export type AdsDashboard = {
  hasData: boolean;
  /** true bila tidak ada akun yang terhubung ke Connection (data seed/demo) */
  isDemo: boolean;
  account: AdsAccountSummary | null;
  campaignOptions: { id: string; name: string }[];
  /** results/cpr/roas hanya dari kampanye konversi (lihat NON_CONVERSION_RESULT_TYPES) */
  kpis: AdKpis;
  prev: AdKpis;
  deltas: {
    spend: Delta;
    results: Delta;
    cpr: Delta;
    impressions: Delta;
    reach: Delta;
    linkClicks: Delta;
    ctr: Delta;
    cpc: Delta;
    cpm: Delta;
    frequency: Delta;
    roas: Delta | null;
  };
  dominantResultType: string | null;
  spendSpark: number[];
  resultsSpark: number[];
  daily: { date: string; spend: number; results: number; spendPrev: number | null; resultsPrev: number | null }[];
  campaignShare: AdsCampaignShare[];
  breakdown: AdsCampaignNode[];
  demographics: AdsDemographics;
  topAds: AdsTopAd[];
};

function emptyDashboard(): AdsDashboard {
  const zero = computeDelta(0, 0);
  const empty = aggregateAds([]);
  return {
    hasData: false,
    isDemo: true,
    account: null,
    campaignOptions: [],
    kpis: empty,
    prev: empty,
    deltas: { spend: zero, results: zero, cpr: zero, impressions: zero, reach: zero, linkClicks: zero, ctr: zero, cpc: zero, cpm: zero, frequency: zero, roas: null },
    dominantResultType: null,
    spendSpark: [],
    resultsSpark: [],
    daily: [],
    campaignShare: [],
    breakdown: [],
    demographics: { hasData: false, byAge: [], byGender: [], matrix: [] },
    topAds: [],
  };
}

/** Dashboard Meta Ads lengkap untuk satu klien + rentang (+ filter kampanye opsional). */
export const getAdsDashboard = cache(
  async (clientId: string, range: DateRange, previous: DateRange, campaignId?: string): Promise<AdsDashboard> => {
    const accounts = await getAdAccounts(clientId);
    if (accounts.length === 0) return emptyDashboard();
    const ids = accounts.map((a) => a.id);

    const allCampaigns = await db.adCampaign.findMany({
      where: { adAccountId: { in: ids } },
      orderBy: { name: "asc" },
      select: { id: true, name: true, status: true, objective: true },
    });
    // Abaikan filter yang tidak dikenal (mis. id basi di URL)
    const filterId = campaignId && allCampaigns.some((c) => c.id === campaignId) ? campaignId : undefined;
    const campaignWhere = filterId ? { campaignId: filterId } : {};

    const [rows, prevRows, demoRows, adSets, ads] = await Promise.all([
      db.adDailyInsight.findMany({
        where: { adAccountId: { in: ids }, adId: { not: null }, date: { gte: range.from, lte: range.to }, ...campaignWhere },
        orderBy: { date: "asc" },
      }),
      db.adDailyInsight.findMany({
        where: { adAccountId: { in: ids }, adId: { not: null }, date: { gte: previous.from, lte: previous.to }, ...campaignWhere },
      }),
      db.adDemographic.findMany({
        where: { adAccountId: { in: ids }, dateFrom: { lte: range.to }, dateTo: { gte: range.from }, ...(filterId ? { campaignId: filterId } : {}) },
      }),
      db.adSet.findMany({
        where: { campaign: { adAccountId: { in: ids }, ...(filterId ? { id: filterId } : {}) } },
        select: { id: true, name: true, status: true, campaignId: true },
      }),
      db.ad.findMany({
        where: { adSet: { campaign: { adAccountId: { in: ids }, ...(filterId ? { id: filterId } : {}) } } },
        select: { id: true, name: true, status: true, thumbnailUrl: true, adSetId: true },
      }),
    ]);

    const kpis = kpisWithConversion(rows);
    const prev = kpisWithConversion(prevRows);

    // Deret harian: hasil non-konversi di-nol-kan bila ada baris konversi (pola overview),
    // agar grafik "Hasil harian" konsisten dengan KPI Hasil.
    const dailySource = (rs: InsightRow[]) =>
      rs.some(isConversion) ? rs.map((r) => (isConversion(r) ? r : { ...r, results: 0 })) : rs;
    const daily = fillDaily(range, dailySource(rows), (r) => r.date, ["spend", "results"]);
    const prevDaily = fillDaily(previous, dailySource(prevRows), (r) => r.date, ["spend", "results"]);
    const aligned = alignPrevious(alignPrevious(daily, prevDaily, "spend"), prevDaily, "results") as {
      date: string;
      spend: number;
      results: number;
      spendPrev: number | null;
      resultsPrev: number | null;
    }[];

    const grouped = groupInsights(rows);
    const scopedCampaigns = filterId ? allCampaigns.filter((c) => c.id === filterId) : allCampaigns;
    const breakdown = buildBreakdown(scopedCampaigns, adSets, ads, grouped);

    const roasDelta = kpis.roas != null && prev.roas != null ? computeDelta(kpis.roas, prev.roas) : null;

    return {
      hasData: rows.length > 0 || prevRows.length > 0,
      isDemo: accounts.every((a) => !a.hasConnection),
      account: accounts[0] ?? null,
      campaignOptions: allCampaigns.map((c) => ({ id: c.id, name: c.name })),
      kpis,
      prev,
      deltas: {
        spend: computeDelta(kpis.spend, prev.spend),
        results: computeDelta(kpis.results, prev.results),
        cpr: computeDelta(kpis.cpr, prev.cpr),
        impressions: computeDelta(kpis.impressions, prev.impressions),
        reach: computeDelta(kpis.reach, prev.reach),
        linkClicks: computeDelta(kpis.linkClicks, prev.linkClicks),
        ctr: computeDelta(kpis.ctr, prev.ctr),
        cpc: computeDelta(kpis.cpc, prev.cpc),
        cpm: computeDelta(kpis.cpm, prev.cpm),
        frequency: computeDelta(kpis.frequency, prev.frequency),
        roas: roasDelta,
      },
      dominantResultType: dominantResultType(rows),
      spendSpark: sparkline(daily, "spend"),
      resultsSpark: sparkline(daily, "results"),
      daily: aligned,
      campaignShare: buildCampaignShare(breakdown),
      breakdown,
      demographics: buildDemographics(demoRows),
      topAds: buildTopAds(allCampaigns, adSets, ads, grouped.byAd),
    };
  },
);
