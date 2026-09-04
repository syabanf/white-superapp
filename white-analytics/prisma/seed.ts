/**
 * Seed — deterministic, realistic demo data for WHITE Analytics.
 * Run: pnpm db:seed   (tsx --env-file=.env prisma/seed.ts)
 */
import { PrismaClient } from "../src/generated/prisma/client";
import { PrismaPg } from "@prisma/adapter-pg";
import { hash } from "bcryptjs";
import { AGES, AGE_WEIGHTS, DEVICES, GENDERS, GENDER_WEIGHTS, ISSUE_CATALOG, SEED_CLIENTS, type SeedClient } from "./seed-data";
import { seedSeoSuite } from "./seed-seo-suite";
import { seedPublishing } from "./seed-publishing";

const adapter = new PrismaPg({ connectionString: process.env.DATABASE_URL });
const db = new PrismaClient({ adapter });

// ── deterministic PRNG ────────────────────────────────────────
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
let rand = mulberry32(20260819);
const rnd = () => rand();
const randn = () => {
  // Box–Muller
  const u = 1 - rnd();
  const v = rnd();
  return Math.sqrt(-2 * Math.log(u)) * Math.cos(2 * Math.PI * v);
};
const int = (min: number, max: number) => Math.floor(rnd() * (max - min + 1)) + min;
const pick = <T>(arr: T[]): T => arr[Math.floor(rnd() * arr.length)]!;
const pickWeighted = <T>(arr: T[], weights: number[]): T => {
  const r = rnd();
  let acc = 0;
  for (let i = 0; i < arr.length; i++) {
    acc += weights[i]!;
    if (r <= acc) return arr[i]!;
  }
  return arr[arr.length - 1]!;
};
const clamp = (v: number, a: number, b: number) => Math.min(b, Math.max(a, v));
const jitter = (base: number, sd: number) => Math.max(0, base + randn() * sd);

// ── dates ─────────────────────────────────────────────────────
const DAYS = 120;
const now = new Date();
const today = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()));
const addDays = (d: Date, n: number) => new Date(d.getTime() + n * 86_400_000);
const END = addDays(today, -1); // yesterday = latest complete day
const START = addDays(END, -(DAYS - 1));
const dayAt = (i: number) => addDays(START, i);
const dow = (d: Date) => (d.getUTCDay() + 6) % 7; // Mon=0

/** weekly seasonality multiplier: weekends differ by industry */
function weekly(d: Date, kind: "b2c" | "b2b" | "travel"): number {
  const w = dow(d);
  if (kind === "b2b") return w >= 5 ? 0.62 : 1.05;
  if (kind === "travel") return w >= 5 ? 1.18 : 0.96;
  return w >= 5 ? 1.12 : 0.97;
}

async function chunked<T>(rows: T[], size: number, fn: (chunk: T[]) => Promise<unknown>) {
  for (let i = 0; i < rows.length; i += size) await fn(rows.slice(i, i + size));
}

