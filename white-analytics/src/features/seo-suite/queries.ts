import "server-only";
import { cache } from "react";
import { db } from "@/lib/db";
import { addDays, toISODate, todayUtc, type DateRange } from "@/lib/dates";
import { computeDelta, fillDaily, weekKey, type Delta } from "@/lib/metrics";
import { dofollowShare, isCacheFresh, keywordGap, siteHost, type KeywordGap, type TopResult } from "@/lib/metrics/seo-suite";
import { isDataForSeoConfigured, seoData, type CompetitorSuggestion, type KeywordIdea, type ResearchMode } from "@/lib/providers/dataforseo";
import type { Prisma } from "@/generated/prisma/client";
import { pickSeriesDomains, topCounts } from "./lib";
import { s } from "./strings";
import {
  dailySeries,
  distributionOn,
  indexSnapshots,
  keywordRank,
  kpiDeltas,
  kpisOn,
  latestSnapshotDate,
  mergeDaily,
  shareOfVoiceOn,
  type KeywordRank,
  type RankDailyPoint,
  type RankKpis,
  type SnapshotRow,
} from "./rank-aggregate";

// ── Context ───────────────────────────────────────────────────

export type SuiteProperty = { id: string; siteUrl: string; host: string; locationCode: number; languageCode: string; auditCadence: "NONE" | "WEEKLY" | "DAILY" };

export const getSuiteProperty = cache(async (clientId: string): Promise<SuiteProperty | null> => {
  const p = await db.seoProperty.findFirst({
    where: { clientId },
    orderBy: { createdAt: "asc" },
    select: { id: true, siteUrl: true, locationCode: true, languageCode: true, auditCadence: true },
  });
  return p ? { ...p, host: siteHost(p.siteUrl) } : null;
});

export const isSuiteDemo = cache(async (): Promise<boolean> => !isDataForSeoConfigured());

/** Most recent successful refresh of a given job kind — used for cooldown hints. */
export const lastJobAt = cache(async (clientId: string, kinds: string[]): Promise<string | null> => {
  const job = await db.syncJob.findFirst({ where: { clientId, kind: { in: kinds } }, orderBy: { startedAt: "desc" }, select: { startedAt: true } });
  return job ? job.startedAt.toISOString() : null;
});

// ── Riset kata kunci ──────────────────────────────────────────

export type ResearchResult = { ideas: KeywordIdea[]; fetchedAt: string; fromCache: boolean };

/** Cache key: the KeywordResearch table has no `mode` column, so the mode is prefixed into `seed`. */
function researchSeedKey(mode: ResearchMode, seed: string): string {
  return `${mode}:${seed}`;
}

/** Return cached ideas when < 7 days old, otherwise call the provider and cache the run. */
export const getResearch = cache(async (clientId: string, seed: string, mode: ResearchMode, locationCode: number, languageCode: string): Promise<ResearchResult> => {
  const key = researchSeedKey(mode, seed);
  const now = new Date();
  const cached = await db.keywordResearch.findFirst({ where: { clientId, seed: key, locationCode, languageCode }, orderBy: { fetchedAt: "desc" } });
  if (cached && isCacheFresh(cached.fetchedAt, now)) {
    return { ideas: cached.results as KeywordIdea[], fetchedAt: cached.fetchedAt.toISOString(), fromCache: true };
  }
  const ideas = await seoData.keywordIdeas(seed, { locationCode, languageCode, mode, limit: 100 });
  const row = await db.keywordResearch.create({
    data: { clientId, seed: key, locationCode, languageCode, provider: isDataForSeoConfigured() ? "dataforseo" : "mock", results: ideas as unknown as Prisma.InputJsonValue },
  });
  // keep the table small: drop older runs for the same key
  await db.keywordResearch.deleteMany({ where: { clientId, seed: key, locationCode, languageCode, id: { not: row.id } } });
  return { ideas, fetchedAt: row.fetchedAt.toISOString(), fromCache: false };
});

