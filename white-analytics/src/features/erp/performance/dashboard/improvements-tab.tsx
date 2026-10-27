"use client";

import { ChartCard } from "@/components/dashboard/chart-card";
import { KpiGrid } from "@/components/dashboard/kpi-tile";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { slotColor } from "@/components/dashboard/charts/chart-primitives";
import { formatDate } from "@/lib/format";
import { sp } from "../strings";
import { LinkedTrendChart } from "./charts";
import { bucketLabel, bucketTick, href, int, pct } from "./lib";
import { ds } from "./strings";
import type { DashImprovements } from "./types";
import { MetricTile, SortTable } from "./widgets";

const m = ds.improvements;
const P = "/erp/performance/improvements";
const GROUPS = ["manual", "ai", "retro", "automatic"] as const;

export function ImprovementsTab({ data, dept }: { data: DashImprovements; dept?: string }) {
  const w = data.window;
  const list = (extra: Record<string, string | undefined> = {}) => href(P, { dept, ...extra });
  const trend = data.trend.map((b) => ({ ...b, created: b.manual + b.ai + b.retro + b.automatic }));
  const sum = (pick: (r: DashImprovements["departments"][number]) => number) =>
    int(data.departments.reduce((a, r) => a + pick(r), 0));

  return (
    <div className="page-stack">
      <KpiGrid cols={5}>
        <MetricTile
          label={m.open}
          hint={ds.summary.improvementsHint}
          metric={data.open}
          format={int}
          lowerIsBetter
          href={list({ status: "open" })}
        />
        <MetricTile label={m.created} metric={data.created} format={int} href={list()} />
        <MetricTile
          label={m.completed}
          metric={data.completed}
          format={int}
          href={list({ status: "done" })}
        />
        <MetricTile
          label={m.overdue}
          hint={m.overdueHint}
          metric={data.overdue}
          format={int}
          lowerIsBetter
          href={list({ status: "open" })}
        />
        <MetricTile
          label={m.ai}
          hint={m.aiHint}
          metric={data.aiAcceptance}
          format={pct}
          href={list({ source: "ai" })}
        />
      </KpiGrid>

      <ChartCard
        title={m.trend}
        description={`${m.trendHint} ${m.groupsHint}`}
        table={
          <SortTable
            rows={trend}
            rowKey={(b) => b.bucket}
            columns={[
              { key: "b", label: "Periode", render: (b) => bucketLabel(b.bucket, w.granularity) },
              ...GROUPS.map((g) => ({
                key: g,
                label: m.groups[g],
                align: "right" as const,
                render: (b: (typeof trend)[number]) => int(b[g]),
              })),
              { key: "done", label: m.completed, align: "right", render: (b) => int(b.completed) },
            ]}
          />
        }
      >
        <LinkedTrendChart
          data={trend}
          xKey="bucket"
          xFormat={(v) => bucketTick(v, w.granularity)}
          yFormat={(v) => int(v)}
          stacked
          height={280}
          series={[
            ...GROUPS.map((g, i) => ({
              key: g,
              label: m.groups[g],
              type: "bar" as const,
              color: slotColor(i),
            })),
            { key: "completed", label: m.completed, color: "var(--foreground)" },
          ]}
          hrefFor={() => list()}
          summary={m.trend}
        />
      </ChartCard>

      <div className="grid gap-(--gap) lg:grid-cols-3">
        <ChartCard title={m.aiTrend} description={m.aiTrendHint} className="lg:col-span-2">
          <LinkedTrendChart
            data={data.trend}
            xKey="bucket"
            xFormat={(v) => bucketTick(v, w.granularity)}
            yFormat={(v) => int(v)}
            stacked
            height={240}
            series={[
              { key: "aiAccepted", label: m.accepted, type: "bar", color: slotColor(0) },
              { key: "aiDeclined", label: m.declined, type: "bar", color: slotColor(4) },
            ]}
            hrefFor={() => list({ source: "ai" })}
            summary={m.aiTrend}
          />
        </ChartCard>
        <ChartCard title={m.status} description={m.statusHint}>
          <HBarChart
            showShare
            format={(v) => int(v)}
            items={data.status
              .filter((s) => s.value > 0)
              .map((s) => ({
                label: sp.improvements.statuses[s.key] ?? s.key,
                value: s.value,
                href: list({ status: s.key }),
              }))}
          />
        </ChartCard>
      </div>

      <div className="grid gap-(--gap) lg:grid-cols-[minmax(0,2fr)_minmax(0,3fr)]">
        <ChartCard title={m.sources}>
          <HBarChart
            showShare
            format={(v) => int(v)}
            items={[...data.sources]
              .sort((a, b) => b.value - a.value)
              .filter((s) => s.value > 0)
              .map((s) => ({
                label: sp.improvements.sources[s.key] ?? s.key,
                value: s.value,
                href: list({ source: s.key }),
              }))}
          />
        </ChartCard>
        <ChartCard title={m.overdueList} description={m.overdueListHint}>
          <SortTable
            rows={data.overdueItems}
            rowKey={(i) => i.id}
            rowHref={() => list({ status: "open" })}
            initialSort={{ key: "late", desc: true }}
            columns={[
              { key: "t", label: "Item", render: (i) => i.title },
              { key: "o", label: m.owner, render: (i) => i.ownerName || i.departmentName || "–" },
              { key: "s", label: "Sumber", render: (i) => sp.improvements.sources[i.source] ?? i.source },
              {
                key: "d",
                label: m.dueDate,
                align: "right",
                render: (i) => formatDate(i.dueDate),
                sort: (i) => i.dueDate,
              },
              {
                key: "late",
                label: m.late,
                align: "right",
                render: (i) => <span className="font-medium text-destructive">{m.days(i.daysOverdue)}</span>,
                sort: (i) => i.daysOverdue,
              },
            ]}
          />
        </ChartCard>
      </div>

      <ChartCard title={m.departments}>
        <SortTable
          rows={data.departments}
          rowKey={(d) => d.departmentId}
          rowHref={(d) => href(P, { dept: d.departmentId })}
          initialSort={{ key: "open", desc: true }}
          columns={[
            {
              key: "n",
              label: ds.filters.department,
              render: (d) => d.departmentName,
              sort: (d) => d.departmentName,
            },
            {
              key: "open",
              label: sp.improvements.statuses.open,
              align: "right",
              render: (d) => int(d.open),
              sort: (d) => d.open,
              total: sum((d) => d.open),
            },
            {
              key: "ip",
              label: m.inProgress,
              align: "right",
              render: (d) => int(d.inProgress),
              sort: (d) => d.inProgress,
              total: sum((d) => d.inProgress),
            },
            {
              key: "done",
              label: m.completed,
              align: "right",
              render: (d) => int(d.done),
              sort: (d) => d.done,
              total: sum((d) => d.done),
            },
            {
              key: "od",
              label: m.overdue,
              align: "right",
              render: (d) => int(d.overdue),
              sort: (d) => d.overdue,
              total: sum((d) => d.overdue),
            },
            {
              key: "sg",
              label: m.suggested,
              align: "right",
              render: (d) => int(d.suggested),
              sort: (d) => d.suggested,
              total: sum((d) => d.suggested),
            },
          ]}
        />
      </ChartCard>
    </div>
  );
}
