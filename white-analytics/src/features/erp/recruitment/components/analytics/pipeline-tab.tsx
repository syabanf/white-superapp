"use client";

import Link from "next/link";
import { ChartCard } from "@/components/dashboard/chart-card";
import { StatStrip } from "@/components/dashboard/stat-strip";
import { BoxPlot } from "@/components/dashboard/charts/box-plot";
import { FunnelChart } from "@/components/dashboard/charts/funnel-chart";
import { StackedBarChart, type CategoryRow } from "@/components/dashboard/charts/stacked-bar-chart";
import { formatDate, formatDecimal, formatNumber, formatPercent } from "@/lib/format";
import { candidateHref, candidatesHref, pipelineHref, rate, topN, type DrillContext } from "../../analytics";
import { CANDIDATE_STATUS, ra } from "../../strings";
import type { FunnelStageStat, RecruitmentDashboard, StuckCandidate } from "../../types";
import { CandidateBadge } from "../ui";
import { AnalyticsTable } from "./table";

const STACK = ["applied", "screening", "psikotes", "interview", "offer", "hired", "rejected", "talent_pool"] as const;

export function PipelineTab({ data, ctx }: { data: RecruitmentDashboard; ctx: DrillContext }) {
  const s = ra.pipeline;
  const funnel = data.funnel;
  const applied = funnel[0]?.reached ?? 0;
  const hired = funnel.at(-1)?.reached ?? 0;
  const medians = new Map(data.stageDurations.map((d) => [d.stage, d]));
  const cycle = funnel.slice(0, -1).reduce((a, f) => a + (medians.get(f.stage)?.medianDays ?? 0), 0);
  const exits = funnel.reduce((a, f) => a + f.exited, 0);
  const conversion = rate(hired, applied);

  // opening x status, top 10 openings by applications, the rest folded
  const byOpening = new Map<string, CategoryRow & { total: number }>();
  for (const r of data.openingStages) {
    const status = r.status === "archived" ? "talent_pool" : r.status;
    const row = byOpening.get(r.jobOpeningId) ?? { label: r.title || ra.sources.others, href: pipelineHref(ctx, r.jobOpeningId), total: 0 };
    row[status] = Number(row[status] ?? 0) + r.count;
    row.total += r.count;
    byOpening.set(r.jobOpeningId, row);
  }
  const openingRows = topN([...byOpening.values()], 10, (r) => r.total, (rest) => {
    const other: CategoryRow & { total: number } = { label: ra.sources.others, total: 0 };
    for (const r of rest) {
      for (const st of STACK) other[st] = Number(other[st] ?? 0) + Number(r[st] ?? 0);
      other.total += r.total;
    }
    return other;
  });

  return (
    <div className="flex flex-col gap-4 sm:gap-5">
      <StatStrip
        items={[
          { label: s.conversion, value: conversion == null ? "–" : formatPercent(conversion, 1), caption: `${formatNumber(hired)} dari ${formatNumber(applied)}` },
          { label: s.medianCycle, value: ra.days(formatDecimal(cycle, 1)) },
          { label: s.stuckCount, value: formatNumber(data.stuck.length), caption: `≥ ${data.stuckAfterDays} hari` },
          { label: s.exits, value: formatNumber(exits), caption: applied ? formatPercent((exits / applied) * 100, 0) : undefined },
        ]}
      />

      <div className="grid gap-4 sm:gap-5 xl:grid-cols-2">
        <ChartCard title={s.funnel} description={s.funnelHint}>
          <FunnelChart
            steps={funnel.map((f) => ({
              key: f.stage,
              label: CANDIDATE_STATUS[f.stage] ?? f.label,
              value: f.reached,
              exited: f.exited,
              href: candidatesHref(ctx, { status: f.stage }),
            }))}
          />
        </ChartCard>
        <ChartCard title={s.duration} description={s.durationHint}>
          <BoxPlot
            unit=" hr"
            format={(v) => ra.days(formatDecimal(v, 1))}
            rows={funnel
              .map((f) => medians.get(f.stage))
              .filter((d) => d != null)
              .map((d) => ({
                label: CANDIDATE_STATUS[d.stage] ?? d.label,
                min: d.minDays,
                p25: d.p25Days,
                median: d.medianDays,
                p75: d.p75Days,
                max: d.maxDays,
                avg: d.avgDays,
                count: d.count,
              }))}
          />
        </ChartCard>
      </div>

      <ChartCard title={s.byOpening} description={s.byOpeningHint}>
        <StackedBarChart
          horizontal
          categoryWidth={180}
          data={openingRows}
          series={STACK.map((st) => ({ key: st, label: CANDIDATE_STATUS[st] ?? st, format: formatNumber }))}
          valueFormat={formatNumber}
        />
      </ChartCard>

      <ChartCard title={s.stages}>
        <AnalyticsTable<FunnelStageStat>
          rows={funnel}
          rowKey={(r) => r.stage}
          rowHref={(r) => candidatesHref(ctx, { status: r.stage })}
          columns={[
            { key: "stage", header: s.cols.stage, cell: (r) => <span className="font-medium">{CANDIDATE_STATUS[r.stage] ?? r.label}</span> },
            { key: "reached", header: s.cols.reached, align: "right", cell: (r) => formatNumber(r.reached), sort: (r) => r.reached },
            {
              key: "conv",
              header: s.cols.conversion,
              align: "right",
              cell: (r) => {
                const i = funnel.indexOf(r);
                const c = i > 0 ? rate(r.reached, funnel[i - 1]!.reached) : null;
                return c == null ? "–" : formatPercent(c, 0);
              },
            },
            {
              key: "drop",
              header: s.cols.dropped,
              align: "right",
              cell: (r) => {
                const i = funnel.indexOf(r);
                return i > 0 ? formatNumber(funnel[i - 1]!.reached - r.reached) : "–";
              },
            },
            { key: "current", header: s.cols.current, align: "right", cell: (r) => formatNumber(r.current), sort: (r) => r.current, total: formatNumber(funnel.reduce((a, f) => a + f.current, 0)) },
            { key: "exited", header: s.cols.exited, align: "right", cell: (r) => formatNumber(r.exited), sort: (r) => r.exited, total: formatNumber(exits) },
            {
              key: "median",
              header: s.cols.median,
              align: "right",
              cell: (r) => {
                const d = medians.get(r.stage);
                return d ? formatDecimal(d.medianDays, 1) : "–";
              },
            },
          ]}
        />
      </ChartCard>

      <ChartCard title={s.stuck(data.stuckAfterDays)}>
        <AnalyticsTable<StuckCandidate>
          rows={data.stuck}
          rowKey={(r) => r.id}
          rowHref={(r) => candidateHref(r.id)}
          initialSort={{ key: "days", desc: true }}
          emptyMessage={s.stuckEmpty}
          columns={[
            {
              key: "name",
              header: s.cols.candidate,
              cell: (r) => (
                <Link href={candidateHref(r.id)} className="font-medium hover:underline" onClick={(e) => e.stopPropagation()}>
                  {r.fullName}
                </Link>
              ),
              sort: (r) => r.fullName,
            },
            { key: "opening", header: s.cols.opening, cell: (r) => r.jobOpeningTitle || ra.sources.others, sort: (r) => r.jobOpeningTitle },
            { key: "stage", header: s.cols.stage, cell: (r) => <CandidateBadge status={r.status} />, sort: (r) => r.status },
            { key: "days", header: s.cols.daysInStage, align: "right", cell: (r) => formatNumber(r.daysInStage), sort: (r) => r.daysInStage },
            { key: "since", header: s.cols.since, cell: (r) => formatDate(r.enteredAt), sort: (r) => r.enteredAt },
          ]}
        />
      </ChartCard>
    </div>
  );
}
