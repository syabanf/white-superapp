import "server-only";
import { cache } from "react";
import { db } from "@/lib/db";
import type { DateRange } from "@/lib/dates";
import { toISODate } from "@/lib/dates";
import {
  computeDelta,
  declining,
  fillDaily,
  healthScore,
  lowCtr,
  positionDistribution,
  rising,
  sparkline,
  strikingDistance,
  aggregateSearch,
  type Delta,
  type PositionBucket,
} from "@/lib/metrics";
import { aggregateDimRows, mergeCurrentPrevious, urlPath, type DimAgg, type KeywordOpportunities, type QueryAgg } from "./aggregate";

// ── Context ───────────────────────────────────────────────────

export const getSeoProperty = cache(async (clientId: string) => {
  return db.seoProperty.findFirst({
    where: { clientId },
    orderBy: { createdAt: "asc" },
    select: { id: true, siteUrl: true, ga4PropertyId: true, connectionId: true, auditCadence: true },
  });
});

export const hasGoogleConnection = cache(async (clientId: string): Promise<boolean> => {
  const c = await db.connection.findFirst({ where: { clientId, provider: "GOOGLE" }, select: { id: true } });
  return Boolean(c);
});

// ── Ringkasan ─────────────────────────────────────────────────

export type SearchDailyPoint = {
  date: string;
  clicks: number;
  impressions: number;
  position: number | null;
  clicksPrev: number | null;
  impressionsPrev: number | null;
  positionPrev: number | null;
};

export type Ga4DailyPoint = { date: string; organik: number; lainnya: number };

export type SeoOverviewData = {
  hasData: boolean;
  kpis: { clicks: number; impressions: number; ctr: number; position: number };
  deltas: { clicks: Delta; impressions: Delta; ctr: Delta; position: Delta };
  sparks: { clicks: number[]; impressions: number[] };
  daily: SearchDailyPoint[];
  ga4: {
    hasData: boolean;
    organicSessions: number;
    sessions: number;
    conversions: number;
    organicDelta: Delta;
    conversionsDelta: Delta;
    organicSpark: number[];
  };
  ga4Daily: Ga4DailyPoint[];
  health: number | null;
};

