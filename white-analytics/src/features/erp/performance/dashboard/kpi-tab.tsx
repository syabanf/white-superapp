"use client";

import { ChartCard } from "@/components/dashboard/chart-card";
import { KpiGrid } from "@/components/dashboard/kpi-tile";
import { DeltaBadge } from "@/components/dashboard/delta-badge";
import { MUTED_SERIES, slotColor } from "@/components/dashboard/charts/chart-primitives";
import { BulletRows, Funnel, LinkedTrendChart, ValueHeatmap } from "./charts";
import { href, int, scoreDelta, monthOf, monthTick, pct, score } from "./lib";
import { ds } from "./strings";
import type { DashKpi } from "./types";
import { MetricTile, SortTable, type Column } from "./widgets";

const k = ds.kpi;
const P = "/erp/performance/kpi";

type Person = DashKpi["people"][number];

function fmtValue(v: number | undefined, unit: string): string {
  if (v == null) return "–";
  if (unit === "%") return pct(v, 1);
  return `${v.toLocaleString("id-ID", { maximumFractionDigits: 2 })} ${unit}`.trim();
}

export function KpiTab({ data }: { data: DashKpi }) {
  const w = data.window;
  const scope = w.scope === "me" ? "me" : w.scope === "team" ? "team" : "all";
  const month = monthOf(w.to);
  const hasPrev = !!w.compareFrom;
  const hist = data.histogram.map((b) => ({ bin: b.key, value: b.value, previous: b.previous ?? null }));
  const deptRows = [
    ...new Map(data.departmentMonths.map((d) => [d.departmentId, d.departmentName])).entries(),
  ].map(([key, label]) => ({ key, label }));
  const cell = new Map(data.departmentMonths.map((d) => [`${d.departmentId}|${d.period}`, d.score]));
  const scored = data.people.filter((p) => p.score != null);
  const top = scored.slice(0, 5);
  const bottom = [...scored].reverse().slice(0, 5);
  const funnel = data.funnel.map((f) => ({
    label: k.funnelSteps[f.key] ?? f.key,
    value: f.value,
    href: f.key === "members" ? undefined : href(P, { month, scope }),
  }));

  const personCols: Column<Person>[] = [
    { key: "name", label: ds.filters.employee, render: (p) => p.name, sort: (p) => p.name },
    {
      key: "dept",
      label: ds.filters.department,
      render: (p) => p.departmentName || "–",
      sort: (p) => p.departmentName,
    },
    {
      key: "score",
      label: ds.summary.score,
      align: "right",
      render: (p) => score(p.score),
      sort: (p) => p.score,
    },
    {
      key: "delta",
      label: ds.summary.delta,
      align: "right",
      render: (p) =>
        p.score != null && p.previous != null ? (
          <DeltaBadge delta={scoreDelta(p.score, p.previous)} mode="abs" digits={1} />
        ) : (
          "–"
        ),
      sort: (p) => (p.score != null && p.previous != null ? p.score - p.previous : undefined),
    },
    { key: "months", label: k.months, align: "right", render: (p) => int(p.months), sort: (p) => p.months },
    {
      key: "final",
      label: k.finalMonths,
      align: "right",
      render: (p) => `${p.finalMonths}/${p.months}`,
      sort: (p) => p.finalMonths / Math.max(1, p.months),
    },
  ];
  const compactCols = [personCols[0]!, personCols[2]!, personCols[3]!];
  const personHref = (p: Person) =>
    href(P, { month, scope: scope === "me" ? "me" : "all", dept: p.departmentId });

  return (
    <div className="flex flex-col gap-4 sm:gap-5">
      <KpiGrid cols={4}>
        <MetricTile
          label={k.avg}
          hint={ds.summary.kpiHint}
          metric={data.avgScore}
          format={score}
          href={href(P, { month, scope })}
        />
        <MetricTile label={k.finalRate} hint={k.finalRateHint} metric={data.finalRate} format={pct} />
        <MetricTile label={k.count} metric={data.scorecards} format={int} />
        <MetricTile label={k.below} hint={k.belowHint} metric={data.belowTarget} format={int} lowerIsBetter />
      </KpiGrid>

      <div className="grid gap-4 sm:gap-5 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
        <ChartCard
          title={k.histogram}
          description={k.histogramHint}
          table={
            <SortTable
              rows={hist}
              rowKey={(r) => r.bin}
              columns={[
                { key: "bin", label: "Skor", render: (r) => r.bin },
                {
                  key: "v",
                  label: ds.current,
                  align: "right",
                  render: (r) => int(r.value),
                  total: int(hist.reduce((a, r) => a + r.value, 0)),
                },
                ...(hasPrev
                  ? [
                      {
                        key: "p",
                        label: ds.previous,
                        align: "right" as const,
                        render: (r: (typeof hist)[number]) => int(r.previous),
                      },
                    ]
                  : []),
              ]}
            />
          }
        >
          <LinkedTrendChart
            data={hist}
            xKey="bin"
            xFormat={(v) => v}
            yFormat={(v) => int(v)}
            series={[
              {
                key: "value",
                label: ds.current,
                type: "bar",
                color: slotColor(0),
                format: (v: number) => `${int(v)} orang`,
              },
              ...(hasPrev
                ? [
                    {
                      key: "previous",
                      label: ds.previous,
                      type: "bar" as const,
                      color: MUTED_SERIES,
                      format: (v: number) => `${int(v)} orang`,
                    },
                  ]
                : []),
            ]}
            hrefFor={() => href(P, { month, scope })}
            summary={k.histogram}
          />
        </ChartCard>
        <ChartCard title={k.funnel} description={k.funnelHint}>
          <Funnel steps={funnel} />
        </ChartCard>
      </div>

      <ChartCard title={k.heatmap} description={k.heatmapHint}>
        <ValueHeatmap
          rows={deptRows}
          cols={data.months}
          value={(r, c) => cell.get(`${r}|${c}`)}
          format={(v) => score(v)}
          colLabel={monthTick}
          hrefFor={(r, c) => href(P, { month: c, scope: scope === "me" ? "me" : "all", dept: r })}
        />
      </ChartCard>

      <ChartCard
        title={k.indicators}
        description={k.indicatorsHint}
        table={
          <SortTable
            rows={data.indicators}
            rowKey={(i) => i.indicatorId}
            columns={[
              { key: "n", label: "Indikator", render: (i) => i.name },
              { key: "a", label: k.actual, align: "right", render: (i) => fmtValue(i.actual, i.unit) },
              { key: "t", label: k.target, align: "right", render: (i) => fmtValue(i.target, i.unit) },
              { key: "c", label: k.attainment, align: "right", render: (i) => pct(i.attainment) },
            ]}
          />
        }
      >
        <BulletRows
          rows={[...data.indicators]
            .sort((a, b) => (a.attainment ?? 0) - (b.attainment ?? 0))
            .map((i) => ({
              label: i.name,
              value: i.attainment,
              previous: i.previousAttainment,
              caption: `${fmtValue(i.actual, i.unit)} / ${fmtValue(i.target, i.unit)}`,
            }))}
        />
      </ChartCard>

      <div className="grid gap-4 sm:gap-5 lg:grid-cols-2">
        <ChartCard title={k.top}>
          <SortTable
            rows={top}
            rowKey={(p) => p.employeeId}
            rowHref={personHref}
            columns={compactCols}
            minWidth={0}
          />
        </ChartCard>
        <ChartCard title={k.bottom}>
          <SortTable
            rows={bottom}
            rowKey={(p) => p.employeeId}
            rowHref={personHref}
            columns={compactCols}
            minWidth={0}
          />
        </ChartCard>
      </div>

      <ChartCard title={k.people}>
        <SortTable
          rows={data.people}
          rowKey={(p) => p.employeeId}
          rowHref={personHref}
          columns={personCols}
          initialSort={{ key: "score", desc: true }}
          limit={50}
        />
      </ChartCard>
    </div>
  );
}
