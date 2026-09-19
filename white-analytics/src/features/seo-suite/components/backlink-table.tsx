"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { DataTable, type ColMeta } from "@/components/dashboard/data-table";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { formatDate, formatNumber, truncate } from "@/lib/format";
import { backlinkMatchesView, isToxic, type BacklinkView } from "@/lib/metrics/seo-suite";
import { t } from "@/i18n/id";
import type { BacklinkRow } from "@/features/seo-suite/queries";
import { s } from "@/features/seo-suite/strings";

const right: ColMeta = { align: "right" };
const VIEWS: BacklinkView[] = ["all", "new", "lost", "toxic"];
const LABELS: Record<BacklinkView, string> = {
  all: s.tabAll,
  new: s.tabNew,
  lost: s.tabLost,
  toxic: s.tabToxic,
};
const DESCS: Record<BacklinkView, string | null> = {
  all: null,
  new: s.tabNewDesc,
  lost: s.tabLostDesc,
  toxic: s.tabToxicDesc,
};

export function BacklinkTable({ rows, today }: { rows: BacklinkRow[]; today: string }) {
  const [view, setView] = React.useState<BacklinkView>("all");
  const now = React.useMemo(() => new Date(today), [today]);
  const counts = React.useMemo(
    () =>
      Object.fromEntries(
        VIEWS.map((v) => [v, rows.filter((r) => backlinkMatchesView(r, v, now)).length]),
      ) as Record<BacklinkView, number>,
    [rows, now],
  );
  const filtered = React.useMemo(
    () => rows.filter((r) => backlinkMatchesView(r, view, now)),
    [rows, view, now],
  );

  const columns: ColumnDef<BacklinkRow, unknown>[] = [
    {
      accessorKey: "sourceDomain",
      header: s.sourceUrl,
      cell: ({ row }) => (
        <div className="flex min-w-0 flex-col">
          <a
            href={row.original.sourceUrl}
            target="_blank"
            rel="noreferrer"
            className="font-medium underline-offset-2 hover:underline"
            title={row.original.sourceUrl}
          >
            {row.original.sourceDomain}
          </a>
          <span className="truncate text-xs text-muted-foreground" title={row.original.sourceUrl}>
            {truncate(row.original.sourceUrl.replace(/^https?:\/\/[^/]+/, ""), 40)}
          </span>
        </div>
      ),
    },
    {
      accessorKey: "anchor",
      header: s.anchor,
      cell: ({ row }) => (
        <span title={row.original.anchor}>
          {row.original.anchor ? (
            truncate(row.original.anchor, 28)
          ) : (
            <span className="text-muted-foreground">–</span>
          )}
        </span>
      ),
    },
    {
      accessorKey: "targetUrl",
      header: s.target,
      cell: ({ row }) => (
        <span className="text-xs text-muted-foreground" title={row.original.targetUrl}>
          {truncate(row.original.targetUrl.replace(/^https?:\/\/[^/]+/, "") || "/", 24)}
        </span>
      ),
    },
    {
      accessorKey: "domainRank",
      header: s.dr,
      meta: right,
      cell: ({ row }) =>
        row.original.domainRank == null ? (
          "–"
        ) : (
          <span className="tabular font-medium">{formatNumber(row.original.domainRank)}</span>
        ),
    },
    {
      accessorKey: "dofollow",
      header: s.follow,
      cell: ({ row }) => (
        <Badge
          variant={row.original.dofollow ? "secondary" : "outline"}
          className="px-1.5 py-0 text-xs font-normal"
        >
          {row.original.dofollow ? s.dofollow : s.nofollow}
        </Badge>
      ),
    },
    {
      accessorKey: "spamScore",
      header: s.spamScore,
      meta: right,
      cell: ({ row }) =>
        row.original.spamScore == null ? (
          "–"
        ) : isToxic(row.original.spamScore) ? (
          <StatusBadge kind="critical">{formatNumber(row.original.spamScore)}</StatusBadge>
        ) : (
          <span className="tabular text-muted-foreground">{formatNumber(row.original.spamScore)}</span>
        ),
    },
    {
      accessorKey: "firstSeen",
      header: s.firstSeen,
      cell: ({ row }) => <span className="text-xs">{formatDate(row.original.firstSeen)}</span>,
    },
    {
      accessorKey: "lastSeen",
      header: s.lastSeen,
      cell: ({ row }) => <span className="text-xs">{formatDate(row.original.lastSeen)}</span>,
    },
    {
      accessorKey: "isLost",
      header: t.common.status,
      cell: ({ row }) =>
        row.original.isLost ? (
          <StatusBadge kind="neutral">{s.tabLost}</StatusBadge>
        ) : (
          <StatusBadge kind="good">{t.common.active}</StatusBadge>
        ),
    },
  ];

  return (
    <Card className="gap-0 py-0">
      <CardContent className="space-y-4 p-6">
        <div className="flex flex-col gap-2">
          <div className="max-w-full overflow-x-auto scrollbar-thin">
            <Tabs value={view} onValueChange={(v) => setView(v as BacklinkView)}>
              <TabsList>
                {VIEWS.map((v) => (
                  <TabsTrigger key={v} value={v}>
                    {LABELS[v]}{" "}
                    <span className="tabular text-xs text-muted-foreground">{formatNumber(counts[v])}</span>
                  </TabsTrigger>
                ))}
              </TabsList>
            </Tabs>
          </div>
          {DESCS[view] ? <p className="text-sm text-muted-foreground">{DESCS[view]}</p> : null}
        </div>
        <DataTable
          data={filtered}
          columns={columns}
          searchKey="sourceDomain"
          searchPlaceholder={s.searchBacklink}
          exportName={`backlink-${view}`}
          initialSorting={[{ id: "domainRank", desc: true }]}
          pageSize={20}
          emptyMessage={t.common.noResults}
          getRowId={(r) => r.id}
        />
      </CardContent>
    </Card>
  );
}
