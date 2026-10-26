import { resolvePeriod, type ResolvedPeriod } from "@/features/erp/shared/dashboard-period";
import type { DashboardQuery } from "./types";

type SearchParams = Record<string, string | string[] | undefined>;

const one = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v) || undefined;

/** URL params of the dimension filters (besides the period ones). */
export const FILTER_PARAMS = ["pipeline", "owner", "line", "source"] as const;

export const DASH_TABS = ["ringkasan", "pipeline", "penjualan", "tim", "lead", "tagihan"] as const;
export type DashTab = (typeof DASH_TABS)[number];

/** Reads period, comparison, filters and tab from the page's search params. */
export function readDashboardParams(sp: SearchParams, now = new Date()) {
  const period: ResolvedPeriod = resolvePeriod(
    { preset: one(sp.preset), from: one(sp.from), to: one(sp.to), compare: one(sp.compare) },
    "ytd",
    now,
  );
  const query: DashboardQuery = {
    from: period.from,
    to: period.to,
    compare: period.compare,
    pipelineId: one(sp.pipeline),
    ownerUserId: one(sp.owner),
    serviceLine: one(sp.line),
    source: one(sp.source),
  };
  const raw = one(sp.tab);
  const tab: DashTab = DASH_TABS.find((t) => t === raw) ?? "ringkasan";
  return { period, query, tab };
}

/** Builds a list-page link carrying only the params that page understands. */
export function drill(path: string, params: Record<string, string | undefined>): string {
  const s = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) if (v) s.set(k, v);
  const q = s.toString();
  return q ? `${path}?${q}` : path;
}
