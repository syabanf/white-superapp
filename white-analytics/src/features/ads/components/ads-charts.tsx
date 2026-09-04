"use client";

import type { ColumnDef } from "@tanstack/react-table";
import { ChartCard } from "@/components/dashboard/chart-card";
import { TimeSeriesChart } from "@/components/dashboard/charts/time-series-chart";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { DataTable } from "@/components/dashboard/data-table";
import { formatCompact, formatCurrency, formatDate, formatNumber, formatPercent } from "@/lib/format";
import { t } from "@/i18n/id";
import { resultTypeLabel, s } from "@/features/ads/strings";

export type AdsDailyPoint = { date: string; spend: number; results: number; spendPrev: number | null; resultsPrev: number | null };
export type CampaignShareItem = { id: string; name: string; spend: number; results: number; resultType: string | null };

/** (a) Belanja harian — bar slot 2 (oranye) + overlay periode sebelumnya. */
export function SpendDailyCard({ daily, currency, compare }: { daily: AdsDailyPoint[]; currency: string; compare: boolean }) {
  const money = (v: number) => formatCurrency(v, currency, { compact: true });
  const columns: ColumnDef<AdsDailyPoint, unknown>[] = [
    { accessorKey: "date", header: t.common.date, cell: ({ row }) => formatDate(row.original.date) },
    { accessorKey: "spend", header: t.ads.spend, cell: ({ row }) => formatCurrency(row.original.spend, currency), meta: { align: "right" } },
    ...(compare
      ? [
          {
            accessorKey: "spendPrev",
            header: t.common.previousPeriod,
            cell: ({ row }) => (row.original.spendPrev == null ? "–" : formatCurrency(row.original.spendPrev, currency)),
            meta: { align: "right" },
          } satisfies ColumnDef<AdsDailyPoint, unknown>,
        ]
      : []),
  ];
  return (
    <ChartCard
      title={t.ads.spendChart}
      description={compare ? s.spendDailyDesc : `${t.ads.spendChart} · ${t.common.currentPeriod}`}
      table={<DataTable bare dense columns={columns} data={daily} pageSize={daily.length} paginate={false} />}
    >
      <TimeSeriesChart
        data={daily}
        series={[{ key: "spend", label: t.ads.spend, type: "bar", color: "var(--chart-2)", format: money }]}
        previousKey={compare ? "spendPrev" : undefined}
        yFormat={(v) => formatCompact(v)}
        height={240}
      />
    </ChartCard>
  );
}

/** (b) Hasil harian — bar slot 1 (small multiple terpisah, bukan dual axis). */
export function ResultsDailyCard({ daily }: { daily: AdsDailyPoint[] }) {
  const columns: ColumnDef<AdsDailyPoint, unknown>[] = [
    { accessorKey: "date", header: t.common.date, cell: ({ row }) => formatDate(row.original.date) },
    { accessorKey: "results", header: t.ads.results, cell: ({ row }) => formatNumber(row.original.results), meta: { align: "right" } },
  ];
  return (
    <ChartCard
      title={t.ads.resultsChart}
      description={s.resultsDailyDesc}
      table={<DataTable bare dense columns={columns} data={daily} pageSize={daily.length} paginate={false} />}
    >
      <TimeSeriesChart
        data={daily}
        series={[{ key: "results", label: t.ads.results, type: "bar", format: (v) => formatNumber(v) }]}
        yFormat={(v) => formatCompact(v)}
        height={240}
      />
    </ChartCard>
  );
}

/** (c) Belanja per kampanye — HBar dengan porsi (share). */
export function CampaignShareCard({ items, currency }: { items: CampaignShareItem[]; currency: string }) {
  const total = items.reduce((a, b) => a + b.spend, 0);
  const columns: ColumnDef<CampaignShareItem, unknown>[] = [
    { accessorKey: "name", header: t.ads.campaign },
    { accessorKey: "spend", header: t.ads.spend, cell: ({ row }) => formatCurrency(row.original.spend, currency), meta: { align: "right" } },
    {
      id: "share",
      header: s.shareLabel,
      cell: ({ row }) => formatPercent(total > 0 ? (row.original.spend / total) * 100 : 0, 1),
      meta: { align: "right" },
    },
    {
      accessorKey: "results",
      header: t.ads.results,
      cell: ({ row }) => `${formatNumber(row.original.results)} ${resultTypeLabel(row.original.resultType).toLowerCase()}`,
      meta: { align: "right" },
    },
  ];
  return (
    <ChartCard
      title={s.spendByCampaign}
      description={s.spendByCampaignDesc}
      table={<DataTable bare dense columns={columns} data={items} pageSize={Math.max(items.length, 1)} paginate={false} />}
    >
      <HBarChart
        items={items.map((c) => ({ label: c.name, value: c.spend }))}
        format={(v) => formatCurrency(v, currency, { compact: true })}
        showShare
        emptyLabel={t.common.noData}
      />
    </ChartCard>
  );
}
