"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import type { ColumnDef } from "@tanstack/react-table";
import { ErpDataTable, type ServerPage } from "@/features/erp/shared/data-table";
import { Money } from "@/features/erp/shared/money-display";
import { DateRangeFilter } from "@/features/erp/shared/period-picker";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { ResetFilters, SearchFilter, SelectFilter } from "@/features/erp/shared/url-filters";
import { formatDate } from "@/lib/format";
import type { JournalEntry } from "../types";
import { fs } from "../strings";

const j = fs.journals;

export function JournalsTable({ entries, page }: { entries: JournalEntry[]; page: ServerPage }) {
  const router = useRouter();
  const columns: ColumnDef<JournalEntry, unknown>[] = [
    {
      accessorKey: "entryNo",
      header: j.entryNo,
      cell: ({ row }) => (
        <Link href={`/erp/finance/journals/${row.original.id}`} className="font-mono text-xs font-medium hover:underline">
          {row.original.entryNo || "–"}
        </Link>
      ),
    },
    { accessorKey: "entryDate", header: fs.date, cell: ({ row }) => formatDate(row.original.entryDate) },
    {
      accessorKey: "description",
      header: fs.description,
      cell: ({ row }) => (
        <span className="line-clamp-2 max-w-md">{row.original.description || fs.noDescription}</span>
      ),
    },
    {
      accessorKey: "entryType",
      header: fs.type,
      cell: ({ row }) => (
        <span className="text-xs text-muted-foreground">
          {fs.entryType[row.original.entryType] ?? row.original.entryType}
          {row.original.sourceModule ? ` · ${row.original.sourceModule}` : ""}
        </span>
      ),
    },
    { accessorKey: "fiscalPeriodName", header: j.period },
    {
      accessorKey: "totalDebit",
      header: fs.amount,
      meta: { align: "right" },
      cell: ({ row }) => <Money value={row.original.totalDebit} />,
    },
    { accessorKey: "status", header: fs.status, cell: ({ row }) => <ErpStatusBadge status={row.original.status} /> },
  ];
  return (
    <ErpDataTable
      columns={columns}
      data={entries}
      pagination={page}
      emptyMessage={j.empty}
      onRowClick={(r) => router.push(`/erp/finance/journals/${r.original.id}`)}
      toolbar={
        <>
          <SearchFilter placeholder={j.searchPh} />
          <SelectFilter
            param="status"
            label={fs.status}
            allLabel={fs.status}
            options={[
              { value: "DRAFT", label: "Draf" },
              { value: "POSTED", label: "Diposting" },
              { value: "VOID", label: "Dibatalkan" },
            ]}
          />
          <SelectFilter
            param="type"
            label={fs.type}
            allLabel={j.allTypes}
            options={Object.entries(fs.entryType).map(([value, label]) => ({ value, label }))}
          />
          <DateRangeFilter />
          <ResetFilters params={["q", "status", "type", "from", "to", "account"]} />
        </>
      }
    />
  );
}
