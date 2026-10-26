"use client";

import * as React from "react";
import Link from "next/link";
import { ArrowDown, ArrowUp, ArrowUpDown } from "lucide-react";
import { KpiTile } from "@/components/dashboard/kpi-tile";
import { cn } from "@/lib/utils";
import { metricDelta } from "./lib";
import { ds } from "./strings";
import type { DashMetric } from "./types";

/** KPI tile for a dashboard metric: value, delta, previous and sparkline. */
export function MetricTile({
  label,
  metric,
  format,
  hint,
  lowerIsBetter,
  href,
  size,
}: {
  label: string;
  metric: DashMetric;
  format: (v: number | undefined) => string;
  hint?: string;
  lowerIsBetter?: boolean;
  href?: string;
  size?: "sm" | "md";
}) {
  return (
    <KpiTile
      label={label}
      value={format(metric.value)}
      delta={metricDelta(metric)}
      lowerIsBetter={lowerIsBetter}
      previous={metric.previous != null ? format(metric.previous) : undefined}
      spark={metric.spark}
      hint={hint}
      href={href}
      size={size}
    />
  );
}

export type Column<T> = {
  key: string;
  label: string;
  align?: "right";
  render: (row: T) => React.ReactNode;
  sort?: (row: T) => number | string | undefined;
  total?: React.ReactNode;
};

/**
 * Sortable detail table with an optional totals row. Rows link to
 * rowHref; it scrolls sideways inside its card on phones.
 */
export function SortTable<T>({
  rows,
  columns,
  rowKey,
  rowHref,
  initialSort,
  limit,
  totalLabel = ds.total,
  minWidth = 520,
}: {
  rows: T[];
  columns: Column<T>[];
  rowKey: (row: T) => string;
  rowHref?: (row: T) => string | undefined;
  initialSort?: { key: string; desc: boolean };
  limit?: number;
  totalLabel?: string;
  /** px before the table scrolls sideways */
  minWidth?: number;
}) {
  const [sort, setSort] = React.useState(initialSort);
  const sorted = React.useMemo(() => {
    const col = columns.find((c) => c.key === sort?.key);
    if (!col?.sort || !sort) return rows;
    const get = col.sort;
    return [...rows].sort((a, b) => {
      const x = get(a);
      const y = get(b);
      if (x == null) return 1;
      if (y == null) return -1;
      const r =
        typeof x === "number" && typeof y === "number" ? x - y : String(x).localeCompare(String(y), "id");
      return sort.desc ? -r : r;
    });
  }, [rows, columns, sort]);
  const shown = limit ? sorted.slice(0, limit) : sorted;
  const hasTotals = columns.some((c) => c.total != null);
  if (rows.length === 0) return <p className="py-8 text-center text-sm text-muted-foreground">{ds.noData}</p>;
  return (
    <div className="overflow-x-auto scrollbar-thin">
      <table className="w-full text-sm" style={{ minWidth }}>
        <thead>
          <tr className="border-b text-xs text-muted-foreground">
            {columns.map((c) => {
              const active = sort?.key === c.key;
              const Icon = !active ? ArrowUpDown : sort.desc ? ArrowDown : ArrowUp;
              return (
                <th
                  key={c.key}
                  scope="col"
                  className={cn(
                    "px-2 py-2 font-medium whitespace-nowrap",
                    c.align === "right" ? "text-right" : "text-left",
                  )}
                >
                  {c.sort ? (
                    <button
                      type="button"
                      className={cn(
                        "inline-flex items-center gap-1 hover:text-foreground",
                        active && "text-foreground",
                      )}
                      onClick={() => setSort({ key: c.key, desc: active ? !sort.desc : true })}
                    >
                      {c.label}
                      <Icon className="size-3" />
                    </button>
                  ) : (
                    c.label
                  )}
                </th>
              );
            })}
          </tr>
        </thead>
        <tbody>
          {shown.map((r) => {
            const to = rowHref?.(r);
            return (
              <tr key={rowKey(r)} className="border-b border-border/60 last:border-0 hover:bg-muted/40">
                {columns.map((c, i) => (
                  <td key={c.key} className={cn("px-2 py-2", c.align === "right" && "text-right tabular")}>
                    {i === 0 && to ? (
                      <Link href={to} className="font-medium underline-offset-4 hover:underline">
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
            <tr className="border-t-2 font-semibold">
              {columns.map((c, i) => (
                <td key={c.key} className={cn("px-2 py-2", c.align === "right" && "text-right tabular")}>
                  {i === 0 ? totalLabel : c.total}
                </td>
              ))}
            </tr>
          </tfoot>
        ) : null}
      </table>
    </div>
  );
}
