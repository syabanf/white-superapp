/**
 * Integration registry — pure (no DB, no env access) so the wizard, the env
 * hydrator and the tests agree on one list.
 */
export const INTEGRATION_KEYS = ["META", "GOOGLE", "DATAFORSEO", "APIFY", "OPENROUTER"] as const;
export type IntegrationKey = (typeof INTEGRATION_KEYS)[number];

export type IntegrationDef = {
  key: IntegrationKey;
  /** env var holding the non-secret half (app id / client id / login / model) */
  publicEnv: string;
  /** env var holding the secret half */
  secretEnv: string;
  /** both halves required for "configured"? OpenRouter only needs the key. */
  publicRequired: boolean;
};

export const INTEGRATIONS: Record<IntegrationKey, IntegrationDef> = {
  META: { key: "META", publicEnv: "META_APP_ID", secretEnv: "META_APP_SECRET", publicRequired: true },
  GOOGLE: { key: "GOOGLE", publicEnv: "GOOGLE_CLIENT_ID", secretEnv: "GOOGLE_CLIENT_SECRET", publicRequired: true },
  DATAFORSEO: { key: "DATAFORSEO", publicEnv: "DATAFORSEO_LOGIN", secretEnv: "DATAFORSEO_PASSWORD", publicRequired: true },
  APIFY: { key: "APIFY", publicEnv: "APIFY_ACTORS", secretEnv: "APIFY_TOKEN", publicRequired: false },
  OPENROUTER: { key: "OPENROUTER", publicEnv: "OPENROUTER_MODEL", secretEnv: "OPENROUTER_API_KEY", publicRequired: false },
};

export function isIntegrationKey(v: string): v is IntegrationKey {
  return (INTEGRATION_KEYS as readonly string[]).includes(v);
}

export function isConfiguredPair(def: IntegrationDef, publicId: string | null | undefined, secret: string | null | undefined): boolean {
  if (!secret) return false;
  return def.publicRequired ? Boolean(publicId) : true;
}

/** `sk-or-v1-abc…xyz` → `sk-o••••wxyz` — enough to recognise, never enough to use. */
export function maskSecret(secret: string): string {
  if (secret.length <= 8) return "••••";
  return `${secret.slice(0, 4)}••••${secret.slice(-4)}`;
}

// ── Meta App Review ───────────────────────────────────────────

export const META_REVIEW_STATUSES = ["NOT_REQUESTED", "SUBMITTED", "APPROVED"] as const;
export type MetaReviewStatus = (typeof META_REVIEW_STATUSES)[number];

export type MetaPermission = {
  name: string;
  /** which product area it unlocks (for the checklist copy) */
  unlocks: "social" | "publish" | "ads" | "engagement";
  /** false = nice to have later (Fase C) */
  required: boolean;
};

export const META_PERMISSIONS: MetaPermission[] = [
  { name: "pages_show_list", unlocks: "social", required: true },
  { name: "pages_read_engagement", unlocks: "social", required: true },
  { name: "instagram_basic", unlocks: "social", required: true },
  { name: "instagram_manage_insights", unlocks: "social", required: true },
  { name: "business_management", unlocks: "ads", required: true },
  { name: "ads_read", unlocks: "ads", required: true },
  { name: "pages_manage_posts", unlocks: "publish", required: true },
  { name: "instagram_content_publish", unlocks: "publish", required: true },
  { name: "instagram_manage_comments", unlocks: "engagement", required: false },
  { name: "pages_messaging", unlocks: "engagement", required: false },
];

export type MetaReviewState = {
  permissions: Record<string, MetaReviewStatus>;
  appLive: boolean;
  businessVerified: boolean;
};

export const EMPTY_META_REVIEW: MetaReviewState = { permissions: {}, appLive: false, businessVerified: false };

export function parseMetaReview(raw: unknown): MetaReviewState {
  if (!raw || typeof raw !== "object") return EMPTY_META_REVIEW;
  const o = raw as Partial<MetaReviewState>;
  const permissions: Record<string, MetaReviewStatus> = {};
  if (o.permissions && typeof o.permissions === "object") {
    for (const [k, v] of Object.entries(o.permissions)) {
      if ((META_REVIEW_STATUSES as readonly string[]).includes(String(v))) permissions[k] = v as MetaReviewStatus;
    }
  }
  return { permissions, appLive: Boolean(o.appLive), businessVerified: Boolean(o.businessVerified) };
}

