/**
 * Pure mappers from Meta Marketing API insight rows to AdDailyInsight /
 * AdDemographic aggregates. Tested in tests/sync-map.test.ts.
 */
import type { MetaInsightRow } from "@/lib/providers/meta-marketing/types";

export type DailyInsightAgg = {
  date: string;
  campaignExternalId: string;
  adSetExternalId: string;
  adExternalId: string;
  spend: number;
  impressions: number;
  reach: number;
  clicks: number;
  linkClicks: number;
  results: number;
  resultType: string | null;
  purchaseValue: number | null;
};

/** Ad-level daily rows → one aggregate per (date, campaign, adset, ad). */
export function aggregateDailyInsights(rows: MetaInsightRow[]): DailyInsightAgg[] {
  const by = new Map<string, DailyInsightAgg>();
  for (const r of rows) {
    if (!/^\d{4}-\d{2}-\d{2}$/.test(r.date) || !r.adId) continue;
    const key = `${r.date}|${r.campaignId}|${r.adSetId}|${r.adId}`;
    const acc = by.get(key) ?? {
      date: r.date, campaignExternalId: r.campaignId, adSetExternalId: r.adSetId, adExternalId: r.adId,
      spend: 0, impressions: 0, reach: 0, clicks: 0, linkClicks: 0, results: 0, resultType: null, purchaseValue: null,
    };
    acc.spend += r.spend;
    acc.impressions += r.impressions;
    acc.reach += r.reach;
    acc.clicks += r.clicks;
    acc.linkClicks += r.linkClicks;
    acc.results += r.results;
    if (r.resultType) acc.resultType = r.resultType;
    if (r.purchaseValue != null) acc.purchaseValue = (acc.purchaseValue ?? 0) + r.purchaseValue;
    by.set(key, acc);
  }
  return [...by.values()];
}

export type DemographicAgg = { campaignExternalId: string; age: string; gender: string; spend: number; impressions: number; reach: number; results: number };

/** Campaign-level rows with age/gender breakdown → aggregate over the whole window. */
export function aggregateDemographics(rows: MetaInsightRow[]): DemographicAgg[] {
  const by = new Map<string, DemographicAgg>();
  for (const r of rows) {
    if (!r.campaignId) continue;
    const age = r.age?.trim() || "unknown";
    const gender = (r.gender?.trim() || "unknown").toLowerCase();
    const key = `${r.campaignId}|${age}|${gender}`;
    const acc = by.get(key) ?? { campaignExternalId: r.campaignId, age, gender, spend: 0, impressions: 0, reach: 0, results: 0 };
    acc.spend += r.spend;
    acc.impressions += r.impressions;
    acc.reach += r.reach;
    acc.results += r.results;
    by.set(key, acc);
  }
  return [...by.values()];
}
