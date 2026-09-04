/**
 * End-to-end smoke for the real sync pipelines against the local DB using the
 * MOCK adapters (no credentials, no network). Creates a throwaway client with
 * one SEO property, one ad account and one IG account, runs GSC/GA4, Meta Ads
 * and Meta Insights sync with injected providers, asserts row counts, and
 * deletes the client (cascade). Run: pnpm sync:smoke
 */
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import { PrismaClient } from "@/generated/prisma/client";
import { PrismaPg } from "@prisma/adapter-pg";
import { mockGoogleSearchProvider } from "@/lib/providers/google-search/mock";
import { mockMetaMarketing } from "@/lib/providers/meta-marketing/mock";
import { mockMetaGraph } from "@/lib/providers/meta-graph/mock";
import { syncSeoProperty } from "@/features/seo/sync";
import { syncAdAccount } from "@/features/ads/sync";
import { syncSocialMeta } from "@/features/social/sync";

const db = new PrismaClient({ adapter: new PrismaPg({ connectionString: process.env.DATABASE_URL }) });
let clientId = "";

beforeAll(async () => {
  const client = await db.client.create({ data: { name: "Smoke Sync", slug: `smoke-sync-${Date.now()}`, shareEnabled: false } });
  clientId = client.id;
});
afterAll(async () => {
  if (clientId) await db.client.delete({ where: { id: clientId } });
  await db.$disconnect();
});

describe("sync pipelines (mock adapters → DB)", () => {
  it("GSC + GA4 → SeoDailyMetric / SeoDimensionMetric / Ga4DailyMetric (idempotent)", async () => {
    const property = await db.seoProperty.create({ data: { clientId, siteUrl: "https://smoke.example/", ga4PropertyId: "properties/123" } });
    const r1 = await syncSeoProperty(property, { days: 14, provider: mockGoogleSearchProvider, token: "mock" });
    expect(r1.mode).toBe("real");
    expect(r1.daily).toBe(14);
    expect(r1.dimensions).toBeGreaterThan(0);
    expect(r1.ga4).toBe(14);
    const r2 = await syncSeoProperty(property, { days: 14, provider: mockGoogleSearchProvider, token: "mock" });
    expect(await db.seoDailyMetric.count({ where: { propertyId: property.id } })).toBe(14);
    expect(await db.seoDimensionMetric.count({ where: { propertyId: property.id } })).toBe(r2.dimensions);
    expect(await db.ga4DailyMetric.count({ where: { propertyId: property.id } })).toBe(14);
    const demo = await syncSeoProperty({ ...property, connectionId: null }, { days: 7 });
    expect(demo.mode).toBe("demo");
  });

  it("Meta Marketing API → hierarchy + AdDailyInsight + AdDemographic (idempotent)", async () => {
    const account = await db.adAccount.create({ data: { clientId, externalId: "act_smoke", name: "Smoke Ads" } });
    const r1 = await syncAdAccount(account, { days: 14, adapter: mockMetaMarketing });
    expect(r1.mode).toBe("real");
    expect(r1.campaigns).toBeGreaterThan(0);
    expect(r1.ads).toBeGreaterThan(0);
    expect(r1.days).toBeGreaterThan(0);
    const before = await db.adDailyInsight.count({ where: { adAccountId: account.id } });
    await syncAdAccount(account, { days: 14, adapter: mockMetaMarketing });
    expect(await db.adDailyInsight.count({ where: { adAccountId: account.id } })).toBe(before);
    expect(await db.adDemographic.count({ where: { adAccountId: account.id } })).toBe(r1.demographics);
    expect((await db.adAccount.findUnique({ where: { id: account.id } }))?.lastSyncedAt).not.toBeNull();
  });

  it("Meta Insights → SocialSnapshot (reach/impressions) + SocialPost with per-post insights", async () => {
    await db.socialAccount.create({ data: { clientId, platform: "INSTAGRAM", externalId: "ig_smoke", username: "smoke", displayName: "Smoke" } });
    const r = await syncSocialMeta({ clientId, platform: "INSTAGRAM", days: 14, provider: mockMetaGraph, token: "mock" });
    expect(r.updated).toBe(1);
    expect(r.failed).toEqual([]);
    const snaps = await db.socialSnapshot.findMany({ where: { account: { clientId } } });
    expect(snaps.length).toBeGreaterThanOrEqual(14);
    expect(snaps.some((s) => (s.reach ?? 0) > 0)).toBe(true);
    const posts = await db.socialPost.findMany({ where: { account: { clientId } } });
    expect(posts.length).toBeGreaterThan(0);
    expect(posts.some((p) => p.reach > 0 || p.saves > 0)).toBe(true);
    // no connection + no injected token → skipped, nothing thrown
    const skipped = await syncSocialMeta({ clientId, platform: "INSTAGRAM", days: 7 });
    expect(skipped.skipped).toBe(1);
  });
});
