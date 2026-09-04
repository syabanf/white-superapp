/**
 * Meta Marketing API adapter — kontrak bersama mock & real.
 * Semua tanggal berformat ISO YYYY-MM-DD.
 */

export type MetaCampaign = {
  id: string;
  name: string;
  objective: string;
  status: string; // ACTIVE | PAUSED | ARCHIVED
  dailyBudget: number | null;
  startTime: string | null; // ISO datetime
};

export type MetaAdSet = {
  id: string;
  campaignId: string;
  name: string;
  status: string;
  dailyBudget: number | null;
};

export type MetaAd = {
  id: string;
  adSetId: string;
  campaignId: string;
  name: string;
  status: string;
  thumbnailUrl: string | null;
};

export type MetaInsightRow = {
  campaignId: string;
  adSetId: string;
  adId: string;
  /** date_start (ISO YYYY-MM-DD) */
  date: string;
  spend: number;
  impressions: number;
  reach: number;
  clicks: number;
  linkClicks: number;
  frequency: number;
  /** hasil utama yang dipilih dari `actions` sesuai prioritas objective */
  results: number;
  resultType: string | null;
  /** nilai konversi pembelian dari `action_values` bila ada */
  purchaseValue: number | null;
  /** breakdown opsional */
  age?: string;
  gender?: string;
};

export type InsightsQuery = {
  since: string; // YYYY-MM-DD
  until: string; // YYYY-MM-DD
  level?: "ad" | "adset" | "campaign";
  timeIncrement?: number; // 1 = harian
  breakdowns?: Array<"age" | "gender">;
};

export interface MetaMarketingAdapter {
  listCampaigns(actId: string): Promise<MetaCampaign[]>;
  listAdSets(actId: string): Promise<MetaAdSet[]>;
  listAds(actId: string): Promise<MetaAd[]>;
  getInsights(actId: string, query: InsightsQuery): Promise<MetaInsightRow[]>;
}
