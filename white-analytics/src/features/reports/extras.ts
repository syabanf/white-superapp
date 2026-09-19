/**
 * Report extras: rank tracker, backlinks and publishing performance.
 * Pure shaping only (tested in tests/reports-extras.test.ts). Formulas stay in
 * src/lib/metrics/*; rank and backlink aggregates arrive from seo-suite queries.
 * The PDF and the builder preview both render the `ReportSection`s built here.
 */
import { formatCompact, formatDateShort, formatNumber, formatPercent, truncate } from "@/lib/format";
import { postEngagements, sumBy, type Delta } from "@/lib/metrics";
import type { BacklinkData, RankData, RankKeywordRow } from "@/features/seo-suite/queries";
import type { PdfLang, pdfLabels } from "@/features/reports/strings";
import type { PdfTable } from "@/features/reports/pdf/types";

export type SectionKpi = { label: string; value: string; delta: Delta | null; lowerIsBetter?: boolean };

export type ReportSection = {
  key: "RANK" | "BACKLINKS" | "PUBLISHING";
  title: string;
  kpis: SectionKpi[];
  table: PdfTable | null;
};

type Labels = (typeof pdfLabels)[PdfLang];

const PLATFORM_NAME: Record<string, string> = { INSTAGRAM: "Instagram", FACEBOOK: "Facebook", TIKTOK: "TikTok" };
const platformName = (p: string) => PLATFORM_NAME[p] ?? p;

// ── Rank tracker ──────────────────────────────────────────────

type RankRowLike = Pick<RankKeywordRow, "keyword" | "position" | "volume" | "change">;

/** Best positions first; ties and unranked keywords fall back to search volume. */
export function topRankRows<T extends RankRowLike>(rows: T[], n = 10): T[] {
  return [...rows]
    .sort((a, b) => (a.position ?? Infinity) - (b.position ?? Infinity) || (b.volume ?? 0) - (a.volume ?? 0) || a.keyword.localeCompare(b.keyword))
    .slice(0, n);
}

/** `delta` is start minus current, so a positive number means the keyword moved up. */
export function rankChangeText(change: RankRowLike["change"], L: Labels): string {
  switch (change.kind) {
    case "up":
      return `${L.rankUp} ${formatNumber(Math.abs(change.delta ?? 0))}`;
    case "down":
      return `${L.rankDown} ${formatNumber(Math.abs(change.delta ?? 0))}`;
    case "flat":
      return L.rankFlat;
    case "new":
      return L.rankNew;
    case "lost":
      return L.rankLost;
    default:
      return "-";
  }
}

export function buildRankSection(rank: Pick<RankData, "hasKeywords" | "latestDate" | "kpis" | "deltas" | "rows">, compare: boolean, L: Labels): ReportSection | null {
  if (!rank.hasKeywords || !rank.latestDate) return null;
  const d = (x: Delta) => (compare ? x : null);
  return {
    key: "RANK",
    title: L.rankTitle,
    kpis: [
      { label: L.rankVisibility, value: formatPercent(rank.kpis.visibility, 1), delta: d(rank.deltas.visibility) },
      { label: L.rankAvgPosition, value: rank.kpis.avgPosition == null ? "-" : formatNumber(rank.kpis.avgPosition, 1), delta: d(rank.deltas.avgPosition), lowerIsBetter: true },
      { label: L.rankTop3, value: formatNumber(rank.kpis.top3), delta: d(rank.deltas.top3) },
      { label: L.rankTop10, value: formatNumber(rank.kpis.top10), delta: d(rank.deltas.top10) },
    ],
    table: {
      title: L.rankTable,
      columns: [
        { label: L.colKeyword, width: 46 },
        { label: L.colPosition, width: 14, align: "right" },
        { label: L.colChange, width: 22, align: "right" },
        { label: L.colVolume, width: 18, align: "right" },
      ],
      rows: topRankRows(rank.rows).map((r) => [
        truncate(r.keyword, 60),
        r.position == null ? L.rankUnranked : formatNumber(r.position),
        rankChangeText(r.change, L),
        r.volume == null ? "-" : formatCompact(r.volume),
      ]),
    },
  };
}

// ── Backlinks ─────────────────────────────────────────────────

