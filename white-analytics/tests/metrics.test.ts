import { describe, it, expect } from "vitest";
import {
  computeDelta,
  isGoodChange,
  weightedMean,
  postEngagementRate,
  avgEngagementRate,
  followerGrowth,
  postingHeatmap,
  ctr,
  aggregateSearch,
  positionBucket,
  positionDistribution,
  expectedCtr,
  strikingDistance,
  lowCtr,
  declining,
  rateCwv,
  healthScore,
  aggregateAds,
  adCtr,
  costPerResult,
  cpm,
  frequency,
  fillDaily,
} from "@/lib/metrics";
import { rangeForPreset, previousRange, dayCount, resolveRange, utcDate, toISODate } from "@/lib/dates";
import { formatCompact, formatCurrency, formatDeltaPercent, formatNumber, formatPercent, slugify } from "@/lib/format";

describe("delta", () => {
  it("computes abs, pct and direction", () => {
    expect(computeDelta(120, 100)).toEqual({ abs: 20, pct: 20, direction: "up" });
    expect(computeDelta(80, 100)).toEqual({ abs: -20, pct: -20, direction: "down" });
    expect(computeDelta(100, 100).direction).toBe("flat");
  });
  it("returns null pct when previous is 0", () => {
    expect(computeDelta(50, 0).pct).toBeNull();
    expect(computeDelta(50, undefined).pct).toBeNull();
  });
  it("respects lower-is-better metrics", () => {
    const d = computeDelta(80, 100); // went down
    expect(isGoodChange(d)).toBe(false);
    expect(isGoodChange(d, true)).toBe(true);
    expect(isGoodChange(computeDelta(1, 1))).toBeNull();
  });
  it("weighted mean", () => {
    expect(weightedMean([[2, 100], [10, 100]])).toBe(6);
    expect(weightedMean([[2, 300], [10, 100]])).toBe(4);
    expect(weightedMean([])).toBe(0);
  });
});

describe("social metrics", () => {
  const post = { likes: 100, comments: 20, shares: 5, saves: 5, publishedAt: new Date("2026-08-01T10:00:00Z") };
  it("engagement rate by followers = (likes+comments+shares+saves)/followers*100", () => {
    expect(postEngagementRate(post, 10_000)).toBeCloseTo(1.3);
  });
  it("avg ER is mean of per-post ER; 0 for empty", () => {
    const p2 = { ...post, likes: 300 };
    expect(avgEngagementRate([post, p2], 10_000)).toBeCloseTo((1.3 + 3.3) / 2);
    expect(avgEngagementRate([], 10_000)).toBe(0);
    expect(avgEngagementRate([post], 0)).toBe(0);
  });
  it("follower growth", () => {
    expect(followerGrowth(1000, 1100)).toEqual({ abs: 100, pct: 10 });
    expect(followerGrowth(0, 100).pct).toBeNull();
  });
  it("heatmap has 168 cells and buckets by Jakarta time (Mon=0)", () => {
    // 2026-08-03 is a Monday; 02:00Z = 09:00 WIB
    const cells = postingHeatmap([{ ...post, publishedAt: new Date("2026-08-03T02:00:00Z") }]);
    expect(cells).toHaveLength(168);
    const hit = cells.find((c) => c.posts > 0)!;
    expect(hit.day).toBe(0);
    expect(hit.hour).toBe(9);
    expect(hit.avgEngagement).toBe(130);
  });
});

describe("seo metrics", () => {
  it("ctr and aggregation with impression-weighted position", () => {
    expect(ctr(50, 1000)).toBe(5);
    const agg = aggregateSearch([
      { clicks: 10, impressions: 100, position: 2 },
      { clicks: 10, impressions: 300, position: 10 },
    ]);
    expect(agg.clicks).toBe(20);
    expect(agg.impressions).toBe(400);
    expect(agg.ctr).toBe(5);
    expect(agg.position).toBe(8);
  });
  it("position buckets", () => {
    expect(positionBucket(1)).toBe("1-3");
    expect(positionBucket(3.4)).toBe("4-10");
    expect(positionBucket(10)).toBe("4-10");
    expect(positionBucket(15)).toBe("11-20");
    expect(positionBucket(21)).toBe("21+");
    const dist = positionDistribution([
      { clicks: 1, impressions: 10, position: 2 },
      { clicks: 1, impressions: 10, position: 30 },
    ]);
    expect(dist.map((d) => d.queries)).toEqual([1, 0, 0, 1]);
  });
  it("expected CTR curve", () => {
    expect(expectedCtr(1)).toBe(28);
    expect(expectedCtr(10)).toBe(2.5);
    expect(expectedCtr(15)).toBe(1.5);
    expect(expectedCtr(40)).toBe(1);
  });
  it("opportunities: striking distance, low ctr, declining", () => {
    const rows = [
      { key: "kopi susu", clicks: 20, impressions: 2000, position: 6, prevClicks: 25 },
      { key: "kopi hitam", clicks: 5, impressions: 5000, position: 2, prevClicks: 5 }, // low ctr (0.1% vs 15%)
      { key: "kopi murah", clicks: 10, impressions: 500, position: 4, prevClicks: 40 }, // declining 75%
      { key: "tiny", clicks: 0, impressions: 10, position: 5, prevClicks: 0 }, // below minImpressions
    ];
    expect(strikingDistance(rows).map((o) => o.key)).toEqual(["kopi susu", "kopi murah"]);
    expect(lowCtr(rows).map((o) => o.key)).toContain("kopi hitam");
    expect(declining(rows).map((o) => o.key)).toEqual(["kopi murah"]);
  });
  it("cwv rating and health score", () => {
    expect(rateCwv("lcpMs", 2000)).toBe("good");
    expect(rateCwv("lcpMs", 3000)).toBe("needs-improvement");
    expect(rateCwv("lcpMs", 5000)).toBe("poor");
    expect(rateCwv("cls", null)).toBeNull();
    expect(healthScore({ performance: 100, seo: 100, accessibility: 100, bestPractices: 100 }, { errors: 0, warnings: 0 })).toBe(100);
    expect(healthScore({ performance: 100, seo: 100, accessibility: 100, bestPractices: 100 }, { errors: 5, warnings: 10 })).toBe(85);
    expect(healthScore(null, { errors: 0, warnings: 0 })).toBeNull();
  });
});

