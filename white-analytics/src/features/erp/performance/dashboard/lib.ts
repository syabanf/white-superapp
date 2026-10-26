import { computeDelta, type Delta } from "@/lib/metrics";
import { formatDecimal, formatNumber } from "@/lib/format";
import type { DashMetric, DashWindow } from "./types";

const SHORT = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];

/** "2026-10" → "Okt 26" */
export function monthTick(ym: string): string {
  const [y, m] = ym.split("-").map(Number);
  return `${SHORT[(m ?? 1) - 1]} ${String(y).slice(2)}`;
}

/** "2026-10-05" → "5 Okt" */
export function dayTick(day: string): string {
  const [, m, d] = day.split("-").map(Number);
  return `${d} ${SHORT[(m ?? 1) - 1]}`;
}

/** Bucket key label for the window's granularity. */
export function bucketTick(key: string, granularity: DashWindow["granularity"]): string {
  return granularity === "month" ? monthTick(key.slice(0, 7)) : dayTick(key);
}

export function bucketLabel(key: string, granularity: DashWindow["granularity"]): string {
  if (granularity === "month") return monthTick(key.slice(0, 7));
  if (granularity === "week") return `Minggu ${dayTick(key)}`;
  return dayTick(key);
}

/** Fraction 0-1 → "83%" (one decimal under 10%). */
export function pct(v: number | null | undefined, digits = 0): string {
  if (v == null || Number.isNaN(v)) return "–";
  return `${formatNumber(v * 100, digits)}%`;
}

export const score = (v: number | null | undefined) => formatDecimal(v ?? null, 1);
export const int = (v: number | null | undefined) => formatNumber(v ?? null);

/** Delta for a tile; null without both values. */
export function metricDelta(m: DashMetric): Delta | null {
  if (m.value == null || m.previous == null) return null;
  return computeDelta(m.value, m.previous);
}

export function ratio(num: number, den: number): number | undefined {
  return den > 0 ? num / den : undefined;
}

/** "2026-10-05" → "2026-10" */
export const monthOf = (day: string) => day.slice(0, 7);

/** Last day of a bucket clipped to `to`. */
export function bucketEnd(key: string, granularity: DashWindow["granularity"], to: string): string {
  const d = new Date(`${key}T00:00:00Z`);
  if (granularity === "week") d.setUTCDate(d.getUTCDate() + 6);
  if (granularity === "month") d.setUTCMonth(d.getUTCMonth() + 1, 0);
  const end = d.toISOString().slice(0, 10);
  return end < to ? end : to;
}

/** Builds a drill-down href, dropping empty params. */
export function href(path: string, params: Record<string, string | undefined | null>): string {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) if (v) sp.set(k, v);
  const s = sp.toString();
  return s ? `${path}?${s}` : path;
}

/** Score delta on one decimal, so a rounding speck reads as flat. */
export function scoreDelta(cur: number, prev: number): Delta {
  return computeDelta(Math.round(cur * 10) / 10, Math.round(prev * 10) / 10);
}

/** Score axis from the nearest ten below the lowest value up to 100. */
export function scoreDomain(values: (number | undefined)[]): [number, number] {
  const v = values.filter((x): x is number => x != null);
  if (v.length === 0) return [0, 100];
  return [Math.max(0, Math.floor(Math.min(...v) / 10) * 10), Math.max(100, Math.ceil(Math.max(...v)))];
}
