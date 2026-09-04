/**
 * Publishing — pure rules (client-safe, no server access). Tested in
 * tests/publishing.test.ts: platform limits, validation, the status state
 * machine, the UTM builder and the publish-outcome resolver.
 */
import type { PostStatus } from "@/generated/prisma/enums";

export type { PostStatus };
export const PLATFORMS = ["INSTAGRAM", "FACEBOOK", "TIKTOK"] as const;
export type Platform = (typeof PLATFORMS)[number];
export type MediaKind = "IMAGE" | "VIDEO";

export type PlatformLimit = {
  caption: number;
  hashtags: number | null;
  /** min/max number of media items */
  media: { min: number; max: number };
  /** every media item must be a video */
  videoOnly: boolean;
};

export const PLATFORM_LIMITS: Record<Platform, PlatformLimit> = {
  INSTAGRAM: { caption: 2200, hashtags: 30, media: { min: 1, max: 10 }, videoOnly: false },
  FACEBOOK: { caption: 63206, hashtags: null, media: { min: 0, max: 10 }, videoOnly: false },
  TIKTOK: { caption: 2200, hashtags: null, media: { min: 1, max: 1 }, videoOnly: true },
};

export const MIN_SCHEDULE_LEAD_MS = 5 * 60 * 1000;
export const MAX_PUBLISH_ATTEMPTS = 3;

export function isPlatform(v: string): v is Platform {
  return (PLATFORMS as readonly string[]).includes(v);
}

