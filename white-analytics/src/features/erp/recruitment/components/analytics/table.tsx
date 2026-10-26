"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { ArrowDown, ArrowUp, ArrowUpDown } from "lucide-react";
import { Table, TableBody, TableCell, TableFooter, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { cn } from "@/lib/utils";
import { ra } from "../../strings";

export type Column<T> = {
  key: string;
  header: string;
  align?: "right";
  cell: (row: T) => React.ReactNode;
  /** enables sorting on this column */
  sort?: (row: T) => number | string;
  /** totals row content */
  total?: React.ReactNode;
  className?: string;
};

/**
 * Detail table under the charts: sortable headers, optional totals row and
 * row links. Scrolls inside its card on narrow screens.
 */
export function AnalyticsTable<T>({
  rows,
  columns,
  rowKey,
  rowHref,
  initialSort,
  limit,
  emptyMessage = ra.empty,
}: {
  rows: T[];
  columns: Column<T>[];
  rowKey: (row: T) => string;
  rowHref?: (row: T) => string | undefined;
  initialSort?: { key: string; desc?: boolean };
  /** show only the first n rows after sorting */
  limit?: number;
  emptyMessage?: string;
}) {
  const router = useRouter();
  const [sort, setSort] = React.useState(initialSort ?? null);
  const sorted = React.useMemo(() => {
    const col = sort && columns.find((c) => c.key === sort.key);
    if (!col?.sort) return rows;
    const get = col.sort;
    return [...rows].sort((a, b) => {
      const x = get(a);
      const y = get(b);
      const cmp = typeof x === "number" && typeof y === "number" ? x - y : String(x).localeCompare(String(y), "id");
      return sort!.desc ? -cmp : cmp;
    });
  }, [rows, columns, sort]);
  const shown = limit ? sorted.slice(0, limit) : sorted;
  const hasTotals = columns.some((c) => c.total != null);

  return (
    <div className="overflow-x-auto rounded-lg border scrollbar-thin">
      <Table className="min-w-[560px]">
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            {columns.map((c) => {
              const active = sort?.key === c.key;
              return (
                <TableHead
                  key={c.key}
                  className={cn("whitespace-nowrap text-xs font-medium text-muted-foreground", c.align === "right" && "text-right")}
                  aria-sort={active ? (sort!.desc ? "descending" : "ascending") : undefined}
                >
                  {c.sort ? (
                    <button
                      type="button"
                      aria-label={ra.table.sortBy(c.header)}
                      onClick={() => setSort(active ? { key: c.key, desc: !sort!.desc } : { key: c.key, desc: true })}
                      className={cn("inline-flex items-center gap-1 hover:text-foreground", c.align === "right" && "flex-row-reverse")}
                    >
                      {c.header}
                      {active ? (
                        sort!.desc ? (
                          <ArrowDown className="size-3" />
                        ) : (
                          <ArrowUp className="size-3" />
                        )
                      ) : (
                        <ArrowUpDown className="size-3 opacity-40" />
                      )}
                    </button>
                  ) : (
                    c.header
                  )}
                </TableHead>
              );
            })}
          </TableRow>
        </TableHeader>
        <TableBody>
          {shown.length === 0 ? (
            <TableRow>
              <TableCell colSpan={columns.length} className="h-20 text-center text-sm text-muted-foreground">
                {emptyMessage}
              </TableCell>
            </TableRow>
          ) : (
            shown.map((row) => {
              const link = rowHref?.(row);
              return (
                <TableRow
                  key={rowKey(row)}
                  onClick={link ? () => router.push(link) : undefined}
                  className={cn(link && "cursor-pointer hover:bg-accent/60")}
                >
                  {columns.map((c) => (
                    <TableCell key={c.key} className={cn("text-sm", c.align === "right" && "text-right tabular", c.className)}>
                      {c.cell(row)}
                    </TableCell>
                  ))}
                </TableRow>
              );
            })
          )}
        </TableBody>
        {hasTotals && shown.length > 0 ? (
          <TableFooter>
            <TableRow className="hover:bg-transparent">
              {columns.map((c, i) => (
                <TableCell key={c.key} className={cn("text-sm font-semibold", c.align === "right" && "text-right tabular")}>
                  {i === 0 ? ra.table.total : c.total}
                </TableCell>
              ))}
            </TableRow>
          </TableFooter>
        ) : null}
      </Table>
    </div>
  );
}
