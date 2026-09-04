import "server-only";
import type { DateRange } from "@/lib/dates";
import { dayCount, toISODate } from "@/lib/dates";
import { truncate } from "@/lib/format";
import { contentType } from "@/lib/metrics";
import { getOverviewAds, getOverviewSeo, getOverviewSocial } from "@/features/overview/queries";
import { getCampaignRows, getTopPostRows, getTopQueryRows, type InsightModule } from "@/features/reports/queries";
import type { InsightPayload, InsightRangeInfo } from "@/lib/providers/openrouter";
import { db } from "@/lib/db";

export function toRangeInfo(range: DateRange, previous: DateRange): InsightRangeInfo {
  return {
    from: toISODate(range.from),
    to: toISODate(range.to),
    days: dayCount(range),
    prevFrom: toISODate(previous.from),
    prevTo: toISODate(previous.to),
  };
}

/**
 * Compact KPI snapshot for the insight provider. OVERVIEW includes every module;
 * SOCIAL/SEO/ADS narrow the payload to their own numbers plus supporting lists.
 * `modules` can further narrow OVERVIEW payloads (report builder module selection).
 */
export async function buildInsightPayload(
  clientId: string,
  module: InsightModule,
  range: DateRange,
  previous: DateRange,
  modules?: string[],
): Promise<InsightPayload> {
  const want = (m: "SOCIAL" | "SEO" | "ADS") =>
    module === m || (module === "OVERVIEW" && (!modules || modules.length === 0 || modules.includes(m)));

  const payload: InsightPayload = {};

  if (want("SOCIAL")) {
    const [social, topPosts] = await Promise.all([getOverviewSocial(clientId, range, previous), getTopPostRows(clientId, range, 5)]);
    if (social.hasData) {
      payload.social = {
        followers: social.followers,
        followerGrowth: social.followersDelta.abs,
        followerGrowthPct: social.followersDelta.pct,
        engagementRate: social.engagementRate,
        engagementRateDeltaPct: social.engagementRateDelta.pct,
        posts: social.posts,
        reach: social.reach,
        reachDeltaPct: social.reachDelta.pct,
        topPosts: topPosts.map((p) => ({
          platform: p.platform,
          caption: truncate(p.caption, 80),
          type: contentType(p),
          engagements: p.engagements,
          likes: p.likes,
          comments: p.comments,
        })),
      };
    }
  }

  if (want("SEO")) {
    const [seo, topQueries] = await Promise.all([getOverviewSeo(clientId, range, previous), getTopQueryRows(clientId, range, module === "SEO" ? 10 : 5)]);
    if (seo.hasData) {
      payload.seo = {
        clicks: seo.clicks,
        clicksDeltaPct: seo.clicksDelta.pct,
        impressions: seo.impressions,
        impressionsDeltaPct: seo.impressionsDelta.pct,
        ctr: seo.ctr,
        position: seo.position,
        positionDeltaAbs: seo.positionDelta.abs,
        health: seo.health,
        topQueries: topQueries.map((q) => ({ query: q.query, clicks: q.clicks, impressions: q.impressions, ctr: q.ctr, position: q.position })),
      };
    }
  }

  if (want("ADS")) {
    const [ads, campaigns, client] = await Promise.all([
      getOverviewAds(clientId, range, previous),
      getCampaignRows(clientId, range),
      db.client.findUnique({ where: { id: clientId }, select: { currency: true } }),
    ]);
    if (ads.hasData) {
      payload.ads = {
        currency: client?.currency ?? "IDR",
        spend: ads.kpis.spend,
        spendDeltaPct: ads.spendDelta.pct,
        results: ads.kpis.results,
        resultsDeltaPct: ads.resultsDelta.pct,
        cpr: ads.kpis.cpr,
        cprDeltaPct: ads.cprDelta.pct,
        ctr: ads.kpis.ctr,
        cpm: ads.kpis.cpm,
        roas: ads.kpis.roas,
        campaigns: campaigns.slice(0, 6).map((c) => ({
          name: c.name,
          objective: c.objective,
          spend: c.spend,
          results: c.results,
          resultType: c.resultType,
          cpr: c.cpr,
        })),
      };
    }
  }

  return payload;
}
