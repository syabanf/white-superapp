/**
 * Google Search Console (searchanalytics.query) + GA4 Data API (runReport) adapter contract.
 * The real adapter talks HTTP with a bearer token; the mock returns deterministic data.
 */

export type GscDimension = "date" | "query" | "page" | "country" | "device";

export type GscQueryRequest = {
  /** GSC property, e.g. "https://example.com/" or "sc-domain:example.com" */
  siteUrl: string;
  /** inclusive, YYYY-MM-DD */
  startDate: string;
  /** inclusive, YYYY-MM-DD */
  endDate: string;
  dimensions: GscDimension[];
  rowLimit?: number;
  startRow?: number;
};

export type GscRow = {
  /** one entry per requested dimension, in order */
  keys: string[];
  clicks: number;
  impressions: number;
  /** percentage 0–100 (normalized from Google's 0–1 fraction) */
  ctr: number;
  position: number;
};

export type Ga4ReportRequest = {
  /** "properties/123456789" */
  propertyId: string;
  startDate: string;
  endDate: string;
  /** GA4 metric API names, e.g. sessions, engagedSessions, totalUsers, conversions */
  metrics: string[];
  /** GA4 dimension API names, e.g. date, sessionDefaultChannelGroup */
  dimensions: string[];
  limit?: number;
};

export type Ga4ReportRow = {
  /** one entry per requested dimension, in order */
  dimensions: string[];
  /** one numeric entry per requested metric, in order */
  metrics: number[];
};

export interface GoogleSearchProvider {
  querySearchAnalytics(accessToken: string, req: GscQueryRequest): Promise<GscRow[]>;
  runGa4Report(accessToken: string, req: Ga4ReportRequest): Promise<Ga4ReportRow[]>;
}
