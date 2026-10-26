"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { ChartCard } from "@/components/dashboard/chart-card";
import { KpiGrid } from "@/components/dashboard/kpi-tile";
import { StatStrip } from "@/components/dashboard/stat-strip";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { HBarChart, type HBarItem } from "@/components/dashboard/charts/hbar-chart";
import { WaterfallChart } from "@/components/dashboard/charts/waterfall-chart";
import { formatDate, formatNumber, formatPercent } from "@/lib/format";
import { formatMonth } from "@/features/erp/shared/period-picker";
import { SelectFilter } from "@/features/erp/shared/url-filters";
import {
  FinanceKpi,
  MonthlyChart,
  SortTable,
  StackRows,
  rampColor,
  slotColor,
  type Column,
  type MonthSeries,
  type StackSegment,
} from "./charts";
import { compactIdr, exactIdr, journalsHref, kpiMap, ledgerHref, monthBounds, num, nums, rangeQs } from "./lib";
import type {
  AnalyticsAmount,
  AnalyticsSeries,
  ClaimSummary,
  DashboardFilters,
  FinanceArAp,
  FinanceCash,
  FinanceProfitability,
  FinanceSpend,
  FinanceSummary,
  OverdueInvoice,
  PartyBalance,
} from "./types";
import { ds } from "./strings";

type Props<T> = { data: T; filters: DashboardFilters };

const money = (v: string) => num(v) ?? 0;
const seriesOf = (list: AnalyticsSeries[], key: string) => list.find((s) => s.key === key);

/** Top n slices plus one "Lainnya" slice, as horizontal bars. */
function ranked(items: AnalyticsAmount[], n: number, href?: (a: AnalyticsAmount) => string | undefined): HBarItem[] {
  const top = items.slice(0, n).map((a) => ({
    label: a.code ? `${a.code} ${a.label}` : a.label,
    value: money(a.value),
    href: href?.(a),
  }));
  const rest = items.slice(n).reduce((s, a) => s + money(a.value), 0);
  return rest > 0 ? [...top, { label: ds.others, value: rest, muted: true }] : top;
}

/** Top n series plus "Lainnya", each colored by fixed slot. */
function topSeries(list: AnalyticsSeries[], n: number, type: "bar" | "line" = "bar"): MonthSeries[] {
  const top: MonthSeries[] = list.slice(0, n).map((s, i) => ({ key: s.key, label: s.label, color: slotColor(i), type, values: nums(s.values) }));
  const rest = list.slice(n);
  if (rest.length) {
    const len = list[0]?.values.length ?? 0;
    top.push({
      key: "others",
      label: ds.others,
      color: "var(--chart-emphasis-muted)",
      type,
      values: Array.from({ length: len }, (_, i) => rest.reduce((s, r) => s + money(r.values[i] ?? ""), 0)),
    });
  }
  return top;
}

function useMonthJournals(months: string[]) {
  const router = useRouter();
  return (i: number) => {
    const m = months[i];
    if (!m) return;
    const { from, to } = monthBounds(m);
    router.push(journalsHref(from, to));
  };
}

const pctOf = (n: number, d: number) => (d === 0 ? null : (n / d) * 100);

// ─── Ringkasan ──────────────────────────────────────────────

