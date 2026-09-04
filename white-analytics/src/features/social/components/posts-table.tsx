"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Badge } from "@/components/ui/badge";
import { DataTable } from "@/components/dashboard/data-table";
import { formatDate, formatNumber, formatPercent, truncate } from "@/lib/format";
import type { SocialPostRow } from "@/features/social/queries";
import { t } from "@/i18n/id";

function numCol(key: keyof SocialPostRow, header: string): ColumnDef<SocialPostRow, unknown> {
  return {
    accessorKey: key,
    header,
    meta: { align: "right" },
    cell: ({ getValue }) => formatNumber(Number(getValue())),
  };
}

/** Tabel "Postingan terbaru" — sortable, cari caption, ekspor CSV. */
export function PostsTable({ rows, exportName }: { rows: SocialPostRow[]; exportName: string }) {
  const columns = React.useMemo<ColumnDef<SocialPostRow, unknown>[]>(
    () => [
      {
        accessorKey: "caption",
        header: t.social.caption,
        enableSorting: false,
        size: 340,
        cell: ({ row }) => {
          const p = row.original;
          return (
            <span className="flex items-center gap-2.5">
              <span className="relative size-9 shrink-0 overflow-hidden rounded-md bg-muted">
                {p.thumbnailUrl ? (
                  // eslint-disable-next-line @next/next/no-img-element -- URL eksternal demo (picsum)
                  <img
                    src={p.thumbnailUrl}
                    alt=""
                    loading="lazy"
                    className="size-full object-cover"
                    onError={(e) => {
                      e.currentTarget.style.display = "none";
                    }}
                  />
                ) : null}
              </span>
              <span className="max-w-[260px] truncate" title={p.caption}>
                {truncate(p.caption, 70)}
              </span>
            </span>
          );
        },
      },
      {
        accessorKey: "type",
        header: t.social.type,
        cell: ({ getValue }) => <Badge variant="outline">{String(getValue())}</Badge>,
      },
      {
        accessorKey: "publishedAt",
        header: t.social.published,
        cell: ({ getValue }) => <span className="whitespace-nowrap">{formatDate(String(getValue()))}</span>,
      },
      numCol("likes", t.social.likes),
      numCol("comments", t.social.comments),
      numCol("shares", t.social.shares),
      numCol("saves", t.social.saves),
      numCol("views", t.social.views),
      numCol("reach", t.social.reach),
      {
        accessorKey: "er",
        header: t.social.er,
        meta: { align: "right" },
        cell: ({ getValue }) => formatPercent(Number(getValue())),
      },
    ],
    [],
  );

  return (
    <DataTable
      columns={columns}
      data={rows}
      searchKey="caption"
      exportName={exportName}
      initialSorting={[{ id: "publishedAt", desc: true }]}
      dense
      getRowId={(r) => r.id}
      emptyMessage={t.common.noData}
    />
  );
}
