import "server-only";
import { cache } from "react";
import { db } from "@/lib/db";
import type { DateRange } from "@/lib/dates";
import { addDays } from "@/lib/dates";
import {
  aggregateAds,
  aggregateSearch,
  avgEngagementRate,
  computeDelta,
  fillDaily,
  healthScore,
  sparkline,
  type AdKpis,
  type Delta,
} from "@/lib/metrics";

export type OverviewSocial = {
  hasData: boolean;
  isDemo: boolean;
  followers: number;
  followersDelta: Delta;
  followersSpark: number[];
  posts: number;
  postsDelta: Delta;
  engagementRate: number;
  engagementRateDelta: Delta;
  reach: number;
  reachDelta: Delta;
  accounts: { platform: string; username: string; followers: number }[];
};

export type OverviewSeo = {
  hasData: boolean;
  isDemo: boolean;
  clicks: number;
  clicksDelta: Delta;
  clicksSpark: number[];
  impressions: number;
  impressionsDelta: Delta;
  ctr: number;
  ctrDelta: Delta;
  position: number;
  positionDelta: Delta;
  health: number | null;
  daily: { date: string; clicks: number; impressions: number }[];
};

export type OverviewAds = {
  hasData: boolean;
  isDemo: boolean;
  kpis: AdKpis;
  prev: AdKpis;
  spendDelta: Delta;
  resultsDelta: Delta;
  cprDelta: Delta;
  spendSpark: number[];
  daily: { date: string; spend: number; results: number }[];
};

/** Result types that are not conversions (awareness objectives). */
export const NON_CONVERSION_RESULT_TYPES = new Set(["reach", "impressions", "video_view", "thruplay"]);

export type OverviewSyncJob = {
  id: string;
  kind: string;
  status: string;
  startedAt: Date;
  message: string | null;
  error: string | null;
};

