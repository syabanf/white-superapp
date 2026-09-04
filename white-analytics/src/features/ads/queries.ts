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
import { NON_CONVERSION_RESULT_TYPES } from "@/features/overview/queries";

const AGE_ORDER = ["13-17", "18-24", "25-34", "35-44", "45-54", "55-64", "65+", "unknown"];
const GENDER_ORDER = ["female", "male", "unknown"];

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

export type AdsNodeKpis = {
  spend: number;
  results: number;
  resultType: string | null;
  cpr: number;
  impressions: number;
  reach: number;
  linkClicks: number;
  ctr: number;
  cpc: number;
  frequency: number;
};

export type AdsAdNode = { id: string; name: string; status: string; thumbnailUrl: string | null; kpis: AdsNodeKpis };
export type AdsAdSetNode = { id: string; name: string; status: string; kpis: AdsNodeKpis; ads: AdsAdNode[] };
export type AdsCampaignNode = {
  id: string;
  name: string;
  status: string;
  objective: string;
  kpis: AdsNodeKpis;
  adSets: AdsAdSetNode[];
};

export type AdsTopAd = {
  id: string;
  name: string;
  campaignName: string;
  status: string;
  thumbnailUrl: string | null;
  spend: number;
  results: number;
  resultType: string | null;
  cpr: number;
  ctr: number;
};

export type AdsDemographicRow = { age: string; gender: string; spend: number; results: number };

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
  campaignShare: { id: string; name: string; spend: number; results: number; resultType: string | null }[];
  breakdown: AdsCampaignNode[];
  demographics: { hasData: boolean; byAge: AdsDemographicRow[]; byGender: AdsDemographicRow[]; matrix: AdsDemographicRow[] };
  topAds: AdsTopAd[];
};

type InsightRow = {
  date: Date;
  campaignId: string | null;
  adSetId: string | null;
  adId: string | null;
  spend: number;
  impressions: number;
  reach: number;
  clicks: number;
  linkClicks: number;
  results: number;
  resultType: string | null;
  purchaseValue: number | null;
};

function isConversion(r: { resultType: string | null }): boolean {
  return !NON_CONVERSION_RESULT_TYPES.has(r.resultType ?? "");
}

/** KPI agregat dengan results/cpr/roas hanya dari baris konversi (pola getOverviewAds). */
function kpisWithConversion(rows: InsightRow[]): AdKpis {
  const all = aggregateAds(rows);
  const conv = aggregateAds(rows.filter(isConversion));
  return { ...all, results: conv.results, cpr: conv.cpr, roas: conv.roas };
}

/** Tipe hasil dominan: berdasar total hasil baris konversi; fallback ke semua baris. */
function dominantResultType(rows: InsightRow[]): string | null {
  const tally = (rs: InsightRow[]) => {
    const m = new Map<string, number>();
    for (const r of rs) {
      if (!r.resultType) continue;
      m.set(r.resultType, (m.get(r.resultType) ?? 0) + r.results);
    }
    let best: string | null = null;
    let max = -1;
    for (const [k, v] of m) {
      if (v > max) {
        max = v;
        best = k;
      }
    }
    return best;
  };
  return tally(rows.filter(isConversion)) ?? tally(rows);
}

function nodeKpis(rows: InsightRow[]): AdsNodeKpis {
  const k = aggregateAds(rows);
  let resultType: string | null = null;
  for (const r of rows) {
    if (r.resultType) {
      resultType = r.resultType;
      break;
    }
  }
  return {
    spend: k.spend,
    results: k.results,
    resultType,
    cpr: k.cpr,
    impressions: k.impressions,
    reach: k.reach,
    linkClicks: k.linkClicks,
    ctr: k.ctr,
    cpc: k.cpc,
    frequency: k.frequency,
  };
}

function pushTo<T>(map: Map<string, T[]>, key: string, value: T): void {
  const arr = map.get(key);
  if (arr) arr.push(value);
  else map.set(key, [value]);
}

