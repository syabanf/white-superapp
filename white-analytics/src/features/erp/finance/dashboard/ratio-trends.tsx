"use client";

import { useRouter } from "next/navigation";
import { ChartCard } from "@/components/dashboard/chart-card";
import { DeltaText } from "@/components/dashboard/delta-badge";
import { seriesDelta } from "@/lib/metrics";
import { useDrillParams } from "@/features/erp/shared/drill";
import { formatNumber } from "@/lib/format";
import { formatMetric } from "../components/analysis";
import { fs } from "../strings";
import { MonthlyChart, slotColor } from "./charts";
import { compactIdr, monthBounds } from "./lib";
import type { RatioTrend } from "./types";
import { ds } from "./strings";

const GROUPS = ["profitability", "liquidity", "leverage", "efficiency", "cash"] as const;

const axis = (unit: string) => (v: number) =>
  unit === "idr" ? compactIdr(v) : unit === "pct" ? `${formatNumber(v, 0)}%` : formatNumber(v, unit === "ratio" ? 1 : 0);

/** Each ratio over the last 12 months with its healthy threshold, grouped like the metric cards. */
export function RatioTrends({ data }: { data: RatioTrend }) {
  const router = useRouter();
  const { hrefWith } = useDrillParams();
  // A month zooms the analysis above into that month.
  const onMonth = (i: number) => {
    const m = data.months[i];
    if (m) router.push(hrefWith({ ...monthBounds(m), preset: null }), { scroll: true });
  };
  return (
    <div className="page-stack">
      {GROUPS.map((g, gi) => {
        const series = data.series.filter((s) => s.group === g);
        if (!series.length) return null;
        return (
          <section key={g} className="flex flex-col gap-(--gap-inner)">
            <h2 className="flex min-h-8 items-center text-base font-semibold">
              {ds.ratios.title}: {fs.analysis.groups[g]}
            </h2>
            <div className="grid gap-(--gap) md:grid-cols-2 xl:grid-cols-3">
              {series.map((s) => {
                const values = s.points.map((p) => (p.unavailable ? 0 : p.value));
                const last = s.points.at(-1);
                const bench = s.benchmark ?? null;
                return (
                  <ChartCard
                    key={s.key}
                    title={s.label}
                    description={`${s.lowerIsBetter ? ds.ratios.belowIsGood : ds.ratios.aboveIsGood}.${bench != null ? ` ${ds.ratios.benchmark}: ${formatMetric(bench, s.unit)}.` : ""}`}
                    actions={
                      <span className="flex flex-col items-end text-sm font-semibold tabular">
                        {last && !last.unavailable ? formatMetric(last.value, s.unit) : ds.ratios.unavailable}
                        <DeltaText
                          delta={seriesDelta(s.points.filter((p) => !p.unavailable).map((p) => p.value))}
                          lowerIsBetter={s.lowerIsBetter}
                          className="text-xs"
                          title={ds.ratios.twelveMonths}
                        />
                      </span>
                    }
                  >
                    <MonthlyChart
                      months={data.months}
                      height={180}
                      yFormat={axis(s.unit)}
                      valueFormat={(v) => formatMetric(v, s.unit)}
                      referenceY={bench}
                      referenceLabel={ds.ratios.benchmark}
                      onMonthClick={onMonth}
                      series={[{ key: s.key, label: s.label, color: slotColor(gi), type: "line", values }]}
                    />
                  </ChartCard>
                );
              })}
            </div>
          </section>
        );
      })}
    </div>
  );
}
