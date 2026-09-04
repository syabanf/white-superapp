"use server";

import { z } from "zod";
import { db } from "@/lib/db";
import { Prisma } from "@/generated/prisma/client";
import { fail, ok, type ActionResult } from "@/lib/action-result";
import { clientApprovers, notify } from "@/lib/notify";
import { issuesToMessage, resolvePublishingAccess, revalidatePublishing, type PublishingAccess } from "@/features/publishing/access";
import {
  canDelete,
  canEdit,
  canReschedule,
  canTransition,
  isPlatform,
  MIN_SCHEDULE_LEAD_MS,
  statusAfterEdit,
  validatePost,
  type Platform,
  type PostStatus,
} from "@/features/publishing/lib";
import { publishPost, type PublishPostResult } from "@/features/publishing/scheduler";
import { p } from "@/features/publishing/strings";
import { t } from "@/i18n/id";

// ── shared ────────────────────────────────────────────────────

const postInclude = {
  targets: { include: { account: { select: { platform: true, isCompetitor: true, clientId: true } } } },
  media: { orderBy: { order: "asc" as const }, include: { asset: { select: { kind: true } } } },
};

type LoadedPost = NonNullable<Awaited<ReturnType<typeof loadPost>>>["post"];

async function loadPost(postId: string) {
  if (!postId || typeof postId !== "string") return null;
  const post = await db.contentPost.findUnique({ where: { id: postId }, include: postInclude });
  if (!post) return null;
  const access = await resolvePublishingAccess(post.clientId);
  if (!access) return null;
  return { post, access, isAuthor: post.createdById === access.userId };
}

function draftInput(post: LoadedPost) {
  return {
    body: post.body,
    targets: post.targets.map((tg) => ({ platform: (isPlatform(tg.account.platform) ? tg.account.platform : "INSTAGRAM") as Platform, bodyOverride: tg.bodyOverride })),
    media: post.media.map((m) => ({ kind: m.asset.kind })),
    scheduledAt: post.scheduledAt,
    linkUrl: post.linkUrl,
    firstComment: post.firstComment,
  };
}

async function setStatus(
  post: LoadedPost,
  access: PublishingAccess,
  status: PostStatus,
  activity: string,
  extra: { data?: Prisma.ContentPostUncheckedUpdateInput; meta?: Prisma.InputJsonValue } = {},
) {
  await db.$transaction([
    db.contentPost.update({ where: { id: post.id }, data: { status, ...(extra.data ?? {}) } }),
    db.postTarget.updateMany({ where: { postId: post.id, status: { not: "PUBLISHED" } }, data: { status } }),
    db.postActivity.create({ data: { postId: post.id, userId: access.userId, type: activity, meta: extra.meta ?? { from: post.status, to: status } } }),
  ]);
  revalidatePublishing(access.slug, post.id);
}

const postTitle = (post: { title: string; body: string; id: string }) => post.title || post.body.slice(0, 40) || post.id;
const postHref = (slug: string, id: string) => `/clients/${slug}/publish/posts/${id}`;

// ── savePost (create / update) ────────────────────────────────

const utmSchema = z
  .object({ source: z.string().max(100), medium: z.string().max(100), campaign: z.string().max(100), term: z.string().max(100), content: z.string().max(100) })
  .partial();

const saveSchema = z.object({
  clientId: z.string().min(1),
  id: z.string().min(1).optional(),
  title: z.string().max(160).default(""),
  body: z.string().max(65_000).default(""),
  targets: z.array(z.object({ socialAccountId: z.string().min(1), bodyOverride: z.string().max(65_000).nullable().optional() })).max(20).default([]),
  mediaIds: z.array(z.string().min(1)).max(10).default([]),
  mediaAlt: z.record(z.string(), z.string().max(500)).optional(),
  linkUrl: z.string().max(2000).nullable().optional(),
  utm: utmSchema.optional(),
  firstComment: z.string().max(2200).nullable().optional(),
  labels: z.array(z.string().trim().min(1).max(40)).max(20).default([]),
  scheduledAt: z.iso.datetime({ offset: true }).nullable().optional(),
});
export type SavePostInput = z.input<typeof saveSchema>;

