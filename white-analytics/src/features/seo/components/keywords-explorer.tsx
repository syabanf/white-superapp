"use client";

import Link from "next/link";
import { Info } from "lucide-react";
import type { ColumnDef } from "@tanstack/react-table";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { DataTable, type ColMeta } from "@/components/dashboard/data-table";
import { DeltaBadge } from "@/components/dashboard/delta-badge";
import { computeDelta, type PositionBucket } from "@/lib/metrics";
import { formatNumber, formatPercent, truncate } from "@/lib/format";
import { cn } from "@/lib/utils";
import { t } from "@/i18n/id";
import { s } from "@/features/seo/strings";
import type { KeywordTableRow, KeywordViewKey } from "@/features/seo/aggregate";
import { BUCKET_COLORS, BUCKET_LABELS } from "./buckets";

const right: ColMeta = { align: "right" };

const VIEW_LABELS: Record<KeywordViewKey, string> = {
  all: t.common.all,
  striking: t.seo.strikingDistance,
  low_ctr: t.seo.lowCtr,
  declining: t.seo.declining,
  rising: t.seo.rising,
};

const VIEW_DESCRIPTIONS: Record<KeywordViewKey, string> = {
  all: s.allDesc,
  striking: t.seo.strikingDistanceDesc,
  low_ctr: t.seo.lowCtrDesc,
  declining: t.seo.decliningDesc,
  rising: t.seo.risingDesc,
};

type DistributionRow = { bucket: PositionBucket; queries: number };

function ViewTabs({ view, counts, basePath }: { view: KeywordViewKey; counts: Record<KeywordViewKey, number>; basePath: string }) {
  const views: KeywordViewKey[] = ["all", "striking", "low_ctr", "declining", "rising"];
  return (
    <div className="flex w-fit max-w-full flex-wrap items-center gap-1 rounded-lg bg-muted p-1">
      {views.map((v) => {
        const active = v === view;
        return (
          <Link
            key={v}
            href={v === "all" ? basePath : `${basePath}?view=${v}`}
            aria-current={active ? "page" : undefined}
            className={cn(
              "inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-medium transition-colors",
              active ? "bg-background text-foreground shadow-sm" : "text-muted-foreground hover:text-foreground",
            )}
          >
            {VIEW_LABELS[v]}
            <span className={cn("tabular text-[10px]", active ? "text-muted-foreground" : "text-muted-foreground/70")}>{formatNumber(counts[v])}</span>
          </Link>
        );
      })}
    </div>
  );
}

/** Compact stacked strip of the query count per position bucket. */
function DistributionStrip({ distribution }: { distribution: DistributionRow[] }) {
  const total = distribution.reduce((a, b) => a + b.queries, 0);
  if (total === 0) return null;
  return (
    <div className="space-y-1.5">
      <div className="flex h-2 w-full max-w-md overflow-hidden rounded-full bg-muted">
        {distribution.map((d) => (
          <div key={d.bucket} style={{ width: `${(d.queries / total) * 100}%`, background: BUCKET_COLORS[d.bucket] }} title={`${BUCKET_LABELS[d.bucket]}: ${d.queries}`} />
        ))}
      </div>
      <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
        {distribution.map((d) => (
          <span key={d.bucket} className="inline-flex items-center gap-1.5">
            <span aria-hidden className="size-2 rounded-[2px]" style={{ background: BUCKET_COLORS[d.bucket] }} />
            {BUCKET_LABELS[d.bucket]} · <span className="tabular">{formatNumber(d.queries)}</span>
          </span>
        ))}
      </div>
    </div>
  );
}

export function KeywordsExplorer({
  view,
  rows,
  counts,
  distribution,
  compare,
  basePath,
}: {
  view: KeywordViewKey;
  rows: KeywordTableRow[];
  counts: Record<KeywordViewKey, number>;
  distribution: DistributionRow[];
  compare: boolean;
  basePath: string;
}) {
  const showExpected = view === "striking" || view === "low_ctr";
  const showPotential = view === "striking" || view === "low_ctr";

  const columns: ColumnDef<KeywordTableRow, unknown>[] = [
    {
      accessorKey: "key",
      header: t.seo.query,
      cell: ({ row }) => <span title={row.original.key}>{truncate(row.original.key, 48)}</span>,
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
    ...(showExpected
      ? [
          {
            accessorKey: "expectedCtr",
            header: t.seo.expectedCtr,
            meta: right,
            cell: ({ row }) => (row.original.expectedCtr == null ? "–" : formatPercent(row.original.expectedCtr, 1)),
          } satisfies ColumnDef<KeywordTableRow, unknown>,
        ]
      : []),
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
    ...(showPotential
      ? [
          {
            accessorKey: "potentialClicks",
            header: t.seo.potentialClicks,
            meta: right,
            cell: ({ row }) => (row.original.potentialClicks == null ? "–" : <span className="font-medium tabular">{formatNumber(row.original.potentialClicks)}</span>),
          } satisfies ColumnDef<KeywordTableRow, unknown>,
        ]
      : []),
    {
      accessorKey: "bucket",
      header: s.bucket,
      cell: ({ row }) => (
        <Badge variant="outline" className="gap-1.5 font-normal text-muted-foreground">
          <span aria-hidden className="size-2 rounded-[2px]" style={{ background: BUCKET_COLORS[row.original.bucket] }} />
          {row.original.bucket}
        </Badge>
      ),
    },
  ];

  return (
    <Card className="gap-0 py-0">
      <CardContent className="space-y-4 p-5">
        <div className="flex flex-col gap-3">
          <ViewTabs view={view} counts={counts} basePath={basePath} />
          <p className="text-sm text-muted-foreground">{VIEW_DESCRIPTIONS[view]}</p>
          <DistributionStrip distribution={distribution} />
          <p className="inline-flex items-center gap-1.5 text-xs text-muted-foreground">
            <Info className="size-3.5 shrink-0" /> {t.seo.ctrCurveHint}
            {showPotential ? ` · ${s.potentialHint}` : ""}
          </p>
        </div>
        <DataTable
          data={rows}
          columns={columns}
          searchKey="key"
          searchPlaceholder={s.searchQuery}
          exportName={`kata-kunci-${view}`}
          initialSorting={showPotential ? [{ id: "potentialClicks", desc: true }] : [{ id: "clicks", desc: true }]}
          pageSize={10}
          emptyMessage={t.common.noResults}
        />
      </CardContent>
    </Card>
  );
}
