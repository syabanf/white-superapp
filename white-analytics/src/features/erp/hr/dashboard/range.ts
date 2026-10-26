/**
 * Period filter for the HR dashboard. All state lives in the URL:
 * `preset` (thisMonth | lastMonth | quarter | ytd | 12m), or `from`/`to`
 * for a custom range, and `compare` (previous | year). Dates are WIB
 * calendar days ("YYYY-MM-DD").
 */

export const HR_PRESETS = ["thisMonth", "lastMonth", "quarter", "ytd", "12m"] as const;
export type HrPreset = (typeof HR_PRESETS)[number];
export type HrCompare = "previous" | "year";

export type HrRange = {
  preset: HrPreset | null;
  from: string;
  to: string;
  compare: HrCompare;
};

const ISO = /^\d{4}-\d{2}-\d{2}$/;

function iso(y: number, m0: number, d: number): string {
  const t = new Date(Date.UTC(y, m0, d));
  return t.toISOString().slice(0, 10);
}

function parts(day: string): [number, number, number] {
  const [y, m, d] = day.split("-").map(Number);
  return [y!, m! - 1, d!];
}

export function presetRange(preset: HrPreset, today: string): { from: string; to: string } {
  const [y, m, d] = parts(today);
  switch (preset) {
    case "lastMonth":
      return { from: iso(y, m - 1, 1), to: iso(y, m, 0) };
    case "quarter":
      return { from: iso(y, m - (m % 3), 1), to: today };
    case "ytd":
      return { from: iso(y, 0, 1), to: today };
    case "12m":
      return { from: iso(y, m - 11, 1), to: today };
    default:
      return { from: iso(y, m, 1), to: iso(y, m, d) };
  }
}

/** Resolves the URL params; an invalid custom range falls back to this month. */
export function resolveHrRange(
  sp: { preset?: string; from?: string; to?: string; compare?: string },
  today: string,
): HrRange {
  const compare: HrCompare = sp.compare === "year" ? "year" : "previous";
  if (sp.from && sp.to && ISO.test(sp.from) && ISO.test(sp.to) && sp.from <= sp.to) {
    return { preset: null, from: sp.from, to: sp.to, compare };
  }
  const preset = HR_PRESETS.find((p) => p === sp.preset) ?? "thisMonth";
  return { preset, ...presetRange(preset, today), compare };
}

/** "2026-10-01" → "2026-10" */
export const ymOf = (day: string) => day.slice(0, 7);

/** Last day of the month of `day`, capped at `cap`. */
export function monthEndOf(day: string, cap?: string): string {
  const [y, m] = parts(day);
  const end = iso(y, m + 1, 0);
  return cap && cap < end ? cap : end;
}

/** Last day of a dashboard bucket starting at `start`, capped at `cap`. */
export function bucketEndOf(start: string, bucket: "day" | "week" | "month", cap: string): string {
  if (bucket === "day") return start;
  if (bucket === "month") return monthEndOf(start, cap);
  const [y, m, d] = parts(start);
  const end = iso(y, m, d + 6);
  return cap < end ? cap : end;
}

/** Comparison period, the same rule as the API: same length right before, or the same dates a year earlier. */
export function compareRange(r: HrRange): { from: string; to: string } {
  const [fy, fm, fd] = parts(r.from);
  const [ty, tm, td] = parts(r.to);
  if (r.compare === "year") return { from: iso(fy - 1, fm, fd), to: iso(ty - 1, tm, td) };
  const days = Math.round((Date.UTC(ty, tm, td) - Date.UTC(fy, fm, fd)) / 86_400_000) + 1;
  return { from: iso(fy, fm, fd - days), to: iso(fy, fm, fd - 1) };
}
