/**
 * DataForSEO adapter — shared contract for the real REST v3 client and the deterministic mock.
 * All money is IDR, all dates ISO `YYYY-MM-DD` strings so results are serialisable/cacheable.
 */

export type KeywordIntentKey = "INFORMATIONAL" | "NAVIGATIONAL" | "COMMERCIAL" | "TRANSACTIONAL";
export type DeviceKey = "DESKTOP" | "MOBILE";
export type ResearchMode = "ideas" | "questions" | "related";

export type Market = { locationCode: number; languageCode: string };

export type KeywordIdea = {
  keyword: string;
  /** monthly searches */
  volume: number;
  /** keyword difficulty 0–100 */
  difficulty: number;
  /** IDR */
  cpc: number;
  /** paid competition 0–1 */
  competition: number;
  intent: KeywordIntentKey;
  /** last 12 months, oldest first */
  trend: number[];
  serpFeatures: string[];
};

export type SerpRankRequest = { keyword: string; device: DeviceKey };

export type SerpTopResult = { domain: string; position: number; url: string };

export type RankResult = {
  keyword: string;
  device: DeviceKey;
  /** null when the target does not appear in the top 100 */
  position: number | null;
  url: string | null;
  serpFeatures: string[];
  top10: SerpTopResult[];
};

export type BacklinkSummary = {
  backlinks: number;
  referringDomains: number;
  dofollow: number;
  nofollow: number;
  domainRank: number;
  newLast30: number;
  lostLast30: number;
  /** 0–1 share of links with spam score ≥ 60 */
  toxicShare: number;
};

export type BacklinkItem = {
  sourceUrl: string;
  sourceDomain: string;
  targetUrl: string;
  anchor: string;
  dofollow: boolean;
  domainRank: number;
  spamScore: number;
  firstSeen: string;
  lastSeen: string;
  isLost: boolean;
};

export type DomainOverview = {
  domain: string;
  organicKeywords: number;
  organicTraffic: number;
  /** IDR equivalent of the organic traffic bought as ads */
  organicCost: number;
  top3: number;
  top10: number;
  top100: number;
  backlinks: number;
  referringDomains: number;
  domainRank: number;
};

export type DomainKeyword = { keyword: string; position: number; volume: number; url: string };

export type CompetitorSuggestion = { domain: string; intersections: number; organicTraffic: number };

export interface SeoDataProvider {
  keywordIdeas(seed: string, opts: Market & { mode: ResearchMode; limit?: number }): Promise<KeywordIdea[]>;
  keywordMetrics(keywords: string[], locationCode: number, languageCode: string): Promise<KeywordIdea[]>;
  serpRanks(items: SerpRankRequest[], opts: Market & { targetDomain: string }): Promise<RankResult[]>;
  backlinkSummary(domain: string): Promise<BacklinkSummary>;
  backlinks(domain: string, opts: { limit?: number; mode: "all" | "new" | "lost" }): Promise<BacklinkItem[]>;
  domainOverview(domain: string, locationCode: number, languageCode: string): Promise<DomainOverview>;
  domainKeywords(domain: string, locationCode: number, languageCode: string, limit?: number): Promise<DomainKeyword[]>;
  competitors(domain: string, locationCode: number, languageCode: string): Promise<CompetitorSuggestion[]>;
}
