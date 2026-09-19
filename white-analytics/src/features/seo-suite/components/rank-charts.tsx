"use client";

import type { ColumnDef } from "@tanstack/react-table";
import { ChartCard } from "@/components/dashboard/chart-card";
import { TimeSeriesChart } from "@/components/dashboard/charts/time-series-chart";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { DataTable, type ColMeta } from "@/components/dashboard/data-table";
import { formatDate, formatNumber, formatPercent } from "@/lib/format";
import type { RankBucket } from "@/lib/metrics/seo-suite";
import { t } from "@/i18n/id";
import type { RankDailyPoint } from "@/features/seo-suite/rank-aggregate";
import { BUCKET_LABELS } from "@/features/seo-suite/lib";
import { s } from "@/features/seo-suite/strings";

const right: ColMeta = { align: "right" };

export function RankTrendCards({ daily, compare }: { daily: RankDailyPoint[]; compare: boolean }) {
  const columns: ColumnDef<RankDailyPoint, unknown>[] = [
    { accessorKey: "date", header: t.common.date, cell: ({ row }) => formatDate(row.original.date) },
    { accessorKey: "visibility", header: s.visibility, meta: right, cell: ({ row }) => formatPercent(row.original.visibility, 1) },
    { accessorKey: "avgPosition", header: t.seo.position, meta: right, cell: ({ row }) => formatNumber(row.original.avgPosition, 1) },
  ];
  const table = <DataTable bare dense data={daily} columns={columns} pageSize={14} />;
  return (
    <div className="grid gap-5 lg:grid-cols-2">
      <ChartCard title={s.visibilityChart} description={s.visibilityChartDesc} table={table}>
        <TimeSeriesChart
          data={daily}
          series={[{ key: "visibility", label: s.visibility, type: "area", format: (v) => formatPercent(v, 1) }]}
          previousKey={compare ? "visibilityPrev" : undefined}
          yFormat={(v) => formatPercent(v, 0)}
          height={220}
        />
      </ChartCard>
      <ChartCard title={t.seo.positionChart} description={s.positionChartDesc} table={table}>
        <TimeSeriesChart
          data={daily}
          series={[{ key: "avgPosition", label: t.seo.position, type: "line", format: (v) => formatNumber(v, 1) }]}
          previousKey={compare ? "avgPositionPrev" : undefined}
          yFormat={(v) => formatNumber(v, 0)}
          reverseY
          height={220}
        />
      </ChartCard>
    </div>
  );
}

export function RankBreakdownCards({ distribution, sov }: { distribution: { bucket: RankBucket; count: number }[]; sov: { domain: string; weight: number; share: number }[] }) {
  const distColumns: ColumnDef<{ bucket: RankBucket; count: number }, unknown>[] = [
    { accessorKey: "bucket", header: t.seo.position, cell: ({ row }) => BUCKET_LABELS[row.original.bucket] },
    { accessorKey: "count", header: s.keywordsTable, meta: right },
  ];
  const sovColumns: ColumnDef<{ domain: string; weight: number; share: number }, unknown>[] = [
    { accessorKey: "domain", header: s.domainLabel },
    { accessorKey: "share", header: s.sovTitle, meta: right, cell: ({ row }) => formatPercent(row.original.share, 1) },
  ];
  return (
    <div className="grid gap-5 lg:grid-cols-2">
      <ChartCard title={s.distributionTitle} description={s.distributionDesc} table={<DataTable bare dense data={distribution} columns={distColumns} paginate={false} />}>
        <HBarChart items={distribution.map((d) => ({ label: BUCKET_LABELS[d.bucket], value: d.count, muted: d.bucket === "unranked" }))} format={(v) => formatNumber(v)} className="py-1" emptyLabel={t.common.noData} />
      </ChartCard>
      <ChartCard title={s.sovTitle} description={s.sovDesc} table={<DataTable bare dense data={sov} columns={sovColumns} paginate={false} />}>
        <HBarChart items={sov.map((r) => ({ label: r.domain, value: r.share, muted: r.domain === s.others }))} format={(v) => formatPercent(v, 1)} className="py-1" emptyLabel={t.common.noData} />
      </ChartCard>
    </div>
  );
}
