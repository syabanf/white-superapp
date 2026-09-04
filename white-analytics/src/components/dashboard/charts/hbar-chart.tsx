"use client";

import * as React from "react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { formatCompact, formatPercent } from "@/lib/format";
import { slotColor } from "./chart-primitives";

export type HBarItem = {
  label: string;
  value: number;
  /** optional secondary text at right (e.g. "12,3%") */
  secondary?: string;
  /** optional color override — for ordinal ramps or emphasis */
  color?: string;
  href?: string;
  muted?: boolean;
};

/**
 * Horizontal bar list — the default for comparing categories with long names.
 * One series → one hue (slot 1). Bars ≤ 20px thick, 4px rounded data-end, square at baseline.
 * Value at the tip (outside), label above. Every row is its own hover target.
 */
export function HBarChart({
  items,
  format = formatCompact,
  showShare = false,
  color = slotColor(0),
  className,
  maxItems,
  emptyLabel = "–",
}: {
  items: HBarItem[];
  format?: (v: number) => string;
  /** show each item's share of the total as secondary text */
  showShare?: boolean;
  color?: string;
  className?: string;
  maxItems?: number;
  emptyLabel?: string;
}) {
  const rows = maxItems ? items.slice(0, maxItems) : items;
  const max = Math.max(0, ...rows.map((r) => r.value));
  const total = rows.reduce((a, b) => a + b.value, 0);
  if (rows.length === 0) return <p className="py-6 text-center text-sm text-muted-foreground">{emptyLabel}</p>;
  return (
    <ul className={cn("space-y-2.5", className)}>
      {rows.map((r, i) => {
        const pct = max > 0 ? (r.value / max) * 100 : 0;
        const share = total > 0 ? (r.value / total) * 100 : 0;
        const barColor = r.color ?? (r.muted ? "var(--chart-emphasis-muted)" : color);
        const content = (
          <li className="group">
            <div className="mb-1 flex items-center justify-between gap-3 text-xs">
              <span className="truncate text-foreground/90" title={r.label}>
                {r.label}
              </span>
              <span className="shrink-0 tabular text-muted-foreground">
                <span className="font-medium text-foreground">{format(r.value)}</span>
                {showShare ? <span className="ml-1.5">{formatPercent(share, 1)}</span> : r.secondary ? <span className="ml-1.5">{r.secondary}</span> : null}
              </span>
            </div>
            <div className="h-2 w-full rounded-r-[4px] bg-muted/70">
              <div
                className="h-2 rounded-r-[4px] transition-[width] duration-300 group-hover:brightness-110"
                style={{ width: `${Math.max(pct, r.value > 0 ? 1.5 : 0)}%`, background: barColor }}
              />
            </div>
          </li>
        );
        return (
          <Tooltip key={`${r.label}-${i}`}>
            <TooltipTrigger asChild>{r.href ? <a href={r.href} className="block">{content}</a> : content}</TooltipTrigger>
            <TooltipContent side="top" className="text-xs">
              <span className="font-medium">{r.label}</span> · {format(r.value)}
              {showShare ? ` · ${formatPercent(share, 1)}` : ""}
            </TooltipContent>
          </Tooltip>
        );
      })}
    </ul>
  );
}