export function buildBacklinkSection(data: Pick<BacklinkData, "latest" | "dofollowShare" | "weekly">, L: Labels): ReportSection | null {
  if (!data.latest) return null;
  return {
    key: "BACKLINKS",
    title: L.backlinksTitle,
    kpis: [
      { label: L.backlinksTotal, value: formatCompact(data.latest.backlinks), delta: null },
      { label: L.referringDomains, value: formatCompact(data.latest.referringDomains), delta: null },
      { label: L.domainRank, value: data.latest.domainRank == null ? "-" : formatNumber(data.latest.domainRank), delta: null },
      { label: L.dofollowShare, value: formatPercent(data.dofollowShare, 1), delta: null },
      { label: L.backlinksNew, value: formatNumber(sumBy(data.weekly, (w) => w.baru)), delta: null },
      { label: L.backlinksLost, value: formatNumber(sumBy(data.weekly, (w) => w.hilang)), delta: null },
    ],
    table: null,
  };
}

// ── Publishing performance ────────────────────────────────────

export type PublishedTarget = {
  postId: string;
  title: string;
  body: string;
  platform: string;
  socialAccountId: string;
  externalId: string | null;
  publishedAt: Date;
};

export type SyncedPostStats = { socialAccountId: string; externalId: string; publishedAt: Date; likes: number; comments: number; shares: number; saves: number };

export type PublishingSummary = {
  posts: number;
  perPlatform: { platform: string; count: number }[];
  failed: number;
  /** "engagement" when at least one published target matches a synced post, else "recent". */
  mode: "engagement" | "recent";
  top: (PublishedTarget & { engagements: number | null })[];
};

export function summarizePublishing(published: PublishedTarget[], failed: number, synced: SyncedPostStats[], n = 5): PublishingSummary {
  const counts = new Map<string, number>();
  for (const t of published) counts.set(t.platform, (counts.get(t.platform) ?? 0) + 1);
  const statsByKey = new Map(synced.map((s) => [`${s.socialAccountId}:${s.externalId}`, s]));
  const withStats = published.map((t) => {
    const stats = t.externalId ? statsByKey.get(`${t.socialAccountId}:${t.externalId}`) : undefined;
    return { ...t, engagements: stats ? postEngagements(stats) : null };
  });
  const matched = withStats.filter((t) => t.engagements != null);
  const byRecency = (a: PublishedTarget, b: PublishedTarget) => b.publishedAt.getTime() - a.publishedAt.getTime();
  return {
    posts: new Set(published.map((t) => t.postId)).size,
    perPlatform: [...counts.entries()].map(([platform, count]) => ({ platform, count })).sort((a, b) => b.count - a.count || a.platform.localeCompare(b.platform)),
    failed,
    mode: matched.length > 0 ? "engagement" : "recent",
    top: (matched.length > 0 ? matched.sort((a, b) => b.engagements! - a.engagements! || byRecency(a, b)) : withStats.sort(byRecency)).slice(0, n),
  };
}

export function buildPublishingSection(summary: PublishingSummary, L: Labels): ReportSection | null {
  if (summary.posts === 0) return null;
  const byEngagement = summary.mode === "engagement";
  return {
    key: "PUBLISHING",
    title: L.publishingTitle,
    kpis: [
      { label: L.publishedPosts, value: formatNumber(summary.posts), delta: null },
      ...summary.perPlatform.map((p) => ({ label: `${L.publishedOn} ${platformName(p.platform)}`, value: formatNumber(p.count), delta: null })),
      { label: L.publishFailed, value: formatNumber(summary.failed), delta: null },
    ],
    table: {
      title: byEngagement ? L.publishedTop : L.publishedRecent,
      columns: [
        { label: "Platform", width: 16 },
        { label: L.colPost, width: byEngagement ? 52 : 66 },
        { label: L.colPublishedAt, width: 18, align: "right" },
        ...(byEngagement ? [{ label: L.colEngagements, width: 14, align: "right" as const }] : []),
      ],
      rows: summary.top.map((t) => [
        platformName(t.platform),
        truncate((t.title || t.body).replace(/\s+/g, " "), byEngagement ? 70 : 90),
        formatDateShort(t.publishedAt),
        ...(byEngagement ? [formatNumber(t.engagements ?? 0)] : []),
      ]),
    },
  };
}