export function SummaryTab({ data, filters }: Props<FinanceSummary>) {
  const k = kpiMap(data.kpis);
  const w = data.window;
  const onMonth = useMonthJournals(w.months);
  const rev = seriesOf(data.pnl, "revenue");
  const exp = seriesOf(data.pnl, "expense");
  const net = seriesOf(data.pnl, "net_income");
  const cash = data.cash[0];
  const bs = `/erp/finance/reports?r=balance-sheet&asOf=${filters.to}`;
  // One hue per side, shaded per group, so the legend never runs past the palette.
  const segs = (list: AnalyticsAmount[], hue: string): StackSegment[] =>
    list.map((a, i) => ({ key: a.key, label: a.label, value: money(a.value), color: rampColor(list.length - 1 - i, list.length, hue), href: bs }));
  const assetSegs = segs(data.assets, slotColor(0));
  const leSegs = [...segs(data.liabilities, slotColor(1)), ...segs(data.equity, slotColor(2))];
  const tbDiff = money(data.trialBalanceDiff);
  const rows = w.months.map((m, i) => {
    const r = money(rev?.values[i] ?? "");
    const n = money(net?.values[i] ?? "");
    return { month: m, revenue: r, expense: money(exp?.values[i] ?? ""), net: n, margin: pctOf(n, r), cash: money(cash?.values[i] ?? "") };
  });
  const cols: Column<(typeof rows)[number]>[] = [
    { key: "month", label: "Bulan", render: (r) => formatMonth(r.month), sort: (r) => r.month },
    { key: "revenue", label: ds.kpi.revenue, align: "right", render: (r) => exactIdr(r.revenue), sort: (r) => r.revenue, total: exactIdr(rows.reduce((s, r) => s + r.revenue, 0)) },
    { key: "expense", label: ds.kpi.expense, align: "right", render: (r) => exactIdr(r.expense), sort: (r) => r.expense, total: exactIdr(rows.reduce((s, r) => s + r.expense, 0)) },
    {
      key: "net",
      label: ds.kpi.net_income,
      align: "right",
      render: (r) => <span className={r.net < 0 ? "text-negative" : undefined}>{exactIdr(r.net)}</span>,
      sort: (r) => r.net,
      total: exactIdr(rows.reduce((s, r) => s + r.net, 0)),
    },
    { key: "margin", label: ds.kpi.net_margin, align: "right", render: (r) => formatPercent(r.margin, 1), sort: (r) => r.margin ?? -Infinity },
    { key: "cash", label: ds.kpi.cash, align: "right", render: (r) => exactIdr(r.cash), sort: (r) => r.cash },
  ];
  return (
    <>
      <KpiGrid cols={4}>
        <FinanceKpi kpi={k.revenue} href={`/erp/finance/reports?r=income-statement&${rangeQs(filters.from, filters.to)}`} />
        <FinanceKpi kpi={k.expense} lowerIsBetter href={`/erp/finance/reports?r=income-statement&${rangeQs(filters.from, filters.to)}`} />
        <FinanceKpi kpi={k.net_income} />
        <FinanceKpi kpi={k.net_margin} unit="pct" />
        <FinanceKpi kpi={k.cash} href="/erp/finance/cash-bank" />
        <FinanceKpi kpi={k.receivable} lowerIsBetter href="/erp/finance/receivables?tab=aging" />
        <FinanceKpi kpi={k.payable} href="/erp/finance/payables?tab=aging" />
        <FinanceKpi kpi={k.pending_reimbursement} href="/erp/finance/reimbursements" />
      </KpiGrid>
      <div className="grid gap-5 lg:grid-cols-3">
        <ChartCard title={ds.summary.trend} description={ds.summary.trendQ} className="lg:col-span-2">
          <MonthlyChart
            months={w.months}
            compareMonths={w.compareMonths}
            onMonthClick={onMonth}
            series={[
              { key: "revenue", label: ds.kpi.revenue, color: slotColor(0), type: "bar", values: nums(rev?.values ?? []), compare: nums(rev?.compareValues ?? []) },
              { key: "expense", label: ds.kpi.expense, color: slotColor(1), type: "bar", values: nums(exp?.values ?? []) },
              { key: "net", label: ds.kpi.net_income, color: slotColor(2), type: "line", values: nums(net?.values ?? []) },
            ]}
          />
        </ChartCard>
        <ChartCard title={ds.summary.composition} description={ds.summary.compositionQ}>
          <StackRows
            rows={[
              { key: "assets", label: ds.summary.assets, total: assetSegs.reduce((s, x) => s + x.value, 0), segments: assetSegs },
              { key: "le", label: `${ds.summary.liabilities} & ${ds.summary.equity.toLowerCase()}`, total: leSegs.reduce((s, x) => s + x.value, 0), segments: leSegs },
            ]}
            legend={[...assetSegs, ...leSegs]}
          />
        </ChartCard>
      </div>
      <div className="grid gap-5 lg:grid-cols-3">
        <ChartCard title={ds.summary.cashTrend} description={ds.summary.cashTrendQ} className="lg:col-span-2">
          <MonthlyChart
            months={w.months}
            compareMonths={w.compareMonths}
            height={240}
            series={[{ key: "cash", label: ds.kpi.cash, color: slotColor(0), type: "line", values: nums(cash?.values ?? []), compare: nums(cash?.compareValues ?? []) }]}
          />
        </ChartCard>
        <div className="flex flex-col gap-3">
          <h3 className="text-[15px] font-semibold">{ds.summary.health}</h3>
          <StatStrip
            minWidth={150}
            items={[
              {
                label: ds.summary.tbDiff,
                value: exactIdr(tbDiff),
                caption: (
                  <StatusBadge kind={Math.abs(tbDiff) < 0.01 ? "good" : "warning"}>
                    {Math.abs(tbDiff) < 0.01 ? ds.summary.balanced : ds.summary.unbalanced}
                  </StatusBadge>
                ),
              },
              { label: ds.summary.posted, value: formatNumber(data.postedEntries) },
              { label: ds.summary.drafts, value: formatNumber(data.draftEntries) },
            ]}
          />
        </div>
      </div>
      <ChartCard title={ds.summary.monthly}>
        <SortTable
          rows={rows}
          columns={cols}
          rowKey={(r) => r.month}
          rowHref={(r) => {
            const b = monthBounds(r.month);
            return journalsHref(b.from, b.to);
          }}
        />
      </ChartCard>
    </>
  );
}