export async function savePost(input: SavePostInput): Promise<ActionResult<{ id: string; status: PostStatus }>> {
  const parsed = saveSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const d = parsed.data;
  const access = await resolvePublishingAccess(d.clientId);
  if (!access || access.role === "VIEWER") return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");

  const accountIds = Array.from(new Set(d.targets.map((tg) => tg.socialAccountId)));
  const mediaIds = Array.from(new Set(d.mediaIds));
  const [accounts, assets] = await Promise.all([
    db.socialAccount.findMany({ where: { id: { in: accountIds }, clientId: d.clientId, isCompetitor: false }, select: { id: true, platform: true } }),
    db.mediaAsset.findMany({ where: { id: { in: mediaIds }, clientId: d.clientId }, select: { id: true, kind: true } }),
  ]);
  if (accounts.length !== accountIds.length) return fail(p.err.accountNotOwn, "VALIDATION");
  if (assets.length !== mediaIds.length) return fail(p.err.mediaNotOwn, "VALIDATION");
  const platformOf = new Map(accounts.map((a) => [a.id, (isPlatform(a.platform) ? a.platform : "INSTAGRAM") as Platform]));
  const kindOf = new Map(assets.map((a) => [a.id, a.kind]));

  const check = validatePost(
    {
      body: d.body,
      targets: d.targets.map((tg) => ({ platform: platformOf.get(tg.socialAccountId)!, bodyOverride: tg.bodyOverride })),
      media: mediaIds.map((id) => ({ kind: kindOf.get(id)! })),
      linkUrl: d.linkUrl,
      firstComment: d.firstComment,
    },
    { level: "draft" },
  );
  if (!check.ok) return fail(issuesToMessage(p.err.fixIssues, check.issues), "VALIDATION");

  const utm = d.utm && Object.values(d.utm).some(Boolean) ? d.utm : undefined;
  const base = {
    title: d.title.trim(),
    body: d.body,
    linkUrl: d.linkUrl?.trim() || null,
    utm: utm ?? undefined,
    firstComment: d.firstComment?.trim() || null,
    labels: Array.from(new Set(d.labels.map((l) => l.trim()).filter(Boolean))),
    scheduledAt: d.scheduledAt ? new Date(d.scheduledAt) : null,
  };
  const targetRows = d.targets.map((tg) => ({ socialAccountId: tg.socialAccountId, bodyOverride: tg.bodyOverride?.trim() || null }));
  const mediaRows = mediaIds.map((assetId, order) => ({ assetId, order }));
  const altUpdates = Object.entries(d.mediaAlt ?? {})
    .filter(([id]) => kindOf.has(id))
    .map(([id, altText]) => db.mediaAsset.update({ where: { id }, data: { altText: altText.trim() || null } }));

  if (d.id) {
    const existing = await db.contentPost.findFirst({ where: { id: d.id, clientId: d.clientId }, select: { id: true, status: true } });
    if (!existing) return fail(p.err.notFound, "NOT_FOUND");
    if (!canEdit(existing.status, access.role)) return fail(p.cannotEditStatus, "VALIDATION");
    const status = statusAfterEdit(existing.status);
    await db.$transaction([
      db.contentPost.update({
        where: { id: existing.id },
        data: { ...base, status, utm: utm ?? Prisma.JsonNull, ...(status === "DRAFT" && existing.status !== "DRAFT" ? { approvedById: null, approvedAt: null } : {}) },
      }),
      db.postTarget.deleteMany({ where: { postId: existing.id, socialAccountId: { notIn: targetRows.map((r) => r.socialAccountId) } } }),
      ...targetRows.map((r) =>
        db.postTarget.upsert({
          where: { postId_socialAccountId: { postId: existing.id, socialAccountId: r.socialAccountId } },
          create: { postId: existing.id, ...r, status },
          update: { bodyOverride: r.bodyOverride, status },
        }),
      ),
      db.postMedia.deleteMany({ where: { postId: existing.id } }),
      ...(mediaRows.length ? [db.postMedia.createMany({ data: mediaRows.map((m) => ({ postId: existing.id, ...m })) })] : []),
      ...altUpdates,
      db.postActivity.create({ data: { postId: existing.id, userId: access.userId, type: "EDITED", meta: { from: existing.status, to: status } } }),
    ]);
    revalidatePublishing(access.slug, existing.id);
    return ok({ id: existing.id, status });
  }

  const created = await db.contentPost.create({
    data: {
      clientId: d.clientId,
      createdById: access.userId,
      status: "DRAFT",
      ...base,
      targets: { create: targetRows.map((r) => ({ ...r, status: "DRAFT" as const })) },
      media: { create: mediaRows },
      activities: { create: { userId: access.userId, type: "CREATED" } },
    },
    select: { id: true },
  });
  if (altUpdates.length) await db.$transaction(altUpdates);
  revalidatePublishing(access.slug, created.id);
  return ok({ id: created.id, status: "DRAFT" });
}

