import { describe, it, expect } from "vitest";
import { aggregateDimRows, buildKeywordRows, countryLabel, mergeCurrentPrevious, normalizeKeywordView, urlPath, type KeywordOpportunities } from "@/features/seo/aggregate";
import { lowCtr, rising, strikingDistance, declining } from "@/lib/metrics";

describe("aggregateDimRows", () => {
  it("sums clicks/impressions and weights position by impressions", () => {
    const out = aggregateDimRows([
      { key: "kopi susu", clicks: 10, impressions: 100, position: 4 },
      { key: "kopi susu", clicks: 30, impressions: 300, position: 8 },
      { key: "kopi hitam", clicks: 5, impressions: 50, position: 2 },
    ]);
    const kopiSusu = out.find((r) => r.key === "kopi susu")!;
    expect(kopiSusu.clicks).toBe(40);
    expect(kopiSusu.impressions).toBe(400);
    expect(kopiSusu.position).toBeCloseTo((4 * 100 + 8 * 300) / 400); // 7
    expect(kopiSusu.ctr).toBeCloseTo(10);
    // sorted by clicks desc
    expect(out[0]!.key).toBe("kopi susu");
  });
});

describe("mergeCurrentPrevious", () => {
  it("attaches previous aggregates and null when absent", () => {
    const merged = mergeCurrentPrevious(
      aggregateDimRows([
        { key: "a", clicks: 10, impressions: 200, position: 5 },
        { key: "b", clicks: 4, impressions: 80, position: 12 },
      ]),
      aggregateDimRows([{ key: "a", clicks: 20, impressions: 250, position: 6 }]),
    );
    const a = merged.find((r) => r.key === "a")!;
    const b = merged.find((r) => r.key === "b")!;
    expect(a.prevClicks).toBe(20);
    expect(a.prevPosition).toBeCloseTo(6);
    expect(b.prevClicks).toBeNull();
    expect(b.prevPosition).toBeNull();
  });
});

describe("buildKeywordRows", () => {
  const rows = mergeCurrentPrevious(
    aggregateDimRows([
      { key: "peluang", clicks: 10, impressions: 1000, position: 6 }, // page 1, ctr 1% << expected 5%
      { key: "juara", clicks: 300, impressions: 1000, position: 1.2 },
      { key: "naik daun", clicks: 100, impressions: 500, position: 9 },
    ]),
    aggregateDimRows([
      { key: "peluang", clicks: 30, impressions: 900, position: 7 },
      { key: "juara", clicks: 280, impressions: 950, position: 1.4 },
      { key: "naik daun", clicks: 50, impressions: 300, position: 11 },
    ]),
  );
  const opportunities: KeywordOpportunities = {
    striking: strikingDistance(rows),
    low_ctr: lowCtr(rows),
    declining: declining(rows),
    rising: rising(rows),
  };

  it("returns every query for the all view with buckets", () => {
    const all = buildKeywordRows(rows, opportunities, "all");
    expect(all).toHaveLength(3);
    expect(all.find((r) => r.key === "juara")!.bucket).toBe("1-3");
    expect(all.find((r) => r.key === "peluang")!.bucket).toBe("4-10");
    expect(all.every((r) => r.expectedCtr != null)).toBe(true);
  });

  it("builds opportunity rows with potential clicks and prev data", () => {
    const low = buildKeywordRows(rows, opportunities, "low_ctr");
    expect(low.map((r) => r.key)).toContain("peluang");
    const p = low.find((r) => r.key === "peluang")!;
    expect(p.potentialClicks).toBeGreaterThan(0);
    expect(p.prevClicks).toBe(30);
    expect(p.prevPosition).toBeCloseTo(7);

    const risingRows = buildKeywordRows(rows, opportunities, "rising");
    expect(risingRows.map((r) => r.key)).toContain("naik daun");
    const decliningRows = buildKeywordRows(rows, opportunities, "declining");
    expect(decliningRows.map((r) => r.key)).toContain("peluang");
  });
});

describe("view + display helpers", () => {
  it("normalizes the ?view= param", () => {
    expect(normalizeKeywordView("striking")).toBe("striking");
    expect(normalizeKeywordView("low_ctr")).toBe("low_ctr");
    expect(normalizeKeywordView(undefined)).toBe("all");
    expect(normalizeKeywordView("hacky")).toBe("all");
  });

  it("shortens URLs to paths and keeps invalid input as-is", () => {
    expect(urlPath("https://kopinusantara.id/menu")).toBe("/menu");
    expect(urlPath("https://kopinusantara.id/")).toBe("/");
    expect(urlPath("https://kopinusantara.id/promo?ref=ig")).toBe("/promo?ref=ig");
    expect(urlPath("bukan-url")).toBe("bukan-url");
  });

  it("labels GSC country codes in Indonesian", () => {
    expect(countryLabel("idn")).toBe("Indonesia");
    expect(countryLabel("usa")).toBe("Amerika Serikat");
    expect(countryLabel("zzz")).toBe("ZZZ");
  });
});