export const getSeoOverview = cache(async (propertyId: string, range: DateRange, previous: DateRange): Promise<SeoOverviewData> => {
  const [rows, prevRows, ga4Rows, ga4PrevRows, latestAudit, latestCrawl] = await Promise.all([
    db.seoDailyMetric.findMany({ where: { propertyId, date: { gte: range.from, lte: range.to } }, orderBy: { date: "asc" } }),
    db.seoDailyMetric.findMany({ where: { propertyId, date: { gte: previous.from, lte: previous.to } } }),
    db.ga4DailyMetric.findMany({ where: { propertyId, date: { gte: range.from, lte: range.to } }, orderBy: { date: "asc" } }),
    db.ga4DailyMetric.findMany({ where: { propertyId, date: { gte: previous.from, lte: previous.to } } }),
    db.seoAudit.findFirst({ where: { propertyId, strategy: "MOBILE" }, orderBy: { runAt: "desc" } }),
    db.seoCrawl.findFirst({
      where: { propertyId, status: "SUCCESS" },
      orderBy: { startedAt: "desc" },
      include: { issues: { select: { severity: true } } },
    }),
  ]);

  const cur = aggregateSearch(rows);
  const prev = aggregateSearch(prevRows);

  const daily = fillDaily(range, rows, (r) => r.date, ["clicks", "impressions"], { missing: null });
  const dailyPos = fillDaily(range, rows, (r) => r.date, ["position"], { missing: null });
  const prevDaily = fillDaily(previous, prevRows, (r) => r.date, ["clicks", "impressions"], { missing: null });
  const prevPos = fillDaily(previous, prevRows, (r) => r.date, ["position"], { missing: null });

  const merged: SearchDailyPoint[] = daily.map((p, i) => ({
    date: p.date,
    clicks: (p.clicks as number) ?? 0,
    impressions: (p.impressions as number) ?? 0,
    position: (dailyPos[i]?.position as number | null) ?? null,
    clicksPrev: (prevDaily[i]?.clicks as number | null) ?? null,
    impressionsPrev: (prevDaily[i]?.impressions as number | null) ?? null,
    positionPrev: (prevPos[i]?.position as number | null) ?? null,
  }));

  const organic = ga4Rows.reduce((a, r) => a + r.organicSessions, 0);
  const sessions = ga4Rows.reduce((a, r) => a + r.sessions, 0);
  const conversions = ga4Rows.reduce((a, r) => a + r.conversions, 0);
  const prevOrganic = ga4PrevRows.reduce((a, r) => a + r.organicSessions, 0);
  const prevConversions = ga4PrevRows.reduce((a, r) => a + r.conversions, 0);

  const ga4DailyRaw = ga4Rows.map((r) => ({ date: r.date, organik: r.organicSessions, lainnya: Math.max(0, r.sessions - r.organicSessions) }));
  const ga4Daily = fillDaily(range, ga4DailyRaw, (r) => r.date, ["organik", "lainnya"], { missing: null }).map((p) => ({
    date: p.date,
    organik: (p.organik as number) ?? 0,
    lainnya: (p.lainnya as number) ?? 0,
  }));

  const errors = latestCrawl?.issues.filter((i) => i.severity === "ERROR").length ?? 0;
  const warnings = latestCrawl?.issues.filter((i) => i.severity === "WARNING").length ?? 0;

  return {
    hasData: rows.length > 0 || prevRows.length > 0,
    kpis: { clicks: cur.clicks, impressions: cur.impressions, ctr: cur.ctr, position: cur.position },
    deltas: {
      clicks: computeDelta(cur.clicks, prev.clicks),
      impressions: computeDelta(cur.impressions, prev.impressions),
      ctr: computeDelta(cur.ctr, prev.ctr),
      position: computeDelta(cur.position, prev.position),
    },
    sparks: { clicks: sparkline(daily, "clicks"), impressions: sparkline(daily, "impressions") },
    daily: merged,
    ga4: {
      hasData: ga4Rows.length > 0,
      organicSessions: organic,
      sessions,
      conversions,
      organicDelta: computeDelta(organic, prevOrganic),
      conversionsDelta: computeDelta(conversions, prevConversions),
      organicSpark: sparkline(ga4Daily, "organik"),
    },
    ga4Daily,
    health: latestAudit ? healthScore(latestAudit, { errors, warnings }) : null,
  };
});

// ── Dimensi: QUERY / PAGE / DEVICE / COUNTRY ─────────────────

async function dimRows(propertyId: string, dimension: "QUERY" | "PAGE" | "COUNTRY" | "DEVICE", range: DateRange) {
  return db.seoDimensionMetric.findMany({
    where: { propertyId, dimension, date: { gte: range.from, lte: range.to } },
    select: { key: true, clicks: true, impressions: true, position: true },
  });
}

export type QueryStats = {
  rows: QueryAgg[];
  distribution: { bucket: PositionBucket; queries: number; clicks: number; impressions: number }[];
  opportunities: KeywordOpportunities;
};

export const getQueryStats = cache(async (propertyId: string, range: DateRange, previous: DateRange): Promise<QueryStats> => {
  const [curRaw, prevRaw] = await Promise.all([dimRows(propertyId, "QUERY", range), dimRows(propertyId, "QUERY", previous)]);
  const rows = mergeCurrentPrevious(aggregateDimRows(curRaw), aggregateDimRows(prevRaw));
  const opportunities: KeywordOpportunities = {
    striking: strikingDistance(rows),
    low_ctr: lowCtr(rows),
    declining: declining(rows),
    rising: rising(rows),
  };
  return { rows, distribution: positionDistribution(rows), opportunities };
});

export type PageAgg = QueryAgg & { path: string };

