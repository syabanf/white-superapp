"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { ChartCard } from "@/components/dashboard/chart-card";
import { DataTable } from "@/components/dashboard/data-table";
import { TimeSeriesChart } from "@/components/dashboard/charts/time-series-chart";
import { formatDate, formatNumber } from "@/lib/format";
import type { CompetitorChart } from "@/features/social/queries";
import { t } from "@/i18n/id";

type Point = CompetitorChart["data"][number];

/**
 * Perbandingan pertumbuhan followers (maks. 4 seri) — diindeks ke 100 di awal
 * periode agar satu sumbu tetap adil untuk akun dengan skala berbeda.
 */
export function CompetitorGrowthCard({ chart }: { chart: CompetitorChart }) {
  const columns = React.useMemo<ColumnDef<Point, unknown>[]>(
    () => [
      { accessorKey: "date", header: t.common.date, cell: ({ getValue }) => formatDate(String(getValue())) },
      ...chart.series.map<ColumnDef<Point, unknown>>((sd) => ({
        accessorKey: sd.key,
        header: sd.label,
        meta: { align: "right" },
        cell: ({ getValue }) => {
          const v = getValue() as number | null;
          return v == null ? "–" : formatNumber(v, 1);
        },
      })),
    ],
    [chart.series],
  );

  return (
    <ChartCard
      title={t.social.growthOverlay}
      description={t.social.growthOverlayDesc}
      table={<DataTable columns={columns} data={chart.data} bare dense paginate={false} />}
    >
      <TimeSeriesChart
        data={chart.data}
        series={chart.series.map((sd) => ({
          key: sd.key,
          label: sd.label,
          type: "line" as const,
          format: (v: number) => formatNumber(v, 1),
        }))}
        yDomain={["auto", "auto"]}
        yFormat={(v) => formatNumber(v, 0)}
        height={280}
      />
    </ChartCard>
  );
}
