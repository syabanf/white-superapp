/**
 * Meta Graph API (versi di ./version.ts) — implementasi nyata dengan `fetch` polos (tanpa SDK).
 * Field mengikuti design doc §7 Social. Error dinormalisasi ke `ProviderError`:
 *   190 → TOKEN_EXPIRED · 4/17/32/613 → RATE_LIMIT · 10/200–299 → PERMISSION · 100 → NOT_FOUND.
 *
 * Token: fungsi menerima User Token koneksi. Endpoint Instagram memakainya
 * langsung; endpoint Page Facebook menukarnya dulu ke Page Access Token.
 * Metrik mengikuti v26: `impressions` & `profile_views` IG sudah dihapus Meta
 * (diganti `views`, hanya total_value), reach Page = `page_total_media_view_unique`.
 */
import { ProviderError } from "@/lib/action-result";
import { addDays, eachDay, parseISODate, toISODate } from "@/lib/dates";
import type {
  FbPage,
  FbPageDailyInsight,
  FbPost,
  IgAccount,
  IgBusinessDiscovery,
  IgMedia,
  IgMediaProductType,
  IgMediaType,
  IgDailyInsight,
  MetaGraphProvider,
} from "./types";
import { META_GRAPH_BASE } from "./version";

const GRAPH_BASE = META_GRAPH_BASE;

type GraphErrorBody = { error?: { message?: string; type?: string; code?: number; error_subcode?: number } };

export function mapMetaError(status: number, body: GraphErrorBody | null): ProviderError {
  const err = body?.error;
  const code = err?.code;
  const message = err?.message ?? `Meta Graph API error (HTTP ${status})`;
  if (code === 190) return new ProviderError("meta", "TOKEN_EXPIRED", message);
  if (code === 4 || code === 17 || code === 32 || code === 613) return new ProviderError("meta", "RATE_LIMIT", message);
  // 10 & 200–299 = permission family
  if (code === 10 || (typeof code === "number" && code >= 200 && code <= 299)) {
    return new ProviderError("meta", "PERMISSION", message);
  }
  if (code === 100) return new ProviderError("meta", "NOT_FOUND", message);
  return new ProviderError("meta", "UNKNOWN", message);
}

async function graphGet<T>(path: string, token: string, params: Record<string, string> = {}): Promise<T> {
  const url = new URL(`${GRAPH_BASE}/${path.replace(/^\//, "")}`);
  for (const [k, v] of Object.entries(params)) url.searchParams.set(k, v);
  url.searchParams.set("access_token", token);
  let res: Response;
  try {
    res = await fetch(url.toString(), { cache: "no-store" });
  } catch (cause) {
    throw new ProviderError("meta", "NETWORK", "Tidak dapat menghubungi Meta Graph API", { cause });
  }
  const body = (await res.json().catch(() => null)) as (T & GraphErrorBody) | null;
  if (!res.ok || body == null || body.error) throw mapMetaError(res.status, body);
  return body;
}

// ── Page Access Token ─────────────────────────────────────────
const PAGE_TOKEN_TTL_MS = 10 * 60_000;
const pageTokens = new Map<string, { token: string; at: number }>();

/**
 * Tukar User Token ke Page Access Token (`GET /{page-id}?fields=access_token`).
 * Insight Page, daftar post Page, dan posting ke Page menolak User Token.
 * Disimpan di memori 10 menit saja, tidak pernah ke database.
 */
export async function getPageAccessToken(userToken: string, pageId: string): Promise<string> {
  const key = `${pageId}:${userToken.slice(-16)}`;
  const hit = pageTokens.get(key);
  if (hit && Date.now() - hit.at < PAGE_TOKEN_TTL_MS) return hit.token;
  const r = await graphGet<{ access_token?: string }>(pageId, userToken, { fields: "access_token" });
  if (!r.access_token) {
    throw new ProviderError("meta", "PERMISSION", "Token Page tidak tersedia. Pastikan akun yang login adalah admin Page ini.");
  }
  pageTokens.set(key, { token: r.access_token, at: Date.now() });
  return r.access_token;
}

/** Jalankan `fn` untuk tiap item, maksimal `limit` sekaligus (hormati rate limit Graph). */
async function mapLimit<T, R>(items: T[], limit: number, fn: (item: T) => Promise<R>): Promise<R[]> {
  const out = new Array<R>(items.length);
  let next = 0;
  const worker = async () => {
    while (next < items.length) {
      const i = next++;
      out[i] = await fn(items[i]!);
    }
  };
  await Promise.all(Array.from({ length: Math.min(limit, items.length) }, worker));
  return out;
}

