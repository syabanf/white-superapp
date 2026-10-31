const timeFmt = new Intl.DateTimeFormat("id-ID", { timeZone: "Asia/Jakarta", hour: "2-digit", minute: "2-digit" });
const weekdayFmt = new Intl.DateTimeFormat("id-ID", { timeZone: "UTC", weekday: "short", day: "numeric", month: "short" });

/** RFC 3339 timestamp → "08.05" in WIB; "–" when empty. */
export function timeWib(ts: string | null | undefined): string {
  if (!ts) return "–";
  return timeFmt.format(new Date(ts));
}

const hhmmFmt = new Intl.DateTimeFormat("en-GB", { timeZone: "Asia/Jakarta", hour: "2-digit", minute: "2-digit", hourCycle: "h23" });

/** RFC 3339 timestamp → "08:05" in WIB for a time input; "" when empty. */
export function hhmmWib(ts: string | null | undefined): string {
  return ts ? hhmmFmt.format(new Date(ts)) : "";
}

/** "2026-10-26" → "Sen, 26 Okt". */
export function weekdayDate(day: string): string {
  return weekdayFmt.format(new Date(`${day}T00:00:00Z`));
}

export function sumHours(values: number[]): number {
  return Math.round(values.reduce((a, b) => a + b, 0) * 100) / 100;
}

/** "?a=1&b=2" from non-empty values; safe in client components. */
export function toQuery(params: Record<string, string | number | undefined | null>): string {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) if (v != null && v !== "") sp.set(k, String(v));
  const s = sp.toString();
  return s ? `?${s}` : "";
}

