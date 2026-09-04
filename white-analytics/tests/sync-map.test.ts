import { describe, expect, it } from "vitest";
import { GA4_METRICS, ga4Date, mapGa4Daily, mapGscDaily, mapGscDimension } from "@/features/seo/sync-map";
import { aggregateDailyInsights, aggregateDemographics } from "@/features/ads/sync-map";

describe("GSC mappers", () => {
  it("maps daily rows and drops malformed dates", () => {
    const out = mapGscDaily([
      { keys: ["2026-09-01"], clicks: 10.4, impressions: 200, ctr: 5.2, position: 12.345 },
      { keys: ["bad"], clicks: 1, impressions: 1, ctr: 1, position: 1 },
    ]);
    expect(out).toEqual([{ date: "2026-09-01", clicks: 10, impressions: 200, ctr: 5.2, position: 12.35 }]);
  });
  it("normalises dimension keys like the seed (country lower, device upper, query lower)", () => {
    const rows = [{ keys: ["2026-09-01", "IDN"], clicks: 1, impressions: 2, ctr: 50, position: 3 }];
    expect(mapGscDimension(rows, "country")[0]).toMatchObject({ dimension: "COUNTRY", key: "idn" });
    expect(mapGscDimension([{ keys: ["2026-09-01", "mobile"], clicks: 1, impressions: 2, ctr: 50, position: 3 }], "device")[0]).toMatchObject({ dimension: "DEVICE", key: "MOBILE" });
    expect(mapGscDimension([{ keys: ["2026-09-01", "Kopi Susu"], clicks: 1, impressions: 2, ctr: 50, position: 3 }], "query")[0]).toMatchObject({ dimension: "QUERY", key: "kopi susu" });
    expect(mapGscDimension([{ keys: ["2026-09-01", ""], clicks: 1, impressions: 2, ctr: 50, position: 3 }], "page")).toEqual([]);
  });
});

describe("GA4 mapper", () => {
  it("parses compact dates and aggregates channels into one row per day", () => {
    expect(ga4Date("20260901")).toBe("2026-09-01");
    expect(ga4Date("2026-09-01")).toBe("2026-09-01");
    expect(ga4Date("nope")).toBeNull();
    expect(GA4_METRICS).toHaveLength(5);
    const out = mapGa4Daily([
      { dimensions: ["20260901", "Organic Search"], metrics: [100, 60, 90, 5, 12_000] },
      { dimensions: ["20260901", "Direct"], metrics: [50, 20, 40, 1, 3_000] },
      { dimensions: ["20260902", "Organic Search"], metrics: [10, 5, 9, 0, 0] },
    ]);
    expect(out).toHaveLength(2);
    expect(out[0]).toEqual({ date: "2026-09-01", sessions: 150, organicSessions: 100, users: 130, engagedSessions: 80, conversions: 6, avgEngagementSec: 100 });
    expect(out[1]!.avgEngagementSec).toBe(0);
  });
});

describe("Meta Ads mappers", () => {
  const row = (o: Partial<Parameters<typeof aggregateDailyInsights>[0][number]>) => ({
    campaignId: "c1", adSetId: "s1", adId: "a1", date: "2026-09-01", spend: 10, impressions: 100, reach: 80, clicks: 5, linkClicks: 4, frequency: 1.2, results: 2, resultType: "link_click", purchaseValue: null, ...o,
  });
  it("aggregates ad-level rows per (date, campaign, adset, ad) and sums purchase value", () => {
    const out = aggregateDailyInsights([row({}), row({ spend: 5, purchaseValue: 100 }), row({ adId: "a2" }), row({ adId: "" })]);
    expect(out).toHaveLength(2);
    const a1 = out.find((x) => x.adExternalId === "a1")!;
    expect(a1.spend).toBe(15);
    expect(a1.purchaseValue).toBe(100);
    expect(a1.resultType).toBe("link_click");
  });
  it("aggregates demographics per campaign/age/gender with unknown fallbacks", () => {
    const out = aggregateDemographics([row({ age: "25-34", gender: "Female" }), row({ age: "25-34", gender: "female", spend: 1 }), row({})]);
    expect(out).toHaveLength(2);
    expect(out[0]).toMatchObject({ campaignExternalId: "c1", age: "25-34", gender: "female", spend: 11 });
    expect(out[1]).toMatchObject({ age: "unknown", gender: "unknown" });
  });
});
