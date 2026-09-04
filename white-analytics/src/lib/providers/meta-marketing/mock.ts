/**
 * Adapter mock Meta Marketing — data deterministik (seeded PRNG per actId)
 * agar UI/dev konsisten tanpa kredensial. Tidak menyentuh jaringan.
 */
import type { InsightsQuery, MetaAd, MetaAdSet, MetaCampaign, MetaInsightRow, MetaMarketingAdapter } from "./types";

/** mulberry32 — PRNG deterministik kecil. */
function mulberry32(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a |= 0;
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

function hashString(s: string): number {
  let h = 2166136261;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return h >>> 0;
}

const BLUEPRINT = [
  {
    key: "traffic",
    name: "[Prospecting] Traffic – Demo",
    objective: "OUTCOME_TRAFFIC",
    status: "ACTIVE",
    dailyBudget: 300_000,
    resultType: "link_click",
    cpr: 1_800,
    ctr: 1.8,
    adSets: [
      { name: "Broad 18-34", ads: ["Video 15s", "Carousel Produk"] },
      { name: "Lookalike 1%", ads: ["Reels UGC", "Static Promo"] },
    ],
  },
  {
    key: "sales",
    name: "[Retargeting] Sales – Demo",
    objective: "OUTCOME_SALES",
    status: "ACTIVE",
    dailyBudget: 200_000,
    resultType: "purchase",
    cpr: 42_000,
    ctr: 2.4,
    adSets: [{ name: "Website Visitors 30D", ads: ["Promo Bundling", "Testimoni"] }],
  },
  {
    key: "leads",
    name: "[Leads] Form – Demo",
    objective: "OUTCOME_LEADS",
    status: "PAUSED",
    dailyBudget: 150_000,
    resultType: "lead",
    cpr: 55_000,
    ctr: 1.1,
    adSets: [{ name: "Interest Bisnis 30-50", ads: ["Lead Form 2026"] }],
  },
] as const;

const AGES = ["18-24", "25-34", "35-44", "45-54", "55-64", "65+"] as const;
const GENDERS = ["female", "male", "unknown"] as const;

function campaignId(actId: string, i: number): string {
  return `${120_000_000_000 + (hashString(actId) % 900_000) * 100 + i * 10}`;
}

function build(actId: string) {
  const campaigns: MetaCampaign[] = [];
  const adSets: MetaAdSet[] = [];
  const ads: MetaAd[] = [];
  for (const [i, bp] of BLUEPRINT.entries()) {
    const cid = campaignId(actId, i);
    campaigns.push({
      id: cid,
      name: bp.name,
      objective: bp.objective,
      status: bp.status,
      dailyBudget: bp.dailyBudget,
      startTime: "2026-01-01T00:00:00+0700",
    });
    for (const [j, set] of bp.adSets.entries()) {
      const sid = `${cid}${j + 1}`;
      adSets.push({ id: sid, campaignId: cid, name: set.name, status: bp.status, dailyBudget: bp.dailyBudget / bp.adSets.length });
      for (const [m, adName] of set.ads.entries()) {
        ads.push({
          id: `${sid}${m + 1}`,
          adSetId: sid,
          campaignId: cid,
          name: adName,
          status: bp.status,
          thumbnailUrl: `https://picsum.photos/seed/meta${i}${j}${m}/400/400`,
        });
      }
    }
  }
  return { campaigns, adSets, ads };
}

function eachIsoDay(since: string, until: string): string[] {
  const out: string[] = [];
  const from = new Date(`${since}T00:00:00Z`);
  const to = new Date(`${until}T00:00:00Z`);
  for (let d = from; d <= to && out.length < 400; d = new Date(d.getTime() + 86_400_000)) {
    out.push(d.toISOString().slice(0, 10));
  }
  return out;
}

export const mockMetaMarketing: MetaMarketingAdapter = {
  async listCampaigns(actId) {
    return build(actId).campaigns;
  },
  async listAdSets(actId) {
    return build(actId).adSets;
  },
  async listAds(actId) {
    return build(actId).ads;
  },
  async getInsights(actId, query: InsightsQuery) {
    const { ads } = build(actId);
    const bpByCampaign = new Map(BLUEPRINT.map((b, i) => [campaignId(actId, i), b]));
    const days = eachIsoDay(query.since, query.until);
    const rows: MetaInsightRow[] = [];
    const withBreakdown = (query.breakdowns?.length ?? 0) > 0;
    for (const day of days) {
      for (const [ai, ad] of ads.entries()) {
        const bp = bpByCampaign.get(ad.campaignId) ?? BLUEPRINT[ai % BLUEPRINT.length]!;
        const rnd = mulberry32(hashString(`${actId}|${ad.id}|${day}`));
        const spend = Math.round((bp.dailyBudget / 2) * (0.7 + rnd() * 0.6));
        const impressions = Math.round((spend / (25_000 + rnd() * 12_000)) * 1000);
        const reach = Math.round(impressions / (1.5 + rnd()));
        const linkClicks = Math.round(impressions * (bp.ctr / 100) * (0.75 + rnd() * 0.5));
        const clicks = Math.round(linkClicks * 1.3);
        const results = Math.max(0, Math.round(spend / (bp.cpr * (0.8 + rnd() * 0.4))));
        const base: MetaInsightRow = {
          campaignId: ad.campaignId,
          adSetId: ad.adSetId,
          adId: ad.id,
          date: day,
          spend,
          impressions,
          reach,
          clicks,
          linkClicks,
          frequency: reach > 0 ? impressions / reach : 0,
          results,
          resultType: bp.resultType,
          purchaseValue: bp.resultType === "purchase" ? Math.round(results * bp.cpr * 3) : null,
        };
        if (!withBreakdown) {
          rows.push(base);
          continue;
        }
        // Breakdown deterministik per usia × gender
        for (const [x, age] of AGES.entries()) {
          for (const [y, gender] of GENDERS.entries()) {
            const w = [0.12, 0.36, 0.28, 0.14, 0.07, 0.03][x]! * [0.55, 0.43, 0.02][y]!;
            rows.push({
              ...base,
              age,
              gender,
              spend: Math.round(base.spend * w),
              impressions: Math.round(base.impressions * w),
              reach: Math.round(base.reach * w),
              clicks: Math.round(base.clicks * w),
              linkClicks: Math.round(base.linkClicks * w),
              results: Math.round(base.results * w),
              purchaseValue: base.purchaseValue != null ? Math.round(base.purchaseValue * w) : null,
            });
          }
        }
      }
    }
    return rows;
  },
};