async function main() {
  console.log("→ Menghapus data lama…");
  await db.$transaction([
    db.syncJob.deleteMany(),
    db.aiInsight.deleteMany(),
    db.report.deleteMany(),
    db.adDemographic.deleteMany(),
    db.adDailyInsight.deleteMany(),
    db.ad.deleteMany(),
    db.adSet.deleteMany(),
    db.adCampaign.deleteMany(),
    db.adAccount.deleteMany(),
    db.seoIssue.deleteMany(),
    db.seoCrawlPage.deleteMany(),
    db.seoCrawl.deleteMany(),
    db.seoAudit.deleteMany(),
    db.ga4DailyMetric.deleteMany(),
    db.seoDimensionMetric.deleteMany(),
    db.seoDailyMetric.deleteMany(),
    db.seoProperty.deleteMany(),
    db.socialPost.deleteMany(),
    db.socialSnapshot.deleteMany(),
    db.socialAccount.deleteMany(),
    db.connection.deleteMany(),
    db.clientMember.deleteMany(),
    db.client.deleteMany(),
    db.user.deleteMany(),
  ]);

  console.log("→ Users…");
  const admin = await db.user.create({
    data: { email: "admin@white.id", name: "Admin WHITE", passwordHash: await hash("admin12345", 10), role: "ADMIN" },
  });
  const member = await db.user.create({
    data: { email: "tim@white.id", name: "Tim Analis", passwordHash: await hash("member12345", 10), role: "MEMBER" },
  });
  await db.user.create({
    data: { email: "aloha.mrwhite@gmail.com", name: "Mr. White", passwordHash: await hash("waliwisbodas-ganti", 10), role: "ADMIN" },
  });

  for (const [ci, c] of SEED_CLIENTS.entries()) {
    rand = mulberry32(1000 + ci); // stable per client
    console.log(`→ Klien: ${c.name}`);
    const client = await db.client.create({
      data: {
        name: c.name,
        slug: c.slug,
        description: c.description,
        industry: c.industry,
        websiteUrl: c.websiteUrl,
        currency: "IDR",
        timezone: "Asia/Jakarta",
        shareEnabled: true,
        sharePinHash: ci === 0 ? await hash("123456", 10) : null,
        shareModules: ["SOCIAL", "SEO", "ADS"],
        createdById: admin.id,
      },
    });
    if (ci < 2) await db.clientMember.create({ data: { userId: member.id, clientId: client.id, role: ci === 0 ? "MANAGER" : "VIEWER" } });

    await seedSocial(client.id, c, ci);
    await seedSeo(client.id, c, ci);
    await seedAds(client.id, c, ci);

    await db.report.createMany({
      data: [
        {
          clientId: client.id,
          title: `Laporan Bulanan — ${c.name}`,
          dateFrom: addDays(END, -30),
          dateTo: addDays(END, -1),
          compare: true,
          modules: ["SOCIAL", "SEO", "ADS"],
          language: "id",
          aiSummary: `**Ringkasan eksekutif**\n\nPerforma ${c.name} pada periode ini menunjukkan tren positif di kanal organik dan sosial. Pertumbuhan followers stabil, klik organik meningkat, dan biaya per hasil iklan tetap terkendali.\n\n**Rekomendasi**\n1. Perkuat konten video pendek yang terbukti mendapat engagement tertinggi.\n2. Optimasi title & meta pada kata kunci dengan CTR rendah di halaman 1.\n3. Alokasikan ulang anggaran dari kampanye awareness ke kampanye konversi.`,
          createdById: admin.id,
          createdAt: addDays(END, -1),
        },
        {
          clientId: client.id,
          title: `Laporan Mingguan — ${c.name}`,
          dateFrom: addDays(END, -7),
          dateTo: END,
          compare: true,
          modules: ["SOCIAL", "SEO"],
          language: "id",
          createdById: member.id,
          createdAt: END,
        },
      ],
    });

    await db.syncJob.createMany({
      data: [
        { clientId: client.id, kind: "SOCIAL_SNAPSHOT", status: "SUCCESS", startedAt: addDays(today, 0), finishedAt: addDays(today, 0), message: "3 akun, 12 postingan diperbarui" },
        { clientId: client.id, kind: "SEO_GSC", status: "SUCCESS", startedAt: addDays(today, 0), finishedAt: addDays(today, 0), message: "2 hari data baru" },
        { clientId: client.id, kind: "ADS_INSIGHTS", status: ci === 1 ? "FAILED" : "SUCCESS", startedAt: addDays(today, 0), finishedAt: addDays(today, 0), message: ci === 1 ? undefined : "16 iklan diperbarui", error: ci === 1 ? "TOKEN_EXPIRED: Sesi Meta kedaluwarsa, hubungkan ulang." : undefined },
      ],
    });
  }

  // Platform modules (own files, idempotent, need the clients above to exist).
  await seedSeoSuite(db);
  await seedPublishing(db);
  console.log("✓ Seed selesai.");
}