export const getTrackedKeywordSet = cache(async (propertyId: string): Promise<string[]> => {
  const rows = await db.trackedKeyword.findMany({ where: { propertyId }, select: { keyword: true } });
  return rows.map((r) => r.keyword.toLowerCase());
});

export const getCompetitorDomains = cache(async (propertyId: string): Promise<{ id: string; domain: string }[]> => {
  return db.competitorDomain.findMany({ where: { propertyId }, orderBy: { createdAt: "asc" }, select: { id: true, domain: true } });
});

export const getKeywordGap = cache(async (ownDomain: string, competitorDomain: string, locationCode: number, languageCode: string): Promise<KeywordGap> => {
  const [own, comp] = await Promise.all([seoData.domainKeywords(ownDomain, locationCode, languageCode, 200), seoData.domainKeywords(competitorDomain, locationCode, languageCode, 200)]);
  return keywordGap(own, comp);
});

// ── Peringkat ─────────────────────────────────────────────────

export type RankKeywordRow = KeywordRank & {
  id: string;
  keyword: string;
  device: "DESKTOP" | "MOBILE";
  tags: string[];
  volume: number | null;
  difficulty: number | null;
  cpc: number | null;
  intent: "INFORMATIONAL" | "NAVIGATIONAL" | "COMMERCIAL" | "TRANSACTIONAL" | null;
};

export type RankData = {
  hasKeywords: boolean;
  latestDate: string | null;
  kpis: RankKpis;
  deltas: ReturnType<typeof kpiDeltas>;
  strip: { tracked: number; ranked: number; unranked: number; improved: number; declined: number };
  daily: RankDailyPoint[];
  distribution: ReturnType<typeof distributionOn>;
  sov: ReturnType<typeof shareOfVoiceOn>;
  rows: RankKeywordRow[];
  tags: string[];
};

/**
 * Rank snapshots are taken during the day, so the window runs one day past the last
 * complete day (today is included when the range ends yesterday). Previous window likewise.
 */
export const getRankData = cache(async (propertyId: string, range: DateRange, previous: DateRange): Promise<RankData> => {
  const curEnd = addDays(range.to, 1);
  const prevEnd = addDays(previous.to, 1);
  const sparkFrom = addDays(curEnd, -29);
  const loadFrom = new Date(Math.min(previous.from.getTime(), sparkFrom.getTime()));

  const keywords = await db.trackedKeyword.findMany({ where: { propertyId }, orderBy: { createdAt: "asc" } });
  const snaps = keywords.length
    ? await db.rankSnapshot.findMany({
        where: { keywordId: { in: keywords.map((k) => k.id) }, date: { gte: loadFrom, lte: curEnd } },
        select: { keywordId: true, date: true, position: true, url: true, serpFeatures: true, topResults: true },
      })
    : [];
  const rows: SnapshotRow[] = snaps.map((r) => ({
    keywordId: r.keywordId,
    date: toISODate(r.date),
    position: r.position,
    url: r.url,
    serpFeatures: r.serpFeatures,
    topResults: Array.isArray(r.topResults) ? (r.topResults as TopResult[]) : null,
  }));
  const by = indexSnapshots(rows);
  const curEndIso = toISODate(curEnd);
  const prevEndIso = toISODate(prevEnd);
  const latestDate = latestSnapshotDate(rows, curEndIso);
  const prevDate = latestSnapshotDate(rows, prevEndIso);
  const kpis = kpisOn(by, latestDate);
  const prevKpis = kpisOn(by, prevDate);

  const fromIso = toISODate(range.from);
  const table: RankKeywordRow[] = keywords.map((k) => ({
    id: k.id,
    keyword: k.keyword,
    device: k.device,
    tags: k.tags,
    volume: k.volume,
    difficulty: k.difficulty,
    cpc: k.cpc,
    intent: k.intent,
    ...keywordRank(by.get(k.id), fromIso, curEndIso, toISODate(sparkFrom)),
  }));

  const tags = [...new Set(keywords.flatMap((k) => k.tags))].sort();
  return {
    hasKeywords: keywords.length > 0,
    latestDate,
    kpis,
    deltas: kpiDeltas(kpis, prevKpis),
    strip: {
      tracked: keywords.length,
      ranked: table.filter((r) => r.position != null).length,
      unranked: table.filter((r) => r.position == null).length,
      improved: table.filter((r) => r.change.kind === "up" || r.change.kind === "new").length,
      declined: table.filter((r) => r.change.kind === "down" || r.change.kind === "lost").length,
    },
    daily: mergeDaily(dailySeries(by, { from: range.from, to: curEnd }), dailySeries(by, { from: previous.from, to: prevEnd })),
    distribution: distributionOn(by, latestDate),
    sov: shareOfVoiceOn(by, latestDate, s.others),
    rows: table,
    tags,
  };
});

