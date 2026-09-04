/**
 * Seed — SEO suite demo data (tracked keywords + 120 days of ranks, competitor domains + domain
 * snapshots, backlink snapshots + backlink rows, cached keyword research) for every seed client's
 * SeoProperty. Deterministic; idempotent (deletes its own tables first).
 *
 * Ranks, backlinks and domain metrics are derived from the SAME deterministic functions the mock
 * DataForSEO adapter uses, so the seeded history and the data produced by "Perbarui sekarang" /
 * the daily job in demo mode form one continuous series (no spurious jumps or alerts).
 *
 * Standalone: pnpm tsx --env-file=.env prisma/seed-seo-suite.ts
 * From seed.ts: import { seedSeoSuite } from "./seed-seo-suite"; await seedSeoSuite(db);
 */
import { PrismaClient } from "../src/generated/prisma/client";
import { PrismaPg } from "@prisma/adapter-pg";
import { SEED_CLIENTS, type SeedClient } from "./seed-data";
import { mockBacklinkList, mockPositionOn, mockSeoData, mockTop10, rankKey } from "../src/lib/providers/dataforseo/mock";

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
let rand = mulberry32(20260902);
const rnd = () => rand();
const int = (min: number, max: number) => Math.floor(rnd() * (max - min + 1)) + min;
const pick = <T>(arr: readonly T[]): T => arr[Math.floor(rnd() * arr.length)]!;
const clamp = (v: number, a: number, b: number) => Math.min(b, Math.max(a, v));
const randn = () => {
  const u = 1 - rnd();
  const v = rnd();
  return Math.sqrt(-2 * Math.log(u)) * Math.cos(2 * Math.PI * v);
};

// ── dates: 120 days ending TODAY (ranks are taken during the day) ─
const DAYS = 120;
const addDays = (d: Date, n: number) => new Date(d.getTime() + n * 86_400_000);
const now = new Date();
const TODAY = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()));
const START = addDays(TODAY, -(DAYS - 1));
const dayAt = (i: number) => addDays(START, i);

// ── static content ────────────────────────────────────────────
const COMPETITORS: Record<string, string[]> = {
  "kopi-nusantara": ["kopikenangan.com", "janjijiwa.com", "fore.coffee"],
  "adiharjo-property": ["rumah123.com", "summarecon.com", "sinarmasland.com"],
  "bali-villa-escapes": ["balivillas.com", "elitehavens.com", "villa-bali.com"],
};
const GENERIC = ["tokopedia.com", "shopee.co.id", "kompas.com"];
const SERP = ["featured_snippet", "people_also_ask", "images", "video", "local_pack", "shopping", "sitelinks", "reviews"];
const MODIFIERS = ["harga", "terdekat", "terbaik", "murah", "review", "2026", "promo", "online", "rekomendasi"];
const CITIES = ["jakarta", "bandung", "surabaya", "yogyakarta", "bali", "medan"];