function median(values: number[]): number {
  if (values.length === 0) return 0;
  const sorted = [...values].sort((a, b) => a - b);
  const mid = Math.floor(sorted.length / 2);
  return sorted.length % 2 === 0 ? (sorted[mid - 1]! + sorted[mid]!) / 2 : sorted[mid]!;
}

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

    // Rincian per kampanye → set iklan → iklan
    const byCampaign = new Map<string, InsightRow[]>();
    const byAdSet = new Map<string, InsightRow[]>();
    const byAd = new Map<string, InsightRow[]>();
    for (const r of rows) {
      if (r.campaignId) pushTo(byCampaign, r.campaignId, r);
      if (r.adSetId) pushTo(byAdSet, r.adSetId, r);
      if (r.adId) pushTo(byAd, r.adId, r);
    }

    const adsBySet = new Map<string, typeof ads>();
    for (const a of ads) pushTo(adsBySet, a.adSetId, a);
    const setsByCampaign = new Map<string, typeof adSets>();
    for (const s of adSets) pushTo(setsByCampaign, s.campaignId, s);

    const scopedCampaigns = filterId ? allCampaigns.filter((c) => c.id === filterId) : allCampaigns;
    const breakdown: AdsCampaignNode[] = scopedCampaigns
      .map((c) => {
        const adSetNodes: AdsAdSetNode[] = (setsByCampaign.get(c.id) ?? [])
          .map((s) => {
            const adNodes: AdsAdNode[] = (adsBySet.get(s.id) ?? [])
              .map((a) => ({ id: a.id, name: a.name, status: a.status, thumbnailUrl: a.thumbnailUrl, kpis: nodeKpis(byAd.get(a.id) ?? []) }))
              .sort((x, y) => y.kpis.spend - x.kpis.spend);
            return { id: s.id, name: s.name, status: s.status, kpis: nodeKpis(byAdSet.get(s.id) ?? []), ads: adNodes };
          })
          .sort((x, y) => y.kpis.spend - x.kpis.spend);
        return {
          id: c.id,
          name: c.name,
          status: c.status,
          objective: c.objective,
          kpis: nodeKpis(byCampaign.get(c.id) ?? []),
          adSets: adSetNodes,
        };
      })
      .sort((x, y) => y.kpis.spend - x.kpis.spend);

    const campaignShare = breakdown
      .filter((c) => c.kpis.spend > 0)
      .map((c) => ({ id: c.id, name: c.name, spend: c.kpis.spend, results: c.kpis.results, resultType: c.kpis.resultType }));

    // Demografi (baris AdDemographic yang beririsan dengan rentang)
    const aggDemo = (key: (r: (typeof demoRows)[number]) => string) => {
      const m = new Map<string, { spend: number; results: number }>();
      for (const r of demoRows) {
        const k = key(r);
        const acc = m.get(k) ?? { spend: 0, results: 0 };
        acc.spend += r.spend;
        acc.results += r.results;
        m.set(k, acc);
      }
      return m;
    };
    const orderIndex = (order: string[], v: string) => {
      const i = order.indexOf(v);
      return i === -1 ? order.length : i;
    };
    const byAge: AdsDemographicRow[] = [...aggDemo((r) => r.age)]
      .map(([age, v]) => ({ age, gender: "", ...v }))
      .sort((a, b) => orderIndex(AGE_ORDER, a.age) - orderIndex(AGE_ORDER, b.age) || a.age.localeCompare(b.age));
    const byGender: AdsDemographicRow[] = [...aggDemo((r) => r.gender)]
      .map(([gender, v]) => ({ age: "", gender, ...v }))
      .sort((a, b) => orderIndex(GENDER_ORDER, a.gender) - orderIndex(GENDER_ORDER, b.gender));
    const matrixMap = aggDemo((r) => `${r.age}|${r.gender}`);
    const matrix: AdsDemographicRow[] = [...matrixMap]
      .map(([k, v]) => {
        const [age, gender] = k.split("|") as [string, string];
        return { age, gender, ...v };
      })
      .sort(
        (a, b) =>
          orderIndex(AGE_ORDER, a.age) - orderIndex(AGE_ORDER, b.age) ||
          orderIndex(GENDER_ORDER, a.gender) - orderIndex(GENDER_ORDER, b.gender),
      );

    // Iklan terbaik: CPR terendah di antara iklan konversi bervolume memadai
    const adMeta = new Map(ads.map((a) => [a.id, a]));
    const setToCampaign = new Map(adSets.map((s) => [s.id, s.campaignId]));
    const campaignName = new Map(allCampaigns.map((c) => [c.id, c.name]));
    const perAd = [...byAd.entries()]
      .map(([adId, rs]) => ({ adId, rows: rs.filter(isConversion) }))
      .filter((x) => x.rows.length > 0)
      .map((x) => {
        const k = aggregateAds(x.rows);
        return { adId: x.adId, spend: k.spend, results: k.results, cpr: k.cpr, ctr: k.ctr, resultType: x.rows[0]!.resultType };
      })
      .filter((x) => x.results > 0);
    const med = median(perAd.map((x) => x.results));
    const topAds: AdsTopAd[] = perAd
      .filter((x) => x.results >= med || x.results >= 5)
      .sort((a, b) => a.cpr - b.cpr)
      .slice(0, 6)
      .map((x) => {
        const meta = adMeta.get(x.adId);
        const cid = meta ? setToCampaign.get(meta.adSetId) : undefined;
        return {
          id: x.adId,
          name: meta?.name ?? x.adId,
          campaignName: (cid && campaignName.get(cid)) || "–",
          status: meta?.status ?? "ACTIVE",
          thumbnailUrl: meta?.thumbnailUrl ?? null,
          spend: x.spend,
          results: x.results,
          resultType: x.resultType,
          cpr: x.cpr,
          ctr: x.ctr,
        };
      });

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
      campaignShare,
      breakdown,
      demographics: { hasData: demoRows.length > 0, byAge, byGender, matrix },
      topAds,
    };
  },
);
