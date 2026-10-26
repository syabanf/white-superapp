"use client";

import * as React from "react";
import {
  Area,
  Bar,
  CartesianGrid,
  Cell,
  ComposedChart,
  Line,
  Pie,
  PieChart,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { Tooltip as Tip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  AXIS_TICK,
  CURSOR_STROKE,
  GRID_STROKE,
  MUTED_SERIES,
  SeriesLegend,
  TooltipBox,
  slotColor,
  type SeriesDef,
} from "@/components/dashboard/charts/chart-primitives";
import { cn } from "@/lib/utils";
import { formatCompact, formatPercent } from "@/lib/format";
import { thd } from "@/features/erp/hr/dashboard/strings";

export type ChartRow = { label: string; tip?: string; [k: string]: number | string | null | undefined };

export function ChartEmpty({ height = 220, children }: { height?: number; children?: React.ReactNode }) {
  return (
    <div className="flex items-center justify-center rounded-xl bg-muted/40 text-sm text-muted-foreground" style={{ height }}>
      {children ?? thd.empty}
    </div>
  );
}

/**
 * Category chart on one y-axis: bars (grouped or stacked), lines and areas.
 * Each row is one x category (`label`, optional longer `tip` for the tooltip).
 * `onSelect` makes every bar and point a drill-down target.
 */
export function ComboChart({
  data,
  series,
  height = 240,
  stacked = false,
  yFormat = formatCompact,
  yDomain,
  onSelect,
  reference,
  summary,
  legend = true,
  className,
}: {
  data: ChartRow[];
  series: SeriesDef[];
  height?: number;
  stacked?: boolean;
  yFormat?: (v: number) => string;
  /** "percent" fits a 0-100 scale to the data and stops at 100 */
  yDomain?: [number | "auto" | "dataMin" | "dataMax", number | "auto" | "dataMin" | "dataMax"] | "percent";
  onSelect?: (row: ChartRow, index: number) => void;
  /** vertical marker on a category label, e.g. the late tolerance */
  reference?: { x: string; label: string };
  summary: string;
  legend?: boolean;
  className?: string;
}) {
  const resolved = React.useMemo(
    () => series.map((s, i) => ({ ...s, color: s.color ?? slotColor(i), type: s.type ?? "bar" })),
    [series],
  );
  const map = React.useMemo(() => Object.fromEntries(resolved.map((s) => [s.key, s])), [resolved]);
  const hasBars = resolved.some((s) => s.type === "bar");
  const bars = resolved.filter((s) => s.type === "bar");
  const topBar = bars.at(-1)?.key;
  if (data.length === 0) return <ChartEmpty height={height} />;
  const domain = yDomain === "percent" ? percentDomain(data, resolved.map((s) => s.key)) : (yDomain ?? [0, "auto"]);
  const select = onSelect
    ? (_: unknown, index: number) => {
        const row = data[index];
        if (row) onSelect(row, index);
      }
    : undefined;
  return (
    <div className={className}>
      <p className="sr-only">{summary}</p>
      <div style={{ height }} role="img" aria-label={summary} className={cn("w-full", onSelect && "[&_.recharts-bar-rectangle]:cursor-pointer")}>
        <ResponsiveContainer width="100%" height="100%" minWidth={0} initialDimension={{ width: 520, height }}>
          <ComposedChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: 0 }} barGap={2} barCategoryGap="22%">
            <CartesianGrid vertical={false} stroke={GRID_STROKE} strokeDasharray="3 4" />
            <XAxis
              dataKey="label"
              tickLine={false}
              axisLine={{ stroke: "var(--axis)" }}
              tick={AXIS_TICK}
              interval={data.length <= 8 ? 0 : "preserveStartEnd"}
              minTickGap={14}
              tickMargin={8}
            />
            <YAxis
              tickLine={false}
              axisLine={false}
              tick={AXIS_TICK}
              tickFormatter={(v: number) => yFormat(v)}
              width={58}
              domain={domain}
              allowDecimals={false}
            />
            <Tooltip
              cursor={hasBars ? { fill: "var(--muted)", fillOpacity: 0.5 } : { stroke: CURSOR_STROKE, strokeWidth: 1 }}
              content={
                <TooltipBox
                  seriesMap={map}
                  labelFormatter={(label) => data.find((r) => r.label === label)?.tip ?? String(label ?? "")}
                />
              }
              isAnimationActive={false}
              wrapperStyle={{ outline: "none" }}
            />
            {reference ? (
              <ReferenceLine
                x={reference.x}
                stroke="var(--axis)"
                strokeDasharray="4 3"
                label={{ value: reference.label, position: "insideTopRight", fill: "var(--muted-foreground)", fontSize: 11 }}
              />
            ) : null}
            {resolved.map((s) => {
              if (s.type === "line" || s.type === "area") {
                const common = {
                  dataKey: s.key,
                  stroke: s.color,
                  strokeWidth: 2,
                  dot: data.length <= 2 ? { r: 3, fill: s.color } : false,
                  activeDot: { r: 4, strokeWidth: 2, stroke: "var(--card)" },
                  isAnimationActive: false,
                  connectNulls: true,
                } as const;
                return s.type === "area" ? (
                  <Area key={s.key} type="monotone" {...common} fill={s.color} fillOpacity={0.1} />
                ) : (
                  <Line key={s.key} type="monotone" {...common} strokeDasharray={s.color === MUTED_SERIES ? "5 4" : undefined} />
                );
              }
              const isTop = !stacked || s.key === topBar;
              return (
                <Bar
                  key={s.key}
                  dataKey={s.key}
                  fill={s.color}
                  stackId={stacked ? "s" : undefined}
                  radius={isTop ? [4, 4, 0, 0] : 0}
                  maxBarSize={28}
                  stroke="var(--card)"
                  strokeWidth={stacked ? 1 : 0}
                  isAnimationActive={false}
                  onClick={select}
                />
              );
            })}
          </ComposedChart>
        </ResponsiveContainer>
      </div>
      {legend ? <SeriesLegend series={resolved} className="mt-2" /> : null}
    </div>
  );
}

