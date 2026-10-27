"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ArrowDown, ArrowUp, ArrowUpDown } from "lucide-react";
import {
  Area,
  Bar,
  CartesianGrid,
  ComposedChart,
  Line,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { Tooltip as UiTooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { KpiTile } from "@/components/dashboard/kpi-tile";
import {
  AXIS_TICK,
  BAR_CURSOR,
  BAR_RADIUS,
  GRID_DASH,
  GRID_STROKE,
  MUTED_SERIES,
  SeriesLegend,
  TooltipBox,
  areaFill,
  barFill,
  ringDot,
  slotColor,
  type SeriesDef,
} from "@/components/dashboard/charts/chart-primitives";
import { formatCompact, formatNumber } from "@/lib/format";
import { cn } from "@/lib/utils";
import { formatMonth } from "@/features/erp/shared/period-picker";
import { delta, exactIdr, formatUnit, monthShort, num, nums, type KpiUnit } from "./lib";
import type { AnalyticsKpi } from "./types";
import { ds } from "./strings";

export type MonthSeries = SeriesDef & {
  color: string;
  type: "bar" | "line";
  values: number[];
  /** values over the comparison months, drawn as a muted dashed line */
  compare?: number[];
};

/**
 * Month-by-month chart on one axis: bars (grouped, or stacked with signs
 * kept apart) and lines, with an optional comparison overlay per series.
 * Clicking a month calls onMonthClick with its index.
 */
export function MonthlyChart({
  months,
  compareMonths,
  series,
  stacked = false,
  height = 280,
  yFormat = (v: number) => formatCompact(v),
  valueFormat = (v: number) => exactIdr(v),
  onMonthClick,
  referenceY,
  referenceLabel,
}: {
  months: string[];
  compareMonths?: string[];
  series: MonthSeries[];
  stacked?: boolean;
  height?: number;
  yFormat?: (v: number) => string;
  valueFormat?: (v: number) => string;
  onMonthClick?: (index: number) => void;
  referenceY?: number | null;
  referenceLabel?: string;
}) {
  const data = months.map((m, i) => {
    const row: Record<string, string | number | null> = { month: m };
    for (const s of series) {
      row[s.key] = s.values[i] ?? null;
      if (s.compare) row[`${s.key}__cmp`] = s.compare[i] ?? null;
    }
    return row;
  });
  const legend = [
    ...series,
    ...series
      .filter((s) => s.compare)
      .map((s) => ({ key: `${s.key}__cmp`, label: `${s.label} (${ds.previous.toLowerCase()})`, color: MUTED_SERIES, type: "line" as const })),
  ];
  const seriesMap: Record<string, SeriesDef & { color: string }> = Object.fromEntries(
    legend.map((s) => [s.key, { ...s, format: valueFormat }]),
  );
  const empty = series.every((s) => s.values.every((v) => !v));
  if (empty) return <p className="py-10 text-center text-sm text-muted-foreground">{ds.noData}</p>;
  const summary = series.map((s) => s.label).join(", ") + `. ${months.length} bulan.`;
  return (
    <div className="flex flex-col gap-2">
      <div style={{ height }} role="img" aria-label={summary} className="w-full">
        <ResponsiveContainer width="100%" height="100%" minWidth={0} initialDimension={{ width: 560, height }}>
          <ComposedChart
            data={data}
            margin={{ top: 8, right: 8, bottom: 0, left: 0 }}
            stackOffset={stacked ? "sign" : undefined}
            barGap={2}
            onClick={(e) => {
              const idx = typeof e?.activeTooltipIndex === "number" ? e.activeTooltipIndex : Number(e?.activeTooltipIndex);
              if (onMonthClick && Number.isFinite(idx)) onMonthClick(idx);
            }}
            style={onMonthClick ? { cursor: "pointer" } : undefined}
          >
            <CartesianGrid vertical={false} stroke={GRID_STROKE} strokeDasharray={GRID_DASH} />
            <XAxis
              dataKey="month"
              tick={AXIS_TICK}
              tickLine={false}
              axisLine={false}
              tickFormatter={monthShort}
              interval="preserveStartEnd"
              minTickGap={12}
              tickMargin={8}
            />
            <YAxis tick={AXIS_TICK} tickLine={false} axisLine={false} width={58} tickFormatter={(v: number) => yFormat(v)} />
            {stacked || series.some((s) => s.values.some((v) => v < 0)) ? <ReferenceLine y={0} stroke="var(--axis)" /> : null}
            {referenceY != null ? (
              <ReferenceLine
                y={referenceY}
                stroke="var(--foreground)"
                strokeDasharray="5 4"
                strokeOpacity={0.55}
                label={{ value: referenceLabel, position: "insideTopRight", fill: "var(--muted-foreground)", fontSize: 11 }}
              />
            ) : null}
            <Tooltip
              cursor={BAR_CURSOR}
              isAnimationActive={false}
              wrapperStyle={{ outline: "none" }}
              content={(p) => (
                <TooltipBox
                  active={p.active}
                  payload={p.payload as never}
                  label={p.label}
                  seriesMap={seriesMap}
                  labelFormatter={(l) => {
                    const i = months.indexOf(String(l));
                    const cmp = compareMonths?.[i];
                    return cmp && series.some((s) => s.compare) ? `${formatMonth(String(l))} · ${ds.previous} ${formatMonth(cmp)}` : formatMonth(String(l));
                  }}
                />
              )}
            />
            {series.map((s) =>
              s.type === "bar" ? (
                <Bar
                  key={s.key}
                  dataKey={s.key}
                  fill={barFill(s.color)}
                  stackId={stacked ? "a" : undefined}
                  radius={stacked ? 0 : BAR_RADIUS}
                  maxBarSize={stacked ? 28 : 18}
                  stroke="var(--card)"
                  strokeWidth={stacked ? 1 : 0}
                  isAnimationActive={false}
                />
              ) : (
                <Area
                  key={s.key}
                  dataKey={s.key}
                  stroke={s.color}
                  fill={series.length <= 2 && series.every((x) => x.type !== "bar") ? areaFill(s.color) : "none"}
                  strokeWidth={2.5}
                  dot={false}
                  activeDot={ringDot(s.color)}
                  type="monotone"
                  connectNulls
                  isAnimationActive={false}
                />
              ),
            )}
            {series
              .filter((s) => s.compare)
              .map((s) => (
                <Line
                  key={`${s.key}__cmp`}
                  dataKey={`${s.key}__cmp`}
                  stroke={MUTED_SERIES}
                  strokeWidth={2}
                  strokeDasharray="4 3"
                  dot={false}
                  type="monotone"
                  connectNulls
                  isAnimationActive={false}
                />
              ))}
          </ComposedChart>
        </ResponsiveContainer>
      </div>
      <SeriesLegend series={legend} />
    </div>
  );
}

export type StackSegment = { key: string; label: string; value: number; color: string; href?: string };

/**
 * Horizontal 100% bars, one per row, split into segments: composition at
 * a glance (balance sheet sides, aging buckets). Segments are links when
 * they carry an href and show the exact amount on hover.
 */
export function StackRows({
  rows,
  legend,
  format = exactIdr,
}: {
  rows: { key: string; label: string; total: number; segments: StackSegment[] }[];
  legend: { key: string; label: string; color: string }[];
  format?: (v: number) => string;
}) {
  if (rows.every((r) => r.total === 0)) return <p className="py-10 text-center text-sm text-muted-foreground">{ds.noData}</p>;
  return (
    <div className="flex flex-col gap-4">
      {rows.map((r) => (
        <div key={r.key} className="flex flex-col gap-1.5">
          <div className="flex items-baseline justify-between gap-3 text-xs">
            <span className="font-medium">{r.label}</span>
            <span className="font-semibold tabular">{format(r.total)}</span>
          </div>
          <div className="flex h-7 w-full gap-0.5 overflow-hidden rounded-md bg-muted/60">
            {r.segments
              .filter((s) => s.value > 0)
              .map((s) => {
                const pct = r.total > 0 ? (s.value / r.total) * 100 : 0;
                const body = (
                  <span
                    className="block h-full transition-[filter] hover:brightness-110"
                    style={{ width: "100%", background: s.color }}
                    aria-label={`${s.label}: ${format(s.value)}`}
                  />
                );
                return (
                  <UiTooltip key={s.key}>
                    <TooltipTrigger asChild>
                      {s.href ? (
                        <Link href={s.href} className="block h-full" style={{ width: `${pct}%` }}>
                          {body}
                        </Link>
                      ) : (
                        <span className="block h-full" style={{ width: `${pct}%` }}>
                          {body}
                        </span>
                      )}
                    </TooltipTrigger>
                    <TooltipContent side="top" className="text-xs">
                      <span className="font-medium">{s.label}</span> · {format(s.value)} · {formatNumber(pct, 1)}%
                    </TooltipContent>
                  </UiTooltip>
                );
              })}
          </div>
        </div>
      ))}
      <ul className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
        {legend.map((l) => (
          <li key={l.key} className="flex items-center gap-1.5">
            <span aria-hidden className="size-2.5 rounded-[3px]" style={{ background: l.color }} />
            {l.label}
          </li>
        ))}
      </ul>
    </div>
  );
}

/** Ordinal ramp of one hue for ordered buckets (aging): light to dark. */
export function rampColor(i: number, n: number, hue = "var(--chart-2)"): string {
  const pct = Math.round(30 + (70 * i) / Math.max(1, n - 1));
  return `color-mix(in oklab, ${hue} ${pct}%, var(--card))`;
}

export { slotColor };

/**
 * KPI tile fed by an AnalyticsKpi: compact value, exact amount in the hint,
 * delta, previous and sparkline. Percentage KPIs show the change in points.
 */
export function FinanceKpi({
  kpi,
  unit = "idr",
  lowerIsBetter,
  href,
  hint,
  label,
  neutral = false,
}: {
  kpi: AnalyticsKpi | undefined;
  unit?: KpiUnit;
  lowerIsBetter?: boolean;
  href?: string;
  hint?: string;
  label?: string;
  /** neither direction is good or bad (DPO, payables): the change stays muted */
  neutral?: boolean;
}) {
  if (!kpi) return null;
  const cur = num(kpi.value);
  const prev = num(kpi.previous);
  const d = delta(cur, prev);
  const exact = unit === "idr" && cur != null ? ds.exact(exactIdr(cur)) : undefined;
  const hintText = [hint ?? ds.kpiHint[kpi.key], exact, neutral ? ds.drill.neutral : undefined].filter(Boolean).join(" ");
  const shown = unit === "pct" && d ? { ...d, pct: d.abs } : d;
  const previous = kpi.previous !== "" && prev != null ? formatUnit(prev, unit) : undefined;
  return (
    <KpiTile
      label={label ?? ds.kpi[kpi.key] ?? kpi.key}
      value={formatUnit(cur, unit)}
      delta={shown}
      lowerIsBetter={neutral ? "neutral" : lowerIsBetter}
      previous={previous}
      spark={kpi.trend.length > 1 ? nums(kpi.trend) : undefined}
      hint={hintText || undefined}
      href={href}
    />
  );
}

export type Column<T> = {
  key: string;
  label: string;
  align?: "left" | "right";
  render: (row: T) => React.ReactNode;
  sort?: (row: T) => number | string;
  total?: React.ReactNode;
  className?: string;
};

/**
 * Detail table with sortable headers, an optional totals row and an
 * optional row link. Scrolls sideways inside its card on phones.
 */
export function SortTable<T>({
  rows,
  columns,
  rowKey,
  rowHref,
  initialSort,
  limit,
  empty = ds.noData,
}: {
  rows: T[];
  columns: Column<T>[];
  rowKey: (row: T) => string;
  rowHref?: (row: T) => string | undefined;
  initialSort?: { key: string; desc: boolean };
  limit?: number;
  empty?: string;
}) {
  const router = useRouter();
  const [sort, setSort] = React.useState(initialSort);
  const sorted = React.useMemo(() => {
    const col = columns.find((c) => c.key === sort?.key);
    if (!col?.sort || !sort) return rows;
    const pick = col.sort;
    return [...rows].sort((a, b) => {
      const x = pick(a);
      const y = pick(b);
      const cmp = typeof x === "number" && typeof y === "number" ? x - y : String(x).localeCompare(String(y), "id");
      return sort.desc ? -cmp : cmp;
    });
  }, [rows, columns, sort]);
  const shown = limit ? sorted.slice(0, limit) : sorted;
  const hasTotal = columns.some((c) => c.total != null);
  if (rows.length === 0) return <p className="py-8 text-center text-sm text-muted-foreground">{empty}</p>;
  return (
    <div className="card-bleed overflow-x-auto px-(--card-pad) scrollbar-thin">
      <table className="w-full min-w-[560px] text-sm">
        <thead className="text-xs text-muted-foreground">
          <tr className="border-b">
            {columns.map((c) => {
              const active = sort?.key === c.key;
              const Icon = active ? (sort?.desc ? ArrowDown : ArrowUp) : ArrowUpDown;
              return (
                <th
                  key={c.key}
                  scope="col"
                  aria-sort={active ? (sort?.desc ? "descending" : "ascending") : undefined}
                  className={cn("py-2 font-medium whitespace-nowrap", c.align === "right" ? "pl-3 text-right" : "pr-3 text-left")}
                >
                  {c.sort ? (
                    <button
                      type="button"
                      className={cn("inline-flex items-center gap-1 hover:text-foreground", active && "text-foreground")}
                      onClick={() => setSort({ key: c.key, desc: active ? !sort?.desc : true })}
                    >
                      {c.label}
                      <Icon className="size-3" />
                    </button>
                  ) : (
                    c.label
                  )}
                </th>
              );
            })}
          </tr>
        </thead>
        <tbody>
          {shown.map((r) => {
            const href = rowHref?.(r);
            return (
              <tr
                key={rowKey(r)}
                className={cn("border-b border-border/50", href && "cursor-pointer hover:bg-muted/40")}
                onClick={href ? () => router.push(href) : undefined}
              >
                {columns.map((c, i) => (
                  <td
                    key={c.key}
                    className={cn("py-2 tabular", c.align === "right" ? "pl-3 text-right whitespace-nowrap" : "pr-3", c.className)}
                  >
                    {i === 0 && href ? (
                      <Link href={href} className="hover:underline" onClick={(e) => e.stopPropagation()}>
                        {c.render(r)}
                      </Link>
                    ) : (
                      c.render(r)
                    )}
                  </td>
                ))}
              </tr>
            );
          })}
        </tbody>
        {hasTotal ? (
          <tfoot>
            <tr className="font-semibold">
              {columns.map((c, i) => (
                <td key={c.key} className={cn("pt-2.5 tabular", c.align === "right" ? "pl-3 text-right whitespace-nowrap" : "pr-3")}>
                  {i === 0 ? ds.total : c.total}
                </td>
              ))}
            </tr>
          </tfoot>
        ) : null}
      </table>
    </div>
  );
}
