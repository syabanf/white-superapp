"use client";

import { ChartCard } from "@/components/dashboard/chart-card";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { StackedBarChart, type CategoryRow } from "@/components/dashboard/charts/stacked-bar-chart";
import { slotColor } from "@/components/dashboard/charts/chart-primitives";
import { formatDateShort, formatDecimal, formatNumber, formatPercent } from "@/lib/format";
import { dashboardHref, rate, type DrillContext } from "../../analytics";
import { SOURCE, ra } from "../../strings";
import type { RecruitmentDashboard, SourceStat } from "../../types";
import { useRecruitmentDrill, weekRange } from "./drill";
import { AnalyticsTable } from "./table";

const TOP_SOURCES = 5;

export function SourcesTab({ data, ctx }: { data: RecruitmentDashboard; ctx: DrillContext }) {
  const s = ra.sources;
  const sources = data.sources;
  const drill = useRecruitmentDrill(ctx);
  const bySource = (src: string, extra: Record<string, string> = {}) => drill("cand", { source: src, dim: "source", l: SOURCE[src] ?? src, ...extra });
  const label = (src: string) => SOURCE[src] ?? src;
  const sum = (pick: (x: SourceStat) => number, list = sources) => list.reduce((a, b) => a + pick(b), 0);

  // colors follow the source, fixed by its overall rank, so the trend keeps them
  const top = sources.slice(0, TOP_SOURCES).map((x) => x.source);
  const weeks = new Map<string, CategoryRow>();
  for (const w of data.sourceTrend) {
    const row = weeks.get(w.start) ?? { label: w.start };
    const key = top.includes(w.source) ? w.source : "others";
    row[key] = Number(row[key] ?? 0) + w.count;
    weeks.set(w.start, row);
  }
  const trendSeries = [
    ...top.map((src) => ({ key: src, label: label(src), format: formatNumber })),
    ...(sources.length > TOP_SOURCES ? [{ key: "others", label: s.others, format: formatNumber, color: "var(--chart-emphasis-muted)" }] : []),
  ];

  const free = sources.filter((x) => !x.paid);
  const paid = sources.filter((x) => x.paid);

  return (
    <div className="flex flex-col gap-4 sm:gap-5">
      <div className="grid gap-4 sm:gap-5 lg:grid-cols-2">
        <ChartCard title={s.volume} description={s.volumeHint}>
          <HBarChart
            items={sources.map((x) => ({ label: label(x.source), value: x.applications, key: x.source }))}
            onSelect={(it) => bySource(it.key!)}
            format={formatNumber}
            showShare
            emptyLabel={ra.empty}
          />
        </ChartCard>
        <ChartCard title={s.quality} description={s.qualityHint}>
          <HBarChart
            items={[...sources]
              .map((x) => ({ x, r: rate(x.hires, x.applications) ?? 0 }))
              .sort((a, b) => b.r - a.r)
              .map(({ x, r }) => ({
                label: label(x.source),
                value: r,
                secondary: s.hireRateOf(formatNumber(x.hires), formatNumber(x.applications)),
                key: x.source,
              }))}
            onSelect={(it) => bySource(it.key!, { status: "hired", l: `${label(it.key!)} · ${s.cols.hires}` })}
            format={(v) => formatPercent(v, 1)}
            color={slotColor(2)}
            emptyLabel={ra.empty}
          />
        </ChartCard>
      </div>

      <div className="grid gap-4 sm:gap-5 xl:grid-cols-3">
        <ChartCard title={s.split} description={s.splitHint}>
          <div className="flex flex-col gap-5">
            <SplitRow title={s.cols.applications} free={sum((x) => x.applications, free)} paid={sum((x) => x.applications, paid)} />
            <SplitRow title={s.cols.hires} free={sum((x) => x.hires, free)} paid={sum((x) => x.hires, paid)} />
            <dl className="grid grid-cols-2 gap-3 text-sm">
              {[
                { name: s.free, list: free },
                { name: s.paid, list: paid },
              ].map((g) => {
                const r = rate(sum((x) => x.hires, g.list), sum((x) => x.applications, g.list));
                return (
                  <div key={g.name} className="rounded-lg bg-muted/50 px-3 py-2">
                    <dt className="text-xs text-muted-foreground">
                      {s.cols.hireRate} {g.name.toLowerCase()}
                    </dt>
                    <dd className="text-lg font-semibold">{r == null ? "–" : formatPercent(r, 1)}</dd>
                  </div>
                );
              })}
            </dl>
          </div>
        </ChartCard>
        <ChartCard className="xl:col-span-2" title={s.trend} description={s.trendHint}>
          <StackedBarChart
            data={[...weeks.values()]}
            series={trendSeries}
            labelFormat={(d) => formatDateShort(d)}
            valueFormat={formatNumber}
            height={260}
            onSelect={(row, key) => {
              const r = weekRange(String(row.label));
              drill("cand", { ...r, week: 1, source: key && key !== "others" ? key : undefined, l: `${formatDateShort(String(row.label))}${key && key !== "others" ? ` · ${label(key)}` : ""}` });
            }}
          />
        </ChartCard>
      </div>

      <ChartCard title={s.table}>
        <AnalyticsTable<SourceStat>
          rows={sources}
          rowKey={(r) => r.source}
          rowHref={(r) => dashboardHref(ctx, { tab: "sumber", source: r.source })}
          initialSort={{ key: "applications", desc: true }}
          columns={[
            { key: "source", header: s.cols.source, cell: (r) => <span className="font-medium">{label(r.source)}</span>, sort: (r) => label(r.source) },
            { key: "channel", header: s.cols.channel, cell: (r) => (r.paid ? s.paid : s.free), sort: (r) => (r.paid ? 1 : 0) },
            { key: "applications", header: s.cols.applications, align: "right", cell: (r) => formatNumber(r.applications), sort: (r) => r.applications, total: formatNumber(sum((x) => x.applications)) },
            { key: "screened", header: s.cols.screened, align: "right", cell: (r) => formatNumber(r.screened), sort: (r) => r.screened, total: formatNumber(sum((x) => x.screened)) },
            { key: "interviewed", header: s.cols.interviewed, align: "right", cell: (r) => formatNumber(r.interviewed), sort: (r) => r.interviewed, total: formatNumber(sum((x) => x.interviewed)) },
            { key: "offered", header: s.cols.offered, align: "right", cell: (r) => formatNumber(r.offered), sort: (r) => r.offered, total: formatNumber(sum((x) => x.offered)) },
            { key: "hires", header: s.cols.hires, align: "right", cell: (r) => formatNumber(r.hires), sort: (r) => r.hires, total: formatNumber(sum((x) => x.hires)) },
            {
              key: "rate",
              header: s.cols.hireRate,
              align: "right",
              cell: (r) => {
                const v = rate(r.hires, r.applications);
                return v == null ? "–" : formatPercent(v, 1);
              },
              sort: (r) => rate(r.hires, r.applications) ?? -1,
              total: (() => {
                const v = rate(sum((x) => x.hires), sum((x) => x.applications));
                return v == null ? "–" : formatPercent(v, 1);
              })(),
            },
            {
              key: "tth",
              header: s.cols.timeToHire,
              align: "right",
              cell: (r) => (r.avgDaysToHire == null ? "–" : formatDecimal(r.avgDaysToHire, 1)),
              sort: (r) => r.avgDaysToHire ?? Number.MAX_SAFE_INTEGER,
            },
          ]}
        />
      </ChartCard>
    </div>
  );
}

