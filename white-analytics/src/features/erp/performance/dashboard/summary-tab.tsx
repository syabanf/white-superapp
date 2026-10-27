"use client";

import { ChartCard } from "@/components/dashboard/chart-card";
import { KpiGrid } from "@/components/dashboard/kpi-tile";
import { DeltaBadge } from "@/components/dashboard/delta-badge";
import { slotColor } from "@/components/dashboard/charts/chart-primitives";
import { LinkedTrendChart } from "./charts";
import { href, int, scoreDelta, scoreDomain, monthOf, monthTick, pct, score } from "./lib";
import { ds } from "./strings";
import type { DashSummary } from "./types";
import { MetricTile, SortTable } from "./widgets";

const s = ds.summary;
const P = "/erp/performance";

/** Most departments the multi-line chart draws before the rest fold into the table. */
const MAX_LINES = 8;

export function SummaryTab({ data, dept }: { data: DashSummary; dept?: string }) {
  const w = data.window;
  const month = monthOf(w.to);
  const depts = [...new Set(data.departmentTrend.map((d) => d.departmentId))];
  const names = Object.fromEntries(data.departmentTrend.map((d) => [d.departmentId, d.departmentName]));
  const lines = depts.slice(0, MAX_LINES);
  const trend = data.months.map((m) => {
    const row: Record<string, string | number | null> = { month: m };
    for (const d of data.departmentTrend) if (d.period === m) row[d.departmentId] = d.score ?? null;
    return row;
  });
  const totals = data.departments.reduce(
    (a, r) => ({
      members: a.members + r.members,
      overdue: a.overdue + r.overdueTasks,
      imp: a.imp + r.openImprovements,
    }),
    { members: 0, overdue: 0, imp: 0 },
  );

  return (
    <div className="page-stack">
      <KpiGrid cols={6} className="xl:grid-cols-3">
        <MetricTile
          label={s.kpi}
          hint={s.kpiHint}
          metric={data.kpiScore}
          format={score}
          href={href(`${P}/kpi`, { month, scope: w.scope === "me" ? "me" : "all" })}
        />
        <MetricTile
          label={s.completion}
          hint={s.completionHint}
          metric={data.taskCompletion}
          format={pct}
          href={href(`${P}/tasks`, { tab: "monitor", month, dept })}
        />
        <MetricTile
          label={s.onTime}
          hint={s.onTimeHint}
          metric={data.onTimeRate}
          format={pct}
          href={href(`${P}/tasks`, { tab: "monitor", month, dept })}
        />
        <MetricTile
          label={s.overdue}
          hint={s.overdueHint}
          metric={data.overdueTasks}
          format={int}
          lowerIsBetter
          href={href(`${P}/tasks`, { tab: "monitor", month, dept })}
        />
        <MetricTile
          label={s.logbook}
          hint={s.logbookHint}
          metric={data.logbookRate}
          format={pct}
          href={href(`${P}/logbook`, { tab: "history", dept, from: w.from, to: w.to })}
        />
        <MetricTile
          label={s.improvements}
          hint={s.improvementsHint}
          metric={data.openImprovements}
          format={int}
          lowerIsBetter
          href={href(`${P}/improvements`, { dept, status: "open" })}
        />
      </KpiGrid>

      <ChartCard
        title={s.trend}
        description={s.trendHint}
        table={
          <SortTable
            rows={data.months}
            rowKey={(m) => m}
            columns={[
              { key: "m", label: "Bulan", render: (m) => monthTick(m) },
              ...lines.map((d) => ({
                key: d,
                label: names[d] ?? d,
                align: "right" as const,
                render: (m: string) =>
                  score(data.departmentTrend.find((x) => x.period === m && x.departmentId === d)?.score),
              })),
            ]}
          />
        }
      >
        <LinkedTrendChart
          data={trend}
          xKey="month"
          xFormat={monthTick}
          yFormat={(v) => score(v)}
          yDomain={scoreDomain(data.departmentTrend.map((d) => d.score))}
          height={300}
          series={lines.map((d, i) => ({
            key: d,
            label: names[d] ?? d,
            color: slotColor(i),
            format: (v: number) => score(v),
          }))}
          hrefFor={(row) => href(`${P}/kpi`, { month: String(row.month), scope: "all" })}
          summary={`${s.trend}, ${data.months.length} bulan`}
        />
      </ChartCard>

      <ChartCard title={s.deptTable} description={s.deptTableHint}>
        <SortTable
          rows={data.departments}
          rowKey={(r) => r.departmentId}
          rowHref={(r) => href(`${P}/tasks`, { tab: "monitor", dept: r.departmentId, month })}
          initialSort={{ key: "score", desc: true }}
          columns={[
            {
              key: "name",
              label: ds.filters.department,
              render: (r) => r.departmentName,
              sort: (r) => r.departmentName,
            },
            {
              key: "members",
              label: s.members,
              align: "right",
              render: (r) => int(r.members),
              sort: (r) => r.members,
              total: int(totals.members),
            },
            {
              key: "score",
              label: s.score,
              align: "right",
              render: (r) => score(r.kpiScore),
              sort: (r) => r.kpiScore,
            },
            {
              key: "delta",
              label: s.delta,
              align: "right",
              render: (r) =>
                r.kpiScore != null && r.kpiScorePrevious != null ? (
                  <DeltaBadge delta={scoreDelta(r.kpiScore, r.kpiScorePrevious)} mode="abs" digits={1} />
                ) : (
                  "–"
                ),
              sort: (r) =>
                r.kpiScore != null && r.kpiScorePrevious != null
                  ? r.kpiScore - r.kpiScorePrevious
                  : undefined,
            },
            {
              key: "done",
              label: s.completion,
              align: "right",
              render: (r) => pct(r.taskCompletion),
              sort: (r) => r.taskCompletion,
            },
            {
              key: "ontime",
              label: s.onTime,
              align: "right",
              render: (r) => pct(r.onTimeRate),
              sort: (r) => r.onTimeRate,
            },
            {
              key: "overdue",
              label: s.overdue,
              align: "right",
              render: (r) => int(r.overdueTasks),
              sort: (r) => r.overdueTasks,
              total: int(totals.overdue),
            },
            {
              key: "logbook",
              label: s.logbook,
              align: "right",
              render: (r) => pct(r.logbookRate),
              sort: (r) => r.logbookRate,
            },
            {
              key: "imp",
              label: s.improvements,
              align: "right",
              render: (r) => int(r.openImprovements),
              sort: (r) => r.openImprovements,
              total: int(totals.imp),
            },
          ]}
        />
      </ChartCard>
    </div>
  );
}
