"use client";

import type { ColumnDef, Row } from "@tanstack/react-table";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { DataTable } from "@/components/dashboard/data-table";
import { useUrlFilters } from "./url-state";
import { ss } from "./strings";

export type ServerPage = { page: number; pageSize: number; total: number };

/**
 * ERP list table: the dashboard DataTable plus a filter toolbar slot and,
 * when `pagination` is given, server-side paging through `page`/`pageSize`
 * URL params. Without it the table pages on the client.
 */
export function ErpDataTable<TData>({
  columns,
  data,
  toolbar,
  pagination,
  emptyMessage,
  exportName,
  onRowClick,
  getRowId,
  dense,
}: {
  columns: ColumnDef<TData, unknown>[];
  data: TData[];
  toolbar?: React.ReactNode;
  pagination?: ServerPage;
  emptyMessage?: string;
  exportName?: string;
  onRowClick?: (row: Row<TData>) => void;
  getRowId?: (row: TData, index: number) => string;
  dense?: boolean;
}) {
  return (
    <div className="flex flex-col gap-3">
      <DataTable
        columns={columns}
        data={data}
        toolbar={toolbar}
        paginate={pagination ? false : undefined}
        emptyMessage={emptyMessage ?? ss.noData}
        exportName={exportName}
        onRowClick={onRowClick}
        getRowId={getRowId}
        dense={dense}
      />
      {pagination && pagination.total > 0 ? <UrlPagination {...pagination} /> : null}
    </div>
  );
}

/** Pager bound to `page` and `pageSize` URL params. */
export function UrlPagination({ page, pageSize, total }: ServerPage) {
  const { set, isPending } = useUrlFilters(["pageSize"]);
  const pages = Math.max(1, Math.ceil(total / pageSize));
  const from = (page - 1) * pageSize + 1;
  const to = Math.min(page * pageSize, total);
  return (
    <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
      <div className="tabular">{ss.showing(from, to, total)}</div>
      <div className="flex items-center gap-2">
        <Select value={String(pageSize)} onValueChange={(v) => set({ pageSize: v, page: null })}>
          <SelectTrigger size="sm" className="w-[72px]" aria-label={ss.rowsPerPage}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {[10, 20, 50, 100].map((n) => (
              <SelectItem key={n} value={String(n)}>
                {n}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button
          variant="outline"
          size="icon-sm"
          disabled={page <= 1 || isPending}
          onClick={() => set({ page: page - 1 > 1 ? String(page - 1) : null })}
          aria-label={ss.prev}
        >
          <ChevronLeft className="size-4" />
        </Button>
        <span className="tabular">{ss.pageOf(page, pages)}</span>
        <Button
          variant="outline"
          size="icon-sm"
          disabled={page >= pages || isPending}
          onClick={() => set({ page: String(page + 1) })}
          aria-label={ss.next}
        >
          <ChevronRight className="size-4" />
        </Button>
      </div>
    </div>
  );
}
