/** Insight rows and the account-level KPI rules shared by the ads dashboard queries. */
import "server-only";
import { aggregateAds, type AdKpis } from "@/lib/metrics";
import { NON_CONVERSION_RESULT_TYPES } from "@/features/overview/queries";

export type InsightRow = {
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

export function isConversion(r: { resultType: string | null }): boolean {
  return !NON_CONVERSION_RESULT_TYPES.has(r.resultType ?? "");
}

/** KPI agregat dengan results/cpr/roas hanya dari baris konversi (pola getOverviewAds). */
export function kpisWithConversion(rows: InsightRow[]): AdKpis {
  const all = aggregateAds(rows);
  const conv = aggregateAds(rows.filter(isConversion));
  return { ...all, results: conv.results, cpr: conv.cpr, roas: conv.roas };
}

/** Tipe hasil dominan: berdasar total hasil baris konversi; fallback ke semua baris. */
export function dominantResultType(rows: InsightRow[]): string | null {
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
