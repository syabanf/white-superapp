/**
 * Mock DataForSEO — deterministic (hash of the inputs), Indonesian-flavoured data used when
 * DATAFORSEO_LOGIN/PASSWORD are missing. No network, no secrets. Ranks are a stable random
 * walk keyed by keyword + date so history built from repeated calls looks real.
 */
import { addDays, toISODate, todayUtc } from "@/lib/dates";
import { hashSeed, mulberry32 } from "@/lib/providers/meta-graph/mock";
import type {
  BacklinkItem,
  BacklinkSummary,
  CompetitorSuggestion,
  DomainKeyword,
  DomainOverview,
  KeywordIdea,
  KeywordIntentKey,
  RankResult,
  ResearchMode,
  SeoDataProvider,
  SerpRankRequest,
  SerpTopResult,
} from "./types";

const MODIFIERS = ["harga", "terdekat", "terbaik", "murah", "review", "2026", "promo", "online", "rekomendasi", "grosir"];
const QUESTION_PREFIX = ["cara", "apa itu", "bagaimana", "kenapa", "berapa harga", "apakah", "dimana beli", "kapan"];
const CITIES = ["jakarta", "bandung", "surabaya", "yogyakarta", "bali", "medan", "semarang", "makassar", "tangerang", "bekasi"];
const RELATED_SUFFIX = ["premium", "lokal", "kekinian", "untuk pemula", "paket", "bisnis", "rumahan", "modern", "tradisional", "keluarga"];
const SERP_FEATURES = ["featured_snippet", "people_also_ask", "images", "video", "local_pack", "shopping", "sitelinks", "reviews", "top_stories", "knowledge_panel"];

const GENERIC_DOMAINS = ["tokopedia.com", "shopee.co.id", "kompas.com", "detik.com", "tribunnews.com", "wikipedia.org", "idntimes.com", "liputan6.com", "kumparan.com", "cnnindonesia.com", "medium.com", "blogspot.com", "wordpress.com", "traveloka.com", "rumah123.com", "olx.co.id", "bukalapak.com", "tiket.com", "gramedia.com", "sehatq.com"];
const BACKLINK_DOMAINS = ["kompas.com", "detik.com", "tribunnews.com", "medium.com", "blogspot.com", "wordpress.com", "liputan6.com", "kumparan.com", "idntimes.com", "tempo.co", "suara.com", "merdeka.com", "brilio.net", "hipwee.com", "pikiran-rakyat.com", "kaskus.co.id", "quora.com", "reddit.com", "linkedin.com", "pinterest.com", "issuu.com", "weebly.com", "wixsite.com", "zenn.dev", "seo-backlink-cheap.xyz", "linkfarm-id.info", "bestlinks-2024.top", "artikel-murah.site"];
const ANCHORS = ["{brand}", "{brand}", "kunjungi situs", "baca selengkapnya", "klik di sini", "{brand} resmi", "website", "sumber", "{brand}.id", "lihat di sini", "info lengkap"];

const INDUSTRIES: { key: string; match: RegExp; seed: string; competitors: string[] }[] = [
  { key: "kopi", match: /kopi|coffee|cafe|kafe|ngopi|arabika|roast|brew/, seed: "kopi", competitors: ["kopikenangan.com", "janjijiwa.com", "fore.coffee", "tomoro.id", "kopisenja.id", "ngopidulu.com"] },
  { key: "properti", match: /propert|residence|rumah|land|estate|realty|adiharjo|cluster|home/, seed: "rumah dijual", competitors: ["rumah123.com", "summarecon.com", "sinarmasland.com", "ciputra.com", "lippokarawaci.co.id", "agungpodomoro.com"] },
  { key: "villa", match: /villa|bali|hotel|stay|escape|resort|travel|trip/, seed: "villa bali", competitors: ["airbnb.com", "booking.com", "balivillas.com", "elitehavens.com", "thebalibible.com", "villa-bali.com"] },
];
const GENERIC_INDUSTRY = { key: "umum", seed: "jasa", competitors: ["kompas.com", "detik.com", "tokopedia.com", "shopee.co.id", "olx.co.id", "medium.com"] };

