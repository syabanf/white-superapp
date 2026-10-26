"use client";

import type { ColumnDef } from "@tanstack/react-table";
import { ChartCard } from "@/components/dashboard/chart-card";
import { DataTable, type ColMeta } from "@/components/dashboard/data-table";
import { StatStrip } from "@/components/dashboard/stat-strip";
import { formatDate, formatNumber, formatPercent } from "@/lib/format";
import { t } from "@/i18n/id";
import { s } from "@/features/seo/strings";
import type { DimensionDetail } from "@/features/seo/queries";

type Daily = DimensionDetail["daily"][number];
const right: ColMeta = { align: "right" };

const columns: ColumnDef<Daily, unknown>[] = [
  { accessorKey: "date", header: t.common.date, cell: ({ row }) => formatDate(row.original.date) },
  { accessorKey: "clicks", header: t.seo.clicks, meta: right, cell: ({ row }) => formatNumber(row.original.clicks) },
  { accessorKey: "impressions", header: t.seo.impressions, meta: right, cell: ({ row }) => formatNumber(row.original.impressions) },
  { accessorKey: "ctr", header: t.seo.ctr, meta: right, cell: ({ row }) => formatPercent(row.original.ctr) },
  { accessorKey: "position", header: t.seo.position, meta: right, cell: ({ row }) => formatNumber(row.original.position, 1) },
];

/**
 * Landing for a device or country drill. Search Console rows are stored per
 * dimension, so the keyword list itself cannot be split by device or country.
 */
export function DimensionFocus({ detail, label, compare }: { detail: DimensionDetail; label: string; compare: boolean }) {
  const d = <T,>(x: T) => (compare ? x : null);
  return (
    <ChartCard title={label} description={s.dimensionFocusDesc}>
      <div className="space-y-4">
        <StatStrip
          minWidth={140}
          items={[
            { label: t.seo.clicks, value: formatNumber(detail.totals.clicks), delta: d(detail.deltas.clicks) },
            { label: t.seo.impressions, value: formatNumber(detail.totals.impressions), delta: d(detail.deltas.impressions) },
            { label: t.seo.ctr, value: formatPercent(detail.totals.ctr), delta: d(detail.deltas.ctr) },
            {
              label: t.seo.position,
              value: formatNumber(detail.totals.position, 1),
              delta: d(detail.deltas.position),
              lowerIsBetter: true,
            },
          ]}
        />
        <DataTable bare dense data={detail.daily} columns={columns} pageSize={7} emptyMessage={t.common.noData} />
      </div>
    </ChartCard>
  );
}
