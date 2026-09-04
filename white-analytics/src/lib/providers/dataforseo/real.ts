/**
 * Real DataForSEO REST v3 client. Basic auth from DATAFORSEO_LOGIN / DATAFORSEO_PASSWORD,
 * plain fetch, every failure normalised into ProviderError("dataforseo", …).
 * Docs: https://docs.dataforseo.com/v3/
 */
import "server-only";
import { ProviderError, type ProviderErrorCode } from "@/lib/action-result";
import { toISODate, todayUtc } from "@/lib/dates";
import type {
  BacklinkItem,
  BacklinkSummary,
  CompetitorSuggestion,
  DomainKeyword,
  DomainOverview,
  KeywordIdea,
  KeywordIntentKey,
  RankResult,
  SeoDataProvider,
  SerpTopResult,
} from "./types";

const BASE = "https://api.dataforseo.com";
const TIMEOUT_MS = 60_000;
/** DataForSEO prices in USD; we present IDR. */
const USD_TO_IDR = Number(process.env.DATAFORSEO_USD_IDR ?? 16_000);

type DfsResponse<T> = {
  status_code: number;
  status_message: string;
  tasks?: Array<{ status_code: number; status_message: string; result?: T[] | null }>;
};

function codeForStatus(status: number): ProviderErrorCode {
  if (status === 401) return "TOKEN_EXPIRED";
  if (status === 402 || status === 403) return "PERMISSION";
  if (status === 429) return "RATE_LIMIT";
  if (status === 404) return "NOT_FOUND";
  return "UNKNOWN";
}

function messageFor(code: ProviderErrorCode, api: string): string {
  switch (code) {
    case "TOKEN_EXPIRED":
      return `Kredensial DataForSEO ditolak saat memanggil ${api}. Periksa DATAFORSEO_LOGIN/PASSWORD.`;
    case "PERMISSION":
      return `Akun DataForSEO tidak punya akses/saldo untuk ${api}.`;
    case "RATE_LIMIT":
      return `Batas permintaan DataForSEO tercapai (${api}). Coba lagi beberapa menit.`;
    case "NOT_FOUND":
      return `Endpoint DataForSEO tidak ditemukan (${api}).`;
    case "NETWORK":
      return `Gangguan jaringan saat memanggil DataForSEO (${api}).`;
    default:
      return `DataForSEO mengembalikan kesalahan saat memanggil ${api}.`;
  }
}

async function post<T>(path: string, payload: unknown[]): Promise<T[]> {
  const login = process.env.DATAFORSEO_LOGIN ?? "";
  const password = process.env.DATAFORSEO_PASSWORD ?? "";
  const auth = Buffer.from(`${login}:${password}`).toString("base64");
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), TIMEOUT_MS);
  let res: Response;
  try {
    res = await fetch(`${BASE}${path}`, {
      method: "POST",
      headers: { Authorization: `Basic ${auth}`, "Content-Type": "application/json" },
      body: JSON.stringify(payload),
      signal: controller.signal,
    });
  } catch (e) {
    throw new ProviderError("dataforseo", "NETWORK", messageFor("NETWORK", path), { cause: e });
  } finally {
    clearTimeout(timer);
  }
  if (!res.ok) {
    const code = codeForStatus(res.status);
    throw new ProviderError("dataforseo", code, messageFor(code, path));
  }
  const json = (await res.json()) as DfsResponse<T>;
  if (json.status_code !== 20000) {
    const code: ProviderErrorCode = json.status_code === 40200 || json.status_code === 40201 ? "PERMISSION" : json.status_code === 40202 ? "RATE_LIMIT" : "UNKNOWN";
    throw new ProviderError("dataforseo", code, `${messageFor(code, path)} (${json.status_message})`);
  }
  const task = json.tasks?.[0];
  if (!task) return [];
  if (task.status_code !== 20000) {
    const code: ProviderErrorCode = task.status_code === 40402 ? "NOT_FOUND" : task.status_code >= 40200 && task.status_code < 40300 ? "PERMISSION" : "UNKNOWN";
    throw new ProviderError("dataforseo", code, `${messageFor(code, path)} (${task.status_message})`);
  }
  return task.result ?? [];
}

// ── normalisers ──────────────────────────────────────────────

