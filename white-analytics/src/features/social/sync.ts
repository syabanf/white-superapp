/**
 * Social sync through public data (Apify). Auth-free write path shared by the
 * "Sinkronkan" action and the daily cron. For every account of the platform
 * (own + competitors): today's SocialSnapshot (followers / following / media
 * count — reach & impressions stay null, they are not public) and the latest
 * posts upserted with fresh engagement numbers.
 */
import "server-only";
import { db } from "@/lib/db";
import { todayUtc } from "@/lib/dates";
import { publicSocial, type PublicPlatform } from "@/lib/providers/apify";

export type SocialSyncResult = { updated: number; posts: number; failed: { username: string; error: string }[] };

export async function syncSocialPublic(opts: { clientId: string; platform: PublicPlatform; includeCompetitors?: boolean; postLimit?: number }): Promise<SocialSyncResult> {
  const accounts = await db.socialAccount.findMany({
    where: { clientId: opts.clientId, platform: opts.platform, ...(opts.includeCompetitors === false ? { isCompetitor: false } : {}) },
    select: { id: true, username: true, isCompetitor: true },
  });
  const today = todayUtc();
  const result: SocialSyncResult = { updated: 0, posts: 0, failed: [] };

  for (const acc of accounts) {
    try {
      const [profile, posts] = await Promise.all([
        publicSocial.profile(opts.platform, acc.username),
        publicSocial.posts(opts.platform, acc.username, opts.postLimit ?? (acc.isCompetitor ? 20 : 30)),
      ]);
      await db.$transaction([
        db.socialAccount.update({
          where: { id: acc.id },
          data: { displayName: profile.displayName, biography: profile.biography, avatarUrl: profile.avatarUrl ?? undefined },
        }),
        db.socialSnapshot.upsert({
          where: { socialAccountId_date: { socialAccountId: acc.id, date: today } },
          create: { socialAccountId: acc.id, date: today, followers: profile.followers, following: profile.following, mediaCount: profile.mediaCount },
          update: { followers: profile.followers, following: profile.following, mediaCount: profile.mediaCount },
        }),
        ...posts.map((p) =>
          db.socialPost.upsert({
            where: { socialAccountId_externalId: { socialAccountId: acc.id, externalId: p.externalId } },
            create: {
              socialAccountId: acc.id,
              externalId: p.externalId,
              caption: p.caption,
              mediaType: p.mediaType,
              productType: p.productType,
              permalink: p.permalink,
              mediaUrl: p.mediaUrl,
              thumbnailUrl: p.thumbnailUrl,
              publishedAt: new Date(p.publishedAt),
              likes: p.likes,
              comments: p.comments,
              shares: p.shares,
              saves: p.saves,
              views: p.views,
            },
            update: { caption: p.caption, likes: p.likes, comments: p.comments, shares: p.shares, saves: p.saves, views: p.views, thumbnailUrl: p.thumbnailUrl ?? undefined, syncedAt: new Date() },
          }),
        ),
      ]);
      result.updated++;
      result.posts += posts.length;
    } catch (e) {
      result.failed.push({ username: acc.username, error: e instanceof Error ? e.message : String(e) });
    }
  }
  return result;
}

// ── Meta Insights (own accounts with a Meta connection) ─────────

import { addDays, latestCompleteDay, parseISODate } from "@/lib/dates";
import { getConnectionToken } from "@/lib/connections";
import { metaGraph, type MetaGraphProvider } from "@/lib/providers/meta-graph";

export type SocialMetaSyncResult = { updated: number; posts: number; skipped: number; failed: { username: string; error: string }[] };

/**
 * Real Meta Graph sync for the client's OWN Instagram / Facebook accounts that
 * carry a Meta connection: account profile, daily insights (reach, impressions,
 * profile views) for the last `days`, and media with per-post insights.
 * Accounts without a connection are counted as `skipped` (Apify/demo cover them).
 */
