import type { DashboardOffer, RecruitmentTrendPoint } from "./types";

/** Period and filters the dashboard was computed for; drill-down links carry them. */
export type DrillContext = {
  from: string;
  to: string;
  compare?: string;
  job?: string;
  department?: string;
  source?: string;
  stage?: string;
};

const BASE = "/erp/hr/recruitment";

function href(path: string, params: Record<string, string | undefined>): string {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) if (v) sp.set(k, v);
  const q = sp.toString();
  return q ? `${BASE}${path}?${q}` : `${BASE}${path}`;
}

/** Candidate list for a slice: the list filters by status, opening, source and application date. */
export function candidatesHref(ctx: DrillContext, p: { status?: string; job?: string; source?: string } = {}): string {
  return href("/candidates", { status: p.status ?? ctx.stage, job: p.job ?? ctx.job, source: p.source ?? ctx.source, from: ctx.from, to: ctx.to });
}

export function interviewsHref(ctx: DrillContext, status?: string): string {
  return href("/interviews", { status, from: ctx.from, to: ctx.to });
}

/** This dashboard narrowed to a slice, e.g. one source on the Sumber tab. */
export function dashboardHref(ctx: DrillContext, patch: Partial<DrillContext> & { tab?: string }): string {
  return href("", { ...ctx, ...patch });
}

export function candidateHref(id: string, tab?: string): string {
  return href(`/candidates/${id}`, { tab });
}

/** Current weeks with the comparison weeks laid over by position. */
export function overlay<K extends keyof RecruitmentTrendPoint>(
  trend: RecruitmentTrendPoint[],
  previous: RecruitmentTrendPoint[],
  key: K,
): { date: string; value: number; previous: number | null }[] {
  return trend.map((p, i) => ({
    date: p.start,
    value: Number(p[key] ?? 0),
    previous: previous[i] ? Number(previous[i]![key] ?? 0) : null,
  }));
}

/** Weekly acceptance rate (null in weeks without a decided offer). */
export function acceptanceSeries(trend: RecruitmentTrendPoint[]): number[] {
  return trend.map((p) => {
    const decided = p.offersAccepted + p.offersDeclined + p.offersExpired;
    return decided > 0 ? (p.offersAccepted / decided) * 100 : 0;
  });
}

/** Keeps the top n items by value and folds the rest into one "others" row. */
export function topN<T>(items: T[], n: number, value: (t: T) => number, others: (rest: T[]) => T): T[] {
  const sorted = [...items].sort((a, b) => value(b) - value(a));
  if (sorted.length <= n + 1) return sorted;
  return [...sorted.slice(0, n), others(sorted.slice(n))];
}

/** Response days of decided offers in four buckets: ≤1, 1-3, 3-7, >7. */
export function responseBuckets(offers: DashboardOffer[]): number[] {
  const out = [0, 0, 0, 0];
  for (const o of offers) {
    if (o.responseDays == null || (o.status !== "accepted" && o.status !== "declined")) continue;
    const d = o.responseDays;
    out[d <= 1 ? 0 : d <= 3 ? 1 : d <= 7 ? 2 : 3]! += 1;
  }
  return out;
}

export function rate(part: number, whole: number): number | null {
  return whole > 0 ? (part / whole) * 100 : null;
}

export function mean(values: number[]): number | null {
  return values.length ? values.reduce((a, b) => a + b, 0) / values.length : null;
}
