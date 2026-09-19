/** Campaign → ad set → ad tables of the ads dashboard: breakdown tree, spend share, top ads. */
import "server-only";
import { aggregateAds } from "@/lib/metrics";
import { isConversion, type InsightRow } from "./queries-insights";

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

export type AdsCampaignShare = { id: string; name: string; spend: number; results: number; resultType: string | null };

type CampaignMeta = { id: string; name: string; status: string; objective: string };
type AdSetMeta = { id: string; name: string; status: string; campaignId: string };
type AdMeta = { id: string; name: string; status: string; thumbnailUrl: string | null; adSetId: string };

export type GroupedInsights = {
  byCampaign: Map<string, InsightRow[]>;
  byAdSet: Map<string, InsightRow[]>;
  byAd: Map<string, InsightRow[]>;
};

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

export function groupInsights(rows: InsightRow[]): GroupedInsights {
  const byCampaign = new Map<string, InsightRow[]>();
  const byAdSet = new Map<string, InsightRow[]>();
  const byAd = new Map<string, InsightRow[]>();
  for (const r of rows) {
    if (r.campaignId) pushTo(byCampaign, r.campaignId, r);
    if (r.adSetId) pushTo(byAdSet, r.adSetId, r);
    if (r.adId) pushTo(byAd, r.adId, r);
  }
  return { byCampaign, byAdSet, byAd };
}

/** Rincian per kampanye → set iklan → iklan, tiap tingkat diurutkan menurut belanja. */
export function buildBreakdown(
  campaigns: CampaignMeta[],
  adSets: AdSetMeta[],
  ads: AdMeta[],
  { byCampaign, byAdSet, byAd }: GroupedInsights,
): AdsCampaignNode[] {
  const adsBySet = new Map<string, AdMeta[]>();
  for (const a of ads) pushTo(adsBySet, a.adSetId, a);
  const setsByCampaign = new Map<string, AdSetMeta[]>();
  for (const s of adSets) pushTo(setsByCampaign, s.campaignId, s);

  return campaigns
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
}

export function buildCampaignShare(breakdown: AdsCampaignNode[]): AdsCampaignShare[] {
  return breakdown
    .filter((c) => c.kpis.spend > 0)
    .map((c) => ({ id: c.id, name: c.name, spend: c.kpis.spend, results: c.kpis.results, resultType: c.kpis.resultType }));
}

/** Iklan terbaik: CPR terendah di antara iklan konversi bervolume memadai. */
export function buildTopAds(
  allCampaigns: { id: string; name: string }[],
  adSets: AdSetMeta[],
  ads: AdMeta[],
  byAd: Map<string, InsightRow[]>,
): AdsTopAd[] {
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
  return perAd
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
}