function industryFor(domain: string) {
  const d = domain.toLowerCase();
  return INDUSTRIES.find((i) => i.match.test(d)) ?? GENERIC_INDUSTRY;
}

function brandFromDomain(domain: string): string {
  return domain.replace(/^www\./, "").split(".")[0] ?? domain;
}

function unit(key: string): number {
  return hashSeed(key) / 4294967296;
}

/** Log-normal volume: heavy tail, most keywords small. */
function volumeFor(keyword: string, base: number): number {
  const r = mulberry32(hashSeed(`vol:${keyword}`));
  const u = 1 - r();
  const v = r();
  const z = Math.sqrt(-2 * Math.log(u)) * Math.cos(2 * Math.PI * v);
  const raw = Math.exp(Math.log(base) + z * 0.9);
  const rounded = raw < 100 ? Math.round(raw / 10) * 10 : raw < 1000 ? Math.round(raw / 50) * 50 : Math.round(raw / 100) * 100;
  return Math.max(10, rounded);
}

function intentFor(keyword: string): KeywordIntentKey {
  const k = keyword.toLowerCase();
  if (/^(cara|apa itu|bagaimana|kenapa|apakah|kapan|manfaat|beda|resep|tips)/.test(k) || /\b(cara|manfaat|resep|tips|arti)\b/.test(k)) return "INFORMATIONAL";
  if (/\b(beli|jual|harga|promo|murah|diskon|order|pesan|grosir|paket)\b/.test(k)) return "TRANSACTIONAL";
  if (/\b(terbaik|review|rekomendasi|vs|terdekat|top|perbandingan)\b/.test(k)) return "COMMERCIAL";
  return unit(`intent:${k}`) < 0.15 ? "NAVIGATIONAL" : "INFORMATIONAL";
}

function ideaFor(keyword: string, seedBase: number): KeywordIdea {
  const volume = volumeFor(keyword, seedBase);
  const r = mulberry32(hashSeed(`idea:${keyword}`));
  // KD correlated with log(volume) + noise
  const kd = Math.round(Math.min(100, Math.max(2, 12 * Math.log10(volume) + (r() - 0.5) * 30)));
  const intent = intentFor(keyword);
  const cpcBase = intent === "TRANSACTIONAL" ? 6000 : intent === "COMMERCIAL" ? 4000 : intent === "NAVIGATIONAL" ? 1200 : 1500;
  const cpc = Math.round(Math.min(15_000, Math.max(500, cpcBase * (0.5 + r() * 1.4) + kd * 40)) / 50) * 50;
  const competition = Math.round(Math.min(1, Math.max(0.02, (intent === "TRANSACTIONAL" ? 0.6 : 0.25) + (r() - 0.5) * 0.5)) * 100) / 100;
  const slope = (r() - 0.45) * 0.04;
  const phase = r() * Math.PI * 2;
  const trend = Array.from({ length: 12 }, (_, i) => Math.max(10, Math.round(volume * (1 + slope * (i - 11) + 0.12 * Math.sin(phase + (i / 12) * Math.PI * 2) + (r() - 0.5) * 0.1))));
  trend[11] = volume;
  const features = SERP_FEATURES.filter((f, i) => r() < (i < 2 ? 0.45 : 0.2));
  if (intent === "TRANSACTIONAL" && !features.includes("shopping") && r() < 0.6) features.push("shopping");
  if (/terdekat|jakarta|bandung|surabaya|bali/.test(keyword) && !features.includes("local_pack")) features.push("local_pack");
  return { keyword, volume, difficulty: kd, cpc, competition, intent, trend, serpFeatures: features };
}

