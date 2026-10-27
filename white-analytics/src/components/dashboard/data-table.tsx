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
import { DeltaText } from "@/components/dashboard/delta-badge";
import { Toolbar, ToolbarActions } from "@/components/dashboard/layout";
import { computeDelta, type Delta } from "@/lib/metrics";

export type DataTableProps<TData, TValue> = {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  /** column id to apply the search box to (client-side contains filter) */
  searchKey?: string;
  /** prefill for the search box, e.g. from a `?q=` URL param */
  initialSearch?: string;
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
  /** Show regular tables as labeled stacked rows below the sm breakpoint. */
  mobileCards?: boolean;
  mobileCardBreakpoint?: "sm" | "md";
};

export function DataTable<TData, TValue>({
  columns,
  data,
  searchKey,
  initialSearch,
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
  mobileCards,
  mobileCardBreakpoint = "sm",
}: DataTableProps<TData, TValue>) {
  const [sorting, setSorting] = React.useState<SortingState>(initialSorting);
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>(
    searchKey && initialSearch ? [{ id: searchKey, value: initialSearch }] : [],
  );
  const shouldPaginate = paginate ?? data.length > pageSize;
  const useMobileCards = mobileCards ?? !bare;

  // TanStack Table intentionally returns functions that the React Compiler cannot memoize.
  // eslint-disable-next-line react-hooks/incompatible-library
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
    const header = cols.map((c) =>
      csvEscape(String(typeof c.columnDef.header === "string" ? c.columnDef.header : c.id)),
    );
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
        <Toolbar>
          {searchKey ? (
            <div className="relative max-sm:w-full">
              <Search className="pointer-events-none absolute top-1/2 left-3.5 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder={searchPlaceholder ?? t.common.search}
                value={(table.getColumn(searchKey)?.getFilterValue() as string) ?? ""}
                onChange={(e) => table.getColumn(searchKey)?.setFilterValue(e.target.value)}
                className="w-full pl-9 sm:w-64"
              />
            </div>
          ) : null}
          {toolbar}
          {exportName ? (
            <ToolbarActions>
              <Button variant="outline" onClick={exportCsv}>
                <Download /> {t.common.exportCsv}
              </Button>
            </ToolbarActions>
          ) : null}
        </Toolbar>
      ) : null}
      {/* Standalone: the table is its own card and the first column sits at the card padding.
          bare: it bleeds to the edges of the card it lives in, so its first column lines up
          with that card's header text. Phones with mobile cards drop the frame. */}
      <div
        data-slot="data-table-frame"
        className={cn(
          bare
            ? "card-bleed"
            : useMobileCards
              ? mobileCardBreakpoint === "md"
                ? "md:overflow-hidden md:rounded-[1.5rem] md:bg-card md:shadow-(--card-shadow)"
                : "sm:overflow-hidden sm:rounded-[1.5rem] sm:bg-card sm:shadow-(--card-shadow)"
              : "overflow-hidden rounded-[1.5rem] bg-card shadow-(--card-shadow)",
        )}
      >
        <div className={cn("scrollbar-thin", useMobileCards ? (mobileCardBreakpoint === "md" ? "md:overflow-x-auto" : "sm:overflow-x-auto") : "overflow-x-auto")}>
          <Table
            className={cn(
              dense && "[&_td]:py-1.5 [&_th]:h-8",
              useMobileCards &&
                (mobileCardBreakpoint === "md"
                  ? "max-md:block max-md:[&_thead]:hidden max-md:[&_tbody]:block max-md:[&_tr]:mb-2 max-md:[&_tr]:block max-md:[&_tr]:rounded-2xl max-md:[&_tr]:border max-md:[&_tr]:border-border/70 max-md:[&_tr]:bg-card max-md:[&_td]:flex max-md:[&_td]:px-[var(--card-spacing,var(--card-pad))] max-md:[&_td]:items-start max-md:[&_td]:justify-between max-md:[&_td]:gap-4 max-md:[&_td]:border-b max-md:[&_td]:text-right max-md:[&_td]:before:text-left max-md:[&_td]:before:text-xs max-md:[&_td]:before:font-medium max-md:[&_td]:before:text-muted-foreground max-md:[&_td]:before:content-[attr(data-label)] max-md:[&_td:last-child]:border-b-0 max-md:[&_td:empty]:hidden"
                  : "max-sm:block max-sm:[&_thead]:hidden max-sm:[&_tbody]:block max-sm:[&_tr]:mb-2 max-sm:[&_tr]:block max-sm:[&_tr]:rounded-2xl max-sm:[&_tr]:border max-sm:[&_tr]:border-border/70 max-sm:[&_tr]:bg-card max-sm:[&_td]:flex max-sm:[&_td]:px-[var(--card-spacing,var(--card-pad))] max-sm:[&_td]:items-start max-sm:[&_td]:justify-between max-sm:[&_td]:gap-4 max-sm:[&_td]:border-b max-sm:[&_td]:text-right max-sm:[&_td]:before:text-left max-sm:[&_td]:before:text-xs max-sm:[&_td]:before:font-medium max-sm:[&_td]:before:text-muted-foreground max-sm:[&_td]:before:content-[attr(data-label)] max-sm:[&_td:last-child]:border-b-0 max-sm:[&_td:empty]:hidden"),
            )}
          >
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
                        className={cn(
                          "whitespace-nowrap text-xs font-medium text-muted-foreground",
                          align === "right" && "text-right",
                        )}
                      >
                        {header.isPlaceholder ? null : canSort ? (
                          <button
                            type="button"
                            onClick={header.column.getToggleSortingHandler()}
                            className={cn(
                              "inline-flex items-center gap-1 hover:text-foreground",
                              align === "right" && "flex-row-reverse",
                            )}
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
                      const header = cell.column.columnDef.header;
                      const dataLabel = typeof header === "string" ? header : cell.column.id;
                      return (
                        <TableCell
                          key={cell.id}
                          data-label={dataLabel}
                          className={cn(
                            "text-sm",
                            meta?.align === "right" && "text-right tabular",
                            meta?.mono && "font-mono text-xs",
                            meta?.className,
                          )}
                        >
                          {flexRender(cell.column.columnDef.cell, cell.getContext())}
                        </TableCell>
                      );
                    })}
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell
                    colSpan={columns.length}
                    className="h-40 text-center text-sm text-muted-foreground"
                  >
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
            {t.common.showing} {pageIndex * ps + 1}–{Math.min((pageIndex + 1) * ps, total)} {t.common.of}{" "}
            {total}
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
            <Button
              variant="outline"
              size="icon-sm"
              onClick={() => table.previousPage()}
              disabled={!table.getCanPreviousPage()}
              aria-label={t.common.prev}
            >
              <ChevronLeft className="size-4" />
            </Button>
            <span className="tabular">
              {pageIndex + 1} / {table.getPageCount()}
            </span>
            <Button
              variant="outline"
              size="icon-sm"
              onClick={() => table.nextPage()}
              disabled={!table.getCanNextPage()}
              aria-label={t.common.next}
            >
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

/**
 * "vs" column: colored change between two numbers of a row, sortable by the
 * percentage (or absolute) change. Use `lowerIsBetter` for costs, late rates,
 * positions and the like.
 */
export function deltaColumn<TData>({
  id,
  header,
  current,
  previous,
  lowerIsBetter,
  mode = "pct",
  format,
}: {
  id: string;
  header: string;
  current: (row: TData) => number | null | undefined;
  previous: (row: TData) => number | null | undefined;
  lowerIsBetter?: boolean;
  mode?: "pct" | "abs";
  format?: (delta: Delta) => string;
}): ColumnDef<TData, number | null> {
  const deltaOf = (row: TData) => {
    const cur = current(row);
    const prev = previous(row);
    return cur == null || prev == null ? null : computeDelta(cur, prev);
  };
  return {
    id,
    header,
    accessorFn: (row) => {
      const d = deltaOf(row);
      return d == null ? null : mode === "pct" ? d.pct : d.abs;
    },
    sortUndefined: "last",
    cell: ({ row }) => <DeltaText delta={deltaOf(row.original)} lowerIsBetter={lowerIsBetter} mode={mode} format={format} />,
    meta: { align: "right" } satisfies ColMeta,
  };
}