function percentDomain(data: ChartRow[], keys: string[]): [number, number] {
  let min = 100;
  for (const r of data) for (const k of keys) if (typeof r[k] === "number") min = Math.min(min, r[k] as number);
  return [Math.max(0, Math.floor((min - 2) / 5) * 5), 100];
}

export type DonutSlice = { key: string; label: string; value: number; color?: string };

/**
 * Composition donut: at most six slices (the rest fold into "Lainnya"),
 * a 2px surface gap between slices, total in the middle, legend with share.
 */
export function Donut({
  slices,
  format = (v: number) => formatCompact(v),
  centerLabel,
  onSelect,
  height = 180,
}: {
  slices: DonutSlice[];
  format?: (v: number) => string;
  centerLabel?: string;
  onSelect?: (slice: DonutSlice) => void;
  height?: number;
}) {
  const sorted = [...slices].filter((s) => s.value > 0).sort((a, b) => b.value - a.value);
  const shown = sorted.length > 6 ? sorted.slice(0, 5) : sorted;
  const rest = sorted.length > 6 ? sorted.slice(5).reduce((a, s) => a + s.value, 0) : 0;
  const items: DonutSlice[] = [
    ...shown.map((s, i) => ({ ...s, color: s.color ?? slotColor(i) })),
    ...(rest > 0 ? [{ key: "__other", label: thd.others, value: rest, color: MUTED_SERIES }] : []),
  ];
  const total = items.reduce((a, s) => a + s.value, 0);
  if (total === 0) return <ChartEmpty height={height} />;
  return (
    <div className="flex flex-col items-center gap-4">
      <div className="relative shrink-0" style={{ width: height, height }} role="img" aria-label={items.map((s) => `${s.label} ${format(s.value)}`).join(", ")}>
        <ResponsiveContainer width="100%" height="100%" initialDimension={{ width: height, height }}>
          <PieChart>
            <Pie
              data={items}
              dataKey="value"
              nameKey="label"
              innerRadius="62%"
              outerRadius="100%"
              paddingAngle={0}
              stroke="var(--card)"
              strokeWidth={2}
              isAnimationActive={false}
              onClick={onSelect ? (_: unknown, i: number) => items[i] && items[i].key !== "__other" && onSelect(items[i]) : undefined}
            >
              {items.map((s) => (
                <Cell key={s.key} fill={s.color} className={onSelect && s.key !== "__other" ? "cursor-pointer" : undefined} />
              ))}
            </Pie>
            <Tooltip
              content={({ active, payload }) =>
                active && payload?.[0] ? (
                  <div className="max-w-64 rounded-lg border bg-popover px-3 py-2 text-xs shadow-lg">
                    <span className="font-medium">{String(payload[0].name)}</span>
                    <span className="ml-2 font-semibold tabular">{format(Number(payload[0].value))}</span>
                    <span className="ml-1.5 text-muted-foreground">{formatPercent((Number(payload[0].value) / total) * 100, 1)}</span>
                  </div>
                ) : null
              }
              isAnimationActive={false}
            />
          </PieChart>
        </ResponsiveContainer>
        <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
          <span className="text-xl font-bold tabular">{format(total)}</span>
          {centerLabel ? <span className="text-[11px] text-muted-foreground">{centerLabel}</span> : null}
        </div>
      </div>
      <ul className="w-full min-w-0 space-y-1.5 text-xs">
        {items.map((s) => (
          <li key={s.key}>
            <button
              type="button"
              disabled={!onSelect || s.key === "__other"}
              onClick={() => onSelect?.(s)}
              className="flex w-full items-center gap-2 rounded-md px-1 py-0.5 text-left enabled:hover:bg-muted/60"
            >
              <span aria-hidden className="size-2.5 shrink-0 rounded-[3px]" style={{ background: s.color }} />
              <span className="min-w-0 flex-1 truncate text-foreground/90">{s.label}</span>
              <span className="shrink-0 font-medium tabular">{format(s.value)}</span>
              <span className="w-12 shrink-0 text-right text-muted-foreground tabular">{formatPercent((s.value / total) * 100, 0)}</span>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}

const SEQ = ["--seq-100", "--seq-200", "--seq-300", "--seq-400", "--seq-500", "--seq-600", "--seq-700"];

/**
 * Sequential heatmap with flexible columns: each cell shows value / total
 * as a share (one hue, light to dark), with a tooltip and optional click.
 */
export function HeatGrid({
  rows,
  cols,
  cell,
  onSelect,
  valueLabel,
}: {
  rows: { key: string; label: string }[];
  cols: { key: number; label: string }[];
  cell: (row: string, col: number) => { value: number; total: number } | undefined;
  onSelect?: (row: string, col: number) => void;
  valueLabel: string;
}) {
  // Scale between the lowest and highest share so close values still separate.
  let max = 0;
  let min = Infinity;
  for (const r of rows)
    for (const c of cols) {
      const v = cell(r.key, c.key);
      if (v && v.total > 0) {
        max = Math.max(max, v.value / v.total);
        min = Math.min(min, v.value / v.total);
      }
    }
  const span = max - min;
  if (rows.length === 0) return <ChartEmpty height={160} />;
  return (
    <div className="overflow-x-auto scrollbar-thin">
      <div
        className="grid min-w-[420px] gap-[3px]"
        style={{ gridTemplateColumns: `minmax(88px, 1.4fr) repeat(${cols.length}, minmax(28px, 1fr))` }}
      >
        <div />
        {cols.map((c) => (
          <div key={c.key} className="pb-1 text-center text-[11px] text-muted-foreground">
            {c.label}
          </div>
        ))}
        {rows.map((r) => (
          <React.Fragment key={r.key}>
            <div className="truncate pr-2 text-xs leading-7 text-muted-foreground" title={r.label}>
              {r.label}
            </div>
            {cols.map((c) => {
              const v = cell(r.key, c.key);
              const share = v && v.total > 0 ? v.value / v.total : 0;
              const pos = span > 0 ? (share - min) / span : 1;
              const step = v && v.total > 0 ? SEQ[Math.min(SEQ.length - 1, Math.floor(pos * (SEQ.length - 0.01)))] : null;
              const text = `${r.label} · ${c.label}: ${formatPercent(share * 100, 1)} (${v?.value ?? 0} dari ${v?.total ?? 0})`;
              return (
                <Tip key={c.key}>
                  <TooltipTrigger asChild>
                    <button
                      type="button"
                      aria-label={text}
                      disabled={!onSelect}
                      onClick={() => onSelect?.(r.key, c.key)}
                      className="h-7 rounded-[4px] transition-transform enabled:hover:scale-105 enabled:hover:ring-2 enabled:hover:ring-foreground/30"
                      style={{ background: step ? `var(${step})` : "var(--muted)" }}
                    />
                  </TooltipTrigger>
                  <TooltipContent className="text-xs">
                    <div className="font-medium">
                      {r.label} · {c.label}
                    </div>
                    <div>
                      {valueLabel}: <span className="font-semibold tabular">{formatPercent(share * 100, 1)}</span>
                    </div>
                    <div className="tabular">
                      {v?.value ?? 0} dari {v?.total ?? 0}
                    </div>
                  </TooltipContent>
                </Tip>
              );
            })}
          </React.Fragment>
        ))}
      </div>
      <div className="mt-3 flex items-center gap-1.5 text-xs text-muted-foreground">
        <span className="tabular">{formatPercent(min === Infinity ? 0 : min * 100, 0)}</span>
        {SEQ.map((s) => (
          <span key={s} className="h-2 w-4 rounded-[2px]" style={{ background: `var(${s})` }} />
        ))}
        <span className="tabular">{formatPercent(max * 100, 0)}</span>
      </div>
    </div>
  );
}

export type WaterfallStep = { label: string; value: number; kind: "total" | "minus" | "plus" };

/** Bar segments for a bridge: totals reset the running sum, steps float on it. */
function bridgeRows(steps: WaterfallStep[]) {
  const rows = [];
  let running = 0;
  for (const s of steps) {
    if (s.kind === "total") {
      running = s.value;
      rows.push({ label: s.label, base: 0, total: s.value, minus: 0, plus: 0, raw: s.value });
    } else if (s.kind === "minus") {
      running -= s.value;
      rows.push({ label: s.label, base: running, total: 0, minus: s.value, plus: 0, raw: -s.value });
    } else {
      rows.push({ label: s.label, base: running, total: 0, minus: 0, plus: s.value, raw: s.value });
      running += s.value;
    }
  }
  return rows;
}

/** Bridge chart: totals stand on the baseline, steps float from the running sum. */
export function Waterfall({
  steps,
  format,
  height = 260,
  summary,
}: {
  steps: WaterfallStep[];
  format: (v: number) => string;
  height?: number;
  summary: string;
}) {
  const data = bridgeRows(steps);
  if (steps.every((s) => s.value === 0)) return <ChartEmpty height={height} />;
  const series = [
    { key: "total", label: thd.series.subtotal, color: slotColor(0) },
    { key: "minus", label: thd.series.decrease, color: slotColor(1) },
    { key: "plus", label: thd.series.increase, color: slotColor(2) },
  ];
  return (
    <div>
      <p className="sr-only">{summary}</p>
      <div style={{ height }} role="img" aria-label={summary}>
        <ResponsiveContainer width="100%" height="100%" minWidth={0} initialDimension={{ width: 520, height }}>
          <ComposedChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: 0 }} barCategoryGap="18%">
            <CartesianGrid vertical={false} stroke={GRID_STROKE} strokeDasharray="3 4" />
            <XAxis dataKey="label" tickLine={false} axisLine={{ stroke: "var(--axis)" }} tick={AXIS_TICK} interval={0} tickMargin={8} />
            <YAxis tickLine={false} axisLine={false} tick={AXIS_TICK} tickFormatter={(v: number) => format(v)} width={66} />
            <Tooltip
              cursor={{ fill: "var(--muted)", fillOpacity: 0.5 }}
              content={({ active, payload, label }) => {
                const row = active ? (payload?.[0]?.payload as { raw: number } | undefined) : undefined;
                return row ? (
                  <div className="max-w-64 rounded-lg border bg-popover px-3 py-2 text-xs shadow-lg">
                    <div className="mb-1 font-medium text-muted-foreground">{String(label)}</div>
                    <div className="font-semibold tabular">{(row.raw < 0 ? "-" : "") + format(Math.abs(row.raw))}</div>
                  </div>
                ) : null;
              }}
              isAnimationActive={false}
            />
            <Bar dataKey="base" stackId="w" fill="transparent" tooltipType="none" isAnimationActive={false} />
            <Bar dataKey="total" stackId="w" fill={series[0]!.color} radius={[4, 4, 0, 0]} maxBarSize={44} isAnimationActive={false} />
            <Bar dataKey="minus" stackId="w" fill={series[1]!.color} radius={[4, 4, 0, 0]} maxBarSize={44} isAnimationActive={false} />
            <Bar dataKey="plus" stackId="w" fill={series[2]!.color} radius={[4, 4, 0, 0]} maxBarSize={44} isAnimationActive={false} />
          </ComposedChart>
        </ResponsiveContainer>
      </div>
      <SeriesLegend
        series={series.filter((x) => steps.some((st) => st.kind === x.key)).map((x) => ({ ...x, type: "bar" as const }))}
        className="mt-2"
      />
    </div>
  );
}

/** Progress bars against a quota, one per row (used vs quota). */
export function BulletList({
  items,
  format,
}: {
  items: { key: string; label: string; value: number; target: number; href?: string }[];
  format: (value: number, target: number) => string;
}) {
  if (items.length === 0) return <ChartEmpty height={160} />;
  return (
    <ul className="space-y-3">
      {items.map((it) => {
        const share = it.target > 0 ? Math.min(1, it.value / it.target) : 0;
        const body = (
          <>
            <div className="mb-1 flex items-center justify-between gap-3 text-xs">
              <span className="truncate text-foreground/90">{it.label}</span>
              <span className="shrink-0 tabular text-muted-foreground">
                <span className="font-medium text-foreground">{format(it.value, it.target)}</span>
                <span className="ml-1.5">{formatPercent(share * 100, 0)}</span>
              </span>
            </div>
            <div className="h-2.5 w-full rounded-full bg-muted/80">
              <div className="h-2.5 rounded-full" style={{ width: `${share * 100}%`, background: slotColor(0) }} />
            </div>
          </>
        );
        return (
          <li key={it.key}>
            {it.href ? (
              <a href={it.href} className="block rounded-md hover:bg-muted/40">
                {body}
              </a>
            ) : (
              body
            )}
          </li>
        );
      })}
    </ul>
  );
}

/** Min to max range per row with the average marked, on one shared scale. */
export function RangeBars({
  items,
  format,
}: {
  items: { key: string; label: string; min: number; max: number; avg: number; count: number }[];
  format: (v: number) => string;
}) {
  if (items.length === 0) return <ChartEmpty height={160} />;
  const top = Math.max(...items.map((i) => i.max)) || 1;
  return (
    <ul className="space-y-4">
      {items.map((it) => (
        <li key={it.key} className="text-xs">
          <div className="mb-1.5 flex items-center justify-between gap-3">
            <span className="truncate text-foreground/90">
              {it.label} <span className="text-muted-foreground">· {it.count} orang</span>
            </span>
            <span className="shrink-0 tabular">
              <span className="text-muted-foreground">{thd.series.average} </span>
              <span className="font-semibold">{format(it.avg)}</span>
            </span>
          </div>
          <Tip>
            <TooltipTrigger asChild>
              <div className="relative h-3 w-full rounded-full bg-muted/70">
                <div
                  className="absolute top-0 h-3 rounded-full"
                  style={{ left: `${(it.min / top) * 100}%`, width: `${Math.max(1, ((it.max - it.min) / top) * 100)}%`, background: "var(--seq-200)" }}
                />
                <div className="absolute -top-0.5 h-4 w-1 rounded-full" style={{ left: `calc(${(it.avg / top) * 100}% - 2px)`, background: slotColor(0) }} />
              </div>
            </TooltipTrigger>
            <TooltipContent className="text-xs tabular">
              {format(it.min)} – {format(it.max)} · {thd.series.average} {format(it.avg)}
            </TooltipContent>
          </Tip>
        </li>
      ))}
    </ul>
  );
}
