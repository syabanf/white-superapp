import { addDays, dayKey, eachDay, type DateRange } from "@/lib/dates";

export type DailyPoint = { date: string; [k: string]: number | string | null };

/**
 * Fill a daily series so every day in range has a point (missing days → 0 or null).
 * `rows` may contain multiple entries per day; they are summed per numeric field.
 */
export function fillDaily<T extends Record<string, unknown>>(
  range: DateRange,
  rows: T[],
  getDate: (row: T) => Date,
  fields: (keyof T & string)[],
  opts: { missing?: 0 | null } = {},
): DailyPoint[] {
  const missing = opts.missing ?? 0;
  const map = new Map<string, Record<string, number>>();
  for (const r of rows) {
    const k = dayKey(getDate(r));
    let acc = map.get(k);
    if (!acc) {
      acc = {};
      map.set(k, acc);
    }
    for (const f of fields) {
      const v = r[f];
      acc[f] = (acc[f] ?? 0) + (typeof v === "number" ? v : 0);
    }
  }
  return eachDay(range).map((d) => {
    const k = dayKey(d);
    const acc = map.get(k);
    const point: DailyPoint = { date: k };
    for (const f of fields) point[f] = acc ? (acc[f] ?? 0) : missing;
    return point;
  });
}

/** Simple sparkline extraction: last N values of a field */
export function sparkline(points: DailyPoint[], field: string, n = 14): number[] {
  return points.slice(-n).map((p) => (typeof p[field] === "number" ? (p[field] as number) : 0));
}

/** Rolling mean smoothing (window w) — for noisy daily series */
export function rollingMean(values: number[], w = 7): number[] {
  const out: number[] = [];
  for (let i = 0; i < values.length; i++) {
    const start = Math.max(0, i - w + 1);
    const slice = values.slice(start, i + 1);
    out.push(slice.reduce((a, b) => a + b, 0) / slice.length);
  }
  return out;
}

/** Align a previous-period series onto the current period's dates for overlay charts */
export function alignPrevious(current: DailyPoint[], previous: DailyPoint[], field: string, outField = `${field}Prev`): DailyPoint[] {
  return current.map((p, i) => ({ ...p, [outField]: previous[i] ? (previous[i][field] as number | null) : null }));
}

/** Split an inclusive range into weekly buckets (Mon-start) — returns bucket start ISO keys per day */
export function weekKey(d: Date): string {
  const day = (d.getUTCDay() + 6) % 7; // Mon=0
  return dayKey(addDays(d, -day));
}
