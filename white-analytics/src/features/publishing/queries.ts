import "server-only";
import { cache } from "react";
import { db } from "@/lib/db";
import { postingHeatmap } from "@/lib/metrics";
import { bestPostingTimes, postStatusCounts, type PostStatusKey } from "@/lib/metrics/publishing";
import { truncate } from "@/lib/format";
import { isPlatform, parseUtm, postInstant, type Platform, type PostStatus, type Utm } from "@/features/publishing/lib";
import { nextOccurrences } from "@/features/publishing/time";

// ── types (serializable) ──────────────────────────────────────

export type OwnAccount = {
  id: string;
  platform: Platform;
  username: string;
  displayName: string;
  avatarUrl: string | null;
};

export type MediaRow = {
  id: string;
  kind: "IMAGE" | "VIDEO";
  url: string;
  thumbnailUrl: string | null;
  filename: string;
  mimeType: string;
  sizeBytes: number;
  width: number | null;
  height: number | null;
  altText: string | null;
  tags: string[];
  createdAt: string;
  usageCount: number;
};

export type PostTargetView = {
  id: string;
  socialAccountId: string;
  platform: Platform;
  username: string;
  displayName: string;
  avatarUrl: string | null;
  bodyOverride: string | null;
  status: PostStatus;
  externalId: string | null;
  permalink: string | null;
  error: string | null;
  attempts: number;
  publishedAt: string | null;
};

export type PostActivityView = { id: string; type: string; meta: Record<string, unknown> | null; createdAt: string; user: string | null };
export type PostCommentView = { id: string; body: string; createdAt: string; user: string | null; userId: string | null };

export type PostDetail = {
  id: string;
  clientId: string;
  title: string;
  body: string;
  status: PostStatus;
  scheduledAt: string | null;
  publishedAt: string | null;
  linkUrl: string | null;
  firstComment: string | null;
  labels: string[];
  utm: Utm;
  reviewNote: string | null;
  createdAt: string;
  updatedAt: string;
  createdById: string | null;
  createdBy: string | null;
  approvedBy: string | null;
  approvedAt: string | null;
  targets: PostTargetView[];
  media: MediaRow[];
  comments: PostCommentView[];
  activities: PostActivityView[];
};

export type PostRow = {
  id: string;
  title: string;
  body: string;
  bodyShort: string;
  status: PostStatus;
  /** ISO — scheduledAt ?? publishedAt ?? createdAt */
  at: string;
  scheduledAt: string | null;
  publishedAt: string | null;
  updatedAt: string;
  platforms: Platform[];
  author: string | null;
  thumbnailUrl: string | null;
  failedTargets: number;
};

export type CalendarPost = {
  id: string;
  title: string;
  bodyShort: string;
  status: PostStatus;
  at: string;
  platforms: Platform[];
  thumbnailUrl: string | null;
};

export type BestTimeSlot = { day: number; hour: number; at: string; fromHistory: boolean };

export type PublishingStats = { scheduled7d: number; inReview: number; published30d: number; failed: number; drafts: number };

// ── helpers ───────────────────────────────────────────────────

const iso = (d: Date | null | undefined) => (d ? d.toISOString() : null);

function toPlatform(v: string): Platform {
  return isPlatform(v) ? v : "INSTAGRAM";
}

function mediaRow(a: {
  id: string;
  kind: "IMAGE" | "VIDEO";
  url: string;
  thumbnailUrl: string | null;
  filename: string;
  mimeType: string;
  sizeBytes: number;
  width: number | null;
  height: number | null;
  altText: string | null;
  tags: string[];
  createdAt: Date;
  _count?: { usages: number };
}): MediaRow {
  return {
    id: a.id,
    kind: a.kind,
    url: a.url,
    thumbnailUrl: a.thumbnailUrl,
    filename: a.filename,
    mimeType: a.mimeType,
    sizeBytes: a.sizeBytes,
    width: a.width,
    height: a.height,
    altText: a.altText,
    tags: a.tags,
    createdAt: a.createdAt.toISOString(),
    usageCount: a._count?.usages ?? 0,
  };
}

