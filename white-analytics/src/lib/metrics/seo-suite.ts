/**
 * SEO suite formulas — rank tracking, keyword research, backlinks, domain benchmark.
 * Pure functions only (tested in tests/seo-suite.test.ts). Import from this file
 * directly: `@/lib/metrics/seo-suite` (not re-exported from the metrics index).
 */
import { clamp, mean } from "./delta";
import { expectedCtr } from "./seo";

// ── Visibility & positions ────────────────────────────────────

/**
 * Visibility (%) = mean over tracked keywords of expectedCtr(position).
 * `expectedCtr` is the benchmark organic CTR curve in percent (pos 1 ≈ 28 %, pos 10 ≈ 2,5 %);
 * a keyword that does not rank (null / > 100) contributes 0. Everything at #1 → 28 %.
 */
export function visibility(positions: Array<number | null | undefined>): number {
  if (positions.length === 0) return 0;
  return mean(positions.map((p) => (p == null || p > 100 ? 0 : expectedCtr(p))));
}

/** Mean position over ranked keywords only (unranked ignored). null when none rank. */
export function avgPosition(positions: Array<number | null | undefined>): number | null {
  const ranked = positions.filter((p): p is number => p != null && p <= 100);
  return ranked.length ? mean(ranked) : null;
}

export function countTop(positions: Array<number | null | undefined>, n: number): number {
  return positions.filter((p) => p != null && p <= n).length;
}

export type RankBucket = "1-3" | "4-10" | "11-20" | "21-50" | "51-100" | "unranked";
export const RANK_BUCKETS: RankBucket[] = ["1-3", "4-10", "11-20", "21-50", "51-100", "unranked"];

export function rankBucket(position: number | null | undefined): RankBucket {
  if (position == null || position > 100 || position < 1) return "unranked";
  if (position <= 3) return "1-3";
  if (position <= 10) return "4-10";
  if (position <= 20) return "11-20";
  if (position <= 50) return "21-50";
  return "51-100";
}

export function rankDistribution(positions: Array<number | null | undefined>): { bucket: RankBucket; count: number }[] {
  const acc = new Map<RankBucket, number>(RANK_BUCKETS.map((b) => [b, 0]));
  for (const p of positions) acc.set(rankBucket(p), (acc.get(rankBucket(p)) ?? 0) + 1);
  return RANK_BUCKETS.map((bucket) => ({ bucket, count: acc.get(bucket) ?? 0 }));
}

export type RankChangeKind = "up" | "down" | "flat" | "new" | "lost" | "none";

/**
 * Classify the move between two snapshots (positions; null = not in top 100).
 * `delta` is prev − cur, so positive = improvement (lower-is-better metric).
 */
export function classifyRankChange(prev: number | null | undefined, cur: number | null | undefined): { kind: RankChangeKind; delta: number | null } {
  const p = prev == null || prev > 100 ? null : prev;
  const c = cur == null || cur > 100 ? null : cur;
  if (p == null && c == null) return { kind: "none", delta: null };
  if (p == null) return { kind: "new", delta: null };
  if (c == null) return { kind: "lost", delta: null };
  const delta = p - c;
  return { kind: delta > 0 ? "up" : delta < 0 ? "down" : "flat", delta };
}

/** Alert rule: dropped > 5 positions, fell out of the top 10, or left the top 100 entirely. */
export function isRankDropAlert(prev: number | null | undefined, cur: number | null | undefined): boolean {
  const change = classifyRankChange(prev, cur);
  if (change.kind === "lost") return true;
  if (change.kind !== "down") return false;
  const p = prev as number;
  const c = cur as number;
  return c - p > 5 || (p <= 10 && c > 10);
}

/** Sparkline-friendly transform: higher = better, unranked → 0. */
export function positionToScore(position: number | null | undefined): number {
  if (position == null || position > 100) return 0;
  return 101 - clamp(position, 1, 100);
}

// ── Share of voice ────────────────────────────────────────────

export type TopResult = { domain: string; position: number; url?: string | null };

/**
 * Share of voice per domain from the latest top-10 lists: each appearance is weighted by
 * expectedCtr(position) so a #1 counts far more than a #10. Returns top `n` + "Lainnya",
 * each with its share (%) of the total weight.
 */
export function shareOfVoice(lists: Array<TopResult[] | null | undefined>, n = 8, otherLabel = "Lainnya"): { domain: string; weight: number; share: number }[] {
  const acc = new Map<string, number>();
  for (const list of lists) {
    for (const r of list ?? []) {
      if (!r?.domain || r.position == null) continue;
      acc.set(r.domain, (acc.get(r.domain) ?? 0) + expectedCtr(r.position));
    }
  }
  const total = [...acc.values()].reduce((a, b) => a + b, 0);
  const sorted = [...acc.entries()].sort((a, b) => b[1] - a[1]);
  const head = sorted.slice(0, n);
  const rest = sorted.slice(n).reduce((a, [, w]) => a + w, 0);
  const rows = head.map(([domain, weight]) => ({ domain, weight, share: total ? (weight / total) * 100 : 0 }));
  if (rest > 0) rows.push({ domain: otherLabel, weight: rest, share: total ? (rest / total) * 100 : 0 });
  return rows;
}

