import "server-only";
import { cache } from "react";
import { db } from "@/lib/db";
import type { DateRange } from "@/lib/dates";
import { addDays, dayKey, eachDay } from "@/lib/dates";
import { aggregateSearch, postEngagements, safeDiv } from "@/lib/metrics";
import { NON_CONVERSION_RESULT_TYPES } from "@/features/overview/queries";
import { summarizePublishing, type PublishingSummary } from "@/features/reports/extras";

export type InsightModule = "OVERVIEW" | "SOCIAL" | "SEO" | "ADS";

/** Latest cached AI insight for a client+module+exact range (null when none). */
export const getLatestInsight = cache(async (clientId: string, module: InsightModule, range: DateRange) => {
  return db.aiInsight.findFirst({
    where: { clientId, module, dateFrom: range.from, dateTo: range.to },
    orderBy: { createdAt: "desc" },
  });
});

// ── Extra aggregates shared by the report builder, share page, PDF and AI payload ──

export type TopQueryRow = { query: string; clicks: number; impressions: number; ctr: number; position: number };

/** Top N queries by clicks in range (impression-weighted position). */
export const getTopQueryRows = cache(async (clientId: string, range: DateRange, n = 5): Promise<TopQueryRow[]> => {
  const props = await db.seoProperty.findMany({ where: { clientId }, select: { id: true } });
  if (props.length === 0) return [];
  const rows = await db.seoDimensionMetric.findMany({
    where: { propertyId: { in: props.map((p) => p.id) }, dimension: "QUERY", date: { gte: range.from, lte: range.to } },
    select: { key: true, clicks: true, impressions: true, position: true },
  });
  const byKey = new Map<string, { clicks: number; impressions: number; position: number }[]>();
  for (const r of rows) {
    const list = byKey.get(r.key) ?? [];
    list.push(r);
    byKey.set(r.key, list);
  }
  return [...byKey.entries()]
    .map(([query, list]) => {
      const agg = aggregateSearch(list);
      return { query, clicks: agg.clicks, impressions: agg.impressions, ctr: agg.ctr, position: agg.position };
    })
    .sort((a, b) => b.clicks - a.clicks)
    .slice(0, n);
});

export type TopPostRow = {
  id: string;
  platform: string;
  username: string;
  caption: string;
  mediaType: string;
  productType: string | null;
  permalink: string | null;
  publishedAt: Date;
  likes: number;
  comments: number;
  shares: number;
  saves: number;
  engagements: number;
  reach: number;
};

/** Top N own-account posts by total engagements in range. */
export const getTopPostRows = cache(async (clientId: string, range: DateRange, n = 5): Promise<TopPostRow[]> => {
  const posts = await db.socialPost.findMany({
    where: {
      account: { clientId, isCompetitor: false },
      publishedAt: { gte: range.from, lt: addDays(range.to, 1) },
    },
    select: {
      id: true,
      caption: true,
      mediaType: true,
      productType: true,
      permalink: true,
      publishedAt: true,
      likes: true,
      comments: true,
      shares: true,
      saves: true,
      reach: true,
      account: { select: { platform: true, username: true } },
    },
  });
  return posts
    .map((p) => ({
      id: p.id,
      platform: p.account.platform,
      username: p.account.username,
      caption: p.caption,
      mediaType: p.mediaType,
      productType: p.productType,
      permalink: p.permalink,
      publishedAt: p.publishedAt,
      likes: p.likes,
      comments: p.comments,
      shares: p.shares,
      saves: p.saves,
      reach: p.reach,
      engagements: postEngagements(p),
    }))
    .sort((a, b) => b.engagements - a.engagements)
    .slice(0, n);
});

export type CampaignRow = {
  id: string;
  name: string;
  objective: string;
  status: string;
  spend: number;
  impressions: number;
  results: number;
  resultType: string | null;
  /** 0 for awareness campaigns (non-conversion result types) */
  cpr: number;
  isConversion: boolean;
};

/** Campaigns active in range, ordered by spend desc. CPR only for conversion result types. */
export const getCampaignRows = cache(async (clientId: string, range: DateRange): Promise<CampaignRow[]> => {
  const accounts = await db.adAccount.findMany({ where: { clientId }, select: { id: true } });
  if (accounts.length === 0) return [];
  const ids = accounts.map((a) => a.id);
  const [rows, campaigns] = await Promise.all([
    db.adDailyInsight.findMany({
      where: { adAccountId: { in: ids }, adId: { not: null }, campaignId: { not: null }, date: { gte: range.from, lte: range.to } },
      select: { campaignId: true, spend: true, impressions: true, results: true, resultType: true },
    }),
    db.adCampaign.findMany({ where: { adAccountId: { in: ids } }, select: { id: true, name: true, objective: true, status: true } }),
  ]);
  const byCampaign = new Map<string, { spend: number; impressions: number; results: number; resultType: string | null }>();
  for (const r of rows) {
    const key = r.campaignId!;
    const acc = byCampaign.get(key) ?? { spend: 0, impressions: 0, results: 0, resultType: null };
    acc.spend += r.spend;
    acc.impressions += r.impressions;
    acc.results += r.results;
    acc.resultType = acc.resultType ?? r.resultType;
    byCampaign.set(key, acc);
  }
  const nameById = new Map(campaigns.map((c) => [c.id, c]));
  return [...byCampaign.entries()]
    .map(([id, agg]) => {
      const c = nameById.get(id);
      const isConversion = !NON_CONVERSION_RESULT_TYPES.has(agg.resultType ?? "");
      return {
        id,
        name: c?.name ?? "—",
        objective: c?.objective ?? "",
        status: c?.status ?? "ACTIVE",
        spend: agg.spend,
        impressions: agg.impressions,
        results: agg.results,
        resultType: agg.resultType,
        cpr: isConversion ? safeDiv(agg.spend, agg.results) : 0,
        isConversion,
      };
    })
    .sort((a, b) => b.spend - a.spend);
});