// ── Backlink ──────────────────────────────────────────────────

export type BacklinkRow = {
  id: string;
  sourceUrl: string;
  sourceDomain: string;
  targetUrl: string;
  anchor: string;
  dofollow: boolean;
  domainRank: number | null;
  spamScore: number | null;
  firstSeen: string;
  lastSeen: string;
  isLost: boolean;
};

export type BacklinkData = {
  latest: { date: string; backlinks: number; referringDomains: number; dofollow: number; nofollow: number; newBacklinks: number; lostBacklinks: number; domainRank: number | null; toxicShare: number | null } | null;
  deltas: { backlinks: Delta; referringDomains: Delta; domainRank: Delta; dofollowShare: Delta } | null;
  dofollowShare: number;
  daily: { date: string; referringDomains: number | null }[];
  weekly: { date: string; baru: number; hilang: number }[];
  rows: BacklinkRow[];
  topAnchors: { label: string; count: number }[];
  topDomains: { label: string; count: number }[];
  today: string;
};

export const getBacklinkData = cache(async (propertyId: string, range: DateRange): Promise<BacklinkData> => {
  const [latest, series, links] = await Promise.all([
    db.backlinkSnapshot.findFirst({ where: { propertyId }, orderBy: { date: "desc" } }),
    db.backlinkSnapshot.findMany({ where: { propertyId, date: { gte: range.from, lte: addDays(range.to, 1) } }, orderBy: { date: "asc" } }),
    db.backlink.findMany({ where: { propertyId }, orderBy: [{ isLost: "asc" }, { domainRank: "desc" }] }),
  ]);
  const compareAt = latest ? await db.backlinkSnapshot.findFirst({ where: { propertyId, date: { lte: addDays(latest.date, -30) } }, orderBy: { date: "desc" } }) : null;

  const daily = fillDaily({ from: range.from, to: addDays(range.to, 1) }, series, (r) => r.date, ["referringDomains"], { missing: null }).map((p) => ({
    date: p.date,
    referringDomains: (p.referringDomains as number | null) ?? null,
  }));
  const weeks = new Map<string, { date: string; baru: number; hilang: number }>();
  for (const r of series) {
    const k = weekKey(r.date);
    const w = weeks.get(k) ?? { date: k, baru: 0, hilang: 0 };
    w.baru += r.newBacklinks;
    w.hilang += r.lostBacklinks;
    weeks.set(k, w);
  }
  const live = links.filter((l) => !l.isLost);
  const share = latest ? dofollowShare(latest.dofollow, latest.nofollow) : 0;
  return {
    latest: latest ? { ...latest, date: toISODate(latest.date) } : null,
    deltas:
      latest && compareAt
        ? {
            backlinks: computeDelta(latest.backlinks, compareAt.backlinks),
            referringDomains: computeDelta(latest.referringDomains, compareAt.referringDomains),
            domainRank: computeDelta(latest.domainRank ?? 0, compareAt.domainRank ?? 0),
            dofollowShare: computeDelta(share, dofollowShare(compareAt.dofollow, compareAt.nofollow)),
          }
        : null,
    dofollowShare: share,
    daily,
    weekly: [...weeks.values()].sort((a, b) => a.date.localeCompare(b.date)),
    rows: links.map((l) => ({ ...l, firstSeen: toISODate(l.firstSeen), lastSeen: toISODate(l.lastSeen) })),
    topAnchors: topCounts(live.map((l) => l.anchor)),
    topDomains: topCounts(live.map((l) => l.sourceDomain)),
    today: toISODate(todayUtc()),
  };
});

