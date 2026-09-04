"use client";

import * as React from "react";
import { useTransition } from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Trash2 } from "lucide-react";
import { toast } from "@/lib/toast";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/dashboard/data-table";
import { DeltaBadge } from "@/components/dashboard/delta-badge";
import { formatDeltaNumber, formatDeltaPercent, formatNumber, formatPercent, initials } from "@/lib/format";
import { removeCompetitor } from "@/features/social/actions";
import type { CompetitorRow } from "@/features/social/queries";
import { s } from "@/features/social/strings";
import { t } from "@/i18n/id";

function RemoveCompetitorButton({ accountId, username }: { accountId: string; username: string }) {
  const [pending, startTransition] = useTransition();
  const run = () =>
    startTransition(async () => {
      const res = await removeCompetitor(accountId);
      if (res.ok) toast.success(s.competitorRemoved, { description: `@${username}` });
      else toast.error(t.common.error, { description: res.error });
    });
  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>
        <Button variant="ghost" size="icon-sm" disabled={pending} aria-label={`${t.social.removeCompetitor} @${username}`}>
          <Trash2 className="size-4" />
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent size="sm">
        <AlertDialogHeader>
          <AlertDialogTitle>{s.removeConfirmTitle}</AlertDialogTitle>
          <AlertDialogDescription>
            @{username} — {s.removeConfirmDesc}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{t.common.cancel}</AlertDialogCancel>
          <AlertDialogAction onClick={run}>{t.common.delete}</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

/** Tabel perbandingan akun sendiri vs kompetitor. Baris "Akun Anda" disorot. */
export function CompetitorTable({
  rows,
  canManage,
  compare,
  exportName,
}: {
  rows: CompetitorRow[];
  canManage: boolean;
  compare: boolean;
  exportName: string;
}) {
  const columns = React.useMemo<ColumnDef<CompetitorRow, unknown>[]>(() => {
    const base: ColumnDef<CompetitorRow, unknown>[] = [
      {
        accessorKey: "username",
        header: s.account,
        cell: ({ row }) => {
          const r = row.original;
          return (
            <span className="flex min-w-0 items-center gap-2.5">
              <Avatar size="sm">
                {r.avatarUrl ? <AvatarImage src={r.avatarUrl} alt="" /> : null}
                <AvatarFallback>{initials(r.displayName || r.username)}</AvatarFallback>
              </Avatar>
              <span className="min-w-0">
                <span className="flex items-center gap-1.5">
                  <span className="truncate font-medium">@{r.username}</span>
                  {r.isOwn ? <Badge variant="secondary">{t.social.yourAccount}</Badge> : null}
                </span>
                <span className="block truncate text-xs text-muted-foreground">{r.displayName}</span>
              </span>
            </span>
          );
        },
      },
      {
        accessorKey: "followers",
        header: t.social.followers,
        meta: { align: "right" },
        cell: ({ getValue }) => formatNumber(Number(getValue())),
      },
      {
        accessorKey: "growthAbs",
        header: t.social.growth30d,
        meta: { align: "right" },
        cell: ({ row }) => {
          const r = row.original;
          return (
            <span className="inline-flex flex-col items-end gap-0.5">
              <span className="tabular">
                {formatDeltaNumber(r.growthAbs)}
                <span className="ml-1 text-xs text-muted-foreground">
                  {r.growthPct == null ? "" : `(${formatDeltaPercent(r.growthPct)})`}
                </span>
              </span>
              {compare ? <DeltaBadge delta={r.growthDelta} mode="abs" /> : null}
            </span>
          );
        },
      },
      {
        accessorKey: "avgEr",
        header: s.avgEr,
        meta: { align: "right" },
        cell: ({ getValue }) => formatPercent(Number(getValue())),
      },
      {
        accessorKey: "postsPerWeek",
        header: t.social.postsPerWeek,
        meta: { align: "right" },
        cell: ({ getValue }) => formatNumber(Number(getValue()), 1),
      },
      {
        accessorKey: "mediaCount",
        header: t.social.mediaCount,
        meta: { align: "right" },
        cell: ({ getValue }) => formatNumber(Number(getValue())),
      },
    ];
    if (canManage) {
      base.push({
        id: "actions",
        header: t.common.actions,
        enableSorting: false,
        meta: { align: "right" },
        cell: ({ row }) =>
          row.original.isOwn ? null : (
            <RemoveCompetitorButton accountId={row.original.accountId} username={row.original.username} />
          ),
      });
    }
    return base;
  }, [canManage, compare]);

  return (
    <DataTable
      columns={columns}
      data={rows}
      exportName={exportName}
      getRowId={(r) => r.accountId}
      rowClassName={(row) => (row.original.isOwn ? "bg-accent/50 hover:bg-accent/60" : undefined)}
      emptyMessage={s.noCompetitors}
    />
  );
}
