/**
 * Rank tracker aggregation — pure, works on plain snapshot rows so it is testable
 * without a database. Windows are inclusive `[from, to]` ISO day strings.
 */
import { eachDay, toISODate, type DateRange } from "@/lib/dates";
import { computeDelta, type Delta } from "@/lib/metrics";
import { avgPosition, classifyRankChange, countTop, positionToScore, rankDistribution, shareOfVoice, visibility, type RankBucket, type RankChangeKind, type TopResult } from "@/lib/metrics/seo-suite";

export type SnapshotRow = { keywordId: string; date: string; position: number | null; url: string | null; serpFeatures: string[]; topResults: TopResult[] | null };

export type RankKpis = { visibility: number; avgPosition: number | null; top3: number; top10: number };

export type RankDailyPoint = { date: string; visibility: number | null; visibilityPrev: number | null; avgPosition: number | null; avgPositionPrev: number | null };

export type KeywordRank = {
  position: number | null;
  url: string | null;
  serpFeatures: string[];
  startPosition: number | null;
  best: number | null;
  change: { kind: RankChangeKind; delta: number | null };
  spark: number[];
};

type ByKeyword = Map<string, Map<string, SnapshotRow>>;

export function indexSnapshots(rows: SnapshotRow[]): ByKeyword {
  const by: ByKeyword = new Map();
  for (const r of rows) {
    let m = by.get(r.keywordId);
    if (!m) {
      m = new Map();
      by.set(r.keywordId, m);
    }
    m.set(r.date, r);
  }
  return by;
}

/** Latest snapshot per keyword on or before `date` (ISO). */
export function latestOnOrBefore(by: ByKeyword, date: string): Map<string, SnapshotRow> {
  const out = new Map<string, SnapshotRow>();
  for (const [kid, m] of by) {
    let best: SnapshotRow | null = null;
    for (const [d, r] of m) if (d <= date && (!best || d > best.date)) best = r;
    if (best) out.set(kid, best);
  }
  return out;
}

/** Most recent day (≤ `date`) that has any snapshot — the "current" day for KPIs. */
export function latestSnapshotDate(rows: SnapshotRow[], date: string): string | null {
  let latest: string | null = null;
  for (const r of rows) if (r.date <= date && (latest == null || r.date > latest)) latest = r.date;
  return latest;
}

export function kpisOn(by: ByKeyword, date: string | null): RankKpis {
  if (!date) return { visibility: 0, avgPosition: null, top3: 0, top10: 0 };
  const positions = [...by.values()].map((m) => m.get(date)).filter((r): r is SnapshotRow => Boolean(r)).map((r) => r.position);
  return { visibility: visibility(positions), avgPosition: avgPosition(positions), top3: countTop(positions, 3), top10: countTop(positions, 10) };
}

export function kpiDeltas(cur: RankKpis, prev: RankKpis): { visibility: Delta; avgPosition: Delta; top3: Delta; top10: Delta } {
  return {
    visibility: computeDelta(cur.visibility, prev.visibility),
    avgPosition: computeDelta(cur.avgPosition ?? 0, prev.avgPosition ?? 0),
    top3: computeDelta(cur.top3, prev.top3),
    top10: computeDelta(cur.top10, prev.top10),
  };
}

/** Daily visibility / avg position for a window; days without any snapshot are gaps (null). */
export function dailySeries(by: ByKeyword, window: DateRange): { date: string; visibility: number | null; avgPosition: number | null }[] {
  return eachDay(window).map((d) => {
    const date = toISODate(d);
    const positions: Array<number | null> = [];
    for (const m of by.values()) {
      const r = m.get(date);
      if (r) positions.push(r.position);
    }
    if (positions.length === 0) return { date, visibility: null, avgPosition: null };
    return { date, visibility: visibility(positions), avgPosition: avgPosition(positions) };
  });
}

export function mergeDaily(cur: ReturnType<typeof dailySeries>, prev: ReturnType<typeof dailySeries>): RankDailyPoint[] {
  return cur.map((p, i) => ({
    date: p.date,
    visibility: p.visibility,
    visibilityPrev: prev[i]?.visibility ?? null,
    avgPosition: p.avgPosition,
    avgPositionPrev: prev[i]?.avgPosition ?? null,
  }));
}

/** Per-keyword figures for the table: current (≤ `to`), start (first ≥ `from`), best, 30-day spark. */
export function keywordRank(m: Map<string, SnapshotRow> | undefined, from: string, to: string, sparkFrom: string): KeywordRank {
  if (!m || m.size === 0) return { position: null, url: null, serpFeatures: [], startPosition: null, best: null, change: { kind: "none", delta: null }, spark: [] };
  const dates = [...m.keys()].sort();
  const cur = [...dates].reverse().find((d) => d <= to);
  const start = dates.find((d) => d >= from && d <= to);
  const curRow = cur ? m.get(cur)! : null;
  const startRow = start ? m.get(start)! : null;
  let best: number | null = null;
  for (const d of dates) {
    if (d < from || d > to) continue;
    const p = m.get(d)!.position;
    if (p != null && (best == null || p < best)) best = p;
  }
  const spark = dates.filter((d) => d >= sparkFrom && d <= to).map((d) => positionToScore(m.get(d)!.position));
  return {
    position: curRow?.position ?? null,
    url: curRow?.url ?? null,
    serpFeatures: curRow?.serpFeatures ?? [],
    startPosition: startRow?.position ?? null,
    best,
    change: classifyRankChange(startRow?.position ?? null, curRow?.position ?? null),
    spark,
  };
}

export function distributionOn(by: ByKeyword, date: string | null): { bucket: RankBucket; count: number }[] {
  const latest = date ? latestOnOrBefore(by, date) : new Map<string, SnapshotRow>();
  return rankDistribution([...latest.values()].map((r) => r.position));
}

export function shareOfVoiceOn(by: ByKeyword, date: string | null, otherLabel: string): { domain: string; weight: number; share: number }[] {
  if (!date) return [];
  const latest = latestOnOrBefore(by, date);
  return shareOfVoice([...latest.values()].map((r) => r.topResults), 8, otherLabel);
}
