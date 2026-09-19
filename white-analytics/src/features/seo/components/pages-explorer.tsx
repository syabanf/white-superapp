"use client";

import type { ColumnDef } from "@tanstack/react-table";
import { Card, CardContent } from "@/components/ui/card";
import { ChartCard } from "@/components/dashboard/chart-card";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { DataTable, type ColMeta } from "@/components/dashboard/data-table";
import { DeltaBadge } from "@/components/dashboard/delta-badge";
import { computeDelta } from "@/lib/metrics";
import { formatNumber, formatPercent, truncate } from "@/lib/format";
import { t } from "@/i18n/id";
import { s } from "@/features/seo/strings";
import type { PageAgg } from "@/features/seo/queries";

const right: ColMeta = { align: "right" };

/** "Klik per halaman" — top 10 pages by organic clicks. */
export function ClicksPerPageCard({ rows }: { rows: PageAgg[] }) {
  const top = rows.slice(0, 10);
  const columns: ColumnDef<PageAgg, unknown>[] = [
    { accessorKey: "path", header: t.seo.page, cell: ({ row }) => <span title={row.original.key}>{truncate(row.original.path, 48)}</span> },
    { accessorKey: "clicks", header: t.seo.clicks, meta: right, cell: ({ row }) => formatNumber(row.original.clicks) },
    { accessorKey: "impressions", header: t.seo.impressions, meta: right, cell: ({ row }) => formatNumber(row.original.impressions) },
  ];
  return (
    <ChartCard title={s.clicksPerPage} description={s.clicksPerPageDesc} table={<DataTable bare dense data={top} columns={columns} paginate={false} />}>
      <HBarChart items={top.map((r) => ({ label: r.path, value: r.clicks, href: r.key, secondary: formatPercent(r.ctr, 1) }))} className="py-1" />
    </ChartCard>
  );
}

/** Full PAGE-dimension table with deltas and CSV export. */
export function PagesTable({ rows, compare }: { rows: PageAgg[]; compare: boolean }) {
  const columns: ColumnDef<PageAgg, unknown>[] = [
    {
      accessorKey: "path",
      header: t.seo.page,
      cell: ({ row }) => (
        <span title={row.original.key} className="font-medium">
          {truncate(row.original.path, 56)}
        </span>
      ),
    },
    {
      accessorKey: "clicks",
      header: t.seo.clicks,
      meta: right,
      cell: ({ row }) => (
        <span className="inline-flex items-center justify-end gap-1.5">
          <span className="font-medium tabular">{formatNumber(row.original.clicks)}</span>
          {compare && row.original.prevClicks != null ? <DeltaBadge delta={computeDelta(row.original.clicks, row.original.prevClicks)} mode="abs" /> : null}
        </span>
      ),
    },
    { accessorKey: "impressions", header: t.seo.impressions, meta: right, cell: ({ row }) => formatNumber(row.original.impressions) },
    { accessorKey: "ctr", header: t.seo.ctr, meta: right, cell: ({ row }) => formatPercent(row.original.ctr) },
    {
      accessorKey: "position",
      header: t.seo.position,
      meta: right,
      cell: ({ row }) => (
        <span className="inline-flex items-center justify-end gap-1.5">
          <span className="tabular">{formatNumber(row.original.position, 1)}</span>
          {compare && row.original.prevPosition != null ? (
            <DeltaBadge delta={computeDelta(row.original.position, row.original.prevPosition)} mode="abs" digits={1} lowerIsBetter />
          ) : null}
        </span>
      ),
    },
  ];
  return (
    <Card className="gap-0 py-0">
      <CardContent className="p-6">
        <DataTable
          data={rows}
          columns={columns}
          searchKey="path"
          searchPlaceholder={s.searchPage}
          exportName="halaman-seo"
          initialSorting={[{ id: "clicks", desc: true }]}
          pageSize={10}
          emptyMessage={t.common.noData}
        />
      </CardContent>
    </Card>
  );
}
