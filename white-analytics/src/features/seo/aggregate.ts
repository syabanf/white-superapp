/**
 * Pure aggregation helpers for SEO dimension rows (no server-only imports — unit tested).
 * DB rows are per (date, key); pages aggregate them per key over the selected range
 * with clicks/impressions summed and position weighted by impressions.
 */
import { ctr, expectedCtr, positionBucket, weightedMean, type Opportunity, type PositionBucket } from "@/lib/metrics";

export type DimMetricRow = { key: string; clicks: number; impressions: number; position: number };

export type DimAgg = { key: string; clicks: number; impressions: number; ctr: number; position: number };

/** Group rows by key: Σclicks, Σimpressions, impression-weighted position, recomputed CTR. */
export function aggregateDimRows(rows: DimMetricRow[]): DimAgg[] {
  const acc = new Map<string, { clicks: number; impressions: number; pairs: Array<[number, number]> }>();
  for (const r of rows) {
    let a = acc.get(r.key);
    if (!a) {
      a = { clicks: 0, impressions: 0, pairs: [] };
      acc.set(r.key, a);
    }
    a.clicks += r.clicks;
    a.impressions += r.impressions;
    a.pairs.push([r.position, r.impressions]);
  }
  const out: DimAgg[] = [];
  for (const [key, a] of acc) {
    out.push({ key, clicks: a.clicks, impressions: a.impressions, ctr: ctr(a.clicks, a.impressions), position: weightedMean(a.pairs) });
  }
  out.sort((x, y) => y.clicks - x.clicks || y.impressions - x.impressions);
  return out;
}

export type QueryAgg = DimAgg & { prevClicks: number | null; prevPosition: number | null; prevImpressions: number | null };

/** Attach previous-period aggregates per key (null when the key had no previous data). */
export function mergeCurrentPrevious(current: DimAgg[], previous: DimAgg[]): QueryAgg[] {
  const prevByKey = new Map(previous.map((p) => [p.key, p]));
  return current.map((c) => {
    const p = prevByKey.get(c.key);
    return {
      ...c,
      prevClicks: p ? p.clicks : null,
      prevPosition: p ? p.position : null,
      prevImpressions: p ? p.impressions : null,
    };
  });
}

export type KeywordViewKey = "all" | "striking" | "low_ctr" | "declining" | "rising";

export const KEYWORD_VIEWS: KeywordViewKey[] = ["all", "striking", "low_ctr", "declining", "rising"];

export function normalizeKeywordView(raw: string | undefined): KeywordViewKey {
  return (KEYWORD_VIEWS as string[]).includes(raw ?? "") ? (raw as KeywordViewKey) : "all";
}

export type KeywordTableRow = {
  key: string;
  clicks: number;
  impressions: number;
  ctr: number;
  position: number;
  prevClicks: number | null;
  prevPosition: number | null;
  expectedCtr: number | null;
  potentialClicks: number | null;
  bucket: PositionBucket;
};

export type KeywordOpportunities = {
  striking: Opportunity[];
  low_ctr: Opportunity[];
  declining: Opportunity[];
  rising: Opportunity[];
};

/** Build serializable table rows for the keyword explorer for the given view. */
export function buildKeywordRows(rows: QueryAgg[], opportunities: KeywordOpportunities, view: KeywordViewKey): KeywordTableRow[] {
  const byKey = new Map(rows.map((r) => [r.key, r]));
  if (view === "all") {
    return rows.map((r) => ({
      key: r.key,
      clicks: r.clicks,
      impressions: r.impressions,
      ctr: r.ctr,
      position: r.position,
      prevClicks: r.prevClicks,
      prevPosition: r.prevPosition,
      expectedCtr: expectedCtr(r.position),
      potentialClicks: null,
      bucket: positionBucket(r.position),
    }));
  }
  const opps = opportunities[view];
  return opps.map((o) => {
    const full = byKey.get(o.key);
    return {
      key: o.key,
      clicks: o.clicks,
      impressions: o.impressions,
      ctr: o.ctr,
      position: o.position,
      prevClicks: o.prevClicks ?? full?.prevClicks ?? null,
      prevPosition: full?.prevPosition ?? null,
      expectedCtr: o.expectedCtr ?? expectedCtr(o.position),
      potentialClicks: o.potentialClicks,
      bucket: positionBucket(o.position),
    };
  });
}

/** Strip the origin from a full URL for compact display ("/menu" instead of "https://…/menu"). */
export function urlPath(fullUrl: string): string {
  try {
    const u = new URL(fullUrl);
    const p = `${u.pathname}${u.search}`;
    return p === "" ? "/" : p;
  } catch {
    return fullUrl;
  }
}

/** ISO3 (GSC country keys) → Indonesian display names for common countries. */
const COUNTRY_NAMES: Record<string, string> = {
  idn: "Indonesia",
  sgp: "Singapura",
  mys: "Malaysia",
  aus: "Australia",
  usa: "Amerika Serikat",
  gbr: "Inggris",
  jpn: "Jepang",
  kor: "Korea Selatan",
  ind: "India",
  nld: "Belanda",
  deu: "Jerman",
  chn: "Tiongkok",
  tha: "Thailand",
  vnm: "Vietnam",
  phl: "Filipina",
  are: "Uni Emirat Arab",
  sau: "Arab Saudi",
  twn: "Taiwan",
  hkg: "Hong Kong",
  can: "Kanada",
};

export function countryLabel(key: string): string {
  return COUNTRY_NAMES[key.toLowerCase()] ?? key.toUpperCase();
}
