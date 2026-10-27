"use client";

import { ChartCard } from "@/components/dashboard/chart-card";
import { slotColor } from "@/components/dashboard/charts/chart-primitives";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { formatRupiah } from "@/features/erp/shared/money";
import { formatNumber, formatPercent } from "@/lib/format";
import { findKpi, monthLabel, rpCompact, sum } from "../format";
import { listHref, monthFilter, periodFilter, SalesDrillSheet, useSalesDrill, baseFilter } from "../drill";
import { dd } from "../strings";
import type { DashboardQuery, DashboardSlice, DashboardSummary } from "../types";
import { MonthlyChart } from "./charts";
import { DetailTable } from "./detail-table";
import { KpiRow } from "./kpi";

const s = dd.summary;

export function SummaryView({ data, query }: { data: DashboardSummary; query: DashboardQuery }) {
  const { period } = data;
  const won = data.trend.find((t) => t.key === "won_value");
  const target = data.trend.find((t) => t.key === "target");
  const open = useSalesDrill(query);
  const base = baseFilter(query);
  const pipelineHref = listHref("deal", { ...base, status: "open" });
  const wonHref = listHref("deal", { ...base, ...periodFilter(query), status: "won" });
  const closedHref = listHref("deal", { ...base, ...periodFilter(query) });
  const openStage = (st: DashboardSlice) => open("deal", { status: "open", stage: st.key, l: st.label });
  const compareLabel = period.compare === "year" ? dd.compare.year : dd.compare.previous;
  const totalValue = sum(data.stages.map((x) => x.value));
  return (
    <>
      <KpiRow
        kpis={data.kpis}
        keys={["pipeline_value", "weighted_value", "won_value", "win_rate", "avg_deal", "cycle_days"]}
        compare={period.compare}
        hrefs={{ pipeline_value: pipelineHref, weighted_value: pipelineHref, won_value: wonHref, win_rate: closedHref, avg_deal: wonHref, cycle_days: wonHref }}
      />
      <div className="grid gap-(--gap) xl:grid-cols-3">
        <ChartCard
          className="xl:col-span-2"
          title={s.wonVsTarget}
          description={`${s.wonVsTargetHint} ${dd.trendRange(monthLabel(period.months[0]!), monthLabel(period.months.at(-1)!))}.`}
          table={
            <DetailTable
              rows={period.months.map((m, i) => ({ m, won: won?.values[i] ?? 0, target: target?.values[i] ?? 0, prev: won?.previous[i] ?? 0 }))}
              rowKey={(r) => r.m}
              columns={[
                { key: "m", label: "Bulan", render: (r) => monthLabel(r.m) },
                { key: "won", label: s.won, numeric: true, render: (r) => formatRupiah(r.won), total: (rs) => formatRupiah(sum(rs.map((r) => r.won))) },
                { key: "target", label: s.target, numeric: true, render: (r) => formatRupiah(r.target), total: (rs) => formatRupiah(sum(rs.map((r) => r.target))) },
                { key: "prev", label: compareLabel, numeric: true, render: (r) => formatRupiah(r.prev), optional: true },
              ]}
            />
          }
        >
          <MonthlyChart
            months={period.months}
            yFormat={rpCompact}
            series={[
              { key: "won", label: s.won, values: won?.values ?? [], format: formatRupiah },
              { key: "target", label: s.target, type: "line", color: "var(--foreground)", values: target?.values ?? [], format: formatRupiah },
            ]}
            previous={won?.previous.length ? { label: compareLabel, values: won.previous, format: formatRupiah } : undefined}
            onMonth={(i) => open("deal", { ...monthFilter(period.months[i]!), date: "closed", status: "won", l: `${s.won} · ${monthLabel(period.months[i]!)}` })}
          />
        </ChartCard>
        <ChartCard title={s.stages} description={s.stagesHint}>
          <HBarChart
            items={data.stages.map((st) => ({
              label: st.label,
              value: st.value,
              secondary: `${st.count} deal`,
              key: st.key,
            }))}
            onSelect={(_it, i) => openStage(data.stages[i]!)}
            format={rpCompact}
            color={slotColor(0)}
            emptyLabel={dd.empty}
          />
          <p className="mt-4 text-xs text-muted-foreground">
            {s.weighted}: <span className="font-medium text-foreground tabular">{formatRupiah(findKpi(data.kpis, "weighted_value").value)}</span>
          </p>
        </ChartCard>
      </div>
      <ChartCard title={s.stageTable}>
        <DetailTable<DashboardSlice>
          rows={data.stages}
          rowKey={(r) => r.key}
          onRow={openStage}
          columns={[
            { key: "label", label: s.stage, render: (r) => r.label },
            { key: "count", label: s.deals, numeric: true, render: (r) => formatNumber(r.count), sort: (r) => r.count, total: (rs) => formatNumber(sum(rs.map((r) => r.count))) },
            { key: "value", label: s.value, numeric: true, render: (r) => formatRupiah(r.value), sort: (r) => r.value, total: (rs) => formatRupiah(sum(rs.map((r) => r.value))) },
            {
              key: "extra",
              label: s.weighted,
              numeric: true,
              render: (r) => formatRupiah(r.extra),
              sort: (r) => r.extra,
              total: (rs) => formatRupiah(sum(rs.map((r) => r.extra))),
              optional: true,
            },
            { key: "share", label: s.share, numeric: true, render: (r) => formatPercent(totalValue ? (r.value / totalValue) * 100 : 0, 0), optional: true },
          ]}
        />
      </ChartCard>
      <SalesDrillSheet query={query} />
    </>
  );
}
