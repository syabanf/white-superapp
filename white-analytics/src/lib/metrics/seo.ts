import { safeDiv, weightedMean, clamp } from "./delta";

export type SearchRow = {
  clicks: number;
  impressions: number;
  /** average position (1 = top) */
  position: number;
  /** ctr as percentage 0..100 (optional; recomputed from clicks/impressions when aggregating) */
  ctr?: number;
};

/** CTR percentage = clicks / impressions × 100 */
export function ctr(clicks: number, impressions: number): number {
  return safeDiv(clicks, impressions) * 100;
}

/** Aggregate many rows: sums + impression-weighted average position */
export function aggregateSearch(rows: SearchRow[]): { clicks: number; impressions: number; ctr: number; position: number } {
  let clicks = 0;
  let impressions = 0;
  const pairs: Array<[number, number]> = [];
  for (const r of rows) {
    clicks += r.clicks;
    impressions += r.impressions;
    pairs.push([r.position, r.impressions]);
  }
  return { clicks, impressions, ctr: ctr(clicks, impressions), position: weightedMean(pairs) };
}

export type PositionBucket = "1-3" | "4-10" | "11-20" | "21+";
export const POSITION_BUCKETS: PositionBucket[] = ["1-3", "4-10", "11-20", "21+"];

export function positionBucket(position: number): PositionBucket {
  if (position <= 3) return "1-3";
  if (position <= 10) return "4-10";
  if (position <= 20) return "11-20";
  return "21+";
}

/** Count rows (queries) per position bucket, weighted by nothing (count) and by clicks */
export function positionDistribution(rows: SearchRow[]): { bucket: PositionBucket; queries: number; clicks: number; impressions: number }[] {
  const acc = new Map<PositionBucket, { queries: number; clicks: number; impressions: number }>();
  for (const b of POSITION_BUCKETS) acc.set(b, { queries: 0, clicks: 0, impressions: 0 });
  for (const r of rows) {
    const b = acc.get(positionBucket(r.position))!;
    b.queries += 1;
    b.clicks += r.clicks;
    b.impressions += r.impressions;
  }
  return POSITION_BUCKETS.map((bucket) => ({ bucket, ...acc.get(bucket)! }));
}

/**
 * Expected organic CTR curve by rounded position (industry benchmark, percentage).
 * Positions ≥ 11 taper toward ~1%.
 */
export const EXPECTED_CTR_BY_POSITION: Record<number, number> = {
  1: 28,
  2: 15,
  3: 11,
  4: 8,
  5: 7,
  6: 5,
  7: 4,
  8: 3,
  9: 3,
  10: 2.5,
};

export function expectedCtr(position: number): number {
  const p = Math.max(1, Math.round(position));
  if (p <= 10) return EXPECTED_CTR_BY_POSITION[p]!;
  if (p <= 20) return 1.5;
  return 1;
}

export type OpportunityKind = "striking_distance" | "low_ctr" | "declining" | "rising";

export type OpportunityRule = {
  /** minimum impressions in current period for a query to be considered */
  minImpressions: number;
  /** striking distance position window (inclusive) */
  strikingMin: number;
  strikingMax: number;
  /** low-CTR: actual CTR below this fraction of expected CTR at that position */
  lowCtrFraction: number;
  /** declining: clicks drop ≥ this fraction vs previous period */
  decliningDrop: number;
  /** rising: clicks grow ≥ this fraction vs previous period */
  risingGrowth: number;
};

export const DEFAULT_OPPORTUNITY_RULE: OpportunityRule = {
  minImpressions: 100,
  strikingMin: 4,
  strikingMax: 15,
  lowCtrFraction: 0.6,
  decliningDrop: 0.3,
  risingGrowth: 0.3,
};

export type QueryWithPrev = SearchRow & { key: string; prevClicks?: number | null; prevPosition?: number | null };

export type Opportunity = {
  kind: OpportunityKind;
  key: string;
  clicks: number;
  impressions: number;
  position: number;
  ctr: number;
  expectedCtr?: number;
  prevClicks?: number | null;
  /** potential extra clicks if CTR reached expected (low_ctr) or position improved to 3 (striking) */
  potentialClicks: number;
};

/** "Striking distance": positions 4–15 with meaningful impressions — one push from page-1 top. */
export function strikingDistance(rows: QueryWithPrev[], rule = DEFAULT_OPPORTUNITY_RULE): Opportunity[] {
  return rows
    .filter((r) => r.impressions >= rule.minImpressions && r.position >= rule.strikingMin && r.position <= rule.strikingMax)
    .map((r) => {
      const c = ctr(r.clicks, r.impressions);
      const targetCtr = expectedCtr(3);
      return {
        kind: "striking_distance" as const,
        key: r.key,
        clicks: r.clicks,
        impressions: r.impressions,
        position: r.position,
        ctr: c,
        expectedCtr: targetCtr,
        potentialClicks: Math.max(0, Math.round((r.impressions * targetCtr) / 100 - r.clicks)),
      };
    })
    .sort((a, b) => b.potentialClicks - a.potentialClicks);
}