export function countHashtags(text: string): number {
  return (text.match(/(^|\s)#[\p{L}\p{N}_]+/gu) ?? []).length;
}

/** Caption actually used for a target: override when non-empty, else master. */
export function captionFor(master: string, override: string | null | undefined): string {
  const o = override?.trim();
  return o ? override! : master;
}

// ── validation ────────────────────────────────────────────────

export type ValidationIssue = { scope: "post" | Platform; field?: string; message: string };
export type ValidationResult = { ok: boolean; issues: ValidationIssue[] };

export type PostDraftInput = {
  body: string;
  targets: { platform: Platform; bodyOverride?: string | null }[];
  media: { kind: MediaKind }[];
  scheduledAt?: Date | null;
  linkUrl?: string | null;
  firstComment?: string | null;
};

export type ValidateOptions = {
  /** "draft" = only hard limits; "full" = everything needed to publish */
  level?: "draft" | "full";
  /** require a valid schedule ≥ 5 minutes ahead */
  requireSchedule?: boolean;
  now?: Date;
};

export const V = {
  noTargets: "Pilih minimal satu akun tujuan.",
  emptyPost: "Isi caption atau tambahkan media.",
  invalidLink: "Tautan harus berupa URL http(s) yang valid.",
  scheduleRequired: "Tentukan jadwal terbit.",
  scheduleTooSoon: "Jadwal minimal 5 menit dari sekarang.",
  captionTooLong: (limit: number) => `Caption melebihi batas ${limit.toLocaleString("id-ID")} karakter.`,
  tooManyHashtags: (limit: number) => `Maksimal ${limit} hashtag.`,
  igNeedsMedia: "Instagram membutuhkan minimal satu gambar atau video.",
  igCarousel: "Carousel Instagram berisi 2–10 gambar.",
  igCarouselVideo: "Carousel Instagram hanya boleh berisi gambar.",
  tiktokOneVideo: "TikTok membutuhkan tepat satu video.",
  fbOneVideo: "Facebook hanya menerima satu video per post.",
  tooManyMedia: (max: number) => `Maksimal ${max} media.`,
  firstCommentTooLong: "Komentar pertama melebihi 2.200 karakter.",
} as const;

export function isHttpUrl(v: string): boolean {
  try {
    const u = new URL(v);
    return u.protocol === "http:" || u.protocol === "https:";
  } catch {
    return false;
  }
}

function validatePlatform(platform: Platform, caption: string, media: { kind: MediaKind }[], full: boolean): ValidationIssue[] {
  const lim = PLATFORM_LIMITS[platform];
  const out: ValidationIssue[] = [];
  if (caption.length > lim.caption) out.push({ scope: platform, field: "body", message: V.captionTooLong(lim.caption) });
  if (lim.hashtags != null && countHashtags(caption) > lim.hashtags) {
    out.push({ scope: platform, field: "body", message: V.tooManyHashtags(lim.hashtags) });
  }
  if (media.length > lim.media.max) out.push({ scope: platform, field: "media", message: V.tooManyMedia(lim.media.max) });
  if (!full) return out;

  const videos = media.filter((m) => m.kind === "VIDEO").length;
  if (platform === "INSTAGRAM") {
    if (media.length === 0) out.push({ scope: platform, field: "media", message: V.igNeedsMedia });
    else if (media.length > 1) {
      if (media.length > 10) out.push({ scope: platform, field: "media", message: V.igCarousel });
      if (videos > 0) out.push({ scope: platform, field: "media", message: V.igCarouselVideo });
    }
  } else if (platform === "TIKTOK") {
    if (media.length !== 1 || videos !== 1) out.push({ scope: platform, field: "media", message: V.tiktokOneVideo });
  } else if (platform === "FACEBOOK") {
    if (videos > 0 && media.length > 1) out.push({ scope: platform, field: "media", message: V.fbOneVideo });
  }
  return out;
}

export function validatePost(input: PostDraftInput, opts: ValidateOptions = {}): ValidationResult {
  const level = opts.level ?? "full";
  const full = level === "full";
  const now = opts.now ?? new Date();
  const issues: ValidationIssue[] = [];

  if (input.linkUrl && input.linkUrl.trim() && !isHttpUrl(input.linkUrl.trim())) {
    issues.push({ scope: "post", field: "linkUrl", message: V.invalidLink });
  }
  if (input.firstComment && input.firstComment.length > 2200) {
    issues.push({ scope: "post", field: "firstComment", message: V.firstCommentTooLong });
  }
  if (full) {
    if (input.targets.length === 0) issues.push({ scope: "post", field: "targets", message: V.noTargets });
    if (input.body.trim().length === 0 && input.media.length === 0) issues.push({ scope: "post", field: "body", message: V.emptyPost });
  }
  if (opts.requireSchedule) {
    if (!input.scheduledAt) issues.push({ scope: "post", field: "scheduledAt", message: V.scheduleRequired });
    else if (input.scheduledAt.getTime() < now.getTime() + MIN_SCHEDULE_LEAD_MS) {
      issues.push({ scope: "post", field: "scheduledAt", message: V.scheduleTooSoon });
    }
  }

  const seen = new Set<string>();
  for (const t of input.targets) {
    const caption = captionFor(input.body, t.bodyOverride);
    const key = `${t.platform}:${caption}`;
    if (seen.has(key)) continue;
    seen.add(key);
    issues.push(...validatePlatform(t.platform, caption, input.media, full));
  }
  return { ok: issues.length === 0, issues };
}

// ── state machine ─────────────────────────────────────────────

export type PostRole = "ADMIN" | "MANAGER" | "VIEWER";
export type TransitionCtx = { role: PostRole; isAuthor: boolean };

/** Human-driven transitions. PUBLISHING → PUBLISHED/FAILED/SCHEDULED is system-only. */
const TRANSITIONS: Record<PostStatus, PostStatus[]> = {
  DRAFT: ["IN_REVIEW", "APPROVED"],
  IN_REVIEW: ["APPROVED", "REJECTED", "DRAFT"],
  REJECTED: ["DRAFT", "IN_REVIEW"],
  APPROVED: ["SCHEDULED", "PUBLISHING", "DRAFT"],
  SCHEDULED: ["APPROVED", "PUBLISHING"],
  PUBLISHING: [],
  PUBLISHED: [],
  FAILED: ["PUBLISHING", "SCHEDULED", "DRAFT"],
};

const REVIEW_TARGETS: PostStatus[] = ["APPROVED", "REJECTED"];

/**
 * ADMIN & MANAGER create/edit/schedule/approve; a MANAGER may not approve or
 * reject their own post (ADMIN may); VIEWER is read-only.
 */
export function canTransition(from: PostStatus, to: PostStatus, ctx: TransitionCtx): boolean {
  if (ctx.role === "VIEWER") return false;
  if (!TRANSITIONS[from].includes(to)) return false;
  if (REVIEW_TARGETS.includes(to) && ctx.role === "MANAGER" && ctx.isAuthor) return false;
  return true;
}

export function allowedTransitions(from: PostStatus, ctx: TransitionCtx): PostStatus[] {
  return TRANSITIONS[from].filter((to) => canTransition(from, to, ctx));
}

/** Content may be edited in these statuses (editing an approved/reviewed post sends it back to DRAFT). */
export const EDITABLE_STATUSES: PostStatus[] = ["DRAFT", "REJECTED", "IN_REVIEW", "APPROVED"];
export function canEdit(status: PostStatus, role: PostRole): boolean {
  return role !== "VIEWER" && EDITABLE_STATUSES.includes(status);
}

/** Calendar drag-and-drop may move these. */
export const RESCHEDULABLE_STATUSES: PostStatus[] = ["DRAFT", "APPROVED", "SCHEDULED"];
export function canReschedule(status: PostStatus, role: PostRole): boolean {
  return role !== "VIEWER" && RESCHEDULABLE_STATUSES.includes(status);
}

export function canDelete(status: PostStatus, role: PostRole): boolean {
  return role !== "VIEWER" && status !== "PUBLISHING";
}

/** Editing a post that was already reviewed resets it to DRAFT (four-eyes). */
export function statusAfterEdit(status: PostStatus): PostStatus {
  return status === "DRAFT" ? "DRAFT" : EDITABLE_STATUSES.includes(status) ? "DRAFT" : status;
}

// ── publish outcome ───────────────────────────────────────────

export type TargetOutcome = { status: PostStatus; attempts: number };

/**
 * Post status after a publish run: PUBLISHED when every target published,
 * FAILED when any target failed and has exhausted its attempts (or the run
 * is manual), otherwise back to SCHEDULED so the next cron run retries.
 */
export function resolvePostOutcome(
  targets: TargetOutcome[],
  opts: { maxAttempts?: number; manual?: boolean } = {},
): "PUBLISHED" | "FAILED" | "SCHEDULED" {
  const max = opts.maxAttempts ?? MAX_PUBLISH_ATTEMPTS;
  if (targets.length > 0 && targets.every((t) => t.status === "PUBLISHED")) return "PUBLISHED";
  const failed = targets.filter((t) => t.status === "FAILED");
  if (failed.length === 0) return targets.length === 0 ? "FAILED" : "SCHEDULED";
  if (opts.manual || failed.some((t) => t.attempts >= max)) return "FAILED";
  return "SCHEDULED";
}

// ── UTM ───────────────────────────────────────────────────────

export type Utm = { source?: string; medium?: string; campaign?: string; term?: string; content?: string };
export const UTM_KEYS = ["source", "medium", "campaign", "term", "content"] as const;

export function parseUtm(raw: unknown): Utm {
  if (!raw || typeof raw !== "object") return {};
  const out: Utm = {};
  for (const k of UTM_KEYS) {
    const v = (raw as Record<string, unknown>)[k];
    if (typeof v === "string" && v.trim()) out[k] = v.trim();
  }
  return out;
}

export function hasUtm(utm: Utm): boolean {
  return UTM_KEYS.some((k) => Boolean(utm[k]));
}

/** Append utm_* params (existing params preserved, existing utm_* overwritten). Invalid URL → returned as-is. */
export function buildUtmUrl(link: string, utm: Utm): string {
  const trimmed = link.trim();
  if (!trimmed || !hasUtm(utm) || !isHttpUrl(trimmed)) return trimmed;
  const u = new URL(trimmed);
  for (const k of UTM_KEYS) {
    const v = utm[k]?.trim();
    if (v) u.searchParams.set(`utm_${k}`, v);
  }
  return u.toString();
}

export function defaultUtm(platform?: Platform): Utm {
  return { source: platform ? platform.toLowerCase() : "social", medium: "social" };
}

// ── status presentation ───────────────────────────────────────

export type StatusTone = "neutral" | "warning" | "info" | "brand" | "good" | "critical";

export const STATUS_TONE: Record<PostStatus, StatusTone> = {
  DRAFT: "neutral",
  IN_REVIEW: "warning",
  APPROVED: "info",
  SCHEDULED: "brand",
  PUBLISHING: "brand",
  PUBLISHED: "good",
  FAILED: "critical",
  REJECTED: "critical",
};

export const STATUS_ORDER: PostStatus[] = ["DRAFT", "IN_REVIEW", "APPROVED", "SCHEDULED", "PUBLISHING", "PUBLISHED", "FAILED", "REJECTED"];

export function isPostStatus(v: string | undefined | null): v is PostStatus {
  return !!v && (STATUS_ORDER as string[]).includes(v);
}

/** The instant a post is (or was) meant to go live, for calendar placement. */
export function postInstant(p: { scheduledAt: string | Date | null; publishedAt: string | Date | null; createdAt: string | Date }): Date {
  const v = p.publishedAt ?? p.scheduledAt ?? p.createdAt;
  return typeof v === "string" ? new Date(v) : v;
}
