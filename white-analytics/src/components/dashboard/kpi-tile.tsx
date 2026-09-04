import type { ReactNode } from "react";
import { Info } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Sparkline } from "@/components/dashboard/sparkline";
import { DeltaBadge } from "@/components/dashboard/delta-badge";
import { cn } from "@/lib/utils";
import type { Delta } from "@/lib/metrics";
import { t } from "@/i18n/id";

export type KpiTileProps = {
  label: string;
  /** already-formatted value string */
  value: string;
  delta?: Delta | null;
  lowerIsBetter?: boolean;
  /** already-formatted previous value */
  previous?: string;
  spark?: number[];
  hint?: string;
  icon?: ReactNode;
  /** small caption under the value (e.g. "42 postingan") */
  caption?: ReactNode;
  className?: string;
  /** sm = compact secondary strip · md = default · lg = hero */
  size?: "sm" | "md" | "lg";
  accentVar?: string;
};

/**
 * Stat tile: label · value · delta · sparkline. The number IS the chart —
 * generous padding and a large figure so it reads as the focal point, not a chip.
 */
export function KpiTile({
  label,
  value,
  delta,
  lowerIsBetter,
  previous,
  spark,
  hint,
  icon,
  caption,
  className,
  size = "md",
  accentVar,
}: KpiTileProps) {
  const pad = size === "sm" ? "px-3.5 py-3" : size === "lg" ? "p-6" : "p-5";
  const valueSize =
    size === "sm" ? "text-lg" : size === "lg" ? "text-[38px] md:text-[44px]" : "text-[26px] md:text-[30px]";

  return (
    <Card className={cn("lift group/kpi relative min-w-0 gap-0 overflow-hidden py-0 hover:ring-brand/25", className)}>
      {accentVar ? <span aria-hidden className="absolute inset-x-0 top-0 h-0.5" style={{ background: accentVar }} /> : null}
      <div className={cn("flex min-w-0 flex-col", pad, size === "sm" ? "gap-1.5" : "gap-3")}>
        <div className="flex min-w-0 items-start gap-1.5 text-muted-foreground">
          {icon ? <span className="text-muted-foreground/80 [&>svg]:size-3.5">{icon}</span> : null}
          <span className="label-mono line-clamp-2 tracking-[0.08em]">{label}</span>
          {hint ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <button
                  type="button"
                  className="mt-px ml-auto shrink-0 text-muted-foreground/50 hover:text-muted-foreground"
                  aria-label={hint}
                >
                  <Info className="size-3.5" />
                </button>
              </TooltipTrigger>
              <TooltipContent className="max-w-60 text-xs">{hint}</TooltipContent>
            </Tooltip>
          ) : null}
        </div>

        <div className="flex min-w-0 items-end justify-between gap-3">
          <span className={cn("min-w-0 leading-none font-bold tracking-[-0.035em] break-words", valueSize)}>
            {value}
          </span>
          {spark && spark.length > 1 ? (
            <Sparkline
              values={spark}
              width={size === "sm" ? 56 : 84}
              height={size === "sm" ? 20 : 30}
              className="mb-0.5 opacity-70 transition-opacity duration-200 group-hover/kpi:opacity-100"
            />
          ) : null}
        </div>

        {delta || caption || previous ? (
          <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
            {delta ? <DeltaBadge delta={delta} lowerIsBetter={lowerIsBetter} /> : null}
            {caption ? <span className="truncate text-xs text-muted-foreground">{caption}</span> : null}
            {previous && !caption ? (
              <span className="truncate text-xs text-muted-foreground">
                {t.common.previousPeriod}: <span className="tabular">{previous}</span>
              </span>
            ) : null}
          </div>
        ) : null}
      </div>
    </Card>
  );
}

export function KpiGrid({
  children,
  className,
  cols,
}: {
  children: ReactNode;
  className?: string;
  cols?: 2 | 3 | 4 | 5 | 6;
}) {
  const grid =
    cols === 2
      ? "grid-cols-1 sm:grid-cols-2"
      : cols === 3
        ? "grid-cols-1 sm:grid-cols-3"
        : cols === 5
          ? "grid-cols-2 md:grid-cols-3 xl:grid-cols-5"
          : cols === 6
            ? "grid-cols-2 md:grid-cols-3 xl:grid-cols-6"
            : "grid-cols-2 lg:grid-cols-4";
  return <div className={cn("grid gap-4", grid, className)}>{children}</div>;
}
