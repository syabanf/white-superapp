"use client";

import type { ColumnDef } from "@tanstack/react-table";
import { ChartCard } from "@/components/dashboard/chart-card";
import { TimeSeriesChart } from "@/components/dashboard/charts/time-series-chart";
import { slotColor } from "@/components/dashboard/charts/chart-primitives";
import { DataTable, type ColMeta } from "@/components/dashboard/data-table";
import { formatDate, formatNumber } from "@/lib/format";
import { t } from "@/i18n/id";
import { s } from "@/features/seo/strings";
import type { ScoreHistoryPoint } from "@/features/seo/queries";

const right: ColMeta = { align: "right" };

/** Riwayat skor Lighthouse (MOBILE) — sparse runs plotted as 4 lines, y locked to 0–100. */
export function ScoreHistoryChart({ history }: { history: ScoreHistoryPoint[] }) {
  const columns: ColumnDef<ScoreHistoryPoint, unknown>[] = [
    { accessorKey: "date", header: t.common.date, cell: ({ row }) => formatDate(row.original.date) },
    { accessorKey: "performance", header: t.seo.performance, meta: right },
    { accessorKey: "seo", header: t.seo.seoScore, meta: right },
    { accessorKey: "accessibility", header: t.seo.accessibility, meta: right },
    { accessorKey: "bestPractices", header: t.seo.bestPractices, meta: right },
  ];
  return (
    <ChartCard title={t.seo.scoreHistory} description={s.historyDesc} table={<DataTable bare dense data={history} columns={columns} paginate={false} />}>
      <TimeSeriesChart
        data={history}
        series={[
          { key: "performance", label: t.seo.performance, type: "line", color: slotColor(0), format: (v) => formatNumber(v) },
          { key: "seo", label: t.seo.seoScore, type: "line", color: slotColor(1), format: (v) => formatNumber(v) },
          { key: "accessibility", label: t.seo.accessibility, type: "line", color: slotColor(2), format: (v) => formatNumber(v) },
          { key: "bestPractices", label: t.seo.bestPractices, type: "line", color: slotColor(3), format: (v) => formatNumber(v) },
        ]}
        yDomain={[0, 100]}
        height={220}
      />
    </ChartCard>
  );
}