/** Social summary for own (non-competitor) accounts across platforms. */
export const getOverviewSocial = cache(
  async (clientId: string, range: DateRange, previous: DateRange): Promise<OverviewSocial> => {
    const accounts = await db.socialAccount.findMany({
      where: { clientId, isCompetitor: false },
      select: { id: true, platform: true, username: true, connectionId: true },
    });
    if (accounts.length === 0) {
      const zero = computeDelta(0, 0);
      return {
        hasData: false,
        isDemo: true,
        followers: 0,
        followersDelta: zero,
        followersSpark: [],
        posts: 0,
        postsDelta: zero,
        engagementRate: 0,
        engagementRateDelta: zero,
        reach: 0,
        reachDelta: zero,
        accounts: [],
      };
    }
    const ids = accounts.map((a) => a.id);
    const [snapshots, posts, prevPosts] = await Promise.all([
      db.socialSnapshot.findMany({
        where: { socialAccountId: { in: ids }, date: { gte: addDays(previous.from, -1), lte: range.to } },
        orderBy: { date: "asc" },
        select: { socialAccountId: true, date: true, followers: true, reach: true },
      }),
      db.socialPost.findMany({
        where: { socialAccountId: { in: ids }, publishedAt: { gte: range.from, lt: addDays(range.to, 1) } },
        select: {
          socialAccountId: true,
          likes: true,
          comments: true,
          shares: true,
          saves: true,
          publishedAt: true,
        },
      }),
      db.socialPost.findMany({
        where: {
          socialAccountId: { in: ids },
          publishedAt: { gte: previous.from, lt: addDays(previous.to, 1) },
        },
        select: {
          socialAccountId: true,
          likes: true,
          comments: true,
          shares: true,
          saves: true,
          publishedAt: true,
        },
      }),
    ]);

    // followers at a date = latest snapshot on/before that date per account
    const followersAt = (d: Date) => {
      const map = new Map<string, number>();
      for (const s of snapshots) if (s.date <= d) map.set(s.socialAccountId, s.followers);
      let sum = 0;
      for (const v of map.values()) sum += v;
      return sum;
    };
    const followersNow = followersAt(range.to);
    const followersStart = followersAt(addDays(range.from, -1));
    const followersPrevStart = followersAt(addDays(previous.from, -1));
    const followersDelta = computeDelta(followersNow - followersStart, followersStart - followersPrevStart);

    // daily total followers series for sparkline
    const dailyTotals = new Map<string, number>();
    const perAcc = new Map<string, number>();
    for (const s of snapshots) {
      if (s.date < range.from) {
        perAcc.set(s.socialAccountId, s.followers);
        continue;
      }
      perAcc.set(s.socialAccountId, s.followers);
      let sum = 0;
      for (const v of perAcc.values()) sum += v;
      dailyTotals.set(s.date.toISOString().slice(0, 10), sum);
    }
    const followersSpark = [...dailyTotals.values()].slice(-14);

    const reachRows = snapshots.filter((s) => s.date >= range.from && s.date <= range.to);
    const prevReachRows = snapshots.filter((s) => s.date >= previous.from && s.date <= previous.to);
    const reach = reachRows.reduce((a, b) => a + (b.reach ?? 0), 0);
    const prevReach = prevReachRows.reduce((a, b) => a + (b.reach ?? 0), 0);

    // ER by followers, per account, then mean weighted by post count
    const followersByAcc = new Map<string, number>();
    for (const s of snapshots) followersByAcc.set(s.socialAccountId, s.followers);
    const erFor = (rows: typeof posts) => {
      if (rows.length === 0) return 0;
      let acc = 0;
      for (const a of accounts) {
        const mine = rows.filter((p) => p.socialAccountId === a.id);
        if (mine.length === 0) continue;
        acc += avgEngagementRate(mine, followersByAcc.get(a.id) ?? 0) * mine.length;
      }
      return acc / rows.length;
    };
    const er = erFor(posts);
    const prevEr = erFor(prevPosts);

    return {
      hasData: true,
      isDemo: accounts.every((a) => a.connectionId == null),
      followers: followersNow,
      followersDelta,
      followersSpark,
      posts: posts.length,
      postsDelta: computeDelta(posts.length, prevPosts.length),
      engagementRate: er,
      engagementRateDelta: computeDelta(er, prevEr),
      reach,
      reachDelta: computeDelta(reach, prevReach),
      accounts: accounts.map((a) => ({
        platform: a.platform,
        username: a.username,
        followers: followersByAcc.get(a.id) ?? 0,
      })),
    };
  },
);

export const getOverviewSeo = cache(
  async (clientId: string, range: DateRange, previous: DateRange): Promise<OverviewSeo> => {
    const props = await db.seoProperty.findMany({
      where: { clientId },
      select: { id: true, connectionId: true },
    });
    const zero = computeDelta(0, 0);
    if (props.length === 0) {
      return {
        hasData: false,
        isDemo: true,
        clicks: 0,
        clicksDelta: zero,
        clicksSpark: [],
        impressions: 0,
        impressionsDelta: zero,
        ctr: 0,
        ctrDelta: zero,
        position: 0,
        positionDelta: zero,
        health: null,
        daily: [],
      };
    }
    const ids = props.map((p) => p.id);
    const [rows, prevRows, latestAudit, latestCrawl] = await Promise.all([
      db.seoDailyMetric.findMany({
        where: { propertyId: { in: ids }, date: { gte: range.from, lte: range.to } },
        orderBy: { date: "asc" },
      }),
      db.seoDailyMetric.findMany({
        where: { propertyId: { in: ids }, date: { gte: previous.from, lte: previous.to } },
      }),
      db.seoAudit.findFirst({
        where: { propertyId: { in: ids }, strategy: "MOBILE" },
        orderBy: { runAt: "desc" },
      }),
      db.seoCrawl.findFirst({
        where: { propertyId: { in: ids }, status: "SUCCESS" },
        orderBy: { startedAt: "desc" },
        include: { _count: { select: { issues: true } }, issues: { select: { severity: true } } },
      }),
    ]);
    const cur = aggregateSearch(rows);
    const prev = aggregateSearch(prevRows);
    const daily = fillDaily(range, rows, (r) => r.date, ["clicks", "impressions"], { missing: null }) as {
      date: string;
      clicks: number;
      impressions: number;
    }[];
    const errors = latestCrawl?.issues.filter((i) => i.severity === "ERROR").length ?? 0;
    const warnings = latestCrawl?.issues.filter((i) => i.severity === "WARNING").length ?? 0;
    return {
      hasData: rows.length > 0 || prevRows.length > 0,
      isDemo: props.every((p) => p.connectionId == null),
      clicks: cur.clicks,
      clicksDelta: computeDelta(cur.clicks, prev.clicks),
      clicksSpark: sparkline(daily, "clicks"),
      impressions: cur.impressions,
      impressionsDelta: computeDelta(cur.impressions, prev.impressions),
      ctr: cur.ctr,
      ctrDelta: computeDelta(cur.ctr, prev.ctr),
      position: cur.position,
      positionDelta: computeDelta(cur.position, prev.position),
      health: latestAudit ? healthScore(latestAudit, { errors, warnings }) : null,
      daily,
    };
  },
);