/** "Low CTR": already on page 1 (≤10) but CTR well below the expected curve → title/meta rewrite. */
export function lowCtr(rows: QueryWithPrev[], rule = DEFAULT_OPPORTUNITY_RULE): Opportunity[] {
  return rows
    .filter((r) => r.impressions >= rule.minImpressions && r.position <= 10)
    .map((r) => {
      const c = ctr(r.clicks, r.impressions);
      const exp = expectedCtr(r.position);
      return { r, c, exp };
    })
    .filter(({ c, exp }) => c < exp * rule.lowCtrFraction)
    .map(({ r, c, exp }) => ({
      kind: "low_ctr" as const,
      key: r.key,
      clicks: r.clicks,
      impressions: r.impressions,
      position: r.position,
      ctr: c,
      expectedCtr: exp,
      potentialClicks: Math.max(0, Math.round((r.impressions * exp) / 100 - r.clicks)),
    }))
    .sort((a, b) => b.potentialClicks - a.potentialClicks);
}

/** "Declining": clicks dropped ≥ decliningDrop vs previous period. */
export function declining(rows: QueryWithPrev[], rule = DEFAULT_OPPORTUNITY_RULE): Opportunity[] {
  return rows
    .filter((r) => (r.prevClicks ?? 0) > 0 && r.impressions >= rule.minImpressions)
    .filter((r) => (r.prevClicks! - r.clicks) / r.prevClicks! >= rule.decliningDrop)
    .map((r) => ({
      kind: "declining" as const,
      key: r.key,
      clicks: r.clicks,
      impressions: r.impressions,
      position: r.position,
      ctr: ctr(r.clicks, r.impressions),
      prevClicks: r.prevClicks,
      potentialClicks: Math.max(0, (r.prevClicks ?? 0) - r.clicks),
    }))
    .sort((a, b) => b.potentialClicks - a.potentialClicks);
}

/** "Rising": clicks grew ≥ risingGrowth vs previous period. */
export function rising(rows: QueryWithPrev[], rule = DEFAULT_OPPORTUNITY_RULE): Opportunity[] {
  return rows
    .filter((r) => (r.prevClicks ?? 0) > 0 && r.impressions >= rule.minImpressions)
    .filter((r) => (r.clicks - r.prevClicks!) / r.prevClicks! >= rule.risingGrowth)
    .map((r) => ({
      kind: "rising" as const,
      key: r.key,
      clicks: r.clicks,
      impressions: r.impressions,
      position: r.position,
      ctr: ctr(r.clicks, r.impressions),
      prevClicks: r.prevClicks,
      potentialClicks: r.clicks - (r.prevClicks ?? 0),
    }))
    .sort((a, b) => b.potentialClicks - a.potentialClicks);
}

// ── Core Web Vitals & audit health ────────────────────────────

export type CwvRating = "good" | "needs-improvement" | "poor";

export const CWV_THRESHOLDS = {
  lcpMs: { good: 2500, poor: 4000 },
  inpMs: { good: 200, poor: 500 },
  cls: { good: 0.1, poor: 0.25 },
  fcpMs: { good: 1800, poor: 3000 },
  ttfbMs: { good: 800, poor: 1800 },
  tbtMs: { good: 200, poor: 600 },
} as const;

export type CwvMetric = keyof typeof CWV_THRESHOLDS;

export function rateCwv(metric: CwvMetric, value: number | null | undefined): CwvRating | null {
  if (value == null) return null;
  const t = CWV_THRESHOLDS[metric];
  if (value <= t.good) return "good";
  if (value <= t.poor) return "needs-improvement";
  return "poor";
}

/** Lighthouse-style score rating (0–100) */
export function rateScore(score: number): CwvRating {
  if (score >= 90) return "good";
  if (score >= 50) return "needs-improvement";
  return "poor";
}

export type AuditScores = { performance: number; seo: number; accessibility: number; bestPractices: number };

/**
 * Health score 0–100: weighted Lighthouse categories (perf 0.3, seo 0.4, a11y 0.15, BP 0.15)
 * minus penalties for crawl issues (ERROR −2 each up to −20, WARNING −0.5 each up to −10).
 */
export function healthScore(scores: AuditScores | null, issues: { errors: number; warnings: number }): number | null {
  if (!scores) return null;
  const base = scores.performance * 0.3 + scores.seo * 0.4 + scores.accessibility * 0.15 + scores.bestPractices * 0.15;
  const penalty = clamp(issues.errors * 2, 0, 20) + clamp(issues.warnings * 0.5, 0, 10);
  return Math.round(clamp(base - penalty, 0, 100));
}
