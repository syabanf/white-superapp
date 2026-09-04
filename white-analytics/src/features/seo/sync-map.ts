/**
 * Pure mappers from Google Search Console / GA4 adapter rows to DB rows.
 * Kept free of DB/React so they are unit-tested (tests/sync-map.test.ts).
 */
import type { Ga4ReportRow, GscRow } from "@/lib/providers/google-search/types";
import type { SeoDimension } from "@/generated/prisma/enums";

export type DailyRow = { date: string; clicks: number; impressions: number; ctr: number; position: number };
export type DimensionRow = DailyRow & { dimension: SeoDimension; key: string };
export type Ga4Row = { date: string; sessions: number; organicSessions: number; users: number; engagedSessions: number; conversions: number; avgEngagementSec: number };

const round2 = (n: number) => Math.round(n * 100) / 100;

/** GSC rows queried with dimensions ["date"]. */
export function mapGscDaily(rows: GscRow[]): DailyRow[] {
  return rows
    .filter((r) => /^\d{4}-\d{2}-\d{2}$/.test(r.keys[0] ?? ""))
    .map((r) => ({ date: r.keys[0]!, clicks: Math.round(r.clicks), impressions: Math.round(r.impressions), ctr: round2(r.ctr), position: round2(r.position) }));
}

const DIMENSION_FOR: Record<"query" | "page" | "country" | "device", SeoDimension> = { query: "QUERY", page: "PAGE", country: "COUNTRY", device: "DEVICE" };

/** GSC rows queried with dimensions ["date", <dim>]; keys normalised like the seed (country lower, device upper). */
export function mapGscDimension(rows: GscRow[], dim: "query" | "page" | "country" | "device"): DimensionRow[] {
  const out: DimensionRow[] = [];
  for (const r of rows) {
    const date = r.keys[0] ?? "";
    let key = (r.keys[1] ?? "").trim();
    if (!/^\d{4}-\d{2}-\d{2}$/.test(date) || !key) continue;
    if (dim === "country") key = key.toLowerCase();
    if (dim === "device") key = key.toUpperCase();
    if (dim === "query") key = key.toLowerCase();
    out.push({ date, dimension: DIMENSION_FOR[dim], key, clicks: Math.round(r.clicks), impressions: Math.round(r.impressions), ctr: round2(r.ctr), position: round2(r.position) });
  }
  return out;
}

/** GA4 "20260901" → "2026-09-01" (already-ISO input passes through). */
export function ga4Date(raw: string): string | null {
  if (/^\d{4}-\d{2}-\d{2}$/.test(raw)) return raw;
  const m = raw.match(/^(\d{4})(\d{2})(\d{2})$/);
  return m ? `${m[1]}-${m[2]}-${m[3]}` : null;
}

export const GA4_METRICS = ["sessions", "engagedSessions", "totalUsers", "conversions", "userEngagementDuration"] as const;
export const GA4_DIMENSIONS = ["date", "sessionDefaultChannelGroup"] as const;

/**
 * GA4 rows with dimensions [date, channelGroup] and GA4_METRICS →
 * one row per date; organic = "Organic Search"; avg engagement = duration / sessions.
 */
export function mapGa4Daily(rows: Ga4ReportRow[]): Ga4Row[] {
  const by = new Map<string, Ga4Row & { duration: number }>();
  for (const r of rows) {
    const date = ga4Date(r.dimensions[0] ?? "");
    if (!date) continue;
    const channel = (r.dimensions[1] ?? "").toLowerCase();
    const [sessions = 0, engaged = 0, users = 0, conversions = 0, duration = 0] = r.metrics;
    const acc = by.get(date) ?? { date, sessions: 0, organicSessions: 0, users: 0, engagedSessions: 0, conversions: 0, avgEngagementSec: 0, duration: 0 };
    acc.sessions += sessions;
    acc.engagedSessions += engaged;
    acc.users += users;
    acc.conversions += conversions;
    acc.duration += duration;
    if (channel === "organic search") acc.organicSessions += sessions;
    by.set(date, acc);
  }
  return [...by.values()]
    .sort((a, b) => a.date.localeCompare(b.date))
    .map(({ duration, ...row }) => ({
      ...row,
      sessions: Math.round(row.sessions),
      organicSessions: Math.round(row.organicSessions),
      users: Math.round(row.users),
      engagedSessions: Math.round(row.engagedSessions),
      conversions: Math.round(row.conversions),
      avgEngagementSec: row.sessions > 0 ? round2(duration / row.sessions) : 0,
    }));
}
