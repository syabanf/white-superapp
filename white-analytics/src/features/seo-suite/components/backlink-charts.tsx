"use client";

import type { ColumnDef } from "@tanstack/react-table";
import { ChartCard } from "@/components/dashboard/chart-card";
import { TimeSeriesChart } from "@/components/dashboard/charts/time-series-chart";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { slotColor } from "@/components/dashboard/charts/chart-primitives";
import { DataTable, type ColMeta } from "@/components/dashboard/data-table";
import { formatDate, formatNumber } from "@/lib/format";
import { t } from "@/i18n/id";
import type { BacklinkData } from "@/features/seo-suite/queries";
import { s } from "@/features/seo-suite/strings";

const right: ColMeta = { align: "right" };

export function BacklinkTrendCards({ daily, weekly }: { daily: BacklinkData["daily"]; weekly: BacklinkData["weekly"] }) {
  const dailyCols: ColumnDef<BacklinkData["daily"][number], unknown>[] = [
    { accessorKey: "date", header: t.common.date, cell: ({ row }) => formatDate(row.original.date) },
    { accessorKey: "referringDomains", header: s.referringDomains, meta: right, cell: ({ row }) => formatNumber(row.original.referringDomains) },
  ];
  const weeklyCols: ColumnDef<BacklinkData["weekly"][number], unknown>[] = [
    { accessorKey: "date", header: t.common.date, cell: ({ row }) => formatDate(row.original.date) },
    { accessorKey: "baru", header: s.newBacklinks, meta: right },
    { accessorKey: "hilang", header: s.lostBacklinks, meta: right },
  ];
  return (
    <div className="grid gap-5 lg:grid-cols-2">
      <ChartCard title={s.refDomainsChart} description={s.refDomainsChartDesc} table={<DataTable bare dense data={daily} columns={dailyCols} pageSize={14} />}>
        <TimeSeriesChart data={daily} series={[{ key: "referringDomains", label: s.referringDomains, type: "area", format: (v) => formatNumber(v) }]} yDomain={["auto", "auto"]} height={220} />
      </ChartCard>
      <ChartCard title={s.newLostChart} description={s.newLostChartDesc} table={<DataTable bare dense data={weekly} columns={weeklyCols} paginate={false} />}>
        <TimeSeriesChart
          data={weekly}
          series={[
            { key: "baru", label: s.newBacklinks, type: "bar", color: slotColor(0), format: (v) => formatNumber(v) },
            { key: "hilang", label: s.lostBacklinks, type: "bar", color: slotColor(1), format: (v) => formatNumber(v) },
          ]}
          yFormat={(v) => formatNumber(v)}
          height={220}
        />
      </ChartCard>
    </div>
  );
}

export function BacklinkTopCards({ topAnchors, topDomains }: { topAnchors: BacklinkData["topAnchors"]; topDomains: BacklinkData["topDomains"] }) {
  const cols: ColumnDef<{ label: string; count: number }, unknown>[] = [
    { accessorKey: "label", header: t.common.name },
    { accessorKey: "count", header: s.backlinksKpi, meta: right },
  ];
  return (
    <div className="grid gap-5 lg:grid-cols-2">
      <ChartCard title={s.topAnchors} description={s.topAnchorsDesc} table={<DataTable bare dense data={topAnchors} columns={cols} paginate={false} />}>
        <HBarChart items={topAnchors.map((a) => ({ label: a.label, value: a.count }))} format={(v) => formatNumber(v)} className="py-1" emptyLabel={t.common.noData} />
      </ChartCard>
      <ChartCard title={s.topRefDomains} description={s.topRefDomainsDesc} table={<DataTable bare dense data={topDomains} columns={cols} paginate={false} />}>
        <HBarChart items={topDomains.map((a) => ({ label: a.label, value: a.count }))} format={(v) => formatNumber(v)} className="py-1" emptyLabel={t.common.noData} />
      </ChartCard>
    </div>
  );
}
