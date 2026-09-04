import "server-only";
import { db } from "@/lib/db";
import { getConnectionToken } from "@/lib/connections";
import { isProviderError } from "@/lib/action-result";
import { clientAudience, notify } from "@/lib/notify";
import { isPublisherConfigured, mockPublisher, socialPublisher, type PublishInput } from "@/lib/providers/social-publisher";
import { buildUtmUrl, captionFor, isPlatform, MAX_PUBLISH_ATTEMPTS, parseUtm, resolvePostOutcome, type Platform } from "@/features/publishing/lib";
import { p } from "@/features/publishing/strings";
import { t } from "@/i18n/id";

export type PublishPostResult = {
  status: "PUBLISHED" | "FAILED" | "SCHEDULED";
  published: number;
  failed: number;
  errors: { target: string; error: string }[];
};

export type PublishPostOptions = {
  now?: Date;
  /** Manual "publish now" / retry: no automatic retry, actor recorded on the activity. */
  manual?: boolean;
  userId?: string;
  /** Caller already moved the post to PUBLISHING (cron claim). */
  claimed?: boolean;
  /** Write a per-post SyncJob (cron writes one per run instead). */
  syncJob?: boolean;
};

const CLAIMABLE = ["APPROVED", "SCHEDULED", "FAILED"] as const;

/**
 * Publish one post to every target that is not yet PUBLISHED. Per target:
 * PUBLISHED (externalId, permalink, publishedAt) or FAILED (error, attempts+1).
 * The post becomes PUBLISHED when all targets published, FAILED when any target
 * exhausted its attempts (or the run is manual), else SCHEDULED for the next run.
 */
export async function publishPost(postId: string, opts: PublishPostOptions = {}): Promise<PublishPostResult | null> {
  const now = opts.now ?? new Date();
  if (!opts.claimed) {
    const claim = await db.contentPost.updateMany({
      where: { id: postId, status: { in: [...CLAIMABLE] } },
      data: { status: "PUBLISHING" },
    });
    if (claim.count !== 1) return null;
  }

  const post = await db.contentPost.findUnique({
    where: { id: postId },
    include: {
      client: { select: { id: true, slug: true } },
      targets: { include: { account: { include: { connection: { select: { accessTokenEnc: true } } } } } },
      media: { orderBy: { order: "asc" }, include: { asset: { select: { url: true, kind: true } } } },
    },
  });
  if (!post) return null;

  const href = `/clients/${post.client.slug}/publish/posts/${post.id}`;
  const link = post.linkUrl ? buildUtmUrl(post.linkUrl, parseUtm(post.utm)) : null;
  const media = post.media.map((m) => ({ url: m.asset.url, kind: m.asset.kind }));
  const pending = post.targets.filter((tg) => tg.status !== "PUBLISHED");
  const errors: PublishPostResult["errors"] = [];
  const outcomes: { status: "PUBLISHED" | "FAILED"; attempts: number }[] = post.targets
    .filter((tg) => tg.status === "PUBLISHED")
    .map((tg) => ({ status: "PUBLISHED" as const, attempts: tg.attempts }));

  if (pending.length > 0) {
    await db.postTarget.updateMany({ where: { id: { in: pending.map((tg) => tg.id) } }, data: { status: "PUBLISHING" } });
  }

  for (const tg of pending) {
    const platform: Platform = isPlatform(tg.account.platform) ? tg.account.platform : "INSTAGRAM";
    let text = captionFor(post.body, tg.bodyOverride);
    if (link && !text.includes(link)) text = text.trim() ? `${text.trim()}\n\n${link}` : link;
    // Real publisher only with app credentials AND a usable (non-expired) connection token;
    // an expired Meta token surfaces as TOKEN_EXPIRED on the target instead of a silent mock publish.
    const useReal = isPublisherConfigured() && Boolean(tg.account.connectionId);
    const publisher = useReal ? socialPublisher : mockPublisher;
    try {
      const resolved = useReal ? await getConnectionToken(tg.account.connectionId) : null;
      const input: PublishInput = {
        platform,
        accessToken: resolved?.token ?? "",
        accountExternalId: tg.account.externalId,
        text,
        media,
        linkUrl: link ?? undefined,
        firstComment: platform === "INSTAGRAM" ? (post.firstComment ?? undefined) : undefined,
      };
      const res = await publisher.publish(input);
      await db.postTarget.update({
        where: { id: tg.id },
        data: { status: "PUBLISHED", externalId: res.externalId, permalink: res.permalink, error: null, publishedAt: now, attempts: tg.attempts + 1 },
      });
      outcomes.push({ status: "PUBLISHED", attempts: tg.attempts + 1 });
    } catch (e) {
      const message = isProviderError(e) ? `${t.errors[e.code]} (${e.message})` : e instanceof Error ? e.message : t.errors.UNKNOWN;
      await db.postTarget.update({
        where: { id: tg.id },
        data: { status: "FAILED", error: message.slice(0, 500), attempts: tg.attempts + 1 },
      });
      outcomes.push({ status: "FAILED", attempts: tg.attempts + 1 });
      errors.push({ target: `${platform.toLowerCase()} @${tg.account.username}`, error: message });
    }
  }

  const status = resolvePostOutcome(outcomes, { manual: opts.manual, maxAttempts: MAX_PUBLISH_ATTEMPTS });
  const published = outcomes.filter((o) => o.status === "PUBLISHED").length;
  const failed = outcomes.filter((o) => o.status === "FAILED").length;

  await db.contentPost.update({
    where: { id: post.id },
    data: { status, publishedAt: status === "PUBLISHED" ? now : null },
  });
  await db.postActivity.create({
    data: {
      postId: post.id,
      userId: opts.userId ?? null,
      type: status === "PUBLISHED" ? "PUBLISHED" : status === "FAILED" ? "PUBLISH_FAILED" : "PUBLISH_RETRY",
      meta: { published, failed, errors, manual: Boolean(opts.manual), mock: !isPublisherConfigured() },
    },
  });
  if (opts.syncJob !== false) {
    await db.syncJob.create({
      data: {
        clientId: post.clientId,
        kind: "PUBLISH",
        status: status === "FAILED" ? "FAILED" : "SUCCESS",
        startedAt: now,
        finishedAt: new Date(),
        message: `${post.title || post.id}: ${published} terbit, ${failed} gagal${isPublisherConfigured() ? "" : " (mock)"}`,
        error: errors[0]?.error,
      },
    });
  }

  const title = post.title || post.body.slice(0, 40) || post.id;
  if (status === "PUBLISHED") {
    await notify(await clientAudience(post.clientId), { type: "POST_PUBLISHED", title: p.notif.published(title), href, clientId: post.clientId });
  } else if (status === "FAILED") {
    await notify(await clientAudience(post.clientId), {
      type: "POST_FAILED",
      title: p.notif.failed(title),
      body: errors[0]?.error ?? "",
      href,
      clientId: post.clientId,
    });
  }
  return { status, published, failed, errors };
}

