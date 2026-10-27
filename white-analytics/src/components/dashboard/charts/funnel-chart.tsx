"use client";

import * as React from "react";
import Link from "next/link";
import { ArrowDown } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { formatNumber, formatPercent } from "@/lib/format";
import { cssBarFill, slotColor } from "./chart-primitives";

export type FunnelStep = {
  key: string;
  label: string;
  value: number;
  /** left the funnel at this step (shown as drop-off detail) */
  exited?: number;
  href?: string;
};

/**
 * Ordered funnel as left-anchored bars (one hue): bar length is the count,
 * and between steps a connector shows the step conversion and the drop.
 * Each step is a hover target with a tooltip and an optional drill-down link.
 */
export function FunnelChart({
  steps,
  format = formatNumber,
  color = slotColor(0),
  exitedLabel = "Keluar di tahap ini",
  className,
  onSelect,
}: {
  steps: FunnelStep[];
  format?: (v: number) => string;
  color?: string;
  exitedLabel?: string;
  className?: string;
  /** used for steps without `href` */
  onSelect?: (step: FunnelStep, index: number) => void;
}) {
  const top = steps[0]?.value ?? 0;
  if (top === 0)
    return <p className="py-8 text-center text-sm text-muted-foreground">Belum ada data pada periode ini</p>;
  return (
    <ol className={cn("flex flex-col", className)}>
      {steps.map((s, i) => {
        const prev = i > 0 ? steps[i - 1]!.value : null;
        const conv = prev ? (s.value / prev) * 100 : null;
        const ofTop = (s.value / top) * 100;
        const bar = (
          <div className="group">
            <div className="mb-1 flex items-baseline justify-between gap-3 text-xs">
              <span className="truncate font-medium text-foreground/90">{s.label}</span>
              <span className="shrink-0 tabular text-muted-foreground">
                <span className="font-semibold text-foreground">{format(s.value)}</span>
                <span className="ml-1.5">{formatPercent(ofTop, 0)}</span>
              </span>
            </div>
            <div className="h-3 w-full rounded-full bg-muted">
              <div
                className="h-3 rounded-full transition-[width] duration-300 group-hover:brightness-110"
                style={{ width: `${Math.max(ofTop, s.value > 0 ? 1.5 : 0)}%`, background: cssBarFill(color) }}
              />
            </div>
          </div>
        );
        return (
          <li key={s.key}>
            {conv != null ? (
              <div className="flex items-center gap-2 py-1.5 pl-1 text-[11px] text-muted-foreground tabular">
                <ArrowDown className="size-3" aria-hidden />
                <span>
                  <span className="font-medium text-foreground">{formatPercent(conv, 0)}</span> lanjut
                </span>
                {prev != null && prev - s.value > 0 ? <span>· {format(prev - s.value)} berhenti</span> : null}
              </div>
            ) : null}
            <Tooltip>
              <TooltipTrigger asChild>
                {s.href ? (
                  <Link
                    href={s.href}
                    className="block cursor-pointer rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  >
                    {bar}
                  </Link>
                ) : onSelect ? (
                  <button
                    type="button"
                    onClick={() => onSelect(s, i)}
                    className="block w-full cursor-pointer rounded-md text-left outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  >
                    {bar}
                  </button>
                ) : (
                  <div
                    tabIndex={0}
                    className="rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  >
                    {bar}
                  </div>
                )}
              </TooltipTrigger>
              <TooltipContent side="top" className="text-xs">
                <div className="font-medium">{s.label}</div>
                <div className="tabular">
                  {format(s.value)} ({formatPercent(ofTop, 1)} dari awal)
                  {conv != null ? ` · konversi ${formatPercent(conv, 1)}` : ""}
                </div>
                {s.exited ? (
                  <div className="tabular">
                    {exitedLabel}: {format(s.exited)}
                  </div>
                ) : null}
              </TooltipContent>
            </Tooltip>
          </li>
        );
      })}
    </ol>
  );
}
