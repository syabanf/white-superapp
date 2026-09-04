"use client";

import type { ColumnDef } from "@tanstack/react-table";
import { ChartCard } from "@/components/dashboard/chart-card";
import { TimeSeriesChart } from "@/components/dashboard/charts/time-series-chart";
import { MUTED_SERIES, slotColor } from "@/components/dashboard/charts/chart-primitives";
import { DataTable, type ColMeta } from "@/components/dashboard/data-table";
import { formatCompact, formatDate, formatNumber } from "@/lib/format";
import { t } from "@/i18n/id";
import { s } from "@/features/seo/strings";
import type { Ga4DailyPoint, SearchDailyPoint } from "@/features/seo/queries";

const right: ColMeta = { align: "right" };

function dailyColumns(label: string, key: keyof SearchDailyPoint, prevKey: keyof SearchDailyPoint, compare: boolean, digits = 0): ColumnDef<SearchDailyPoint, unknown>[] {
  const cols: ColumnDef<SearchDailyPoint, unknown>[] = [
    { accessorKey: "date", header: t.common.date, cell: ({ row }) => formatDate(row.original.date) },
    { accessorKey: key, header: label, meta: right, cell: ({ row }) => formatNumber(row.original[key] as number | null, digits) },
  ];
  if (compare) {
    cols.push({ accessorKey: prevKey, header: t.common.previousPeriod, meta: right, cell: ({ row }) => formatNumber(row.original[prevKey] as number | null, digits) });
  }
  return cols;
}

/** (a) Klik & impresi — two small multiples (NOT one dual-axis chart), previous overlay. */
export function SearchTrendCards({ daily, compare }: { daily: SearchDailyPoint[]; compare: boolean }) {
  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <ChartCard
        title={t.seo.clicks}
        description={s.clicksCardDesc}
        table={<DataTable bare dense data={daily} columns={dailyColumns(t.seo.clicks, "clicks", "clicksPrev", compare)} pageSize={10} />}
      >
        <TimeSeriesChart
          data={daily}
          series={[{ key: "clicks", label: t.seo.clicks, type: "area", color: slotColor(0), format: (v) => formatNumber(v) }]}
          previousKey={compare ? "clicksPrev" : undefined}
          height={220}
        />
      </ChartCard>
      <ChartCard
        title={t.seo.impressions}
        description={s.impressionsCardDesc}
        table={<DataTable bare dense data={daily} columns={dailyColumns(t.seo.impressions, "impressions", "impressionsPrev", compare)} pageSize={10} />}
      >
        <TimeSeriesChart
          data={daily}
          series={[{ key: "impressions", label: t.seo.impressions, type: "area", color: slotColor(2), format: (v) => formatCompact(v) }]}
          previousKey={compare ? "impressionsPrev" : undefined}
          height={220}
        />
      </ChartCard>
    </div>
  );
}

/** (b) Tren posisi rata-rata — reversed Y (1 is best), previous overlay. */
export function PositionTrendCard({ daily, compare }: { daily: SearchDailyPoint[]; compare: boolean }) {
  return (
    <ChartCard
      title={t.seo.positionChart}
      description={s.positionCardDesc}
      table={<DataTable bare dense data={daily} columns={dailyColumns(t.seo.position, "position", "positionPrev", compare, 1)} pageSize={10} />}
    >
      <TimeSeriesChart
        data={daily}
        series={[{ key: "position", label: t.seo.position, type: "line", color: slotColor(0), format: (v) => formatNumber(v, 1) }]}
        previousKey={compare ? "positionPrev" : undefined}
        reverseY
        yFormat={(v) => formatNumber(v, 0)}
        height={220}
      />
    </ChartCard>
  );
}

/** (e) GA4 — sesi organik vs kanal lain, stacked daily bars (part-to-whole). */
export function Ga4SessionsCard({ daily }: { daily: Ga4DailyPoint[] }) {
  const columns: ColumnDef<Ga4DailyPoint, unknown>[] = [
    { accessorKey: "date", header: t.common.date, cell: ({ row }) => formatDate(row.original.date) },
    { accessorKey: "organik", header: t.seo.channelOrganic, meta: right, cell: ({ row }) => formatNumber(row.original.organik) },
    { accessorKey: "lainnya", header: t.seo.channelOther, meta: right, cell: ({ row }) => formatNumber(row.original.lainnya) },
  ];
  return (
    <ChartCard title={s.ga4CardTitle} description={s.ga4CardDesc} table={<DataTable bare dense data={daily} columns={columns} pageSize={10} />}>
      <TimeSeriesChart
        data={daily}
        series={[
          { key: "organik", label: t.seo.channelOrganic, type: "bar", color: slotColor(0), format: (v) => formatNumber(v) },
          { key: "lainnya", label: t.seo.channelOther, type: "bar", color: MUTED_SERIES, format: (v) => formatNumber(v) },
        ]}
        stacked
        height={220}
      />
    </ChartCard>
  );
}