// ─── Profitabilitas ─────────────────────────────────────────

export function ProfitTab({ data, filters }: Props<FinanceProfitability>) {
  const router = useRouter();
  const k = kpiMap(data.kpis);
  const w = data.window;
  const get = (key: string) => seriesOf(data.pnl, key);
  const margin = (key: string, compare = false) => {
    const rev = get("revenue");
    const s = get(key);
    const a = compare ? s?.compareValues : s?.values;
    const r = compare ? rev?.compareValues : rev?.values;
    return (a ?? []).map((v, i) => pctOf(money(v), money(r?.[i] ?? "")) ?? 0);
  };
  const isHref = `/erp/finance/reports?r=income-statement&${rangeQs(filters.from, filters.to)}`;
  const expenseTotal = data.expenseGroups.reduce((s, g) => s + money(g.value), 0);
  const monthCols: Column<AnalyticsSeries>[] = [
    { key: "account", label: ds.profit.account, render: (s) => <span><span className="mr-1.5 font-mono text-xs text-muted-foreground">{s.code}</span>{s.label}</span>, sort: (s) => s.code, className: "min-w-56" },
    ...w.months.map((m, i) => ({
      key: m,
      label: formatMonth(m).slice(0, 3) + " " + m.slice(2, 4),
      align: "right" as const,
      render: (s: AnalyticsSeries) => compactIdr(money(s.values[i] ?? "")),
      sort: (s: AnalyticsSeries) => money(s.values[i] ?? ""),
    })),
    { key: "total", label: ds.total, align: "right", render: (s) => <span className="font-semibold">{compactIdr(money(s.total))}</span>, sort: (s) => money(s.total) },
  ];
  const changeCols: Column<AnalyticsAmount>[] = [
    { key: "label", label: ds.profit.account, render: (a) => <span><span className="mr-1.5 font-mono text-xs text-muted-foreground">{a.code}</span>{a.label}</span>, sort: (a) => a.label },
    { key: "group", label: ds.filters.accountGroup, render: (a) => <span className="text-muted-foreground">{a.group}</span>, sort: (a) => a.group },
    { key: "previous", label: ds.previous, align: "right", render: (a) => exactIdr(money(a.previous)), sort: (a) => money(a.previous), total: exactIdr(data.expenseAccounts.reduce((s, a) => s + money(a.previous), 0)) },
    { key: "value", label: ds.kpi.expense, align: "right", render: (a) => exactIdr(money(a.value)), sort: (a) => money(a.value), total: exactIdr(expenseTotal) },
    {
      key: "change",
      label: ds.profit.change,
      align: "right",
      render: (a) => {
        const d = money(a.value) - money(a.previous);
        return <span className={d > 0 ? "text-negative" : d < 0 ? "text-positive" : undefined}>{`${d > 0 ? "+" : ""}${compactIdr(d)}`}</span>;
      },
      sort: (a) => money(a.value) - money(a.previous),
    },
  ];
  return (
    <>
      <KpiGrid cols={6}>
        <FinanceKpi kpi={k.revenue} href={isHref} />
        <FinanceKpi kpi={k.gross_profit} href={isHref} />
        <FinanceKpi kpi={k.gross_margin} unit="pct" />
        <FinanceKpi kpi={k.operating_margin} unit="pct" />
        <FinanceKpi kpi={k.net_margin} unit="pct" />
        <FinanceKpi kpi={k.opex_ratio} unit="pct" lowerIsBetter />
      </KpiGrid>
      <div className="grid gap-5 lg:grid-cols-2">
        <ChartCard title={ds.profit.waterfall} description={ds.profit.waterfallQ}>
          <WaterfallChart
            steps={data.waterfall.map((s) => ({ key: s.key, label: s.label, value: money(s.value), total: s.total }))}
            format={(v) => compactIdr(v)}
            onStepClick={() => router.push(isHref)}
          />
        </ChartCard>
        <ChartCard title={ds.profit.margins} description={ds.profit.marginsQ}>
          <MonthlyChart
            months={w.months}
            compareMonths={w.compareMonths}
            yFormat={(v) => `${formatNumber(v, 0)}%`}
            valueFormat={(v) => formatPercent(v, 1)}
            series={[
              { key: "gross", label: ds.kpi.gross_margin, color: slotColor(0), type: "line", values: margin("gross_profit") },
              { key: "operating", label: ds.kpi.operating_margin, color: slotColor(1), type: "line", values: margin("operating_income") },
              { key: "net", label: ds.kpi.net_margin, color: slotColor(2), type: "line", values: margin("net_income"), compare: margin("net_income", true) },
            ]}
          />
        </ChartCard>
      </div>
      <div className="grid gap-5 lg:grid-cols-3">
        <ChartCard title={ds.profit.revenueClients} description={ds.profit.revenueClientsQ}>
          <HBarChart
            items={ranked(data.revenueClients, 8, (a) => `/erp/finance/receivables?q=${encodeURIComponent(a.key)}`)}
            format={(v) => compactIdr(v)}
            showShare
            emptyLabel={ds.noData}
          />
        </ChartCard>
        <ChartCard title={ds.profit.expenseGroups} description={ds.profit.expenseGroupsQ}>
          <HBarChart items={ranked(data.expenseGroups, 7)} format={(v) => compactIdr(v)} showShare color={slotColor(1)} emptyLabel={ds.noMatch} />
        </ChartCard>
        <ChartCard title={ds.profit.expenseAccounts}>
          <HBarChart
            items={ranked(data.expenseAccounts, 8, (a) => ledgerHref(a.key, filters.from, filters.to))}
            format={(v) => compactIdr(v)}
            color={slotColor(1)}
            emptyLabel={ds.noMatch}
          />
        </ChartCard>
      </div>
      <ChartCard title={ds.profit.expenseAccounts} description={`${ds.vs(formatDate(w.compareFrom), formatDate(w.compareTo))}`}>
        <SortTable
          rows={data.expenseAccounts}
          columns={changeCols}
          rowKey={(a) => a.key}
          rowHref={(a) => ledgerHref(a.key, filters.from, filters.to)}
          initialSort={{ key: "value", desc: true }}
          empty={ds.noMatch}
        />
      </ChartCard>
      <ChartCard title={ds.profit.mom} description={ds.profit.momQ}>
        <SortTable
          rows={data.accountMonths}
          columns={monthCols}
          rowKey={(s) => s.key}
          rowHref={(s) => ledgerHref(s.key, w.months[0] + "-01", filters.to)}
          empty={ds.noMatch}
        />
      </ChartCard>
    </>
  );
}

