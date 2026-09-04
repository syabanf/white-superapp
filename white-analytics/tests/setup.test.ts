import { describe, expect, it } from "vitest";
import {
  DEFAULT_SEO_PRICES,
  EMPTY_META_REVIEW,
  INTEGRATIONS,
  estimateSeoCost,
  isConfiguredPair,
  maskSecret,
  parseMetaReview,
  reviewProgress,
} from "@/features/setup/integrations";
import { SETUP_STEPS, nextSetupStep, prevSetupStep, resolveSetupStep, setupHref } from "@/features/setup/steps";
import { estimateApifyCost } from "@/features/setup/integrations";

describe("setup wizard — steps", () => {
  it("resolves unknown steps to the first one and links deterministically", () => {
    expect(resolveSetupStep(undefined)).toBe("meta");
    expect(resolveSetupStep("nope")).toBe("meta");
    expect(resolveSetupStep("seo-data")).toBe("seo-data");
    expect(setupHref("meta")).toBe("/setup");
    expect(setupHref("ai")).toBe("/setup?step=ai");
    expect(nextSetupStep("seo-data")).toBe("apify");
    expect(SETUP_STEPS.length).toBe(6);
    expect(nextSetupStep("selesai")).toBeNull();
    expect(prevSetupStep("meta")).toBeNull();
    expect(SETUP_STEPS.at(-1)).toBe("selesai");
  });
});

describe("integrations — configured & masking", () => {
  it("needs both halves unless the provider is secret-only", () => {
    expect(isConfiguredPair(INTEGRATIONS.META, "id", "secret")).toBe(true);
    expect(isConfiguredPair(INTEGRATIONS.META, null, "secret")).toBe(false);
    expect(isConfiguredPair(INTEGRATIONS.META, "id", null)).toBe(false);
    expect(isConfiguredPair(INTEGRATIONS.OPENROUTER, null, "sk-or")).toBe(true);
  });
  it("masks secrets without leaking the middle", () => {
    expect(maskSecret("short")).toBe("••••");
    expect(maskSecret("sk-or-v1-abcdefghijklmnop")).toBe("sk-o••••mnop");
  });
});

describe("meta app review", () => {
  it("parses tolerant of garbage and reports publish readiness only when live + approved", () => {
    expect(parseMetaReview(null)).toEqual(EMPTY_META_REVIEW);
    const st = parseMetaReview({ permissions: { instagram_content_publish: "APPROVED", pages_manage_posts: "WHAT" }, appLive: 1 });
    expect(st.permissions).toEqual({ instagram_content_publish: "APPROVED" });
    expect(st.appLive).toBe(true);
    expect(reviewProgress(st).publishReady).toBe(false);
    const ready = parseMetaReview({ permissions: { instagram_content_publish: "APPROVED", pages_manage_posts: "APPROVED" }, appLive: true });
    expect(reviewProgress(ready).publishReady).toBe(true);
    expect(reviewProgress({ ...ready, appLive: false }).publishReady).toBe(false);
    expect(reviewProgress(ready).total).toBe(8);
  });
});

describe("dataforseo cost estimate", () => {
  it("scales with keywords and converts to IDR", () => {
    const e = estimateSeoCost({ trackedKeywords: 100, competitorDomains: 3, researchRunsPerMonth: 10, usdIdr: 16_000 });
    expect(e.rankUsd).toBeCloseTo(100 * 30 * DEFAULT_SEO_PRICES.serpPerKeyword);
    expect(e.domainsUsd).toBeCloseTo(4 * 30 * DEFAULT_SEO_PRICES.domainOverviewPerDomainPerDay);
    expect(e.totalUsd).toBeCloseTo(e.rankUsd + e.backlinksUsd + e.domainsUsd + e.researchUsd);
    expect(e.totalIdr).toBeCloseTo(e.totalUsd * 16_000);
    expect(estimateSeoCost({ trackedKeywords: -5, competitorDomains: 0, researchRunsPerMonth: 0, usdIdr: 1 }).rankUsd).toBe(0);
  });
});

describe("apify cost estimate", () => {
  it("adds SERP and social parts and never goes negative", () => {
    const e = estimateApifyCost({ trackedKeywords: 100, socialAccounts: 10, postsPerAccount: 30, usdIdr: 16_000 });
    expect(e.serpUsd).toBeCloseTo(100 * 30 * 0.0035);
    expect(e.socialUsd).toBeGreaterThan(0);
    expect(e.totalUsd).toBeCloseTo(e.serpUsd + e.socialUsd);
    expect(e.totalIdr).toBeCloseTo(e.totalUsd * 16_000);
    expect(estimateApifyCost({ trackedKeywords: -1, socialAccounts: -1, postsPerAccount: -1, usdIdr: 1 }).totalUsd).toBe(0);
  });
});
