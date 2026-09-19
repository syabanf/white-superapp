import "server-only";
import { cache } from "react";
import { db } from "@/lib/db";
import { addDays, dayCount, type DateRange } from "@/lib/dates";
import { truncate } from "@/lib/format";
import {
  alignPrevious,
  avgEngagementRate,
  computeDelta,
  contentType,
  followerGrowth,
  postEngagementRate,
  postEngagements,
  postingHeatmap,
  postsPerWeek,
  round,
  sumBy,
  topPosts,
  type Delta,
} from "@/lib/metrics";
import {
  dailyFollowerSeries,
  erByContentType,
  followersAt,
  indexTo100,
  type ErByType,
  type Platform,
} from "@/features/social/lib";

// ── types ─────────────────────────────────────────────────────

export type SocialAccountSummary = {
  id: string;
  platform: string;
  username: string;
  displayName: string;
  avatarUrl: string | null;
  isCompetitor: boolean;
  followers: number;
};

export type SocialPostRow = {
  id: string;
  caption: string;
  captionShort: string;
  thumbnailUrl: string | null;
  permalink: string | null;
  type: string;
  /** ISO timestamp */
  publishedAt: string;
  likes: number;
  comments: number;
  shares: number;
  saves: number;
  views: number;
  reach: number;
  engagements: number;
  er: number;
  /** hanya untuk kartu postingan kompetitor */
  username?: string;
};

export type SocialKpis = {
  followers: number;
  /** pertumbuhan periode ini vs pertumbuhan periode sebelumnya */
  followersDelta: Delta;
  followersSpark: number[];
  growthAbs: number;
  growthPct: number | null;
  er: number;
  erDelta: Delta;
  reach: number;
  reachDelta: Delta;
  impressions: number;
  impressionsDelta: Delta;
  posts: number;
  postsDelta: Delta;
  postsPerWeek: number;
};

export type GrowthPoint = { date: string; followers: number | null; followersPrev: number | null };
export type ReachPoint = { date: string; reach: number; impressions: number };
export type HeatCellData = { day: number; hour: number; posts: number; avgEngagement: number };

export type SocialDashboard = {
  isDemo: boolean;
  account: { id: string; username: string; displayName: string; avatarUrl: string | null } | null;
  kpis: SocialKpis | null;
  growthSeries: GrowthPoint[];
  reachSeries: ReachPoint[];
  erByType: ErByType[];
  heatmap: HeatCellData[];
  topPosts: SocialPostRow[];
  posts: SocialPostRow[];
};

export type CompetitorRow = {
  accountId: string;
  username: string;
  displayName: string;
  avatarUrl: string | null;
  isOwn: boolean;
  followers: number;
  growthAbs: number;
  growthPct: number | null;
  /** pertumbuhan periode ini vs periode sebelumnya (untuk badge saat compare aktif) */
  growthDelta: Delta;
  avgEr: number;
  postsPerWeek: number;
  mediaCount: number;
};

export type CompetitorChart = {
  data: ({ date: string } & Record<string, number | string | null>)[];
  series: { key: string; label: string }[];
};

export type CompetitorDashboard = {
  hasOwn: boolean;
  rows: CompetitorRow[];
  chart: CompetitorChart;
  topPosts: SocialPostRow[];
};

// ── helpers ───────────────────────────────────────────────────

type PostRecord = {
  id: string;
  caption: string;
  mediaType: string;
  productType: string | null;
  permalink: string | null;
  thumbnailUrl: string | null;
  publishedAt: Date;
  likes: number;
  comments: number;
  shares: number;
  saves: number;
  views: number;
  reach: number;
};

function toPostRow(p: PostRecord, followers: number, username?: string): SocialPostRow {
  return {
    id: p.id,
    caption: p.caption,
    captionShort: truncate(p.caption, 90),
    thumbnailUrl: p.thumbnailUrl,
    permalink: p.permalink,
    type: contentType(p),
    publishedAt: p.publishedAt.toISOString(),
    likes: p.likes,
    comments: p.comments,
    shares: p.shares,
    saves: p.saves,
    views: p.views,
    reach: p.reach,
    engagements: postEngagements(p),
    er: round(postEngagementRate(p, followers), 2),
    username,
  };
}

// ── queries ───────────────────────────────────────────────────

/** Semua akun sosial klien (sendiri + kompetitor) + followers terkini. */
export const getSocialAccounts = cache(async (clientId: string): Promise<SocialAccountSummary[]> => {
  const accounts = await db.socialAccount.findMany({
    where: { clientId },
    orderBy: [{ isCompetitor: "asc" }, { platform: "asc" }, { createdAt: "asc" }],
    select: {
      id: true,
      platform: true,
      username: true,
      displayName: true,
      avatarUrl: true,
      isCompetitor: true,
    },
  });
  if (accounts.length === 0) return [];
  const latest = await db.socialSnapshot.findMany({
    where: { socialAccountId: { in: accounts.map((a) => a.id) } },
    orderBy: [{ socialAccountId: "asc" }, { date: "desc" }],
    distinct: ["socialAccountId"],
    select: { socialAccountId: true, followers: true },
  });
  const byAcc = new Map(latest.map((l) => [l.socialAccountId, l.followers]));
  return accounts.map((a) => ({ ...a, followers: byAcc.get(a.id) ?? 0 }));
});

