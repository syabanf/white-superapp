"use server";

import { revalidatePath } from "next/cache";
import { z } from "zod";
import { db } from "@/lib/db";
import { getSessionUser } from "@/lib/rbac";
import { fail, isProviderError, ok, type ActionResult } from "@/lib/action-result";
import { addDays, latestCompleteDay, todayUtc } from "@/lib/dates";
import { metaGraph } from "@/lib/providers/meta-graph";
import { isApifyConfigured, publicSocial } from "@/lib/providers/apify";
import { syncSocialMeta, syncSocialPublic } from "@/features/social/sync";
import { buildFollowerSeries, hashSeed, mulberry32 } from "@/lib/providers/meta-graph/mock";
import { normalizeIgUsername, PLATFORMS } from "@/features/social/lib";
import { s } from "@/features/social/strings";
import { t } from "@/i18n/id";

const SNAPSHOT_HISTORY_DAYS = 120;

type ClientAccess = { slug: string; canManage: boolean };

/** RBAC di lapisan data: user login + akses ke klien (bukan hanya UI). */
async function clientAccess(clientId: string): Promise<ClientAccess | null> {
  const user = await getSessionUser();
  if (!user) return null;
  const client = await db.client.findUnique({ where: { id: clientId }, select: { slug: true } });
  if (!client) return null;
  if (user.role === "ADMIN") return { slug: client.slug, canManage: true };
  const membership = await db.clientMember.findUnique({
    where: { userId_clientId: { userId: user.id, clientId } },
    select: { role: true },
  });
  if (!membership) return null;
  return { slug: client.slug, canManage: membership.role === "MANAGER" };
}

function revalidateSocial(slug: string) {
  revalidatePath(`/clients/${slug}/social`);
  revalidatePath(`/clients/${slug}/social/competitors`);
}

// ── addCompetitor ─────────────────────────────────────────────

const addCompetitorSchema = z.object({
  clientId: z.string().min(1),
  username: z.string().min(1).max(64),
});

export async function addCompetitor(input: {
  clientId: string;
  username: string;
}): Promise<ActionResult<{ accountId: string; username: string }>> {
  const parsed = addCompetitorSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const username = normalizeIgUsername(parsed.data.username);
  if (!username) return fail(s.usernameInvalid, "VALIDATION");

  const access = await clientAccess(parsed.data.clientId);
  if (!access?.canManage) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const { clientId } = parsed.data;

  const existing = await db.socialAccount.findFirst({
    where: { clientId, platform: "INSTAGRAM", username, isCompetitor: true },
    select: { id: true },
  });
  if (existing) return fail(s.competitorExists, "VALIDATION");

  const own = await db.socialAccount.findFirst({
    where: { clientId, platform: "INSTAGRAM", isCompetitor: false },
    select: { externalId: true },
  });

  try {
    // Public profile via Apify when configured; otherwise Business Discovery
    // through the Meta adapter (deterministic mock without credentials).
    const bd = isApifyConfigured()
      ? await (async () => {
          const [profile, posts] = await Promise.all([publicSocial.profile("INSTAGRAM", username), publicSocial.posts("INSTAGRAM", username, 20)]);
          return {
            username: profile.username,
            name: profile.displayName,
            biography: profile.biography,
            profilePictureUrl: profile.avatarUrl,
            followersCount: profile.followers,
            mediaCount: profile.mediaCount,
            media: posts.map((p) => ({
              id: p.externalId,
              caption: p.caption,
              mediaType: p.mediaType,
              mediaProductType: p.productType,
              permalink: p.permalink,
              mediaUrl: p.mediaUrl,
              thumbnailUrl: p.thumbnailUrl,
              timestamp: p.publishedAt,
              likeCount: p.likes,
              commentsCount: p.comments,
            })),
          };
        })()
      : await metaGraph.discoverBusiness("", own?.externalId ?? "", username);

    const account = await db.socialAccount.create({
      data: {
        clientId,
        platform: "INSTAGRAM",
        externalId: `ig_bd_${username}`,
        username: bd.username,
        displayName: bd.name,
        biography: bd.biography,
        avatarUrl: bd.profilePictureUrl,
        isCompetitor: true,
      },
    });

    // Riwayat snapshot 120 hari yang berakhir tepat di followers hasil discovery
    const end = latestCompleteDay();
    const series = buildFollowerSeries(`bd:${username}`, SNAPSHOT_HISTORY_DAYS, bd.followersCount);
    await db.socialSnapshot.createMany({
      data: series.map((followers, i) => ({
        socialAccountId: account.id,
        date: addDays(end, -(series.length - 1 - i)),
        followers,
        mediaCount: Math.max(0, bd.mediaCount - Math.round((series.length - 1 - i) * 0.3)),
      })),
      skipDuplicates: true,
    });

    // ±20 postingan publik (BD hanya memberi likes/comments)
    await db.socialPost.createMany({
      data: bd.media.slice(0, 20).map((m) => ({
        socialAccountId: account.id,
        externalId: m.id,
        caption: m.caption,
        mediaType: m.mediaType,
        productType: m.mediaProductType,
        permalink: m.permalink,
        mediaUrl: m.mediaUrl,
        thumbnailUrl: m.thumbnailUrl,
        publishedAt: new Date(m.timestamp),
        likes: m.likeCount,
        comments: m.commentsCount,
      })),
      skipDuplicates: true,
    });

    revalidateSocial(access.slug);
    return ok({ accountId: account.id, username: bd.username });
  } catch (e) {
    if (isProviderError(e)) return fail(t.errors[e.code], e.code);
    return fail(t.errors.UNKNOWN);
  }
}

