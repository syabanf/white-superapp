import type { ResolvedPeriod } from "@/features/erp/shared/dashboard-period";

/** HR names the quarter preset differently from the shared presets. */
const HR_PRESET: Record<string, string> = { thisQuarter: "quarter" };

/**
 * Link to a module dashboard carrying the home period: a preset stays a
 * preset (so "Bulan ini" keeps rolling), a custom range goes as from/to.
 */
export function dashboardHref(path: string, period: ResolvedPeriod, extra: Record<string, string | undefined> = {}): string {
  const sp = new URLSearchParams();
  if (period.preset === "custom") {
    sp.set("from", period.from);
    sp.set("to", period.to);
  } else {
    sp.set("preset", path === "/erp/hr" ? (HR_PRESET[period.preset] ?? period.preset) : period.preset);
  }
  if (period.compare === "year") sp.set("compare", "year");
  for (const [k, v] of Object.entries(extra)) if (v) sp.set(k, v);
  return `${path}?${sp}`;
}
