"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import type { ColumnDef } from "@tanstack/react-table";
import { FileText } from "lucide-react";
import { ErpDataTable, type ServerPage } from "@/features/erp/shared/data-table";
import { DateRangeFilter } from "@/features/erp/shared/period-picker";
import type { SearchOption } from "@/features/erp/shared/searchable-select";
import { ss } from "@/features/erp/shared/strings";
import { ResetFilters, SearchFilter, SearchSelectFilter, SelectFilter } from "@/features/erp/shared/url-filters";
import { formatDate } from "@/lib/format";
import { CANDIDATE_STATUS, SOURCE, STAGE_ORDER, rs } from "../strings";
import type { Candidate } from "../types";
import { CandidateBadge } from "./ui";

const c = rs.candidates;

export function CandidatesTable({
  candidates,
  pagination,
  jobs,
}: {
  candidates: Candidate[];
  pagination: ServerPage;
  jobs: SearchOption[];
}) {
  const router = useRouter();
  const columns = React.useMemo<ColumnDef<Candidate, unknown>[]>(
    () => [
      {
        accessorKey: "fullName",
        header: c.name,
        cell: ({ row }) => (
          <div className="min-w-44">
            <Link
              href={`/erp/hr/recruitment/candidates/${row.original.id}`}
              className="flex items-center gap-1.5 font-medium hover:underline"
              onClick={(e) => e.stopPropagation()}
            >
              {row.original.fullName}
              {row.original.hasCv ? <FileText className="size-3.5 text-muted-foreground" aria-label={rs.profile.cv} /> : null}
            </Link>
            <div className="text-xs text-muted-foreground">{row.original.email || row.original.phone}</div>
          </div>
        ),
      },
      { accessorKey: "jobOpeningTitle", header: c.job, cell: ({ row }) => row.original.jobOpeningTitle || rs.none },
      { accessorKey: "status", header: c.stage, cell: ({ row }) => <CandidateBadge status={row.original.status} /> },
      { accessorKey: "source", header: c.source, cell: ({ row }) => SOURCE[row.original.source] ?? row.original.source },
      { accessorKey: "domicile", header: c.domicile, cell: ({ row }) => row.original.domicile || rs.none },
      {
        accessorKey: "createdAt",
        header: c.applied,
        cell: ({ row }) => <span className="whitespace-nowrap">{formatDate(row.original.createdAt)}</span>,
      },
    ],
    [],
  );

  return (
    <ErpDataTable
      columns={columns}
      data={candidates}
      pagination={pagination}
      exportName="kandidat"
      emptyMessage={c.empty}
      getRowId={(r) => r.id}
      onRowClick={(row) => router.push(`/erp/hr/recruitment/candidates/${row.original.id}`)}
      toolbar={
        <div className="flex flex-wrap items-center gap-2">
          <SearchFilter placeholder={c.search} />
          <SelectFilter
            param="status"
            label={c.statusFilter}
            allLabel={ss.allStatus}
            options={STAGE_ORDER.map((s) => ({ value: s, label: CANDIDATE_STATUS[s] }))}
          />
          <SearchSelectFilter param="job" options={jobs} placeholder={c.allJobs} className="w-56" />
          <DateRangeFilter />
          <SelectFilter
            param="sort"
            label={c.sort}
            allLabel={c.sortCreated}
            options={[{ value: "updated_at", label: c.sortUpdated }]}
          />
          <ResetFilters params={["q", "status", "job", "from", "to", "sort"]} />
        </div>
      }
    />
  );
}