// ─── Arus kas ───────────────────────────────────────────────

export function CashTab({ data, filters }: Props<FinanceCash>) {
  const router = useRouter();
  const k = kpiMap(data.kpis);
  const w = data.window;
  const onMonth = useMonthJournals(w.months);
  const cfHref = `/erp/finance/reports?r=cash-flow&${rangeQs(filters.from, filters.to)}`;
  const flow = (key: string, i: number, compare = false): MonthSeries => {
    const s = seriesOf(data.flows, key);
    return { key, label: s?.label ?? key, color: slotColor(i), type: "bar", values: nums(s?.values ?? []), compare: compare ? nums(s?.compareValues ?? []) : undefined };
  };
  const net = seriesOf(data.flows, "net");
  const totals = data.accounts.reduce(
    (t, a) => ({ opening: t.opening + money(a.opening), inflow: t.inflow + money(a.inflow), outflow: t.outflow + money(a.outflow), closing: t.closing + money(a.closing) }),
    { opening: 0, inflow: 0, outflow: 0, closing: 0 },
  );
  type Acc = FinanceCash["accounts"][number];
  const cols: Column<Acc>[] = [
    { key: "name", label: ds.filters.cashAccount, render: (a) => <span><span className="mr-1.5 font-mono text-xs text-muted-foreground">{a.code}</span>{a.name}</span>, sort: (a) => a.code },
    { key: "opening", label: ds.cash.opening, align: "right", render: (a) => exactIdr(money(a.opening)), sort: (a) => money(a.opening), total: exactIdr(totals.opening) },
    { key: "inflow", label: ds.cash.inflow, align: "right", render: (a) => <span className="text-positive">{exactIdr(money(a.inflow))}</span>, sort: (a) => money(a.inflow), total: exactIdr(totals.inflow) },
    { key: "outflow", label: ds.cash.outflow, align: "right", render: (a) => <span className="text-negative">{exactIdr(money(a.outflow))}</span>, sort: (a) => money(a.outflow), total: exactIdr(totals.outflow) },
    { key: "closing", label: ds.cash.closing, align: "right", render: (a) => exactIdr(money(a.closing)), sort: (a) => money(a.closing), total: exactIdr(totals.closing) },
  ];
  return (
    <>
      <KpiGrid cols={4}>
        <FinanceKpi kpi={k.cash} href="/erp/finance/cash-bank" />
        <FinanceKpi kpi={k.net_change} />
        <FinanceKpi kpi={k.operating} href={cfHref} />
        <FinanceKpi kpi={k.investing} href={cfHref} />
        <FinanceKpi kpi={k.financing} href={cfHref} />
        <FinanceKpi kpi={k.burn_rate} lowerIsBetter />
        <FinanceKpi kpi={k.runway} unit="months" />
      </KpiGrid>
      <div className="grid gap-5 xl:grid-cols-2">
        <ChartCard title={ds.cash.flows} description={ds.cash.flowsQ}>
          <MonthlyChart
            months={w.months}
            compareMonths={w.compareMonths}
            stacked
            onMonthClick={onMonth}
            series={[
              flow("operating", 0),
              flow("investing", 1),
              flow("financing", 2),
              flow("non_cash", 3),
              { key: "net", label: net?.label ?? "", color: "var(--foreground)", type: "line", values: nums(net?.values ?? []), compare: nums(net?.compareValues ?? []) },
            ]}
          />
        </ChartCard>
        <ChartCard title={ds.cash.bridge} description={ds.cash.bridgeQ}>
          <WaterfallChart
            steps={data.bridge.map((s) => ({ key: s.key, label: s.label, value: money(s.value), total: s.total }))}
            format={(v) => compactIdr(v)}
            onStepClick={() => router.push(cfHref)}
          />
        </ChartCard>
      </div>
      <ChartCard title={ds.cash.balances} description={ds.cash.balancesQ}>
        <MonthlyChart months={w.months} height={240} series={topSeries(data.balances, 7, "line")} />
      </ChartCard>
      <ChartCard title={ds.cash.accounts}>
        <SortTable
          rows={data.accounts}
          columns={cols}
          rowKey={(a) => a.accountId}
          rowHref={(a) => `/erp/finance/cash-bank/${a.accountId}?${rangeQs(filters.from, filters.to)}`}
          initialSort={{ key: "closing", desc: true }}
        />
      </ChartCard>
    </>
  );
}

