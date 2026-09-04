/**
 * DataForSEO adapter picker: real REST client when credentials exist, deterministic mock
 * otherwise (demo mode → `DemoBanner` on the SEO suite pages).
 */
import "server-only";
import { lazyProvider } from "@/lib/providers/lazy";
import { isApifyConfigured } from "@/lib/providers/apify/config";
import { apifySerpRanks } from "@/lib/providers/apify/serp";
import type { SeoDataProvider } from "./types";
import { mockSeoData } from "./mock";
import { realSeoData } from "./real";

export function isDataForSeoConfigured(): boolean {
  return Boolean(process.env.DATAFORSEO_LOGIN && process.env.DATAFORSEO_PASSWORD);
}

/** Mock everywhere except live SERP positions from Apify Google Search Scraper. */
const apifySerpSeoData: SeoDataProvider = { ...mockSeoData, serpRanks: apifySerpRanks };

export type SeoDataMode = "dataforseo" | "apify" | "mock";
export function seoDataMode(): SeoDataMode {
  if (isDataForSeoConfigured()) return "dataforseo";
  if (isApifyConfigured()) return "apify";
  return "mock";
}

export const seoData: SeoDataProvider = lazyProvider(() => {
  const mode = seoDataMode();
  return mode === "dataforseo" ? realSeoData : mode === "apify" ? apifySerpSeoData : mockSeoData;
});

export type * from "./types";