function uniquePlatforms(targets: { account: { platform: string } }[]): Platform[] {
  return Array.from(new Set(targets.map((t) => toPlatform(t.account.platform))));
}

// ── queries ───────────────────────────────────────────────────

export const getOwnAccounts = cache(async (clientId: string): Promise<OwnAccount[]> => {
  const rows = await db.socialAccount.findMany({
    where: { clientId, isCompetitor: false },
    orderBy: [{ platform: "asc" }, { username: "asc" }],
    select: { id: true, platform: true, username: true, displayName: true, avatarUrl: true },
  });
  return rows.map((r) => ({ ...r, platform: toPlatform(r.platform) }));
});

/** Three "best time" suggestions from the client's own posting history (defaults when thin). */
export const getBestTimes = cache(async (clientId: string, timeZone: string, now: Date = new Date()): Promise<BestTimeSlot[]> => {
  const posts = await db.socialPost.findMany({
    where: { account: { clientId, isCompetitor: false } },
    select: { likes: true, comments: true, shares: true, saves: true, publishedAt: true },
    orderBy: { publishedAt: "desc" },
    take: 400,
  });
  const best = bestPostingTimes(postingHeatmap(posts, timeZone));
  const ats = nextOccurrences(best, timeZone, now);
  return best.map((b, i) => ({ day: b.day, hour: b.hour, at: ats[i]!.toISOString(), fromHistory: b.score > 0 }));
});

export const listMedia = cache(async (clientId: string): Promise<MediaRow[]> => {
  const rows = await db.mediaAsset.findMany({
    where: { clientId },
    orderBy: { createdAt: "desc" },
    include: { _count: { select: { usages: true } } },
  });
  return rows.map(mediaRow);
});

export const getPostDetail = cache(async (clientId: string, postId: string): Promise<PostDetail | null> => {
  const post = await db.contentPost.findFirst({
    where: { id: postId, clientId },
    include: {
      createdBy: { select: { name: true } },
      approvedBy: { select: { name: true } },
      targets: { include: { account: { select: { platform: true, username: true, displayName: true, avatarUrl: true } } } },
      media: { orderBy: { order: "asc" }, include: { asset: { include: { _count: { select: { usages: true } } } } } },
      comments: { orderBy: { createdAt: "asc" }, include: { user: { select: { name: true } } } },
      activities: { orderBy: { createdAt: "desc" }, include: { user: { select: { name: true } } } },
    },
  });
  if (!post) return null;
  return {
    id: post.id,
    clientId: post.clientId,
    title: post.title,
    body: post.body,
    status: post.status,
    scheduledAt: iso(post.scheduledAt),
    publishedAt: iso(post.publishedAt),
    linkUrl: post.linkUrl,
    firstComment: post.firstComment,
    labels: post.labels,
    utm: parseUtm(post.utm),
    reviewNote: post.reviewNote,
    createdAt: post.createdAt.toISOString(),
    updatedAt: post.updatedAt.toISOString(),
    createdById: post.createdById,
    createdBy: post.createdBy?.name ?? null,
    approvedBy: post.approvedBy?.name ?? null,
    approvedAt: iso(post.approvedAt),
    targets: post.targets
      .map((t) => ({
        id: t.id,
        socialAccountId: t.socialAccountId,
        platform: toPlatform(t.account.platform),
        username: t.account.username,
        displayName: t.account.displayName,
        avatarUrl: t.account.avatarUrl,
        bodyOverride: t.bodyOverride,
        status: t.status,
        externalId: t.externalId,
        permalink: t.permalink,
        error: t.error,
        attempts: t.attempts,
        publishedAt: iso(t.publishedAt),
      }))
      .sort((a, b) => a.platform.localeCompare(b.platform)),
    media: post.media.map((m) => mediaRow(m.asset)),
    comments: post.comments.map((c) => ({ id: c.id, body: c.body, createdAt: c.createdAt.toISOString(), user: c.user?.name ?? null, userId: c.userId })),
    activities: post.activities.map((a) => ({
      id: a.id,
      type: a.type,
      meta: (a.meta as Record<string, unknown> | null) ?? null,
      createdAt: a.createdAt.toISOString(),
      user: a.user?.name ?? null,
    })),
  };
});

