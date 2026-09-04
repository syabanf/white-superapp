"use client";

import * as React from "react";
import {
  type ColumnDef,
  type SortingState,
  type ColumnFiltersState,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
  type Row,
} from "@tanstack/react-table";
import { ArrowDown, ArrowUp, ArrowUpDown, ChevronLeft, ChevronRight, Download, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";
import { t } from "@/i18n/id";

export type DataTableProps<TData, TValue> = {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  /** column id to apply the search box to (client-side contains filter) */
  searchKey?: string;
  searchPlaceholder?: string;
  pageSize?: number;
  /** enable pagination controls (default true when data > pageSize) */
  paginate?: boolean;
  /** CSV export filename (without extension) — enables the export button */
  exportName?: string;
  emptyMessage?: string;
  onRowClick?: (row: Row<TData>) => void;
  rowClassName?: (row: Row<TData>) => string | undefined;
  toolbar?: React.ReactNode;
  initialSorting?: SortingState;
  className?: string;
  dense?: boolean;
  /** hide toolbar entirely (for embedded chart-card tables) */
  bare?: boolean;
  stickyHeader?: boolean;
  /** identifier for row keys */
  getRowId?: (row: TData, index: number) => string;
};

export function DataTable<TData, TValue>({
  columns,
  data,
  searchKey,
  searchPlaceholder,
  pageSize = 10,
  paginate,
  exportName,
  emptyMessage,
  onRowClick,
  rowClassName,
  toolbar,
  initialSorting = [],
  className,
  dense,
  bare,
  stickyHeader,
  getRowId,
}: DataTableProps<TData, TValue>) {
  const [sorting, setSorting] = React.useState<SortingState>(initialSorting);
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>([]);
  const shouldPaginate = paginate ?? data.length > pageSize;

  const table = useReactTable({
    data,
    columns,
    getRowId,
    state: { sorting, columnFilters },
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: shouldPaginate ? getPaginationRowModel() : undefined,
    initialState: { pagination: { pageSize } },
  });

  const exportCsv = () => {
    const rows = table.getFilteredRowModel().rows;
    const cols = table.getAllLeafColumns().filter((c) => c.getIsVisible());
    const header = cols.map((c) => csvEscape(String(typeof c.columnDef.header === "string" ? c.columnDef.header : c.id)));
    const lines = rows.map((r) =>
      cols
        .map((c) => {
          const v = r.getValue(c.id);
          return csvEscape(v == null ? "" : typeof v === "object" ? JSON.stringify(v) : String(v));
        })
        .join(","),
    );
    const csv = [header.join(","), ...lines].join("\n");
    const blob = new Blob(["﻿" + csv], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${exportName ?? "data"}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const showToolbar = !bare && (searchKey || exportName || toolbar);
  const pageRows = table.getRowModel().rows;
  const total = table.getFilteredRowModel().rows.length;
  const { pageIndex, pageSize: ps } = table.getState().pagination;

  return (
    <div className={cn("flex flex-col gap-3", className)}>
      {showToolbar ? (
        <div className="flex flex-wrap items-center gap-2">
          {searchKey ? (
            <div className="relative">
              <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder={searchPlaceholder ?? t.common.search}
                value={(table.getColumn(searchKey)?.getFilterValue() as string) ?? ""}
                onChange={(e) => table.getColumn(searchKey)?.setFilterValue(e.target.value)}
                className="h-8 w-56 pl-8 text-[13px]"
              />
            </div>
          ) : null}
          {toolbar}
          <div className="ml-auto flex items-center gap-2">
            {exportName ? (
              <Button variant="outline" size="sm" onClick={exportCsv}>
                <Download className="size-3.5" /> {t.common.exportCsv}
              </Button>
            ) : null}
          </div>
        </div>
      ) : null}
      <div className="overflow-hidden rounded-lg border">
        <div className="overflow-x-auto scrollbar-thin">
          <Table className={cn(dense && "[&_td]:py-1.5 [&_th]:h-8")}>
            <TableHeader className={cn(stickyHeader && "sticky top-0 z-10 bg-card")}>
              {table.getHeaderGroups().map((hg) => (
                <TableRow key={hg.id} className="hover:bg-transparent">
                  {hg.headers.map((header) => {
                    const canSort = header.column.getCanSort();
                    const sorted = header.column.getIsSorted();
                    const align = (header.column.columnDef.meta as ColMeta | undefined)?.align;
                    return (
                      <TableHead
                        key={header.id}
                        style={{ width: header.getSize() !== 150 ? header.getSize() : undefined }}
                        className={cn("whitespace-nowrap text-xs font-medium text-muted-foreground", align === "right" && "text-right")}
                      >
                        {header.isPlaceholder ? null : canSort ? (
                          <button
                            type="button"
                            onClick={header.column.getToggleSortingHandler()}
                            className={cn("inline-flex items-center gap-1 hover:text-foreground", align === "right" && "flex-row-reverse")}
                          >
                            {flexRender(header.column.columnDef.header, header.getContext())}
                            {sorted === "asc" ? (
                              <ArrowUp className="size-3" />
                            ) : sorted === "desc" ? (
                              <ArrowDown className="size-3" />
                            ) : (
                              <ArrowUpDown className="size-3 opacity-40" />
                            )}
                          </button>
                        ) : (
                          flexRender(header.column.columnDef.header, header.getContext())
                        )}
                      </TableHead>
                    );
                  })}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {pageRows.length ? (
                pageRows.map((row) => (
                  <TableRow
                    key={row.id}
                    data-state={row.getIsSelected() && "selected"}
                    onClick={onRowClick ? () => onRowClick(row) : undefined}
                    className={cn(onRowClick && "cursor-pointer hover:bg-accent/60", rowClassName?.(row))}
                  >
                    {row.getVisibleCells().map((cell) => {
                      const meta = cell.column.columnDef.meta as ColMeta | undefined;
                      return (
                        <TableCell
                          key={cell.id}
                          className={cn("text-sm", meta?.align === "right" && "text-right tabular", meta?.mono && "font-mono text-xs", meta?.className)}
                        >
                          {flexRender(cell.column.columnDef.cell, cell.getContext())}
                        </TableCell>
                      );
                    })}
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={columns.length} className="h-24 text-center text-sm text-muted-foreground">
                    {emptyMessage ?? t.common.noResults}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      </div>
      {shouldPaginate && total > 0 ? (
        <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
          <div>
            {t.common.showing} {pageIndex * ps + 1}–{Math.min((pageIndex + 1) * ps, total)} {t.common.of} {total}
          </div>
          <div className="flex items-center gap-2">
            <Select value={String(ps)} onValueChange={(v) => table.setPageSize(Number(v))}>
              <SelectTrigger size="sm" className="w-[72px]" aria-label={t.common.rowsPerPage}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {[10, 25, 50, 100].map((n) => (
                  <SelectItem key={n} value={String(n)}>
                    {n}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button variant="outline" size="icon-sm" onClick={() => table.previousPage()} disabled={!table.getCanPreviousPage()} aria-label={t.common.prev}>
              <ChevronLeft className="size-4" />
            </Button>
            <span className="tabular">
              {pageIndex + 1} / {table.getPageCount()}
            </span>
            <Button variant="outline" size="icon-sm" onClick={() => table.nextPage()} disabled={!table.getCanNextPage()} aria-label={t.common.next}>
              <ChevronRight className="size-4" />
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  );
}

export type ColMeta = { align?: "left" | "right"; mono?: boolean; className?: string };

function csvEscape(s: string): string {
  if (/[",\n]/.test(s)) return `"${s.replace(/"/g, '""')}"`;
  return s;
}
