"use client";

import type { ColumnDef } from "@tanstack/react-table";
import { ChartCard } from "@/components/dashboard/chart-card";
import { TimeSeriesChart } from "@/components/dashboard/charts/time-series-chart";
import { DataTable, type ColMeta } from "@/components/dashboard/data-table";
import { formatDate, formatNumber } from "@/lib/format";
import { t } from "@/i18n/id";
import type { IssueTrendPoint } from "@/features/seo/queries";
import { s } from "@/features/seo-suite/strings";

const right: ColMeta = { align: "right" };

/** Issues per crawl by severity (stacked bars) — fed by scheduled audits. */
export function IssueTrendChart({ history }: { history: IssueTrendPoint[] }) {
  const columns: ColumnDef<IssueTrendPoint, unknown>[] = [
    { accessorKey: "date", header: t.common.date, cell: ({ row }) => formatDate(row.original.date) },
    { accessorKey: "errors", header: s.errorsLabel, meta: right },
    { accessorKey: "warnings", header: t.common.warning, meta: right },
    { accessorKey: "notices", header: t.common.notice, meta: right },
    { accessorKey: "pages", header: t.seo.pagesCrawled, meta: right },
  ];
  return (
    <ChartCard title={s.issueTrend} description={s.issueTrendDesc} table={<DataTable bare dense data={history} columns={columns} paginate={false} />}>
      <TimeSeriesChart
        data={history}
        series={[
          { key: "errors", label: s.errorsLabel, type: "bar", color: "var(--status-critical)", format: (v) => formatNumber(v) },
          { key: "warnings", label: t.common.warning, type: "bar", color: "var(--status-warning)", format: (v) => formatNumber(v) },
          { key: "notices", label: t.common.notice, type: "bar", color: "var(--chart-emphasis-muted)", format: (v) => formatNumber(v) },
        ]}
        stacked
        yFormat={(v) => formatNumber(v)}
        height={200}
      />
    </ChartCard>
  );
}
