import { mean, safeDiv } from "./delta";

export type PostLike = {
  likes: number;
  comments: number;
  shares?: number;
  saves?: number;
  views?: number;
  reach?: number;
  publishedAt: Date;
  mediaType?: string | null;
  productType?: string | null;
};

/** Total interactions on a post: likes + comments + shares + saves */
export function postEngagements(p: PostLike): number {
  return p.likes + p.comments + (p.shares ?? 0) + (p.saves ?? 0);
}

/**
 * Engagement rate per post, by followers: (likes+comments+shares+saves) / followers × 100.
 * This is the industry-standard "ER by followers".
 */
export function postEngagementRate(p: PostLike, followers: number): number {
  return safeDiv(postEngagements(p), followers) * 100;
}

/**
 * Engagement rate by reach (when reach is known): interactions / reach × 100.
 */
export function postEngagementRateByReach(p: PostLike): number | null {
  if (!p.reach) return null;
  return safeDiv(postEngagements(p), p.reach) * 100;
}

/**
 * Average account engagement rate for a set of posts:
 * mean of per-post ER (by followers). Returns 0 when there are no posts or followers.
 */
export function avgEngagementRate(posts: PostLike[], followers: number): number {
  if (posts.length === 0 || followers <= 0) return 0;
  return mean(posts.map((p) => postEngagementRate(p, followers)));
}

/** Follower growth between two snapshots. */
export function followerGrowth(startFollowers: number, endFollowers: number): { abs: number; pct: number | null } {
  const abs = endFollowers - startFollowers;
  return { abs, pct: startFollowers === 0 ? null : (abs / startFollowers) * 100 };
}

/** Posting frequency: posts per week over an inclusive day span */
export function postsPerWeek(postCount: number, days: number): number {
  if (days <= 0) return 0;
  return (postCount / days) * 7;
}

/** Content type label used across the app */
export function contentType(p: Pick<PostLike, "mediaType" | "productType">): "Reels" | "Video" | "Carousel" | "Foto" | "Story" {
  const pt = (p.productType ?? "").toUpperCase();
  const mt = (p.mediaType ?? "").toUpperCase();
  if (pt === "REELS") return "Reels";
  if (pt === "STORY") return "Story";
  if (mt === "VIDEO") return "Video";
  if (mt === "CAROUSEL_ALBUM") return "Carousel";
  return "Foto";
}

const DAY_INDEX: Record<string, number> = { Mon: 0, Tue: 1, Wed: 2, Thu: 3, Fri: 4, Sat: 5, Sun: 6 };
const dayHourFormats = new Map<string, Intl.DateTimeFormat>();

/** Weekday (0 = Senin) and hour of a timestamp in the given zone; the heatmap buckets by these. */
export function postDayHour(date: Date, timeZone = "Asia/Jakarta"): { day: number; hour: number } {
  let fmt = dayHourFormats.get(timeZone);
  if (!fmt) {
    fmt = new Intl.DateTimeFormat("en-US", { timeZone, weekday: "short", hour: "numeric", hour12: false });
    dayHourFormats.set(timeZone, fmt);
  }
  const parts = fmt.formatToParts(date);
  const wd = parts.find((x) => x.type === "weekday")?.value ?? "Mon";
  const hour = Number(parts.find((x) => x.type === "hour")?.value ?? 0);
  return { day: DAY_INDEX[wd] ?? 0, hour: hour === 24 ? 0 : hour };
}

/**
 * Best-time-to-post heatmap: 7 (Mon..Sun) × 24 buckets of average engagements.
 * Day index 0 = Senin (Monday) to match Indonesian week start.
 */
export function postingHeatmap(posts: PostLike[], timeZone = "Asia/Jakarta"): { day: number; hour: number; posts: number; avgEngagement: number }[] {
  const sums = new Map<string, { n: number; e: number }>();
  for (const p of posts) {
    const { day, hour } = postDayHour(p.publishedAt, timeZone);
    const key = `${day}-${hour}`;
    const cur = sums.get(key) ?? { n: 0, e: 0 };
    cur.n += 1;
    cur.e += postEngagements(p);
    sums.set(key, cur);
  }
  const out: { day: number; hour: number; posts: number; avgEngagement: number }[] = [];
  for (let d = 0; d < 7; d++) {
    for (let h = 0; h < 24; h++) {
      const c = sums.get(`${d}-${h}`);
      out.push({ day: d, hour: h, posts: c?.n ?? 0, avgEngagement: c ? c.e / c.n : 0 });
    }
  }
  return out;
}

/** Rank posts by engagements desc */
export function topPosts<T extends PostLike>(posts: T[], n = 5): T[] {
  return [...posts].sort((a, b) => postEngagements(b) - postEngagements(a)).slice(0, n);
}