// ── Kompetitor domain ─────────────────────────────────────────

export type BenchmarkRow = {
  domain: string;
  isOwn: boolean;
  date: string | null;
  organicKeywords: number | null;
  organicTraffic: number | null;
  top3: number | null;
  top10: number | null;
  backlinks: number | null;
  referringDomains: number | null;
  domainRank: number | null;
};

export type CompetitorData = {
  ownDomain: string;
  competitors: { id: string; domain: string }[];
  benchmark: BenchmarkRow[];
  traffic: { series: { key: string; label: string }[]; data: Array<{ date: string } & Record<string, number | string | null>> };
  suggestions: CompetitorSuggestion[];
};

export const getCompetitorData = cache(async (property: SuiteProperty, range: DateRange): Promise<CompetitorData> => {
  const competitors = await getCompetitorDomains(property.id);
  const domains = [property.host, ...competitors.map((c) => c.domain)];
  const [snaps, suggestions] = await Promise.all([
    db.domainSnapshot.findMany({ where: { propertyId: property.id, domain: { in: domains }, date: { gte: range.from, lte: addDays(range.to, 1) } }, orderBy: { date: "asc" } }),
    seoData.competitors(property.host, property.locationCode, property.languageCode).catch(() => [] as CompetitorSuggestion[]),
  ]);
  const latestAny = await db.domainSnapshot.findMany({ where: { propertyId: property.id, domain: { in: domains } }, orderBy: { date: "desc" }, distinct: ["domain"] });
  const latestBy = new Map(latestAny.map((r) => [r.domain, r]));

  const benchmark: BenchmarkRow[] = domains.map((d) => {
    const r = latestBy.get(d);
    return {
      domain: d,
      isOwn: d === property.host,
      date: r ? toISODate(r.date) : null,
      organicKeywords: r?.organicKeywords ?? null,
      organicTraffic: r?.organicTraffic ?? null,
      top3: r?.top3 ?? null,
      top10: r?.top10 ?? null,
      backlinks: r?.backlinks ?? null,
      referringDomains: r?.referringDomains ?? null,
      domainRank: r?.domainRank ?? null,
    };
  });

  const latestTraffic: Record<string, number> = {};
  for (const d of domains) latestTraffic[d] = latestBy.get(d)?.organicTraffic ?? 0;
  const plotted = pickSeriesDomains(property.host, latestTraffic, 3).filter((d) => domains.includes(d));
  // Recharts reads dotted dataKeys as paths, so domains are mapped to d0..d3
  const series = plotted.map((d, i) => ({ key: `d${i}`, label: d }));
  const byDate = new Map<string, Record<string, number>>();
  for (const r of snaps) {
    const i = plotted.indexOf(r.domain);
    if (i < 0) continue;
    const k = toISODate(r.date);
    const acc = byDate.get(k) ?? {};
    acc[`d${i}`] = r.organicTraffic;
    byDate.set(k, acc);
  }
  const trafficRows: Array<Record<string, unknown> & { date: Date }> = [...byDate.entries()].map(([date, v]) => ({ date: new Date(date), ...v }));
  const data = fillDaily({ from: range.from, to: addDays(range.to, 1) }, trafficRows, (r) => r.date, series.map((x) => x.key), { missing: null });

  const known = new Set(domains);
  return { ownDomain: property.host, competitors, benchmark, traffic: { series, data }, suggestions: suggestions.filter((x) => !known.has(x.domain)).slice(0, 6) };
});
