import { computeDelta, type Delta } from "@/lib/metrics/delta";
import { formatCurrency, formatNumber, formatPercent } from "@/lib/format";
import { formatRupiah } from "@/features/erp/shared/money";
import type { AnalyticsKpi } from "./types";

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];

/** Decimal string → number; "" (not computable) → null. */
export function num(v: string | undefined | null): number | null {
  if (v == null || v === "") return null;
  const n = Number(v);
  return Number.isFinite(n) ? n : null;
}

export const nums = (vs: string[]) => vs.map((v) => num(v) ?? 0);

/** "2026-10" → "Okt 26" */
export function monthShort(ym: string): string {
  const [y, m] = ym.split("-").map(Number);
  return `${MONTHS[(m ?? 1) - 1]} ${String(y).slice(2)}`;
}

/** First and last day of a "YYYY-MM" month. */
export function monthBounds(ym: string): { from: string; to: string } {
  const [y, m] = ym.split("-").map(Number) as [number, number];
  const last = new Date(Date.UTC(y, m, 0)).getUTCDate();
  return { from: `${ym}-01`, to: `${ym}-${String(last).padStart(2, "0")}` };
}

export const compactIdr = (v: number | null) => (v == null ? "–" : formatCurrency(v, "IDR", { compact: true }));
export const exactIdr = (v: number | null) => (v == null ? "–" : formatRupiah(v));

export function delta(cur: number | null, prev: number | null): Delta | null {
  if (cur == null || prev == null) return null;
  return computeDelta(cur, prev);
}

export type KpiUnit = "idr" | "pct" | "days" | "months" | "count";

export function formatUnit(v: number | null, unit: KpiUnit): string {
  if (v == null) return "–";
  switch (unit) {
    case "idr":
      return compactIdr(v);
    case "pct":
      return formatPercent(v, 1);
    case "days":
      return `${formatNumber(v, 0)} hari`;
    case "months":
      return `${formatNumber(v, 1)} bln`;
    default:
      return formatNumber(v, 0);
  }
}

export function kpiMap(kpis: AnalyticsKpi[]): Record<string, AnalyticsKpi> {
  return Object.fromEntries(kpis.map((k) => [k.key, k]));
}

/** Search params for list pages that take a date range. */
export function rangeQs(from: string, to: string, extra: Record<string, string | undefined> = {}): string {
  const sp = new URLSearchParams({ from, to });
  for (const [k, v] of Object.entries(extra)) if (v) sp.set(k, v);
  return sp.toString();
}

export const ledgerHref = (accountId: string, from: string, to: string) =>
  `/erp/finance/reports?r=general-ledger&${rangeQs(from, to, { account: accountId })}`;

export const journalsHref = (from: string, to: string, q?: string) => `/erp/finance/journals?${rangeQs(from, to, { q })}`;