/** Dashboard per platform untuk akun milik sendiri. */
export const getSocialDashboard = cache(
  async (
    clientId: string,
    platform: Platform,
    range: DateRange,
    previous: DateRange,
  ): Promise<SocialDashboard> => {
    const account = await db.socialAccount.findFirst({
      where: { clientId, platform, isCompetitor: false },
      select: { id: true, username: true, displayName: true, avatarUrl: true, connectionId: true },
    });
    if (!account) {
      return {
        isDemo: true,
        account: null,
        kpis: null,
        growthSeries: [],
        reachSeries: [],
        erByType: [],
        heatmap: [],
        topPosts: [],
        posts: [],
      };
    }

    const [snapshots, posts, prevPosts] = await Promise.all([
      db.socialSnapshot.findMany({
        // sehari sebelum awal periode pembanding → baseline pertumbuhan kedua periode
        where: { socialAccountId: account.id, date: { gte: addDays(previous.from, -1), lte: range.to } },
        orderBy: { date: "asc" },
        select: { date: true, followers: true, reach: true, impressions: true },
      }),
      db.socialPost.findMany({
        where: { socialAccountId: account.id, publishedAt: { gte: range.from, lt: addDays(range.to, 1) } },
        orderBy: { publishedAt: "desc" },
        select: {
          id: true,
          caption: true,
          mediaType: true,
          productType: true,
          permalink: true,
          thumbnailUrl: true,
          publishedAt: true,
          likes: true,
          comments: true,
          shares: true,
          saves: true,
          views: true,
          reach: true,
        },
      }),
      db.socialPost.findMany({
        where: {
          socialAccountId: account.id,
          publishedAt: { gte: previous.from, lt: addDays(previous.to, 1) },
        },
        select: { likes: true, comments: true, shares: true, saves: true, publishedAt: true },
      }),
    ]);

    // followers & pertumbuhan
    const followersNow = followersAt(snapshots, range.to);
    const followersStart = followersAt(snapshots, addDays(range.from, -1));
    const followersPrevStart = followersAt(snapshots, addDays(previous.from, -1));
    const growth = followerGrowth(followersStart, followersNow);
    const prevGrowthAbs = followersStart - followersPrevStart;

    // seri harian + overlay periode sebelumnya (disejajarkan per indeks hari)
    const cur = dailyFollowerSeries(range, snapshots);
    const prev = dailyFollowerSeries(previous, snapshots);
    const growthSeries: GrowthPoint[] = alignPrevious(cur, prev, "followers").map((p) => ({
      date: p.date as string,
      followers: p.followers as number | null,
      followersPrev: (p.followersPrev ?? null) as number | null,
    }));
    const reachSeries: ReachPoint[] = cur.map((p) => ({
      date: p.date,
      reach: p.reach,
      impressions: p.impressions,
    }));
    const followersSpark = growthSeries
      .map((p) => p.followers)
      .filter((v): v is number => v != null)
      .slice(-14);

    // jangkauan & impresi (jumlah dalam periode)
    const inRange = snapshots.filter((sn) => sn.date >= range.from && sn.date <= range.to);
    const inPrev = snapshots.filter((sn) => sn.date >= previous.from && sn.date <= previous.to);
    const reachSum = sumBy(inRange, (sn) => sn.reach);
    const prevReachSum = sumBy(inPrev, (sn) => sn.reach);
    const imprSum = sumBy(inRange, (sn) => sn.impressions);
    const prevImprSum = sumBy(inPrev, (sn) => sn.impressions);

    // engagement rate (ER by followers)
    const er = avgEngagementRate(posts, followersNow);
    const prevEr = avgEngagementRate(prevPosts, followersStart || followersNow);

    const kpis: SocialKpis = {
      followers: followersNow,
      followersDelta: computeDelta(growth.abs, prevGrowthAbs),
      followersSpark,
      growthAbs: growth.abs,
      growthPct: growth.pct,
      er,
      erDelta: computeDelta(er, prevEr),
      reach: reachSum,
      reachDelta: computeDelta(reachSum, prevReachSum),
      impressions: imprSum,
      impressionsDelta: computeDelta(imprSum, prevImprSum),
      posts: posts.length,
      postsDelta: computeDelta(posts.length, prevPosts.length),
      postsPerWeek: postsPerWeek(posts.length, dayCount(range)),
    };

    return {
      isDemo: account.connectionId == null,
      account: {
        id: account.id,
        username: account.username,
        displayName: account.displayName,
        avatarUrl: account.avatarUrl,
      },
      kpis,
      growthSeries,
      reachSeries,
      erByType: erByContentType(posts, followersNow),
      heatmap: postingHeatmap(posts),
      topPosts: topPosts(posts, 6).map((p) => toPostRow(p, followersNow)),
      posts: posts.map((p) => toPostRow(p, followersNow)),
    };
  },
);

