import type { ReactNode } from "react";
import { DeltaBadge } from "@/components/dashboard/delta-badge";
import { cn } from "@/lib/utils";
import { InfoHint } from "@/components/dashboard/info-hint";
import type { Delta } from "@/lib/metrics";

export type StatStripItem = {
  label: string;
  /** already-formatted */
  value: string;
  delta?: Delta | null;
  lowerIsBetter?: boolean;
  hint?: string;
  caption?: ReactNode;
};

/**
 * Secondary metrics as one hairline strip instead of a wall of separate cards.
 * Auto-fits its columns, so any number of items wraps tidily — the 1px gaps
 * show the container behind them and read as rules.
 */
export function StatStrip({
  items,
  className,
  minWidth = 168,
}: {
  items: StatStripItem[];
  className?: string;
  minWidth?: number;
}) {
  if (items.length === 0) return null;
  return (
    <div className={cn("overflow-hidden rounded-[1.5rem] bg-card shadow-(--card-shadow)", className)}>
      {/* flex (not grid) so a short last row stretches instead of leaving a dead band */}
      <div className="flex flex-wrap">
        {items.map((it) => (
          <div
            key={it.label}
            className="-mt-px -ml-px min-w-0 flex-1 border-t border-l px-4 py-3 transition-colors duration-150 hover:bg-muted/40"
            style={{ minWidth: `${minWidth}px` }}
          >
            <div className="flex min-w-0 items-start gap-1.5 text-muted-foreground">
              <span className="label-mono line-clamp-2">{it.label}</span>
              {it.hint ? <InfoHint className="ml-auto">{it.hint}</InfoHint> : null}
            </div>
            <div className="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1">
              <span className="text-[17px] font-bold tracking-[-0.025em] tabular">{it.value}</span>
              {it.delta ? <DeltaBadge delta={it.delta} lowerIsBetter={it.lowerIsBetter} /> : null}
            </div>
            {it.caption ? <div className="mt-0.5 truncate text-xs text-muted-foreground">{it.caption}</div> : null}
          </div>
        ))}
      </div>
    </div>
  );
}
