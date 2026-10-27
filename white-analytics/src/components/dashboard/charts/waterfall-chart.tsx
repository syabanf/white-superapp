"use client";

import * as React from "react";
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { formatCompact } from "@/lib/format";
import {
  AXIS_TICK,
  BAR_CURSOR,
  BAR_RADIUS_ALL,
  GRID_DASH,
  barFill,
  GRID_STROKE,
  MarkButtons,
  signedFill,
} from "./chart-primitives";

export type WaterfallStep = {
  key: string;
  label: string;
  /** signed change, or the running total itself when `total` */
  value: number;
  total?: boolean;
};

/** Fill per step: totals in the brand hue, good steps positive, bad steps negative. */
export function waterfallFill(step: WaterfallStep, lowerIsBetter = false): string {
  return step.total ? "var(--chart-1)" : signedFill(step.value, lowerIsBetter);
}

/** Category label broken onto two lines so neighbouring steps never collide. */
function WrappedTick({
  x,
  y,
  payload,
}: {
  x?: number | string;
  y?: number | string;
  payload?: { value?: string };
}) {
  const words = String(payload?.value ?? "").split(" ");
  const half = Math.ceil(words.length / 2);
  const lines = words.length > 1 ? [words.slice(0, half).join(" "), words.slice(half).join(" ")] : words;
  return (
    <text
      x={Number(x)}
      y={Number(y) + 12}
      textAnchor="middle"
      fill={AXIS_TICK.fill}
      fontSize={AXIS_TICK.fontSize}
    >
      {lines.map((l, i) => (
        <tspan key={i} x={Number(x)} dy={i === 0 ? 0 : 13}>
          {l}
        </tspan>
      ))}
    </text>
  );
}

/** Each step's floating [low, high] range and the running total after it. */
function bridge(steps: WaterfallStep[]) {
  const out: (WaterfallStep & { range: [number, number]; end: number })[] = [];
  for (const s of steps) {
    const before = out.at(-1)?.end ?? 0;
    const start = s.total ? 0 : before;
    const end = s.total ? s.value : before + s.value;
    out.push({ ...s, range: [Math.min(start, end), Math.max(start, end)], end });
  }
  return out;
}

/**
 * Bridge chart: total bars stand on zero, change bars float from the running
 * sum (green = good, red = bad; `lowerIsBetter` flips it for cost bridges).
 * One axis, 2px surface gaps, tooltip per step, optional click per step.
 */
export function WaterfallChart({
  steps,
  format = formatCompact,
  height = 280,
  onSelect,
  onStepClick,
  lowerIsBetter = false,
  emptyLabel = "Belum ada data pada periode ini",
}: {
  steps: WaterfallStep[];
  format?: (v: number) => string;
  height?: number;
  onSelect?: (step: WaterfallStep, index: number) => void;
  /** @deprecated use onSelect */
  onStepClick?: (step: WaterfallStep) => void;
  /** steps are costs: an increase is bad news */
  lowerIsBetter?: boolean;
  emptyLabel?: string;
}) {
  const data = React.useMemo(() => bridge(steps), [steps]);
  const select = onSelect ?? (onStepClick ? (step: WaterfallStep) => onStepClick(step) : undefined);
  if (steps.every((s) => s.value === 0))
    return <p className="py-10 text-center text-sm text-muted-foreground">{emptyLabel}</p>;
  const summary = steps.map((s) => `${s.label} ${format(s.value)}`).join(", ");
  return (
    <div>
      <div style={{ height }} role="img" aria-label={summary} className="w-full">
        <ResponsiveContainer
          width="100%"
          height="100%"
          minWidth={0}
          initialDimension={{ width: 560, height }}
        >
          <BarChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: 0 }} barCategoryGap="18%">
            <CartesianGrid vertical={false} stroke={GRID_STROKE} strokeDasharray={GRID_DASH} />
            <XAxis
              dataKey="label"
              tickLine={false}
              axisLine={false}
              interval={0}
              height={40}
              tick={WrappedTick}
            />
            <YAxis
              tick={AXIS_TICK}
              tickLine={false}
              axisLine={false}
              width={56}
              tickFormatter={(v: number) => format(v)}
            />
            <ReferenceLine y={0} stroke="var(--axis)" />
            <Tooltip
              cursor={BAR_CURSOR}
              isAnimationActive={false}
              content={({ active, payload }) => {
                const p = active ? (payload?.[0]?.payload as (typeof data)[number] | undefined) : undefined;
                if (!p) return null;
                return (
                  <div className="min-w-40 rounded-lg border bg-popover px-3 py-2 text-xs text-popover-foreground shadow-lg">
                    <div className="mb-1 font-medium text-muted-foreground">{p.label}</div>
                    <div className="font-semibold tabular">
                      {p.total ? format(p.value) : `${p.value >= 0 ? "+" : ""}${format(p.value)}`}
                    </div>
                    {!p.total ? (
                      <div className="mt-0.5 text-muted-foreground tabular">→ {format(p.end)}</div>
                    ) : null}
                  </div>
                );
              }}
            />
            <Bar
              dataKey="range"
              radius={BAR_RADIUS_ALL}
              maxBarSize={48}
              stroke="var(--card)"
              strokeWidth={2}
              isAnimationActive={false}
              cursor={select ? "pointer" : undefined}
              onClick={(_entry, idx) => {
                const step = steps[idx];
                if (step) select?.(step, idx);
              }}
            >
              {data.map((d) => (
                <Cell key={d.key} fill={barFill(waterfallFill(d, lowerIsBetter))} />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>
      {select ? <MarkButtons marks={steps} label={(s) => s.label} onSelect={(s, i) => select(s, i)} /> : null}
    </div>
  );
}