type DfsKeywordItem = {
  keyword?: string;
  keyword_info?: { search_volume?: number | null; cpc?: number | null; competition?: number | null; monthly_searches?: Array<{ year: number; month: number; search_volume: number }> | null };
  keyword_properties?: { keyword_difficulty?: number | null };
  search_intent_info?: { main_intent?: string | null };
  serp_info?: { serp_item_types?: string[] | null };
};

function intentFrom(v: string | null | undefined): KeywordIntentKey {
  switch ((v ?? "").toLowerCase()) {
    case "navigational":
      return "NAVIGATIONAL";
    case "commercial":
      return "COMMERCIAL";
    case "transactional":
      return "TRANSACTIONAL";
    default:
      return "INFORMATIONAL";
  }
}

function ideaFrom(item: DfsKeywordItem): KeywordIdea {
  const monthly = [...(item.keyword_info?.monthly_searches ?? [])].sort((a, b) => a.year - b.year || a.month - b.month).slice(-12);
  const volume = item.keyword_info?.search_volume ?? 0;
  const trend = monthly.length ? monthly.map((m) => m.search_volume) : Array.from({ length: 12 }, () => volume);
  while (trend.length < 12) trend.unshift(trend[0] ?? volume);
  return {
    keyword: item.keyword ?? "",
    volume,
    difficulty: Math.round(item.keyword_properties?.keyword_difficulty ?? 0),
    cpc: Math.round((item.keyword_info?.cpc ?? 0) * USD_TO_IDR),
    competition: item.keyword_info?.competition ?? 0,
    intent: intentFrom(item.search_intent_info?.main_intent),
    trend,
    serpFeatures: item.serp_info?.serp_item_types ?? [],
  };
}

function hostOf(url: string): string {
  try {
    return new URL(url).hostname.replace(/^www\./, "");
  } catch {
    return url;
  }
}

function sameDomain(host: string, target: string): boolean {
  const h = host.replace(/^www\./, "");
  return h === target || h.endsWith(`.${target}`);
}

// ── provider ─────────────────────────────────────────────────

