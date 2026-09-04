/**
 * Real Search Console + GA4 sync. Auth-free write path shared by the
 * "Sinkronkan" action and the daily cron. Runs only for properties whose
 * Google connection has a usable token; otherwise reports `mode: "demo"` and
 * writes nothing (the dashboard keeps its seed/demo data).
 */
import "server-only";
import { db } from "@/lib/db";
import { addDays, latestCompleteDay, parseISODate, toISODate } from "@/lib/dates";
import { getConnectionToken } from "@/lib/connections";
import { getGoogleSearchProvider, type GoogleSearchProvider } from "@/lib/providers/google-search";
import { GA4_DIMENSIONS, GA4_METRICS, mapGa4Daily, mapGscDaily, mapGscDimension } from "./sync-map";

export type SeoSyncResult = { mode: "real" | "demo"; daily: number; dimensions: number; ga4: number; error?: string };

type PropertyRow = { id: string; clientId: string; siteUrl: string; ga4PropertyId: string | null; connectionId: string | null };

const DIMS = ["query", "page", "country", "device"] as const;

/** Sync one property for the last `days` complete days. `provider`/`token` are injectable for tests/smoke runs. */
export async function syncSeoProperty(
  property: PropertyRow,
  opts: { days?: number; provider?: GoogleSearchProvider; token?: string } = {},
): Promise<SeoSyncResult> {
  const resolved = opts.token !== undefined ? { token: opts.token } : await getConnectionToken(property.connectionId);
  if (!resolved) return { mode: "demo", daily: 0, dimensions: 0, ga4: 0 };
  const provider = opts.provider ?? getGoogleSearchProvider();
  const endDate = latestCompleteDay();
  const startDate = addDays(endDate, -((opts.days ?? 30) - 1));
  const range = { startDate: toISODate(startDate), endDate: toISODate(endDate) };
  const token = resolved.token;

  // ── GSC daily ──
  const dailyRows = mapGscDaily(await provider.querySearchAnalytics(token, { siteUrl: property.siteUrl, ...range, dimensions: ["date"] }));
  let daily = 0;
  for (const r of dailyRows) {
    const date = parseISODate(r.date);
    if (!date) continue;
    await db.seoDailyMetric.upsert({
      where: { propertyId_date: { propertyId: property.id, date } },
      create: { propertyId: property.id, date, clicks: r.clicks, impressions: r.impressions, ctr: r.ctr, position: r.position },
      update: { clicks: r.clicks, impressions: r.impressions, ctr: r.ctr, position: r.position },
    });
    daily++;
  }

  // ── GSC dimensions (top rows per day) ──
  let dimensions = 0;
  for (const dim of DIMS) {
    const rows = mapGscDimension(
      await provider.querySearchAnalytics(token, { siteUrl: property.siteUrl, ...range, dimensions: ["date", dim], rowLimit: 5000 }),
      dim,
    );
    for (const part of chunk(rows, 200)) {
      await db.$transaction(
        part.map((r) => {
          const date = parseISODate(r.date)!;
          const data = { clicks: r.clicks, impressions: r.impressions, ctr: r.ctr, position: r.position };
          return db.seoDimensionMetric.upsert({
            where: { propertyId_date_dimension_key: { propertyId: property.id, date, dimension: r.dimension, key: r.key } },
            create: { propertyId: property.id, date, dimension: r.dimension, key: r.key, ...data },
            update: data,
          });
        }),
      );
      dimensions += part.length;
    }
  }

  // ── GA4 ──
  let ga4 = 0;
  if (property.ga4PropertyId) {
    const pid = property.ga4PropertyId.startsWith("properties/") ? property.ga4PropertyId : `properties/${property.ga4PropertyId}`;
    const rows = mapGa4Daily(await provider.runGa4Report(token, { propertyId: pid, ...range, metrics: [...GA4_METRICS], dimensions: [...GA4_DIMENSIONS], limit: 10_000 }));
    for (const r of rows) {
      const date = parseISODate(r.date);
      if (!date) continue;
      const data = { sessions: r.sessions, organicSessions: r.organicSessions, users: r.users, engagedSessions: r.engagedSessions, conversions: r.conversions, avgEngagementSec: r.avgEngagementSec };
      await db.ga4DailyMetric.upsert({ where: { propertyId_date: { propertyId: property.id, date } }, create: { propertyId: property.id, date, ...data }, update: data });
      ga4++;
    }
  }
  return { mode: "real", daily, dimensions, ga4 };
}

/** All properties of a client; logs SEO_GSC / SEO_GA4 SyncJobs. */
export async function syncSeoForClient(clientId: string, opts: { days?: number } = {}): Promise<{ properties: number; real: number; results: SeoSyncResult[] }> {
  const properties = await db.seoProperty.findMany({ where: { clientId }, select: { id: true, clientId: true, siteUrl: true, ga4PropertyId: true, connectionId: true } });
  const results: SeoSyncResult[] = [];
  let real = 0;
  for (const p of properties) {
    let r: SeoSyncResult;
    try {
      r = await syncSeoProperty(p, opts);
    } catch (e) {
      r = { mode: "real", daily: 0, dimensions: 0, ga4: 0, error: e instanceof Error ? e.message : String(e) };
    }
    results.push(r);
    if (r.mode === "real" && !r.error) real++;
    const status = r.error ? "FAILED" : "SUCCESS";
    await db.syncJob.create({
      data: { clientId, kind: "SEO_GSC", status, finishedAt: new Date(), message: r.mode === "demo" ? "mock" : `${r.daily} hari, ${r.dimensions} baris dimensi · ${p.siteUrl}`, error: r.error ?? null },
    });
    if (p.ga4PropertyId) {
      await db.syncJob.create({
        data: { clientId, kind: "SEO_GA4", status, finishedAt: new Date(), message: r.mode === "demo" ? "mock" : `${r.ga4} hari · ${p.ga4PropertyId}`, error: r.error ?? null },
      });
    }
  }
  return { properties: properties.length, real, results };
}

function chunk<T>(arr: T[], size: number): T[][] {
  const out: T[][] = [];
  for (let i = 0; i < arr.length; i += size) out.push(arr.slice(i, i + size));
  return out;
}