export const listPosts = cache(async (clientId: string): Promise<{ rows: PostRow[]; counts: Record<PostStatusKey, number> }> => {
  const posts = await db.contentPost.findMany({
    where: { clientId },
    orderBy: [{ scheduledAt: "desc" }, { createdAt: "desc" }],
    include: {
      createdBy: { select: { name: true } },
      targets: { select: { status: true, account: { select: { platform: true } } } },
      media: { orderBy: { order: "asc" }, take: 1, select: { asset: { select: { url: true, thumbnailUrl: true } } } },
    },
  });
  const rows: PostRow[] = posts.map((p) => ({
    id: p.id,
    title: p.title,
    body: p.body,
    bodyShort: truncate(p.body.replace(/\s+/g, " "), 90),
    status: p.status,
    at: postInstant(p).toISOString(),
    scheduledAt: iso(p.scheduledAt),
    publishedAt: iso(p.publishedAt),
    updatedAt: p.updatedAt.toISOString(),
    platforms: uniquePlatforms(p.targets),
    author: p.createdBy?.name ?? null,
    thumbnailUrl: p.media[0]?.asset.thumbnailUrl ?? p.media[0]?.asset.url ?? null,
    failedTargets: p.targets.filter((t) => t.status === "FAILED").length,
  }));
  return { rows, counts: postStatusCounts(posts) };
});

/** Posts whose instant falls in `[from, to)` (UTC bounds of client-tz calendar days). */
export const getCalendarPosts = cache(async (clientId: string, from: Date, to: Date): Promise<CalendarPost[]> => {
  const posts = await db.contentPost.findMany({
    where: {
      clientId,
      OR: [
        { publishedAt: { gte: from, lt: to } },
        { publishedAt: null, scheduledAt: { gte: from, lt: to } },
        { publishedAt: null, scheduledAt: null, createdAt: { gte: from, lt: to } },
      ],
    },
    include: {
      targets: { select: { account: { select: { platform: true } } } },
      media: { orderBy: { order: "asc" }, take: 1, select: { asset: { select: { url: true, thumbnailUrl: true } } } },
    },
  });
  return posts
    .map((p) => ({
      id: p.id,
      title: p.title,
      bodyShort: truncate(p.body.replace(/\s+/g, " "), 60),
      status: p.status,
      at: postInstant(p).toISOString(),
      platforms: uniquePlatforms(p.targets),
      thumbnailUrl: p.media[0]?.asset.thumbnailUrl ?? p.media[0]?.asset.url ?? null,
    }))
    .sort((a, b) => a.at.localeCompare(b.at));
});

export const getPublishingStats = cache(async (clientId: string, now: Date = new Date()): Promise<PublishingStats> => {
  const in7d = new Date(now.getTime() + 7 * 86_400_000);
  const ago30d = new Date(now.getTime() - 30 * 86_400_000);
  const [scheduled7d, inReview, published30d, failed, drafts] = await Promise.all([
    db.contentPost.count({ where: { clientId, status: "SCHEDULED", scheduledAt: { gte: now, lt: in7d } } }),
    db.contentPost.count({ where: { clientId, status: "IN_REVIEW" } }),
    db.contentPost.count({ where: { clientId, status: "PUBLISHED", publishedAt: { gte: ago30d } } }),
    db.contentPost.count({ where: { clientId, status: "FAILED" } }),
    db.contentPost.count({ where: { clientId, status: "DRAFT" } }),
  ]);
  return { scheduled7d, inReview, published30d, failed, drafts };
});
