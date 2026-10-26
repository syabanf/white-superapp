"use client";

import * as React from "react";
import Link from "next/link";
import { ArrowDown, ArrowUp, ArrowUpDown } from "lucide-react";
import { cn } from "@/lib/utils";
import { dayLabel } from "@/features/publishing/time";
import type { ContentDashboard, DashBucket } from "@/features/content/dashboard";
import { kpiDelta } from "@/features/content/dashboard";
import { cd } from "@/features/content/strings";
import type { Delta } from "@/lib/metrics";

export { overviewHref } from "./links";

/** What every tab needs besides the numbers. All plain data (client props). */
export type DashCtx = {
  base: string;
  clientId: string;
  /** current overview query (period + filters), without `tab` */
  query: string;
  names: Record<string, string>;
  /** comparison switch of the top bar picker */
  compare: boolean;
  plans: { id: string; month: string }[];
  /** dashboard period (WIB days) and its label */
  from: string;
  to: string;
  periodLabel: string;
  /** today in the client's time zone, for overdue checks */
  today: string;
};

export type TabProps = { data: ContentDashboard; ctx: DashCtx };

const MONTH_SHORT = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];

/** "6 Okt" for a week bucket, "Okt 26" for a month bucket. */
export function bucketLabel(start: string, granularity: "week" | "month"): string {
  if (granularity === "week") return dayLabel(start);
  return `${MONTH_SHORT[Number(start.slice(5, 7)) - 1]} ${start.slice(2, 4)}`;
}

export function bucketRangeLabel(start: string, granularity: "week" | "month"): string {
  return granularity === "week" ? cd.weekOf(dayLabel(start, true)) : `${MONTH_SHORT[Number(start.slice(5, 7)) - 1]} ${start.slice(0, 4)}`;
}

/** The plan of the month a bucket starts in, when one exists. */
export function planForMonth(ctx: DashCtx, day: string): string | undefined {
  return ctx.plans.find((p) => p.month === day.slice(0, 7))?.id;
}

export function useDelta(ctx: DashCtx) {
  return React.useCallback(
    (cur: number | string, prev: number | string): Delta | null => (ctx.compare ? kpiDelta(cur, prev) : null),
    [ctx.compare],
  );
}

export function spark(buckets: DashBucket[] | undefined, pick: (b: DashBucket) => number | string): number[] {
  return (buckets ?? []).map((b) => Number(pick(b)) || 0);
}

export function personName(ctx: DashCtx, id: string): string {
  return ctx.names[id] ?? cd.unknownPerson;
}

export type Column<T> = {
  key: string;
  label: string;
  align?: "right";
  value: (row: T) => number | string;
  render?: (row: T) => React.ReactNode;
  total?: React.ReactNode;
};

/**
 * Sortable detail table with an optional totals row. Stays one table on
 * phones and scrolls inside its card.
 */
export function DetailTable<T>({
  rows,
  columns,
  rowKey,
  initialSort,
  totalLabel = cd.total,
  showTotals = true,
  minWidth = 560,
}: {
  rows: T[];
  columns: Column<T>[];
  rowKey: (row: T) => string;
  initialSort?: { key: string; desc: boolean };
  totalLabel?: string;
  showTotals?: boolean;
  minWidth?: number;
}) {
  const [sort, setSort] = React.useState(initialSort ?? { key: columns[0]!.key, desc: false });
  const col = columns.find((c) => c.key === sort.key) ?? columns[0]!;
  const sorted = rows.toSorted((a, b) => {
    const x = col.value(a);
    const y = col.value(b);
    const cmp = typeof x === "number" && typeof y === "number" ? x - y : String(x).localeCompare(String(y), "id");
    return sort.desc ? -cmp : cmp;
  });
  if (rows.length === 0) return <p className="px-1 py-4 text-sm text-muted-foreground">{cd.noData}</p>;
  return (
    <div className="overflow-x-auto scrollbar-thin">
      <table className="w-full text-[13px]" style={{ minWidth }}>
        <thead>
          <tr className="border-b text-left text-xs text-muted-foreground">
            {columns.map((c) => {
              const active = c.key === sort.key;
              const Icon = active ? (sort.desc ? ArrowDown : ArrowUp) : ArrowUpDown;
              return (
                <th key={c.key} className={cn("px-2 py-2 font-normal first:pl-0 last:pr-0", c.align === "right" && "text-right")} aria-sort={active ? (sort.desc ? "descending" : "ascending") : "none"}>
                  <button
                    type="button"
                    onClick={() => setSort({ key: c.key, desc: active ? !sort.desc : c.align === "right" })}
                    className={cn("inline-flex items-center gap-1 hover:text-foreground", active && "text-foreground")}
                  >
                    {c.label}
                    <Icon className="size-3 opacity-60" />
                  </button>
                </th>
              );
            })}
          </tr>
        </thead>
        <tbody className="divide-y">
          {sorted.map((r) => (
            <tr key={rowKey(r)} className="hover:bg-muted/30">
              {columns.map((c) => (
                <td key={c.key} className={cn("px-2 py-2 first:pl-0 last:pr-0", c.align === "right" && "text-right tabular")}>
                  {c.render ? c.render(r) : c.value(r)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
        {showTotals ? (
          <tfoot>
            <tr className="border-t font-medium">
              {columns.map((c, i) => (
                <td key={c.key} className={cn("px-2 py-2 first:pl-0 last:pr-0", c.align === "right" && "text-right tabular")}>
                  {i === 0 ? totalLabel : (c.total ?? "")}
                </td>
              ))}
            </tr>
          </tfoot>
        ) : null}
      </table>
    </div>
  );
}

export function TextLink({ href, children, className }: { href: string; children: React.ReactNode; className?: string }) {
  return (
    <Link href={href} className={cn("font-medium hover:underline", className)}>
      {children}
    </Link>
  );
}

/** Table cell that opens a drill (looks like TextLink). */
export function TextButton({ onClick, children, className }: { onClick: () => void; children: React.ReactNode; className?: string }) {
  return (
    <button type="button" onClick={onClick} className={cn("cursor-pointer text-left font-medium outline-none hover:underline focus-visible:underline", className)}>
      {children}
    </button>
  );
}