// ── workflow transitions ──────────────────────────────────────

export async function submitForReview(postId: string): Promise<ActionResult<null>> {
  const loaded = await loadPost(postId);
  if (!loaded) return fail(p.err.notFound, "NOT_FOUND");
  const { post, access, isAuthor } = loaded;
  if (!canTransition(post.status, "IN_REVIEW", { role: access.role, isAuthor })) return fail(p.err.transition, "UNAUTHORIZED");
  const check = validatePost(draftInput(post), { level: "full" });
  if (!check.ok) return fail(issuesToMessage(p.err.fixIssues, check.issues), "VALIDATION");
  await setStatus(post, access, "IN_REVIEW", "SUBMITTED", { data: { reviewNote: null } });
  await notify(await clientApprovers(post.clientId, access.userId), {
    type: "POST_REVIEW",
    title: p.notif.review(postTitle(post)),
    body: p.notif.reviewBody,
    href: postHref(access.slug, post.id),
    clientId: post.clientId,
  });
  return ok(null);
}

export async function approvePost(postId: string): Promise<ActionResult<null>> {
  const loaded = await loadPost(postId);
  if (!loaded) return fail(p.err.notFound, "NOT_FOUND");
  const { post, access, isAuthor } = loaded;
  if (!canTransition(post.status, "APPROVED", { role: access.role, isAuthor })) {
    return fail(access.role === "MANAGER" && isAuthor ? p.selfApprove : p.err.transition, "UNAUTHORIZED");
  }
  const check = validatePost(draftInput(post), { level: "full" });
  if (!check.ok) return fail(issuesToMessage(p.err.fixIssues, check.issues), "VALIDATION");
  await setStatus(post, access, "APPROVED", "APPROVED", { data: { approvedById: access.userId, approvedAt: new Date(), reviewNote: null } });
  if (post.createdById && post.createdById !== access.userId) {
    await notify([post.createdById], { type: "POST_APPROVED", title: p.notif.approved(postTitle(post)), href: postHref(access.slug, post.id), clientId: post.clientId });
  }
  return ok(null);
}

const rejectSchema = z.object({ postId: z.string().min(1), note: z.string().trim().min(1).max(1000) });

