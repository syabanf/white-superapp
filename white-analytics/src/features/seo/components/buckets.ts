import type { PositionBucket } from "@/lib/metrics";

/** Ordinal ramp for position buckets — best positions get the strongest step. */
export const BUCKET_COLORS: Record<PositionBucket, string> = {
  "1-3": "var(--seq-600)",
  "4-10": "var(--seq-450)",
  "11-20": "var(--seq-300)",
  "21+": "var(--seq-200)",
};

export const BUCKET_LABELS: Record<PositionBucket, string> = {
  "1-3": "Posisi 1–3",
  "4-10": "Posisi 4–10",
  "11-20": "Posisi 11–20",
  "21+": "Posisi 21+",
};