/** Perbandingan akun IG sendiri vs kompetitor. */
export const getCompetitorDashboard = cache(
  async (clientId: string, range: DateRange, previous: DateRange): Promise<CompetitorDashboard> => {
    const accounts = await db.socialAccount.findMany({
      where: { clientId, platform: "INSTAGRAM" },
      orderBy: { createdAt: "asc" },
      select: { id: true, username: true, displayName: true, avatarUrl: true, isCompetitor: true },
    });
    if (accounts.length === 0)
      return { hasOwn: false, rows: [], chart: { data: [], series: [] }, topPosts: [] };

    const ids = accounts.map((a) => a.id);
    const [snapshots, posts] = await Promise.all([
      db.socialSnapshot.findMany({
        where: { socialAccountId: { in: ids }, date: { gte: addDays(previous.from, -1), lte: range.to } },
        orderBy: { date: "asc" },
        select: { socialAccountId: true, date: true, followers: true, mediaCount: true },
      }),
      db.socialPost.findMany({
        where: { socialAccountId: { in: ids }, publishedAt: { gte: range.from, lt: addDays(range.to, 1) } },
        orderBy: { publishedAt: "desc" },
        select: {
          id: true,
          socialAccountId: true,
          caption: true,
          mediaType: true,
          productType: true,
          permalink: true,
          thumbnailUrl: true,
          publishedAt: true,
          likes: true,
          comments: true,
          shares: true,
          saves: true,
          views: true,
          reach: true,
        },
      }),
    ]);

    type Snap = (typeof snapshots)[number];
    const snapsByAcc = new Map<string, Snap[]>();
    for (const sn of snapshots) {
      const list = snapsByAcc.get(sn.socialAccountId) ?? [];
      list.push(sn);
      snapsByAcc.set(sn.socialAccountId, list);
    }
    const postsByAcc = new Map<string, typeof posts>();
    for (const p of posts) {
      const list = postsByAcc.get(p.socialAccountId) ?? [];
      list.push(p);
      postsByAcc.set(p.socialAccountId, list);
    }

    const days = dayCount(range);
    const build = (a: (typeof accounts)[number]): CompetitorRow => {
      const snaps = snapsByAcc.get(a.id) ?? [];
      const myPosts = postsByAcc.get(a.id) ?? [];
      const followers = followersAt(snaps, range.to);
      const start = followersAt(snaps, addDays(range.from, -1));
      const prevStart = followersAt(snaps, addDays(previous.from, -1));
      const growth = followerGrowth(start, followers);
      let mediaCount = 0;
      for (const sn of snaps) if (sn.date <= range.to) mediaCount = sn.mediaCount;
      return {
        accountId: a.id,
        username: a.username,
        displayName: a.displayName,
        avatarUrl: a.avatarUrl,
        isOwn: !a.isCompetitor,
        followers,
        growthAbs: growth.abs,
        growthPct: growth.pct,
        growthDelta: computeDelta(growth.abs, start - prevStart),
        avgEr: avgEngagementRate(myPosts, followers),
        postsPerWeek: postsPerWeek(myPosts.length, days),
        mediaCount,
      };
    };

    const own = accounts.find((a) => !a.isCompetitor) ?? null;
    const ownRow = own ? build(own) : null;
    const compRows = accounts
      .filter((a) => a.isCompetitor)
      .map(build)
      .sort((x, y) => y.followers - x.followers);
    const rows = ownRow ? [ownRow, ...compRows] : compRows;

    // Grafik: akun sendiri + 3 kompetitor teratas, diindeks ke 100 di awal periode
    const chartRows = (ownRow ? [ownRow, ...compRows] : compRows).slice(0, 4);
    const perAcc = chartRows.map((r, i) => {
      const series = dailyFollowerSeries(range, snapsByAcc.get(r.accountId) ?? []);
      return {
        key: `a${i}`,
        label: `@${r.username}`,
        dates: series.map((p) => p.date),
        values: indexTo100(series.map((p) => p.followers)).map((v) => (v == null ? null : round(v, 1))),
      };
    });
    const data = (perAcc[0]?.dates ?? []).map((date, i) => {
      const point: { date: string } & Record<string, number | string | null> = { date };
      for (const pa of perAcc) point[pa.key] = pa.values[i] ?? null;
      return point;
    });

    // Postingan kompetitor terbaik lintas akun
    const followersByAcc = new Map(rows.map((r) => [r.accountId, r.followers]));
    const usernameByAcc = new Map(accounts.map((a) => [a.id, a.username]));
    const compIds = new Set(accounts.filter((a) => a.isCompetitor).map((a) => a.id));
    const compPosts = posts.filter((p) => compIds.has(p.socialAccountId));
    const top = topPosts(compPosts, 6).map((p) =>
      toPostRow(p, followersByAcc.get(p.socialAccountId) ?? 0, usernameByAcc.get(p.socialAccountId)),
    );

    return {
      hasOwn: Boolean(own),
      rows,
      chart: { data, series: perAcc.map(({ key, label }) => ({ key, label })) },
      topPosts: top,
    };
  },
);
