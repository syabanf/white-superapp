import { ArrowDownRight, ArrowUpRight, Minus } from "lucide-react";
import { cn } from "@/lib/utils";
import { formatDeltaNumber, formatDeltaPercent } from "@/lib/format";
import { isGoodChange, type Delta } from "@/lib/metrics";
import { t } from "@/i18n/id";

/**
 * Period-over-period badge. Color = direction × whether up is good.
 * Always pairs an icon with the number (never color alone).
 */
export function DeltaBadge({
  delta,
  lowerIsBetter = false,
  mode = "pct",
  digits,
  className,
  suffix,
}: {
  delta: Delta | null | undefined;
  lowerIsBetter?: boolean;
  mode?: "pct" | "abs";
  digits?: number;
  className?: string;
  suffix?: string;
}) {
  if (!delta) return null;
  const good = isGoodChange(delta, lowerIsBetter);
  const label =
    mode === "pct" ? formatDeltaPercent(delta.pct, digits ?? 1) : formatDeltaNumber(delta.abs, digits ?? 0);
  const Icon = delta.direction === "up" ? ArrowUpRight : delta.direction === "down" ? ArrowDownRight : Minus;
  return (
    <span
      className={cn(
        "inline-flex items-center gap-0.5 rounded-full border px-1.5 py-0.5 text-xs font-medium tabular",
        good === true && "border-positive/25 bg-positive/8 text-positive",
        good === false && "border-negative/25 bg-negative/8 text-negative",
        good === null && "border-border bg-muted text-muted-foreground",
        className,
      )}
      title={t.common.vsPrevious}
    >
      <Icon className="size-3" strokeWidth={2.5} />
      {label}
      {suffix ? <span className="font-normal opacity-80">{suffix}</span> : null}
    </span>
  );
}
