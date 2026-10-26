/**
 * Roll the demo data forward to yesterday without touching anything that exists.
 *
 * The seed writes 120 days ending on the day it ran. A week later every chart ends
 * in a cliff and every delta compares against empty days. This script continues
 * each daily series (one social account, one GSC dimension key, one ad, one tracked
 * keyword, ...) from the seed's last day to yesterday:
 *   - weekly fields copy the same weekday one week earlier, with a small jitter,
 *     so weekend dips and campaign rhythms stay intact;
 *   - running totals (followers, backlinks, ...) repeat last week's daily change,
 *     so growth continues instead of sawing back.
 * Only series that were active on the seed's last day are extended (ended campaigns
 * stay ended). Inserts use skipDuplicates, so real synced rows always win and the
 * script is safe to re-run. Run: pnpm db:extend-demo
 */
import { PrismaClient, type Prisma } from "../src/generated/prisma/client";
import { PrismaPg } from "@prisma/adapter-pg";

const db = new PrismaClient({ adapter: new PrismaPg({ connectionString: process.env.DATABASE_URL }) });

const DAY = 86_400_000;
const addDays = (d: Date, n: number) => new Date(d.getTime() + n * DAY);
const key = (d: Date) => d.toISOString().slice(0, 10);

function rng(seed: string): () => number {
  let h = 2166136261;
  for (let i = 0; i < seed.length; i++) h = Math.imul(h ^ seed.charCodeAt(i), 16777619);
  let a = h >>> 0;
  return () => {
    a = (a + 0x6d2b79f5) >>> 0;
    let t = a;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

type Row = Record<string, unknown> & { date: Date };
type Ctx = { src: Row; srcPrev: Row | undefined; prev: Row; rand: () => number };

/** Same weekday last week, jittered ±8 %. */
const weekly = (c: Ctx, f: string) => {
  const v = c.src[f];
  return typeof v === "number" ? Math.max(0, v * (0.92 + c.rand() * 0.16)) : v;
};
/** Yesterday plus last week's daily change for this weekday. */
const trend = (c: Ctx, f: string) => {
  const prev = c.prev[f];
  const a = c.src[f];
  const b = c.srcPrev?.[f];
  if (typeof prev !== "number") return prev;
  const step = typeof a === "number" && typeof b === "number" ? a - b : 0;
  return Math.max(0, prev + step);
};
const int = (v: unknown) => (typeof v === "number" ? Math.round(v) : v);

/**
 * Extend every series of one table. `make` builds the new row's data from the
 * weekly source row, yesterday's row and a deterministic random stream.
 */
async function extend(
  name: string,
  rows: Row[],
  seriesOf: (r: Row) => string,
  seedEnd: Date,
  until: Date,
  make: (c: Ctx, day: Date) => Record<string, unknown>,
  insert: (data: Record<string, unknown>[]) => Promise<{ count: number }>,
): Promise<void> {
  const series = new Map<string, Map<string, Row>>();
  for (const r of rows) {
    const s = seriesOf(r);
    if (!series.has(s)) series.set(s, new Map());
    series.get(s)!.set(key(r.date), r);
  }
  const creates: Record<string, unknown>[] = [];
  for (const [s, byDay] of series) {
    if (!byDay.has(key(seedEnd))) continue; // inactive at seed end: leave it
    for (let d = addDays(seedEnd, 1); d <= until; d = addDays(d, 1)) {
      if (byDay.has(key(d))) continue;
      const prev = byDay.get(key(addDays(d, -1)))!;
      const src = byDay.get(key(addDays(d, -7))) ?? prev;
      const srcPrev = byDay.get(key(addDays(d, -8)));
      const data = { ...make({ src, srcPrev, prev, rand: rng(`${name}|${s}|${key(d)}`) }, d), date: d };
      creates.push(data);
      byDay.set(key(d), data as Row);
    }
  }
  let inserted = 0;
  for (let i = 0; i < creates.length; i += 1000) inserted += (await insert(creates.slice(i, i + 1000))).count;
  console.log(`  · ${name}: ${inserted} baris`);
}

/**
 * Posts are events, not a daily series: replay each account's last eight weeks
 * of posts, shifted by whole eight-week blocks, until yesterday. Weekday and hour
 * stay the same, metrics are jittered, and the copy gets a derived external id so
 * re-runs skip it.
 */
async function extendPosts(until: Date): Promise<void> {
  const last = (await db.socialPost.aggregate({ _max: { publishedAt: true } }))._max.publishedAt;
  if (!last || key(last) >= key(until)) return;
  const block = 56 * DAY;
  const source = await db.socialPost.findMany({ where: { publishedAt: { gt: new Date(last.getTime() - block) } } });
  const metrics = ["likes", "comments", "shares", "saves", "views", "reach", "impressions"] as const;
  const creates: Prisma.SocialPostCreateManyInput[] = [];
  for (let k = 1; last.getTime() + (k - 1) * block <= until.getTime(); k++) {
    for (const p of source) {
      const at = new Date(p.publishedAt.getTime() + k * block);
      if (at <= last || key(at) > key(until)) continue;
      const rand = rng(`SocialPost|${p.id}|${k}`);
      const row: Prisma.SocialPostCreateManyInput = {
        ...p,
        id: undefined,
        syncedAt: undefined,
        externalId: `${p.externalId}-r${k}`,
        publishedAt: at,
      };
      for (const m of metrics) row[m] = Math.round(p[m] * (0.8 + rand() * 0.4));
      creates.push(row);
    }
  }
  const { count } = await db.socialPost.createMany({ data: creates, skipDuplicates: true });
  console.log(`  · SocialPost: ${count} baris`);
}

/**
 * Publishing posts: replay the seed's posts in blocks the length of the seed
 * span (whole weeks, so weekdays and times stay put) until two weeks ahead.
 * A copy landing in the past is published (failures stay failed); one in the
 * future is scheduled, or keeps its draft/review/approved state. Copies carry
 * deterministic ids, so re-runs only add what lies past the last copy.
 */
async function extendPublishing(now: Date): Promise<void> {
  const isCopy = (id: string) => /-d\d+$/.test(id);
  const at = (po: { scheduledAt: Date | null; publishedAt: Date | null; createdAt: Date }) =>
    po.scheduledAt ?? po.publishedAt ?? po.createdAt;
  const horizon = addDays(now, 14);
  const posts = await db.contentPost.findMany({ include: { targets: true, media: true } });
  const byClient = new Map<string, typeof posts>();
  for (const po of posts) byClient.set(po.clientId, [...(byClient.get(po.clientId) ?? []), po]);

  let created = 0;
  for (const list of byClient.values()) {
    // Posts made in the app today (QA, real use) are neither source nor anchor.
    const source = list.filter((po) => !isCopy(po.id) && at(po) <= addDays(now, -1));
    if (source.length === 0) continue;
    const first = Math.min(...source.map((po) => at(po).getTime()));
    const seedEnd = Math.max(...source.map((po) => at(po).getTime()));
    const block = Math.ceil((seedEnd - first) / (7 * DAY) + 1) * 7 * DAY;
    const last = Math.max(seedEnd, ...list.filter((po) => isCopy(po.id)).map((po) => at(po).getTime()));

    for (let k = 1; first + k * block <= horizon.getTime(); k++) {
      for (const src of source) {
        const when = new Date(at(src).getTime() + k * block);
        if (when.getTime() <= last || when > horizon) continue;
        const past = when <= now;
        const status = past
          ? src.status === "FAILED" ? "FAILED" : "PUBLISHED"
          : ["PUBLISHED", "SCHEDULED", "FAILED"].includes(src.status) ? "SCHEDULED" : src.status;
        const id = `${src.id}-d${k}`;
        await db.contentPost.create({
          data: {
            id,
            clientId: src.clientId,
            title: src.title,
            body: src.body,
            status,
            scheduledAt: when,
            publishedAt: status === "PUBLISHED" ? when : null,
            linkUrl: src.linkUrl,
            firstComment: src.firstComment,
            labels: src.labels,
            utm: (src.utm ?? undefined) as Prisma.InputJsonValue | undefined,
            createdById: src.createdById,
            approvedById: ["APPROVED", "SCHEDULED", "PUBLISHED", "FAILED"].includes(status) ? src.approvedById : null,
            approvedAt: ["APPROVED", "SCHEDULED", "PUBLISHED", "FAILED"].includes(status) ? addDays(when, -2) : null,
            createdAt: addDays(when, -5),
            targets: {
              create: src.targets.map((t) => ({
                id: `${t.id}-d${k}`,
                socialAccountId: t.socialAccountId,
                bodyOverride: t.bodyOverride,
                status,
                error: status === "FAILED" ? t.error : null,
                attempts: status === "FAILED" ? t.attempts : 0,
                publishedAt: status === "PUBLISHED" ? when : null,
              })),
            },
            media: { create: src.media.map((m) => ({ assetId: m.assetId, order: m.order })) },
          },
        });
        created++;
      }
    }
  }
  if (created) console.log(`  · ContentPost: ${created} baris`);
}

async function main() {
  const now = new Date();
  const until = addDays(new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate())), -1);
  await extendPosts(until);
  await extendPublishing(now);
  const last = (await db.seoDailyMetric.aggregate({ _max: { date: true } }))._max.date;
  if (!last) return console.log("Belum ada data seed. Jalankan pnpm db:seed dulu.");
  if (last >= until) return console.log(`Data sudah sampai ${key(last)}. Tidak ada yang ditambah.`);
  const seedEnd = last;
  const since = addDays(seedEnd, -13); // two weeks of history feeds the weekly copy
  const window = { date: { gte: since } };
  console.log(`→ Menambah data demo ${key(addDays(seedEnd, 1))} … ${key(until)}`);

  await extend(
    "SocialSnapshot",
    (await db.socialSnapshot.findMany({ where: window })) as Row[],
    (r) => String(r.socialAccountId),
    seedEnd,
    until,
    (c) => ({
      socialAccountId: c.prev.socialAccountId,
      followers: int(trend(c, "followers")),
      following: int(c.prev.following),
      mediaCount: int(trend(c, "mediaCount")),
      reach: int(weekly(c, "reach")),
      impressions: int(weekly(c, "impressions")),
      profileViews: int(weekly(c, "profileViews")),
      websiteClicks: int(weekly(c, "websiteClicks")),
    }),
    (data) => db.socialSnapshot.createMany({ data: data as Prisma.SocialSnapshotCreateManyInput[], skipDuplicates: true }),
  );

  const seoRow = (c: Ctx) => {
    const clicks = Math.round(weekly(c, "clicks") as number);
    const impressions = Math.max(clicks, Math.round(weekly(c, "impressions") as number));
    const position = Math.max(1, (c.src.position as number) + (c.rand() - 0.5) * 0.6);
    return { clicks, impressions, ctr: impressions ? Math.round((clicks / impressions) * 10000) / 100 : 0, position: Math.round(position * 100) / 100 };
  };
  await extend(
    "SeoDailyMetric",
    (await db.seoDailyMetric.findMany({ where: window })) as Row[],
    (r) => String(r.propertyId),
    seedEnd,
    until,
    (c) => ({ propertyId: c.prev.propertyId, ...seoRow(c) }),
    (data) => db.seoDailyMetric.createMany({ data: data as Prisma.SeoDailyMetricCreateManyInput[], skipDuplicates: true }),
  );
  await extend(
    "SeoDimensionMetric",
    (await db.seoDimensionMetric.findMany({ where: window })) as Row[],
    (r) => `${r.propertyId}|${r.dimension}|${r.key}`,
    seedEnd,
    until,
    (c) => ({ propertyId: c.prev.propertyId, dimension: c.prev.dimension, key: c.prev.key, ...seoRow(c) }),
    (data) => db.seoDimensionMetric.createMany({ data: data as Prisma.SeoDimensionMetricCreateManyInput[], skipDuplicates: true }),
  );
  await extend(
    "Ga4DailyMetric",
    (await db.ga4DailyMetric.findMany({ where: window })) as Row[],
    (r) => String(r.propertyId),
    seedEnd,
    until,
    (c) => {
      const sessions = Math.round(weekly(c, "sessions") as number);
      const share = (c.src.organicSessions as number) / Math.max(1, c.src.sessions as number);
      return {
        propertyId: c.prev.propertyId,
        sessions,
        organicSessions: Math.round(sessions * share),
        users: Math.min(sessions, Math.round(weekly(c, "users") as number)),
        engagedSessions: Math.min(sessions, Math.round(weekly(c, "engagedSessions") as number)),
        conversions: Math.round(weekly(c, "conversions") as number),
        avgEngagementSec: Math.round((weekly(c, "avgEngagementSec") as number) * 10) / 10,
      };
    },
    (data) => db.ga4DailyMetric.createMany({ data: data as Prisma.Ga4DailyMetricCreateManyInput[], skipDuplicates: true }),
  );
  await extend(
    "AdDailyInsight",
    (await db.adDailyInsight.findMany({ where: { ...window, adId: { not: null } } })) as Row[],
    (r) => String(r.adId),
    seedEnd,
    until,
    (c) => {
      const impressions = Math.round(weekly(c, "impressions") as number);
      const spend = Math.round(weekly(c, "spend") as number);
      return {
        adAccountId: c.prev.adAccountId,
        campaignId: c.prev.campaignId,
        adSetId: c.prev.adSetId,
        adId: c.prev.adId,
        spend,
        impressions,
        reach: Math.min(impressions, Math.round(weekly(c, "reach") as number)),
        clicks: Math.round(weekly(c, "clicks") as number),
        linkClicks: Math.round(weekly(c, "linkClicks") as number),
        results: Math.round(weekly(c, "results") as number),
        resultType: c.prev.resultType,
        purchaseValue: c.src.purchaseValue == null ? null : Math.round(weekly(c, "purchaseValue") as number),
      };
    },
    (data) => db.adDailyInsight.createMany({ data: data as Prisma.AdDailyInsightCreateManyInput[], skipDuplicates: true }),
  );
  await extend(
    "RankSnapshot",
    (await db.rankSnapshot.findMany({ where: window })) as Row[],
    (r) => String(r.keywordId),
    seedEnd,
    until,
    (c) => {
      const p = c.prev.position as number | null;
      const step = [-1, 0, 0, 0, 1][Math.floor(c.rand() * 5)]!;
      return {
        keywordId: c.prev.keywordId,
        position: p == null ? null : Math.min(100, Math.max(1, p + step)),
        url: c.prev.url,
        serpFeatures: c.prev.serpFeatures,
        topResults: (c.prev.topResults ?? undefined) as Prisma.InputJsonValue | undefined,
      };
    },
    (data) => db.rankSnapshot.createMany({ data: data as Prisma.RankSnapshotCreateManyInput[], skipDuplicates: true }),
  );
  await extend(
    "BacklinkSnapshot",
    (await db.backlinkSnapshot.findMany({ where: window })) as Row[],
    (r) => String(r.propertyId),
    seedEnd,
    until,
    (c) => ({
      propertyId: c.prev.propertyId,
      backlinks: int(trend(c, "backlinks")),
      referringDomains: int(trend(c, "referringDomains")),
      dofollow: int(trend(c, "dofollow")),
      nofollow: int(trend(c, "nofollow")),
      newBacklinks: int(weekly(c, "newBacklinks")),
      lostBacklinks: int(weekly(c, "lostBacklinks")),
      domainRank: c.prev.domainRank,
      toxicShare: c.prev.toxicShare,
    }),
    (data) => db.backlinkSnapshot.createMany({ data: data as Prisma.BacklinkSnapshotCreateManyInput[], skipDuplicates: true }),
  );
  await extend(
    "DomainSnapshot",
    (await db.domainSnapshot.findMany({ where: window })) as Row[],
    (r) => `${r.propertyId}|${r.domain}`,
    seedEnd,
    until,
    (c) => ({
      propertyId: c.prev.propertyId,
      domain: c.prev.domain,
      organicKeywords: int(trend(c, "organicKeywords")),
      organicTraffic: int(weekly(c, "organicTraffic")),
      organicCost: c.src.organicCost == null ? null : int(weekly(c, "organicCost")),
      top3: int(c.prev.top3),
      top10: int(c.prev.top10),
      top100: int(c.prev.top100),
      backlinks: c.prev.backlinks == null ? null : int(trend(c, "backlinks")),
      referringDomains: c.prev.referringDomains == null ? null : int(trend(c, "referringDomains")),
      domainRank: c.prev.domainRank,
    }),
    (data) => db.domainSnapshot.createMany({ data: data as Prisma.DomainSnapshotCreateManyInput[], skipDuplicates: true }),
  );
  console.log("✓ Selesai.");
}

main()
  .catch((e) => {
    console.error(e);
    process.exit(1);
  })
  .finally(async () => {
    await db.$disconnect();
  });