function expandSeed(seed: string, mode: ResearchMode, limit: number): string[] {
  const s = seed.trim().toLowerCase().replace(/\s+/g, " ");
  const r = mulberry32(hashSeed(`expand:${mode}:${s}`));
  const out = new Set<string>();
  if (mode === "ideas") {
    out.add(s);
    for (const m of MODIFIERS) out.add(`${s} ${m}`);
    for (const m of ["harga", "review", "grosir", "rekomendasi"]) if (r() < 0.7) out.add(`${m} ${s}`);
    for (const c of CITIES) out.add(`${s} ${c}`);
    for (const x of RELATED_SUFFIX) out.add(`${s} ${x}`);
    for (const m of MODIFIERS.slice(0, 5)) for (const c of CITIES.slice(0, 4)) out.add(`${s} ${m} ${c}`);
  } else if (mode === "questions") {
    for (const q of QUESTION_PREFIX) out.add(`${q} ${s}`);
    for (const q of QUESTION_PREFIX.slice(0, 5)) for (const x of RELATED_SUFFIX.slice(0, 5)) out.add(`${q} ${s} ${x}`);
    out.add(`${s} itu apa`);
    out.add(`${s} bagus tidak`);
    out.add(`${s} untuk apa`);
    out.add(`kelebihan ${s}`);
    out.add(`kekurangan ${s}`);
    out.add(`manfaat ${s}`);
  } else {
    const words = s.split(" ");
    const head = words[0] ?? s;
    for (const x of RELATED_SUFFIX) out.add(`${head} ${x}`);
    for (const m of MODIFIERS) out.add(`${head} ${m}`);
    out.add(`jenis ${s}`);
    out.add(`${s} vs`);
    out.add(`alternatif ${s}`);
    out.add(`${s} terpopuler`);
    out.add(`supplier ${s}`);
    out.add(`${s} kemasan`);
    for (const c of CITIES.slice(0, 6)) out.add(`${head} ${c}`);
  }
  const arr = [...out];
  // deterministic shuffle so the order is not alphabetical by construction
  for (let i = arr.length - 1; i > 0; i--) {
    const j = Math.floor(r() * (i + 1));
    [arr[i], arr[j]] = [arr[j]!, arr[i]!];
  }
  return arr.slice(0, limit);
}

/** Key that makes desktop and mobile walk independently for the same target. */
export function rankKey(targetDomain: string, device: "DESKTOP" | "MOBILE"): string {
  return device === "DESKTOP" ? `${targetDomain}#d` : targetDomain;
}