function host(siteUrl: string): string {
  return siteUrl.replace(/^sc-domain:/, "").replace(/^https?:\/\//, "").replace(/^www\./, "").replace(/\/.*$/, "");
}

function intentFor(k: string): "INFORMATIONAL" | "NAVIGATIONAL" | "COMMERCIAL" | "TRANSACTIONAL" {
  if (/\b(cara|manfaat|resep|beda|tips|apa)\b/.test(k)) return "INFORMATIONAL";
  if (/\b(harga|beli|promo|murah|grosir|supplier|dijual|sewa|booking)\b/.test(k)) return "TRANSACTIONAL";
  if (/\b(terbaik|review|rekomendasi|terdekat|instagramable|nyaman)\b/.test(k)) return "COMMERCIAL";
  return "INFORMATIONAL";
}

function volumeFor(k: string, brand: boolean): number {
  const base = brand ? 250 : 900;
  const raw = Math.exp(Math.log(base) + randn() * 0.9);
  return Math.max(20, raw < 100 ? Math.round(raw / 10) * 10 : Math.round(raw / 50) * 50);
}

type Idea = { keyword: string; volume: number; difficulty: number; cpc: number; competition: number; intent: string; trend: number[]; serpFeatures: string[] };

function ideaFor(keyword: string, brand = false): Idea {
  const volume = volumeFor(keyword, brand);
  const difficulty = Math.round(clamp(12 * Math.log10(volume) + (rnd() - 0.5) * 30 - (brand ? 15 : 0), 2, 98));
  const intent = brand ? "NAVIGATIONAL" : intentFor(keyword);
  const cpcBase = intent === "TRANSACTIONAL" ? 6000 : intent === "COMMERCIAL" ? 4000 : 1500;
  const cpc = Math.round(clamp(cpcBase * (0.5 + rnd() * 1.4) + difficulty * 40, 500, 15_000) / 50) * 50;
  const slope = (rnd() - 0.45) * 0.04;
  const trend = Array.from({ length: 12 }, (_, i) => Math.max(10, Math.round(volume * (1 + slope * (i - 11) + (rnd() - 0.5) * 0.15))));
  trend[11] = volume;
  return { keyword, volume, difficulty, cpc, competition: Math.round(rnd() * 100) / 100, intent, trend, serpFeatures: SERP.filter(() => rnd() < 0.25) };
}

async function chunked<T>(rows: T[], size: number, fn: (chunk: T[]) => Promise<unknown>) {
  for (let i = 0; i < rows.length; i += size) await fn(rows.slice(i, i + size));
}

// ── per property ──────────────────────────────────────────────
async function seedProperty(db: PrismaClient, propertyId: string, clientId: string, c: SeedClient, ci: number) {
  rand = mulberry32(5000 + ci);
  const own = host(c.siteUrl);
  const competitors = COMPETITORS[c.slug] ?? GENERIC.slice(0, 3);
  const counts = { keywords: 0, ranks: 0, domains: 0, backlinks: 0, research: 0 };

  // competitor domains
  await db.competitorDomain.createMany({ data: competitors.map((domain) => ({ propertyId, domain })) });

  // tracked keywords: 30–40 (brand + top queries); positions come from the mock's deterministic walk
  const nKw = int(30, 40);
  const pool = [...c.brandQueries.map((q) => ({ q, brand: true })), ...c.queries.map((q) => ({ q, brand: false }))].slice(0, nKw);
  for (const { q, brand } of pool) {
    const idea = ideaFor(q, brand);
    const tags = brand ? ["brand"] : [pick(["produk", "produk", "blog", "lokasi", "kategori"])];
    if (!brand && rnd() < 0.3) tags.push("prioritas");
    const device = rnd() < 0.8 ? ("MOBILE" as const) : ("DESKTOP" as const);
    const slug = q.replace(/[^a-z0-9]+/g, "-");
    const kw = await db.trackedKeyword.create({
      data: {
        propertyId,
        keyword: q,
        locationCode: 2360,
        languageCode: "id",
        device,
        tags,
        targetUrl: `https://${own}/${brand ? "" : slug}`,
        volume: idea.volume,
        difficulty: idea.difficulty,
        cpc: idea.cpc,
        intent: idea.intent as "INFORMATIONAL",
        createdAt: START,
      },
    });
    counts.keywords++;
    const target = rankKey(own, device);
    const rows = Array.from({ length: DAYS }, (_, i) => {
      const d = dayAt(i);
      const position = mockPositionOn(q, target, d);
      return {
        keywordId: kw.id,
        date: d,
        position,
        url: position == null ? null : `https://${own}/${slug}`,
        serpFeatures: idea.serpFeatures,
        topResults: mockTop10(q, own, position, d),
      };
    });
    await db.rankSnapshot.createMany({ data: rows });
    counts.ranks += rows.length;
  }

  // domain snapshots: own + competitors, 120 days back-cast from the mock's overview for today
  const domRows: Array<{ propertyId: string; domain: string; date: Date; organicKeywords: number; organicTraffic: number; organicCost: number; top3: number; top10: number; top100: number; backlinks: number; referringDomains: number; domainRank: number }> = [];
  const allDomains = [own, ...competitors];
  for (const domain of allDomains) {
    const o = await mockSeoData.domainOverview(domain, 2360, "id");
    const growth = (rnd() - 0.3) * 0.003; // per day; today = 1
    for (let i = 0; i < DAYS; i++) {
      const back = DAYS - 1 - i;
      const g = Math.max(0.5, 1 - growth * back);
      const week = 1 + 0.03 * Math.sin((i / 7) * Math.PI * 2);
      const noise = i === DAYS - 1 ? 1 : 1 + randn() * 0.012;
      domRows.push({
        propertyId,
        domain,
        date: dayAt(i),
        organicKeywords: Math.round(o.organicKeywords * g * noise),
        organicTraffic: Math.round(o.organicTraffic * g * week * noise),
        organicCost: Math.round(o.organicCost * g * noise),
        top3: Math.round(o.top3 * g),
        top10: Math.round(o.top10 * g),
        top100: Math.round(o.top100 * g),
        backlinks: Math.round(o.backlinks * (1 - 0.0012 * back)),
        referringDomains: Math.round(o.referringDomains * (1 - 0.001 * back)),
        domainRank: o.domainRank - (back > DAYS / 2 ? 1 : 0),
      });
    }
  }
  await chunked(domRows, 500, (chunk) => db.domainSnapshot.createMany({ data: chunk }));
  counts.domains = domRows.length;

  // backlinks: the mock's list for this domain (same rows a refresh would upsert)
  const list = mockBacklinkList(own);
  const blRows = list.map((b) => ({
    propertyId,
    sourceUrl: b.sourceUrl,
    sourceDomain: b.sourceDomain,
    targetUrl: b.targetUrl,
    anchor: b.anchor,
    dofollow: b.dofollow,
    domainRank: b.domainRank,
    spamScore: b.spamScore,
    firstSeen: new Date(b.firstSeen),
    lastSeen: new Date(b.lastSeen),
    isLost: b.isLost,
  }));
  await db.backlink.createMany({ data: blRows, skipDuplicates: true });
  counts.backlinks = blRows.length;

  // daily snapshots: today = mock summary; earlier days back-cast with slow growth; new/lost from the rows
  const summary = await mockSeoData.backlinkSummary(own);
  const snapRows: Array<{ propertyId: string; date: Date; backlinks: number; referringDomains: number; dofollow: number; nofollow: number; newBacklinks: number; lostBacklinks: number; domainRank: number; toxicShare: number }> = [];
  for (let i = 0; i < DAYS; i++) {
    const d = dayAt(i);
    const back = DAYS - 1 - i;
    const g = 1 - 0.0022 * back + (back === 0 ? 0 : 0.006 * Math.sin(i / 9));
    const newToday = blRows.filter((b) => b.firstSeen.getTime() === d.getTime()).length;
    const lostToday = blRows.filter((b) => b.isLost && b.lastSeen.getTime() === d.getTime()).length;
    const spike = i === DAYS - 9 ? 6 : 0; // one "lost" spike to make the alert story visible
    snapRows.push({
      propertyId,
      date: d,
      backlinks: Math.round(summary.backlinks * g),
      referringDomains: Math.round(summary.referringDomains * g),
      dofollow: Math.round(summary.dofollow * g),
      nofollow: Math.round(summary.nofollow * g),
      newBacklinks: back === 0 ? Math.round(summary.newLast30 / 30) : newToday * 2 + int(0, 2),
      lostBacklinks: back === 0 ? Math.round(summary.lostLast30 / 30) : lostToday + spike + (rnd() < 0.3 ? 1 : 0),
      domainRank: summary.domainRank - (back > DAYS / 2 ? 1 : 0),
      toxicShare: summary.toxicShare,
    });
  }
  await db.backlinkSnapshot.createMany({ data: snapRows });

  // cached research runs (mode is prefixed into `seed`, see features/seo-suite/queries.ts)
  const seeds = [c.queries[0]!, c.queries[1]!];
  for (const seed of seeds) {
    const ideas: Idea[] = [ideaFor(seed), ...MODIFIERS.map((m) => ideaFor(`${seed} ${m}`)), ...CITIES.map((city) => ideaFor(`${seed} ${city}`))].sort((a, b) => b.volume - a.volume);
    await db.keywordResearch.create({
      data: { clientId, seed: `ideas:${seed}`, locationCode: 2360, languageCode: "id", provider: "mock", results: ideas, fetchedAt: addDays(TODAY, -int(0, 3)) },
    });
    counts.research++;
  }
  return counts;
}

export async function seedSeoSuite(db: PrismaClient) {
  console.log("→ SEO suite: menghapus data lama…");
  await db.$transaction([
    db.rankSnapshot.deleteMany(),
    db.trackedKeyword.deleteMany(),
    db.keywordResearch.deleteMany(),
    db.backlink.deleteMany(),
    db.backlinkSnapshot.deleteMany(),
    db.domainSnapshot.deleteMany(),
    db.competitorDomain.deleteMany(),
    db.syncJob.deleteMany({ where: { kind: { startsWith: "SEO_SUITE" } } }),
  ]);
  for (const [ci, c] of SEED_CLIENTS.entries()) {
    const client = await db.client.findUnique({ where: { slug: c.slug }, select: { id: true } });
    if (!client) {
      console.log(`  · ${c.slug}: klien belum ada, lewati`);
      continue;
    }
    const property = await db.seoProperty.findFirst({ where: { clientId: client.id }, orderBy: { createdAt: "asc" }, select: { id: true } });
    if (!property) {
      console.log(`  · ${c.slug}: properti SEO belum ada, lewati`);
      continue;
    }
    const counts = await seedProperty(db, property.id, client.id, c, ci);
    console.log(`  · ${c.slug}: ${counts.keywords} kata kunci, ${counts.ranks} rank, ${counts.domains} domain snapshot, ${counts.backlinks} backlink, ${counts.research} riset`);
  }
}

// standalone entry point
if (process.argv[1] && /seed-seo-suite\.ts$/.test(process.argv[1])) {
  const adapter = new PrismaPg({ connectionString: process.env.DATABASE_URL });
  const db = new PrismaClient({ adapter });
  seedSeoSuite(db)
    .catch((e) => {
      console.error(e);
      process.exit(1);
    })
    .finally(async () => {
      await db.$disconnect();
    });
}
