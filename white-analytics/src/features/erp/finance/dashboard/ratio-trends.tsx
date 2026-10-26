"use client";

import { ChartCard } from "@/components/dashboard/chart-card";
import { formatNumber } from "@/lib/format";
import { formatMetric } from "../components/analysis";
import { fs } from "../strings";
import { MonthlyChart, slotColor } from "./charts";
import { compactIdr } from "./lib";
import type { RatioTrend } from "./types";
import { ds } from "./strings";

const GROUPS = ["profitability", "liquidity", "leverage", "efficiency", "cash"] as const;

const axis = (unit: string) => (v: number) =>
  unit === "idr" ? compactIdr(v) : unit === "pct" ? `${formatNumber(v, 0)}%` : formatNumber(v, unit === "ratio" ? 1 : 0);

/** Each ratio over the last 12 months with its healthy threshold, grouped like the metric cards. */
export function RatioTrends({ data }: { data: RatioTrend }) {
  return (
    <div className="flex flex-col gap-6">
      {GROUPS.map((g, gi) => {
        const series = data.series.filter((s) => s.group === g);
        if (!series.length) return null;
        return (
          <section key={g} className="space-y-3">
            <h2 className="text-base font-semibold">
              {ds.ratios.title}: {fs.analysis.groups[g]}
            </h2>
            <div className="grid gap-5 md:grid-cols-2 xl:grid-cols-3">
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
                      <span className="text-sm font-semibold tabular">
                        {last && !last.unavailable ? formatMetric(last.value, s.unit) : ds.ratios.unavailable}
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
