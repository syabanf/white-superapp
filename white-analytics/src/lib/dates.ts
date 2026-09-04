/**
 * Date-range helpers. All "days" are UTC-midnight Date objects (matches Prisma @db.Date).
 * URL representation: YYYY-MM-DD.
 */

export type DateRange = { from: Date; to: Date };

export type PresetKey = "7d" | "14d" | "28d" | "30d" | "90d" | "thisMonth" | "lastMonth" | "12m";

export const PRESETS: { key: PresetKey; label: string }[] = [
  { key: "7d", label: "7 hari terakhir" },
  { key: "14d", label: "14 hari terakhir" },
  { key: "28d", label: "28 hari terakhir" },
  { key: "30d", label: "30 hari terakhir" },
  { key: "90d", label: "90 hari terakhir" },
  { key: "thisMonth", label: "Bulan ini" },
  { key: "lastMonth", label: "Bulan lalu" },
  { key: "12m", label: "12 bulan terakhir" },
];

export const DEFAULT_PRESET: PresetKey = "28d";

/** UTC midnight for a given y/m/d */
export function utcDate(y: number, m0: number, d: number): Date {
  return new Date(Date.UTC(y, m0, d));
}

/** Today's date at UTC midnight (based on the given "now"). */
export function todayUtc(now: Date = new Date()): Date {
  return utcDate(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate());
}

export function addDays(d: Date, n: number): Date {
  return utcDate(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate() + n);
}

export function startOfMonthUtc(d: Date): Date {
  return utcDate(d.getUTCFullYear(), d.getUTCMonth(), 1);
}

export function endOfMonthUtc(d: Date): Date {
  return utcDate(d.getUTCFullYear(), d.getUTCMonth() + 1, 0);
}

/** Inclusive day count between from and to. */
export function dayCount(r: DateRange): number {
  return Math.round((r.to.getTime() - r.from.getTime()) / 86_400_000) + 1;
}

/** Yesterday is the latest complete day for analytics APIs (GSC lags ~2 days, but we keep it simple). */
export function latestCompleteDay(now: Date = new Date()): Date {
  return addDays(todayUtc(now), -1);
}

export function rangeForPreset(key: PresetKey, now: Date = new Date()): DateRange {
  const end = latestCompleteDay(now);
  switch (key) {
    case "7d":
      return { from: addDays(end, -6), to: end };
    case "14d":
      return { from: addDays(end, -13), to: end };
    case "28d":
      return { from: addDays(end, -27), to: end };
    case "30d":
      return { from: addDays(end, -29), to: end };
    case "90d":
      return { from: addDays(end, -89), to: end };
    case "thisMonth": {
      const t = todayUtc(now);
      return { from: startOfMonthUtc(t), to: end < startOfMonthUtc(t) ? t : end };
    }
    case "lastMonth": {
      const t = todayUtc(now);
      const lastMonth = utcDate(t.getUTCFullYear(), t.getUTCMonth() - 1, 1);
      return { from: lastMonth, to: endOfMonthUtc(lastMonth) };
    }
    case "12m":
      return { from: addDays(utcDate(end.getUTCFullYear() - 1, end.getUTCMonth(), end.getUTCDate()), 1), to: end };
  }
}

/** The immediately preceding period of the same length. */
export function previousRange(r: DateRange): DateRange {
  const n = dayCount(r);
  const to = addDays(r.from, -1);
  return { from: addDays(to, -(n - 1)), to };
}

export function toISODate(d: Date): string {
  return d.toISOString().slice(0, 10);
}

export function parseISODate(s: string | null | undefined): Date | null {
  if (!s) return null;
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(s);
  if (!m) return null;
  const d = utcDate(Number(m[1]), Number(m[2]) - 1, Number(m[3]));
  return Number.isNaN(d.getTime()) ? null : d;
}

export type RangeSearchParams = {
  from?: string | string[];
  to?: string | string[];
  preset?: string | string[];
  compare?: string | string[];
};

export type ResolvedRange = {
  range: DateRange;
  previous: DateRange;
  preset: PresetKey | null;
  compare: boolean;
};

function first(v: string | string[] | undefined): string | undefined {
  return Array.isArray(v) ? v[0] : v;
}

/**
 * Resolve the global date range from URL search params.
 * Priority: explicit from/to → preset → default preset (28d).
 * compare defaults to true.
 */
export function resolveRange(sp: RangeSearchParams, now: Date = new Date()): ResolvedRange {
  const compareRaw = first(sp.compare);
  const compare = compareRaw == null ? true : compareRaw !== "0" && compareRaw !== "false";

  const from = parseISODate(first(sp.from));
  const to = parseISODate(first(sp.to));
  if (from && to && from <= to) {
    const range = { from, to };
    return { range, previous: previousRange(range), preset: null, compare };
  }

  const presetRaw = first(sp.preset) as PresetKey | undefined;
  const preset: PresetKey = presetRaw && PRESETS.some((p) => p.key === presetRaw) ? presetRaw : DEFAULT_PRESET;
  const range = rangeForPreset(preset, now);
  return { range, previous: previousRange(range), preset, compare };
}

/** Enumerate each day in range (inclusive). */
export function eachDay(r: DateRange): Date[] {
  const out: Date[] = [];
  for (let d = r.from; d <= r.to; d = addDays(d, 1)) out.push(d);
  return out;
}

/** Bucket key helper for daily series. */
export function dayKey(d: Date): string {
  return toISODate(d);
}

/** Choose a sensible tick interval for N daily points. */
export function tickEvery(n: number): number {
  if (n <= 14) return 1;
  if (n <= 31) return 3;
  if (n <= 62) return 7;
  if (n <= 120) return 14;
  return 30;
}
