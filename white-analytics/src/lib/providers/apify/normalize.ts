/**
 * Pure normalisers for Apify actor output. Actor schemas drift between
 * versions, so every reader is tolerant (several candidate field names,
 * numbers coerced, missing → sensible default). Tested in tests/apify.test.ts.
 */
import type { RankResult, SerpRankRequest, SerpTopResult } from "@/lib/providers/dataforseo/types";
import type { ApifyActors, PublicPlatform, PublicPost, PublicProfile } from "./types";

export const DEFAULT_ACTORS: ApifyActors = {
  serp: "apify/google-search-scraper",
  instagramProfile: "apify/instagram-profile-scraper",
  instagramPosts: "apify/instagram-scraper",
  tiktok: "clockworks/tiktok-scraper",
  facebookPage: "apify/facebook-pages-scraper",
  facebookPosts: "apify/facebook-posts-scraper",
};

/** `{"serp":"owner/name"}` (or invalid JSON) → merged actor map; unknown keys ignored. */
export function parseActorOverrides(raw: string | null | undefined): ApifyActors {
  if (!raw) return DEFAULT_ACTORS;
  try {
    const o = JSON.parse(raw) as Record<string, unknown>;
    const out = { ...DEFAULT_ACTORS };
    for (const k of Object.keys(DEFAULT_ACTORS) as (keyof ApifyActors)[]) {
      const v = o[k];
      if (typeof v === "string" && /^[\w.-]+\/[\w.-]+$/.test(v)) out[k] = v;
    }
    return out;
  } catch {
    return DEFAULT_ACTORS;
  }
}

/** `owner/name` → `owner~name` (Apify REST path form). */
export function actorPath(id: string): string {
  return id.replace("/", "~");
}

/** DataForSEO location codes we use → ISO country for Google. Unknown → "id". */
const LOCATION_COUNTRY: Record<number, string> = { 2360: "id", 2840: "us", 2702: "sg", 2458: "my", 2826: "gb", 2036: "au", 2392: "jp", 2356: "in" };
export function countryForLocation(locationCode: number): string {
  return LOCATION_COUNTRY[locationCode] ?? "id";
}

type Rec = Record<string, unknown>;
const rec = (v: unknown): Rec => (v && typeof v === "object" ? (v as Rec) : {});
const str = (v: unknown): string | null => (typeof v === "string" && v.length > 0 ? v : typeof v === "number" ? String(v) : null);
const num = (v: unknown): number => {
  if (typeof v === "number" && Number.isFinite(v)) return v;
  if (typeof v === "string") {
    const n = Number(v.replace(/[^\d.-]/g, ""));
    return Number.isFinite(n) ? n : 0;
  }
  return 0;
};
const first = (o: Rec, keys: string[]): unknown => {
  for (const k of keys) if (o[k] !== undefined && o[k] !== null && o[k] !== "") return o[k];
  return undefined;
};
const iso = (v: unknown): string => {
  if (typeof v === "number") return new Date(v < 1e12 ? v * 1000 : v).toISOString();
  if (typeof v === "string") {
    const d = new Date(v);
    if (!Number.isNaN(d.getTime())) return d.toISOString();
  }
  return new Date(0).toISOString();
};

export function hostOf(url: string): string {
  try {
    return new URL(url).hostname.replace(/^www\./, "").toLowerCase();
  } catch {
    return "";
  }
}

export function domainMatches(host: string, target: string): boolean {
  const t = target.replace(/^www\./, "").toLowerCase();
  return host === t || host.endsWith(`.${t}`);
}

// ── SERP ──────────────────────────────────────────────────────

/**
 * Google Search Scraper items (one per query page) → RankResult per request,
 * in request order. A query with no item → position null, empty top10.
 */