// ─────────────────────────────────────────────────────────────
// Social
// ─────────────────────────────────────────────────────────────
async function seedSocial(clientId: string, c: SeedClient, ci: number) {
  const platforms = [
    { platform: "INSTAGRAM" as const, username: c.own.ig, name: c.own.igName, base: c.baseFollowers.ig, gain: c.dailyGain.ig },
    { platform: "FACEBOOK" as const, username: c.own.fb, name: c.own.fbName, base: c.baseFollowers.fb, gain: c.dailyGain.fb },
    { platform: "TIKTOK" as const, username: c.own.tiktok, name: c.own.tiktokName, base: c.baseFollowers.tiktok, gain: c.dailyGain.tiktok },
  ];

  for (const p of platforms) {
    const acc = await db.socialAccount.create({
      data: {
        clientId,
        platform: p.platform,
        externalId: `${p.platform.toLowerCase()}_${17800000000 + ci * 1000 + int(1, 999)}`,
        username: p.username,
        displayName: p.name,
        avatarUrl: `https://api.dicebear.com/9.x/initials/svg?seed=${encodeURIComponent(p.name)}&backgroundColor=1c1c1e&textColor=ffffff`,
        biography: c.description.split(".")[0],
        isCompetitor: false,
      },
    });
    await snapshotsFor(acc.id, p.base - p.gain * DAYS, p.gain, p.platform === "TIKTOK" ? 1.6 : 1, ci);
    await postsFor(acc.id, p.platform, p.base, c, false);
  }

  for (const [k, comp] of c.competitors.entries()) {
    const acc = await db.socialAccount.create({
      data: {
        clientId,
        platform: "INSTAGRAM",
        externalId: `ig_comp_${ci}_${k}_${int(1000, 9999)}`,
        username: comp.username,
        displayName: comp.name,
        biography: comp.bio,
        avatarUrl: `https://api.dicebear.com/9.x/initials/svg?seed=${encodeURIComponent(comp.name)}&backgroundColor=8e8e93&textColor=ffffff`,
        isCompetitor: true,
      },
    });
    const gain = Math.round(comp.followers * (0.0004 + rnd() * 0.0008));
    await snapshotsFor(acc.id, comp.followers - gain * DAYS, gain, 1, ci, true);
    await postsFor(acc.id, "INSTAGRAM", comp.followers, c, true);
  }
}

async function snapshotsFor(accountId: string, startFollowers: number, gain: number, volatility: number, ci: number, competitor = false) {
  const rows = [];
  let f = startFollowers;
  const spikeDay = int(30, 100);
  for (let i = 0; i < DAYS; i++) {
    const d = dayAt(i);
    const growth = gain * (0.85 + 0.3 * rnd()) * (i === spikeDay ? 6 : 1) + randn() * gain * 0.35 * volatility;
    f = Math.max(0, Math.round(f + growth));
    const reachBase = f * (0.09 + 0.05 * rnd()) * weekly(d, ci === 1 ? "b2b" : ci === 2 ? "travel" : "b2c") * (i === spikeDay ? 2.5 : 1);
    rows.push({
      socialAccountId: accountId,
      date: d,
      followers: f,
      following: competitor ? null : int(180, 420),
      mediaCount: Math.round(300 + i * 0.35 + (competitor ? 200 : 0)),
      reach: competitor ? null : Math.round(reachBase),
      impressions: competitor ? null : Math.round(reachBase * (1.5 + 0.4 * rnd())),
      profileViews: competitor ? null : Math.round(reachBase * (0.06 + 0.03 * rnd())),
      websiteClicks: competitor ? null : Math.round(reachBase * (0.006 + 0.004 * rnd())),
    });
  }
  await chunked(rows, 500, (chunk) => db.socialSnapshot.createMany({ data: chunk }));
}

