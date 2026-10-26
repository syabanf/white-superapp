import { formatCompact, formatNumber, formatPercent } from "@/lib/format";
import { computeDelta, type Delta } from "@/lib/metrics";
import { formatRupiah, parseMoney } from "@/features/erp/shared/money";

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];

function ymd(day: string): [number, number, number] {
  const [y, m, d] = day.split("-").map(Number);
  return [y!, m!, d!];
}

/** "2026-10-05" → "5 Okt" */
export function dayLabel(day: string): string {
  const [, m, d] = ymd(day);
  return `${d} ${MONTHS[m - 1]}`;
}

/** "2026-10-05" → "5 Okt 2026" */
export function dayLabelLong(day: string): string {
  const [y, m, d] = ymd(day);
  return `${d} ${MONTHS[m - 1]} ${y}`;
}

/** "2026-10-01" → "Okt 26" */
export function monthLabel(day: string): string {
  const [y, m] = ymd(day);
  return `${MONTHS[m - 1]} ${String(y).slice(2)}`;
}

export function bucketLabel(day: string, bucket: "day" | "week" | "month"): string {
  return bucket === "month" ? monthLabel(day) : dayLabel(day);
}

/** "1 Okt – 26 Okt 2026" */
export function rangeLabel(from: string, to: string): string {
  return from.slice(0, 4) === to.slice(0, 4) ? `${dayLabel(from)} – ${dayLabelLong(to)}` : `${dayLabelLong(from)} – ${dayLabelLong(to)}`;
}

/** Compact rupiah for tiles and axes: "Rp 2,1 M", "Rp 450 jt". */
export function rpCompact(v: string | number): string {
  return `Rp ${formatCompact(parseMoney(v))}`;
}

export const rp = (v: string | number) => formatRupiah(v);
export const pct1 = (v: number) => formatPercent(v, 1);
export const num0 = (v: number) => formatNumber(v);
export const num1 = (v: number) => formatNumber(v, 1);

/** Delta for a tile; none when there is nothing to compare against. */
export function delta(cur: number, prev: number): Delta | null {
  return prev === 0 ? null : computeDelta(cur, prev);
}