export const getOverviewAds = cache(
  async (clientId: string, range: DateRange, previous: DateRange): Promise<OverviewAds> => {
    const accounts = await db.adAccount.findMany({
      where: { clientId },
      select: { id: true, connectionId: true },
    });
    const ids = accounts.map((a) => a.id);
    const empty = aggregateAds([]);
    if (ids.length === 0) {
      const zero = computeDelta(0, 0);
      return {
        hasData: false,
        isDemo: true,
        kpis: empty,
        prev: empty,
        spendDelta: zero,
        resultsDelta: zero,
        cprDelta: zero,
        spendSpark: [],
        daily: [],
      };
    }
    const [rows, prevRows] = await Promise.all([
      db.adDailyInsight.findMany({
        where: { adAccountId: { in: ids }, adId: { not: null }, date: { gte: range.from, lte: range.to } },
        orderBy: { date: "asc" },
      }),
      db.adDailyInsight.findMany({
        where: {
          adAccountId: { in: ids },
          adId: { not: null },
          date: { gte: previous.from, lte: previous.to },
        },
      }),
    ]);
    // "Hasil" & CPR mengecualikan kampanye awareness (resultType reach/impressions) agar tidak
    // mencampur satuan; spend/impressions tetap dari semua kampanye.
    const conv = (rs: typeof rows) => rs.filter((r) => !NON_CONVERSION_RESULT_TYPES.has(r.resultType ?? ""));
    const withConv = (all: typeof rows) => {
      const k = aggregateAds(all);
      const c = aggregateAds(conv(all));
      return { ...k, results: c.results, cpr: c.cpr, roas: c.roas };
    };
    const kpis = withConv(rows);
    const prev = withConv(prevRows);
    const daily = fillDaily(
      range,
      conv(rows).length
        ? rows.map((r) => (NON_CONVERSION_RESULT_TYPES.has(r.resultType ?? "") ? { ...r, results: 0 } : r))
        : rows,
      (r) => r.date,
      ["spend", "results"],
    ) as { date: string; spend: number; results: number }[];
    return {
      hasData: rows.length > 0 || prevRows.length > 0,
      isDemo: accounts.every((a) => a.connectionId == null),
      kpis,
      prev,
      spendDelta: computeDelta(kpis.spend, prev.spend),
      resultsDelta: computeDelta(kpis.results, prev.results),
      cprDelta: computeDelta(kpis.cpr, prev.cpr),
      spendSpark: sparkline(daily, "spend"),
      daily,
    };
  },
);

export const getRecentSyncJobs = cache(async (clientId: string, take = 6): Promise<OverviewSyncJob[]> => {
  return db.syncJob.findMany({
    where: { clientId },
    orderBy: { startedAt: "desc" },
    take,
    select: { id: true, kind: true, status: true, startedAt: true, message: true, error: true },
  });
});