// ── response shapes (subset yang kita pakai) ─────────────────
type RawIgAccount = {
  id: string;
  username?: string;
  name?: string;
  biography?: string;
  profile_picture_url?: string;
  followers_count?: number;
  follows_count?: number;
  media_count?: number;
};

type RawMedia = {
  id: string;
  caption?: string;
  like_count?: number;
  comments_count?: number;
  media_type?: string;
  media_product_type?: string;
  permalink?: string;
  media_url?: string;
  thumbnail_url?: string;
  timestamp?: string;
};

type InsightValue = { value?: number | Record<string, number>; end_time?: string };
type InsightsResponse = { data?: { name?: string; values?: InsightValue[] }[] };

const MEDIA_FIELDS =
  "id,caption,like_count,comments_count,media_type,media_product_type,permalink,media_url,thumbnail_url,timestamp";

function toMediaType(v: string | undefined): IgMediaType {
  return v === "VIDEO" || v === "CAROUSEL_ALBUM" ? v : "IMAGE";
}
function toProductType(v: string | undefined): IgMediaProductType | null {
  return v === "REELS" || v === "STORY" || v === "FEED" ? v : null;
}

function mapMedia(m: RawMedia): IgMedia {
  return {
    id: m.id,
    caption: m.caption ?? "",
    mediaType: toMediaType(m.media_type),
    mediaProductType: toProductType(m.media_product_type),
    permalink: m.permalink ?? null,
    mediaUrl: m.media_url ?? null,
    thumbnailUrl: m.thumbnail_url ?? m.media_url ?? null,
    timestamp: m.timestamp ?? new Date(0).toISOString(),
    likeCount: m.like_count ?? 0,
    commentsCount: m.comments_count ?? 0,
  };
}

/**
 * metric → (date → value). `end_time` Graph menandai AKHIR jendela 24 jam,
 * jadi nilai itu milik hari sebelumnya.
 */
function parseInsights(res: InsightsResponse): Map<string, Map<string, number>> {
  const out = new Map<string, Map<string, number>>();
  for (const series of res.data ?? []) {
    const name = series.name ?? "";
    const byDate = new Map<string, number>();
    for (const v of series.values ?? []) {
      const end = parseISODate((v.end_time ?? "").slice(0, 10));
      if (!end) continue;
      const date = toISODate(addDays(end, -1));
      byDate.set(date, typeof v.value === "number" ? v.value : 0);
    }
    out.set(name, byDate);
  }
  return out;
}

