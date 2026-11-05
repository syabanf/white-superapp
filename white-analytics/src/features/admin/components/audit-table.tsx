"use client";

import type { ColumnDef } from "@tanstack/react-table";
import { ErpDataTable, type ServerPage } from "@/features/erp/shared/data-table";
import { DateRangeFilter } from "@/features/erp/shared/period-picker";
import { ResetFilters, SearchFilter, SelectFilter } from "@/features/erp/shared/url-filters";

export type AuditLog = { id: string; at: string; userId: string; userName: string; role: string; method: string; path: string; status: number; ip: string };

const columns: ColumnDef<AuditLog, unknown>[] = [
  {
    accessorKey: "at",
    header: "Waktu",
    cell: ({ row }) => new Date(row.original.at).toLocaleString("id-ID", { timeZone: "Asia/Jakarta", dateStyle: "medium", timeStyle: "medium" }),
  },
  { accessorKey: "userName", header: "Pengguna", cell: ({ row }) => row.original.userName || "Belum masuk" },
  { accessorKey: "method", header: "Aksi", cell: ({ row }) => <span className="font-mono text-xs">{row.original.method}</span> },
  { accessorKey: "path", header: "Alamat", cell: ({ row }) => <span className="font-mono text-xs break-all">{row.original.path}</span> },
  { accessorKey: "status", header: "Hasil", cell: ({ row }) => (row.original.status >= 400 ? <span className="text-negative">Gagal {row.original.status}</span> : "Berhasil") },
  { accessorKey: "ip", header: "IP", cell: ({ row }) => <span className="font-mono text-xs">{row.original.ip}</span> },
];

export function AuditTable({ logs, page }: { logs: AuditLog[]; page: ServerPage }) {
  return (
    <ErpDataTable
      columns={columns}
      data={logs}
      pagination={page}
      getRowId={(l) => l.id}
      emptyMessage="Belum ada catatan."
      toolbar={
        <>
          <SearchFilter placeholder="Cari alamat atau nama pengguna" />
          <SelectFilter param="failed" label="Hasil" allLabel="Semua hasil" options={[{ value: "1", label: "Hanya gagal" }]} />
          <DateRangeFilter />
          <ResetFilters params={["q", "failed", "from", "to"]} />
        </>
      }
    />
  );
}