// ─── Piutang & hutang ───────────────────────────────────────

const BUCKETS = ["current", "1_30", "31_60", "61_90", "90_plus"];

function partyCols(kindLabel: string): Column<PartyBalance>[] {
  return [
    { key: "party", label: kindLabel, render: (p) => p.partyName, sort: (p) => p.partyName, className: "min-w-44" },
    { key: "outstanding", label: ds.arap.outstanding, align: "right", render: (p) => exactIdr(money(p.outstanding)), sort: (p) => money(p.outstanding) },
    {
      key: "overdue",
      label: ds.arap.overdueAmt,
      align: "right",
      render: (p) => <span className={money(p.overdue) > 0 ? "text-negative" : "text-muted-foreground"}>{exactIdr(money(p.overdue))}</span>,
      sort: (p) => money(p.overdue),
    },
    { key: "count", label: ds.arap.invoices, align: "right", render: (p) => formatNumber(p.invoiceCount), sort: (p) => p.invoiceCount },
    { key: "oldest", label: ds.arap.oldest, align: "right", render: (p) => (p.oldestDays ? ds.arap.days(p.oldestDays) : "–"), sort: (p) => p.oldestDays },
    { key: "billed", label: ds.arap.billed, align: "right", render: (p) => exactIdr(money(p.billed)), sort: (p) => money(p.billed) },
  ];
}

