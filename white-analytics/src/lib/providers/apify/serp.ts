import "server-only";
import type { Market, RankResult, SerpRankRequest } from "@/lib/providers/dataforseo/types";
import { runActor } from "./client";
import { countryForLocation, normalizeSerpItems } from "./normalize";
import { getApifyActors } from "./config";

/**
 * Rank tracker via Apify Google Search Scraper: one run per device, all
 * keywords of the batch as newline-separated queries, first page of 100.
 */
export async function apifySerpRanks(items: SerpRankRequest[], opts: Market & { targetDomain: string }): Promise<RankResult[]> {
  if (items.length === 0) return [];
  const actors = getApifyActors();
  const byDevice = new Map<SerpRankRequest["device"], SerpRankRequest[]>();
  for (const it of items) byDevice.set(it.device, [...(byDevice.get(it.device) ?? []), it]);

  const out = new Map<string, RankResult>();
  for (const [device, reqs] of byDevice) {
    const raw = await runActor(actors.serp, {
      queries: reqs.map((r) => r.keyword).join("\n"),
      countryCode: countryForLocation(opts.locationCode),
      languageCode: opts.languageCode,
      maxPagesPerQuery: 1,
      resultsPerPage: 100,
      mobileResults: device === "MOBILE",
      includeUnfilteredResults: false,
      saveHtml: false,
      saveHtmlToKeyValueStore: false,
    }, { timeoutSecs: 180 });
    for (const r of normalizeSerpItems(raw, reqs, opts.targetDomain)) out.set(`${r.device}:${r.keyword.toLowerCase()}`, r);
  }
  return items.map((it) => out.get(`${it.device}:${it.keyword.toLowerCase()}`) ?? { keyword: it.keyword, device: it.device, position: null, url: null, serpFeatures: [], top10: [] });
}