// ── removeCompetitor ──────────────────────────────────────────

export async function removeCompetitor(accountId: string): Promise<ActionResult<null>> {
  if (!accountId || typeof accountId !== "string") return fail(t.errors.VALIDATION, "VALIDATION");
  const account = await db.socialAccount.findUnique({
    where: { id: accountId },
    select: { id: true, clientId: true, isCompetitor: true },
  });
  if (!account || !account.isCompetitor) return fail(t.errors.NOT_FOUND, "NOT_FOUND");
  const access = await clientAccess(account.clientId);
  if (!access?.canManage) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");

  await db.socialAccount.delete({ where: { id: account.id } }); // cascade: snapshot & post
  revalidateSocial(access.slug);
  return ok(null);
}

// ── syncSocial ────────────────────────────────────────────────

const syncSchema = z.object({
  clientId: z.string().min(1),
  platform: z.enum(PLATFORMS),
});

/**
 * Mode demo: upsert snapshot HARI INI untuk akun sendiri dengan ekstrapolasi
 * snapshot terakhir (rata-rata pertumbuhan 7 hari + jitter deterministik),
 * lalu catat SyncJob SOCIAL_SNAPSHOT.
 */
export async function syncSocial(input: {
  clientId: string;
  platform: (typeof PLATFORMS)[number];
}): Promise<ActionResult<{ updated: number }>> {
  const parsed = syncSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const { clientId, platform } = parsed.data;

  const access = await clientAccess(clientId);
  if (!access) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");

  const accounts = await db.socialAccount.findMany({
    where: { clientId, platform, isCompetitor: false },
    select: { id: true, connectionId: true },
  });
  if (accounts.length === 0) return fail(s.syncNoAccount, "NOT_FOUND");

  const today = todayUtc();
  try {
    // 1) Meta Insights for own accounts that carry a Meta connection (reach, impressions, profile views).
    if (platform !== "TIKTOK" && accounts.some((a) => a.connectionId)) {
      const r = await syncSocialMeta({ clientId, platform });
      if (r.updated > 0 || r.failed.length > 0) {
        await db.syncJob.create({
          data: {
            clientId,
            kind: "SOCIAL_SNAPSHOT",
            status: r.updated === 0 ? "FAILED" : "SUCCESS",
            finishedAt: new Date(),
            message: `${r.updated} ${s.accountsUpdated}, ${r.posts} ${s.postsUpdated} (${platform.toLowerCase()} · Meta)`,
            error: r.failed.length ? r.failed.map((f) => `@${f.username}: ${f.error}`).join("; ").slice(0, 1000) : null,
          },
        });
        revalidateSocial(access.slug);
        if (r.updated === 0 && r.failed[0]) return fail(r.failed[0].error);
        return ok({ updated: r.updated });
      }
    }

    // 2) Public data via Apify (any platform, incl. competitors).
    if (isApifyConfigured()) {
      const r = await syncSocialPublic({ clientId, platform, includeCompetitors: true });
      await db.syncJob.create({
        data: {
          clientId,
          kind: "SOCIAL_SNAPSHOT",
          status: r.failed.length && r.updated === 0 ? "FAILED" : "SUCCESS",
          finishedAt: new Date(),
          message: `${r.updated} ${s.accountsUpdated}, ${r.posts} ${s.postsUpdated} (${platform.toLowerCase()} · ${s.viaApify})`,
          error: r.failed.length ? r.failed.map((f) => `@${f.username}: ${f.error}`).join("; ").slice(0, 1000) : null,
        },
      });
      revalidateSocial(access.slug);
      if (r.updated === 0 && r.failed[0]) return fail(r.failed[0].error);
      return ok({ updated: r.updated });
    }

    let updated = 0;
    for (const acc of accounts) {
      const recent = await db.socialSnapshot.findMany({
        where: { socialAccountId: acc.id },
        orderBy: { date: "desc" },
        take: 8,
        select: {
          date: true, followers: true, following: true, mediaCount: true,
          reach: true, impressions: true, profileViews: true, websiteClicks: true,
        },
      });
      const last = recent[0];
      if (!last) continue;

      const diffs: number[] = [];
      for (let i = 0; i < recent.length - 1; i++) diffs.push(recent[i]!.followers - recent[i + 1]!.followers);
      const avgGain = diffs.length
        ? diffs.reduce((a, b) => a + b, 0) / diffs.length
        : Math.max(1, last.followers * 0.001);

      const rand = mulberry32(hashSeed(`sync:${acc.id}:${today.toISOString().slice(0, 10)}`));
      const followers = Math.max(0, Math.round(last.followers + avgGain * (0.7 + rand() * 0.6)));
      const scale = (v: number | null) => (v == null ? null : Math.max(0, Math.round(v * (0.85 + rand() * 0.3))));
      const data = {
        followers,
        following: last.following,
        mediaCount: last.mediaCount + (rand() < 0.3 ? 1 : 0),
        reach: scale(last.reach),
        impressions: scale(last.impressions),
        profileViews: scale(last.profileViews),
        websiteClicks: scale(last.websiteClicks),
      };
      await db.socialSnapshot.upsert({
        where: { socialAccountId_date: { socialAccountId: acc.id, date: today } },
        create: { socialAccountId: acc.id, date: today, ...data },
        update: data,
      });
      updated++;
    }

    await db.syncJob.create({
      data: {
        clientId,
        kind: "SOCIAL_SNAPSHOT",
        status: "SUCCESS",
        finishedAt: new Date(),
        message: `${updated} ${s.accountsUpdated} (${platform.toLowerCase()})`,
      },
    });
    revalidateSocial(access.slug);
    return ok({ updated });
  } catch (e) {
    const message = isProviderError(e) ? t.errors[e.code] : t.errors.UNKNOWN;
    await db.syncJob
      .create({
        data: { clientId, kind: "SOCIAL_SNAPSHOT", status: "FAILED", finishedAt: new Date(), error: message },
      })
      .catch(() => undefined);
    return fail(message, isProviderError(e) ? e.code : undefined);
  }
}
