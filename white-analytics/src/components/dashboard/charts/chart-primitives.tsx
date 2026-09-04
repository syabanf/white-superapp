"use client";

import * as React from "react";
import { cn } from "@/lib/utils";
import { formatCompact, formatDate } from "@/lib/format";

/** Categorical slot color by index (fixed order, never cycled past 8). */
export function slotColor(i: number): string {
  const n = Math.min(Math.max(i, 0), 7) + 1;
  return `var(--chart-${n})`;
}

export const MUTED_SERIES = "var(--chart-emphasis-muted)";

export type SeriesDef = {
  key: string;
  label: string;
  color?: string;
  type?: "line" | "area" | "bar";
  format?: (v: number) => string;
};

export type TooltipRow = {
  dataKey?: string | number;
  name?: string | number;
  value?: number | string | ReadonlyArray<number | string>;
  color?: string;
  stroke?: string;
  fill?: string;
  payload?: Record<string, unknown>;
};

/**
 * Tooltip content — values lead, labels follow; line keys not boxes.
 * Works for both time-series (label = ISO date) and categorical (label = name).
 */
export function TooltipBox({
  active,
  payload,
  label,
  seriesMap,
  labelFormatter,
  hideLabel,
}: {
  active?: boolean;
  payload?: ReadonlyArray<TooltipRow>;
  label?: unknown;
  seriesMap: Record<string, SeriesDef & { color: string }>;
  labelFormatter?: (label: unknown) => React.ReactNode;
  hideLabel?: boolean;
}) {
  if (!active || !payload?.length) return null;
  const rows = payload.filter((p) => p.value != null && p.dataKey != null);
  const heading = labelFormatter ? labelFormatter(label) : typeof label === "string" && /^\d{4}-\d{2}-\d{2}$/.test(label) ? formatDate(label) : String(label ?? "");
  return (
    <div className="min-w-36 rounded-lg border bg-popover px-3 py-2 text-xs text-popover-foreground shadow-lg">
      {!hideLabel && heading ? <div className="mb-1.5 font-medium text-muted-foreground">{heading}</div> : null}
      <div className="grid gap-1">
        {rows.map((r, i) => {
          const key = String(r.dataKey);
          const def = seriesMap[key];
          const color = def?.color ?? r.color ?? r.stroke ?? r.fill ?? "var(--foreground)";
          const raw = typeof r.value === "number" ? r.value : Number(r.value);
          const val = def?.format ? def.format(raw) : formatCompact(raw);
          return (
            <div key={`${key}-${i}`} className="flex items-center gap-2">
              <span aria-hidden className="h-0.5 w-3 shrink-0 rounded-full" style={{ background: color }} />
              <span className="flex-1 truncate text-muted-foreground">{def?.label ?? String(r.name ?? key)}</span>
              <span className="font-semibold tabular text-foreground">{val}</span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

/** Legend — always present for ≥ 2 series. Rect for bars/areas, line for lines. */
export function SeriesLegend({
  series,
  className,
}: {
  series: Array<SeriesDef & { color: string }>;
  className?: string;
}) {
  if (series.length < 2) return null;
  return (
    <ul className={cn("flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground", className)}>
      {series.map((s) => (
        <li key={s.key} className="flex items-center gap-1.5">
          {s.type === "line" || !s.type ? (
            <span aria-hidden className="h-0.5 w-4 rounded-full" style={{ background: s.color }} />
          ) : (
            <span aria-hidden className="size-2.5 rounded-[3px]" style={{ background: s.color }} />
          )}
          {s.label}
        </li>
      ))}
    </ul>
  );
}

export const AXIS_TICK = { fill: "var(--muted-foreground)", fontSize: 11 } as const;
export const GRID_STROKE = "var(--grid)";
export const CURSOR_STROKE = "var(--axis)";
