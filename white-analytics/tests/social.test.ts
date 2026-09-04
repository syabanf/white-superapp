import { describe, it, expect } from "vitest";
import {
  dailyFollowerSeries,
  erByContentType,
  followersAt,
  indexTo100,
  normalizeIgUsername,
  parsePlatform,
} from "@/features/social/lib";
import { buildFollowerSeries, hashSeed, mockMetaGraph, mulberry32 } from "@/lib/providers/meta-graph/mock";
import { mapMetaError } from "@/lib/providers/meta-graph/real";
import { utcDate } from "@/lib/dates";

describe("parsePlatform", () => {
  it("defaults to INSTAGRAM and accepts valid values case-insensitively", () => {
    expect(parsePlatform(undefined)).toBe("INSTAGRAM");
    expect(parsePlatform("tiktok")).toBe("TIKTOK");
    expect(parsePlatform("FACEBOOK")).toBe("FACEBOOK");
    expect(parsePlatform(["FACEBOOK", "TIKTOK"])).toBe("FACEBOOK");
    expect(parsePlatform("twitter")).toBe("INSTAGRAM");
  });
});

describe("normalizeIgUsername", () => {
  it("strips @, trims, lowercases", () => {
    expect(normalizeIgUsername(" @Kopi.Kenangan ")).toBe("kopi.kenangan");
    expect(normalizeIgUsername("@@ngopi_dulu")).toBe("ngopi_dulu");
  });
  it("rejects invalid usernames", () => {
    expect(normalizeIgUsername("a")).toBeNull(); // too short
    expect(normalizeIgUsername("x".repeat(31))).toBeNull(); // too long
    expect(normalizeIgUsername("has space")).toBeNull();
    expect(normalizeIgUsername("semi;colon")).toBeNull();
    expect(normalizeIgUsername(".leadingdot")).toBeNull();
    expect(normalizeIgUsername("trailingdot.")).toBeNull();
  });
});

describe("followersAt / dailyFollowerSeries", () => {
  const snaps = [
    { date: utcDate(2026, 7, 1), followers: 100, reach: 10, impressions: 20 },
    { date: utcDate(2026, 7, 2), followers: 110, reach: 12, impressions: 24 },
    { date: utcDate(2026, 7, 4), followers: 130, reach: 15, impressions: 30 },
  ];
  it("returns last snapshot on/before the date (0 when none)", () => {
    expect(followersAt(snaps, utcDate(2026, 7, 3))).toBe(110);
    expect(followersAt(snaps, utcDate(2026, 7, 4))).toBe(130);
    expect(followersAt(snaps, utcDate(2026, 6, 30))).toBe(0);
  });
  it("forward-fills followers on missing days, keeps reach/impressions per-day", () => {
    const series = dailyFollowerSeries({ from: utcDate(2026, 7, 2), to: utcDate(2026, 7, 4) }, snaps);
    expect(series.map((p) => p.followers)).toEqual([110, 110, 130]); // Aug 3 forward-filled
    expect(series.map((p) => p.reach)).toEqual([12, 0, 15]);
    expect(series[0]!.date).toBe("2026-08-02");
  });
  it("carries the last pre-range snapshot into the range", () => {
    const series = dailyFollowerSeries({ from: utcDate(2026, 7, 3), to: utcDate(2026, 7, 3) }, snaps);
    expect(series[0]!.followers).toBe(110);
  });
});

describe("indexTo100", () => {
  it("indexes to 100 at the first positive value", () => {
    const out = indexTo100([50, 55, 60]);
    expect(out[0]).toBe(100);
    expect(out[1]).toBeCloseTo(110);
    expect(out[2]).toBeCloseTo(120);
  });
  it("nulls out empty/zero values and handles null-leading series", () => {
    const out = indexTo100([null, 200, 220]);
    expect(out[0]).toBeNull();
    expect(out[1]).toBe(100);
    expect(out[2]).toBeCloseTo(110);
    expect(indexTo100([0, 50, 100])).toEqual([null, 100, 200]);
    expect(indexTo100([null, null])).toEqual([null, null]);
  });
});