function isBrandKeyword(keyword: string, targetDomain: string): boolean {
  const brand = brandFromDomain(targetDomain.replace(/#d$/, "")).toLowerCase();
  return brand.length >= 4 && keyword.toLowerCase().replace(/[^a-z0-9]/g, "").includes(brand);
}

/**
 * Position for a keyword on a date — stable per (keyword, target, date) so repeated calls and the
 * seed script agree. Archetypes: brand (#1–3), climbers/sliders (slow drift), a few that fall out of
 * the top 100 for long stretches; plus slow sine drift and small per-day jitter.
 */
export function mockPositionOn(keyword: string, target: string, day: Date): number | null {
  const key = `${target}|${keyword.toLowerCase()}`;
  const r = mulberry32(hashSeed(`rank:${key}`));
  const dayIdx = Math.floor(day.getTime() / 86_400_000);
  const jitter = (unit(`j:${key}:${toISODate(day)}`) - 0.5) * 2.2;
  if (isBrandKeyword(keyword, target)) {
    return Math.max(1, Math.round(1.2 + r() * 1.5 + 0.6 * Math.sin(dayIdx / 11) + jitter * 0.4));
  }
  const arche = r();
  const base = 2 + Math.pow(r(), 1.6) * 60; // most between 2 and ~25
  const amp = 1 + r() * 8;
  const period = 20 + r() * 60;
  const phase = r() * Math.PI * 2;
  let slope = (r() - 0.5) * 0.08; // positions/day drift
  if (arche < 0.08) slope = 0.5; // falls off within ~half a year
  else if (arche < 0.22) slope = -0.12; // steady climber
  else if (arche < 0.34) slope = 0.12; // steady slider
  const pos = base + amp * Math.sin(phase + (dayIdx / period) * Math.PI * 2) + slope * (dayIdx % 365) + jitter;
  if (pos > 100) return null;
  if (arche >= 0.08 && arche < 0.12 && Math.sin(phase + dayIdx / 15) > 0.35) return null; // in-and-out of the top 100
  return Math.max(1, Math.round(pos));
}

export function mockTop10(keyword: string, targetDomain: string, position: number | null, day: Date): SerpTopResult[] {
  const industry = industryFor(targetDomain);
  const pool = [...industry.competitors, ...GENERIC_DOMAINS];
  const r = mulberry32(hashSeed(`top:${keyword}:${toISODate(day).slice(0, 7)}`));
  const picked: string[] = [];
  while (picked.length < 10) {
    const d = pool[Math.floor(r() * pool.length)]!;
    if (!picked.includes(d) && d !== targetDomain) picked.push(d);
  }
  const slug = keyword.toLowerCase().replace(/[^a-z0-9]+/g, "-");
  const out: SerpTopResult[] = [];
  let k = 0;
  for (let p = 1; p <= 10; p++) {
    if (position === p) out.push({ domain: targetDomain, position: p, url: `https://${targetDomain}/${slug}` });
    else {
      const d = picked[k++]!;
      out.push({ domain: d, position: p, url: `https://${d}/${slug}` });
    }
  }
  return out;
}

export function mockBacklinkList(domain: string): BacklinkItem[] {
  const r = mulberry32(hashSeed(`bl:${domain}`));
  const brand = brandFromDomain(domain);
  const today = todayUtc();
  const n = 70 + Math.floor(r() * 50);
  const out: BacklinkItem[] = [];
  for (let i = 0; i < n; i++) {
    const src = BACKLINK_DOMAINS[Math.floor(r() * BACKLINK_DOMAINS.length)]!;
    const spammy = /xyz|info|top|site$/.test(src);
    const spamScore = spammy ? 60 + Math.floor(r() * 40) : Math.floor(r() * 35);
    const domainRank = spammy ? 2 + Math.floor(r() * 12) : 20 + Math.floor(r() * 75);
    const firstAge = Math.floor(Math.pow(r(), 1.4) * 400);
    const firstSeen = addDays(today, -firstAge);
    const isLost = r() < 0.14 && firstAge > 20;
    const lastSeen = isLost ? addDays(firstSeen, Math.max(3, Math.floor(r() * firstAge))) : addDays(today, -Math.floor(r() * 3));
    const anchor = ANCHORS[Math.floor(r() * ANCHORS.length)]!.replace("{brand}", brand);
    const path = ["/", "/blog/", "/promo/", "/tentang", "/produk/"][Math.floor(r() * 5)]!;
    out.push({
      sourceUrl: `https://${src}/${["artikel", "berita", "post", "review", "p"][Math.floor(r() * 5)]}/${brand}-${Math.floor(r() * 90000 + 1000)}`,
      sourceDomain: src,
      targetUrl: `https://${domain}${path}`,
      anchor,
      dofollow: r() < 0.68,
      domainRank,
      spamScore,
      firstSeen: toISODate(firstSeen),
      lastSeen: toISODate(lastSeen),
      isLost,
    });
  }
  return out;
}

export const mockSeoData: SeoDataProvider = {
  async keywordIdeas(seed, { mode, limit = 60 }) {
    const base = 300 + unit(`base:${seed.toLowerCase()}`) * 3000;
    return expandSeed(seed, mode, limit).map((k) => ideaFor(k, base)).sort((a, b) => b.volume - a.volume);
  },

  async keywordMetrics(keywords) {
    return keywords.map((k) => ideaFor(k.trim().toLowerCase(), 200 + unit(`base:${k.split(" ")[0]}`) * 1500));
  },

  async serpRanks(items: SerpRankRequest[], { targetDomain }) {
    const day = todayUtc();
    return items.map((it) => {
      const position = mockPositionOn(it.keyword, rankKey(targetDomain, it.device), day);
      const slug = it.keyword.toLowerCase().replace(/[^a-z0-9]+/g, "-");
      return {
        keyword: it.keyword,
        device: it.device,
        position,
        url: position == null ? null : `https://${targetDomain}/${slug}`,
        serpFeatures: ideaFor(it.keyword, 500).serpFeatures,
        top10: mockTop10(it.keyword, targetDomain, position, day),
      } satisfies RankResult;
    });
  },

  async backlinkSummary(domain) {
    const list = mockBacklinkList(domain);
    const live = list.filter((b) => !b.isLost);
    const today = todayUtc().getTime();
    const dofollow = live.filter((b) => b.dofollow).length;
    const toxic = live.filter((b) => b.spamScore >= 60).length;
    const r = mulberry32(hashSeed(`dr:${domain}`));
    const summary: BacklinkSummary = {
      backlinks: live.length * (12 + Math.floor(r() * 30)),
      referringDomains: new Set(live.map((b) => b.sourceDomain)).size * (3 + Math.floor(r() * 6)),
      dofollow: dofollow * 14,
      nofollow: (live.length - dofollow) * 14,
      domainRank: 18 + Math.floor(r() * 40),
      newLast30: list.filter((b) => today - new Date(b.firstSeen).getTime() <= 30 * 86_400_000).length,
      lostLast30: list.filter((b) => b.isLost && today - new Date(b.lastSeen).getTime() <= 30 * 86_400_000).length,
      toxicShare: live.length ? toxic / live.length : 0,
    };
    return summary;
  },

  async backlinks(domain, { limit = 200, mode }) {
    const today = todayUtc().getTime();
    let list = mockBacklinkList(domain);
    if (mode === "new") list = list.filter((b) => !b.isLost && today - new Date(b.firstSeen).getTime() <= 30 * 86_400_000);
    if (mode === "lost") list = list.filter((b) => b.isLost);
    return list.slice(0, limit);
  },

  async domainOverview(domain) {
    const r = mulberry32(hashSeed(`dom:${domain}`));
    const scale = 0.4 + Math.pow(r(), 1.3) * 6;
    const drift = 1 + ((Math.floor(todayUtc().getTime() / 86_400_000) % 90) / 90) * (r() - 0.4) * 0.2;
    const organicKeywords = Math.round(800 * scale * drift);
    const top100 = Math.round(organicKeywords * 0.92);
    const top10 = Math.round(top100 * (0.08 + r() * 0.1));
    const top3 = Math.round(top10 * (0.25 + r() * 0.2));
    const organicTraffic = Math.round(organicKeywords * (4 + r() * 10) * drift);
    const summary = await mockSeoData.backlinkSummary(domain);
    return {
      domain,
      organicKeywords,
      organicTraffic,
      organicCost: Math.round(organicTraffic * (900 + r() * 2500)),
      top3,
      top10,
      top100,
      backlinks: summary.backlinks,
      referringDomains: summary.referringDomains,
      domainRank: summary.domainRank,
    } satisfies DomainOverview;
  },

  async domainKeywords(domain, _loc, _lang, limit = 100) {
    const industry = industryFor(domain);
    const universe = [
      ...(await mockSeoData.keywordIdeas(industry.seed, { locationCode: 2360, languageCode: "id", mode: "ideas", limit: 60 })),
      ...(await mockSeoData.keywordIdeas(industry.seed, { locationCode: 2360, languageCode: "id", mode: "questions", limit: 25 })),
      ...(await mockSeoData.keywordIdeas(industry.seed, { locationCode: 2360, languageCode: "id", mode: "related", limit: 25 })),
    ];
    const out: DomainKeyword[] = [];
    for (const idea of universe) {
      if (unit(`dk:${domain}:${idea.keyword}`) > 0.58) continue;
      const pos = Math.max(1, Math.round(1 + Math.pow(unit(`dp:${domain}:${idea.keyword}`), 1.5) * 60));
      out.push({ keyword: idea.keyword, position: pos, volume: idea.volume, url: `https://${domain}/${idea.keyword.replace(/[^a-z0-9]+/g, "-")}` });
    }
    return out.sort((a, b) => b.volume - a.volume).slice(0, limit);
  },

  async competitors(domain) {
    const industry = industryFor(domain);
    return industry.competitors
      .filter((d) => d !== domain)
      .map((d) => {
        const r = mulberry32(hashSeed(`comp:${domain}:${d}`));
        return { domain: d, intersections: 40 + Math.floor(r() * 400), organicTraffic: Math.round(2000 + r() * 90_000) } satisfies CompetitorSuggestion;
      })
      .sort((a, b) => b.intersections - a.intersections);
  },
};