export function normalizeSerpItems(items: unknown[], requests: SerpRankRequest[], targetDomain: string): RankResult[] {
  const byTerm = new Map<string, Rec>();
  for (const raw of items) {
    const it = rec(raw);
    const term = str(rec(it.searchQuery).term) ?? str(it.query) ?? str(it.term);
    if (term && !byTerm.has(term.toLowerCase())) byTerm.set(term.toLowerCase(), it);
  }
  return requests.map((req) => {
    const it = byTerm.get(req.keyword.toLowerCase());
    if (!it) return { keyword: req.keyword, device: req.device, position: null, url: null, serpFeatures: [], top10: [] };
    const organic = (Array.isArray(it.organicResults) ? it.organicResults : []).map((o, i) => {
      const r = rec(o);
      const url = str(r.url) ?? "";
      return { url, host: hostOf(url), position: num(r.position) || i + 1 };
    });
    const hit = organic.find((o) => domainMatches(o.host, targetDomain));
    const top10: SerpTopResult[] = organic.slice(0, 10).map((o) => ({ domain: o.host, position: o.position, url: o.url }));
    const features: string[] = [];
    if (Array.isArray(it.paidResults) && it.paidResults.length) features.push("ads");
    if (Array.isArray(it.peopleAlsoAsk) && it.peopleAlsoAsk.length) features.push("people_also_ask");
    if (Array.isArray(it.relatedQueries) && it.relatedQueries.length) features.push("related_searches");
    if (it.aiOverview) features.push("ai_overview");
    if (Array.isArray(it.paidProducts) && it.paidProducts.length) features.push("shopping");
    if (organic.some((o) => o.host === "youtube.com")) features.push("video");
    return { keyword: req.keyword, device: req.device, position: hit ? hit.position : null, url: hit ? hit.url : null, serpFeatures: features, top10 };
  });
}

// ── Instagram ─────────────────────────────────────────────────

function igMediaType(t: unknown): PublicPost["mediaType"] {
  const v = String(t ?? "").toLowerCase();
  if (v === "video" || v === "reel") return "VIDEO";
  if (v === "sidecar" || v === "carousel" || v === "carousel_album") return "CAROUSEL_ALBUM";
  return "IMAGE";
}

export function normalizeInstagramPost(raw: unknown): PublicPost | null {
  const p = rec(raw);
  const id = str(first(p, ["id", "shortCode", "code"]));
  if (!id) return null;
  const shortCode = str(p.shortCode);
  const productType = String(p.productType ?? "").toLowerCase();
  return {
    externalId: id,
    caption: str(p.caption) ?? "",
    mediaType: igMediaType(p.type),
    productType: productType === "clips" || productType === "reels" ? "REELS" : "FEED",
    permalink: str(p.url) ?? (shortCode ? `https://www.instagram.com/p/${shortCode}/` : null),
    mediaUrl: str(first(p, ["videoUrl", "displayUrl"])),
    thumbnailUrl: str(first(p, ["displayUrl", "thumbnailUrl"])),
    publishedAt: iso(first(p, ["timestamp", "takenAt", "taken_at"])),
    likes: num(first(p, ["likesCount", "likes"])),
    comments: num(first(p, ["commentsCount", "comments"])),
    shares: num(first(p, ["sharesCount", "reshareCount"])),
    saves: num(first(p, ["savesCount"])),
    views: num(first(p, ["videoPlayCount", "videoViewCount", "playCount"])),
  };
}

export function normalizeInstagramProfile(raw: unknown, fallbackUsername: string): PublicProfile {
  const p = rec(raw);
  return {
    platform: "INSTAGRAM",
    externalId: str(first(p, ["id", "pk"])) ?? `ig_${fallbackUsername}`,
    username: (str(p.username) ?? fallbackUsername).toLowerCase(),
    displayName: str(first(p, ["fullName", "full_name"])) ?? fallbackUsername,
    biography: str(p.biography),
    avatarUrl: str(first(p, ["profilePicUrlHD", "profilePicUrl", "profile_pic_url"])),
    followers: num(first(p, ["followersCount", "followers"])),
    following: num(first(p, ["followsCount", "following"])) || null,
    mediaCount: num(first(p, ["postsCount", "mediaCount"])),
  };
}

// ── TikTok ────────────────────────────────────────────────────

