import "server-only";
import { cache } from "react";
import type { DateRange } from "@/lib/dates";
import { contentType, type Delta } from "@/lib/metrics";
import { truncate } from "@/lib/format";
import { getOverviewAds, getOverviewSeo, getOverviewSocial } from "@/features/overview/queries";
import { getCampaignRows, getSocialDailyFollowers, getTopPostRows, getTopQueryRows } from "@/features/reports/queries";

/** Serializable view models for the public share page (client chart/table components). */

export type ShareSocialView = {
  followers: number;
  followersDelta: Delta;
  engagementRate: number;
  engagementRateDelta: Delta;
  reach: number;
  reachDelta: Delta;
  posts: number;
  postsDelta: Delta;
  daily: { date: string; followers: number }[];
  topPosts: { id: string; platform: string; username: string; caption: string; type: string; likes: number; comments: number; engagements: number; publishedAt: string }[];
};

export type ShareSeoView = {
  clicks: number;
  clicksDelta: Delta;
  impressions: number;
  impressionsDelta: Delta;
  ctr: number;
  ctrDelta: Delta;
  position: number;
  positionDelta: Delta;
  daily: { date: string; clicks: number; impressions: number }[];
  topQueries: { query: string; clicks: number; impressions: number; ctr: number; position: number }[];
};

export type ShareAdsView = {
  currency: string;
  spend: number;
  spendDelta: Delta;
  results: number;
  resultsDelta: Delta;
  cpr: number;
  cprDelta: Delta;
  ctr: number;
  daily: { date: string; spend: number; results: number }[];
  campaigns: { id: string; name: string; objective: string; spend: number; results: number; resultType: string | null; cpr: number; isConversion: boolean }[];
};

export const getShareSocial = cache(async (clientId: string, range: DateRange, previous: DateRange): Promise<ShareSocialView | null> => {
  const [social, daily, topPosts] = await Promise.all([
    getOverviewSocial(clientId, range, previous),
    getSocialDailyFollowers(clientId, range),
    getTopPostRows(clientId, range, 10),
  ]);
  if (!social.hasData) return null;
  return {
    followers: social.followers,
    followersDelta: social.followersDelta,
    engagementRate: social.engagementRate,
    engagementRateDelta: social.engagementRateDelta,
    reach: social.reach,
    reachDelta: social.reachDelta,
    posts: social.posts,
    postsDelta: social.postsDelta,
    daily,
    topPosts: topPosts.map((p) => ({
      id: p.id,
      platform: p.platform,
      username: p.username,
      caption: truncate(p.caption, 90),
      type: contentType(p),
      likes: p.likes,
      comments: p.comments,
      engagements: p.engagements,
      publishedAt: p.publishedAt.toISOString(),
    })),
  };
});

export const getShareSeo = cache(async (clientId: string, range: DateRange, previous: DateRange): Promise<ShareSeoView | null> => {
  const [seo, topQueries] = await Promise.all([getOverviewSeo(clientId, range, previous), getTopQueryRows(clientId, range, 10)]);
  if (!seo.hasData) return null;
  return {
    clicks: seo.clicks,
    clicksDelta: seo.clicksDelta,
    impressions: seo.impressions,
    impressionsDelta: seo.impressionsDelta,
    ctr: seo.ctr,
    ctrDelta: seo.ctrDelta,
    position: seo.position,
    positionDelta: seo.positionDelta,
    daily: seo.daily,
    topQueries,
  };
});

export const getShareAds = cache(async (clientId: string, range: DateRange, previous: DateRange, currency: string): Promise<ShareAdsView | null> => {
  const [ads, campaigns] = await Promise.all([getOverviewAds(clientId, range, previous), getCampaignRows(clientId, range)]);
  if (!ads.hasData) return null;
  return {
    currency,
    spend: ads.kpis.spend,
    spendDelta: ads.spendDelta,
    results: ads.kpis.results,
    resultsDelta: ads.resultsDelta,
    cpr: ads.kpis.cpr,
    cprDelta: ads.cprDelta,
    ctr: ads.kpis.ctr,
    daily: ads.daily,
    campaigns: campaigns.map((c) => ({
      id: c.id,
      name: c.name,
      objective: c.objective,
      spend: c.spend,
      results: c.results,
      resultType: c.resultType,
      cpr: c.cpr,
      isConversion: c.isConversion,
    })),
  };
});
