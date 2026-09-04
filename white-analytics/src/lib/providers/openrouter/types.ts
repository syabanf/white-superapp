/**
 * OpenRouter adapter contract — strategic insight generation.
 * The payload is a compact, pre-aggregated KPI snapshot built by
 * `features/reports/insights.ts`; adapters never query the DB themselves.
 */

export type InsightModule = "OVERVIEW" | "SOCIAL" | "SEO" | "ADS";
export type InsightLanguage = "id" | "en";

export type InsightRangeInfo = {
  /** ISO date YYYY-MM-DD */
  from: string;
  to: string;
  /** inclusive day count */
  days: number;
  prevFrom: string;
  prevTo: string;
};

export type InsightTopPost = {
  platform: string;
  caption: string;
  type: string;
  engagements: number;
  likes: number;
  comments: number;
};

export type InsightSocialData = {
  followers: number;
  followerGrowth: number;
  followerGrowthPct: number | null;
  engagementRate: number;
  engagementRateDeltaPct: number | null;
  posts: number;
  reach: number;
  reachDeltaPct: number | null;
  topPosts: InsightTopPost[];
};

export type InsightTopQuery = {
  query: string;
  clicks: number;
  impressions: number;
  ctr: number;
  position: number;
};

export type InsightSeoData = {
  clicks: number;
  clicksDeltaPct: number | null;
  impressions: number;
  impressionsDeltaPct: number | null;
  ctr: number;
  position: number;
  positionDeltaAbs: number;
  health: number | null;
  topQueries: InsightTopQuery[];
};

export type InsightCampaign = {
  name: string;
  objective: string;
  spend: number;
  results: number;
  resultType: string | null;
  cpr: number;
};

export type InsightAdsData = {
  currency: string;
  spend: number;
  spendDeltaPct: number | null;
  results: number;
  resultsDeltaPct: number | null;
  cpr: number;
  cprDeltaPct: number | null;
  ctr: number;
  cpm: number;
  roas: number | null;
  campaigns: InsightCampaign[];
};

/** Modules not selected for the insight are simply absent. */
export type InsightPayload = {
  social?: InsightSocialData;
  seo?: InsightSeoData;
  ads?: InsightAdsData;
};

export type GenerateInsightInput = {
  module: InsightModule;
  language: InsightLanguage;
  clientName: string;
  range: InsightRangeInfo;
  data: InsightPayload;
};

export type InsightProvider = {
  /** provider id for logging */
  id: string;
  /** model id stored on AiInsight rows ("mock" for the template adapter) */
  model: string;
  /** Returns a markdown string (headings + lists + **bold**), never HTML. */
  generateInsight(input: GenerateInsightInput): Promise<string>;
};