export const getPageStats = cache(async (propertyId: string, range: DateRange, previous: DateRange): Promise<PageAgg[]> => {
  const [curRaw, prevRaw] = await Promise.all([dimRows(propertyId, "PAGE", range), dimRows(propertyId, "PAGE", previous)]);
  return mergeCurrentPrevious(aggregateDimRows(curRaw), aggregateDimRows(prevRaw)).map((r) => ({ ...r, path: urlPath(r.key) }));
});

export const getDimBreakdown = cache(async (propertyId: string, range: DateRange): Promise<{ devices: DimAgg[]; countries: DimAgg[] }> => {
  const [devices, countries] = await Promise.all([dimRows(propertyId, "DEVICE", range), dimRows(propertyId, "COUNTRY", range)]);
  return { devices: aggregateDimRows(devices), countries: aggregateDimRows(countries) };
});

// ── Audit ─────────────────────────────────────────────────────

export type AuditRow = {
  id: string;
  url: string;
  strategy: "MOBILE" | "DESKTOP";
  runAt: string;
  performance: number;
  seo: number;
  accessibility: number;
  bestPractices: number;
  lcpMs: number | null;
  inpMs: number | null;
  cls: number | null;
  fcpMs: number | null;
  ttfbMs: number | null;
  tbtMs: number | null;
  speedIndexMs: number | null;
};

export type ScoreHistoryPoint = { date: string; performance: number; seo: number; accessibility: number; bestPractices: number };

export type IssueRow = { id: string; severity: "ERROR" | "WARNING" | "NOTICE"; code: string; message: string; url: string };

export type CrawlPageRow = {
  id: string;
  url: string;
  path: string;
  statusCode: number;
  titleLength: number;
  metaLength: number;
  h1Count: number;
  wordCount: number;
  imagesMissingAlt: number;
  indexable: boolean;
  loadMs: number | null;
};

export type AuditData = {
  latestMobile: AuditRow | null;
  latestDesktop: AuditRow | null;
  lastRunAt: string | null;
  history: ScoreHistoryPoint[];
  crawl: {
    id: string;
    status: string;
    startedAt: string;
    finishedAt: string | null;
    pagesCrawled: number;
    maxPages: number;
    error: string | null;
  } | null;
  pages: CrawlPageRow[];
  issues: IssueRow[];
  severityCounts: { errors: number; warnings: number; notices: number };
  codeCounts: { code: string; count: number }[];
};

function serializeAudit(a: {
  id: string;
  url: string;
  strategy: "MOBILE" | "DESKTOP";
  runAt: Date;
  performance: number;
  seo: number;
  accessibility: number;
  bestPractices: number;
  lcpMs: number | null;
  inpMs: number | null;
  cls: number | null;
  fcpMs: number | null;
  ttfbMs: number | null;
  tbtMs: number | null;
  speedIndexMs: number | null;
} | null): AuditRow | null {
  if (!a) return null;
  return {
    id: a.id,
    url: a.url,
    strategy: a.strategy,
    runAt: a.runAt.toISOString(),
    performance: a.performance,
    seo: a.seo,
    accessibility: a.accessibility,
    bestPractices: a.bestPractices,
    lcpMs: a.lcpMs,
    inpMs: a.inpMs,
    cls: a.cls,
    fcpMs: a.fcpMs,
    ttfbMs: a.ttfbMs,
    tbtMs: a.tbtMs,
    speedIndexMs: a.speedIndexMs,
  };
}

