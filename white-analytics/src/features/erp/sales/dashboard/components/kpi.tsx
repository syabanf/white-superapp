import Link from "next/link";
import { Card } from "@/components/ui/card";
import { DeltaBadge } from "@/components/dashboard/delta-badge";
import { InfoHint } from "@/components/dashboard/info-hint";
import { Sparkline } from "@/components/dashboard/sparkline";
import { ArrowDownRight, ArrowUpRight } from "lucide-react";
import { computeDelta, trendTextClass } from "@/lib/metrics/delta";
import { cn } from "@/lib/utils";
import { LOWER_IS_BETTER, NO_COMPARE, findKpi, formatKpi, formatKpiExact, kpiKind } from "../format";
import { dd } from "../strings";
import type { DashboardKpi } from "../types";

/**
 * KPI tile: value, delta vs the comparison (points for rates, colored by
 * whether the change is good), previous value and the monthly sparkline.
 */
export function KpiCard({ kpi, compare, href }: { kpi: DashboardKpi; compare: string; href?: string }) {
  const [label, hint] = dd.kpi[kpi.key] ?? [kpi.key, ""];
  const value = formatKpi(kpi.key, kpi.value);
  const rate = kpiKind(kpi.key) === "pct";
  const delta = computeDelta(kpi.value, kpi.previous);
  const compared = !NO_COMPARE.has(kpi.key);
  const showDelta = compared && (rate || kpi.previous !== 0);
  const exact = formatKpiExact(kpi.key, kpi.value);
  const lower = LOWER_IS_BETTER.has(kpi.key);
  const compareLabel = compare === "year" ? dd.compare.year : dd.compare.previous;
  return (
    <Card className="lift group/kpi relative min-w-0 gap-0 overflow-hidden py-0 hover:ring-brand/25">
      <div className="flex min-w-0 flex-col gap-3 p-4">
        <div className="flex min-w-0 items-start gap-1.5 text-muted-foreground">
          <span className="label-mono line-clamp-2">{label}</span>
          <InfoHint className="ml-auto">{exact === value ? hint : `${hint} ${exact}.`}</InfoHint>
        </div>
        {href ? (
          <Link href={href} className="min-w-0 truncate rounded-sm text-[24px] leading-none font-bold tracking-[-0.035em] underline-offset-4 hover:underline">
            {value}
          </Link>
        ) : (
          <span className="min-w-0 truncate text-[24px] leading-none font-bold tracking-[-0.035em]">{value}</span>
        )}
        <div className="flex min-w-0 items-center justify-between gap-2">
          {showDelta ? <DeltaBadge delta={delta} lowerIsBetter={lower} mode={rate ? "abs" : "pct"} digits={1} suffix={rate ? dd.points : undefined} /> : <span />}
          {kpi.spark.length > 1 ? (
            <Sparkline values={kpi.spark} lowerIsBetter={lower} width={64} height={22} className="hidden shrink-0 opacity-70 group-hover/kpi:opacity-100 min-[420px]:block" />
          ) : null}
        </div>
        {compared ? (
          <p className={cn("truncate text-xs", showDelta ? trendTextClass(delta, lower) : "text-muted-foreground")}>
            {compareLabel}: <span className="tabular">{formatKpi(kpi.key, kpi.previous)}</span>
            {showDelta && delta.direction !== "flat"
              ? (() => {
                  const Icon = delta.direction === "up" ? ArrowUpRight : ArrowDownRight;
                  return <Icon aria-hidden className="ml-0.5 inline size-3 align-[-2px]" strokeWidth={2.5} />;
                })()
              : null}
          </p>
        ) : null}
      </div>
    </Card>
  );
}

/** A row of KPI tiles picked by key, in that order. */
export function KpiRow({
  kpis,
  keys,
  compare,
  hrefs = {},
  className,
}: {
  kpis: DashboardKpi[];
  keys: string[];
  compare: string;
  hrefs?: Record<string, string>;
  className?: string;
}) {
  return (
    <div className={cn("grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6", className)}>
      {keys.map((k) => (
        <KpiCard key={k} kpi={findKpi(kpis, k)} compare={compare} href={hrefs[k]} />
      ))}
    </div>
  );
}