/**
 * Publishing in range: PUBLISHED targets by `publishedAt`, FAILED targets by the post's
 * planned instant (`scheduledAt`, else `createdAt`). Engagement comes from synced
 * `SocialPost` rows that share the target's account and `externalId`.
 */
export const getPublishingSummary = cache(async (clientId: string, range: DateRange): Promise<PublishingSummary> => {
  const window = { gte: range.from, lt: addDays(range.to, 1) };
  const [targets, failed] = await Promise.all([
    db.postTarget.findMany({
      where: { status: "PUBLISHED", publishedAt: window, post: { clientId } },
      select: {
        postId: true,
        socialAccountId: true,
        externalId: true,
        publishedAt: true,
        post: { select: { title: true, body: true } },
        account: { select: { platform: true } },
      },
    }),
    db.postTarget.count({
      where: { status: "FAILED", post: { clientId, OR: [{ scheduledAt: window }, { scheduledAt: null, createdAt: window }] } },
    }),
  ]);
  const externalIds = targets.flatMap((t) => (t.externalId ? [t.externalId] : []));
  const synced = externalIds.length
    ? await db.socialPost.findMany({
        where: { externalId: { in: externalIds }, account: { clientId, isCompetitor: false } },
        select: { socialAccountId: true, externalId: true, publishedAt: true, likes: true, comments: true, shares: true, saves: true },
      })
    : [];
  return summarizePublishing(
    targets.map((t) => ({
      postId: t.postId,
      title: t.post.title,
      body: t.post.body,
      platform: t.account.platform,
      socialAccountId: t.socialAccountId,
      externalId: t.externalId,
      publishedAt: t.publishedAt!,
    })),
    failed,
    synced,
  );
});

/** Daily total followers across own accounts (carry-forward per account). */
export const getSocialDailyFollowers = cache(async (clientId: string, range: DateRange): Promise<{ date: string; followers: number }[]> => {
  const accounts = await db.socialAccount.findMany({ where: { clientId, isCompetitor: false }, select: { id: true } });
  if (accounts.length === 0) return [];
  const snapshots = await db.socialSnapshot.findMany({
    where: { socialAccountId: { in: accounts.map((a) => a.id) }, date: { gte: addDays(range.from, -30), lte: range.to } },
    orderBy: { date: "asc" },
    select: { socialAccountId: true, date: true, followers: true },
  });
  const byDay = new Map<string, { socialAccountId: string; followers: number }[]>();
  for (const s of snapshots) {
    const k = dayKey(s.date);
    const list = byDay.get(k) ?? [];
    list.push(s);
    byDay.set(k, list);
  }
  const perAcc = new Map<string, number>();
  const out: { date: string; followers: number }[] = [];
  for (const d of eachDay({ from: addDays(range.from, -30), to: range.to })) {
    const k = dayKey(d);
    for (const s of byDay.get(k) ?? []) perAcc.set(s.socialAccountId, s.followers);
    if (d >= range.from) {
      let sum = 0;
      for (const v of perAcc.values()) sum += v;
      out.push({ date: k, followers: sum });
    }
  }
  return out;
});

export type ReportListRow = {
  id: string;
  title: string;
  dateFrom: Date;
  dateTo: Date;
  modules: string[];
  language: string;
  compare: boolean;
  hasAiSummary: boolean;
  createdByName: string | null;
  createdAt: Date;
};

/** Report history for a client (newest first). */
export const listReports = cache(async (clientId: string): Promise<ReportListRow[]> => {
  const rows = await db.report.findMany({
    where: { clientId },
    orderBy: { createdAt: "desc" },
    include: { createdBy: { select: { name: true } } },
  });
  return rows.map((r) => ({
    id: r.id,
    title: r.title,
    dateFrom: r.dateFrom,
    dateTo: r.dateTo,
    modules: r.modules,
    language: r.language,
    compare: r.compare,
    hasAiSummary: Boolean(r.aiSummary),
    createdByName: r.createdBy?.name ?? null,
    createdAt: r.createdAt,
  }));
});
