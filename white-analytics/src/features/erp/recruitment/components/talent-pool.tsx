"use client";

import * as React from "react";
import Link from "next/link";
import type { ColumnDef } from "@tanstack/react-table";
import { Archive, RotateCcw, UserCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import { UrlSwitch } from "@/features/erp/hr/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable, type ServerPage } from "@/features/erp/shared/data-table";
import { SearchFilter } from "@/features/erp/shared/url-filters";
import { formatDate } from "@/lib/format";
import { moveStageAction } from "../actions";
import { SOURCE, rs } from "../strings";
import type { Candidate } from "../types";
import { CandidateBadge } from "./ui";

const tp = rs.talentPool;

export function TalentPoolTable({ candidates, pagination, canWrite }: { candidates: Candidate[]; pagination: ServerPage; canWrite: boolean }) {
  const columns = React.useMemo<ColumnDef<Candidate, unknown>[]>(
    () => [
      {
        accessorKey: "fullName",
        header: rs.candidates.name,
        cell: ({ row }) => (
          <div className="min-w-44">
            <Link href={`/erp/hr/recruitment/candidates/${row.original.id}`} className="font-medium hover:underline">
              {row.original.fullName}
            </Link>
            <div className="text-xs text-muted-foreground">{row.original.lastExperience || row.original.email}</div>
          </div>
        ),
      },
      { accessorKey: "jobOpeningTitle", header: rs.candidates.job, cell: ({ row }) => row.original.jobOpeningTitle || rs.none },
      { accessorKey: "status", header: rs.candidates.stage, cell: ({ row }) => <CandidateBadge status={row.original.status} /> },
      { accessorKey: "source", header: rs.candidates.source, cell: ({ row }) => SOURCE[row.original.source] ?? row.original.source },
      { accessorKey: "updatedAt", header: rs.candidates.sortUpdated, cell: ({ row }) => formatDate(row.original.updatedAt) },
      ...(canWrite
        ? [{ id: "actions", header: "", cell: ({ row }: { row: { original: Candidate } }) => <PoolActions candidate={row.original} /> }]
        : []),
    ],
    [canWrite],
  );
  return (
    <ErpDataTable
      columns={columns}
      data={candidates}
      pagination={pagination}
      emptyMessage={tp.empty}
      exportName="talent-pool"
      toolbar={
        <div className="flex flex-wrap items-center gap-2">
          <SearchFilter placeholder={rs.candidates.search} />
          <UrlSwitch param="archived" label={tp.includeArchived} />
        </div>
      }
    />
  );
}

function PoolActions({ candidate }: { candidate: Candidate }) {
  const { run, pending } = useErpAction();
  const move = (status: string) => run(() => moveStageAction(candidate.id, status), { success: rs.candidates.moved });
  return (
    <div className="flex justify-end gap-1.5">
      <Button size="xs" variant="outline" disabled={pending} onClick={() => void move("screening")}>
        <UserCheck className="size-3.5" /> {tp.reactivate}
      </Button>
      {candidate.status === "archived" ? (
        <Button size="xs" variant="ghost" disabled={pending} onClick={() => void move("talent_pool")}>
          <RotateCcw className="size-3.5" /> {tp.restore}
        </Button>
      ) : (
        <Button size="xs" variant="ghost" disabled={pending} onClick={() => void move("archived")}>
          <Archive className="size-3.5" /> {tp.archive}
        </Button>
      )}
    </div>
  );
}