export const realMetaGraph: MetaGraphProvider = {
  async getInstagramAccount(token, igUserId): Promise<IgAccount> {
    const r = await graphGet<RawIgAccount>(igUserId, token, {
      fields: "id,username,name,biography,profile_picture_url,followers_count,follows_count,media_count",
    });
    return {
      id: r.id,
      username: r.username ?? "",
      name: r.name ?? r.username ?? "",
      biography: r.biography ?? null,
      profilePictureUrl: r.profile_picture_url ?? null,
      followersCount: r.followers_count ?? 0,
      followsCount: r.follows_count ?? 0,
      mediaCount: r.media_count ?? 0,
    };
  },

  async getInstagramInsights(token, igUserId, since, until): Promise<IgDailyInsight[]> {
    // reach: satu-satunya metrik akun yang masih punya deret harian (time_series).
    const res = await graphGet<InsightsResponse>(`${igUserId}/insights`, token, {
      metric: "reach",
      period: "day",
      metric_type: "time_series",
      since: toISODate(since),
      until: toISODate(addDays(until, 1)),
    });
    const reach = parseInsights(res).get("reach") ?? new Map<string, number>();
    // views (pengganti impressions) hanya total_value → satu panggilan per hari.
    const days = eachDay({ from: since, to: until });
    const views = await mapLimit(days, 5, async (d) => {
      const r = await graphGet<{ data?: { total_value?: { value?: number } }[] }>(`${igUserId}/insights`, token, {
        metric: "views",
        period: "day",
        metric_type: "total_value",
        since: toISODate(d),
        until: toISODate(addDays(d, 1)),
      }).catch(() => null); // `views` masih "in development" di Meta: jangan gagalkan seluruh sinkron
      return r?.data?.[0]?.total_value?.value ?? null;
    });
    return days.map((d, i) => ({
      date: toISODate(d),
      reach: reach.get(toISODate(d)) ?? 0,
      impressions: views[i] ?? 0,
      profileViews: null,
      followerCount: null,
    }));
  },

  async getInstagramMedia(token, igUserId, limit = 50): Promise<IgMedia[]> {
    const res = await graphGet<{ data?: RawMedia[] }>(`${igUserId}/media`, token, {
      fields: MEDIA_FIELDS,
      limit: String(limit),
    });
    const media = (res.data ?? []).map(mapMedia);
    // Insight per post (FEED & REELS). Gagal di satu post tidak menggagalkan yang lain.
    return mapLimit(media, 5, async (m) => {
      type MediaInsights = { data?: { name?: string; values?: { value?: number }[]; total_value?: { value?: number } }[] };
      const r = await graphGet<MediaInsights>(`${m.id}/insights`, token, { metric: "views,reach,saved,shares" }).catch(() => null);
      if (!r) return m;
      const get = (name: string) => {
        const s = r.data?.find((x) => x.name === name);
        return s?.total_value?.value ?? s?.values?.[0]?.value;
      };
      return { ...m, views: get("views"), reach: get("reach"), saved: get("saved"), shares: get("shares") };
    });
  },

  async discoverBusiness(token, igUserId, username): Promise<IgBusinessDiscovery> {
    const clean = username.trim().replace(/^@+/, "");
    const res = await graphGet<{ business_discovery?: RawIgAccount & { media?: { data?: RawMedia[] } } }>(
      igUserId,
      token,
      {
        fields: `business_discovery.username(${clean}){followers_count,media_count,username,name,biography,profile_picture_url,media.limit(25){${MEDIA_FIELDS}}}`,
      },
    );
    const bd = res.business_discovery;
    if (!bd) throw new ProviderError("meta", "NOT_FOUND", `Akun @${clean} tidak ditemukan lewat Business Discovery`);
    return {
      username: bd.username ?? clean,
      name: bd.name ?? bd.username ?? clean,
      biography: bd.biography ?? null,
      profilePictureUrl: bd.profile_picture_url ?? null,
      followersCount: bd.followers_count ?? 0,
      mediaCount: bd.media_count ?? 0,
      media: (bd.media?.data ?? []).map(mapMedia),
    };
  },

  async getFacebookPage(token, pageId): Promise<FbPage> {
    const r = await graphGet<{ id: string; name?: string; username?: string; fan_count?: number; followers_count?: number }>(
      pageId,
      await getPageAccessToken(token, pageId),
      { fields: "id,name,username,fan_count,followers_count" },
    );
    return {
      id: r.id,
      name: r.name ?? "",
      username: r.username ?? null,
      fanCount: r.fan_count ?? 0,
      followersCount: r.followers_count ?? r.fan_count ?? 0,
    };
  },

  async getPageInsights(token, pageId, since, until): Promise<FbPageDailyInsight[]> {
    // page_impressions_unique dihapus Meta (Jun 2025) → page_total_media_view_unique.
    const res = await graphGet<InsightsResponse>(`${pageId}/insights`, await getPageAccessToken(token, pageId), {
      metric: "page_total_media_view_unique,page_post_engagements",
      period: "day",
      since: toISODate(since),
      until: toISODate(addDays(until, 1)),
    });
    const m = parseInsights(res);
    const dates = new Set<string>();
    for (const byDate of m.values()) for (const d of byDate.keys()) dates.add(d);
    return [...dates].sort().map((date) => ({
      date,
      impressionsUnique: m.get("page_total_media_view_unique")?.get(date) ?? 0,
      postEngagements: m.get("page_post_engagements")?.get(date) ?? 0,
    }));
  },

  async getPagePosts(token, pageId, limit = 25): Promise<FbPost[]> {
    type RawPost = {
      id: string;
      message?: string;
      created_time?: string;
      permalink_url?: string;
      shares?: { count?: number };
      reactions?: { summary?: { total_count?: number } };
      comments?: { summary?: { total_count?: number } };
    };
    const res = await graphGet<{ data?: RawPost[] }>(`${pageId}/posts`, await getPageAccessToken(token, pageId), {
      fields: "id,message,created_time,permalink_url,shares,reactions.summary(true),comments.summary(true)",
      limit: String(limit),
    });
    return (res.data ?? []).map((p) => ({
      id: p.id,
      message: p.message ?? "",
      createdTime: p.created_time ?? new Date(0).toISOString(),
      permalinkUrl: p.permalink_url ?? null,
      shares: p.shares?.count ?? 0,
      reactions: p.reactions?.summary?.total_count ?? 0,
      comments: p.comments?.summary?.total_count ?? 0,
    }));
  },
};