/**
 * Cron entry point (`/api/cron/publish`). Claims due posts atomically — each
 * SCHEDULED row is flipped to PUBLISHING with a conditional `updateMany`, so
 * two overlapping runs can never both own the same post — then publishes them.
 */
export async function publishDuePosts(opts: { now?: Date; limit?: number } = {}): Promise<{ attempted: number; published: number; failed: number }> {
  const now = opts.now ?? new Date();
  const limit = opts.limit ?? 20;
  const candidates = await db.contentPost.findMany({
    where: { status: "SCHEDULED", scheduledAt: { lte: now } },
    orderBy: { scheduledAt: "asc" },
    take: limit,
    select: { id: true, clientId: true },
  });

  const claimed: { id: string; clientId: string }[] = [];
  for (const c of candidates) {
    const r = await db.contentPost.updateMany({
      where: { id: c.id, status: "SCHEDULED", scheduledAt: { lte: now } },
      data: { status: "PUBLISHING" },
    });
    if (r.count === 1) claimed.push(c);
  }
  if (claimed.length === 0) return { attempted: 0, published: 0, failed: 0 };

  // Re-read the claimed ids: only rows we flipped ourselves are processed.
  const owned = await db.contentPost.findMany({ where: { id: { in: claimed.map((c) => c.id) }, status: "PUBLISHING" }, select: { id: true, clientId: true } });

  let published = 0;
  let failed = 0;
  const perClient = new Map<string, { published: number; failed: number; retry: number }>();
  for (const post of owned) {
    const agg = perClient.get(post.clientId) ?? { published: 0, failed: 0, retry: 0 };
    try {
      const res = await publishPost(post.id, { now, claimed: true, syncJob: false });
      if (res?.status === "PUBLISHED") {
        published++;
        agg.published++;
      } else if (res?.status === "FAILED") {
        failed++;
        agg.failed++;
      } else agg.retry++;
    } catch (e) {
      failed++;
      agg.failed++;
      const message = e instanceof Error ? e.message : String(e);
      await db.contentPost.update({ where: { id: post.id }, data: { status: "FAILED" } }).catch(() => undefined);
      await db.postActivity.create({ data: { postId: post.id, type: "PUBLISH_FAILED", meta: { error: message } } }).catch(() => undefined);
    }
    perClient.set(post.clientId, agg);
  }

  for (const [clientId, agg] of perClient) {
    await db.syncJob.create({
      data: {
        clientId,
        kind: "PUBLISH",
        status: agg.failed > 0 && agg.published === 0 ? "FAILED" : "SUCCESS",
        startedAt: now,
        finishedAt: new Date(),
        message: `${agg.published} terbit · ${agg.failed} gagal · ${agg.retry} diulang${isPublisherConfigured() ? "" : " (mock)"}`,
      },
    });
  }
  return { attempted: owned.length, published, failed };
}