export function reviewStatusOf(state: MetaReviewState, permission: string): MetaReviewStatus {
  return state.permissions[permission] ?? "NOT_REQUESTED";
}

/** Approved-required / total-required, plus whether publishing specifically is unlocked. */
export function reviewProgress(state: MetaReviewState): { approved: number; total: number; publishReady: boolean; adsReady: boolean } {
  const required = META_PERMISSIONS.filter((p) => p.required);
  const approved = required.filter((p) => reviewStatusOf(state, p.name) === "APPROVED").length;
  const ready = (area: MetaPermission["unlocks"]) =>
    META_PERMISSIONS.filter((p) => p.unlocks === area && p.required).every((p) => reviewStatusOf(state, p.name) === "APPROVED") &&
    state.appLive;
  return { approved, total: required.length, publishReady: ready("publish"), adsReady: ready("ads") };
}

// ── DataForSEO cost estimate ──────────────────────────────────

export type SeoCostInput = {
  trackedKeywords: number;
  competitorDomains: number;
  researchRunsPerMonth: number;
  /** USD unit prices — editable in the UI, defaults from the public price list (verify before budgeting) */
  prices?: Partial<SeoCostPrices>;
  usdIdr: number;
};

export type SeoCostPrices = {
  serpPerKeyword: number; // live regular SERP, per keyword per day
  backlinkSummaryPerDay: number;
  domainOverviewPerDomainPerDay: number;
  researchPerRun: number;
};

export const DEFAULT_SEO_PRICES: SeoCostPrices = {
  serpPerKeyword: 0.002,
  backlinkSummaryPerDay: 0.02,
  domainOverviewPerDomainPerDay: 0.01,
  researchPerRun: 0.05,
};

export type SeoCostEstimate = { rankUsd: number; backlinksUsd: number; domainsUsd: number; researchUsd: number; totalUsd: number; totalIdr: number };

/** Monthly estimate for the daily job + ad-hoc research. Pure; days fixed at 30. */
export function estimateSeoCost(input: SeoCostInput): SeoCostEstimate {
  const p = { ...DEFAULT_SEO_PRICES, ...input.prices };
  const days = 30;
  const kw = Math.max(0, input.trackedKeywords);
  const domains = 1 + Math.max(0, input.competitorDomains);
  const rankUsd = kw * days * p.serpPerKeyword;
  const backlinksUsd = days * p.backlinkSummaryPerDay;
  const domainsUsd = domains * days * p.domainOverviewPerDomainPerDay;
  const researchUsd = Math.max(0, input.researchRunsPerMonth) * p.researchPerRun;
  const totalUsd = rankUsd + backlinksUsd + domainsUsd + researchUsd;
  return { rankUsd, backlinksUsd, domainsUsd, researchUsd, totalUsd, totalIdr: totalUsd * Math.max(0, input.usdIdr) };
}

// ── Apify cost hints (USD, approximate pay-per-result prices — verify on apify.com/store)
export const APIFY_PRICES = {
  serpPerKeyword: 0.0035, // Google Search Scraper, per query page
  instagramPerResult: 0.0023,
  tiktokPerResult: 0.004,
  facebookPerResult: 0.003,
};

export type ApifyCostInput = { trackedKeywords: number; socialAccounts: number; postsPerAccount: number; usdIdr: number };

/** Monthly estimate: daily SERP per keyword + daily profile/posts refresh per account. */
export function estimateApifyCost(input: ApifyCostInput): { serpUsd: number; socialUsd: number; totalUsd: number; totalIdr: number } {
  const days = 30;
  const serpUsd = Math.max(0, input.trackedKeywords) * days * APIFY_PRICES.serpPerKeyword;
  const perAccountDay = (1 + Math.max(0, input.postsPerAccount)) * APIFY_PRICES.instagramPerResult;
  const socialUsd = Math.max(0, input.socialAccounts) * days * perAccountDay;
  const totalUsd = serpUsd + socialUsd;
  return { serpUsd, socialUsd, totalUsd, totalIdr: totalUsd * Math.max(0, input.usdIdr) };
}
