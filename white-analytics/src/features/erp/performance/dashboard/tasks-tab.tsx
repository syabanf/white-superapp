"use client";

import { ChartCard } from "@/components/dashboard/chart-card";
import { KpiGrid } from "@/components/dashboard/kpi-tile";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { MUTED_SERIES, slotColor } from "@/components/dashboard/charts/chart-primitives";
import { LinkedTrendChart, StackedRows, type Segment } from "./charts";
import { bucketLabel, bucketTick, href, int, monthOf, pct } from "./lib";
import { ds } from "./strings";
import type { DashTasks } from "./types";
import { MetricTile, SortTable } from "./widgets";

const t = ds.tasks;
const P = "/erp/performance/tasks";

const SEGMENTS: Segment[] = [
  { key: "approved", label: t.statuses.approved, color: slotColor(0) },
  { key: "waitingReview", label: t.statuses.waitingReview, color: slotColor(1) },
  { key: "inProgress", label: t.statuses.inProgress, color: slotColor(2) },
  { key: "pending", label: t.statuses.pending, color: slotColor(3) },
  { key: "rejected", label: t.statuses.rejected, color: slotColor(4) },
];

const AGING_RAMP = ["var(--seq-300)", "var(--seq-400)", "var(--seq-500)", "var(--seq-600)", "var(--seq-700)"];

const hours = (v: number | undefined) => (v == null ? "–" : t.hours(v));

