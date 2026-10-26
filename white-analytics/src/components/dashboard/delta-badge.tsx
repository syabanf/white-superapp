import { ArrowDownRight, ArrowUpRight, Minus } from "lucide-react";
import { cn } from "@/lib/utils";
import { formatDeltaNumber, formatDeltaPercent } from "@/lib/format";
import { changeTone, computeDelta, type Delta, type Polarity } from "@/lib/metrics";
import { t } from "@/i18n/id";

export type DeltaBadgeProps = {
  delta?: Delta | null;
  /** Alternative to `delta`: raw values, the delta is computed here. */
  current?: number | null;
  previous?: number | null;
  lowerIsBetter?: Polarity;
  /** pct = percentage change · abs = absolute difference */
  mode?: "pct" | "abs";
  digits?: number;
  /** custom label for the number, e.g. formatted currency; sign is up to the caller */
  format?: (delta: Delta) => string;
  /** badge = pill (KPI tiles) · text = inline colored text (tables, captions) */
  variant?: "badge" | "text";
  className?: string;
  suffix?: string;
  title?: string;
};

function resolveDelta({
  delta,
  current,
  previous,
}: Pick<DeltaBadgeProps, "delta" | "current" | "previous">): Delta | null {
  if (delta) return delta;
  if (current == null || previous == null) return null;
  return computeDelta(current, previous);
}

/**
 * Period-over-period change. Color = direction × whether up is good.
 * Always pairs an icon with the number (never color alone).
 */
export function DeltaBadge({
  delta: deltaProp,
  current,
  previous,
  lowerIsBetter = false,
  mode = "pct",
  digits,
  format,
  variant = "badge",
  className,
  suffix,
  title = t.common.vsPrevious,
}: DeltaBadgeProps) {
  const delta = resolveDelta({ delta: deltaProp, current, previous });
  if (!delta) return null;
  const tone = changeTone(delta, lowerIsBetter);
  const label = format
    ? format(delta)
    : mode === "pct"
      ? formatDeltaPercent(delta.direction === "flat" ? 0 : delta.pct, digits ?? 1)
      : formatDeltaNumber(delta.abs, digits ?? 0);
  const Icon = delta.direction === "up" ? ArrowUpRight : delta.direction === "down" ? ArrowDownRight : Minus;
  return (
    <span
      className={cn(
        "inline-flex items-center gap-0.5 font-medium whitespace-nowrap tabular",
        variant === "badge" && "rounded-full border px-1.5 py-0.5 text-xs",
        variant === "badge" && tone === "positive" && "border-positive/25 bg-positive/8",
        variant === "badge" && tone === "negative" && "border-negative/25 bg-negative/8",
        variant === "badge" && tone === "neutral" && "border-border bg-muted",
        tone === "positive" && "text-positive",
        tone === "negative" && "text-negative",
        tone === "neutral" && "text-muted-foreground",
        className,
      )}
      title={title}
    >
      <Icon className={variant === "badge" ? "size-3" : "size-3.5"} strokeWidth={2.5} aria-hidden />
      {label}
      {suffix ? <span className="font-normal opacity-80">{suffix}</span> : null}
    </span>
  );
}

/** Inline colored change for tables and captions: `<DeltaText current={a} previous={b} />`. */
export function DeltaText(props: Omit<DeltaBadgeProps, "variant">) {
  return <DeltaBadge {...props} variant="text" />;
}
