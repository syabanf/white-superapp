/**
 * SEO suite — pure helpers shared by server and client code (no DB, no React).
 * Tested in tests/seo-suite.test.ts.
 */
import type { DifficultyBand, RankBucket } from "@/lib/metrics/seo-suite";
import type { KeywordIdea, KeywordIntentKey, ResearchMode } from "@/lib/providers/dataforseo/types";
import { s } from "./strings";

export const RESEARCH_MODES: ResearchMode[] = ["ideas", "questions", "related"];

export function normalizeMode(v: string | undefined | null): ResearchMode {
  return RESEARCH_MODES.includes(v as ResearchMode) ? (v as ResearchMode) : "ideas";
}

export const MODE_LABELS: Record<ResearchMode, string> = { ideas: s.modeIdeas, questions: s.modeQuestions, related: s.modeRelated };

export const BAND_LABELS: Record<DifficultyBand, string> = { easy: s.bandEasy, medium: s.bandMedium, hard: s.bandHard, very_hard: s.bandVeryHard };

export const INTENT_LABELS: Record<KeywordIntentKey, string> = {
  INFORMATIONAL: s.intentInformational,
  NAVIGATIONAL: s.intentNavigational,
  COMMERCIAL: s.intentCommercial,
  TRANSACTIONAL: s.intentTransactional,
};

export const BUCKET_LABELS: Record<RankBucket, string> = {
  "1-3": "1–3",
  "4-10": "4–10",
  "11-20": "11–20",
  "21-50": "21–50",
  "51-100": "51–100",
  unranked: s.bucketUnranked,
};

export const SERP_FEATURE_LABELS: Record<string, string> = {
  featured_snippet: "Snippet",
  people_also_ask: "PAA",
  images: "Gambar",
  video: "Video",
  local_pack: "Lokal",
  shopping: "Belanja",
  sitelinks: "Sitelinks",
  reviews: "Ulasan",
  top_stories: "Berita",
  knowledge_panel: "Knowledge",
  knowledge_graph: "Knowledge",
  answer_box: "Snippet",
  carousel: "Carousel",
  map: "Peta",
  twitter: "X",
  paid: "Iklan",
};

export function serpFeatureLabel(key: string): string {
  return SERP_FEATURE_LABELS[key] ?? key.replace(/_/g, " ");
}

/** Normalise a seed / keyword: trim, collapse spaces, lower-case. Empty → "". */
export function normalizeKeyword(v: string): string {
  return v.trim().replace(/\s+/g, " ").toLowerCase();
}

/** Textarea → unique keyword list (one per line or comma-separated), capped. */
export function parseKeywordLines(text: string, max = 100): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const raw of text.split(/[\n,]+/)) {
    const k = normalizeKeyword(raw);
    if (!k || k.length > 120 || seen.has(k)) continue;
    seen.add(k);
    out.push(k);
    if (out.length >= max) break;
  }
  return out;
}

/** "brand, Produk ,, x" → ["brand", "produk", "x"] (unique, lower-case, ≤ 10). */
export function parseTags(text: string, max = 10): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const raw of text.split(/[,\n]+/)) {
    const tag = raw.trim().toLowerCase().replace(/\s+/g, " ");
    if (!tag || tag.length > 30 || seen.has(tag)) continue;
    seen.add(tag);
    out.push(tag);
    if (out.length >= max) break;
  }
  return out;
}

export type IdeaFilters = {
  minVolume: number;
  maxKd: number;
  intent: KeywordIntentKey | "all";
  include: string;
  exclude: string;
};

export const DEFAULT_IDEA_FILTERS: IdeaFilters = { minVolume: 0, maxKd: 100, intent: "all", include: "", exclude: "" };

function words(v: string): string[] {
  return v
    .toLowerCase()
    .split(/[\s,]+/)
    .map((w) => w.trim())
    .filter(Boolean);
}

/** Client-side idea filter: volume floor, KD ceiling, intent, include-all / exclude-any words. */
export function filterIdeas<T extends Pick<KeywordIdea, "keyword" | "volume" | "difficulty" | "intent">>(ideas: T[], f: IdeaFilters): T[] {
  const inc = words(f.include);
  const exc = words(f.exclude);
  return ideas.filter((i) => {
    if (i.volume < f.minVolume) return false;
    if (i.difficulty > f.maxKd) return false;
    if (f.intent !== "all" && i.intent !== f.intent) return false;
    const k = i.keyword.toLowerCase();
    if (inc.length && !inc.every((w) => k.includes(w))) return false;
    if (exc.length && exc.some((w) => k.includes(w))) return false;
    return true;
  });
}

/** Summary numbers for the research StatStrip. */
export function summarizeIdeas(ideas: Array<Pick<KeywordIdea, "volume" | "difficulty">>): { count: number; totalVolume: number; avgKd: number } {
  const count = ideas.length;
  const totalVolume = ideas.reduce((a, i) => a + i.volume, 0);
  const avgKd = count ? ideas.reduce((a, i) => a + i.difficulty, 0) / count : 0;
  return { count, totalVolume, avgKd };
}

/** Top-N label/count pairs from a list of strings (e.g. anchors, referring domains). */
export function topCounts(values: string[], n = 8): { label: string; count: number }[] {
  const acc = new Map<string, number>();
  for (const v of values) {
    const k = v.trim() || "(kosong)";
    acc.set(k, (acc.get(k) ?? 0) + 1);
  }
  return [...acc.entries()]
    .map(([label, count]) => ({ label, count }))
    .sort((a, b) => b.count - a.count)
    .slice(0, n);
}

/** Pick the series to plot: own domain first, then the top `n` competitors by latest value. */
export function pickSeriesDomains(own: string, latestByDomain: Record<string, number>, n = 3): string[] {
  const others = Object.keys(latestByDomain)
    .filter((d) => d !== own)
    .sort((a, b) => (latestByDomain[b] ?? 0) - (latestByDomain[a] ?? 0))
    .slice(0, n);
  return [own, ...others];
}