async function postsFor(accountId: string, platform: "INSTAGRAM" | "FACEBOOK" | "TIKTOK", followers: number, c: SeedClient, competitor: boolean) {
  const perWeek = platform === "TIKTOK" ? 4 : platform === "FACEBOOK" ? 2 : competitor ? 3 : 3.5;
  const total = Math.round((DAYS / 7) * perWeek);
  const rows = [];
  for (let n = 0; n < total; n++) {
    const dayIdx = int(0, DAYS - 1);
    const d = dayAt(dayIdx);
    // publish hours WIB 8–21 skewed to 11 & 19
    const hourWib = pick([8, 9, 10, 11, 11, 12, 13, 15, 17, 18, 19, 19, 20, 21]);
    const publishedAt = new Date(d.getTime() + (hourWib - 7) * 3600_000 + int(0, 59) * 60_000);
    const kind = platform === "TIKTOK" ? "VIDEO" : pickWeighted(["VIDEO", "CAROUSEL_ALBUM", "IMAGE"], [0.42, 0.3, 0.28]);
    const productType = platform === "TIKTOK" ? "REELS" : kind === "VIDEO" ? "REELS" : "FEED";
    // engagement rate by followers: reels higher
    const erBase = (kind === "VIDEO" ? 0.032 : kind === "CAROUSEL_ALBUM" ? 0.026 : 0.019) * (platform === "FACEBOOK" ? 0.35 : platform === "TIKTOK" ? 1.3 : 1);
    const viral = rnd() < 0.05 ? int(3, 8) : 1;
    const engagements = Math.round(followers * erBase * (0.5 + rnd()) * viral);
    const likes = Math.round(engagements * (0.78 + rnd() * 0.1));
    const comments = Math.round(engagements * (0.05 + rnd() * 0.05));
    const shares = Math.round(engagements * (0.04 + rnd() * 0.05));
    const saves = Math.max(0, engagements - likes - comments - shares);
    const reach = Math.round(followers * (kind === "VIDEO" ? 0.4 : 0.22) * (0.6 + rnd() * 0.9) * viral);
    const views = kind === "VIDEO" ? Math.round(reach * (1.4 + rnd())) : 0;
    const caption = `${pick(c.captions)} ${c.hashtags.slice(0, int(3, 6)).join(" ")}`;
    const seedId = `${accountId.slice(-6)}${n}`;
    rows.push({
      socialAccountId: accountId,
      externalId: `${platform.toLowerCase()}_media_${seedId}`,
      caption,
      mediaType: kind,
      productType,
      permalink: platform === "INSTAGRAM" ? `https://www.instagram.com/p/${seedId}/` : platform === "TIKTOK" ? `https://www.tiktok.com/@${c.own.tiktok}/video/${seedId}` : `https://www.facebook.com/${c.own.fb}/posts/${seedId}`,
      mediaUrl: `https://picsum.photos/seed/${seedId}/800/800`,
      thumbnailUrl: `https://picsum.photos/seed/${seedId}/400/400`,
      publishedAt,
      likes,
      comments,
      shares: competitor ? 0 : shares,
      saves: competitor ? 0 : saves,
      views: competitor ? 0 : views,
      reach: competitor ? 0 : reach,
      impressions: competitor ? 0 : Math.round(reach * (1.3 + rnd() * 0.5)),
    });
  }
  await db.socialPost.createMany({ data: rows, skipDuplicates: true });
}