describe("ads metrics", () => {
  it("formulas", () => {
    expect(costPerResult(1_000_000, 50)).toBe(20_000);
    expect(adCtr(50, 10_000)).toBe(0.5);
    expect(cpm(1_000_000, 100_000)).toBe(10_000);
    expect(frequency(30_000, 10_000)).toBe(3);
  });
  it("aggregate", () => {
    const k = aggregateAds([
      { spend: 100, impressions: 1000, reach: 800, clicks: 30, linkClicks: 20, results: 4 },
      { spend: 100, impressions: 1000, reach: 700, clicks: 30, linkClicks: 30, results: 6 },
    ]);
    expect(k.spend).toBe(200);
    expect(k.cpr).toBe(20);
    expect(k.ctr).toBe(2.5);
    expect(k.cpc).toBe(4);
    expect(k.roas).toBeNull();
  });
});

describe("dates", () => {
  const now = new Date("2026-08-19T05:00:00Z");
  it("preset 28d ends yesterday and spans 28 days", () => {
    const r = rangeForPreset("28d", now);
    expect(toISODate(r.to)).toBe("2026-08-18");
    expect(toISODate(r.from)).toBe("2026-07-22");
    expect(dayCount(r)).toBe(28);
  });
  it("previous range is contiguous and same length", () => {
    const r = rangeForPreset("7d", now);
    const p = previousRange(r);
    expect(dayCount(p)).toBe(7);
    expect(toISODate(p.to)).toBe("2026-08-11");
    expect(toISODate(p.from)).toBe("2026-08-05");
  });
  it("last month preset", () => {
    const r = rangeForPreset("lastMonth", now);
    expect(toISODate(r.from)).toBe("2026-07-01");
    expect(toISODate(r.to)).toBe("2026-07-31");
  });
  it("resolveRange prefers explicit dates, falls back to preset, default compare=true", () => {
    const a = resolveRange({ from: "2026-01-01", to: "2026-01-10" }, now);
    expect(dayCount(a.range)).toBe(10);
    expect(a.compare).toBe(true);
    const b = resolveRange({ preset: "7d", compare: "0" }, now);
    expect(dayCount(b.range)).toBe(7);
    expect(b.compare).toBe(false);
    const c = resolveRange({ from: "bad", to: "2026-01-10" }, now);
    expect(c.preset).toBe("28d");
  });
  it("fillDaily fills missing days with 0", () => {
    const range = { from: utcDate(2026, 0, 1), to: utcDate(2026, 0, 3) };
    const pts = fillDaily(range, [{ date: utcDate(2026, 0, 2), clicks: 5 }], (r) => r.date, ["clicks"]);
    expect(pts.map((p) => p.clicks)).toEqual([0, 5, 0]);
  });
});

describe("format (id-ID)", () => {
  it("numbers", () => {
    expect(formatNumber(1234567)).toBe("1.234.567");
    expect(formatCompact(1284)).toBe("1,3\u00a0rb");
    expect(formatCompact(1_284_000)).toBe("1,3\u00a0jt");
    expect(formatCompact(2_100_000_000)).toBe("2,1\u00a0M");
    expect(formatCompact(950)).toBe("950");
    expect(formatPercent(4.234)).toBe("4,23%");
    expect(formatDeltaPercent(12.34)).toBe("+12,3%");
    expect(formatDeltaPercent(-4)).toBe("−4,0%");
    expect(formatDeltaPercent(null)).toBe("–");
  });
  it("currency IDR without decimals", () => {
    expect(formatCurrency(1500000)).toMatch(/Rp\s?1\.500\.000/);
    expect(formatCurrency(1500000, "IDR", { compact: true })).toBe("Rp 1,5\u00a0jt");
  });
  it("slugify", () => {
    expect(slugify("Adiharjo Project!")).toBe("adiharjo-project");
    expect(slugify("  Kopi   Nusantara ")).toBe("kopi-nusantara");
  });
});