export async function rejectPost(input: { postId: string; note: string }): Promise<ActionResult<null>> {
  const parsed = rejectSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const loaded = await loadPost(parsed.data.postId);
  if (!loaded) return fail(p.err.notFound, "NOT_FOUND");
  const { post, access, isAuthor } = loaded;
  if (!canTransition(post.status, "REJECTED", { role: access.role, isAuthor })) {
    return fail(access.role === "MANAGER" && isAuthor ? p.selfApprove : p.err.transition, "UNAUTHORIZED");
  }
  await setStatus(post, access, "REJECTED", "REJECTED", { data: { reviewNote: parsed.data.note }, meta: { note: parsed.data.note } });
  if (post.createdById && post.createdById !== access.userId) {
    await notify([post.createdById], {
      type: "POST_REJECTED",
      title: p.notif.rejected(postTitle(post)),
      body: parsed.data.note,
      href: postHref(access.slug, post.id),
      clientId: post.clientId,
    });
  }
  return ok(null);
}

const scheduleSchema = z.object({ postId: z.string().min(1), scheduledAt: z.iso.datetime({ offset: true }) });

export async function schedulePost(input: { postId: string; scheduledAt: string }): Promise<ActionResult<null>> {
  const parsed = scheduleSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const loaded = await loadPost(parsed.data.postId);
  if (!loaded) return fail(p.err.notFound, "NOT_FOUND");
  const { post, access, isAuthor } = loaded;
  if (!canTransition(post.status, "SCHEDULED", { role: access.role, isAuthor })) return fail(p.err.transition, "UNAUTHORIZED");
  const scheduledAt = new Date(parsed.data.scheduledAt);
  const check = validatePost({ ...draftInput(post), scheduledAt }, { level: "full", requireSchedule: true });
  if (!check.ok) return fail(issuesToMessage(p.err.fixIssues, check.issues), "VALIDATION");
  await setStatus(post, access, "SCHEDULED", "SCHEDULED", { data: { scheduledAt }, meta: { scheduledAt: scheduledAt.toISOString() } });
  return ok(null);
}

export async function unschedulePost(postId: string): Promise<ActionResult<null>> {
  const loaded = await loadPost(postId);
  if (!loaded) return fail(p.err.notFound, "NOT_FOUND");
  const { post, access, isAuthor } = loaded;
  if (post.status !== "SCHEDULED" || !canTransition(post.status, "APPROVED", { role: access.role, isAuthor })) return fail(p.err.transition, "UNAUTHORIZED");
  await setStatus(post, access, "APPROVED", "UNSCHEDULED");
  return ok(null);
}

/** Calendar drag-and-drop / planned-date change. SCHEDULED posts keep the 5-minute lead rule. */
export async function reschedulePost(input: { postId: string; scheduledAt: string }): Promise<ActionResult<null>> {
  const parsed = scheduleSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const loaded = await loadPost(parsed.data.postId);
  if (!loaded) return fail(p.err.notFound, "NOT_FOUND");
  const { post, access } = loaded;
  if (!canReschedule(post.status, access.role)) return fail(p.err.transition, "UNAUTHORIZED");
  const scheduledAt = new Date(parsed.data.scheduledAt);
  if (post.status === "SCHEDULED" && scheduledAt.getTime() < Date.now() + MIN_SCHEDULE_LEAD_MS) {
    return fail(issuesToMessage(p.err.fixIssues, [{ scope: "post", message: "Jadwal minimal 5 menit dari sekarang." }]), "VALIDATION");
  }
  await db.$transaction([
    db.contentPost.update({ where: { id: post.id }, data: { scheduledAt } }),
    db.postActivity.create({
      data: { postId: post.id, userId: access.userId, type: "RESCHEDULED", meta: { from: post.scheduledAt?.toISOString() ?? null, to: scheduledAt.toISOString() } },
    }),
  ]);
  revalidatePublishing(access.slug, post.id);
  return ok(null);
}

