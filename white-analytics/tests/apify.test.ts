import { describe, expect, it } from "vitest";
import {
  DEFAULT_ACTORS,
  actorPath,
  countryForLocation,
  domainMatches,
  normalizeFacebookPage,
  normalizeInstagramPost,
  normalizeInstagramProfile,
  normalizeSerpItems,
  normalizeTikTokItems,
  parseActorOverrides,
} from "@/lib/providers/apify/normalize";

describe("apify — config", () => {
  it("parses actor overrides tolerant of garbage", () => {
    expect(parseActorOverrides(undefined)).toEqual(DEFAULT_ACTORS);
    expect(parseActorOverrides("not json")).toEqual(DEFAULT_ACTORS);
    expect(parseActorOverrides('{"serp":"me/my-serp","bogus":"x/y","tiktok":"invalid actor"}')).toEqual({ ...DEFAULT_ACTORS, serp: "me/my-serp" });
    expect(actorPath("apify/google-search-scraper")).toBe("apify~google-search-scraper");
    expect(countryForLocation(2360)).toBe("id");
    expect(countryForLocation(999)).toBe("id");
    expect(domainMatches("shop.kopinusantara.id", "kopinusantara.id")).toBe(true);
    expect(domainMatches("notkopinusantara.id", "kopinusantara.id")).toBe(false);
  });
});

describe("apify — SERP normaliser", () => {
  it("maps Google Search Scraper items to RankResult in request order", () => {
    const items = [
      {
        searchQuery: { term: "kopi susu" },
        organicResults: [
          { url: "https://www.tomoro.id/menu", position: 1 },
          { url: "https://kopinusantara.id/kopi-susu", position: 2 },
          { url: "https://www.youtube.com/watch?v=1" },
        ],
        peopleAlsoAsk: [{ q: 1 }],
        paidResults: [],
      },
    ];
    const out = normalizeSerpItems(items, [{ keyword: "Kopi Susu", device: "MOBILE" }, { keyword: "missing", device: "MOBILE" }], "kopinusantara.id");
    expect(out[0]!.position).toBe(2);
    expect(out[0]!.url).toBe("https://kopinusantara.id/kopi-susu");
    expect(out[0]!.top10[0]).toEqual({ domain: "tomoro.id", position: 1, url: "https://www.tomoro.id/menu" });
    expect(out[0]!.top10[2]!.position).toBe(3);
    expect(out[0]!.serpFeatures).toEqual(["people_also_ask", "video"]);
    expect(out[1]!.position).toBeNull();
    expect(out[1]!.top10).toEqual([]);
  });
});

describe("apify — social normalisers", () => {
  it("instagram profile + post", () => {
    const p = normalizeInstagramProfile({ id: "1", username: "Kopi", fullName: "Kopi N", followersCount: "1,234", followsCount: 12, postsCount: 9 }, "kopi");
    expect(p).toMatchObject({ platform: "INSTAGRAM", username: "kopi", followers: 1234, following: 12, mediaCount: 9 });
    const post = normalizeInstagramPost({ id: "p1", shortCode: "abc", type: "Sidecar", caption: "hi", likesCount: 10, commentsCount: 2, timestamp: "2026-09-01T10:00:00.000Z", productType: "clips", videoPlayCount: 500 });
    expect(post).toMatchObject({ externalId: "p1", mediaType: "CAROUSEL_ALBUM", productType: "REELS", permalink: "https://www.instagram.com/p/abc/", likes: 10, views: 500 });
    expect(normalizeInstagramPost({})).toBeNull();
  });
  it("tiktok items → profile from authorMeta + video posts", () => {
    const { profile, posts } = normalizeTikTokItems(
      [
        { id: "v1", text: "halo", createTime: 1756720000, authorMeta: { id: "a1", name: "Kopi.ID", nickName: "Kopi", fans: 5000, following: 10, video: 40, heart: 90000 }, diggCount: 100, playCount: 4000, videoMeta: { coverUrl: "c.jpg" } },
        { id: "v2", text: "dua", createTimeISO: "2026-09-01T00:00:00Z", authorMeta: { name: "kopi.id" }, diggCount: 5 },
      ],
      "kopi.id",
    );
    expect(profile).toMatchObject({ platform: "TIKTOK", username: "kopi.id", followers: 5000, mediaCount: 40, totalLikes: 90000 });
    expect(posts).toHaveLength(2);
    expect(posts[0]).toMatchObject({ mediaType: "VIDEO", productType: "REELS", likes: 100, views: 4000, thumbnailUrl: "c.jpg" });
    expect(posts[0]!.publishedAt.startsWith("2025-09-01")).toBe(true); // 1756720000 s = 2025-09-01T09:46:40Z
    expect(normalizeTikTokItems([], "x").profile).toBeNull();
  });
  it("facebook page", () => {
    const p = normalizeFacebookPage({ pageId: "9", title: "Kopi Nusantara", followers: 800, likes: 750 }, "kopinusantaraid");
    expect(p).toMatchObject({ platform: "FACEBOOK", externalId: "9", displayName: "Kopi Nusantara", followers: 800, totalLikes: 750, following: null });
  });
});
