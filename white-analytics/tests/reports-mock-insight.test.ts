import { describe, expect, it } from "vitest";
import { generateMockInsight } from "@/lib/providers/openrouter/mock";
import type { GenerateInsightInput, InsightPayload } from "@/lib/providers/openrouter/types";

const range = { from: "2026-07-22", to: "2026-08-18", days: 28, prevFrom: "2026-06-24", prevTo: "2026-07-21" };

const payload: InsightPayload = {
  social: {
    followers: 125_400,
    followerGrowth: 2_600,
    followerGrowthPct: 2.1,
    engagementRate: 4.32,
    engagementRateDeltaPct: -3.5,
    posts: 24,
    reach: 890_000,
    reachDeltaPct: 5.2,
    topPosts: [
      { platform: "INSTAGRAM", caption: "Kopi single origin Gayo tiba di toko", type: "Reels", engagements: 3_590, likes: 3_210, comments: 148 },
    ],
  },
  seo: {
    clicks: 8_412,
    clicksDeltaPct: 12.3,
    impressions: 402_000,
    impressionsDeltaPct: 8.8,
    ctr: 2.09,
    position: 9.4,
    positionDeltaAbs: -0.6,
    health: 82,
    topQueries: [
      { query: "kopi gayo asli", clicks: 812, impressions: 22_400, ctr: 3.63, position: 2.4 },
      { query: "beli kopi arabica online", clicks: 402, impressions: 31_000, ctr: 1.3, position: 6.8 },
    ],
  },
  ads: {
    currency: "IDR",
    spend: 45_200_000,
    spendDeltaPct: 10.0,
    results: 1_204,
    resultsDeltaPct: 18.2,
    cpr: 37_542,
    cprDeltaPct: -8.0,
    ctr: 1.84,
    cpm: 21_400,
    roas: 3.4,
    campaigns: [
      { name: "Promo Kemerdekaan", objective: "OUTCOME_SALES", spend: 15_100_000, results: 480, resultType: "purchase", cpr: 31_458 },
      { name: "Always-on Traffic", objective: "OUTCOME_TRAFFIC", spend: 12_400_000, results: 402, resultType: "link_click", cpr: 30_845 },
    ],
  },
};

function input(module: GenerateInsightInput["module"], language: GenerateInsightInput["language"] = "id"): GenerateInsightInput {
  return { module, language, clientName: "Kopi Nusantara", range, data: payload };
}

describe("mock insight provider", () => {
  it("produces the mandatory sections in order", () => {
    const md = generateMockInsight(input("OVERVIEW"));
    const sections = ["## Ringkasan eksekutif", "## Sorotan performa", "## Area perbaikan", "## Rekomendasi 30 hari"];
    let last = -1;
    for (const s of sections) {
      const idx = md.indexOf(s);
      expect(idx, `section ${s} present`).toBeGreaterThan(last);
      last = idx;
    }
  });

  it("has exactly 5 numbered recommendations", () => {
    const md = generateMockInsight(input("OVERVIEW"));
    const recBlock = md.split("## Rekomendasi 30 hari")[1]!;
    const items = recBlock.split("\n").filter((l) => /^\d+\.\s/.test(l.trim()));
    expect(items).toHaveLength(5);
    expect(items[0]!.startsWith("1.")).toBe(true);
    expect(items[4]!.startsWith("5.")).toBe(true);
  });

  it("fills real numbers from the payload", () => {
    const md = generateMockInsight(input("OVERVIEW"));
    expect(md).toContain("125,4\u00a0rb"); // followers (formatCompact)
    expect(md).toContain("4,32%"); // engagement rate
    expect(md).toContain("8,4\u00a0rb"); // seo clicks
    expect(md).toContain("Rp 45,2\u00a0jt"); // ad spend
    expect(md).toContain("Kopi Nusantara"); // client name
    expect(md).toContain("28 hari"); // period length
    expect(md).toContain("Promo Kemerdekaan"); // best campaign by CPR... appears in highlights
  });

  it("adds Peluang kata kunci only for SEO and Alokasi anggaran only for ADS", () => {
    const seoMd = generateMockInsight(input("SEO"));
    expect(seoMd).toContain("## Peluang kata kunci");
    expect(seoMd).not.toContain("## Alokasi anggaran");
    expect(seoMd).toContain("beli kopi arabica online"); // opportunity query (position > 3)

    const adsMd = generateMockInsight(input("ADS"));
    expect(adsMd).toContain("## Alokasi anggaran");
    expect(adsMd).not.toContain("## Peluang kata kunci");
    expect(adsMd).toContain("Always-on Traffic");

    const overviewMd = generateMockInsight(input("OVERVIEW"));
    expect(overviewMd).not.toContain("## Peluang kata kunci");
    expect(overviewMd).not.toContain("## Alokasi anggaran");
  });

  it("is deterministic and never mentions AI", () => {
    const a = generateMockInsight(input("OVERVIEW"));
    const b = generateMockInsight(input("OVERVIEW"));
    expect(a).toBe(b);
    expect(a.toLowerCase()).not.toMatch(/\bai\b|kecerdasan buatan|model bahasa|language model/);
  });

  it("keeps Indonesian headings for English output", () => {
    const md = generateMockInsight(input("SEO", "en"));
    expect(md).toContain("## Ringkasan eksekutif");
    expect(md).toContain("## Peluang kata kunci");
    expect(md).toContain("Over the last 28 days");
  });
});