export const getAuditData = cache(async (propertyId: string): Promise<AuditData> => {
  const [latestMobile, latestDesktop, mobileHistory, crawl] = await Promise.all([
    db.seoAudit.findFirst({ where: { propertyId, strategy: "MOBILE" }, orderBy: { runAt: "desc" } }),
    db.seoAudit.findFirst({ where: { propertyId, strategy: "DESKTOP" }, orderBy: { runAt: "desc" } }),
    db.seoAudit.findMany({
      where: { propertyId, strategy: "MOBILE" },
      orderBy: { runAt: "asc" },
      take: 24,
      select: { runAt: true, performance: true, seo: true, accessibility: true, bestPractices: true },
    }),
    db.seoCrawl.findFirst({ where: { propertyId }, orderBy: { startedAt: "desc" } }),
  ]);

  const [pages, issues] = crawl
    ? await Promise.all([
        db.seoCrawlPage.findMany({ where: { crawlId: crawl.id }, orderBy: { url: "asc" } }),
        db.seoIssue.findMany({ where: { crawlId: crawl.id }, orderBy: [{ severity: "asc" }, { code: "asc" }] }),
      ])
    : [[], []];

  // one point per run date (last run of the day wins)
  const historyMap = new Map<string, ScoreHistoryPoint>();
  for (const h of mobileHistory) {
    const date = toISODate(h.runAt);
    historyMap.set(date, { date, performance: h.performance, seo: h.seo, accessibility: h.accessibility, bestPractices: h.bestPractices });
  }

  const severityCounts = {
    errors: issues.filter((i) => i.severity === "ERROR").length,
    warnings: issues.filter((i) => i.severity === "WARNING").length,
    notices: issues.filter((i) => i.severity === "NOTICE").length,
  };
  const codeMap = new Map<string, number>();
  for (const i of issues) codeMap.set(i.code, (codeMap.get(i.code) ?? 0) + 1);
  const codeCounts = [...codeMap.entries()].map(([code, count]) => ({ code, count })).sort((a, b) => b.count - a.count);

  const runDates = [latestMobile?.runAt, latestDesktop?.runAt].filter((d): d is Date => Boolean(d));
  const lastRunAt = runDates.length ? new Date(Math.max(...runDates.map((d) => d.getTime()))).toISOString() : null;

  return {
    latestMobile: serializeAudit(latestMobile),
    latestDesktop: serializeAudit(latestDesktop),
    lastRunAt,
    history: [...historyMap.values()],
    crawl: crawl
      ? {
          id: crawl.id,
          status: crawl.status,
          startedAt: crawl.startedAt.toISOString(),
          finishedAt: crawl.finishedAt ? crawl.finishedAt.toISOString() : null,
          pagesCrawled: crawl.pagesCrawled,
          maxPages: crawl.maxPages,
          error: crawl.error,
        }
      : null,
    pages: pages.map((p) => ({
      id: p.id,
      url: p.url,
      path: urlPath(p.url),
      statusCode: p.statusCode,
      titleLength: p.title?.length ?? 0,
      metaLength: p.metaDescription?.length ?? 0,
      h1Count: p.h1Count,
      wordCount: p.wordCount,
      imagesMissingAlt: p.imagesMissingAlt,
      indexable: p.indexable,
      loadMs: p.loadMs,
    })),
    issues: issues.map((i) => ({ id: i.id, severity: i.severity, code: i.code, message: i.message, url: i.url })),
    severityCounts,
    codeCounts,
  };
});

// ── Tren isu per crawl (SEO suite: audit terjadwal) ───────────

export type IssueTrendPoint = { date: string; errors: number; warnings: number; notices: number; pages: number };

/** Issue counts by severity for the last successful crawls (one point per crawl start date; last of the day wins). */
export const getIssueTrend = cache(async (propertyId: string, take = 30): Promise<IssueTrendPoint[]> => {
  const crawls = await db.seoCrawl.findMany({
    where: { propertyId, status: "SUCCESS" },
    orderBy: { startedAt: "desc" },
    take,
    select: { startedAt: true, pagesCrawled: true, issues: { select: { severity: true } } },
  });
  const map = new Map<string, IssueTrendPoint>();
  for (const c of [...crawls].reverse()) {
    const date = toISODate(c.startedAt);
    map.set(date, {
      date,
      errors: c.issues.filter((i) => i.severity === "ERROR").length,
      warnings: c.issues.filter((i) => i.severity === "WARNING").length,
      notices: c.issues.filter((i) => i.severity === "NOTICE").length,
      pages: c.pagesCrawled,
    });
  }
  return [...map.values()];
});
