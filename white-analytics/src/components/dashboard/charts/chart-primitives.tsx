"use client";

import * as React from "react";
import { cn } from "@/lib/utils";
import { formatCompact, formatDate } from "@/lib/format";
import { changeTone, trendColor } from "@/lib/metrics";

export { areaFill, barFill, cssBarFill } from "./chart-gradients";

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
  /** bars only: color each bar by its sign (positive vs negative tone) instead of `color` */
  signed?: boolean;
  /** with `signed`: a negative value is the good one (e.g. cost change) */
  lowerIsBetter?: boolean;
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
 * Tooltip content: a rounded pill, value first and large, label quiet above it.
 * `tone="dark"` is the navy pill used over line charts for emphasis.
 * Works for both time-series (label = ISO date) and categorical (label = name).
 */
export function TooltipBox({
  active,
  payload,
  label,
  seriesMap,
  labelFormatter,
  hideLabel,
  tone = "light",
}: {
  active?: boolean;
  payload?: ReadonlyArray<TooltipRow>;
  label?: unknown;
  seriesMap: Record<string, SeriesDef & { color: string }>;
  labelFormatter?: (label: unknown) => React.ReactNode;
  hideLabel?: boolean;
  tone?: "light" | "dark";
}) {
  if (!active || !payload?.length) return null;
  const rows = payload.filter((p) => p.value != null && p.dataKey != null);
  const heading = labelFormatter
    ? labelFormatter(label)
    : typeof label === "string" && /^\d{4}-\d{2}-\d{2}$/.test(label)
      ? formatDate(label)
      : String(label ?? "");
  const dark = tone === "dark";
  const quiet = dark ? "text-white/70" : "text-muted-foreground";
  // One series: the value leads, large; several: label left, value right.
  const single = rows.length === 1;
  return (
    <div
      className={cn(
        "max-w-[min(18rem,calc(100vw-2rem))] min-w-32 rounded-2xl px-3.5 py-2.5 text-xs",
        dark
          ? "bg-[#2b2263]/95 text-white shadow-[0_16px_34px_-14px_rgba(43,34,99,0.75)] dark:bg-[#3a2163]/95"
          : "bg-popover text-popover-foreground shadow-(--popover-shadow)",
      )}
    >
      {!hideLabel && heading ? (
        <div className={cn("mb-1 font-medium", quiet)}>{heading}</div>
      ) : null}
      <div className="grid gap-1">
        {rows.map((r, i) => {
          const key = String(r.dataKey);
          const def = seriesMap[key];
          const color = def?.color ?? r.color ?? r.stroke ?? r.fill ?? "var(--foreground)";
          const raw = typeof r.value === "number" ? r.value : Number(r.value);
          const val = def?.format ? def.format(raw) : formatCompact(raw);
          const name = def?.label ?? String(r.name ?? key);
          return (
            <div key={`${key}-${i}`} className="flex items-center gap-2">
              <span aria-hidden className="size-2 shrink-0 rounded-full" style={{ background: color }} />
              <span className={cn("truncate", quiet, single ? "order-last" : "flex-1")}>{name}</span>
              <span className={cn("shrink-0 font-semibold tabular", single ? "text-[15px] leading-tight" : "text-[13px]")}>
                {val}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

/** Active point on a line: the series dot inside a soft ring. Pass as `activeDot={ringDot(color)}`. */
export function ringDot(color: string) {
  return function RingDot({ cx, cy }: { cx?: number; cy?: number }) {
    if (cx == null || cy == null) return <g />;
    return (
      <g>
        <circle cx={cx} cy={cy} r={10} fill={color} fillOpacity={0.16} />
        <circle cx={cx} cy={cy} r={5} fill={color} stroke="var(--card)" strokeWidth={2.5} />
      </g>
    );
  };
}

/**
 * Legend, always present for ≥ 2 series. Rect for bars/areas, line for lines.
 * With `onSelect` each item is a toggle button (e.g. filter by that series);
 * `activeKey` marks the selected one and dims the rest.
 */
export function SeriesLegend({
  series,
  className,
  onSelect,
  activeKey,
}: {
  series: Array<SeriesDef & { color: string }>;
  className?: string;
  onSelect?: (series: SeriesDef) => void;
  activeKey?: string | null;
}) {
  if (series.length < 2) return null;
  return (
    <ul
      className={cn("flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground", className)}
    >
      {series.map((s) => {
        const swatch =
          s.type === "line" || !s.type ? (
            <span aria-hidden className="h-0.5 w-4 rounded-full" style={{ background: s.color }} />
          ) : (
            <span aria-hidden className="size-2.5 rounded-full" style={{ background: s.color }} />
          );
        const dimmed = activeKey != null && activeKey !== s.key;
        return (
          <li key={s.key} className={cn("flex items-center gap-1.5", dimmed && "opacity-50")}>
            {onSelect ? (
              <button
                type="button"
                onClick={() => onSelect(s)}
                aria-pressed={activeKey === s.key}
                className="-mx-1.5 flex min-h-7 cursor-pointer items-center gap-1.5 rounded-full px-2 outline-none hover:bg-muted hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring aria-pressed:font-medium aria-pressed:text-foreground"
              >
                {swatch}
                {s.label}
              </button>
            ) : (
              <>
                {swatch}
                {s.label}
              </>
            )}
          </li>
        );
      })}
    </ul>
  );
}

/**
 * Keyboard and screen-reader route to the clickable marks of a canvas-like
 * chart. Visually hidden until a button inside it takes focus.
 */
export function MarkButtons<T>({
  marks,
  label,
  onSelect,
  title = "Pilih bagian grafik untuk melihat rinciannya",
}: {
  marks: T[];
  label: (mark: T) => string;
  onSelect: (mark: T, index: number) => void;
  title?: string;
}) {
  if (marks.length === 0) return null;
  return (
    <div className="sr-only focus-within:not-sr-only focus-within:mt-3 focus-within:block focus-within:rounded-2xl focus-within:bg-muted/50 focus-within:px-3 focus-within:py-2.5">
      <p className="mb-1.5 text-xs font-medium">{title}</p>
      <div className="flex max-h-40 flex-wrap gap-1.5 overflow-y-auto">
        {marks.map((m, i) => (
          <button
            key={i}
            type="button"
            onClick={() => onSelect(m, i)}
            className="rounded-full border bg-card px-2.5 py-1 text-xs outline-none hover:bg-muted focus-visible:ring-2 focus-visible:ring-ring"
          >
            {label(m)}
          </button>
        ))}
      </div>
    </div>
  );
}

/** Fill for a signed bar: good change in the positive tone, bad in the negative. */
export function signedFill(value: number, lowerIsBetter = false): string {
  return changeTone(value, lowerIsBetter) === "neutral" ? MUTED_SERIES : trendColor(value, lowerIsBetter);
}

/** Recharts chart-level click state to a data index. */
export function activeIndexOf(
  state: { activeTooltipIndex?: number | string | null } | null | undefined,
): number | null {
  const raw = state?.activeTooltipIndex;
  const idx = typeof raw === "number" ? raw : Number(raw);
  return raw != null && Number.isInteger(idx) && idx >= 0 ? idx : null;
}

export const AXIS_TICK = { fill: "var(--muted-foreground)", fontSize: 11 } as const;
export const GRID_STROKE = "var(--grid)";
export const GRID_DASH = "4 6";
export const CURSOR_STROKE = "var(--axis)";
/** Hover band behind a bar group: a faint brand-tinted rounded column. */
export const BAR_CURSOR = { fill: "var(--brand)", fillOpacity: 0.07, radius: 12 } as const;
/** Fully rounded data end (recharts clamps the radius to half the bar). */
export const BAR_RADIUS: [number, number, number, number] = [999, 999, 0, 0];
export const BAR_RADIUS_H: [number, number, number, number] = [0, 999, 999, 0];
/** Floating bars (waterfall steps, ranges): round both ends. */
export const BAR_RADIUS_ALL = 999;
