"use client";

import { ChartCard } from "@/components/dashboard/chart-card";
import { slotColor } from "@/components/dashboard/charts/chart-primitives";
import { Heatmap } from "@/components/dashboard/charts/heatmap";
import { formatRupiah } from "@/features/erp/shared/money";
import { formatNumber, formatPercent } from "@/lib/format";
import { DeltaText } from "@/components/dashboard/delta-badge";
import { cn } from "@/lib/utils";
import { baseFilter, listHref, monthFilter, periodFilter, SalesDrillSheet, useSalesDrill } from "../drill";
import { monthLabel, rpCompact, sum } from "../format";
import { dd } from "../strings";
import type { DashboardMember, DashboardQuery, DashboardTeam } from "../types";
import { BulletList, MonthlyChart } from "./charts";
import { DetailTable } from "./detail-table";
import { KpiRow } from "./kpi";

const t = dd.team;

export function TeamView({ data, query }: { data: DashboardTeam; query: DashboardQuery }) {
  const { period } = data;
  const name = (id: string) => data.members.find((m) => m.userId === id)?.name ?? id;
  const open = useSalesDrill(query);
  const closed = periodFilter(query);
  const wonHref = listHref("deal", { ...baseFilter(query), ...closed, status: "won" });
  const byOwner = (id: string) => open("deal", { ...closed, status: "won", owner: id, dim: "owner", l: name(id) });
  const byMonth = (i: number, owner?: string) =>
    open("deal", { ...monthFilter(period.months[i]!), date: "closed", status: "won", owner, l: [owner ? name(owner) : t.won, monthLabel(period.months[i]!)].join(" · ") });
  const series = data.wonByOwner.slice(0, 7);
  return (
    <>
      <KpiRow
        kpis={data.kpis}
        keys={["won_value", "target_value", "attainment", "activities"]}
        compare={period.compare}
        className="xl:grid-cols-4"
        hrefs={{ won_value: wonHref, target_value: "/erp/sales/reports", attainment: "/erp/sales/reports", activities: "/erp/sales/reports" }}
      />
      <div className="grid gap-4 xl:grid-cols-2">
        <ChartCard title={t.attainment} description={t.attainmentHint}>
          <BulletList
            rows={data.members.map((m) => ({ key: m.userId, label: m.name, value: m.wonValue, target: m.targetValue }))}
            onSelect={(r) => byOwner(r.key)}
            format={rpCompact}
            noTarget={t.noTarget}
          />
        </ChartCard>
        <ChartCard title={t.trend} description={t.trendHint}>
          <MonthlyChart
            months={period.months}
            yFormat={rpCompact}
            series={series.map((s, i) => ({ key: s.key, label: name(s.key), type: "line" as const, color: slotColor(i), values: s.values, format: formatRupiah }))}
            onMonth={(i) => byMonth(i)}
          />
        </ChartCard>
      </div>
      <ChartCard title={t.heat} description={t.heatHint}>
        <Heatmap
          rows={series.length}
          cols={period.months.length}
          rowLabels={series.map((s) => name(s.key))}
          colLabels={period.months.map(monthLabel)}
          cells={series.flatMap((s, r) => s.values.map((v, c) => ({ row: r, col: c, value: v })))}
          format={rpCompact}
          valueLabel={t.won}
          cellSize={44}
          onSelect={(cell) => byMonth(cell.col, series[cell.row]?.key)}
        />
      </ChartCard>
      <ChartCard title={t.leaderboard}>
        <DetailTable<DashboardMember>
          rows={data.members}
          rowKey={(r) => r.userId}
          onRow={(r) => byOwner(r.userId)}
          initialSort={{ key: "won", desc: true }}
          columns={[
            { key: "name", label: t.person, render: (r) => r.name, sort: (r) => r.name },
            {
              key: "pipeline",
              label: t.pipeline,
              numeric: true,
              optional: true,
              render: (r) => formatRupiah(r.pipelineValue),
              sort: (r) => r.pipelineValue,
              total: (rs) => formatRupiah(sum(rs.map((r) => r.pipelineValue))),
            },
            {
              key: "won",
              label: t.won,
              numeric: true,
              render: (r) => formatRupiah(r.wonValue),
              sort: (r) => r.wonValue,
              total: (rs) => formatRupiah(sum(rs.map((r) => r.wonValue))),
            },
            {
              key: "change",
              label: t.change,
              numeric: true,
              optional: true,
              render: (r) => (r.previousWonValue ? <DeltaText current={r.wonValue} previous={r.previousWonValue} /> : "–"),
              sort: (r) => (r.previousWonValue ? (r.wonValue - r.previousWonValue) / r.previousWonValue : -Infinity),
            },
            {
              key: "deals",
              label: dd.summary.deals,
              numeric: true,
              optional: true,
              render: (r) => `${r.wonDeals} / ${r.wonDeals + r.lostDeals}`,
              sort: (r) => r.wonDeals,
              total: (rs) => formatNumber(sum(rs.map((r) => r.wonDeals))),
            },
            { key: "rate", label: t.winRate, numeric: true, render: (r) => formatPercent(r.winRate, 0), sort: (r) => r.winRate },
            {
              key: "acts",
              label: t.activities,
              numeric: true,
              optional: true,
              render: (r) => formatNumber(r.activities),
              sort: (r) => r.activities,
              total: (rs) => formatNumber(sum(rs.map((r) => r.activities))),
            },
            {
              key: "target",
              label: t.target,
              numeric: true,
              optional: true,
              render: (r) => (r.targetValue ? formatRupiah(r.targetValue) : "–"),
              sort: (r) => r.targetValue,
              total: (rs) => formatRupiah(sum(rs.map((r) => r.targetValue))),
            },
            {
              key: "attain",
              label: t.attain,
              numeric: true,
              render: (r) =>
                r.targetValue ? (
                  <span className={cn("font-semibold", r.attainmentPercent >= 100 ? "text-positive" : "text-negative")}>
                    {r.attainmentPercent >= 100 ? "▲" : "▼"} {formatPercent(r.attainmentPercent, 0)}
                  </span>
                ) : (
                  "–"
                ),
              sort: (r) => r.attainmentPercent,
            },
          ]}
        />
      </ChartCard>
      <SalesDrillSheet query={query} />
    </>
  );
}