/** Back to DRAFT from IN_REVIEW (withdraw), REJECTED (revise), APPROVED or FAILED. */
export async function backToDraft(postId: string): Promise<ActionResult<null>> {
  const loaded = await loadPost(postId);
  if (!loaded) return fail(p.err.notFound, "NOT_FOUND");
  const { post, access, isAuthor } = loaded;
  if (!canTransition(post.status, "DRAFT", { role: access.role, isAuthor })) return fail(p.err.transition, "UNAUTHORIZED");
  await setStatus(post, access, "DRAFT", "BACK_TO_DRAFT", { data: { approvedById: null, approvedAt: null } });
  return ok(null);
}

// ── publishing ────────────────────────────────────────────────

export async function publishNow(postId: string): Promise<ActionResult<PublishPostResult>> {
  const loaded = await loadPost(postId);
  if (!loaded) return fail(p.err.notFound, "NOT_FOUND");
  const { post, access, isAuthor } = loaded;
  if (!canTransition(post.status, "PUBLISHING", { role: access.role, isAuthor })) return fail(p.err.transition, "UNAUTHORIZED");
  if (post.targets.length === 0) return fail(p.err.nothingToPublish, "VALIDATION");
  const check = validatePost(draftInput(post), { level: "full" });
  if (!check.ok) return fail(issuesToMessage(p.err.fixIssues, check.issues), "VALIDATION");
  const res = await publishPost(post.id, { manual: true, userId: access.userId });
  revalidatePublishing(access.slug, post.id);
  if (!res) return fail(p.err.transition, "UNAUTHORIZED");
  return ok(res);
}

export async function retryPost(postId: string): Promise<ActionResult<PublishPostResult>> {
  const loaded = await loadPost(postId);
  if (!loaded) return fail(p.err.notFound, "NOT_FOUND");
  if (loaded.post.status !== "FAILED") return fail(p.err.noFailedTargets, "VALIDATION");
  return publishNow(postId);
}

// ── duplicate / delete / comment ──────────────────────────────

export async function duplicatePost(postId: string): Promise<ActionResult<{ id: string }>> {
  const loaded = await loadPost(postId);
  if (!loaded) return fail(p.err.notFound, "NOT_FOUND");
  const { post, access } = loaded;
  if (access.role === "VIEWER") return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const created = await db.contentPost.create({
    data: {
      clientId: post.clientId,
      createdById: access.userId,
      status: "DRAFT",
      title: post.title ? `${post.title} (salinan)` : "",
      body: post.body,
      linkUrl: post.linkUrl,
      utm: post.utm ?? undefined,
      firstComment: post.firstComment,
      labels: post.labels,
      targets: { create: post.targets.map((tg) => ({ socialAccountId: tg.socialAccountId, bodyOverride: tg.bodyOverride, status: "DRAFT" as const })) },
      media: { create: post.media.map((m) => ({ assetId: m.assetId, order: m.order })) },
      activities: { create: { userId: access.userId, type: "CREATED", meta: { duplicatedFrom: post.id } } },
    },
    select: { id: true },
  });
  revalidatePublishing(access.slug, created.id);
  return ok({ id: created.id });
}

export async function deletePost(postId: string): Promise<ActionResult<null>> {
  const loaded = await loadPost(postId);
  if (!loaded) return fail(p.err.notFound, "NOT_FOUND");
  const { post, access } = loaded;
  if (!canDelete(post.status, access.role)) return fail(p.err.transition, "UNAUTHORIZED");
  await db.contentPost.delete({ where: { id: post.id } });
  revalidatePublishing(access.slug, post.id);
  return ok(null);
}

const commentSchema = z.object({ postId: z.string().min(1), body: z.string().trim().min(1).max(2000) });

export async function addPostComment(input: { postId: string; body: string }): Promise<ActionResult<null>> {
  const parsed = commentSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const loaded = await loadPost(parsed.data.postId);
  if (!loaded) return fail(p.err.notFound, "NOT_FOUND");
  const { post, access } = loaded;
  await db.postComment.create({ data: { postId: post.id, userId: access.userId, body: parsed.data.body } });
  revalidatePublishing(access.slug, post.id);
  return ok(null);
}
