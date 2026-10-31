"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Bar, BarChart, CartesianGrid, Cell, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { formatCompact } from "@/lib/format";
import {
  AXIS_TICK,
  BAR_CURSOR,
  BAR_RADIUS,
  BAR_RADIUS_H,
  GRID_DASH,
  GRID_STROKE,
  barFill,
  MarkButtons,
  SeriesLegend,
  TooltipBox,
  activeIndexOf,
  signedFill,
  slotColor,
  type SeriesDef,
} from "./chart-primitives";

export type CategoryRow = { label: string; href?: string; [key: string]: number | string | undefined };

/**
 * Categorical bars, stacked or grouped, on one axis. `horizontal` lays the
 * categories down the y-axis for long names. Bars carry a gradient from the
 * series color to a lighter tint; stacked segments are split by a 2px surface
 * gap and only the outer end is rounded (fully). A row with `href` opens it
 * on click; `onSelect` receives the row and, when a segment was hit, its
 * series key. `onSeriesSelect` makes the legend a filter toggle.
 */
export function StackedBarChart({
  data,
  series,
  stacked = true,
  horizontal = false,
  height,
  valueFormat = formatCompact,
  labelFormat,
  categoryWidth = 120,
  valueDomain,
  summary,
  className,
  onSelect,
  onSeriesSelect,
  activeSeries,
}: {
  data: CategoryRow[];
  series: SeriesDef[];
  stacked?: boolean;
  horizontal?: boolean;
  height?: number;
  valueFormat?: (v: number) => string;
  /** formats category tick labels (e.g. ISO dates) */
  labelFormat?: (label: string) => string;
  /** y-axis width for horizontal layout */
  categoryWidth?: number;
  /** fixed value scale, e.g. [0, 5] for ratings */
  valueDomain?: [number, number];
  summary?: string;
  className?: string;
  onSelect?: (row: CategoryRow, seriesKey: string | null, index: number) => void;
  onSeriesSelect?: (series: SeriesDef) => void;
  activeSeries?: string | null;
}) {
  const router = useRouter();
  // a segment click bubbles to the chart click; remember it so the row fires once
  const segmentHit = React.useRef(false);
  const resolved = React.useMemo(
    () => series.map((s, i) => ({ ...s, color: s.color ?? slotColor(i), type: "bar" as const })),
    [series],
  );
  const seriesMap = React.useMemo(() => Object.fromEntries(resolved.map((s) => [s.key, s])), [resolved]);
  if (data.length === 0)
    return <p className="py-8 text-center text-sm text-muted-foreground">Belum ada data pada periode ini</p>;
  const h = height ?? (horizontal ? Math.max(160, data.length * 34 + 40) : 260);
  const tickLabel = (v: string) => {
    const s = labelFormat ? labelFormat(v) : v;
    const max = Math.floor(categoryWidth / 7);
    return horizontal && s.length > max ? `${s.slice(0, max - 1)}…` : s;
  };
  const clickable = Boolean(onSelect) || data.some((d) => d.href);
  const select = (idx: number, seriesKey: string | null) => {
    const row = data[idx];
    if (!row) return;
    if (onSelect) onSelect(row, seriesKey, idx);
    else if (row.href) router.push(row.href);
  };
  const valueAxis = {
    type: "number" as const,
    tick: AXIS_TICK,
    tickLine: false,
    axisLine: false,
    tickFormatter: (v: number) => valueFormat(v),
    allowDecimals: false,
    domain: valueDomain ?? [0, "auto"],
  };
  const categoryAxis = {
    type: "category" as const,
    dataKey: "label",
    tick: AXIS_TICK,
    tickLine: false,
    axisLine: false,
    tickFormatter: tickLabel,
    interval: horizontal || data.length <= 12 ? 0 : ("preserveStartEnd" as const),
  };
  return (
    <div className={className}>
      <div
        role="img"
        aria-label={summary ?? resolved.map((s) => s.label).join(", ")}
        style={{ height: h }}
        className="w-full"
      >
        <ResponsiveContainer
          width="100%"
          height="100%"
          minWidth={0}
          initialDimension={{ width: 560, height: h }}
        >
          <BarChart
            data={data}
            layout={horizontal ? "vertical" : "horizontal"}
            margin={{ top: 8, right: 12, bottom: 0, left: 0 }}
            barCategoryGap="24%"
            barGap={2}
            onClick={(state) => {
              if (segmentHit.current) {
                segmentHit.current = false;
                return;
              }
              const idx = activeIndexOf(state);
              if (idx != null) select(idx, null);
            }}
            style={clickable ? { cursor: "pointer" } : undefined}
          >
            <CartesianGrid
              horizontal={!horizontal}
              vertical={horizontal}
              stroke={GRID_STROKE}
              strokeWidth={1}
              strokeDasharray={GRID_DASH}
            />
            {horizontal ? (
              <>
                <XAxis {...valueAxis} />
                <YAxis
                  {...categoryAxis}
                  width={categoryWidth}
                  // one line per category: recharts would wrap long names
                  tick={({ x, y, payload }) => (
                    <text
                      x={x}
                      y={y}
                      dy={4}
                      textAnchor="end"
                      fill={AXIS_TICK.fill}
                      fontSize={AXIS_TICK.fontSize}
                    >
                      {tickLabel(String(payload?.value ?? ""))}
                    </text>
                  )}
                />
              </>
            ) : (
              <>
                <XAxis {...categoryAxis} tickMargin={8} minTickGap={8} />
                <YAxis {...valueAxis} width={48} />
              </>
            )}
            <Tooltip
              cursor={BAR_CURSOR}
              content={
                <TooltipBox
                  seriesMap={seriesMap}
                  labelFormatter={(l) => (labelFormat ? labelFormat(String(l)) : String(l))}
                />
              }
              isAnimationActive={false}
              wrapperStyle={{ outline: "none" }}
            />
            {resolved.map((s, i) => {
              const outer = !stacked || i === resolved.length - 1;
              const radius: [number, number, number, number] = outer
                ? horizontal
                  ? BAR_RADIUS_H
                  : BAR_RADIUS
                : [0, 0, 0, 0];
              return (
                <Bar
                  key={s.key}
                  dataKey={s.key}
                  name={s.label}
                  fill={barFill(s.color, horizontal)}
                  stackId={stacked ? "a" : undefined}
                  radius={radius}
                  maxBarSize={24}
                  stroke="var(--card)"
                  strokeWidth={stacked ? 2 : 0}
                  onClick={
                    onSelect
                      ? (_entry, idx) => {
                          segmentHit.current = true;
                          setTimeout(() => (segmentHit.current = false), 0);
                          select(idx, s.key);
                        }
                      : undefined
                  }
                >
                  {s.signed && !stacked
                    ? data.map((d, j) => (
                        <Cell key={j} fill={barFill(signedFill(Number(d[s.key] ?? 0), s.lowerIsBetter), horizontal, Number(d[s.key] ?? 0) < 0)} />
                      ))
                    : null}
                </Bar>
              );
            })}
          </BarChart>
        </ResponsiveContainer>
      </div>
      <SeriesLegend series={resolved} className="mt-2" onSelect={onSeriesSelect} activeKey={activeSeries} />
      {clickable ? (
        <MarkButtons
          marks={data}
          label={(row) => (labelFormat ? labelFormat(row.label) : row.label)}
          onSelect={(_row, idx) => select(idx, null)}
        />
      ) : null}
    </div>
  );
}