const invoiceHref = (kind: "AR" | "AP", q: string) =>
  `/erp/finance/${kind === "AR" ? "receivables" : "payables"}?q=${encodeURIComponent(q)}`;

export function ArApTab({ data }: Props<FinanceArAp>) {
  const k = kpiMap(data.kpis);
  const w = data.window;
  const bucketColor = (b: string) => rampColor(BUCKETS.indexOf(b), BUCKETS.length);
  const agingRow = (kind: "AR" | "AP") => {
    const segs = data.aging
      .filter((a) => a.kind === kind)
      .map((a) => ({
        key: a.bucket,
        label: `${ds.arap.buckets[a.bucket]} (${a.count})`,
        value: money(a.amount),
        color: bucketColor(a.bucket),
        href: `/erp/finance/${kind === "AR" ? "receivables" : "payables"}?tab=aging&asOf=${w.to}`,
      }));
    return { key: kind, label: kind === "AR" ? ds.arap.receivable : ds.arap.payable, total: segs.reduce((s, x) => s + x.value, 0), segments: segs };
  };
  const agingSeries = (list: AnalyticsSeries[]): MonthSeries[] =>
    BUCKETS.map((b) => {
      const s = list.find((x) => x.key === b);
      return { key: b, label: ds.arap.buckets[b] ?? b, color: bucketColor(b), type: "bar" as const, values: nums(s?.values ?? []) };
    });
  const dso = seriesOf(data.days, "dso");
  const dpo = seriesOf(data.days, "dpo");
  const overdueCols: Column<OverdueInvoice>[] = [
    { key: "no", label: ds.arap.invoiceNo, render: (o) => <span className="font-mono text-xs">{o.invoiceNo}</span>, sort: (o) => o.invoiceNo },
    { key: "kind", label: ds.arap.kind, render: (o) => (o.kind === "AR" ? ds.arap.receivable : ds.arap.payable), sort: (o) => o.kind },
    { key: "party", label: ds.arap.party, render: (o) => o.partyName, sort: (o) => o.partyName, className: "min-w-40" },
    { key: "due", label: ds.arap.dueDate, render: (o) => formatDate(o.dueDate), sort: (o) => o.dueDate },
    {
      key: "late",
      label: ds.arap.daysLate,
      align: "right",
      render: (o) => <StatusBadge kind={o.daysOverdue > 60 ? "critical" : o.daysOverdue > 30 ? "warning" : "neutral"}>{ds.arap.days(o.daysOverdue)}</StatusBadge>,
      sort: (o) => o.daysOverdue,
    },
    {
      key: "outstanding",
      label: ds.arap.outstanding,
      align: "right",
      render: (o) => exactIdr(money(o.outstanding)),
      sort: (o) => money(o.outstanding),
      total: exactIdr(data.overdue.reduce((s, o) => s + money(o.outstanding), 0)),
    },
  ];
  return (
    <>
      <KpiGrid cols={6}>
        <FinanceKpi kpi={k.receivable} lowerIsBetter href="/erp/finance/receivables?tab=aging" />
        <FinanceKpi kpi={k.overdue_receivable} lowerIsBetter href="/erp/finance/receivables?tab=aging" />
        <FinanceKpi kpi={k.dso} unit="days" lowerIsBetter />
        <FinanceKpi kpi={k.payable} href="/erp/finance/payables?tab=aging" />
        <FinanceKpi kpi={k.overdue_payable} lowerIsBetter href="/erp/finance/payables?tab=aging" />
        <FinanceKpi kpi={k.dpo} unit="days" />
      </KpiGrid>
      <div className="grid gap-5 lg:grid-cols-2">
        <ChartCard title={ds.arap.aging} description={ds.arap.agingQ}>
          <StackRows rows={[agingRow("AR"), agingRow("AP")]} legend={BUCKETS.map((b) => ({ key: b, label: ds.arap.buckets[b]!, color: bucketColor(b) }))} />
        </ChartCard>
        <ChartCard title={ds.arap.daysTitle} description={ds.arap.daysQ}>
          <MonthlyChart
            months={w.months}
            compareMonths={w.compareMonths}
            height={240}
            yFormat={(v) => `${formatNumber(v, 0)}`}
            valueFormat={(v) => ds.arap.days(Math.round(v))}
            series={[
              { key: "dso", label: "DSO", color: slotColor(0), type: "line", values: nums(dso?.values ?? []), compare: nums(dso?.compareValues ?? []) },
              { key: "dpo", label: "DPO", color: slotColor(1), type: "line", values: nums(dpo?.values ?? []) },
            ]}
          />
        </ChartCard>
      </div>
      <div className="grid gap-5 lg:grid-cols-2">
        <ChartCard title={ds.arap.agingTrend} description={ds.arap.agingTrendQ}>
          <MonthlyChart months={w.months} stacked height={240} series={agingSeries(data.arAgingTrend)} />
        </ChartCard>
        <ChartCard title={ds.arap.apTrend}>
          <MonthlyChart months={w.months} stacked height={240} series={agingSeries(data.apAgingTrend)} />
        </ChartCard>
      </div>
      <div className="grid gap-5 xl:grid-cols-2">
        <ChartCard title={ds.arap.customers}>
          <SortTable
            rows={data.customers}
            columns={partyCols(ds.arap.party)}
            rowKey={(p) => p.partyName}
            rowHref={(p) => invoiceHref("AR", p.partyName)}
            initialSort={{ key: "outstanding", desc: true }}
            limit={10}
          />
        </ChartCard>
        <ChartCard title={ds.arap.vendors}>
          <SortTable
            rows={data.vendors}
            columns={partyCols(ds.arap.party)}
            rowKey={(p) => p.partyName}
            rowHref={(p) => invoiceHref("AP", p.partyName)}
            initialSort={{ key: "outstanding", desc: true }}
            limit={10}
          />
        </ChartCard>
      </div>
      <ChartCard title={ds.arap.overdue} description={ds.arap.overdueQ}>
        <SortTable
          rows={data.overdue}
          columns={overdueCols}
          rowKey={(o) => o.id}
          rowHref={(o) => invoiceHref(o.kind, o.invoiceNo)}
          initialSort={{ key: "late", desc: true }}
        />
      </ChartCard>
    </>
  );
}

