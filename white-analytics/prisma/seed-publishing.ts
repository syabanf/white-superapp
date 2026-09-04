/**
 * Seed — publishing module (media library, content posts, targets, comments,
 * activities) for the 3 demo clients. Idempotent: wipes its own tables first.
 *
 * Called from prisma/seed.ts via `seedPublishing(db)`, or standalone:
 *   pnpm tsx --env-file=.env prisma/seed-publishing.ts
 */
import { PrismaClient, type Prisma } from "../src/generated/prisma/client";
import { PrismaPg } from "@prisma/adapter-pg";
import { SEED_CLIENTS } from "./seed-data";

type Platform = "INSTAGRAM" | "FACEBOOK" | "TIKTOK";
type Status = "DRAFT" | "IN_REVIEW" | "APPROVED" | "SCHEDULED" | "PUBLISHED" | "FAILED";

function mulberry32(seed: number) {
  let a = seed >>> 0;
  return () => {
    a = (a + 0x6d2b79f5) >>> 0;
    let t = a;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

const LABELS = ["promo", "produk", "edukasi", "bts", "testimoni", "event"];
const SAMPLE_VIDEOS = [
  "https://interactive-examples.mdn.mozilla.net/media/cc0-videos/flower.mp4",
  "https://www.w3schools.com/html/mov_bbb.mp4",
];
const HOURS = [9, 11, 13, 17, 19];
const WIB_OFFSET_MS = 7 * 3600_000;

/** Instant for `daysFromToday` at `hour` WIB. */
function atWib(daysFromToday: number, hour: number, minute = 0): Date {
  const now = new Date();
  const dayUtc = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate() + daysFromToday);
  return new Date(dayUtc + hour * 3600_000 + minute * 60_000 - WIB_OFFSET_MS);
}

function permalink(platform: Platform, username: string, id: string): string {
  if (platform === "INSTAGRAM") return `https://www.instagram.com/p/${id.slice(-11)}/`;
  if (platform === "FACEBOOK") return `https://www.facebook.com/${username}/posts/${id}`;
  return `https://www.tiktok.com/@${username}/video/${id}`;
}

// Target pattern per post index: which platforms + what media shape.
const PATTERNS: { platforms: Platform[]; media: "images" | "image" | "none" | "video" }[] = [
  { platforms: ["INSTAGRAM", "FACEBOOK"], media: "images" },
  { platforms: ["INSTAGRAM"], media: "image" },
  { platforms: ["FACEBOOK"], media: "none" },
  { platforms: ["TIKTOK"], media: "video" },
  { platforms: ["INSTAGRAM", "FACEBOOK", "TIKTOK"], media: "video" },
  { platforms: ["INSTAGRAM", "FACEBOOK"], media: "image" },
];

// Future posts (day offset, status) — one FAILED sits in the past.
const FUTURE: { day: number; status: Status }[] = [
  { day: 1, status: "SCHEDULED" },
  { day: 2, status: "SCHEDULED" },
  { day: 3, status: "APPROVED" },
  { day: 4, status: "IN_REVIEW" },
  { day: 6, status: "SCHEDULED" },
  { day: 7, status: "DRAFT" },
  { day: 9, status: "IN_REVIEW" },
  { day: 11, status: "APPROVED" },
  { day: 14, status: "DRAFT" },
  { day: 18, status: "DRAFT" },
  { day: 21, status: "SCHEDULED" },
];

export async function seedPublishing(db: PrismaClient): Promise<void> {
  console.log("→ Publishing: reset");
  await db.postActivity.deleteMany();
  await db.postComment.deleteMany();
  await db.postMedia.deleteMany();
  await db.postTarget.deleteMany();
  await db.contentPost.deleteMany();
  await db.mediaAsset.deleteMany();
  await db.notification.deleteMany({ where: { type: { startsWith: "POST_" } } });

  const [admin, tim] = await Promise.all([
    db.user.findUnique({ where: { email: "admin@white.id" }, select: { id: true } }),
    db.user.findUnique({ where: { email: "tim@white.id" }, select: { id: true } }),
  ]);
  if (!admin) throw new Error("seedPublishing: admin@white.id belum ada — jalankan seed utama dulu");
  const authors = [admin.id, tim?.id ?? admin.id];

  for (const [ci, c] of SEED_CLIENTS.entries()) {
    const client = await db.client.findUnique({ where: { slug: c.slug }, select: { id: true, name: true } });
    if (!client) {
      console.warn(`  · lewati ${c.slug} (klien tidak ada)`);
      continue;
    }
    const rand = mulberry32(90_000 + ci);
    const int = (min: number, max: number) => Math.floor(rand() * (max - min + 1)) + min;
    const pick = <T>(arr: T[]): T => arr[Math.floor(rand() * arr.length)]!;

    const accounts = await db.socialAccount.findMany({
      where: { clientId: client.id, isCompetitor: false },
      select: { id: true, platform: true, username: true },
    });
    const byPlatform = new Map(accounts.map((a) => [a.platform as Platform, a]));

    // ── media library: 10 images + 2 videos ──────────────────
    const images = [];
    for (let i = 0; i < 10; i++) {
      const portrait = i % 3 === 2;
      images.push(
        await db.mediaAsset.create({
          data: {
            clientId: client.id,
            kind: "IMAGE",
            url: `https://picsum.photos/seed/${c.slug}-${i}/${portrait ? 1080 : 1080}/${portrait ? 1350 : 1080}`,
            thumbnailUrl: `https://picsum.photos/seed/${c.slug}-${i}/400/${portrait ? 500 : 400}`,
            filename: `${c.slug}-visual-${String(i + 1).padStart(2, "0")}.jpg`,
            mimeType: "image/jpeg",
            sizeBytes: int(180_000, 950_000),
            width: 1080,
            height: portrait ? 1350 : 1080,
            altText: i % 2 === 0 ? `${c.name} — ${pick(c.captions).slice(0, 60)}` : null,
            tags: [pick(LABELS), pick(LABELS)].filter((v, idx, arr) => arr.indexOf(v) === idx),
            uploadedById: authors[i % 2],
            createdAt: atWib(-int(3, 40), int(8, 18)),
          },
        }),
      );
    }
    const videos = [];
    for (let i = 0; i < 2; i++) {
      videos.push(
        await db.mediaAsset.create({
          data: {
            clientId: client.id,
            kind: "VIDEO",
            url: SAMPLE_VIDEOS[i]!,
            thumbnailUrl: `https://picsum.photos/seed/${c.slug}-video-${i}/400/711`,
            filename: `${c.slug}-reel-${i + 1}.mp4`,
            mimeType: "video/mp4",
            sizeBytes: int(2_500_000, 9_800_000),
            width: 1080,
            height: 1920,
            durationSec: i === 0 ? 12.4 : 10,
            tags: ["reels", pick(LABELS)],
            uploadedById: authors[i % 2],
            createdAt: atWib(-int(3, 40), int(8, 18)),
          },
        }),
      );
    }

    // ── posts ────────────────────────────────────────────────
    const plan: { day: number; status: Status; hour: number }[] = [];
    for (let i = 0; i < 14; i++) plan.push({ day: -(1 + Math.floor((i * 29) / 14)) , status: "PUBLISHED", hour: pick(HOURS) });
    plan.push({ day: -1, status: "FAILED", hour: 20 });
    for (const f of FUTURE) plan.push({ ...f, hour: pick(HOURS) });

    let commentCount = 0;
    for (const [i, item] of plan.entries()) {
      const pattern = PATTERNS[i % PATTERNS.length]!;
      const authorId = authors[i % 2]!;
      const approverId = authorId === admin.id && tim && ci === 0 ? tim.id : admin.id;
      const caption = c.captions[i % c.captions.length]!;
      const tags = c.hashtags.slice(0, int(3, 6)).join(" ");
      const body = item.status === "FAILED" ? `${caption} [fail]\n\n${tags}` : `${caption}\n\n${tags}`;
      const at = atWib(item.day, item.hour, int(0, 1) * 30);
      const hasLink = pattern.media === "none" || i % 4 === 0;
      const mediaAssets =
        pattern.media === "video"
          ? [videos[i % 2]!]
          : pattern.media === "image"
            ? [images[i % 10]!]
            : pattern.media === "images"
              ? [images[i % 10]!, images[(i + 3) % 10]!, images[(i + 6) % 10]!].slice(0, int(2, 3))
              : [];
      const targets = pattern.platforms.map((pl) => byPlatform.get(pl)).filter((a): a is NonNullable<typeof a> => Boolean(a));
      if (targets.length === 0) continue;

      const isPublished = item.status === "PUBLISHED";
      const isFailed = item.status === "FAILED";
      const reviewed = ["APPROVED", "SCHEDULED", "PUBLISHED", "FAILED"].includes(item.status);
      const submitted = reviewed || item.status === "IN_REVIEW";
      const createdAt = new Date(at.getTime() - int(2, 6) * 86_400_000);

      const post = await db.contentPost.create({
        data: {
          clientId: client.id,
          title: caption.split(/[!.?:—]/)[0]!.trim().slice(0, 60),
          body,
          status: item.status,
          scheduledAt: item.status === "DRAFT" && i % 3 === 0 ? null : at,
          publishedAt: isPublished ? at : null,
          linkUrl: hasLink ? `${c.websiteUrl}${c.pages[(i % (c.pages.length - 1)) + 1] ?? "/"}` : null,
          utm: hasLink ? { source: pattern.platforms[0]!.toLowerCase(), medium: "social", campaign: `${c.slug}-${at.toISOString().slice(0, 7)}` } : undefined,
          firstComment: pattern.platforms.includes("INSTAGRAM") && i % 2 === 0 ? c.hashtags.slice(-4).join(" ") : null,
          labels: [LABELS[i % LABELS.length]!, ...(i % 3 === 0 ? [LABELS[(i + 2) % LABELS.length]!] : [])],
          reviewNote: null,
          createdById: authorId,
          approvedById: reviewed ? approverId : null,
          approvedAt: reviewed ? new Date(createdAt.getTime() + 86_400_000) : null,
          createdAt,
          updatedAt: isPublished ? at : createdAt,
          targets: {
            create: targets.map((acc, ti) => {
              const externalId = `${acc.platform.toLowerCase()}_pub_${ci}_${i}_${ti}_${int(100000, 999999)}`;
              return {
                socialAccountId: acc.id,
                bodyOverride: acc.platform === "TIKTOK" && pattern.platforms.length > 1 ? `${caption} #fyp #tiktokindonesia` : null,
                status: isPublished ? "PUBLISHED" : isFailed ? "FAILED" : item.status,
                externalId: isPublished ? externalId : null,
                permalink: isPublished ? permalink(acc.platform as Platform, acc.username, externalId) : null,
                error: isFailed ? "Terjadi kesalahan tak terduga. (Publikasi gagal (simulasi — caption mengandung [fail]))" : null,
                attempts: isPublished ? 1 : isFailed ? 3 : 0,
                publishedAt: isPublished ? at : null,
              };
            }),
          },
          media: { create: mediaAssets.map((m, order) => ({ assetId: m.id, order })) },
        },
        select: { id: true },
      });

      const acts: { type: string; userId: string | null; at: Date; meta?: Prisma.InputJsonValue }[] = [{ type: "CREATED", userId: authorId, at: createdAt }];
      if (submitted) acts.push({ type: "SUBMITTED", userId: authorId, at: new Date(createdAt.getTime() + 3600_000) });
      if (reviewed) acts.push({ type: "APPROVED", userId: approverId, at: new Date(createdAt.getTime() + 86_400_000) });
      if (["SCHEDULED", "PUBLISHED", "FAILED"].includes(item.status)) {
        acts.push({ type: "SCHEDULED", userId: approverId, at: new Date(createdAt.getTime() + 90_000_000), meta: { scheduledAt: at.toISOString() } });
      }
      if (isPublished) acts.push({ type: "PUBLISHED", userId: null, at, meta: { published: targets.length, failed: 0, mock: true } });
      if (isFailed) {
        for (let k = 0; k < 2; k++) acts.push({ type: "PUBLISH_RETRY", userId: null, at: new Date(at.getTime() + k * 300_000), meta: { published: 0, failed: targets.length } });
        acts.push({ type: "PUBLISH_FAILED", userId: null, at: new Date(at.getTime() + 600_000), meta: { published: 0, failed: targets.length, mock: true } });
      }
      await db.postActivity.createMany({ data: acts.map((a) => ({ postId: post.id, userId: a.userId, type: a.type, meta: a.meta, createdAt: a.at })) });

      if (item.status === "IN_REVIEW" || (isPublished && i % 5 === 0)) {
        commentCount += 2;
        await db.postComment.createMany({
          data: [
            { postId: post.id, userId: authorId, body: "Visual sudah final dari tim desain. Mohon dicek captionnya ya.", createdAt: new Date(createdAt.getTime() + 7200_000) },
            { postId: post.id, userId: approverId, body: "Oke, CTA-nya tolong lebih jelas — arahkan ke link di bio.", createdAt: new Date(createdAt.getTime() + 10_800_000) },
          ],
        });
      }
    }
    console.log(`  · ${c.name}: ${images.length + videos.length} media, ${plan.length} post, ${commentCount} komentar`);
  }
}

// ── standalone runner (tsx runs this as CJS, so `require.main` is the guard) ──
if (typeof require !== "undefined" && typeof module !== "undefined" && require.main === module) {
  const adapter = new PrismaPg({ connectionString: process.env.DATABASE_URL });
  const db = new PrismaClient({ adapter });
  seedPublishing(db)
    .then(() => console.log("✓ Publishing seed selesai"))
    .catch((e) => {
      console.error(e);
      process.exit(1);
    })
    .finally(() => db.$disconnect());
}
