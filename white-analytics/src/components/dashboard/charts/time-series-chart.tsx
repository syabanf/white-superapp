"use client";

import * as React from "react";
import {
  Area,
  Bar,
  CartesianGrid,
  ComposedChart,
  Line,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import {
  AXIS_TICK,
  CURSOR_STROKE,
  GRID_STROKE,
  MUTED_SERIES,
  SeriesLegend,
  TooltipBox,
  slotColor,
  type SeriesDef,
} from "./chart-primitives";
import { formatCompact, formatDateShort } from "@/lib/format";
import { tickEvery } from "@/lib/dates";
import { t } from "@/i18n/id";

export type TimeSeriesPoint = { date: string; [k: string]: number | string | null | undefined };

/**
 * Single-axis daily time series. Lines 2px, areas 10% wash, bars ≤24px with 4px data-end radius,
 * hairline solid grid, crosshair tooltip listing every series, legend for ≥2 series.
 * Optional previous-period overlay in the de-emphasis gray (emphasis form, not a second axis).
 */
export function TimeSeriesChart({
  data,
  series,
  height = 240,
  yFormat = formatCompact,
  previousKey,
  stacked = false,
  className,
  showLegend,
  yDomain,
  reverseY = false,
  summary,
}: {
  data: TimeSeriesPoint[];
  series: SeriesDef[];
  height?: number;
  yFormat?: (v: number) => string;
  /** dataKey holding previous-period values for series[0] (rendered muted, behind) */
  previousKey?: string;
  stacked?: boolean;
  className?: string;
  showLegend?: boolean;
  yDomain?: [number | "auto" | "dataMin" | "dataMax", number | "auto" | "dataMin" | "dataMax"];
  /** for "position" charts where 1 is best */
  reverseY?: boolean;
  /** Concise text alternative announced before the interactive chart. */
  summary?: string;
}) {
  const chartPointId = React.useId();
  const [activeIndex, setActiveIndex] = React.useState(0);
  const safeActiveIndex = Math.min(activeIndex, Math.max(0, data.length - 1));
  const resolved = React.useMemo(
    () => series.map((s, i) => ({ ...s, color: s.color ?? slotColor(i), type: s.type ?? "line" })),
    [series],
  );
  const seriesMap = React.useMemo(() => {
    const m: Record<string, SeriesDef & { color: string }> = {};
    for (const s of resolved) m[s.key] = s;
    if (previousKey)
      m[previousKey] = {
        key: previousKey,
        label: t.common.previousPeriod,
        color: MUTED_SERIES,
        format: resolved[0]?.format,
      };
    return m;
  }, [resolved, previousKey]);
  const legendSeries = previousKey
    ? [
        ...resolved,
        { key: previousKey, label: t.common.previousPeriod, color: MUTED_SERIES, type: "line" as const },
      ]
    : resolved;
  const interval = Math.max(0, tickEvery(data.length) - 1);
  const hasBars = resolved.some((s) => s.type === "bar");
  const barSize = Math.min(24, Math.max(4, Math.floor(600 / Math.max(1, data.length)) - 2));
  const accessibleSummary =
    summary ??
    `${resolved.map((item) => item.label).join(", ")}. ${data.length} titik data${data.length ? `, dari ${formatDateShort(data[0]!.date)} sampai ${formatDateShort(data.at(-1)!.date)}` : ""}.`;
  const activePoint = data[safeActiveIndex];
  const activePointText = activePoint
    ? `${formatDateShort(activePoint.date)}. ${resolved
        .map((item) => {
          const value = activePoint[item.key];
          return `${item.label}: ${typeof value === "number" ? (item.format ?? yFormat)(value) : "tidak ada data"}`;
        })
        .join(". ")}`
    : "Grafik tidak memiliki titik data.";

  return (
    <div className={className}>
      <p className="sr-only">{accessibleSummary}</p>
      <div
        style={{ height }}
        className="w-full rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring"
        role="img"
        aria-label={accessibleSummary}
        tabIndex={0}
      >
        <ResponsiveContainer
          width="100%"
          height="100%"
          minWidth={0}
          initialDimension={{ width: 560, height }}
        >
          <ComposedChart
            data={data}
            margin={{ top: 8, right: 8, bottom: 0, left: 0 }}
            barGap={2}
            barCategoryGap="20%"
          >
            <CartesianGrid vertical={false} stroke={GRID_STROKE} strokeWidth={1} />
            <XAxis
              dataKey="date"
              tickLine={false}
              axisLine={{ stroke: "var(--axis)" }}
              tick={AXIS_TICK}
              tickFormatter={(v: string) => formatDateShort(v)}
              interval="preserveStartEnd"
              minTickGap={interval * 8 + 24}
              tickMargin={8}
            />
            <YAxis
              tickLine={false}
              axisLine={false}
              tick={AXIS_TICK}
              tickFormatter={(v: number) => yFormat(v)}
              width={54}
              domain={yDomain ?? (reverseY ? [1, "dataMax"] : [0, "auto"])}
              reversed={reverseY}
              allowDecimals={false}
            />
            <Tooltip
              cursor={
                hasBars
                  ? { fill: "var(--muted)", fillOpacity: 0.5 }
                  : { stroke: CURSOR_STROKE, strokeWidth: 1 }
              }
              content={<TooltipBox seriesMap={seriesMap} />}
              isAnimationActive={false}
              wrapperStyle={{ outline: "none" }}
            />
            {previousKey ? (
              <Line
                type="monotone"
                dataKey={previousKey}
                stroke={MUTED_SERIES}
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 4, strokeWidth: 2, stroke: "var(--card)" }}
                isAnimationActive={false}
                connectNulls
              />
            ) : null}
            {resolved.map((s, i) => {
              if (s.type === "area") {
                return (
                  <Area
                    key={s.key}
                    type="monotone"
                    dataKey={s.key}
                    stroke={s.color}
                    fill={s.color}
                    fillOpacity={0.1}
                    strokeWidth={2}
                    dot={false}
                    activeDot={{ r: 4, strokeWidth: 2, stroke: "var(--card)" }}
                    stackId={stacked ? "a" : undefined}
                    isAnimationActive={false}
                  />
                );
              }
              if (s.type === "bar") {
                const isTop = stacked ? i === resolved.length - 1 : true;
                return (
                  <Bar
                    key={s.key}
                    dataKey={s.key}
                    fill={s.color}
                    stackId={stacked ? "a" : undefined}
                    radius={isTop ? [4, 4, 0, 0] : 0}
                    maxBarSize={24}
                    barSize={barSize}
                    isAnimationActive={false}
                    stroke="var(--card)"
                    strokeWidth={stacked ? 2 : 0}
                  />
                );
              }
              return (
                <Line
                  key={s.key}
                  type="monotone"
                  dataKey={s.key}
                  stroke={s.color}
                  strokeWidth={2}
                  dot={false}
                  activeDot={{ r: 4, strokeWidth: 2, stroke: "var(--card)" }}
                  isAnimationActive={false}
                  connectNulls
                />
              );
            })}
          </ComposedChart>
        </ResponsiveContainer>
      </div>
      {(showLegend ?? legendSeries.length > 1) ? (
        <SeriesLegend series={legendSeries} className="mt-2" />
      ) : null}
      {data.length > 0 ? (
        <div className="mt-3 rounded-md border bg-muted/20 px-3 py-2.5">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <label htmlFor={chartPointId} className="text-xs font-medium">
              Jelajahi titik data
            </label>
            <span className="text-xs text-muted-foreground">
              {safeActiveIndex + 1} dari {data.length}
            </span>
          </div>
          <input
            id={chartPointId}
            type="range"
            min={0}
            max={Math.max(0, data.length - 1)}
            value={safeActiveIndex}
            onChange={(event) => setActiveIndex(Number(event.currentTarget.value))}
            className="mt-2 h-5 w-full accent-brand"
            aria-label="Pilih titik data. Gunakan tombol panah kiri dan kanan."
            aria-valuetext={activePointText}
          />
          <p className="mt-1.5 text-xs text-muted-foreground" aria-live="polite">
            {activePointText}
          </p>
        </div>
      ) : null}
    </div>
  );
}
