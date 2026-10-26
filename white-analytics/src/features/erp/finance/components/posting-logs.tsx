"use client";

import Link from "next/link";
import type { ColumnDef } from "@tanstack/react-table";
import { RotateCcw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { SelectFilter } from "@/features/erp/shared/url-filters";
import { formatDateTime } from "@/lib/format";
import { retryPostingAction } from "../actions";
import type { PostingLog } from "../types";
import { fs } from "../strings";

const l = fs.journals.logs;

export function PostingLogs({ logs, canApprove }: { logs: PostingLog[]; canApprove: boolean }) {
  const { pending, run } = useErpAction();
  const columns: ColumnDef<PostingLog, unknown>[] = [
    { accessorKey: "updatedAt", header: l.updated, cell: ({ row }) => formatDateTime(row.original.updatedAt) },
    { accessorKey: "eventCode", header: l.event, meta: { mono: true } },
    {
      id: "document",
      accessorFn: (r) => r.documentLabel || r.documentId,
      header: l.document,
      cell: ({ row }) => (
        <span>
          <span className="block">{row.original.documentLabel || row.original.documentId}</span>
          <span className="text-xs text-muted-foreground">{row.original.documentType}</span>
        </span>
      ),
    },
    {
      accessorKey: "status",
      header: fs.status,
      cell: ({ row }) => (
        <ErpStatusBadge
          status={row.original.status.toUpperCase()}
          labels={{ POSTED: l.status.posted, DRAFT: l.status.draft, SKIPPED: l.status.skipped }}
          kinds={{ SKIPPED: "warning", DRAFT: "warning" }}
        />
      ),
    },
    {
      accessorKey: "reason",
      header: l.reason,
      cell: ({ row }) => <span className="line-clamp-2 max-w-sm text-xs text-muted-foreground">{row.original.reason || "–"}</span>,
    },
    { accessorKey: "attempts", header: l.attempts, meta: { align: "right" } },
    {
      id: "actions",
      header: fs.actions,
      enableSorting: false,
      meta: { align: "right" },
      cell: ({ row }) => (
        <span className="flex justify-end gap-1">
          {row.original.entryId ? (
            <Button asChild size="xs" variant="ghost">
              <Link href={`/erp/finance/journals/${row.original.entryId}`}>{l.openEntry}</Link>
            </Button>
          ) : null}
          {canApprove && row.original.status !== "posted" ? (
            <Button
              size="xs"
              variant="outline"
              disabled={pending}
              onClick={() => run(() => retryPostingAction(row.original.id), { success: l.retried })}
            >
              <RotateCcw className="size-3.5" /> {l.retry}
            </Button>
          ) : null}
        </span>
      ),
    },
  ];
  return (
    <ErpDataTable
      columns={columns}
      data={logs}
      emptyMessage={l.empty}
      toolbar={
        <SelectFilter
          param="logStatus"
          label={fs.status}
          allLabel={fs.status}
          options={Object.entries(l.status).map(([value, label]) => ({ value, label }))}
        />
      }
    />
  );
}