export function TasksTab({ data, dept }: { data: DashTasks; dept?: string }) {
  const w = data.window;
  const month = monthOf(w.to);
  const hasPrev = !!w.compareFrom;
  const trend = data.trend.map((b) => ({
    bucket: b.bucket,
    completion: b.completion != null ? b.completion * 100 : null,
    onTime: b.onTimeRate != null ? b.onTimeRate * 100 : null,
    prevCompletion: b.previousCompletion != null ? b.previousCompletion * 100 : null,
  }));
  const p100 = (v: number) => `${Math.round(v)}%`;
  const monitor = (deptId?: string, m = month) => href(P, { tab: "monitor", dept: deptId ?? dept, month: m });
  const turnTotal = data.turnaround.reduce((a, c) => a + c.value, 0);
  const sum = (pick: (r: DashTasks["people"][number]) => number) =>
    int(data.people.reduce((a, r) => a + pick(r), 0));

  return (
    <div className="flex flex-col gap-4 sm:gap-5">
      <KpiGrid cols={6} className="xl:grid-cols-3">
        <MetricTile label={t.due} metric={data.due} format={int} href={monitor()} />
        <MetricTile
          label={t.completion}
          hint={ds.summary.completionHint}
          metric={data.completion}
          format={pct}
        />
        <MetricTile label={t.onTime} hint={ds.summary.onTimeHint} metric={data.onTimeRate} format={pct} />
        <MetricTile
          label={t.overdue}
          hint={ds.summary.overdueHint}
          metric={data.overdue}
          format={int}
          lowerIsBetter
          href={monitor()}
        />
        <MetricTile
          label={t.waiting}
          metric={data.waitingReview}
          format={int}
          lowerIsBetter
          href={monitor()}
        />
        <MetricTile
          label={t.turnaround}
          hint={t.turnaroundHint}
          metric={data.turnaroundHours}
          format={hours}
          lowerIsBetter
        />
      </KpiGrid>

      <ChartCard
        title={t.trend}
        description={t.trendHint}
        table={
          <SortTable
            rows={data.trend}
            rowKey={(b) => b.bucket}
            columns={[
              { key: "b", label: "Periode", render: (b) => bucketLabel(b.bucket, w.granularity) },
              {
                key: "due",
                label: t.due,
                align: "right",
                render: (b) => int(b.due),
                total: sum((r) => r.due),
              },
              {
                key: "ok",
                label: t.approved,
                align: "right",
                render: (b) => int(b.approved),
                total: sum((r) => r.approved),
              },
              { key: "c", label: t.completion, align: "right", render: (b) => pct(b.completion) },
              { key: "o", label: t.onTime, align: "right", render: (b) => pct(b.onTimeRate) },
              { key: "h", label: t.turnaround, align: "right", render: (b) => hours(b.turnaroundHours) },
            ]}
          />
        }
      >
        <LinkedTrendChart
          data={trend}
          xKey="bucket"
          xFormat={(v) => bucketTick(v, w.granularity)}
          yFormat={p100}
          yDomain={[0, 100]}
          height={280}
          series={[
            { key: "completion", label: t.completion, color: slotColor(0), format: p100 },
            { key: "onTime", label: t.onTime, color: slotColor(1), format: p100 },
            ...(hasPrev
              ? [
                  {
                    key: "prevCompletion",
                    label: `${t.completion} (${ds.previous.toLowerCase()})`,
                    color: MUTED_SERIES,
                    dashed: true,
                    format: p100,
                  },
                ]
              : []),
          ]}
          hrefFor={(row) => monitor(undefined, String(row.bucket).slice(0, 7))}
          summary={t.trend}
        />
      </ChartCard>

      <div className="grid gap-4 sm:gap-5 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
        <ChartCard title={t.byDept} description={t.byDeptHint}>
          <StackedRows
            segments={SEGMENTS}
            rows={data.departments.map((d) => ({
              label: d.departmentName,
              href: monitor(d.departmentId),
              values: {
                approved: d.approved,
                waitingReview: d.waitingReview,
                inProgress: d.inProgress,
                pending: d.pending,
                rejected: d.rejected,
              },
            }))}
          />
        </ChartCard>
        <ChartCard title={t.aging} description={t.agingHint}>
          <HBarChart
            format={(v) => `${int(v)} tugas`}
            items={data.aging.map((a, i) => ({
              label: t.agingKeys[a.key] ?? a.key,
              value: a.value,
              color: AGING_RAMP[i],
              href: monitor(),
            }))}
          />
        </ChartCard>
      </div>

      <div className="grid gap-4 sm:gap-5 lg:grid-cols-2">
        <ChartCard title={t.workload} description={t.workloadHint}>
          <HBarChart
            maxItems={10}
            format={(v) => `${int(v)} tugas`}
            items={data.people.map((p) => ({ label: p.name, value: p.due, secondary: pct(p.completion) }))}
          />
        </ChartCard>
        <ChartCard title={t.turnaroundDist} description={t.turnaroundDistHint}>
          <HBarChart
            format={(v) => int(v)}
            items={data.turnaround.map((c) => ({
              label: t.turnaroundKeys[c.key] ?? c.key,
              value: c.value,
              secondary: turnTotal ? pct(c.value / turnTotal) : undefined,
            }))}
          />
        </ChartCard>
      </div>

      <ChartCard title={t.monitor}>
        <SortTable
          rows={data.people}
          rowKey={(p) => p.employeeId}
          initialSort={{ key: "overdue", desc: true }}
          limit={25}
          columns={[
            { key: "n", label: ds.filters.employee, render: (p) => p.name, sort: (p) => p.name },
            {
              key: "d",
              label: ds.filters.department,
              render: (p) => p.departmentName,
              sort: (p) => p.departmentName,
            },
            {
              key: "due",
              label: t.due,
              align: "right",
              render: (p) => int(p.due),
              sort: (p) => p.due,
              total: sum((r) => r.due),
            },
            {
              key: "c",
              label: t.completion,
              align: "right",
              render: (p) => pct(p.completion),
              sort: (p) => p.completion,
            },
            {
              key: "overdue",
              label: t.overdue,
              align: "right",
              render: (p) => int(p.overdue),
              sort: (p) => p.overdue,
              total: sum((r) => r.overdue),
            },
            {
              key: "w",
              label: t.waiting,
              align: "right",
              render: (p) => int(p.waitingReview),
              sort: (p) => p.waitingReview,
              total: sum((r) => r.waitingReview),
            },
            {
              key: "r",
              label: t.statuses.rejected,
              align: "right",
              render: (p) => int(p.rejected),
              sort: (p) => p.rejected,
              total: sum((r) => r.rejected),
            },
          ]}
        />
      </ChartCard>
    </div>
  );
}