/** One 100% bar split into free and paid with both counts labelled. */
function SplitRow({ title, free, paid }: { title: string; free: number; paid: number }) {
  const total = free + paid;
  const freePct = total ? (free / total) * 100 : 0;
  return (
    <div>
      <div className="mb-1.5 flex items-baseline justify-between text-xs">
        <span className="font-medium">{title}</span>
        <span className="tabular text-muted-foreground">{formatNumber(total)}</span>
      </div>
      <div className="flex h-3 w-full gap-0.5 overflow-hidden rounded-[4px]" role="img" aria-label={`${title}: gratis ${free}, berbayar ${paid}`}>
        {free > 0 ? <span style={{ width: `${freePct}%`, background: slotColor(2) }} /> : null}
        {paid > 0 ? <span style={{ width: `${100 - freePct}%`, background: slotColor(1) }} /> : null}
      </div>
      <div className="mt-1.5 flex justify-between text-xs tabular text-muted-foreground">
        <span className="flex items-center gap-1.5">
          <span aria-hidden className="size-2.5 rounded-[3px]" style={{ background: slotColor(2) }} />
          {ra.sources.free} {formatNumber(free)} ({formatPercent(freePct, 0)})
        </span>
        <span className="flex items-center gap-1.5">
          <span aria-hidden className="size-2.5 rounded-[3px]" style={{ background: slotColor(1) }} />
          {ra.sources.paid} {formatNumber(paid)} ({formatPercent(total ? 100 - freePct : 0, 0)})
        </span>
      </div>
    </div>
  );
}
