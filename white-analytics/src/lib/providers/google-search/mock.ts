/**
 * Deterministic mock for GSC + GA4 — same inputs always yield the same rows,
 * so demo dashboards are stable across reloads.
 */
import type { Ga4ReportRequest, Ga4ReportRow, GoogleSearchProvider, GscDimension, GscQueryRequest, GscRow } from "./types";

// xmur3 string hash → 32-bit seed
function hashSeed(str: string): number {
  let h = 1779033703 ^ str.length;
  for (let i = 0; i < str.length; i++) {
    h = Math.imul(h ^ str.charCodeAt(i), 3432918353);
    h = (h << 13) | (h >>> 19);
  }
  h = Math.imul(h ^ (h >>> 16), 2246822507);
  h = Math.imul(h ^ (h >>> 13), 3266489909);
  return (h ^= h >>> 16) >>> 0;
}

function mulberry32(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a |= 0;
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

function* eachDate(startDate: string, endDate: string): Generator<string> {
  const start = new Date(`${startDate}T00:00:00Z`);
  const end = new Date(`${endDate}T00:00:00Z`);
  for (let t = start.getTime(), guard = 0; t <= end.getTime() && guard < 500; t += 86_400_000, guard++) {
    yield new Date(t).toISOString().slice(0, 10);
  }
}

const MOCK_QUERIES = [
  "kopi susu gula aren", "kedai kopi jakarta", "biji kopi arabika", "cold brew coffee", "kopi kekinian",
  "harga kopi susu", "kopi terdekat", "resep es kopi susu", "kopi single origin", "kopi toraja",
  "coffee shop instagramable", "kopi arabika gayo", "franchise kopi", "kopi literan", "promo kopi hari ini",
  "kopi drip bag", "supplier biji kopi", "kopi tanpa gula", "manfaat kopi hitam", "cara seduh v60",
];
const MOCK_PAGES = ["/", "/menu", "/lokasi", "/promo", "/blog/resep-kopi", "/produk/biji-kopi", "/tentang", "/kontak", "/blog/arabika-vs-robusta", "/franchise"];
const MOCK_COUNTRIES = ["idn", "sgp", "mys", "aus", "usa"];
const MOCK_DEVICES = ["MOBILE", "DESKTOP", "TABLET"];
const GA4_CHANNELS = ["Organic Search", "Direct", "Paid Search", "Organic Social"];

function keysFor(dim: GscDimension, req: GscQueryRequest): string[] {
  switch (dim) {
    case "date":
      return [...eachDate(req.startDate, req.endDate)];
    case "query":
      return MOCK_QUERIES;
    case "page":
      return MOCK_PAGES.map((p) => `${req.siteUrl.replace(/\/$/, "").replace(/^sc-domain:/, "https://")}${p}`);
    case "country":
      return MOCK_COUNTRIES;
    case "device":
      return MOCK_DEVICES;
  }
}

/** Cartesian product of per-dimension key lists, capped. */
function crossProduct(lists: string[][], cap: number): string[][] {
  let out: string[][] = [[]];
  for (const list of lists) {
    const next: string[][] = [];
    for (const combo of out) {
      for (const k of list) {
        next.push([...combo, k]);
        if (next.length >= cap) break;
      }
      if (next.length >= cap) break;
    }
    out = next;
  }
  return out;
}

export const mockGoogleSearchProvider: GoogleSearchProvider = {
  async querySearchAnalytics(_accessToken: string, req: GscQueryRequest): Promise<GscRow[]> {
    const limit = Math.min(req.rowLimit ?? 1000, 5000);
    const lists = req.dimensions.map((d) => keysFor(d, req));
    const combos = crossProduct(lists.length ? lists : [["(all)"]], limit);
    return combos.map((keys) => {
      const rng = mulberry32(hashSeed(`gsc|${req.siteUrl}|${keys.join("|")}`));
      const base = 20 + Math.floor(rng() * 400);
      const impressions = base * (5 + Math.floor(rng() * 30));
      const position = 1 + rng() * 30;
      const ctrPct = Math.max(0.2, 30 / position - rng() * 2);
      const clicks = Math.min(impressions, Math.round((impressions * ctrPct) / 100));
      return { keys, clicks, impressions, ctr: impressions ? (clicks / impressions) * 100 : 0, position };
    });
  },

  async runGa4Report(_accessToken: string, req: Ga4ReportRequest): Promise<Ga4ReportRow[]> {
    const wantsDate = req.dimensions.includes("date");
    const wantsChannel = req.dimensions.includes("sessionDefaultChannelGroup");
    const dates = wantsDate ? [...eachDate(req.startDate, req.endDate)] : ["(all)"];
    const channels = wantsChannel ? GA4_CHANNELS : ["(all)"];
    const rows: Ga4ReportRow[] = [];
    for (const date of dates) {
      for (const channel of channels) {
        const rng = mulberry32(hashSeed(`ga4|${req.propertyId}|${date}|${channel}`));
        const channelShare = channel === "Organic Search" ? 0.45 : channel === "Direct" ? 0.3 : channel === "Paid Search" ? 0.15 : channel === "Organic Social" ? 0.1 : 1;
        const sessions = Math.round((800 + rng() * 900) * channelShare);
        const values: Record<string, number> = {
          sessions,
          engagedSessions: Math.round(sessions * (0.5 + rng() * 0.2)),
          totalUsers: Math.round(sessions * (0.75 + rng() * 0.15)),
          conversions: Math.round(sessions * (0.015 + rng() * 0.02)),
          userEngagementDuration: Math.round(sessions * (40 + rng() * 60)),
        };
        const dims: string[] = req.dimensions.map((d) => (d === "date" ? date.replaceAll("-", "") : d === "sessionDefaultChannelGroup" ? channel : "(other)"));
        rows.push({ dimensions: dims, metrics: req.metrics.map((m) => values[m] ?? 0) });
      }
    }
    return rows.slice(0, req.limit ?? rows.length);
  },
};
