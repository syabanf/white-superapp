"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { ChartCard } from "@/components/dashboard/chart-card";
import { DataTable } from "@/components/dashboard/data-table";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { Heatmap } from "@/components/dashboard/charts/heatmap";
import { TimeSeriesChart } from "@/components/dashboard/charts/time-series-chart";
import { formatCompact, formatDate, formatNumber, formatPercent } from "@/lib/format";
import type { ErByType } from "@/features/social/lib";
import type { GrowthPoint, HeatCellData, ReachPoint } from "@/features/social/queries";
import { s } from "@/features/social/strings";
import { t } from "@/i18n/id";

const num = (v: unknown, digits = 0) => (v == null ? "–" : formatNumber(Number(v), digits));

// ── Pertumbuhan followers ────────────────────────────────────

export function GrowthChartCard({ data }: { data: GrowthPoint[] }) {
  const columns = React.useMemo<ColumnDef<GrowthPoint, unknown>[]>(
    () => [
      { accessorKey: "date", header: t.common.date, cell: ({ getValue }) => formatDate(String(getValue())) },
      { accessorKey: "followers", header: t.social.followers, meta: { align: "right" }, cell: ({ getValue }) => num(getValue()) },
      { accessorKey: "followersPrev", header: t.common.previousPeriod, meta: { align: "right" }, cell: ({ getValue }) => num(getValue()) },
    ],
    [],
  );
  return (
    <ChartCard
      title={t.social.growthChart}
      description={t.social.growthChartDesc}
      table={<DataTable columns={columns} data={data} bare dense paginate={false} />}
    >
      <TimeSeriesChart
        data={data}
        series={[{ key: "followers", label: t.social.followers, type: "area", format: (v) => formatNumber(v) }]}
        previousKey="followersPrev"
        yDomain={["auto", "auto"]}
        height={260}
      />
    </ChartCard>
  );
}

// ── Jangkauan & impresi harian ───────────────────────────────

export function ReachChartCard({ data }: { data: ReachPoint[] }) {
  const columns = React.useMemo<ColumnDef<ReachPoint, unknown>[]>(
    () => [
      { accessorKey: "date", header: t.common.date, cell: ({ getValue }) => formatDate(String(getValue())) },
      { accessorKey: "reach", header: t.social.reach, meta: { align: "right" }, cell: ({ getValue }) => num(getValue()) },
      { accessorKey: "impressions", header: t.social.impressions, meta: { align: "right" }, cell: ({ getValue }) => num(getValue()) },
    ],
    [],
  );
  return (
    <ChartCard
      title={t.social.reachChart}
      description={s.reachChartDesc}
      table={<DataTable columns={columns} data={data} bare dense paginate={false} />}
    >
      <TimeSeriesChart
        data={data}
        series={[
          { key: "reach", label: t.social.reach, type: "area", format: (v) => formatNumber(v) },
          { key: "impressions", label: t.social.impressions, type: "line", format: (v) => formatNumber(v) },
        ]}
        height={260}
      />
    </ChartCard>
  );
}

// ── Engagement per tipe konten ───────────────────────────────

export function ErByTypeCard({ items }: { items: ErByType[] }) {
  const columns = React.useMemo<ColumnDef<ErByType, unknown>[]>(
    () => [
      { accessorKey: "type", header: t.social.type },
      { accessorKey: "avgEr", header: s.avgEr, meta: { align: "right" }, cell: ({ getValue }) => formatPercent(Number(getValue())) },
      { accessorKey: "count", header: t.social.posts, meta: { align: "right" }, cell: ({ getValue }) => num(getValue()) },
    ],
    [],
  );
  return (
    <ChartCard
      title={t.social.erByType}
      description={t.social.erByTypeDesc}
      table={<DataTable columns={columns} data={items} bare dense paginate={false} />}
    >
      <HBarChart
        items={items.map((it) => ({
          label: it.type,
          value: it.avgEr,
          secondary: `${it.count} ${s.post}`,
        }))}
        format={(v) => formatPercent(v)}
        emptyLabel={s.noPosts}
        className="py-1"
      />
    </ChartCard>
  );
}

// ── Waktu posting terbaik (heatmap hari × jam) ───────────────

export function PostingHeatmapCard({ cells }: { cells: HeatCellData[] }) {
  const active = React.useMemo(
    () =>
      cells
        .filter((c) => c.posts > 0)
        .sort((a, b) => b.avgEngagement - a.avgEngagement)
        .map((c) => ({
          day: s.days[c.day] ?? String(c.day),
          hour: `${String(c.hour).padStart(2, "0")}:00`,
          posts: c.posts,
          avgEngagement: Math.round(c.avgEngagement),
        })),
    [cells],
  );
  const columns = React.useMemo<ColumnDef<(typeof active)[number], unknown>[]>(
    () => [
      { accessorKey: "day", header: s.day },
      { accessorKey: "hour", header: s.hour },
      { accessorKey: "posts", header: s.heatmapCount, meta: { align: "right" }, cell: ({ getValue }) => num(getValue()) },
      { accessorKey: "avgEngagement", header: s.heatmapValue, meta: { align: "right" }, cell: ({ getValue }) => num(getValue()) },
    ],
    [],
  );
  return (
    <ChartCard
      title={t.social.heatmap}
      description={t.social.heatmapDesc}
      table={<DataTable columns={columns} data={active} bare dense paginate={false} emptyMessage={s.noPosts} />}
    >
      <Heatmap
        cells={cells.map((c) => ({ row: c.day, col: c.hour, value: c.avgEngagement, count: c.posts }))}
        rows={7}
        cols={24}
        rowLabels={[...s.days]}
        colLabels={Array.from({ length: 24 }, (_, h) => String(h).padStart(2, "0"))}
        format={(v) => formatCompact(v)}
        valueLabel={s.heatmapValue}
        countLabel={s.heatmapCount}
      />
    </ChartCard>
  );
}