// ─────────────────────────────────────────────────────────────
// SEO
// ─────────────────────────────────────────────────────────────
async function seedSeo(clientId: string, c: SeedClient, ci: number) {
  const prop = await db.seoProperty.create({
    data: { clientId, siteUrl: c.siteUrl, ga4PropertyId: c.ga4PropertyId },
  });
  const kind = ci === 1 ? "b2b" : ci === 2 ? "travel" : "b2c";

  // per-query static profile
  const allQueries = [...c.brandQueries.map((q) => ({ q, brand: true })), ...c.queries.map((q) => ({ q, brand: false }))];
  const profiles = allQueries.map(({ q, brand }, i) => {
    const pos = brand ? 1 + rnd() * 2 : i % 5 === 0 ? 3 + rnd() * 4 : i % 3 === 0 ? 8 + rnd() * 8 : 12 + rnd() * 25;
    const impr = brand ? 40 + rnd() * 120 : (i % 4 === 0 ? 300 : 90) * (0.4 + rnd() * 1.4);
    // ctr from expected curve with noise, some deliberately low-CTR
    const expected = pos <= 1.5 ? 28 : pos <= 2.5 ? 15 : pos <= 3.5 ? 11 : pos <= 5 ? 7.5 : pos <= 8 ? 4 : pos <= 10 ? 2.5 : pos <= 20 ? 1.4 : 0.8;
    const lowCtr = !brand && i % 7 === 0;
    const ctr = expected * (lowCtr ? 0.35 : 0.75 + rnd() * 0.5);
    const trend = (rnd() - 0.4) * 0.006 + (i % 9 === 0 ? -0.006 : 0) + (i % 11 === 0 ? 0.008 : 0);
    return { q, brand, pos, impr, ctr, trend };
  });
  const pageProfiles = c.pages.map((p, i) => ({ p, share: i === 0 ? 0.28 : i < 4 ? 0.12 : 0.04 + rnd() * 0.03 }));
  const pageTotal = pageProfiles.reduce((a, b) => a + b.share, 0);

  const daily: Array<{ propertyId: string; date: Date; clicks: number; impressions: number; ctr: number; position: number }> = [];
  const dims: Array<{ propertyId: string; date: Date; dimension: "QUERY" | "PAGE" | "COUNTRY" | "DEVICE"; key: string; clicks: number; impressions: number; ctr: number; position: number }> = [];
  const ga4: Array<{ propertyId: string; date: Date; sessions: number; organicSessions: number; users: number; engagedSessions: number; conversions: number; avgEngagementSec: number }> = [];

  for (let i = 0; i < DAYS; i++) {
    const d = dayAt(i);
    const season = weekly(d, kind);
    const growth = 1 + c.seoBase.trend * i;
    let dayClicks = 0;
    let dayImpr = 0;
    let posNum = 0;
    for (const pr of profiles) {
      const impr = Math.round(jitter(pr.impr * season * growth * (1 + pr.trend * i), pr.impr * 0.25));
      const pos = clamp(pr.pos - pr.trend * i * 40 + randn() * 0.6, 1, 60);
      const clicks = Math.round(impr * (pr.ctr / 100) * (0.7 + rnd() * 0.6));
      dims.push({ propertyId: prop.id, date: d, dimension: "QUERY", key: pr.q, clicks, impressions: impr, ctr: impr ? (clicks / impr) * 100 : 0, position: pos });
      dayClicks += clicks;
      dayImpr += impr;
      posNum += pos * impr;
    }
    const position = dayImpr ? posNum / dayImpr : c.seoBase.position;
    daily.push({ propertyId: prop.id, date: d, clicks: dayClicks, impressions: dayImpr, ctr: dayImpr ? (dayClicks / dayImpr) * 100 : 0, position });

    for (const pp of pageProfiles) {
      const share = pp.share / pageTotal;
      const impr = Math.round(dayImpr * share * (0.85 + rnd() * 0.3));
      const clicks = Math.round(dayClicks * share * (0.85 + rnd() * 0.3));
      dims.push({ propertyId: prop.id, date: d, dimension: "PAGE", key: `${c.websiteUrl}${pp.p}`, clicks, impressions: impr, ctr: impr ? (clicks / impr) * 100 : 0, position: clamp(position + randn() * 3, 1, 60) });
    }
    for (const co of c.countries) {
      const impr = Math.round(dayImpr * co.share * (0.9 + rnd() * 0.2));
      const clicks = Math.round(dayClicks * co.share * (0.9 + rnd() * 0.2));
      dims.push({ propertyId: prop.id, date: d, dimension: "COUNTRY", key: co.key, clicks, impressions: impr, ctr: impr ? (clicks / impr) * 100 : 0, position: clamp(position + randn() * 2, 1, 60) });
    }
    for (const dv of DEVICES) {
      const impr = Math.round(dayImpr * dv.share * (0.9 + rnd() * 0.2));
      const clicks = Math.round(dayClicks * dv.share * (0.9 + rnd() * 0.2));
      dims.push({ propertyId: prop.id, date: d, dimension: "DEVICE", key: dv.key, clicks, impressions: impr, ctr: impr ? (clicks / impr) * 100 : 0, position: clamp(position + (dv.key === "MOBILE" ? -0.4 : 0.8) + randn(), 1, 60) });
    }

    const sessions = Math.round(jitter(c.ga4Base.sessions * season * growth, c.ga4Base.sessions * 0.12));
    const organic = Math.round(sessions * c.ga4Base.organicShare * (0.9 + rnd() * 0.2));
    ga4.push({
      propertyId: prop.id,
      date: d,
      sessions,
      organicSessions: organic,
      users: Math.round(sessions * (0.78 + rnd() * 0.1)),
      engagedSessions: Math.round(sessions * (0.55 + rnd() * 0.15)),
      conversions: Math.round(sessions * c.ga4Base.conv * (0.7 + rnd() * 0.6)),
      avgEngagementSec: 45 + rnd() * 60,
    });
  }
  await chunked(daily, 500, (chunk) => db.seoDailyMetric.createMany({ data: chunk }));
  await chunked(dims, 1000, (chunk) => db.seoDimensionMetric.createMany({ data: chunk }));
  await chunked(ga4, 500, (chunk) => db.ga4DailyMetric.createMany({ data: chunk }));

  // audits: 4 runs, both strategies
  const auditRows = [];
  for (const [k, s] of c.auditScores.entries()) {
    const runAt = addDays(END, -(c.auditScores.length - 1 - k) * 30 + (k === c.auditScores.length - 1 ? 0 : -int(0, 3)));
    for (const strategy of ["MOBILE", "DESKTOP"] as const) {
      const bump = strategy === "DESKTOP" ? 14 : 0;
      auditRows.push({
        propertyId: prop.id,
        url: c.auditUrl,
        strategy,
        runAt: new Date(runAt.getTime() + 9 * 3600_000),
        performance: clamp(s.performance + bump + int(-2, 2), 0, 100),
        seo: clamp(s.seo + int(-1, 1), 0, 100),
        accessibility: clamp(s.accessibility + int(-1, 2), 0, 100),
        bestPractices: clamp(s.bestPractices + int(-1, 1), 0, 100),
        lcpMs: Math.max(600, (strategy === "MOBILE" ? 4600 : 2400) - k * 350 + int(-150, 150)),
        inpMs: Math.max(30, (strategy === "MOBILE" ? 310 : 140) - k * 30 + int(-20, 20)),
        cls: Math.max(0.01, (strategy === "MOBILE" ? 0.21 : 0.08) - k * 0.03 + (rnd() - 0.5) * 0.02),
        fcpMs: Math.max(400, (strategy === "MOBILE" ? 2600 : 1300) - k * 200 + int(-100, 100)),
        ttfbMs: Math.max(120, 900 - k * 90 + int(-60, 60)),
        tbtMs: Math.max(10, (strategy === "MOBILE" ? 620 : 180) - k * 45 + int(-30, 30)),
        speedIndexMs: Math.max(700, (strategy === "MOBILE" ? 5200 : 2600) - k * 400 + int(-150, 150)),
      });
    }
  }
  await db.seoAudit.createMany({ data: auditRows });

  // crawl
  const crawl = await db.seoCrawl.create({
    data: {
      propertyId: prop.id,
      startedAt: addDays(END, 0),
      finishedAt: new Date(addDays(END, 0).getTime() + 4 * 60_000),
      status: "SUCCESS",
      pagesCrawled: c.pages.length + 12,
      maxPages: 50,
    },
  });
  const pageRows = [];
  const issueRows = [];
  const urls = [...c.pages, ...Array.from({ length: 12 }, (_, i) => `/blog/artikel-${i + 1}`)];
  for (const [i, path] of urls.entries()) {
    const url = `${c.websiteUrl}${path}`;
    const wc = path === "/" ? 420 : path.startsWith("/blog") ? 700 + int(0, 900) : 180 + int(0, 400);
    const missingAlt = int(0, 4);
    const titleLen = int(24, 72);
    pageRows.push({
      crawlId: crawl.id,
      url,
      statusCode: i === urls.length - 1 ? 404 : 200,
      title: `${c.name} — ${path === "/" ? "Beranda" : path.split("/").pop()!.replace(/-/g, " ")}`,
      metaDescription: i % 6 === 0 ? null : c.description.slice(0, 150),
      h1Count: i % 9 === 0 ? 0 : i % 13 === 0 ? 2 : 1,
      wordCount: wc,
      canonical: i % 10 === 0 ? null : url,
      indexable: i % 17 !== 0,
      imagesTotal: int(3, 14),
      imagesMissingAlt: missingAlt,
      internalLinks: int(8, 40),
      externalLinks: int(0, 6),
      loadMs: int(300, 2400),
      hasJsonLd: i % 4 !== 0,
      hasHreflang: ci === 2 && i % 2 === 0,
    });
    if (i % 6 === 0) issueRows.push({ crawlId: crawl.id, url, ...ISSUE_CATALOG.find((x) => x.code === "MISSING_META_DESCRIPTION")! });
    if (i % 9 === 0) issueRows.push({ crawlId: crawl.id, url, ...ISSUE_CATALOG.find((x) => x.code === "MISSING_H1")! });
    if (i % 13 === 0) issueRows.push({ crawlId: crawl.id, url, ...ISSUE_CATALOG.find((x) => x.code === "MULTIPLE_H1")! });
    if (missingAlt > 0) issueRows.push({ crawlId: crawl.id, url, ...ISSUE_CATALOG.find((x) => x.code === "IMG_MISSING_ALT")!, message: `${missingAlt} gambar tanpa atribut alt.` });
    if (wc < 300) issueRows.push({ crawlId: crawl.id, url, ...ISSUE_CATALOG.find((x) => x.code === "THIN_CONTENT")! });
    if (titleLen > 60) issueRows.push({ crawlId: crawl.id, url, ...ISSUE_CATALOG.find((x) => x.code === "TITLE_TOO_LONG")! });
    if (i % 10 === 0) issueRows.push({ crawlId: crawl.id, url, ...ISSUE_CATALOG.find((x) => x.code === "MISSING_CANONICAL")! });
    if (i % 4 === 0) issueRows.push({ crawlId: crawl.id, url, ...ISSUE_CATALOG.find((x) => x.code === "NO_STRUCTURED_DATA")! });
    if (i === urls.length - 1) issueRows.push({ crawlId: crawl.id, url, ...ISSUE_CATALOG.find((x) => x.code === "BROKEN_INTERNAL_LINK")! });
    if (i % 17 === 0 && i > 0) issueRows.push({ crawlId: crawl.id, url, ...ISSUE_CATALOG.find((x) => x.code === "NOINDEX")! });
  }
  await db.seoCrawlPage.createMany({ data: pageRows });
  await db.seoIssue.createMany({ data: issueRows.map(({ code, severity, message, url, crawlId }) => ({ code, severity, message, url, crawlId })) });
}

