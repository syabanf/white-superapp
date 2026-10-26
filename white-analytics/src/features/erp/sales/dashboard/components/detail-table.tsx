"use client";

import * as React from "react";
import Link from "next/link";
import { ArrowDown, ArrowUp, ArrowUpDown } from "lucide-react";
import { cn } from "@/lib/utils";
import { dd } from "../strings";

export type Column<T> = {
  key: string;
  label: string;
  /** numeric columns are right-aligned and sortable by `sort` */
  numeric?: boolean;
  render: (row: T) => React.ReactNode;
  sort?: (row: T) => number | string;
  /** totals row cell; omit to leave it blank */
  total?: (rows: T[]) => React.ReactNode;
  /** hide below the sm breakpoint */
  optional?: boolean;
  /** hide below the 2xl breakpoint (secondary detail in half-width cards) */
  wide?: boolean;
};

/** Sortable detail table with an optional totals row; rows can link to a drill-down. */
export function DetailTable<T>({
  rows,
  columns,
  rowKey,
  href,
  initialSort,
  limit,
  empty = dd.empty,
}: {
  rows: T[];
  columns: Column<T>[];
  rowKey: (row: T) => string;
  href?: (row: T) => string | undefined;
  initialSort?: { key: string; desc: boolean };
  limit?: number;
  empty?: string;
}) {
  const [sort, setSort] = React.useState(initialSort);
  const sorted = React.useMemo(() => {
    const col = columns.find((c) => c.key === sort?.key);
    if (!col?.sort) return rows;
    const pick = col.sort;
    return [...rows].sort((a, b) => {
      const x = pick(a);
      const y = pick(b);
      const cmp = typeof x === "number" && typeof y === "number" ? x - y : String(x).localeCompare(String(y), "id");
      return sort!.desc ? -cmp : cmp;
    });
  }, [rows, columns, sort]);
  const shown = limit ? sorted.slice(0, limit) : sorted;
  const hasTotals = columns.some((c) => c.total);
  if (!rows.length) return <p className="py-6 text-center text-sm text-muted-foreground">{empty}</p>;
  const cell = (c: Column<T>) => cn("px-3 py-2", c.numeric ? "text-right tabular whitespace-nowrap" : "text-left", c.optional && "max-sm:hidden", c.wide && "max-2xl:hidden");
  return (
    <div className="-mx-1 overflow-x-auto scrollbar-thin">
      <table className="w-full min-w-[28rem] text-sm max-sm:min-w-0">
        <thead className="text-xs text-muted-foreground">
          <tr className="border-b">
            {columns.map((c) => {
              const active = sort?.key === c.key;
              const Icon = !active ? ArrowUpDown : sort!.desc ? ArrowDown : ArrowUp;
              return (
                <th key={c.key} className={cn(cell(c), "font-medium")} aria-sort={active ? (sort!.desc ? "descending" : "ascending") : undefined}>
                  {c.sort ? (
                    <button
                      type="button"
                      className={cn("inline-flex items-center gap-1 hover:text-foreground", c.numeric && "flex-row-reverse")}
                      onClick={() => setSort({ key: c.key, desc: active ? !sort!.desc : true })}
                    >
                      {c.label}
                      <Icon className={cn("size-3", !active && "opacity-40")} />
                    </button>
                  ) : (
                    c.label
                  )}
                </th>
              );
            })}
          </tr>
        </thead>
        <tbody className="divide-y">
          {shown.map((r) => {
            const link = href?.(r);
            return (
              <tr key={rowKey(r)} className={cn(link && "hover:bg-muted/40")}>
                {columns.map((c, i) => (
                  <td key={c.key} className={cell(c)}>
                    {link && i === 0 ? (
                      <Link href={link} className="font-medium hover:underline">
                        {c.render(r)}
                      </Link>
                    ) : (
                      c.render(r)
                    )}
                  </td>
                ))}
              </tr>
            );
          })}
        </tbody>
        {hasTotals ? (
          <tfoot>
            <tr className="border-t bg-muted/30 font-semibold">
              {columns.map((c, i) => (
                <td key={c.key} className={cell(c)}>
                  {i === 0 ? dd.total : c.total?.(rows)}
                </td>
              ))}
            </tr>
          </tfoot>
        ) : null}
      </table>
    </div>
  );
}
