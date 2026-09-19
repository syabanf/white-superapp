"use client";

import Link from "next/link";
import { ArrowUpRight } from "lucide-react";
import type { ColumnDef } from "@tanstack/react-table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { ChartCard } from "@/components/dashboard/chart-card";
import { TimeSeriesChart } from "@/components/dashboard/charts/time-series-chart";
import { slotColor } from "@/components/dashboard/charts/chart-primitives";
import { DataTable, type ColMeta } from "@/components/dashboard/data-table";
import { formatCompact, formatDate, formatNumber } from "@/lib/format";
import { t } from "@/i18n/id";
import type { BenchmarkRow, CompetitorData } from "@/features/seo-suite/queries";
import { s } from "@/features/seo-suite/strings";

const right: ColMeta = { align: "right" };
const num = (v: number | null) =>
  v == null ? (
    <span className="text-muted-foreground">–</span>
  ) : (
    <span className="tabular">{formatNumber(v)}</span>
  );

export function BenchmarkTable({ rows, gapHref }: { rows: BenchmarkRow[]; gapHref: string }) {
  const columns: ColumnDef<BenchmarkRow, unknown>[] = [
    {
      accessorKey: "domain",
      header: s.domainLabel,
      cell: ({ row }) => (
        <span className="inline-flex items-center gap-2">
          <span className={row.original.isOwn ? "font-semibold" : ""}>{row.original.domain}</span>
          {row.original.isOwn ? (
            <Badge variant="secondary" className="text-xs">
              {s.yourDomain}
            </Badge>
          ) : null}
        </span>
      ),
    },
    {
      accessorKey: "organicKeywords",
      header: s.organicKeywords,
      meta: right,
      cell: ({ row }) => num(row.original.organicKeywords),
    },
    {
      accessorKey: "organicTraffic",
      header: s.organicTraffic,
      meta: right,
      cell: ({ row }) =>
        row.original.organicTraffic == null ? (
          "–"
        ) : (
          <span className="tabular font-medium">{formatCompact(row.original.organicTraffic)}</span>
        ),
    },
    { accessorKey: "top3", header: s.top3, meta: right, cell: ({ row }) => num(row.original.top3) },
    { accessorKey: "top10", header: s.top10, meta: right, cell: ({ row }) => num(row.original.top10) },
    {
      accessorKey: "backlinks",
      header: s.backlinksKpi,
      meta: right,
      cell: ({ row }) =>
        row.original.backlinks == null ? (
          "–"
        ) : (
          <span className="tabular">{formatCompact(row.original.backlinks)}</span>
        ),
    },
    {
      accessorKey: "referringDomains",
      header: s.referringDomains,
      meta: right,
      cell: ({ row }) => num(row.original.referringDomains),
    },
    { accessorKey: "domainRank", header: s.dr, meta: right, cell: ({ row }) => num(row.original.domainRank) },
    {
      accessorKey: "date",
      header: t.common.date,
      cell: ({ row }) => (
        <span className="text-xs text-muted-foreground">
          {row.original.date ? formatDate(row.original.date) : "–"}
        </span>
      ),
    },
    {
      id: "gap",
      header: "",
      enableSorting: false,
      cell: ({ row }) =>
        row.original.isOwn ? null : (
          <Button asChild variant="ghost" size="xs" className="group/nudge">
            <Link href={`${gapHref}?gap=${encodeURIComponent(row.original.domain)}#gap`}>
              {s.gapLink} <ArrowUpRight className="nudge size-3.5" />
            </Link>
          </Button>
        ),
    },
  ];
  return (
    <Card className="gap-0 py-0">
      <CardContent className="p-6">
        <DataTable
          data={rows}
          columns={columns}
          exportName="benchmark-domain"
          paginate={false}
          initialSorting={[{ id: "organicTraffic", desc: true }]}
          emptyMessage={t.common.noData}
          getRowId={(r) => r.domain}
        />
      </CardContent>
    </Card>
  );
}

export function TrafficChart({ traffic }: { traffic: CompetitorData["traffic"] }) {
  const columns: ColumnDef<CompetitorData["traffic"]["data"][number], unknown>[] = [
    { accessorKey: "date", header: t.common.date, cell: ({ row }) => formatDate(row.original.date) },
    ...traffic.series.map(
      (sr) =>
        ({
          accessorKey: sr.key,
          header: sr.label,
          meta: right,
          cell: ({ row }) => formatNumber(row.original[sr.key] as number | null),
        }) satisfies ColumnDef<CompetitorData["traffic"]["data"][number], unknown>,
    ),
  ];
  return (
    <ChartCard
      title={s.trafficChart}
      description={s.trafficChartDesc}
      table={<DataTable bare dense data={traffic.data} columns={columns} pageSize={14} />}
    >
      <TimeSeriesChart
        data={traffic.data}
        series={traffic.series.map((sr, i) => ({
          key: sr.key,
          label: sr.label,
          type: "line" as const,
          color: slotColor(i),
          format: (v: number) => formatNumber(v),
        }))}
        yDomain={["auto", "auto"]}
        height={260}
      />
    </ChartCard>
  );
}