export async function syncSocialMeta(opts: { clientId: string; platform: "INSTAGRAM" | "FACEBOOK"; days?: number; provider?: MetaGraphProvider; token?: string }): Promise<SocialMetaSyncResult> {
  const accounts = await db.socialAccount.findMany({
    where: { clientId: opts.clientId, platform: opts.platform, isCompetitor: false },
    select: { id: true, externalId: true, username: true, connectionId: true },
  });
  const provider = opts.provider ?? metaGraph;
  const until = latestCompleteDay();
  const since = addDays(until, -((opts.days ?? 30) - 1));
  const today = todayUtc();
  const result: SocialMetaSyncResult = { updated: 0, posts: 0, skipped: 0, failed: [] };

  for (const acc of accounts) {
    try {
      const resolved = opts.token !== undefined ? { token: opts.token } : await getConnectionToken(acc.connectionId);
      if (!resolved) {
        result.skipped++;
        continue;
      }
      const token = resolved.token;
      if (opts.platform === "INSTAGRAM") {
        const [account, insights, media] = await Promise.all([
          provider.getInstagramAccount(token, acc.externalId),
          provider.getInstagramInsights(token, acc.externalId, since, until),
          provider.getInstagramMedia(token, acc.externalId, 50),
        ]);
        await db.socialAccount.update({
          where: { id: acc.id },
          data: { username: account.username, displayName: account.name, biography: account.biography, avatarUrl: account.profilePictureUrl ?? undefined },
        });
        for (const d of insights) {
          const date = parseISODate(d.date);
          if (!date) continue;
          const data = { reach: d.reach, impressions: d.impressions, profileViews: d.profileViews };
          await db.socialSnapshot.upsert({
            where: { socialAccountId_date: { socialAccountId: acc.id, date } },
            create: { socialAccountId: acc.id, date, followers: account.followersCount, following: account.followsCount, mediaCount: account.mediaCount, ...data },
            update: data,
          });
        }
        await db.socialSnapshot.upsert({
          where: { socialAccountId_date: { socialAccountId: acc.id, date: today } },
          create: { socialAccountId: acc.id, date: today, followers: account.followersCount, following: account.followsCount, mediaCount: account.mediaCount },
          update: { followers: account.followersCount, following: account.followsCount, mediaCount: account.mediaCount },
        });
        for (const m of media) {
          const data = {
            caption: m.caption, mediaType: m.mediaType, productType: m.mediaProductType, permalink: m.permalink, mediaUrl: m.mediaUrl, thumbnailUrl: m.thumbnailUrl,
            likes: m.likeCount, comments: m.commentsCount, shares: m.shares ?? 0, saves: m.saved ?? 0, views: m.views ?? 0, reach: m.reach ?? 0, impressions: m.impressions ?? 0, syncedAt: new Date(),
          };
          await db.socialPost.upsert({
            where: { socialAccountId_externalId: { socialAccountId: acc.id, externalId: m.id } },
            create: { socialAccountId: acc.id, externalId: m.id, publishedAt: new Date(m.timestamp), ...data },
            update: data,
          });
        }
        result.posts += media.length;
      } else {
        const [page, insights, posts] = await Promise.all([
          provider.getFacebookPage(token, acc.externalId),
          provider.getPageInsights(token, acc.externalId, since, until),
          provider.getPagePosts(token, acc.externalId, 50),
        ]);
        await db.socialAccount.update({ where: { id: acc.id }, data: { username: page.username ?? acc.username, displayName: page.name } });
        for (const d of insights) {
          const date = parseISODate(d.date);
          if (!date) continue;
          await db.socialSnapshot.upsert({
            where: { socialAccountId_date: { socialAccountId: acc.id, date } },
            create: { socialAccountId: acc.id, date, followers: page.followersCount, reach: d.impressionsUnique },
            update: { reach: d.impressionsUnique },
          });
        }
        await db.socialSnapshot.upsert({
          where: { socialAccountId_date: { socialAccountId: acc.id, date: today } },
          create: { socialAccountId: acc.id, date: today, followers: page.followersCount },
          update: { followers: page.followersCount },
        });
        for (const p of posts) {
          const data = { caption: p.message, permalink: p.permalinkUrl, likes: p.reactions, comments: p.comments, shares: p.shares, syncedAt: new Date() };
          await db.socialPost.upsert({
            where: { socialAccountId_externalId: { socialAccountId: acc.id, externalId: p.id } },
            create: { socialAccountId: acc.id, externalId: p.id, publishedAt: new Date(p.createdTime), mediaType: "IMAGE", ...data },
            update: data,
          });
        }
        result.posts += posts.length;
      }
      result.updated++;
    } catch (e) {
      result.failed.push({ username: acc.username, error: e instanceof Error ? e.message : String(e) });
    }
  }
  return result;
}
