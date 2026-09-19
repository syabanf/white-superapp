"use client";

import * as React from "react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { formatCompact } from "@/lib/format";

const SEQ_STEPS = ["--seq-100", "--seq-200", "--seq-300", "--seq-400", "--seq-500", "--seq-600", "--seq-700"];

export type HeatCell = { row: number; col: number; value: number; count?: number };

/**
 * Sequential heatmap (one hue, light→dark). Zero cells recede to the surface.
 * Each cell is its own hover target with a tooltip.
 */
export function Heatmap({
  cells,
  rows,
  cols,
  rowLabels,
  colLabels,
  format = formatCompact,
  valueLabel,
  countLabel,
  className,
  cellSize = 16,
}: {
  cells: HeatCell[];
  rows: number;
  cols: number;
  rowLabels: string[];
  colLabels: string[];
  format?: (v: number) => string;
  valueLabel?: string;
  countLabel?: string;
  className?: string;
  cellSize?: number;
}) {
  const max = Math.max(0, ...cells.map((c) => c.value));
  const grid = React.useMemo(() => {
    const m = new Map<string, HeatCell>();
    for (const c of cells) m.set(`${c.row}-${c.col}`, c);
    return m;
  }, [cells]);

  return (
    <div className={cn("overflow-x-auto scrollbar-thin", className)}>
      <div
        className="inline-grid gap-[2px]"
        style={{ gridTemplateColumns: `auto repeat(${cols}, ${cellSize}px)` }}
      >
        <div />
        {colLabels.map((l, i) => (
          <div
            key={i}
            className="text-center text-xs text-muted-foreground tabular"
            style={{ width: cellSize }}
          >
            {i % 3 === 0 ? l : ""}
          </div>
        ))}
        {Array.from({ length: rows }).map((_, r) => (
          <React.Fragment key={r}>
            <div className="pr-2 text-xs leading-[16px] text-muted-foreground" style={{ height: cellSize }}>
              {rowLabels[r]}
            </div>
            {Array.from({ length: cols }).map((_, c) => {
              const cell = grid.get(`${r}-${c}`);
              const v = cell?.value ?? 0;
              const step =
                max > 0 && v > 0
                  ? SEQ_STEPS[Math.min(SEQ_STEPS.length - 1, Math.floor((v / max) * SEQ_STEPS.length))]
                  : null;
              const bg = step ? `var(${step})` : "var(--muted)";
              return (
                <Tooltip key={c}>
                  <TooltipTrigger asChild>
                    <div
                      role="img"
                      aria-label={`${rowLabels[r]} ${colLabels[c]}: ${format(v)}`}
                      className="rounded-[3px] transition-transform hover:scale-110 hover:ring-2 hover:ring-foreground/40"
                      style={{ width: cellSize, height: cellSize, background: bg }}
                    />
                  </TooltipTrigger>
                  <TooltipContent className="text-xs">
                    <div className="font-medium">
                      {rowLabels[r]} · {colLabels[c]}
                    </div>
                    <div>
                      {valueLabel ?? "Nilai"}: <span className="font-semibold tabular">{format(v)}</span>
                    </div>
                    {cell?.count != null ? (
                      <div>
                        {countLabel ?? "Jumlah"}: <span className="tabular">{cell.count}</span>
                      </div>
                    ) : null}
                  </TooltipContent>
                </Tooltip>
              );
            })}
          </React.Fragment>
        ))}
      </div>
      <div className="mt-2 flex items-center gap-1.5 text-xs text-muted-foreground">
        <span>Rendah</span>
        {SEQ_STEPS.map((s) => (
          <span key={s} className="h-2 w-4 rounded-[2px]" style={{ background: `var(${s})` }} />
        ))}
        <span>Tinggi</span>
      </div>
    </div>
  );
}
