/**
 * Period-over-period delta. The single source of truth for every "vs periode sebelumnya"
 * badge in the app — never hard-code trend percentages.
 */

export type Delta = {
  /** current − previous (absolute) */
  abs: number;
  /** (current − previous) / |previous| × 100 ; null when previous is 0/absent */
  pct: number | null;
  /** direction of change */
  direction: "up" | "down" | "flat";
};

export function computeDelta(current: number, previous: number | null | undefined): Delta {
  const prev = previous ?? 0;
  const abs = current - prev;
  const pct = prev === 0 ? null : (abs / Math.abs(prev)) * 100;
  const direction: Delta["direction"] = Math.abs(abs) < 1e-9 ? "flat" : abs > 0 ? "up" : "down";
  return { abs, pct, direction };
}

/**
 * Whether the change is good news. Some metrics are "lower is better" (CPR, CPC, position, LCP).
 */
export function isGoodChange(delta: Delta, lowerIsBetter = false): boolean | null {
  if (delta.direction === "flat") return null;
  const up = delta.direction === "up";
  return lowerIsBetter ? !up : up;
}

/** Sum a numeric field over rows */
export function sumBy<T>(rows: T[], pick: (row: T) => number | null | undefined): number {
  let s = 0;
  for (const r of rows) s += pick(r) ?? 0;
  return s;
}

/** Arithmetic mean; returns 0 for empty */
export function mean(values: number[]): number {
  if (values.length === 0) return 0;
  return values.reduce((a, b) => a + b, 0) / values.length;
}

/** Safe division: returns 0 when denominator is 0 */
export function safeDiv(numer: number, denom: number): number {
  return denom === 0 ? 0 : numer / denom;
}

/** Weighted mean: Σ(value×weight)/Σweight */
export function weightedMean(pairs: Array<[value: number, weight: number]>): number {
  let num = 0;
  let den = 0;
  for (const [v, w] of pairs) {
    num += v * w;
    den += w;
  }
  return den === 0 ? 0 : num / den;
}

export function clamp(v: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, v));
}

export function round(v: number, digits = 2): number {
  const f = 10 ** digits;
  return Math.round(v * f) / f;
}