// ─── Beban & reimburse ──────────────────────────────────────

export function SpendTab({ data, filters }: Props<FinanceSpend>) {
  const k = kpiMap(data.kpis);
  const w = data.window;
  const onMonth = useMonthJournals(w.months);
  const month = filters.from.slice(0, 7) === filters.to.slice(0, 7) ? filters.from.slice(0, 7) : undefined;
  const claimsHref = (extra: Record<string, string | undefined>) => {
    const sp = new URLSearchParams({ tab: "all" });
    for (const [key, v] of Object.entries({ ...extra, month })) if (v) sp.set(key, v);
    return `/erp/finance/reimbursements?${sp}`;
  };
  const named = (list: AnalyticsAmount[], names: Record<string, string>) => list.map((a) => ({ ...a, label: names[a.key] ?? a.label }));
  const statusSegs: StackSegment[] = named(data.byStatus, ds.spend.statusNames).map((a, i) => ({
    key: a.key,
    label: `${a.label} (${a.count})`,
    value: money(a.value),
    color: slotColor(i),
    href: claimsHref({ status: a.key }),
  }));
  const lead = seriesOf(data.leadTimeTrend, "lead_days");
  const leadBench = data.leadTime.find((l) => l.key === "total");
  const pendingCols: Column<ClaimSummary>[] = [
    { key: "no", label: ds.spend.claimNo, render: (c) => <span className="font-mono text-xs">{c.claimNo}</span>, sort: (c) => c.claimNo },
    { key: "employee", label: ds.spend.employee, render: (c) => c.employeeName, sort: (c) => c.employeeName, className: "min-w-36" },
    { key: "dept", label: ds.filters.department, render: (c) => c.departmentName || "–", sort: (c) => c.departmentName },
    { key: "category", label: ds.spend.category, render: (c) => ds.spend.categoryNames[c.category] ?? c.category, sort: (c) => c.category },
    { key: "status", label: ds.spend.status, render: (c) => <StatusBadge kind="warning">{ds.spend.statusNames[c.status] ?? c.status}</StatusBadge>, sort: (c) => c.status },
    { key: "age", label: ds.spend.age, align: "right", render: (c) => ds.arap.days(c.ageDays), sort: (c) => c.ageDays },
    {
      key: "amount",
      label: ds.spend.amount,
      align: "right",
      render: (c) => exactIdr(money(c.amount)),
      sort: (c) => money(c.amount),
      total: exactIdr(data.pending.reduce((s, c) => s + money(c.amount), 0)),
    },
  ];
  return (
    <>
      <div className="flex flex-wrap items-center gap-2">
        <SelectFilter
          param="department"
          label={ds.filters.department}
          allLabel={ds.filters.allDepartments}
          options={data.departments.map((d) => ({ value: d, label: d }))}
        />
      </div>
      <KpiGrid cols={6}>
        <FinanceKpi kpi={k.expense} lowerIsBetter />
        <FinanceKpi kpi={k.reimburse_paid} href={claimsHref({ status: "PAID" })} />
        <FinanceKpi kpi={k.reimburse_pending} href="/erp/finance/reimbursements" />
        <FinanceKpi kpi={k.claims} unit="count" />
        <FinanceKpi kpi={k.approval_days} unit="days" lowerIsBetter />
        <FinanceKpi kpi={k.rejection_rate} unit="pct" lowerIsBetter />
      </KpiGrid>
      <div className="grid gap-5 lg:grid-cols-2">
        <ChartCard title={ds.spend.groups} description={ds.spend.groupsQ}>
          <MonthlyChart months={w.months} stacked onMonthClick={onMonth} series={topSeries(data.expenseGroups, 6)} />
        </ChartCard>
        <ChartCard title={ds.spend.categories} description={ds.spend.categoriesQ}>
          <MonthlyChart
            months={w.months}
            stacked
            series={data.claimCategories.map((s, i) => ({ key: s.key, label: ds.spend.categoryNames[s.key] ?? s.label, color: slotColor(i), type: "bar" as const, values: nums(s.values) }))}
          />
        </ChartCard>
      </div>
      <div className="grid gap-5 lg:grid-cols-3">
        <ChartCard title={ds.spend.byCategory}>
          <HBarChart
            items={named(data.byCategory, ds.spend.categoryNames).map((a) => ({ label: `${a.label} · ${a.count}`, value: money(a.value), href: claimsHref({ category: a.key }) }))}
            format={(v) => compactIdr(v)}
            showShare
            emptyLabel={ds.noData}
          />
        </ChartCard>
        <ChartCard title={ds.spend.byDepartment}>
          <HBarChart
            items={data.byDepartment.map((a) => ({ label: `${a.label} · ${a.count}`, value: money(a.value) }))}
            format={(v) => compactIdr(v)}
            showShare
            color={slotColor(1)}
            emptyLabel={ds.noData}
          />
        </ChartCard>
        <ChartCard title={ds.spend.byStatus} description={ds.spend.byStatusQ}>
          <StackRows
            rows={[{ key: "status", label: ds.spend.count, total: statusSegs.reduce((s, x) => s + x.value, 0), segments: statusSegs }]}
            legend={statusSegs}
          />
        </ChartCard>
      </div>
      <div className="grid gap-5 lg:grid-cols-2">
        <ChartCard title={ds.spend.leadTime} description={ds.spend.leadTimeQ}>
          <HBarChart
            items={data.leadTime.map((l) => ({ label: `${l.label} · ${l.count}`, value: num(l.value) ?? 0, color: l.key === "total" ? slotColor(0) : undefined }))}
            format={(v) => `${formatNumber(v, 1)} ${ds.units.days}`}
            color={slotColor(3)}
            emptyLabel={ds.noData}
          />
        </ChartCard>
        <ChartCard title={ds.spend.leadTrend}>
          <MonthlyChart
            months={w.months}
            height={220}
            yFormat={(v) => formatNumber(v, 0)}
            valueFormat={(v) => `${formatNumber(v, 1)} ${ds.units.days}`}
            referenceY={num(leadBench?.value)}
            referenceLabel={ds.spend.avgDays}
            series={[{ key: "lead", label: ds.spend.leadTrend, color: slotColor(0), type: "line", values: (lead?.values ?? []).map((v) => num(v) ?? 0) }]}
          />
        </ChartCard>
      </div>
      <ChartCard title={ds.spend.pending} description={ds.spend.pendingQ}>
        <SortTable
          rows={data.pending}
          columns={pendingCols}
          rowKey={(c) => c.id}
          rowHref={(c) => `/erp/finance/reimbursements/${c.id}`}
          initialSort={{ key: "age", desc: true }}
        />
      </ChartCard>
    </>
  );
}
