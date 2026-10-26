/**
 * Period presets and comparison for ERP analytics dashboards. Pure and
 * safe on server and client; every day is a WIB "YYYY-MM-DD" string.
 * URL params: `preset`, `from`, `to` (custom range) and `compare`.
 */

export type PeriodPreset = "thisMonth" | "lastMonth" | "thisQuarter" | "ytd" | "12m" | "custom";
export type CompareMode = "previous" | "year";

export const PERIOD_PRESETS: { value: Exclude<PeriodPreset, "custom">; label: string }[] = [
  { value: "thisMonth", label: "Bulan ini" },
  { value: "lastMonth", label: "Bulan lalu" },
  { value: "thisQuarter", label: "Kuartal ini" },
  { value: "ytd", label: "Tahun berjalan (YTD)" },
  { value: "12m", label: "12 bulan terakhir" },
];

export const COMPARE_MODES: { value: CompareMode; label: string }[] = [
  { value: "previous", label: "vs periode lalu" },
  { value: "year", label: "vs tahun lalu" },
];

export const CUSTOM_LABEL = "Rentang kustom";

/** The URL params the period controls own. */
export const PERIOD_PARAMS = ["preset", "from", "to", "compare"] as const;

export type ResolvedPeriod = {
  preset: PeriodPreset;
  from: string;
  to: string;
  compare: CompareMode;
  /** "Bulan ini" or "1 Agu 2026 – 26 Okt 2026" for a custom range */
  label: string;
};

const ISO = /^\d{4}-\d{2}-\d{2}$/;

export function todayWib(now = new Date()): string {
  return new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Jakarta" }).format(now);
}

function iso(y: number, m0: number, d: number): string {
  return new Date(Date.UTC(y, m0, d)).toISOString().slice(0, 10);
}

/** Shifts a day by n days. */
export function addDaysIso(day: string, n: number): string {
  const d = new Date(`${day}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() + n);
  return d.toISOString().slice(0, 10);
}

export function presetRange(preset: Exclude<PeriodPreset, "custom">, today: string): { from: string; to: string } {
  const [y, m, d] = today.split("-").map(Number) as [number, number, number];
  switch (preset) {
    case "thisMonth":
      return { from: iso(y, m - 1, 1), to: today };
    case "lastMonth":
      return { from: iso(y, m - 2, 1), to: iso(y, m - 1, 0) };
    case "thisQuarter":
      return { from: iso(y, Math.floor((m - 1) / 3) * 3, 1), to: today };
    case "ytd":
      return { from: iso(y, 0, 1), to: today };
    case "12m":
      return { from: addDaysIso(iso(y - 1, m - 1, d), 1), to: today };
  }
}

const SHORT_MONTHS = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];

/** "2026-10-26" → "26 Okt 2026" */
export function formatDay(day: string): string {
  const [y, m, d] = day.split("-").map(Number);
  return `${d} ${SHORT_MONTHS[(m ?? 1) - 1]} ${y}`;
}

export function formatPeriod(from: string, to: string): string {
  return `${formatDay(from)} – ${formatDay(to)}`;
}

/** Resolves the period from search params; invalid input falls back to the default preset. */
export function resolvePeriod(
  sp: { preset?: string | null; from?: string | null; to?: string | null; compare?: string | null },
  defaultPreset: Exclude<PeriodPreset, "custom"> = "12m",
  now = new Date(),
): ResolvedPeriod {
  const today = todayWib(now);
  const compare: CompareMode = sp.compare === "year" ? "year" : "previous";
  if (sp.from && sp.to && ISO.test(sp.from) && ISO.test(sp.to) && sp.from <= sp.to) {
    return { preset: "custom", from: sp.from, to: sp.to, compare, label: formatPeriod(sp.from, sp.to) };
  }
  const found = PERIOD_PRESETS.find((p) => p.value === sp.preset) ?? PERIOD_PRESETS.find((p) => p.value === defaultPreset)!;
  return { preset: found.value, ...presetRange(found.value, today), compare, label: found.label };
}
