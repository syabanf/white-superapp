"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  Area,
  Bar,
  CartesianGrid,
  ComposedChart,
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
  MarkButtons,
  SeriesLegend,
  TooltipBox,
  areaFill,
  barFill,
  ringDot,
  slotColor,
  type SeriesDef,
} from "@/components/dashboard/charts/chart-primitives";
import { Tooltip as UiTooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { DeltaText } from "@/components/dashboard/delta-badge";
import { cn } from "@/lib/utils";
import { ds } from "./strings";

export const EMPTY = <p className="py-10 text-center text-sm text-muted-foreground">{ds.noData}</p>;

type Row = { [k: string]: number | string | null | undefined };

/**
 * Categorical trend on one y axis: lines, bars (stacked when asked) and a
 * dashed muted line for the comparison period. Clicking a column calls
 * onPick with that row (drill-down).
 */
export function TrendChart({
  data,
  xKey,
  series,
  xFormat,
  yFormat,
  height = 260,
  stacked = false,
  yDomain,
  onPick,
  summary,
}: {
  data: Row[];
  xKey: string;
  series: (SeriesDef & { dashed?: boolean })[];
  xFormat: (v: string) => string;
  yFormat: (v: number) => string;
  height?: number;
  stacked?: boolean;
  yDomain?: [number | "auto" | "dataMin" | "dataMax", number | "auto" | "dataMin" | "dataMax"];
  onPick?: (row: Row) => void;
  summary: string;
}) {
  const resolved = series.map((s, i) => ({ ...s, color: s.color ?? slotColor(i), type: s.type ?? "line" }));
  const map = Object.fromEntries(resolved.map((s) => [s.key, { ...s, format: s.format ?? yFormat }]));
  const bars = resolved.filter((s) => s.type === "bar");
  const lastBar = bars.at(-1)?.key;
  // A wash under the line reads for one or two lines on their own; with bars or more lines it muddies.
  const washLines = bars.length === 0 && resolved.length <= 2;
  if (data.length === 0) return EMPTY;
  return (
    <div>
      <div style={{ height }} className="w-full" role="img" aria-label={summary}>
        <ResponsiveContainer
          width="100%"
          height="100%"
          minWidth={0}
          initialDimension={{ width: 560, height }}
        >
          <ComposedChart
            data={data}
            margin={{ top: 8, right: 8, bottom: 0, left: 0 }}
            barCategoryGap="22%"
            onClick={(state) => {
              const i = Number(state?.activeTooltipIndex);
              if (onPick && Number.isInteger(i) && data[i]) onPick(data[i]);
            }}
            style={onPick ? { cursor: "pointer" } : undefined}
          >
            <CartesianGrid vertical={false} stroke={GRID_STROKE} strokeDasharray={GRID_DASH} />
            <XAxis
              dataKey={xKey}
              tickLine={false}
              axisLine={false}
              tick={AXIS_TICK}
              tickFormatter={(v: string) => xFormat(String(v))}
              interval="preserveStartEnd"
              minTickGap={18}
              tickMargin={8}
            />
            <YAxis
              tickLine={false}
              axisLine={false}
              tick={AXIS_TICK}
              tickFormatter={(v: number) => yFormat(v)}
              width={48}
              domain={yDomain ?? [0, "auto"]}
            />
            <Tooltip
              cursor={
                bars.length
                  ? BAR_CURSOR
                  : { stroke: CURSOR_STROKE, strokeWidth: 1 }
              }
              content={<TooltipBox seriesMap={map} labelFormatter={(l) => xFormat(String(l))} />}
              isAnimationActive={false}
              wrapperStyle={{ outline: "none" }}
            />
            {resolved.map((s) =>
              s.type === "bar" ? (
                <Bar
                  key={s.key}
                  dataKey={s.key}
                  fill={barFill(s.color)}
                  stackId={stacked ? "a" : undefined}
                  radius={!stacked || s.key === lastBar ? BAR_RADIUS : 0}
                  maxBarSize={28}
                  stroke="var(--card)"
                  strokeWidth={stacked ? 1 : 0}
                  isAnimationActive={false}
                />
              ) : (
                <Area
                  key={s.key}
                  type="monotone"
                  dataKey={s.key}
                  stroke={s.color}
                  fill={washLines && !s.dashed ? areaFill(s.color) : "none"}
                  strokeWidth={2.5}
                  strokeDasharray={s.dashed ? "5 4" : undefined}
                  dot={data.length < 16 ? { r: 3, strokeWidth: 0, fill: s.color } : false}
                  activeDot={ringDot(s.color)}
                  connectNulls
                  isAnimationActive={false}
                />
              ),
            )}
          </ComposedChart>
        </ResponsiveContainer>
      </div>
      <SeriesLegend series={resolved} className="mt-2" />
      {onPick ? <MarkButtons marks={data} label={(row) => xFormat(String(row[xKey]))} onSelect={(row) => onPick(row)} /> : null}
    </div>
  );
}

/** TrendChart whose column click navigates to hrefFor(row). */
export function LinkedTrendChart({
  hrefFor,
  ...props
}: Omit<React.ComponentProps<typeof TrendChart>, "onPick"> & { hrefFor: (row: Row) => string }) {
  const router = useRouter();
  return <TrendChart {...props} onPick={(row) => router.push(hrefFor(row))} />;
}

export type Segment = { key: string; label: string; color: string };

/**
 * One horizontal stacked bar per row (composition), 2px surface gaps
 * between segments, total at the right. Rows link to hrefs.
 */
export function StackedRows<R extends { label: string; href?: string; values: Record<string, number>; total?: number }>({
  rows,
  segments,
  format = (v) => v.toLocaleString("id-ID"),
  onSelect,
}: {
  rows: R[];
  segments: Segment[];
  format?: (v: number) => string;
  /** a segment click passes its key; the row label passes null */
  onSelect?: (row: R, segment: string | null) => void;
}) {
  if (rows.length === 0) return EMPTY;
  const max = Math.max(
    1,
    ...rows.map((r) => r.total ?? segments.reduce((a, s) => a + (r.values[s.key] ?? 0), 0)),
  );
  return (
    <div className="space-y-3">
      <ul className="space-y-2.5">
        {rows.map((r) => {
          const total = r.total ?? segments.reduce((a, s) => a + (r.values[s.key] ?? 0), 0);
          const body = (
            <>
              <div className="mb-1 flex items-center justify-between gap-3 text-xs">
                {onSelect ? (
                  <button type="button" onClick={() => onSelect(r, null)} className="min-w-0 cursor-pointer truncate rounded-sm text-left font-medium underline-offset-4 outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring">
                    {r.label}
                  </button>
                ) : (
                  <span className="truncate font-medium">{r.label}</span>
                )}
                <span className="shrink-0 tabular text-muted-foreground">{format(total)}</span>
              </div>
              <div
                className="flex h-2.5 w-full gap-[2px] overflow-hidden rounded-r-[4px]"
                style={{ width: `${(total / max) * 100}%` }}
              >
                {segments.map((s) => {
                  const v = r.values[s.key] ?? 0;
                  if (v <= 0) return null;
                  return (
                    <UiTooltip key={s.key}>
                      <TooltipTrigger asChild>
                        {onSelect ? (
                          <button
                            type="button"
                            aria-label={`${r.label} · ${s.label}: ${format(v)}`}
                            onClick={() => onSelect(r, s.key)}
                            className="h-full min-w-[3px] cursor-pointer outline-none hover:brightness-110 focus-visible:ring-2 focus-visible:ring-ring"
                            style={{ flexGrow: v, flexBasis: 0, background: s.color }}
                          />
                        ) : (
                          <span className="h-full min-w-[3px]" style={{ flexGrow: v, flexBasis: 0, background: s.color }} />
                        )}
                      </TooltipTrigger>
                      <TooltipContent className="text-xs">
                        {r.label} · {s.label}: <span className="font-semibold tabular">{format(v)}</span>
                      </TooltipContent>
                    </UiTooltip>
                  );
                })}
              </div>
            </>
          );
          return (
            <li key={r.label}>
              {r.href ? (
                <Link
                  href={r.href}
                  className="block rounded-md outline-none hover:bg-muted/40 focus-visible:ring-2 focus-visible:ring-ring"
                >
                  {body}
                </Link>
              ) : (
                body
              )}
            </li>
          );
        })}
      </ul>
      <SeriesLegend
        series={segments.map((s) => ({ key: s.key, label: s.label, color: s.color, type: "bar" as const }))}
      />
    </div>
  );
}

/**
 * Funnel: each step's bar relative to the first. Between steps it shows the
 * conversion from the previous step, or the share of the first step when
 * the steps are not strictly nested (relative="first").
 */
export function Funnel({
  steps,
  relative = "previous",
}: {
  steps: { label: string; value: number; href?: string }[];
  relative?: "previous" | "first";
}) {
  const first = steps[0]?.value ?? 0;
  if (!first) return EMPTY;
  return (
    <ol className="space-y-1.5">
      {steps.map((s, i) => {
        const prev = i === 0 ? undefined : relative === "first" ? first : steps[i - 1]?.value;
        const conv = prev ? Math.round((s.value / prev) * 100) : null;
        const bar = (
          <div className="flex items-center gap-3">
            <div className="h-8 flex-1 rounded-r-lg bg-muted/60">
              <div
                className="flex h-8 items-center rounded-r-lg px-2.5 text-xs font-semibold text-white tabular"
                style={{
                  width: `${Math.max(4, (s.value / first) * 100)}%`,
                  background: slotColor(0),
                  opacity: 1 - i * 0.12,
                }}
              >
                {s.value.toLocaleString("id-ID")}
              </div>
            </div>
            <span className="w-28 shrink-0 text-xs text-muted-foreground sm:w-36">{s.label}</span>
          </div>
        );
        return (
          <li key={s.label}>
            {conv != null ? (
              <p className="pb-1 pl-1 text-[11px] text-muted-foreground tabular">↓ {conv}%</p>
            ) : null}
            {s.href ? (
              <Link
                href={s.href}
                className="block rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                {bar}
              </Link>
            ) : (
              bar
            )}
          </li>
        );
      })}
    </ol>
  );
}

/**
 * Bullet rows: attainment bar (0-150%), a tick at the 100% target and a
 * hollow marker for the comparison period.
 */
export function BulletRows<R extends { label: string; value?: number; previous?: number; caption: string }>({
  rows,
  onSelect,
}: {
  rows: R[];
  onSelect?: (row: R) => void;
}) {
  if (rows.length === 0) return EMPTY;
  const MAX = 1.5;
  const pos = (v: number) => `${(Math.min(MAX, Math.max(0, v)) / MAX) * 100}%`;
  return (
    <ul className="space-y-3.5">
      {rows.map((r) => (
        <li key={r.label}>
          <div className="mb-1 flex items-baseline justify-between gap-3 text-xs">
            {onSelect ? (
              <button type="button" onClick={() => onSelect(r)} className="min-w-0 cursor-pointer truncate rounded-sm text-left font-medium underline-offset-4 outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring">
                {r.label}
              </button>
            ) : (
              <span className="truncate font-medium">{r.label}</span>
            )}
            <span className="shrink-0 tabular">
              <span className="font-semibold">{r.value == null ? "–" : `${Math.round(r.value * 100)}%`}</span>
              {r.value != null && r.previous != null ? <DeltaText current={Math.round(r.value * 100)} previous={Math.round(r.previous * 100)} mode="abs" suffix=" poin" className="ml-1.5" /> : null}
              <span className="ml-1.5 text-muted-foreground">{r.caption}</span>
            </span>
          </div>
          <UiTooltip>
            <TooltipTrigger asChild>
              <div className="relative h-3 rounded-r-[4px] bg-muted/70">
                <div
                  className="absolute inset-y-0 left-0 rounded-r-[4px]"
                  style={{
                    width: r.value == null ? 0 : pos(r.value),
                    background: r.value != null && r.value < 0.9 ? "var(--status-warning)" : slotColor(0),
                  }}
                />
                <span
                  aria-hidden
                  className="absolute -inset-y-1 w-0.5 rounded bg-foreground"
                  style={{ left: pos(1) }}
                />
                {r.previous != null ? (
                  <span
                    aria-hidden
                    className="absolute top-1/2 size-2.5 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-[var(--chart-emphasis-muted)] bg-card"
                    style={{ left: pos(r.previous) }}
                  />
                ) : null}
              </div>
            </TooltipTrigger>
            <TooltipContent className="text-xs">
              {r.label}: {r.value == null ? "–" : `${Math.round(r.value * 100)}%`}{" "}
              {ds.kpi.attainment.toLowerCase()}
              {r.previous != null ? ` · ${ds.previous.toLowerCase()} ${Math.round(r.previous * 100)}%` : ""}
            </TooltipContent>
          </UiTooltip>
        </li>
      ))}
    </ul>
  );
}

const SEQ = ["--seq-100", "--seq-200", "--seq-300", "--seq-400", "--seq-500", "--seq-600", "--seq-700"];

/**
 * Value heatmap (rows x columns) on one sequential hue scaled between the
 * smallest and largest value; numbers sit in the cells, cells link.
 */
export function ValueHeatmap({
  rows,
  cols,
  value,
  format,
  hrefFor,
  colLabel,
}: {
  rows: { key: string; label: string }[];
  cols: string[];
  value: (row: string, col: string) => number | undefined;
  format: (v: number) => string;
  hrefFor?: (row: string, col: string) => string;
  colLabel: (c: string) => string;
}) {
  const values = rows.flatMap((r) => cols.map((c) => value(r.key, c))).filter((v): v is number => v != null);
  if (values.length === 0) return EMPTY;
  const min = Math.min(...values);
  const max = Math.max(...values);
  const step = (v: number) =>
    SEQ[Math.min(SEQ.length - 1, Math.floor(((v - min) / Math.max(1e-9, max - min)) * SEQ.length))]!;
  return (
    <div className="overflow-x-auto scrollbar-thin">
      <table className="w-full min-w-[560px] border-separate border-spacing-[2px] text-xs">
        <thead>
          <tr>
            <th className="w-28" />
            {cols.map((c) => (
              <th key={c} className="pb-1 text-center font-normal text-muted-foreground tabular">
                {colLabel(c)}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.key}>
              <th scope="row" className="truncate pr-2 text-left font-normal text-muted-foreground">
                {r.label}
              </th>
              {cols.map((c) => {
                const v = value(r.key, c);
                const dark = v != null && SEQ.indexOf(step(v)) >= 4;
                const cell = (
                  <span
                    className={cn(
                      "flex h-8 min-w-11 items-center justify-center rounded-[4px] px-1 tabular",
                      dark ? "text-white dark:text-neutral-950" : "text-foreground dark:text-white",
                    )}
                    style={{ background: v == null ? "var(--muted)" : `var(${step(v)})` }}
                    aria-label={`${r.label} ${colLabel(c)}: ${v == null ? "–" : format(v)}`}
                  >
                    {v == null ? "" : format(v)}
                  </span>
                );
                return (
                  <td key={c} className="p-0">
                    {hrefFor && v != null ? (
                      <Link
                        href={hrefFor(r.key, c)}
                        className="block rounded-[4px] outline-none hover:ring-2 hover:ring-foreground/40 focus-visible:ring-2 focus-visible:ring-ring"
                      >
                        {cell}
                      </Link>
                    ) : (
                      cell
                    )}
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
      <div className="mt-2 flex items-center gap-1.5 text-xs text-muted-foreground">
        <span className="tabular">{format(min)}</span>
        {SEQ.map((s) => (
          <span key={s} className="h-2 w-4 rounded-[2px]" style={{ background: `var(${s})` }} />
        ))}
        <span className="tabular">{format(max)}</span>
      </div>
    </div>
  );
}

/** Paired bars per category: this period (slot 1) over the comparison (muted). */
export function PairedBars({
  rows,
  max,
  format,
  currentLabel,
  previousLabel,
}: {
  rows: { label: string; value?: number; previous?: number; href?: string }[];
  max: number;
  format: (v: number) => string;
  currentLabel: string;
  previousLabel?: string;
}) {
  if (rows.every((r) => r.value == null)) return EMPTY;
  const w = (v?: number) => `${v == null ? 0 : Math.max(1.5, (v / max) * 100)}%`;
  return (
    <div className="space-y-3">
      <ul className="space-y-3">
        {rows.map((r) => (
          <li key={r.label} className="text-xs">
            <div className="mb-1 flex justify-between gap-3">
              {r.href ? (
                <Link href={r.href} className="truncate font-medium underline-offset-4 hover:underline">
                  {r.label}
                </Link>
              ) : (
                <span className="truncate font-medium">{r.label}</span>
              )}
              <span className="shrink-0 tabular">
                <span className="font-semibold">{r.value == null ? "–" : format(r.value)}</span>
                {r.previous != null ? (
                  <span className="ml-1.5 text-muted-foreground">{format(r.previous)}</span>
                ) : null}
                {r.value != null && r.previous != null ? <DeltaText current={r.value} previous={r.previous} mode="abs" digits={1} className="ml-1.5" /> : null}
              </span>
            </div>
            <div className="space-y-[2px]">
              <div
                className="h-2.5 rounded-r-[4px]"
                style={{ width: w(r.value), background: slotColor(0) }}
              />
              {previousLabel ? (
                <div
                  className="h-1.5 rounded-r-[4px]"
                  style={{ width: w(r.previous), background: MUTED_SERIES }}
                />
              ) : null}
            </div>
          </li>
        ))}
      </ul>
      {previousLabel ? (
        <SeriesLegend
          series={[
            { key: "c", label: currentLabel, color: slotColor(0), type: "bar" },
            { key: "p", label: previousLabel, color: MUTED_SERIES, type: "bar" },
          ]}
        />
      ) : null}
    </div>
  );
}
