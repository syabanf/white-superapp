"use client";

import Link from "next/link";
import { ArrowUpRight } from "lucide-react";
import type { ColumnDef } from "@tanstack/react-table";
import { Button } from "@/components/ui/button";
import { ChartCard } from "@/components/dashboard/chart-card";
import { DataTable, type ColMeta } from "@/components/dashboard/data-table";
import { DeltaBadge } from "@/components/dashboard/delta-badge";
import { computeDelta } from "@/lib/metrics";
import { formatNumber, formatPercent, truncate } from "@/lib/format";
import { t } from "@/i18n/id";
import type { PageAgg } from "@/features/seo/queries";
import type { QueryAgg } from "@/features/seo/aggregate";

const right: ColMeta = { align: "right" };

function ViewAllLink({ href }: { href: string }) {
  return (
    <Button asChild variant="ghost" size="xs" className="group/nudge">
      <Link href={href}>
        {t.common.viewAll} <ArrowUpRight className="size-3.5" />
      </Link>
    </Button>
  );
}

function ClicksCell({ clicks, prevClicks, compare }: { clicks: number; prevClicks: number | null; compare: boolean }) {
  return (
    <span className="inline-flex items-center justify-end gap-1.5">
      <span className="font-medium tabular">{formatNumber(clicks)}</span>
      {compare && prevClicks != null ? <DeltaBadge delta={computeDelta(clicks, prevClicks)} mode="abs" /> : null}
    </span>
  );
}

/** Kata kunci teratas — top 10 by clicks with Δ clicks, position, CTR. */
export function TopQueriesCard({ rows, href, compare }: { rows: QueryAgg[]; href: string; compare: boolean }) {
  const columns: ColumnDef<QueryAgg, unknown>[] = [
    { accessorKey: "key", header: t.seo.query, cell: ({ row }) => <span title={row.original.key}>{truncate(row.original.key, 42)}</span> },
    {
      accessorKey: "clicks",
      header: t.seo.clicks,
      meta: right,
      cell: ({ row }) => <ClicksCell clicks={row.original.clicks} prevClicks={row.original.prevClicks} compare={compare} />,
    },
    { accessorKey: "position", header: t.seo.position, meta: right, cell: ({ row }) => formatNumber(row.original.position, 1) },
    { accessorKey: "ctr", header: t.seo.ctr, meta: right, cell: ({ row }) => formatPercent(row.original.ctr) },
  ];
  return (
    <ChartCard title={t.seo.topQueries} description={`10 ${t.seo.keywords.toLowerCase()} dengan klik terbanyak`} actions={<ViewAllLink href={href} />}>
      <DataTable bare dense data={rows.slice(0, 10)} columns={columns} paginate={false} emptyMessage={t.common.noData} />
    </ChartCard>
  );
}

/** Halaman teratas — top 10 by clicks. */
export function TopPagesCard({ rows, href, compare }: { rows: PageAgg[]; href: string; compare: boolean }) {
  const columns: ColumnDef<PageAgg, unknown>[] = [
    { accessorKey: "path", header: t.seo.page, cell: ({ row }) => <span title={row.original.key}>{truncate(row.original.path, 42)}</span> },
    {
      accessorKey: "clicks",
      header: t.seo.clicks,
      meta: right,
      cell: ({ row }) => <ClicksCell clicks={row.original.clicks} prevClicks={row.original.prevClicks} compare={compare} />,
    },
    { accessorKey: "position", header: t.seo.position, meta: right, cell: ({ row }) => formatNumber(row.original.position, 1) },
    { accessorKey: "ctr", header: t.seo.ctr, meta: right, cell: ({ row }) => formatPercent(row.original.ctr) },
  ];
  return (
    <ChartCard title={t.seo.topPages} description={`10 ${t.seo.pages.toLowerCase()} dengan klik terbanyak`} actions={<ViewAllLink href={href} />}>
      <DataTable bare dense data={rows.slice(0, 10)} columns={columns} paginate={false} emptyMessage={t.common.noData} />
    </ChartCard>
  );
}
