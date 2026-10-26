"use client";

import { ChartCard } from "@/components/dashboard/chart-card";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { formatRupiah } from "@/features/erp/shared/money";
import { formatNumber } from "@/lib/format";
import { LINE_ORDER, SEQ, lineColor, lineLabel } from "../colors";
import { rpCompact, sum } from "../format";
import { drill } from "../params";
import { dd } from "../strings";
import type { DashboardDeal, DashboardPipeline, DashboardQuery, DashboardSlice } from "../types";
import { FunnelChart, StackedBars, type StackRow } from "./charts";
import { DetailTable } from "./detail-table";
import { KpiRow } from "./kpi";

const p = dd.pipeline;
const AGE_BUCKETS = ["0-7", "8-14", "15-30", "31+"];

/** Groups two-dimensional slices into stacked rows, keeping the stage order. */
function stackRows(slices: DashboardSlice[], href: string, pick: (s: DashboardSlice) => number): StackRow[] {
  const rows: StackRow[] = [];
  for (const sl of slices) {
    let row = rows.find((r) => r.key === sl.key);
    if (!row) rows.push((row = { key: sl.key, label: sl.label, href, segments: [] }));
    row.segments.push({ key: sl.group, value: pick(sl) });
  }
  return rows;
}

export function PipelineView({ data, query }: { data: DashboardPipeline; query: DashboardQuery }) {
  const href = drill("/erp/sales/pipeline", { pipeline: query.pipelineId ?? data.pipelineId, owner: query.ownerUserId });
  const lineGroups = [...LINE_ORDER, "lainnya"].map((k) => ({ key: k, label: lineLabel(k), color: lineColor(k) }));
  const ageGroups = AGE_BUCKETS.map((b, i) => ({ key: b, label: p.ageBucket(b), color: SEQ[i + 1]! }));
  return (
    <>
      <KpiRow
        kpis={data.kpis}
        keys={["open_deals", "pipeline_value", "weighted_value", "stuck_deals", "avg_age_days", "velocity"]}
        compare={data.period.compare}
        hrefs={{ open_deals: href, pipeline_value: href }}
      />
      <div className="grid gap-4 xl:grid-cols-2">
        <ChartCard title={p.funnel} description={p.funnelHint}>
          <FunnelChart steps={data.funnel.map((f) => ({ key: f.key, label: f.label, count: f.count, value: f.value, href }))} valueFormat={rpCompact} />
        </ChartCard>
        <ChartCard title={p.byLine} description={p.byLineHint}>
          <StackedBars rows={stackRows(data.stageLines, href, (s) => s.value)} groups={lineGroups} format={rpCompact} />
        </ChartCard>
        <ChartCard title={p.aging} description={p.agingHint}>
          <StackedBars rows={stackRows(data.aging, href, (s) => s.count)} groups={ageGroups} format={(v) => dd.deals(v)} />
        </ChartCard>
        <ChartCard title={p.stageDays} description={p.stageDaysHint}>
          <HBarChart
            items={data.stageDays.map((s) => ({ label: s.label, value: s.value, secondary: p.exits(s.count) }))}
            format={(v) => dd.days(formatNumber(v, 1))}
            emptyLabel={dd.empty}
          />
        </ChartCard>
      </div>
      <ChartCard title={p.stuck} description={p.stuckHint}>
        <DetailTable<DashboardDeal>
          rows={data.stuck}
          rowKey={(r) => r.id}
          href={(r) => `/erp/sales/pipeline/${r.id}`}
          empty={p.noStuck}
          initialSort={{ key: "days", desc: true }}
          columns={[
            { key: "title", label: p.deal, render: (r) => <span className="line-clamp-1">{r.title}</span>, sort: (r) => r.title },
            { key: "stage", label: dd.summary.stage, render: (r) => r.stageName, sort: (r) => r.stageName },
            { key: "owner", label: p.owner, render: (r) => r.ownerName || "–", optional: true },
            { key: "value", label: dd.summary.value, numeric: true, render: (r) => formatRupiah(r.value), sort: (r) => r.value, total: (rs) => formatRupiah(sum(rs.map((r) => r.value))) },
            { key: "days", label: p.daysInStage, numeric: true, render: (r) => `${r.daysInStage} / ${r.thresholdDays}`, sort: (r) => r.daysInStage / Math.max(1, r.thresholdDays) },
          ]}
        />
      </ChartCard>
    </>
  );
}
