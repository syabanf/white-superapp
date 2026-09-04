/**
 * Publishing metrics — pure helpers (tested in tests/publishing.test.ts).
 * Not re-exported from `./index` on purpose (parallel work); import directly.
 */

export type HeatCell = { day: number; hour: number; posts: number; avgEngagement: number };
export type BestTime = { day: number; hour: number; score: number };

/** Sensible defaults when an account has no posting history (Mon 11:00, Wed 19:00, Fri 12:00). */
export const DEFAULT_BEST_TIMES: BestTime[] = [
  { day: 0, hour: 11, score: 0 },
  { day: 2, hour: 19, score: 0 },
  { day: 4, hour: 12, score: 0 },
];

/**
 * Rank posting slots from a `postingHeatmap` result. Score = average engagement
 * weighted by log(1 + posts) so a single lucky post cannot dominate. Slots
 * with fewer than `minPosts` are ignored; falls back to defaults (topped up)
 * when history is too thin. Returned slots are on distinct weekdays when possible.
 */
export function bestPostingTimes(cells: HeatCell[], opts: { n?: number; minPosts?: number } = {}): BestTime[] {
  const n = opts.n ?? 3;
  const minPosts = opts.minPosts ?? 2;
  const ranked = cells
    .filter((c) => c.posts >= minPosts && c.avgEngagement > 0)
    .map((c) => ({ day: c.day, hour: c.hour, score: c.avgEngagement * Math.log(1 + c.posts) }))
    .sort((a, b) => b.score - a.score || a.day - b.day || a.hour - b.hour);

  const out: BestTime[] = [];
  const usedDays = new Set<number>();
  for (const r of ranked) {
    if (out.length >= n) break;
    if (usedDays.has(r.day)) continue;
    usedDays.add(r.day);
    out.push(r);
  }
  // Allow same-day slots if we still have room.
  for (const r of ranked) {
    if (out.length >= n) break;
    if (!out.some((o) => o.day === r.day && o.hour === r.hour)) out.push(r);
  }
  for (const d of DEFAULT_BEST_TIMES) {
    if (out.length >= n) break;
    if (!out.some((o) => o.day === d.day && o.hour === d.hour)) out.push(d);
  }
  return out.slice(0, n);
}

export type PostStatusKey = "DRAFT" | "IN_REVIEW" | "APPROVED" | "SCHEDULED" | "PUBLISHING" | "PUBLISHED" | "FAILED" | "REJECTED";

export const POST_STATUSES: PostStatusKey[] = ["DRAFT", "IN_REVIEW", "APPROVED", "SCHEDULED", "PUBLISHING", "PUBLISHED", "FAILED", "REJECTED"];

/** Count posts per status (every status present, zero when empty). */
export function postStatusCounts(posts: { status: string }[]): Record<PostStatusKey, number> {
  const out = Object.fromEntries(POST_STATUSES.map((s) => [s, 0])) as Record<PostStatusKey, number>;
  for (const p of posts) if (p.status in out) out[p.status as PostStatusKey] += 1;
  return out;
}

/** Share of failed targets among finished ones (0–100), null when nothing finished. */
export function failureRate(targets: { status: string }[]): number | null {
  const done = targets.filter((t) => t.status === "PUBLISHED" || t.status === "FAILED");
  if (done.length === 0) return null;
  return (done.filter((t) => t.status === "FAILED").length / done.length) * 100;
}
