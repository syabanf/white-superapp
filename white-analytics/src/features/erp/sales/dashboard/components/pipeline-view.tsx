"use client";

import { ArrowUpRight } from "lucide-react";
import { cn } from "@/lib/utils";
import { ChartCard } from "@/components/dashboard/chart-card";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { formatRupiah } from "@/features/erp/shared/money";
import { formatNumber } from "@/lib/format";
import { LINE_ORDER, SEQ, lineColor, lineLabel } from "../colors";
import { rpCompact, sum } from "../format";
import { baseFilter, listHref, periodFilter, SalesDrillSheet, useSalesDrill } from "../drill";
import { dd } from "../strings";
import type { DashboardDeal, DashboardPipeline, DashboardQuery, DashboardSlice } from "../types";
import { FunnelChart, StackedBars, type StackRow } from "./charts";
import { DetailTable } from "./detail-table";
import { KpiRow } from "./kpi";

const p = dd.pipeline;
const AGE_BUCKETS = ["0-7", "8-14", "15-30", "31+"];

/** Groups two-dimensional slices into stacked rows, keeping the stage order. */
function stackRows(slices: DashboardSlice[], pick: (s: DashboardSlice) => number): StackRow[] {
  const rows: StackRow[] = [];
  for (const sl of slices) {
    let row = rows.find((r) => r.key === sl.key);
    if (!row) rows.push((row = { key: sl.key, label: sl.label, segments: [] }));
    row.segments.push({ key: sl.group, value: pick(sl) });
  }
  return rows;
}

export function PipelineView({ data, query }: { data: DashboardPipeline; query: DashboardQuery }) {
  const open = useSalesDrill(query);
  const base = { ...baseFilter(query), pipeline: query.pipelineId ?? data.pipelineId };
  const href = listHref("deal", { ...base, status: "open" });
  const stage = (key: string, label: string, extra: Record<string, string> = {}) => open("deal", { pipeline: base.pipeline, status: "open", stage: key, l: label, ...extra });
  const lineGroups = [...LINE_ORDER, "lainnya"].map((k) => ({ key: k, label: lineLabel(k), color: lineColor(k) }));
  const ageGroups = AGE_BUCKETS.map((b, i) => ({ key: b, label: p.ageBucket(b), color: SEQ[i + 1]! }));
  return (
    <>
      <KpiRow
        kpis={data.kpis}
        keys={["open_deals", "pipeline_value", "weighted_value", "stuck_deals", "avg_age_days", "velocity"]}
        compare={data.period.compare}
        hrefs={{
          open_deals: href,
          pipeline_value: href,
          weighted_value: href,
          stuck_deals: "#deal-macet",
          avg_age_days: href,
          velocity: listHref("deal", { ...base, ...periodFilter(query), status: "won" }),
        }}
      />
      <div className="grid gap-(--gap) xl:grid-cols-2">
        <ChartCard title={p.funnel} description={p.funnelHint}>
          <FunnelChart
            steps={data.funnel.map((f) => ({ key: f.key, label: f.label, count: f.count, value: f.value }))}
            valueFormat={rpCompact}
            onSelect={(f) => open("deal", { pipeline: base.pipeline, stage: f.key, ...periodFilter(query, "created"), dim: "none", l: `${f.label} · ${p.cohort}` })}
          />
        </ChartCard>
        <ChartCard title={p.byLine} description={p.byLineHint}>
          <StackedBars
            rows={stackRows(data.stageLines, (s) => s.value)}
            groups={lineGroups}
            format={rpCompact}
            onSelect={(r, g) => stage(r.key, g ? `${r.label} · ${lineLabel(g)}` : r.label, g && g !== "lainnya" ? { line: g } : {})}
          />
        </ChartCard>
        <ChartCard title={p.aging} description={p.agingHint}>
          <StackedBars
            rows={stackRows(data.aging, (s) => s.count)}
            groups={ageGroups}
            format={(v) => dd.deals(v)}
            onSelect={(r, g) => stage(r.key, g ? `${r.label} · ${p.ageBucket(g)}` : r.label, g ? { age: g } : {})}
          />
        </ChartCard>
        <ChartCard title={p.stageDays} description={p.stageDaysHint}>
          <HBarChart
            items={data.stageDays.map((s) => ({ label: s.label, value: s.value, secondary: p.exits(s.count), key: s.key }))}
            onSelect={(it) => stage(it.key!, it.label)}
            format={(v) => dd.days(formatNumber(v, 1))}
            emptyLabel={dd.empty}
          />
        </ChartCard>
      </div>
      <div id="deal-macet" className="scroll-mt-20">
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
              {
                key: "value",
                label: dd.summary.value,
                numeric: true,
                render: (r) => formatRupiah(r.value),
                sort: (r) => r.value,
                total: (rs) => formatRupiah(sum(rs.map((r) => r.value))),
              },
              {
                key: "days",
                label: p.daysInStage,
                numeric: true,
                render: (r) => (
                  <span className={cn("inline-flex items-center gap-0.5", r.daysInStage > r.thresholdDays ? "text-negative" : "text-muted-foreground")}>
                    {r.daysInStage > r.thresholdDays ? <ArrowUpRight aria-hidden className="size-3" /> : null}
                    {r.daysInStage} / {r.thresholdDays}
                  </span>
                ),
                sort: (r) => r.daysInStage / Math.max(1, r.thresholdDays),
              },
            ]}
          />
        </ChartCard>
      </div>
      <SalesDrillSheet query={query} />
    </>
  );
}