export const realSeoData: SeoDataProvider = {
  async keywordIdeas(seed, { locationCode, languageCode, mode, limit = 100 }) {
    const common = { location_code: locationCode, language_code: languageCode, limit, include_serp_info: true };
    let items: Array<{ items?: DfsKeywordItem[] }>;
    if (mode === "ideas") {
      items = await post("/v3/dataforseo_labs/google/keyword_ideas/live", [{ keywords: [seed], ...common, include_clickstream_data: false }]);
    } else if (mode === "questions") {
      items = await post("/v3/dataforseo_labs/google/keyword_suggestions/live", [{ keyword: seed, ...common, filters: ["keyword_info.search_volume", ">", 0] }]);
    } else {
      const rel = await post<{ items?: Array<{ keyword_data?: DfsKeywordItem }> }>("/v3/dataforseo_labs/google/related_keywords/live", [{ keyword: seed, ...common, depth: 2 }]);
      items = rel.map((r) => ({ items: (r.items ?? []).map((i) => i.keyword_data ?? {}) }));
    }
    let ideas = items.flatMap((r) => r.items ?? []).map(ideaFrom).filter((i) => i.keyword);
    if (mode === "questions") {
      const q = /^(cara|apa|bagaimana|kenapa|mengapa|berapa|apakah|dimana|di mana|kapan|siapa|how|what|why|where|when|who)\b/i;
      ideas = ideas.filter((i) => q.test(i.keyword));
    }
    return ideas.sort((a, b) => b.volume - a.volume).slice(0, limit);
  },

  async keywordMetrics(keywords, locationCode, languageCode) {
    if (keywords.length === 0) return [];
    const [volumes, difficulty] = await Promise.all([
      post<{ items?: DfsKeywordItem[] }>("/v3/dataforseo_labs/google/historical_search_volume/live", [{ keywords, location_code: locationCode, language_code: languageCode, include_serp_info: true }]).catch(async () => {
        // fall back to Google Ads volumes (no intent/serp info)
        const rows = await post<{ keyword: string; search_volume: number | null; cpc: number | null; competition_index: number | null; monthly_searches: Array<{ year: number; month: number; search_volume: number }> | null }>(
          "/v3/keywords_data/google_ads/search_volume/live",
          [{ keywords, location_code: locationCode, language_code: languageCode }],
        );
        return [{ items: rows.map((r) => ({ keyword: r.keyword, keyword_info: { search_volume: r.search_volume, cpc: r.cpc, competition: (r.competition_index ?? 0) / 100, monthly_searches: r.monthly_searches } })) }];
      }),
      post<{ items?: Array<{ keyword: string; keyword_difficulty: number | null }> }>("/v3/dataforseo_labs/google/bulk_keyword_difficulty/live", [{ keywords, location_code: locationCode, language_code: languageCode }]).catch(() => []),
    ]);
    const kd = new Map<string, number>();
    for (const r of difficulty) for (const i of r.items ?? []) kd.set(i.keyword.toLowerCase(), Math.round(i.keyword_difficulty ?? 0));
    const byKeyword = new Map<string, KeywordIdea>();
    for (const r of volumes) for (const i of r.items ?? []) {
      const idea = ideaFrom(i);
      idea.difficulty = kd.get(idea.keyword.toLowerCase()) ?? idea.difficulty;
      byKeyword.set(idea.keyword.toLowerCase(), idea);
    }
    return keywords.map((k) => byKeyword.get(k.toLowerCase()) ?? { keyword: k, volume: 0, difficulty: kd.get(k.toLowerCase()) ?? 0, cpc: 0, competition: 0, intent: "INFORMATIONAL" as const, trend: Array.from({ length: 12 }, () => 0), serpFeatures: [] });
  },

  async serpRanks(items, { locationCode, languageCode, targetDomain }) {
    if (items.length === 0) return [];
    type Item = { type: string; rank_absolute?: number; rank_group?: number; domain?: string; url?: string };
    const tasks = items.map((it) => ({ keyword: it.keyword, location_code: locationCode, language_code: languageCode, device: it.device.toLowerCase(), depth: 100 }));
    const results: RankResult[] = [];
    // live/regular accepts one task per call; run in small parallel batches
    for (let i = 0; i < tasks.length; i += 5) {
      const batch = tasks.slice(i, i + 5);
      const settled = await Promise.all(batch.map((task) => post<{ item_types?: string[]; items?: Item[] }>("/v3/serp/google/organic/live/regular", [task])));
      settled.forEach((res, j) => {
        const src = items[i + j]!;
        const organic = (res[0]?.items ?? []).filter((x) => x.type === "organic");
        const mine = organic.find((x) => sameDomain(x.domain ?? hostOf(x.url ?? ""), targetDomain));
        results.push({
          keyword: src.keyword,
          device: src.device,
          position: mine?.rank_group ?? mine?.rank_absolute ?? null,
          url: mine?.url ?? null,
          serpFeatures: (res[0]?.item_types ?? []).filter((t) => t !== "organic"),
          top10: organic.slice(0, 10).map((x, k): SerpTopResult => ({ domain: (x.domain ?? hostOf(x.url ?? "")).replace(/^www\./, ""), position: x.rank_group ?? k + 1, url: x.url ?? "" })),
        });
      });
    }
    return results;
  },

  async backlinkSummary(domain) {
    type Sum = { backlinks?: number; referring_domains?: number; rank?: number; referring_links_attributes?: Record<string, number>; backlinks_spam_score?: number; referring_domains_nofollow?: number };
    const [sum] = await post<Sum>("/v3/backlinks/summary/live", [{ target: domain, include_subdomains: true, backlinks_status_type: "live" }]);
    const [hist] = await post<{ items?: Array<{ new_backlinks?: number; lost_backlinks?: number }> }>("/v3/backlinks/timeseries_new_lost_summary/live", [{ target: domain, date_from: toISODate(new Date(todayUtc().getTime() - 30 * 86_400_000)), group_range: "month" }]).catch(() => [{ items: [] }]);
    const nofollow = sum?.referring_links_attributes?.nofollow ?? 0;
    const backlinks = sum?.backlinks ?? 0;
    return {
      backlinks,
      referringDomains: sum?.referring_domains ?? 0,
      dofollow: Math.max(0, backlinks - nofollow),
      nofollow,
      domainRank: Math.round((sum?.rank ?? 0) / 10),
      newLast30: (hist?.items ?? []).reduce((a, i) => a + (i.new_backlinks ?? 0), 0),
      lostLast30: (hist?.items ?? []).reduce((a, i) => a + (i.lost_backlinks ?? 0), 0),
      toxicShare: Math.min(1, (sum?.backlinks_spam_score ?? 0) / 100),
    } satisfies BacklinkSummary;
  },

  async backlinks(domain, { limit = 200, mode }) {
    type Row = { url_from: string; domain_from: string; url_to: string; anchor?: string | null; dofollow?: boolean; domain_from_rank?: number; backlink_spam_score?: number; first_seen?: string; last_seen?: string; is_lost?: boolean };
    const filters = mode === "lost" ? ["is_lost", "=", true] : mode === "new" ? ["is_new", "=", true] : undefined;
    const [res] = await post<{ items?: Row[] }>("/v3/backlinks/backlinks/live", [{ target: domain, mode: "as_is", limit, order_by: ["rank,desc"], backlinks_status_type: mode === "lost" ? "all" : "live", ...(filters ? { filters } : {}) }]);
    return (res?.items ?? []).map(
      (r): BacklinkItem => ({
        sourceUrl: r.url_from,
        sourceDomain: r.domain_from,
        targetUrl: r.url_to,
        anchor: r.anchor ?? "",
        dofollow: r.dofollow ?? true,
        domainRank: Math.round((r.domain_from_rank ?? 0) / 10),
        spamScore: r.backlink_spam_score ?? 0,
        firstSeen: (r.first_seen ?? "").slice(0, 10),
        lastSeen: (r.last_seen ?? "").slice(0, 10),
        isLost: r.is_lost ?? false,
      }),
    );
  },

  async domainOverview(domain, locationCode, languageCode) {
    type Item = { metrics?: { organic?: { count?: number; etv?: number; estimated_paid_traffic_cost?: number; pos_1?: number; pos_2_3?: number; pos_4_10?: number; pos_11_20?: number; pos_21_30?: number; pos_31_40?: number; pos_41_50?: number; pos_51_60?: number; pos_61_70?: number; pos_71_80?: number; pos_81_90?: number; pos_91_100?: number } } };
    const [[res], links] = await Promise.all([
      post<{ items?: Item[] }>("/v3/dataforseo_labs/google/domain_rank_overview/live", [{ target: domain, location_code: locationCode, language_code: languageCode }]),
      realSeoData.backlinkSummary(domain).catch(() => null),
    ]);
    const o = res?.items?.[0]?.metrics?.organic ?? {};
    const top3 = (o.pos_1 ?? 0) + (o.pos_2_3 ?? 0);
    const top10 = top3 + (o.pos_4_10 ?? 0);
    return {
      domain,
      organicKeywords: o.count ?? 0,
      organicTraffic: Math.round(o.etv ?? 0),
      organicCost: Math.round((o.estimated_paid_traffic_cost ?? 0) * USD_TO_IDR),
      top3,
      top10,
      top100: o.count ?? 0,
      backlinks: links?.backlinks ?? 0,
      referringDomains: links?.referringDomains ?? 0,
      domainRank: links?.domainRank ?? 0,
    } satisfies DomainOverview;
  },

  async domainKeywords(domain, locationCode, languageCode, limit = 200) {
    type Item = { keyword_data?: { keyword?: string; keyword_info?: { search_volume?: number | null } }; ranked_serp_element?: { serp_item?: { rank_group?: number; url?: string } } };
    const [res] = await post<{ items?: Item[] }>("/v3/dataforseo_labs/google/ranked_keywords/live", [{ target: domain, location_code: locationCode, language_code: languageCode, limit, order_by: ["keyword_data.keyword_info.search_volume,desc"] }]);
    return (res?.items ?? [])
      .map(
        (i): DomainKeyword => ({
          keyword: i.keyword_data?.keyword ?? "",
          position: i.ranked_serp_element?.serp_item?.rank_group ?? 100,
          volume: i.keyword_data?.keyword_info?.search_volume ?? 0,
          url: i.ranked_serp_element?.serp_item?.url ?? "",
        }),
      )
      .filter((k) => k.keyword);
  },

  async competitors(domain, locationCode, languageCode) {
    type Item = { domain: string; intersections?: number; metrics?: { organic?: { etv?: number } } };
    const [res] = await post<{ items?: Item[] }>("/v3/dataforseo_labs/google/competitors_domain/live", [{ target: domain, location_code: locationCode, language_code: languageCode, limit: 10, exclude_top_domains: true }]);
    return (res?.items ?? [])
      .filter((i) => i.domain && i.domain !== domain)
      .map((i): CompetitorSuggestion => ({ domain: i.domain, intersections: i.intersections ?? 0, organicTraffic: Math.round(i.metrics?.organic?.etv ?? 0) }));
  },
};
