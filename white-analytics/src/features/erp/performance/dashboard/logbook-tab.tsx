"use client";

import { ChartCard } from "@/components/dashboard/chart-card";
import { KpiGrid } from "@/components/dashboard/kpi-tile";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { Heatmap } from "@/components/dashboard/charts/heatmap";
import { MUTED_SERIES, slotColor } from "@/components/dashboard/charts/chart-primitives";
import { TrendChart, ValueHeatmap } from "./charts";
import { PerfDrillSheet, usePerfDrill } from "./drill";
import { bucketEnd, bucketLabel, bucketTick, dayTick, href, int, pct, ratio, score } from "./lib";
import { ds } from "./strings";
import type { DashLogbook } from "./types";
import { MetricTile, SortTable } from "./widgets";

const l = ds.logbook;
const P = "/erp/performance/logbook";
/** Weeks the weekday heatmap shows (the latest ones). */
const HEAT_WEEKS = 26;

export function LogbookTab({ data, dept }: { data: DashLogbook; dept?: string }) {
  const w = data.window;
  const hasPrev = !!w.compareFrom;
  const history = (extra: Record<string, string | undefined> = {}) =>
    href(P, { tab: "history", dept, from: w.from, to: w.to, ...extra });
  const p100 = (v: number) => `${Math.round(v)}%`;
  const drill = usePerfDrill();
  const dayOf = (week: string, weekday: number) => {
    const d = new Date(`${week}T00:00:00Z`);
    d.setUTCDate(d.getUTCDate() + weekday - 1);
    return d.toISOString().slice(0, 10);
  };
  const trend = data.trend.map((b) => ({
    bucket: b.bucket,
    rate: b.expected ? (b.submitted / b.expected) * 100 : null,
    recaps: b.expected ? Math.min(100, (b.recaps / b.expected) * 100) : null,
    prev: b.previousRate != null ? b.previousRate * 100 : null,
  }));

  const buckets = [...new Set(data.departments.map((d) => d.bucket))];
  const deptRows = [
    ...new Map(data.departments.map((d) => [d.departmentId, d.departmentName])).entries(),
  ].map(([key, label]) => ({ key, label }));
  const cell = new Map(
    data.departments.map((d) => [`${d.departmentId}|${d.bucket}`, ratio(d.submitted, d.expected)]),
  );

  const weeks = [...new Set(data.heatmap.map((c) => c.week))].slice(-HEAT_WEEKS);
  const weekdays = [
    ...new Set(data.heatmap.filter((c) => weeks.includes(c.week)).map((c) => c.weekday)),
  ].sort();
  const heatCells = data.heatmap
    .filter((c) => weeks.includes(c.week))
    .map((c) => ({
      row: weekdays.indexOf(c.weekday),
      col: weeks.indexOf(c.week),
      value: c.expected ? Math.round((c.submitted / c.expected) * 100) : c.submitted ? 100 : 0,
      count: c.submitted,
    }));

  const totals = data.people.reduce(
    (a, p) => ({ e: a.e + p.expected, s: a.s + p.submitted, r: a.r + p.rejected, c: a.c + p.recaps }),
    { e: 0, s: 0, r: 0, c: 0 },
  );

  return (
    <div className="page-stack">
      <KpiGrid cols={4}>
        <MetricTile
          label={l.rate}
          hint={ds.summary.logbookHint}
          metric={data.submissionRate}
          format={pct}
          href={history()}
        />
        <MetricTile label={l.score} hint={l.scoreHint} metric={data.avgScore} format={score} href={history({ status: "reviewed" })} />
        <MetricTile
          label={l.rejected}
          hint={l.rejectedHint}
          metric={data.rejectionRate}
          format={(v) => pct(v, 1)}
          lowerIsBetter
          href={history({ status: "rejected" })}
        />
        <MetricTile
          label={l.recaps}
          hint={l.recapsHint}
          metric={data.recapCoverage}
          format={pct}
          href={href("/erp/performance/recaps", { dept })}
        />
      </KpiGrid>

      <ChartCard
        title={l.trend}
        description={l.trendHint}
        table={
          <SortTable
            rows={data.trend}
            rowKey={(b) => b.bucket}
            columns={[
              { key: "b", label: "Periode", render: (b) => bucketLabel(b.bucket, w.granularity) },
              { key: "e", label: l.expected, align: "right", render: (b) => int(b.expected) },
              { key: "s", label: l.sent, align: "right", render: (b) => int(b.submitted) },
              {
                key: "r",
                label: ds.logbook.statusKeys.rejected,
                align: "right",
                render: (b) => int(b.rejected),
              },
              { key: "sc", label: l.score100, align: "right", render: (b) => score(b.avgScore) },
              { key: "rc", label: l.recapsCol, align: "right", render: (b) => int(b.recaps) },
            ]}
          />
        }
      >
        <TrendChart
          data={trend}
          xKey="bucket"
          xFormat={(v) => bucketTick(v, w.granularity)}
          yFormat={p100}
          yDomain={[0, 100]}
          height={280}
          series={[
            { key: "rate", label: l.rate, color: slotColor(0), format: p100 },
            { key: "recaps", label: l.recaps, color: slotColor(1), format: p100 },
            ...(hasPrev
              ? [
                  {
                    key: "prev",
                    label: `${l.rate} (${ds.previous.toLowerCase()})`,
                    color: MUTED_SERIES,
                    dashed: true,
                    format: p100,
                  },
                ]
              : []),
          ]}
          onPick={(row) => {
            const b = String(row.bucket);
            drill("log", { from: b, to: bucketEnd(b, w.granularity, w.to), l: `${l.trend} · ${bucketLabel(b, w.granularity)}` });
          }}
          summary={l.trend}
        />
      </ChartCard>

      <ChartCard title={l.byDept} description={l.byDeptHint}>
        <ValueHeatmap
          rows={deptRows}
          cols={buckets}
          value={(r, c) => cell.get(`${r}|${c}`)}
          format={(v) => pct(v)}
          colLabel={(c) => bucketTick(c, w.granularity)}
          hrefFor={(r, c) =>
            href(P, { tab: "history", dept: r, from: c, to: bucketEnd(c, w.granularity, w.to) })
          }
        />
      </ChartCard>

      <div className="grid gap-(--gap) lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
        <ChartCard title={l.heatmap} description={l.heatmapHint}>
          {heatCells.length ? (
            <Heatmap
              cells={heatCells}
              rows={weekdays.length}
              cols={weeks.length}
              rowLabels={weekdays.map((d) => l.weekdays[d] ?? "")}
              colLabels={weeks.map(dayTick)}
              format={(v) => `${v}%`}
              valueLabel={l.sent}
              countLabel="Logbook"
              cellSize={18}
              onSelect={(c) => {
                const day = dayOf(weeks[c.col]!, weekdays[c.row]!);
                drill("log", { from: day, to: day, dim: "none", l: `${l.weekdays[weekdays[c.row]!]} ${dayTick(day)}` });
              }}
            />
          ) : (
            <p className="py-10 text-center text-sm text-muted-foreground">{ds.noData}</p>
          )}
        </ChartCard>
        <ChartCard title={l.status}>
          <HBarChart
            showShare
            format={(v) => int(v)}
            items={data.status.map((s) => ({
              label: l.statusKeys[s.key] ?? s.key,
              value: s.value,
              href: history({ status: s.key }),
            }))}
          />
        </ChartCard>
      </div>

      <ChartCard title={l.people} description={l.peopleHint}>
        <SortTable
          rows={data.people}
          rowKey={(p) => p.employeeId}
          onRow={(p) => drill("log", { from: w.from, to: w.to, emp: p.employeeId, dim: "emp", l: p.name })}
          limit={40}
          columns={[
            { key: "n", label: ds.filters.employee, render: (p) => p.name, sort: (p) => p.name },
            {
              key: "d",
              label: ds.filters.department,
              render: (p) => p.departmentName,
              sort: (p) => p.departmentName,
            },
            {
              key: "e",
              label: l.expected,
              align: "right",
              render: (p) => int(p.expected),
              sort: (p) => p.expected,
              total: int(totals.e),
            },
            {
              key: "s",
              label: l.sent,
              align: "right",
              render: (p) => int(p.submitted),
              sort: (p) => p.submitted,
              total: int(totals.s),
            },
            {
              key: "rate",
              label: l.rate,
              align: "right",
              render: (p) => pct(ratio(p.submitted, p.expected)),
              sort: (p) => ratio(p.submitted, p.expected),
              total: pct(ratio(totals.s, totals.e)),
            },
            {
              key: "r",
              label: l.statusKeys.rejected,
              align: "right",
              render: (p) => int(p.rejected),
              sort: (p) => p.rejected,
              total: int(totals.r),
            },
            {
              key: "sc",
              label: l.score100,
              align: "right",
              render: (p) => score(p.avgScore),
              sort: (p) => p.avgScore,
            },
            {
              key: "rc",
              label: l.recapsCol,
              align: "right",
              render: (p) => int(p.recaps),
              sort: (p) => p.recaps,
              total: int(totals.c),
            },
          ]}
        />
      </ChartCard>
      <PerfDrillSheet period={w} />
    </div>
  );
}
