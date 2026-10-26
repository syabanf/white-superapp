/**
 * Time drill helpers: a mark that stands for a bucket (year, quarter, month,
 * week or day starting at an ISO date) maps to a from/to range and the next
 * finer bucket. Pure; dates are "YYYY-MM-DD" strings.
 */

export type TimeBucket = "year" | "quarter" | "month" | "week" | "day";

const MONTHS = [
  "Januari",
  "Februari",
  "Maret",
  "April",
  "Mei",
  "Juni",
  "Juli",
  "Agustus",
  "September",
  "Oktober",
  "November",
  "Desember",
];
const SHORT = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];

function iso(y: number, m0: number, d: number): string {
  return new Date(Date.UTC(y, m0, d)).toISOString().slice(0, 10);
}

function parts(start: string): [number, number, number] {
  const [y, m = 1, d = 1] = start.slice(0, 10).split("-").map(Number);
  return [y!, m, d];
}

/** Inclusive from/to of the bucket that starts at (or contains) `start`. "2026-03" works for months. */
export function bucketRange(start: string, bucket: TimeBucket): { from: string; to: string } {
  const [y, m, d] = parts(start);
  switch (bucket) {
    case "year":
      return { from: iso(y, 0, 1), to: iso(y, 11, 31) };
    case "quarter": {
      const q0 = Math.floor((m - 1) / 3) * 3;
      return { from: iso(y, q0, 1), to: iso(y, q0 + 3, 0) };
    }
    case "month":
      return { from: iso(y, m - 1, 1), to: iso(y, m, 0) };
    case "week":
      return { from: iso(y, m - 1, d), to: iso(y, m - 1, d + 6) };
    case "day":
      return { from: iso(y, m - 1, d), to: iso(y, m - 1, d) };
  }
}

const CHILD: Record<TimeBucket, TimeBucket | null> = {
  year: "month",
  quarter: "month",
  month: "day",
  week: "day",
  day: null,
};

export function childBucket(bucket: TimeBucket): TimeBucket | null {
  return CHILD[bucket];
}

/** Range and next bucket for drilling into a time mark; null at day level. */
export function timeDrill(
  start: string,
  bucket: TimeBucket,
): { from: string; to: string; bucket: TimeBucket } | null {
  const child = CHILD[bucket];
  return child ? { ...bucketRange(start, bucket), bucket: child } : null;
}

/** Human label for a bucket: "2026", "K1 2026", "Maret 2026", "Minggu 2 Mar 2026", "2 Mar 2026". */
export function bucketLabel(start: string, bucket: TimeBucket): string {
  const [y, m, d] = parts(start);
  switch (bucket) {
    case "year":
      return String(y);
    case "quarter":
      return `K${Math.floor((m - 1) / 3) + 1} ${y}`;
    case "month":
      return `${MONTHS[m - 1]} ${y}`;
    case "week":
      return `Minggu ${d} ${SHORT[m - 1]} ${y}`;
    case "day":
      return `${d} ${SHORT[m - 1]} ${y}`;
  }
}

/** Picks a sensible bucket for a range length: ≤ 45 days daily, ≤ 2 years monthly, else yearly. */
export function bucketForRange(from: string, to: string): TimeBucket {
  const days =
    (Date.parse(`${to.slice(0, 10)}T00:00:00Z`) - Date.parse(`${from.slice(0, 10)}T00:00:00Z`)) / 86_400_000 +
    1;
  if (days <= 45) return "day";
  if (days <= 731) return "month";
  return "year";
}
