import type { ReactNode } from "react";
import Link from "next/link";
import { DeltaBadge } from "@/components/dashboard/delta-badge";
import { cn } from "@/lib/utils";
import { InfoHint } from "@/components/dashboard/info-hint";
import { computeDelta, type Delta, type Polarity } from "@/lib/metrics";
import { PreviousLine } from "@/components/dashboard/kpi-tile";

export type StatStripItem = {
  label: string;
  /** already-formatted */
  value: string;
  delta?: Delta | null;
  /** Raw values: when `delta` is not given, it is computed from these. */
  currentValue?: number | null;
  previousValue?: number | null;
  lowerIsBetter?: Polarity;
  /** already-formatted previous value, shown under the value and tinted by the delta */
  previous?: string;
  hint?: string;
  caption?: ReactNode;
  href?: string;
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
  scrollOnMobile = false,
  mobileScrollHint,
}: {
  items: StatStripItem[];
  className?: string;
  minWidth?: number;
  /** Keep a single compact row on phones; useful above dense primary content such as a calendar. */
  scrollOnMobile?: boolean;
  mobileScrollHint?: string;
}) {
  if (items.length === 0) return null;
  return (
    <div
      className={cn(
        "rounded-[1.5rem] bg-card shadow-(--card-shadow)",
        scrollOnMobile ? "overflow-x-auto scrollbar-thin md:overflow-hidden" : "overflow-hidden",
        className,
      )}
    >
      {/* flex (not grid) so a short last row stretches instead of leaving a dead band */}
      <div className={cn("flex", scrollOnMobile ? "flex-nowrap md:flex-wrap" : "flex-wrap")}>
        {items.map((it) => {
          const delta =
            it.delta ??
            (it.currentValue != null && it.previousValue != null
              ? computeDelta(it.currentValue, it.previousValue)
              : null);
          const className = cn(
            "-mt-px -ml-px min-w-0 flex-1 border-t border-l px-(--card-pad) py-3 transition-colors duration-150 hover:bg-muted/40",
            it.href && "outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-brand",
            scrollOnMobile && "shrink-0 md:shrink",
          );
          const content = (
            <>
              <div className="flex min-w-0 items-start gap-1.5 text-muted-foreground">
                <span className="label-mono line-clamp-2">{it.label}</span>
                {it.hint ? <InfoHint className="ml-auto">{it.hint}</InfoHint> : null}
              </div>
              <div className="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1">
                <span className="text-[17px] font-bold tracking-[-0.025em] tabular">{it.value}</span>
                {delta ? <DeltaBadge delta={delta} lowerIsBetter={it.lowerIsBetter} /> : null}
              </div>
              {it.caption ? (
                <div className="mt-0.5 truncate text-xs text-muted-foreground">{it.caption}</div>
              ) : it.previous ? (
                <PreviousLine
                  value={it.previous}
                  delta={delta}
                  lowerIsBetter={it.lowerIsBetter}
                  className="mt-0.5 block"
                />
              ) : null}
            </>
          );
          return it.href ? (
            <Link key={it.label} href={it.href} className={className} style={{ minWidth: `${minWidth}px` }}>
              {content}
            </Link>
          ) : (
            <div key={it.label} className={className} style={{ minWidth: `${minWidth}px` }}>
              {content}
            </div>
          );
        })}
      </div>
      {scrollOnMobile && mobileScrollHint ? (
        <p className="sticky left-0 border-t px-4 py-1.5 text-xs text-muted-foreground md:hidden">
          {mobileScrollHint} <span aria-hidden>→</span>
        </p>
      ) : null}
    </div>
  );
}
