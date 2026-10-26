"use client";

import { useRouter } from "next/navigation";
import { slotColor } from "@/components/dashboard/charts/chart-primitives";
import { formatPercent } from "@/lib/format";
import { formatRupiah } from "@/features/erp/shared/money";
import { MonthlyChart as FinanceMonthlyChart } from "@/features/erp/finance/dashboard/charts";
import { journalsHref, monthBounds } from "@/features/erp/finance/dashboard/lib";
import { ComboChart } from "@/features/erp/hr/components/dashboard/charts";
import { hrLinks } from "@/features/erp/hr/dashboard/links";
import { monthEndOf } from "@/features/erp/hr/dashboard/range";
import { LinkedTrendChart } from "@/features/erp/performance/dashboard/charts";
import { href as perfHref, monthTick, score, scoreDomain } from "@/features/erp/performance/dashboard/lib";
import { MonthlyChart as SalesMonthlyChart } from "@/features/erp/sales/dashboard/components/charts";
import { rpCompact } from "@/features/erp/sales/dashboard/format";
import { hs } from "./strings";

/** Revenue and expense bars with the net income line; a month opens its journals. */
export function FinanceTrend({
  months,
  compareMonths,
  revenue,
  revenueCompare,
  expense,
  net,
  labels,
}: {
  months: string[];
  compareMonths: string[];
  revenue: number[];
  revenueCompare: number[];
  expense: number[];
  net: number[];
  labels: { revenue: string; expense: string; net: string };
}) {
  const router = useRouter();
  return (
    <FinanceMonthlyChart
      months={months}
      compareMonths={compareMonths}
      height={260}
      onMonthClick={(i) => {
        const m = months[i];
        if (!m) return;
        const b = monthBounds(m);
        router.push(journalsHref(b.from, b.to));
      }}
      series={[
        { key: "revenue", label: labels.revenue, color: slotColor(0), type: "bar", values: revenue, compare: revenueCompare },
        { key: "expense", label: labels.expense, color: slotColor(1), type: "bar", values: expense },
        { key: "net", label: labels.net, color: slotColor(2), type: "line", values: net },
      ]}
    />
  );
}

/** Won value bars against the monthly target line, comparison won value behind; a month opens the sales dashboard for that month. */
export function SalesTrend({
  months,
  won,
  target,
  previous,
  labels,
}: {
  months: string[];
  won: number[];
  target: number[];
  previous: number[];
  labels: { won: string; target: string; previous: string };
}) {
  const router = useRouter();
  return (
    <SalesMonthlyChart
      months={months}
      yFormat={rpCompact}
      height={260}
      onMonth={(i) => {
        const m = months[i];
        if (!m) return;
        const b = monthBounds(m.slice(0, 7));
        router.push(`/erp/sales?${new URLSearchParams({ from: b.from, to: b.to, tab: "penjualan" })}`);
      }}
      series={[
        { key: "won", label: labels.won, values: won, format: formatRupiah },
        { key: "target", label: labels.target, type: "line", color: "var(--foreground)", values: target, format: formatRupiah },
      ]}
      previous={previous.length ? { label: labels.previous, values: previous, format: formatRupiah } : undefined}
    />
  );
}

export type HrMonth = { date: string; label: string; headcount: number; attendanceRate: number | null };

/**
 * Headcount and attendance rate as two small charts on a shared month axis,
 * since a count and a percentage never share one y-scale.
 */
export function HrTrend({ months, to }: { months: HrMonth[]; to: string }) {
  const router = useRouter();
  const rows = months.map((m) => ({ label: m.label, tip: m.label, date: m.date, headcount: m.headcount, attendance: m.attendanceRate }));
  const open = (_: unknown, i: number) => {
    const m = months[i];
    if (m) router.push(hrLinks.attendance({ from: m.date, to: monthEndOf(m.date, to) }));
  };
  return (
    <div className="grid gap-2">
      <p className="text-xs font-medium text-muted-foreground">{hs.hr.headcount}</p>
      <ComboChart
        data={rows}
        height={130}
        legend={false}
        series={[{ key: "headcount", label: hs.hr.headcount, type: "bar", color: slotColor(0) }]}
        onSelect={open}
        summary={`${hs.hr.headcount}, ${months.length} bulan`}
      />
      <p className="mt-2 text-xs font-medium text-muted-foreground">{hs.hr.attendance}</p>
      <ComboChart
        data={rows}
        height={130}
        legend={false}
        yDomain="percent"
        yFormat={(v) => formatPercent(v, 0)}
        series={[{ key: "attendance", label: hs.hr.attendance, type: "line", color: slotColor(1), format: (v) => formatPercent(v, 1) }]}
        onSelect={open}
        summary={`${hs.hr.attendance}, ${months.length} bulan`}
      />
    </div>
  );
}

/** One line per department (at most eight); a month opens that month's KPI page. */
export function PerformanceTrend({
  months,
  lines,
  rows,
  scope,
}: {
  months: string[];
  lines: { id: string; name: string }[];
  rows: Record<string, string | number | null>[];
  scope: "me" | "all";
}) {
  const values = rows.flatMap((r) => lines.map((l) => (typeof r[l.id] === "number" ? (r[l.id] as number) : undefined)));
  return (
    <LinkedTrendChart
      data={rows}
      xKey="month"
      xFormat={monthTick}
      yFormat={(v) => score(v)}
      yDomain={scoreDomain(values)}
      height={260}
      series={lines.map((l, i) => ({ key: l.id, label: l.name, color: slotColor(i), format: (v: number) => score(v) }))}
      hrefFor={(row) => perfHref("/erp/performance/kpi", { month: String(row.month), scope })}
      summary={`${hs.performance.chart}, ${months.length} bulan`}
    />
  );
}
