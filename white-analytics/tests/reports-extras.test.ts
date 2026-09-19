import { describe, expect, it } from "vitest";
import { computeDelta } from "@/lib/metrics";
import { buildBacklinkSection, buildPublishingSection, buildRankSection, rankChangeText, summarizePublishing, topRankRows, type PublishedTarget } from "@/features/reports/extras";
import { pdfLabels } from "@/features/reports/strings";

const L = pdfLabels.id;
const flat = { kind: "flat" as const, delta: 0 };

const kw = (keyword: string, position: number | null, volume: number | null, change: { kind: "up" | "down" | "flat" | "new" | "lost" | "none"; delta: number | null } = flat) => ({
  keyword,
  position,
  volume,
  change,
});

describe("topRankRows", () => {
  it("orders by position, then volume, and keeps unranked keywords last", () => {
    const rows = [kw("c", null, 9000), kw("a", 4, 100), kw("b", 4, 500), kw("d", 1, 10)];
    expect(topRankRows(rows).map((r) => r.keyword)).toEqual(["d", "b", "a", "c"]);
  });

  it("caps the list at n without mutating the input", () => {
    const rows = Array.from({ length: 14 }, (_, i) => kw(`k${i}`, 14 - i, 10));
    expect(topRankRows(rows)).toHaveLength(10);
    expect(topRankRows(rows)[0]!.keyword).toBe("k13");
    expect(rows[0]!.keyword).toBe("k0");
  });
});

describe("rankChangeText", () => {
  it("names the direction instead of printing a signed number", () => {
    expect(rankChangeText({ kind: "up", delta: 3 }, L)).toBe("Naik 3");
    expect(rankChangeText({ kind: "down", delta: -2 }, L)).toBe("Turun 2");
    expect(rankChangeText(flat, L)).toBe("Tetap");
    expect(rankChangeText({ kind: "new", delta: null }, L)).toBe("Baru masuk");
    expect(rankChangeText({ kind: "lost", delta: null }, L)).toBe("Keluar 100 besar");
    expect(rankChangeText({ kind: "none", delta: null }, L)).toBe("-");
  });
});

describe("buildRankSection", () => {
  const deltas = { visibility: computeDelta(12, 10), avgPosition: computeDelta(8, 10), top3: computeDelta(2, 1), top10: computeDelta(5, 5) };
  const base = { hasKeywords: true, latestDate: "2026-09-01", kpis: { visibility: 12, avgPosition: 8, top3: 2, top10: 5 }, deltas, rows: [] };

  it("is absent without tracked keywords or without any snapshot", () => {
    expect(buildRankSection({ ...base, hasKeywords: false }, true, L)).toBeNull();
    expect(buildRankSection({ ...base, latestDate: null }, true, L)).toBeNull();
  });

  it("drops deltas when compare is off and marks position as lower-is-better", () => {
    const on = buildRankSection(base, true, L)!;
    const off = buildRankSection(base, false, L)!;
    expect(on.kpis.map((k) => k.delta?.abs)).toEqual([2, -2, 1, 0]);
    expect(on.kpis[1]!.lowerIsBetter).toBe(true);
    expect(off.kpis.every((k) => k.delta === null)).toBe(true);
  });
});

describe("buildBacklinkSection", () => {
  it("is absent without a snapshot", () => {
    expect(buildBacklinkSection({ latest: null, dofollowShare: 0, weekly: [] }, L)).toBeNull();
  });

  it("sums new and lost backlinks across the weeks in the period", () => {
    const latest = { date: "2026-09-01", backlinks: 1200, referringDomains: 300, dofollow: 900, nofollow: 300, newBacklinks: 4, lostBacklinks: 1, domainRank: null, toxicShare: null };
    const weekly = [
      { date: "2026-08-24", baru: 7, hilang: 2 },
      { date: "2026-08-31", baru: 5, hilang: 1 },
    ];
    const section = buildBacklinkSection({ latest, dofollowShare: 75, weekly }, L)!;
    const value = (label: string) => section.kpis.find((k) => k.label === label)?.value;
    expect(value(L.backlinksNew)).toBe("12");
    expect(value(L.backlinksLost)).toBe("3");
    expect(value(L.domainRank)).toBe("-");
    expect(value(L.dofollowShare)).toBe("75,0%");
  });
});

describe("summarizePublishing", () => {
  const target = (postId: string, platform: string, externalId: string | null, day: number): PublishedTarget => ({
    postId,
    title: `Post ${postId}`,
    body: "",
    platform,
    socialAccountId: `acc-${platform}`,
    externalId,
    publishedAt: new Date(Date.UTC(2026, 7, day)),
  });
  const synced = (externalId: string, likes: number, comments: number, shares = 0, saves = 0) => ({ socialAccountId: "acc-INSTAGRAM", externalId, publishedAt: new Date(Date.UTC(2026, 7, 1)), likes, comments, shares, saves });
  const published = [target("p1", "INSTAGRAM", "ig1", 3), target("p1", "FACEBOOK", "fb1", 3), target("p2", "INSTAGRAM", "ig2", 9), target("p3", "INSTAGRAM", null, 12)];

  it("counts posts once and targets per platform", () => {
    const s = summarizePublishing(published, 2, []);
    expect(s.posts).toBe(3);
    expect(s.failed).toBe(2);
    expect(s.perPlatform).toEqual([
      { platform: "INSTAGRAM", count: 3 },
      { platform: "FACEBOOK", count: 1 },
    ]);
  });

  it("lists the most recent targets when no synced post matches", () => {
    const s = summarizePublishing(published, 0, [synced("other", 9, 9, 9, 9)]);
    expect(s.mode).toBe("recent");
    expect(s.top.map((t) => t.postId)).toEqual(["p3", "p2", "p1", "p1"]);
    expect(buildPublishingSection(s, L)!.table!.columns).toHaveLength(3);
  });

  it("ranks matched targets by engagement and requires the same account", () => {
    const s = summarizePublishing(published, 0, [
      synced("ig1", 10, 2, 1, 1),
      synced("ig2", 50, 5),
      synced("fb1", 999, 0),
    ]);
    expect(s.mode).toBe("engagement");
    expect(s.top.map((t) => [t.externalId, t.engagements])).toEqual([
      ["ig2", 55],
      ["ig1", 14],
    ]);
    const section = buildPublishingSection(s, L)!;
    expect(section.table!.title).toBe(L.publishedTop);
    expect(section.table!.rows[0]).toEqual(["Instagram", "Post p2", "9 Agu", "55"]);
  });

  it("produces no section when nothing was published", () => {
    expect(buildPublishingSection(summarizePublishing([], 3, []), L)).toBeNull();
  });
});
