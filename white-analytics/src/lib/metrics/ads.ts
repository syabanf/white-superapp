import { safeDiv } from "./delta";

export type AdInsightLike = {
  spend: number;
  impressions: number;
  reach: number;
  clicks: number;
  linkClicks: number;
  results: number;
  purchaseValue?: number | null;
};

export type AdKpis = {
  spend: number;
  impressions: number;
  reach: number;
  clicks: number;
  linkClicks: number;
  results: number;
  /** cost per result = spend / results */
  cpr: number;
  /** click-through rate (%) = linkClicks / impressions × 100 — the REAL CTR (not results/impressions) */
  ctr: number;
  /** cost per link click */
  cpc: number;
  /** cost per mille = spend / impressions × 1000 */
  cpm: number;
  /** frequency = impressions / reach */
  frequency: number;
  /** return on ad spend when purchase value known */
  roas: number | null;
};

export const EMPTY_AD_KPIS: AdKpis = {
  spend: 0,
  impressions: 0,
  reach: 0,
  clicks: 0,
  linkClicks: 0,
  results: 0,
  cpr: 0,
  ctr: 0,
  cpc: 0,
  cpm: 0,
  frequency: 0,
  roas: null,
};

export function costPerResult(spend: number, results: number): number {
  return safeDiv(spend, results);
}
export function adCtr(linkClicks: number, impressions: number): number {
  return safeDiv(linkClicks, impressions) * 100;
}
export function cpc(spend: number, linkClicks: number): number {
  return safeDiv(spend, linkClicks);
}
export function cpm(spend: number, impressions: number): number {
  return safeDiv(spend, impressions) * 1000;
}
export function frequency(impressions: number, reach: number): number {
  return safeDiv(impressions, reach);
}
export function roas(purchaseValue: number | null | undefined, spend: number): number | null {
  if (purchaseValue == null || spend === 0) return null;
  return purchaseValue / spend;
}

/** Aggregate rows into KPIs (reach is summed — note: cross-day reach sums over-count; treat as approximation) */
export function aggregateAds(rows: AdInsightLike[]): AdKpis {
  let spend = 0;
  let impressions = 0;
  let reach = 0;
  let clicks = 0;
  let linkClicks = 0;
  let results = 0;
  let purchase = 0;
  let hasPurchase = false;
  for (const r of rows) {
    spend += r.spend;
    impressions += r.impressions;
    reach += r.reach;
    clicks += r.clicks;
    linkClicks += r.linkClicks;
    results += r.results;
    if (r.purchaseValue != null) {
      hasPurchase = true;
      purchase += r.purchaseValue;
    }
  }
  return {
    spend,
    impressions,
    reach,
    clicks,
    linkClicks,
    results,
    cpr: costPerResult(spend, results),
    ctr: adCtr(linkClicks, impressions),
    cpc: cpc(spend, linkClicks),
    cpm: cpm(spend, impressions),
    frequency: frequency(impressions, reach),
    roas: hasPurchase ? roas(purchase, spend) : null,
  };
}

/** Ad fatigue heuristic used by Meta media buyers: frequency above 3 in the period. */
export const FREQUENCY_FATIGUE_THRESHOLD = 3.0;

export function isFatigued(freq: number): boolean {
  return freq > FREQUENCY_FATIGUE_THRESHOLD;
}