export function normalizeTikTokItems(items: unknown[], fallbackUsername: string): { profile: PublicProfile | null; posts: PublicPost[] } {
  let profile: PublicProfile | null = null;
  const posts: PublicPost[] = [];
  for (const raw of items) {
    const it = rec(raw);
    const a = rec(first(it, ["authorMeta", "author"]));
    if (!profile && Object.keys(a).length) {
      profile = {
        platform: "TIKTOK",
        externalId: str(a.id) ?? `tt_${fallbackUsername}`,
        username: (str(first(a, ["name", "uniqueId"])) ?? fallbackUsername).toLowerCase(),
        displayName: str(first(a, ["nickName", "nickname"])) ?? fallbackUsername,
        biography: str(a.signature),
        avatarUrl: str(first(a, ["avatar", "avatarLarger", "avatarThumb"])),
        followers: num(first(a, ["fans", "followerCount"])),
        following: num(first(a, ["following", "followingCount"])) || null,
        mediaCount: num(first(a, ["video", "videoCount"])),
        totalLikes: num(first(a, ["heart", "heartCount"])) || null,
      };
    }
    const id = str(it.id);
    if (!id) continue;
    const video = rec(it.videoMeta);
    posts.push({
      externalId: id,
      caption: str(first(it, ["text", "desc"])) ?? "",
      mediaType: "VIDEO",
      productType: "REELS",
      permalink: str(first(it, ["webVideoUrl", "url"])),
      mediaUrl: str(first(video, ["downloadAddr", "playAddr"])),
      thumbnailUrl: str(first(video, ["coverUrl", "originalCoverUrl"])) ?? str(first(it, ["cover", "coverUrl"])),
      publishedAt: iso(first(it, ["createTimeISO", "createTime"])),
      likes: num(first(it, ["diggCount", "likes"])),
      comments: num(first(it, ["commentCount", "comments"])),
      shares: num(first(it, ["shareCount", "shares"])),
      saves: num(first(it, ["collectCount", "saves"])),
      views: num(first(it, ["playCount", "views"])),
    });
  }
  return { profile, posts };
}

// ── Facebook ──────────────────────────────────────────────────

export function normalizeFacebookPage(raw: unknown, fallbackUsername: string): PublicProfile {
  const p = rec(raw);
  return {
    platform: "FACEBOOK",
    externalId: str(first(p, ["pageId", "facebookId", "id"])) ?? `fb_${fallbackUsername}`,
    username: (str(first(p, ["pageName", "username"])) ?? fallbackUsername).toLowerCase(),
    displayName: str(first(p, ["title", "name", "pageName"])) ?? fallbackUsername,
    biography: str(first(p, ["intro", "about", "description"])),
    avatarUrl: str(first(p, ["profilePictureUrl", "profilePhoto", "pageLogo"])),
    followers: num(first(p, ["followers", "followersCount"])),
    following: null,
    mediaCount: num(first(p, ["postsCount"])),
    totalLikes: num(first(p, ["likes", "likesCount"])) || null,
  };
}

export function normalizeFacebookPost(raw: unknown): PublicPost | null {
  const p = rec(raw);
  const id = str(first(p, ["postId", "id", "legacyId"]));
  if (!id) return null;
  const media = Array.isArray(p.media) ? rec(p.media[0]) : {};
  const hasVideo = Array.isArray(p.media) && p.media.some((m) => /video/i.test(String(rec(m).__typename ?? rec(m).type ?? "")));
  return {
    externalId: id,
    caption: str(first(p, ["text", "message"])) ?? "",
    mediaType: hasVideo ? "VIDEO" : Array.isArray(p.media) && p.media.length > 1 ? "CAROUSEL_ALBUM" : "IMAGE",
    productType: "FEED",
    permalink: str(first(p, ["url", "postUrl", "topLevelUrl"])),
    mediaUrl: str(first(media, ["url", "photo_image", "image"])),
    thumbnailUrl: str(first(media, ["thumbnail", "photo_image", "image"])),
    publishedAt: iso(first(p, ["time", "timestamp", "publishedAt"])),
    likes: num(first(p, ["likes", "likesCount", "reactionsCount"])),
    comments: num(first(p, ["comments", "commentsCount"])),
    shares: num(first(p, ["shares", "sharesCount"])),
    saves: 0,
    views: num(first(p, ["viewsCount", "videoViewCount"])),
  };
}

export function profileUrl(platform: PublicPlatform, username: string): string {
  const u = username.replace(/^@/, "");
  if (platform === "INSTAGRAM") return `https://www.instagram.com/${u}/`;
  if (platform === "TIKTOK") return `https://www.tiktok.com/@${u}`;
  return `https://www.facebook.com/${u}`;
}