describe("erByContentType", () => {
  const base = { publishedAt: new Date("2026-08-01T10:00:00Z"), shares: 0, saves: 0 };
  it("groups by content type and sorts by avg ER desc", () => {
    const posts = [
      { ...base, likes: 300, comments: 0, mediaType: "VIDEO", productType: "REELS" }, // Reels ER 3%
      { ...base, likes: 100, comments: 0, mediaType: "IMAGE", productType: "FEED" }, // Foto ER 1%
      { ...base, likes: 200, comments: 0, mediaType: "IMAGE", productType: "FEED" }, // Foto ER 2%
    ];
    const out = erByContentType(posts, 10_000);
    expect(out[0]).toEqual({ type: "Reels", avgEr: 3, count: 1 });
    expect(out[1]!.type).toBe("Foto");
    expect(out[1]!.avgEr).toBeCloseTo(1.5);
    expect(out[1]!.count).toBe(2);
  });
  it("returns empty array for no posts", () => {
    expect(erByContentType([], 1000)).toEqual([]);
  });
});

describe("meta-graph mock determinism", () => {
  it("hashSeed & mulberry32 are stable", () => {
    expect(hashSeed("kopi.senja")).toBe(hashSeed("kopi.senja"));
    const a = mulberry32(hashSeed("x"));
    const b = mulberry32(hashSeed("x"));
    expect([a(), a(), a()]).toEqual([b(), b(), b()]);
  });

  it("buildFollowerSeries has the right length, ends exactly at endFollowers, stays non-negative", () => {
    const s1 = buildFollowerSeries("bd:kopi.senja", 120, 186_000);
    const s2 = buildFollowerSeries("bd:kopi.senja", 120, 186_000);
    expect(s1).toEqual(s2); // deterministic
    expect(s1).toHaveLength(120);
    expect(s1[s1.length - 1]).toBe(186_000);
    expect(s1.every((v) => v >= 0)).toBe(true);
    expect(s1[0]!).toBeLessThan(186_000); // grows overall
  });

  it("discoverBusiness returns identical plausible profiles for the same username", async () => {
    const a = await mockMetaGraph.discoverBusiness("", "", "@Kopi.Senja");
    const b = await mockMetaGraph.discoverBusiness("", "", "kopi.senja");
    expect(a.username).toBe("kopi.senja");
    expect(a.followersCount).toBe(b.followersCount);
    expect(a.followersCount).toBeGreaterThanOrEqual(8_000);
    expect(a.media).toHaveLength(25);
    expect(a.media.map((m) => m.id)).toEqual(b.media.map((m) => m.id));
    const first = a.media[0]!;
    expect(first.likeCount).toBeGreaterThan(0);
    expect(["IMAGE", "VIDEO", "CAROUSEL_ALBUM"]).toContain(first.mediaType);
  });
});

describe("meta-graph error mapping", () => {
  it("maps Meta error codes to ProviderError codes", () => {
    expect(mapMetaError(400, { error: { code: 190, message: "expired" } }).code).toBe("TOKEN_EXPIRED");
    expect(mapMetaError(400, { error: { code: 4 } }).code).toBe("RATE_LIMIT");
    expect(mapMetaError(400, { error: { code: 17 } }).code).toBe("RATE_LIMIT");
    expect(mapMetaError(400, { error: { code: 32 } }).code).toBe("RATE_LIMIT");
    expect(mapMetaError(400, { error: { code: 613 } }).code).toBe("RATE_LIMIT");
    expect(mapMetaError(400, { error: { code: 10 } }).code).toBe("PERMISSION");
    expect(mapMetaError(400, { error: { code: 200 } }).code).toBe("PERMISSION");
    expect(mapMetaError(400, { error: { code: 100 } }).code).toBe("NOT_FOUND");
    expect(mapMetaError(500, null).code).toBe("UNKNOWN");
    expect(mapMetaError(400, { error: { code: 190 } }).requiresReconnect).toBe(true);
  });
});
