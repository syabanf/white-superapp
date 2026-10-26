"use client";

import * as React from "react";
import Link from "next/link";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { formatCompact } from "@/lib/format";
import { slotColor } from "./chart-primitives";

export type BulletRow = {
  key: string;
  label: string;
  sublabel?: string;
  value: number;
  /** qualitative band, e.g. a salary range */
  rangeMin?: number | null;
  rangeMax?: number | null;
  /** comparison tick, e.g. the expected salary */
  marker?: number | null;
  href?: string;
  /** extra tooltip lines */
  detail?: React.ReactNode;
};

/**
 * Bullet rows on one shared scale: a muted band for the range, a thin bar for
 * the value and a tick for the comparison. A value outside the band is drawn
 * in the warning hue with its label, never color alone.
 */
export function BulletChart({
  rows,
  format = formatCompact,
  rangeLabel = "Rentang",
  valueLabel = "Nilai",
  markerLabel = "Pembanding",
  outsideLabel = "Di luar rentang",
  className,
  onSelect,
}: {
  rows: BulletRow[];
  format?: (v: number) => string;
  rangeLabel?: string;
  valueLabel?: string;
  markerLabel?: string;
  outsideLabel?: string;
  className?: string;
  /** used for rows without `href` */
  onSelect?: (row: BulletRow) => void;
}) {
  if (rows.length === 0)
    return <p className="py-8 text-center text-sm text-muted-foreground">Belum ada data pada periode ini</p>;
  const max = Math.max(...rows.flatMap((r) => [r.value, r.rangeMax ?? 0, r.marker ?? 0])) * 1.08 || 1;
  const pct = (v: number) => `${Math.min(100, (v / max) * 100)}%`;
  return (
    <div className={cn("flex flex-col gap-2.5", className)}>
      {rows.map((r) => {
        const outside =
          r.rangeMax != null && r.rangeMin != null && (r.value > r.rangeMax || r.value < r.rangeMin);
        const body = (
          <div className="group">
            <div className="mb-1 flex items-baseline justify-between gap-3 text-xs">
              <span className="min-w-0 truncate">
                <span className="font-medium text-foreground/90">{r.label}</span>
                {r.sublabel ? <span className="text-muted-foreground"> · {r.sublabel}</span> : null}
              </span>
              <span className="shrink-0 tabular">
                {outside ? (
                  <span className="mr-1.5 text-[11px] text-muted-foreground">{outsideLabel}</span>
                ) : null}
                <span className="font-semibold">{format(r.value)}</span>
              </span>
            </div>
            <div className="relative h-4">
              <span aria-hidden className="absolute inset-y-0 left-0 right-0 rounded-[4px] bg-muted/50" />
              {r.rangeMin != null && r.rangeMax != null ? (
                <span
                  aria-hidden
                  className="absolute inset-y-0 rounded-[4px] bg-[var(--chart-emphasis-muted)] opacity-60"
                  style={{ left: pct(r.rangeMin), width: `calc(${pct(r.rangeMax)} - ${pct(r.rangeMin)})` }}
                />
              ) : null}
              <span
                aria-hidden
                className="absolute top-1/2 left-0 h-1.5 -translate-y-1/2 rounded-r-[4px] group-hover:brightness-110"
                style={{ width: pct(r.value), background: outside ? "var(--status-warning)" : slotColor(0) }}
              />
              {r.marker != null ? (
                <span
                  aria-hidden
                  className="absolute -inset-y-0.5 w-0.5 rounded-full bg-foreground"
                  style={{ left: pct(r.marker) }}
                />
              ) : null}
            </div>
          </div>
        );
        return (
          <Tooltip key={r.key}>
            <TooltipTrigger asChild>
              {r.href ? (
                <Link
                  href={r.href}
                  className="block cursor-pointer rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  {body}
                </Link>
              ) : onSelect ? (
                <button
                  type="button"
                  onClick={() => onSelect(r)}
                  className="block w-full cursor-pointer rounded-md text-left outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  {body}
                </button>
              ) : (
                <div
                  tabIndex={0}
                  className="rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  {body}
                </div>
              )}
            </TooltipTrigger>
            <TooltipContent side="top" className="text-xs tabular">
              <div className="mb-1 font-medium">{r.label}</div>
              <div>
                {valueLabel}: {format(r.value)}
              </div>
              {r.rangeMin != null && r.rangeMax != null ? (
                <div>
                  {rangeLabel}: {format(r.rangeMin)} sampai {format(r.rangeMax)}
                </div>
              ) : null}
              {r.marker != null ? (
                <div>
                  {markerLabel}: {format(r.marker)}
                </div>
              ) : null}
              {r.detail}
            </TooltipContent>
          </Tooltip>
        );
      })}
      <div className="mt-1 flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
        <span className="flex items-center gap-1.5">
          <span aria-hidden className="h-2.5 w-4 rounded-[3px] bg-[var(--chart-emphasis-muted)]" />{" "}
          {rangeLabel}
        </span>
        <span className="flex items-center gap-1.5">
          <span aria-hidden className="h-1.5 w-4 rounded-full" style={{ background: slotColor(0) }} />{" "}
          {valueLabel}
        </span>
        <span className="flex items-center gap-1.5">
          <span aria-hidden className="h-3 w-0.5 rounded-full bg-foreground" /> {markerLabel}
        </span>
      </div>
    </div>
  );
}
