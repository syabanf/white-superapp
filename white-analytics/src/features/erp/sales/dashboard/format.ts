import { formatRupiah } from "@/features/erp/shared/money";
import { formatCompact, formatNumber, formatPercent } from "@/lib/format";
import type { DashboardKpi, DashboardSlice } from "./types";
import { dd } from "./strings";

/** 2_100_000_000 → "Rp 2,1 M" */
export const rpCompact = (v: number) => `Rp\u00a0${formatCompact(v)}`;

type KpiKind = "money" | "count" | "pct" | "days" | "minutes";

const KIND: Record<string, KpiKind> = {
  win_rate: "pct",
  conversion_rate: "pct",
  quote_acceptance: "pct",
  avg_discount: "pct",
  attainment: "pct",
  collection_rate: "pct",
  cycle_days: "days",
  avg_age_days: "days",
  time_to_accept_days: "days",
  days_to_pay: "days",
  first_response_minutes: "minutes",
  open_deals: "count",
  won_deals: "count",
  lost_deals: "count",
  stuck_deals: "count",
  quotes_sent: "count",
  activities: "count",
  leads: "count",
  qualified: "count",
  converted: "count",
  conversations: "count",
  sla_breaches: "count",
  submissions: "count",
};

/** Keys where a drop is good news. */
export const LOWER_IS_BETTER = new Set([
  "cycle_days",
  "avg_age_days",
  "stuck_deals",
  "lost_deals",
  "avg_discount",
  "time_to_accept_days",
  "first_response_minutes",
  "sla_breaches",
  "outstanding",
  "overdue",
  "days_to_pay",
]);

/** Snapshot KPIs the API does not compare. */
export const NO_COMPARE = new Set(["stuck_deals", "avg_age_days"]);

export const kpiKind = (key: string): KpiKind => KIND[key] ?? "money";

/** Tile value: compact for money so nothing truncates. */
export function formatKpi(key: string, v: number): string {
  switch (kpiKind(key)) {
    case "pct":
      return formatPercent(v, 1);
    case "count":
      return formatNumber(v);
    case "days":
      return dd.days(formatNumber(v, 1));
    case "minutes":
      return dd.minutes(formatNumber(v, 1));
    default:
      return rpCompact(v);
  }
}

/** Exact value for hints and tooltips. */
export function formatKpiExact(key: string, v: number): string {
  return kpiKind(key) === "money" ? formatRupiah(v) : formatKpi(key, v);
}

export function findKpi(kpis: DashboardKpi[], key: string): DashboardKpi {
  return kpis.find((k) => k.key === key) ?? { key, value: 0, previous: 0, spark: [] };
}

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];

/** "2026-01" → "Jan 26" */
export function monthLabel(ym: string): string {
  const [y, m] = ym.split("-");
  return `${MONTHS[Number(m) - 1] ?? m} ${y?.slice(2)}`;
}

/** Keeps the top n slices by value and folds the rest into "Lainnya". */
export function topN(items: DashboardSlice[], n: number): DashboardSlice[] {
  if (items.length <= n) return items;
  const rest = items.slice(n - 1);
  const other: DashboardSlice = {
    key: "__others__",
    label: dd.others,
    group: "",
    value: rest.reduce((a, b) => a + b.value, 0),
    count: rest.reduce((a, b) => a + b.count, 0),
    extra: rest.reduce((a, b) => a + b.extra, 0),
  };
  return [...items.slice(0, n - 1), other];
}

export const sum = (xs: number[]) => xs.reduce((a, b) => a + b, 0);
