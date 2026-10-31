"use client";

import * as React from "react";
import {
  Area,
  Bar,
  CartesianGrid,
  Cell,
  ComposedChart,
  Line,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import {
  AXIS_TICK,
  BAR_CURSOR,
  BAR_RADIUS,
  CURSOR_STROKE,
  GRID_DASH,
  GRID_STROKE,
  MUTED_SERIES,
  barFill,
  ringDot,
  SeriesLegend,
  TooltipBox,
  activeIndexOf,
  signedFill,
  slotColor,
  type SeriesDef,
} from "./chart-primitives";
import { formatCompact, formatDateShort } from "@/lib/format";
import { tickEvery } from "@/lib/dates";
import { t } from "@/i18n/id";

export type TimeSeriesPoint = { date: string; [k: string]: number | string | null | undefined };

/**
 * Single-axis daily time series. Smooth 2.5px lines with a soft wash underneath (up to two
 * lines), gradient bars ≤24px with a fully rounded data end, recessive dashed grid, pill
 * tooltip listing every series, legend for ≥2 series.
 * Optional previous-period overlay in the de-emphasis gray (emphasis form, not a second axis).
 * `onSelect` makes every point/bar clickable (drill sheet or time drill); `onSeriesSelect`
 * turns the legend into filter toggles. Bar series with `signed` color by sign.
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
  onSelect,
  selectLabel = "Lihat rincian",
  onSeriesSelect,
  activeSeries,
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
  /** click on a point/bar (any series) */
  onSelect?: (point: TimeSeriesPoint, index: number) => void;
  /** keyboard button label next to the point explorer */
  selectLabel?: string;
  /** legend click; pair with `activeSeries` to show the active filter */
  onSeriesSelect?: (series: SeriesDef) => void;
  activeSeries?: string | null;
}) {
  const chartPointId = React.useId();
  const gradientId = `area${chartPointId.replace(/[^a-zA-Z0-9]/g, "")}`;
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
  // A wash under each line reads well for one or two lines; more would muddy.
  const washLines = !reverseY && resolved.filter((s) => s.type === "line").length <= 2;
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
            onClick={
              onSelect
                ? (state) => {
                    const idx = activeIndexOf(state);
                    if (idx != null && data[idx]) onSelect(data[idx], idx);
                  }
                : undefined
            }
            style={onSelect ? { cursor: "pointer" } : undefined}
          >
            <defs>
              {resolved.map((s, i) =>
                s.type !== "bar" ? (
                  <linearGradient key={s.key} id={`${gradientId}-${i}`} x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" style={{ stopColor: s.color, stopOpacity: s.type === "area" ? 0.26 : 0.18 }} />
                    <stop offset="100%" style={{ stopColor: s.color, stopOpacity: 0 }} />
                  </linearGradient>
                ) : null,
              )}
            </defs>
            <CartesianGrid vertical={false} stroke={GRID_STROKE} strokeWidth={1} strokeDasharray={GRID_DASH} />
            <XAxis
              dataKey="date"
              tickLine={false}
              axisLine={false}
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
              cursor={hasBars ? BAR_CURSOR : { stroke: CURSOR_STROKE, strokeWidth: 1, strokeDasharray: "4 4" }}
              content={<TooltipBox seriesMap={seriesMap} tone={hasBars ? "light" : "dark"} />}
              isAnimationActive={false}
              wrapperStyle={{ outline: "none" }}
            />
            {previousKey ? (
              <Line
                type="monotone"
                dataKey={previousKey}
                stroke={MUTED_SERIES}
                strokeWidth={2}
                strokeDasharray="5 5"
                dot={false}
                activeDot={ringDot(MUTED_SERIES)}
                connectNulls
              />
            ) : null}
            {resolved.map((s, i) => {
              if (s.type === "area" || (s.type === "line" && washLines)) {
                return (
                  <Area
                    key={s.key}
                    type="monotone"
                    dataKey={s.key}
                    stroke={s.color}
                    fill={`url(#${gradientId}-${i})`}
                    fillOpacity={1}
                    strokeWidth={s.type === "line" ? 2.5 : 2}
                    dot={false}
                    activeDot={ringDot(s.color)}
                    stackId={stacked && s.type === "area" ? "a" : undefined}
                    connectNulls={s.type === "line"}
                  />
                );
              }
              if (s.type === "bar") {
                const isTop = stacked ? i === resolved.length - 1 : true;
                return (
                  <Bar
                    key={s.key}
                    dataKey={s.key}
                    fill={barFill(s.color)}
                    stackId={stacked ? "a" : undefined}
                    radius={isTop ? BAR_RADIUS : 0}
                    maxBarSize={24}
                    barSize={barSize}
                    stroke="var(--card)"
                    strokeWidth={stacked ? 2 : 0}
                  >
                    {s.signed
                      ? data.map((d, j) => (
                          <Cell key={j} fill={barFill(signedFill(Number(d[s.key] ?? 0), s.lowerIsBetter), false, Number(d[s.key] ?? 0) < 0)} />
                        ))
                      : null}
                  </Bar>
                );
              }
              return (
                <Line
                  key={s.key}
                  type="monotone"
                  dataKey={s.key}
                  stroke={s.color}
                  strokeWidth={2.5}
                  dot={false}
                  activeDot={ringDot(s.color)}
                  connectNulls
                />
              );
            })}
          </ComposedChart>
        </ResponsiveContainer>
      </div>
      {(showLegend ?? legendSeries.length > 1) ? (
        <SeriesLegend
          series={legendSeries}
          className="mt-2"
          onSelect={onSeriesSelect ? (item) => item.key !== previousKey && onSeriesSelect(item) : undefined}
          activeKey={activeSeries}
        />
      ) : null}
      {data.length > 0 ? (
        // Keyboard and screen-reader access to every point. Hidden for pointer users,
        // it slides in when focus lands on the slider (Tab from the chart).
        <div className="sr-only focus-within:not-sr-only focus-within:mt-3 focus-within:block focus-within:rounded-2xl focus-within:bg-muted/50 focus-within:px-3 focus-within:py-2.5">
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
          {onSelect && activePoint ? (
            <button
              type="button"
              onClick={() => onSelect(activePoint, safeActiveIndex)}
              className="mt-2 rounded-full border bg-card px-2.5 py-1 text-xs font-medium outline-none hover:bg-muted focus-visible:ring-2 focus-visible:ring-ring"
            >
              {selectLabel}: {formatDateShort(activePoint.date)}
            </button>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