// ─────────────────────────────────────────────────────────────
// Ads
// ─────────────────────────────────────────────────────────────
async function seedAds(clientId: string, c: SeedClient, ci: number) {
  const account = await db.adAccount.create({
    data: { clientId, externalId: c.ads.accountId, name: c.ads.accountName, currency: "IDR", lastSyncedAt: addDays(today, 0) },
  });
  const insightRows: Array<{
    adAccountId: string;
    date: Date;
    campaignId: string;
    adSetId: string;
    adId: string;
    spend: number;
    impressions: number;
    reach: number;
    clicks: number;
    linkClicks: number;
    results: number;
    resultType: string;
    purchaseValue: number | null;
  }> = [];
  const demoRows = [];

  for (const [k, camp] of c.ads.campaigns.entries()) {
    const campaign = await db.adCampaign.create({
      data: {
        adAccountId: account.id,
        externalId: `${120000000000 + ci * 100000 + k * 1000}`,
        name: camp.name,
        objective: camp.objective,
        status: camp.status,
        dailyBudget: camp.dailyBudget,
        startTime: dayAt(camp.startDay),
      },
    });
    const adRefs: { id: string; adSetId: string; weight: number }[] = [];
    for (const [j, set] of camp.adSets.entries()) {
      const adSet = await db.adSet.create({
        data: {
          campaignId: campaign.id,
          externalId: `${campaign.externalId}${j + 1}`,
          name: set.name,
          status: camp.status,
          dailyBudget: camp.dailyBudget / camp.adSets.length,
        },
      });
      for (const [m, adName] of set.ads.entries()) {
        const seedId = `${ci}${k}${j}${m}`;
        const ad = await db.ad.create({
          data: {
            adSetId: adSet.id,
            externalId: `${adSet.externalId}${m + 1}`,
            name: adName,
            status: camp.status === "PAUSED" ? "PAUSED" : m === 1 && k === 0 && j === 1 ? "PAUSED" : "ACTIVE",
            thumbnailUrl: `https://picsum.photos/seed/ad${seedId}/400/400`,
          },
        });
        adRefs.push({ id: ad.id, adSetId: adSet.id, weight: 0.6 + rnd() * 0.8 });
      }
    }
    const wsum = adRefs.reduce((a, b) => a + b.weight, 0);
    const endDay = camp.endDay ?? DAYS - 1;

    // demographic profile for this campaign (whole active window)
    const demoWeights = AGES.map((_, i) => AGE_WEIGHTS[i]! * (0.7 + rnd() * 0.6));
    const dwSum = demoWeights.reduce((a, b) => a + b, 0);
    let campSpend = 0;
    let campImpr = 0;
    let campReach = 0;
    let campResults = 0;

    for (let i = camp.startDay; i <= endDay; i++) {
      const d = dayAt(i);
      // learning phase first 5 days: lower delivery
      const ramp = i - camp.startDay < 5 ? 0.6 : 1;
      const daySpend = jitter(camp.dailyBudget * ramp * (0.86 + rnd() * 0.16), camp.dailyBudget * 0.05);
      const cpm = 22_000 + rnd() * 16_000 + (ci === 2 ? 20_000 : 0);
      for (const ad of adRefs) {
        const share = ad.weight / wsum;
        const spend = Math.round(daySpend * share);
        const impressions = Math.round((spend / cpm) * 1000);
        const reach = Math.round(impressions / (1.4 + rnd() * 1.4));
        const linkClicks = Math.round(impressions * (camp.ctr / 100) * (0.7 + rnd() * 0.6));
        const clicks = Math.round(linkClicks * (1.25 + rnd() * 0.3));
        const results =
          camp.resultType === "reach" ? reach : Math.round(spend / (camp.cpr * (0.75 + rnd() * 0.6) * (ad.weight > 1 ? 0.85 : 1.15)));
        const purchaseValue = camp.resultType === "purchase" ? Math.round(results * camp.cpr * (2.8 + rnd() * 1.6)) : null;
        insightRows.push({
          adAccountId: account.id,
          date: d,
          campaignId: campaign.id,
          adSetId: ad.adSetId,
          adId: ad.id,
          spend,
          impressions,
          reach,
          clicks,
          linkClicks,
          results,
          resultType: camp.resultType,
          purchaseValue,
        });
        campSpend += spend;
        campImpr += impressions;
        campReach += reach;
        campResults += results;
      }
    }
    for (const [ai, age] of AGES.entries()) {
      for (const [gi, gender] of GENDERS.entries()) {
        const w = (demoWeights[ai]! / dwSum) * GENDER_WEIGHTS[gi]!;
        demoRows.push({
          adAccountId: account.id,
          dateFrom: dayAt(camp.startDay),
          dateTo: dayAt(endDay),
          campaignId: campaign.id,
          age,
          gender,
          spend: Math.round(campSpend * w),
          impressions: Math.round(campImpr * w),
          reach: Math.round(campReach * w),
          results: Math.round(campResults * w * (gender === "female" ? 1.1 : 0.9)),
        });
      }
    }
  }
  await chunked(insightRows, 1000, (chunk) => db.adDailyInsight.createMany({ data: chunk }));
  await db.adDemographic.createMany({ data: demoRows });
}

main()
  .catch((e) => {
    console.error(e);
    process.exit(1);
  })
  .finally(async () => {
    await db.$disconnect();
  });
