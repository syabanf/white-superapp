"use client";

import * as React from "react";
import Link from "next/link";
import { ArrowDown, ArrowUp, ArrowUpDown } from "lucide-react";
import { Table, TableBody, TableCell, TableFooter, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { cn } from "@/lib/utils";
import { thd } from "@/features/erp/hr/dashboard/strings";

export type SortColumn<T> = {
  key: string;
  label: string;
  /** value used for sorting; strings sort alphabetically */
  value: (row: T) => number | string;
  cell?: (row: T) => React.ReactNode;
  numeric?: boolean;
  /** totals row content; omit to leave the cell empty */
  total?: React.ReactNode;
  className?: string;
};

/**
 * Small sortable table for dashboard detail blocks: click a header to sort,
 * optional totals row, rows link to a drill-down page.
 */
export function SortTable<T>({
  rows,
  columns,
  rowKey,
  rowHref,
  initialSort,
  showTotal = true,
  empty = thd.empty,
}: {
  rows: T[];
  columns: SortColumn<T>[];
  rowKey: (row: T) => string;
  rowHref?: (row: T) => string | undefined;
  initialSort?: { key: string; desc: boolean };
  showTotal?: boolean;
  empty?: string;
}) {
  const [sort, setSort] = React.useState(initialSort ?? { key: columns[0]!.key, desc: false });
  const col = columns.find((c) => c.key === sort.key) ?? columns[0]!;
  const sorted = [...rows].sort((a, b) => {
    const x = col.value(a);
    const y = col.value(b);
    const r = typeof x === "number" && typeof y === "number" ? x - y : String(x).localeCompare(String(y), "id");
    return sort.desc ? -r : r;
  });
  if (rows.length === 0) return <p className="py-6 text-center text-sm text-muted-foreground">{empty}</p>;
  return (
    <div className="card-bleed overflow-x-auto scrollbar-thin">
      <Table className="text-[13px]">
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            {columns.map((c) => {
              const active = c.key === sort.key;
              const Icon = active ? (sort.desc ? ArrowDown : ArrowUp) : ArrowUpDown;
              return (
                <TableHead
                  key={c.key}
                  className={cn("h-9 whitespace-nowrap px-2", c.numeric && "text-right", c.className)}
                  aria-sort={active ? (sort.desc ? "descending" : "ascending") : "none"}
                >
                  <button
                    type="button"
                    onClick={() => setSort({ key: c.key, desc: active ? !sort.desc : Boolean(c.numeric) })}
                    className={cn("inline-flex items-center gap-1 hover:text-foreground", c.numeric && "flex-row-reverse")}
                  >
                    {c.label}
                    <Icon className={cn("size-3", !active && "opacity-40")} />
                  </button>
                </TableHead>
              );
            })}
          </TableRow>
        </TableHeader>
        <TableBody>
          {sorted.map((r) => {
            const href = rowHref?.(r);
            return (
              <TableRow key={rowKey(r)} className={href ? "cursor-pointer" : undefined}>
                {columns.map((c, i) => {
                  const content = c.cell ? c.cell(r) : c.value(r);
                  return (
                    <TableCell key={c.key} className={cn("whitespace-nowrap px-2 py-2", c.numeric && "text-right tabular", c.className)}>
                      {href && i === 0 ? (
                        <Link href={href} className="font-medium underline-offset-4 hover:underline">
                          {content}
                        </Link>
                      ) : (
                        content
                      )}
                    </TableCell>
                  );
                })}
              </TableRow>
            );
          })}
        </TableBody>
        {showTotal ? (
          <TableFooter>
            <TableRow className="hover:bg-transparent">
              {columns.map((c, i) => (
                <TableCell key={c.key} className={cn("whitespace-nowrap px-2 py-2 font-semibold", c.numeric && "text-right tabular", c.className)}>
                  {i === 0 ? thd.cols.total : c.total}
                </TableCell>
              ))}
            </TableRow>
          </TableFooter>
        ) : null}
      </Table>
    </div>
  );
}