// ── Keyword research ──────────────────────────────────────────

export type DifficultyBand = "easy" | "medium" | "hard" | "very_hard";

/** KD bands: 0–29 mudah · 30–59 sedang · 60–79 sulit · 80–100 sangat sulit. */
export function difficultyBand(kd: number | null | undefined): DifficultyBand | null {
  if (kd == null || Number.isNaN(kd)) return null;
  const v = clamp(kd, 0, 100);
  if (v < 30) return "easy";
  if (v < 60) return "medium";
  if (v < 80) return "hard";
  return "very_hard";
}

/** Research cache is reusable for 7 days. */
export const RESEARCH_MAX_AGE_DAYS = 7;

export function isCacheFresh(fetchedAt: Date | string, now: Date, maxAgeDays = RESEARCH_MAX_AGE_DAYS): boolean {
  const t = typeof fetchedAt === "string" ? new Date(fetchedAt).getTime() : fetchedAt.getTime();
  if (Number.isNaN(t)) return false;
  const age = now.getTime() - t;
  return age >= 0 && age < maxAgeDays * 86_400_000;
}

export type GapRow = { keyword: string; volume: number; ownPosition: number | null; competitorPosition: number | null };
export type KeywordGap = { missing: GapRow[]; weak: GapRow[]; strong: GapRow[]; shared: GapRow[] };

type RankedKeyword = { keyword: string; position: number; volume: number };

/**
 * Keyword gap between own domain and a competitor:
 *  missing — competitor ranks, we don't · weak — both rank, competitor is better ·
 *  strong — both rank, we are better (or equal) · shared — every keyword both rank for.
 * Each set is sorted by volume (desc).
 */
export function keywordGap(own: RankedKeyword[], competitor: RankedKeyword[]): KeywordGap {
  const ownMap = new Map<string, RankedKeyword>();
  for (const k of own) ownMap.set(k.keyword.toLowerCase(), k);
  const byVolume = (a: GapRow, b: GapRow) => b.volume - a.volume;
  const missing: GapRow[] = [];
  const weak: GapRow[] = [];
  const strong: GapRow[] = [];
  const shared: GapRow[] = [];
  for (const c of competitor) {
    const key = c.keyword.toLowerCase();
    const mine = ownMap.get(key);
    if (!mine) {
      missing.push({ keyword: c.keyword, volume: c.volume, ownPosition: null, competitorPosition: c.position });
      continue;
    }
    const row: GapRow = { keyword: c.keyword, volume: Math.max(c.volume, mine.volume), ownPosition: mine.position, competitorPosition: c.position };
    shared.push(row);
    if (mine.position > c.position) weak.push(row);
    else strong.push(row);
  }
  return { missing: missing.sort(byVolume), weak: weak.sort(byVolume), strong: strong.sort(byVolume), shared: shared.sort(byVolume) };
}

// ── Backlinks ─────────────────────────────────────────────────

export const TOXIC_SPAM_SCORE = 60;

export function isToxic(spamScore: number | null | undefined): boolean {
  return spamScore != null && spamScore >= TOXIC_SPAM_SCORE;
}

export type BacklinkView = "all" | "new" | "lost" | "toxic";

type BacklinkLike = { firstSeen: Date | string; isLost: boolean; spamScore: number | null };

/** "new" = first seen within the last 30 days; "lost" = flagged lost; "toxic" = spamScore ≥ 60 (live links only). */
export function backlinkMatchesView(b: BacklinkLike, view: BacklinkView, now: Date, newDays = 30): boolean {
  switch (view) {
    case "all":
      return true;
    case "lost":
      return b.isLost;
    case "toxic":
      return !b.isLost && isToxic(b.spamScore);
    case "new": {
      const t = typeof b.firstSeen === "string" ? new Date(b.firstSeen).getTime() : b.firstSeen.getTime();
      return !b.isLost && now.getTime() - t <= newDays * 86_400_000;
    }
  }
}

export function dofollowShare(dofollow: number, nofollow: number): number {
  const total = dofollow + nofollow;
  return total === 0 ? 0 : (dofollow / total) * 100;
}

// ── Domains ───────────────────────────────────────────────────

/**
 * Normalise user input to a bare host: strips scheme, credentials, path, query, port and a
 * leading "www.", lower-cases. Returns null when it is not a plausible hostname.
 */
export function normalizeDomain(input: string): string | null {
  let s = input.trim().toLowerCase();
  if (!s) return null;
  s = s.replace(/^sc-domain:/, "");
  s = s.replace(/^[a-z][a-z0-9+.-]*:\/\//, "");
  s = s.replace(/^[^/@]*@/, "");
  s = s.split(/[/?#]/)[0] ?? "";
  s = s.replace(/:\d+$/, "");
  s = s.replace(/^www\./, "");
  s = s.replace(/\.+$/, "");
  if (!/^(?=.{1,253}$)([a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$/.test(s)) return null;
  return s;
}

/** Own host from a GSC property (`https://x.id/` or `sc-domain:x.id`). */
export function siteHost(siteUrl: string): string {
  return normalizeDomain(siteUrl) ?? siteUrl.replace(/^sc-domain:/, "");
}

export const MAX_COMPETITOR_DOMAINS = 5;
