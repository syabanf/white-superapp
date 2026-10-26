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

export type ChangeTone = "positive" | "negative" | "neutral";

/**
 * Good/bad/neutral tone of a change. Accepts a Delta or a signed number
 * (e.g. a net figure or an absolute difference). Respects lower-is-better
 * metrics; flat, null and NaN are neutral.
 */
export function changeTone(change: Delta | number | null | undefined, lowerIsBetter = false): ChangeTone {
  if (change == null) return "neutral";
  if (typeof change === "number") {
    if (!Number.isFinite(change) || Math.abs(change) < 1e-9) return "neutral";
    return change > 0 !== lowerIsBetter ? "positive" : "negative";
  }
  const good = isGoodChange(change, lowerIsBetter);
  return good === null ? "neutral" : good ? "positive" : "negative";
}

const TONE_VAR: Record<ChangeTone, string> = {
  positive: "var(--positive)",
  negative: "var(--negative)",
  neutral: "var(--muted-foreground)",
};

const TONE_TEXT: Record<ChangeTone, string> = {
  positive: "text-positive",
  negative: "text-negative",
  neutral: "text-muted-foreground",
};

/** CSS color for a change (text, strokes, signed bars). */
export function trendColor(change: Delta | number | null | undefined, lowerIsBetter = false): string {
  return TONE_VAR[changeTone(change, lowerIsBetter)];
}

/** Tailwind text class for a change. */
export function trendTextClass(change: Delta | number | null | undefined, lowerIsBetter = false): string {
  return TONE_TEXT[changeTone(change, lowerIsBetter)];
}

/** Delta between the first and last finite value of a series (sparklines, trend lines). */
export function seriesDelta(values: ReadonlyArray<number | null | undefined>): Delta | null {
  const finite = values.filter((v): v is number => typeof v === "number" && Number.isFinite(v));
  if (finite.length < 2) return null;
  return computeDelta(finite.at(-1)!, finite[0]!);
}
