"use client";

import * as React from "react";
import Link from "next/link";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { formatDecimal, formatNumber } from "@/lib/format";
import { slotColor } from "./chart-primitives";

export type BoxRow = {
  label: string;
  min: number;
  p25: number;
  median: number;
  p75: number;
  max: number;
  avg?: number;
  count?: number;
  href?: string;
};

function niceMax(v: number): number {
  if (v <= 0) return 1;
  const pow = 10 ** Math.floor(Math.log10(v));
  const n = v / pow;
  return (n <= 1 ? 1 : n <= 2 ? 2 : n <= 5 ? 5 : 10) * pow;
}

/**
 * Horizontal box plots on one shared axis: whisker min..max, box p25..p75,
 * a median rule and an optional mean dot. Rows are hover targets; a row with
 * `href` links, `onSelect` makes rows buttons.
 */
export function BoxPlot({
  rows,
  format = (v) => formatDecimal(v, 1),
  unit = "",
  color = slotColor(0),
  className,
  countLabel = "kandidat",
  onSelect,
}: {
  rows: BoxRow[];
  format?: (v: number) => string;
  /** appended to axis ticks, e.g. " hr" */
  unit?: string;
  color?: string;
  className?: string;
  /** noun after the count in the tooltip, e.g. "kandidat" */
  countLabel?: string;
  onSelect?: (row: BoxRow) => void;
}) {
  if (rows.length === 0)
    return <p className="py-8 text-center text-sm text-muted-foreground">Belum ada data pada periode ini</p>;
  const max = niceMax(Math.max(...rows.map((r) => r.max)));
  const pct = (v: number) => `${Math.min(100, Math.max(0, (v / max) * 100))}%`;
  const ticks = [0, 0.25, 0.5, 0.75, 1].map((t) => t * max);
  return (
    <div className={cn("flex flex-col gap-1", className)}>
      <div className="grid grid-cols-[minmax(5rem,7rem)_minmax(0,1fr)] gap-3">
        <span />
        <div className="relative h-4 text-[11px] text-muted-foreground tabular">
          {ticks.map((t, i) => (
            <span
              key={t}
              className={cn(
                "absolute top-0 whitespace-nowrap",
                i === 0 ? "" : i === ticks.length - 1 ? "-translate-x-full" : "-translate-x-1/2",
              )}
              style={{ left: pct(t) }}
            >
              {formatNumber(t)}
              {unit}
            </span>
          ))}
        </div>
      </div>
      {rows.map((r) => (
        <Tooltip key={r.label}>
          <TooltipTrigger asChild>
            <BoxRowTarget row={r} onPick={onSelect}>
              <span className="truncate text-xs text-foreground/90">{r.label}</span>
              <div className="relative h-6">
                {ticks.map((t) => (
                  <span
                    key={t}
                    aria-hidden
                    className="absolute inset-y-0 w-px bg-[var(--grid)]"
                    style={{ left: pct(t) }}
                  />
                ))}
                <span
                  aria-hidden
                  className="absolute top-1/2 h-px bg-muted-foreground/60"
                  style={{ left: pct(r.min), width: `calc(${pct(r.max)} - ${pct(r.min)})` }}
                />
                <span
                  aria-hidden
                  className="absolute top-1.5 bottom-1.5 w-px bg-muted-foreground/60"
                  style={{ left: pct(r.min) }}
                />
                <span
                  aria-hidden
                  className="absolute top-1.5 bottom-1.5 w-px bg-muted-foreground/60"
                  style={{ left: pct(r.max) }}
                />
                <span
                  aria-hidden
                  className="absolute inset-y-0.5 rounded-[4px] opacity-30"
                  style={{
                    left: pct(r.p25),
                    width: `max(2px, calc(${pct(r.p75)} - ${pct(r.p25)}))`,
                    background: color,
                  }}
                />
                <span
                  aria-hidden
                  className="absolute inset-y-0.5 w-0.5 rounded-full"
                  style={{ left: pct(r.median), background: color }}
                />
                {r.avg != null ? (
                  <span
                    aria-hidden
                    className="absolute top-1/2 size-2 -translate-x-1/2 -translate-y-1/2 rounded-full ring-2 ring-card"
                    style={{ left: pct(r.avg), background: "var(--foreground)" }}
                  />
                ) : null}
              </div>
            </BoxRowTarget>
          </TooltipTrigger>
          <TooltipContent side="top" className="text-xs tabular">
            <div className="mb-1 font-medium">{r.label}</div>
            <div>Median {format(r.median)}</div>
            <div>
              Kuartil {format(r.p25)} sampai {format(r.p75)}
            </div>
            <div>
              Rentang {format(r.min)} sampai {format(r.max)}
            </div>
            {r.avg != null ? <div>Rata-rata {format(r.avg)}</div> : null}
            {r.count != null ? (
              <div>
                {formatNumber(r.count)} {countLabel}
              </div>
            ) : null}
          </TooltipContent>
        </Tooltip>
      ))}
      <div className="mt-1 flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
        <span className="flex items-center gap-1.5">
          <span aria-hidden className="h-2.5 w-4 rounded-[3px] opacity-30" style={{ background: color }} />{" "}
          Kuartil 25 sampai 75
        </span>
        <span className="flex items-center gap-1.5">
          <span aria-hidden className="h-3 w-0.5 rounded-full" style={{ background: color }} /> Median
        </span>
        <span className="flex items-center gap-1.5">
          <span aria-hidden className="size-2 rounded-full bg-foreground" /> Rata-rata
        </span>
      </div>
    </div>
  );
}

const ROW_CLASS =
  "grid w-full grid-cols-[minmax(5rem,7rem)_minmax(0,1fr)] items-center gap-3 rounded-md py-1.5 text-left outline-none hover:bg-muted/40 focus-visible:ring-2 focus-visible:ring-ring";

function BoxRowTarget({
  row,
  onPick,
  children,
  ...rest
}: {
  row: BoxRow;
  onPick?: (row: BoxRow) => void;
  children: React.ReactNode;
} & React.HTMLAttributes<HTMLElement>) {
  if (row.href)
    return (
      <Link href={row.href} className={`${ROW_CLASS} cursor-pointer`} {...rest}>
        {children}
      </Link>
    );
  if (onPick)
    return (
      <button
        type="button"
        {...rest}
        onClick={(e) => {
          rest.onClick?.(e);
          onPick(row);
        }}
        className={`${ROW_CLASS} cursor-pointer`}
      >
        {children}
      </button>
    );
  return (
    <div tabIndex={0} className={ROW_CLASS} {...rest}>
      {children}
    </div>
  );
}
