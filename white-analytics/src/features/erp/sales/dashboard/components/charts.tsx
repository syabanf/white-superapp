"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Bar, CartesianGrid, Cell, ComposedChart, Line, Pie, PieChart, ResponsiveContainer, Tooltip as ChartTooltip, XAxis, YAxis } from "recharts";
import {
  AXIS_TICK,
  BAR_CURSOR,
  BAR_RADIUS,
  GRID_DASH,
  GRID_STROKE,
  MUTED_SERIES,
  MarkButtons,
  SeriesLegend,
  TooltipBox,
  barFill,
  ringDot,
  slotColor,
  type SeriesDef,
} from "@/components/dashboard/charts/chart-primitives";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { formatCompact, formatNumber, formatPercent } from "@/lib/format";
import { cn } from "@/lib/utils";
import { monthLabel } from "../format";
import { dd } from "../strings";

export function Empty({ className }: { className?: string }) {
  return <p className={cn("py-10 text-center text-sm text-muted-foreground", className)}>{dd.empty}</p>;
}

export type MonthSeries = SeriesDef & { values: (number | null)[] };

/**
 * Monthly bars and lines on one axis, optional comparison overlay drawn as a
 * muted dashed line behind. Clicking a month calls onMonth with its index.
 */
export function MonthlyChart({
  months,
  series,
  previous,
  stacked,
  yFormat = formatCompact,
  height = 260,
  onMonth,
  yMax,
}: {
  months: string[];
  series: MonthSeries[];
  previous?: { label: string; values: (number | null)[]; format?: (v: number) => string };
  stacked?: boolean;
  yFormat?: (v: number) => string;
  height?: number;
  onMonth?: (index: number) => void;
  yMax?: number;
}) {
  const resolved = series.map((s, i) => ({ ...s, color: s.color ?? slotColor(i), type: s.type ?? "bar" }));
  const data = months.map((m, i) => {
    const row: Record<string, number | string | null> = { month: m };
    for (const s of resolved) row[s.key] = s.values[i] ?? null;
    if (previous) row.__prev = previous.values[i] ?? null;
    return row;
  });
  const seriesMap: Record<string, SeriesDef & { color: string }> = Object.fromEntries(resolved.map((s) => [s.key, s]));
  if (previous)
    seriesMap.__prev = {
      key: "__prev",
      label: previous.label,
      color: MUTED_SERIES,
      format: previous.format ?? resolved[0]?.format,
    };
  const legend = previous ? [...resolved, { key: "__prev", label: previous.label, color: MUTED_SERIES, type: "line" as const }] : resolved;
  const total = resolved.reduce((a, s) => a + s.values.reduce<number>((x, y) => x + (y ?? 0), 0), 0);
  if (!months.length || (total === 0 && !previous?.values.some(Boolean))) return <Empty />;
  const bars = resolved.filter((s) => s.type === "bar");
  return (
    <div>
      <div style={{ height }} className="w-full" role="img" aria-label={`${resolved.map((s) => s.label).join(", ")}: ${months.length} bulan`}>
        <ResponsiveContainer width="100%" height="100%" minWidth={0} initialDimension={{ width: 560, height }}>
          <ComposedChart
            data={data}
            margin={{ top: 8, right: 8, bottom: 0, left: 0 }}
            barGap={2}
            barCategoryGap="22%"
            onClick={(e) => {
              const i = typeof e?.activeTooltipIndex === "number" ? e.activeTooltipIndex : Number(e?.activeTooltipIndex);
              if (onMonth && Number.isFinite(i)) onMonth(i);
            }}
            className={onMonth ? "cursor-pointer" : undefined}
          >
            <CartesianGrid vertical={false} stroke={GRID_STROKE} strokeDasharray={GRID_DASH} />
            <XAxis
              dataKey="month"
              tickLine={false}
              axisLine={false}
              tick={AXIS_TICK}
              tickFormatter={monthLabel}
              interval="preserveStartEnd"
              minTickGap={16}
              tickMargin={8}
            />
            <YAxis tickLine={false} axisLine={false} tick={AXIS_TICK} tickFormatter={(v: number) => yFormat(v)} width={66} domain={[0, yMax ?? "auto"]} allowDecimals={false} />
            <ChartTooltip
              cursor={bars.length ? BAR_CURSOR : { stroke: "var(--axis)" }}
              content={<TooltipBox seriesMap={seriesMap} labelFormatter={(l) => monthLabel(String(l))} />}
              isAnimationActive={false}
            />
            {previous ? <Line dataKey="__prev" type="monotone" stroke={MUTED_SERIES} strokeWidth={2} strokeDasharray="4 3" dot={false} isAnimationActive={false} /> : null}
            {resolved.map((s) =>
              s.type === "bar" ? (
                <Bar
                  key={s.key}
                  dataKey={s.key}
                  fill={barFill(s.color)}
                  stackId={stacked ? "a" : undefined}
                  radius={!stacked || s.key === bars.at(-1)?.key ? BAR_RADIUS : 0}
                  maxBarSize={28}
                  stroke="var(--card)"
                  strokeWidth={stacked ? 1 : 0}
                  isAnimationActive={false}
                />
              ) : (
                <Line
                  key={s.key}
                  dataKey={s.key}
                  type="monotone"
                  stroke={s.color}
                  strokeWidth={2}
                  dot={{ r: 3, strokeWidth: 0, fill: s.color }}
                  activeDot={ringDot(s.color)}
                  isAnimationActive={false}
                />
              ),
            )}
          </ComposedChart>
        </ResponsiveContainer>
      </div>
      <SeriesLegend series={legend} className="mt-2" />
      {onMonth ? <MarkButtons marks={months} label={monthLabel} onSelect={(_m, i) => onMonth(i)} /> : null}
    </div>
  );
}

export type StackRow = {
  key: string;
  label: string;
  href?: string;
  segments: { key: string; value: number }[];
};
export type StackGroup = { key: string; label: string; color: string };

/**
 * Horizontal stacked bars (one row per category), scaled to the largest row
 * total. Each segment has its own tooltip; a row links to its drill-down.
 */
export function StackedBars({
  rows,
  groups,
  format = formatCompact,
  onSelect,
}: {
  rows: StackRow[];
  groups: StackGroup[];
  format?: (v: number) => string;
  /** a segment click passes its group; the row label passes null */
  onSelect?: (row: StackRow, group: string | null) => void;
}) {
  const totals = rows.map((r) => r.segments.reduce((a, s) => a + s.value, 0));
  const max = Math.max(0, ...totals);
  if (!rows.length || max === 0) return <Empty />;
  const present = groups.filter((g) => rows.some((r) => r.segments.some((s) => s.key === g.key && s.value > 0)));
  return (
    <div className="space-y-3">
      <ul className="space-y-3">
        {rows.map((r, i) => {
          const body = (
            <>
              <div className="mb-1 flex items-center justify-between gap-3 text-xs">
                {onSelect ? (
                  <button
                    type="button"
                    onClick={() => onSelect(r, null)}
                    className="min-w-0 cursor-pointer truncate rounded-sm text-left text-foreground/90 underline-offset-4 outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring"
                  >
                    {r.label}
                  </button>
                ) : (
                  <span className="truncate text-foreground/90">{r.label}</span>
                )}
                <span className="shrink-0 font-medium tabular">{format(totals[i]!)}</span>
              </div>
              <div className="flex h-3 w-full gap-[2px] rounded-r-[4px] bg-muted/50" style={{ width: "100%" }}>
                <div className="flex h-3 gap-[2px]" style={{ width: `${(totals[i]! / max) * 100}%` }}>
                  {groups.map((g) => {
                    const v = r.segments.find((s) => s.key === g.key)?.value ?? 0;
                    if (v <= 0) return null;
                    return (
                      <Tooltip key={g.key}>
                        <TooltipTrigger asChild>
                          {onSelect ? (
                            <button
                              type="button"
                              aria-label={`${r.label} · ${g.label}: ${format(v)}`}
                              onClick={() => onSelect(r, g.key)}
                              className="block h-3 min-w-[3px] cursor-pointer outline-none last:rounded-r-[4px] hover:brightness-110 focus-visible:ring-2 focus-visible:ring-ring"
                              style={{ flexGrow: v, flexBasis: 0, background: g.color }}
                            />
                          ) : (
                            <span className="block h-3 min-w-[3px] last:rounded-r-[4px] hover:brightness-110" style={{ flexGrow: v, flexBasis: 0, background: g.color }} />
                          )}
                        </TooltipTrigger>
                        <TooltipContent className="text-xs">
                          {r.label} · {g.label}: <span className="font-semibold tabular">{format(v)}</span>
                        </TooltipContent>
                      </Tooltip>
                    );
                  })}
                </div>
              </div>
            </>
          );
          return (
            <li key={r.key}>
              {r.href ? (
                <Link href={r.href} className="block rounded-sm outline-none focus-visible:ring-2 focus-visible:ring-ring">
                  {body}
                </Link>
              ) : (
                body
              )}
            </li>
          );
        })}
      </ul>
      <SeriesLegend series={present.map((g) => ({ ...g, type: "bar" as const }))} />
    </div>
  );
}

export type FunnelStep = { key: string; label: string; count: number; value: number; href?: string };

/** Funnel with stage-to-stage conversion and drop-off between the bars. */
export function FunnelChart({ steps, valueFormat, onSelect }: { steps: FunnelStep[]; valueFormat: (v: number) => string; onSelect?: (step: FunnelStep) => void }) {
  const first = steps[0]?.count ?? 0;
  if (!first) return <Empty />;
  return (
    <ol className="space-y-1">
      {steps.map((s, i) => {
        const prev = steps[i - 1];
        const conv = prev && prev.count ? (s.count / prev.count) * 100 : null;
        const width = Math.max(2, (s.count / first) * 100);
        const inside = width > 45;
        const bar = (
          <div className="group grid grid-cols-[minmax(0,7.5rem)_minmax(0,1fr)] items-center gap-3 sm:grid-cols-[9rem_minmax(0,1fr)]">
            <span className="truncate text-xs text-foreground/90">{s.label}</span>
            <div className="relative h-8">
              <div
                className="absolute inset-y-0 left-0 rounded-r-[4px] transition-[filter] group-hover:brightness-110"
                style={{ width: `${width}%`, background: slotColor(0), opacity: 1 - i * 0.1 }}
              />
              <span
                className={cn("absolute inset-y-0 flex items-center gap-2 text-xs font-medium tabular", inside ? "left-2 text-white" : "text-foreground")}
                style={inside ? undefined : { left: `calc(${width}% + 8px)` }}
              >
                {formatNumber(s.count)}
                <span className={cn("hidden font-normal sm:inline", inside ? "text-white/85" : "text-muted-foreground")}>{valueFormat(s.value)}</span>
              </span>
            </div>
          </div>
        );
        return (
          <li key={s.key}>
            {conv != null ? (
              <div className="grid grid-cols-[minmax(0,7.5rem)_minmax(0,1fr)] gap-3 py-0.5 text-[11px] text-muted-foreground sm:grid-cols-[9rem_minmax(0,1fr)]">
                <span />
                <span className="tabular">
                  ↓ {formatPercent(conv, 0)} {dd.pipeline.conversion} · {formatNumber(prev!.count - s.count)} {dd.pipeline.dropoff}
                  <span className="hidden sm:inline">
                    {" "}
                    · {formatPercent((s.count / first) * 100, 0)} {dd.pipeline.fromStart}
                  </span>
                </span>
              </div>
            ) : null}
            {onSelect ? (
              <button
                type="button"
                onClick={() => onSelect(s)}
                className="block w-full cursor-pointer rounded-sm text-left outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                {bar}
              </button>
            ) : s.href ? (
              <Link href={s.href} className="block rounded-sm outline-none focus-visible:ring-2 focus-visible:ring-ring">
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

export type DonutItem = { key: string; label: string; value: number; color: string; href?: string };

/** Composition donut with the total in the middle and a value legend. */
export function Donut({ items, format = formatNumber, centerLabel }: { items: DonutItem[]; format?: (v: number) => string; centerLabel: string }) {
  const router = useRouter();
  const total = items.reduce((a, b) => a + b.value, 0);
  if (!total) return <Empty />;
  return (
    <div className="flex flex-col items-center gap-4 sm:flex-row">
      <div className="relative size-40 shrink-0">
        <ResponsiveContainer width="100%" height="100%" initialDimension={{ width: 160, height: 160 }}>
          <PieChart>
            <Pie
              data={items}
              dataKey="value"
              nameKey="label"
              innerRadius="64%"
              outerRadius="100%"
              paddingAngle={2}
              stroke="var(--card)"
              strokeWidth={2}
              isAnimationActive={false}
              onClick={(d: { payload?: DonutItem }) => d.payload?.href && router.push(d.payload.href)}
            >
              {items.map((it) => (
                <Cell key={it.key} fill={it.color} className={it.href ? "cursor-pointer" : undefined} />
              ))}
            </Pie>
            <ChartTooltip
              content={<TooltipBox hideLabel seriesMap={Object.fromEntries(items.map((it) => [it.label, { key: it.label, label: it.label, color: it.color, format }]))} />}
              isAnimationActive={false}
            />
          </PieChart>
        </ResponsiveContainer>
        <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
          <span className="text-lg font-bold tabular">{format(total)}</span>
          <span className="text-[11px] text-muted-foreground">{centerLabel}</span>
        </div>
      </div>
      <ul className="w-full min-w-0 space-y-1.5 text-xs">
        {items.map((it) => {
          const row = (
            <span className="flex items-center gap-2">
              <span aria-hidden className="size-2.5 shrink-0 rounded-[3px]" style={{ background: it.color }} />
              <span className="flex-1 truncate text-muted-foreground">{it.label}</span>
              <span className="font-medium tabular">{format(it.value)}</span>
              <span className="w-10 text-right text-muted-foreground tabular">{formatPercent((it.value / total) * 100, 0)}</span>
            </span>
          );
          return (
            <li key={it.key}>
              {it.href ? (
                <Link href={it.href} className="block rounded-sm hover:bg-muted/60">
                  {row}
                </Link>
              ) : (
                row
              )}
            </li>
          );
        })}
      </ul>
    </div>
  );
}

export type BulletRow = { key: string; label: string; value: number; target: number; href?: string };

/** Bullet bars: actual as the bar, target as a tick, attainment at the right. */
export function BulletList({ rows, format, noTarget, onSelect }: { rows: BulletRow[]; format: (v: number) => string; noTarget: string; onSelect?: (row: BulletRow) => void }) {
  const max = Math.max(0, ...rows.flatMap((r) => [r.value, r.target]));
  if (!rows.length || !max) return <Empty />;
  return (
    <ul className="space-y-4">
      {rows.map((r) => {
        const pct = r.target ? (r.value / r.target) * 100 : null;
        const body = (
          <>
            <div className="mb-1.5 flex items-center justify-between gap-3 text-xs">
              <span className="truncate font-medium">{r.label}</span>
              <span className="shrink-0 text-muted-foreground tabular">
                <span className="font-medium text-foreground">{format(r.value)}</span> / {r.target ? format(r.target) : noTarget}
                {pct != null ? (
                  <span className={cn("ml-2 font-semibold", pct >= 100 ? "text-positive" : "text-negative")}>
                    {pct >= 100 ? "▲ " : "▼ "}
                    {formatPercent(pct, 0)}
                  </span>
                ) : null}
              </span>
            </div>
            <div className="relative h-4 rounded-r-[4px] bg-muted/60">
              <div className="absolute inset-y-1 left-0 rounded-r-[4px]" style={{ width: `${(r.value / max) * 100}%`, background: slotColor(0) }} />
              {r.target ? <div aria-hidden className="absolute -inset-y-0.5 w-0.5 rounded bg-foreground" style={{ left: `calc(${(r.target / max) * 100}% - 1px)` }} /> : null}
            </div>
          </>
        );
        return (
          <li key={r.key}>
            {onSelect ? (
              <button
                type="button"
                onClick={() => onSelect(r)}
                className="block w-full cursor-pointer rounded-sm text-left outline-none hover:bg-muted/30 focus-visible:ring-2 focus-visible:ring-ring"
              >
                {body}
              </button>
            ) : r.href ? (
              <Link href={r.href} className="block rounded-sm hover:bg-muted/30">
                {body}
              </Link>
            ) : (
              body
            )}
          </li>
        );
      })}
    </ul>
  );
}

export type ColumnItem = { key: string; label: string; value: number; color?: string; href?: string };

/** Vertical columns for ordered categories (score buckets, aging buckets). */
export function ColumnChart({
  items,
  format = formatNumber,
  height = 220,
  seriesLabel,
  onSelect,
}: {
  items: ColumnItem[];
  format?: (v: number) => string;
  height?: number;
  seriesLabel: string;
  onSelect?: (item: ColumnItem) => void;
}) {
  const router = useRouter();
  if (!items.some((i) => i.value > 0)) return <Empty />;
  const pick = (it?: ColumnItem) => {
    if (!it) return;
    if (onSelect) onSelect(it);
    else if (it.href) router.push(it.href);
  };
  return (
    <div>
      <div style={{ height }} className="w-full" role="img" aria-label={`${seriesLabel}: ${items.map((i) => `${i.label} ${format(i.value)}`).join(", ")}`}>
        <ResponsiveContainer width="100%" height="100%" minWidth={0} initialDimension={{ width: 400, height }}>
          <ComposedChart data={items} margin={{ top: 8, right: 8, bottom: 0, left: 0 }} barCategoryGap="18%">
            <CartesianGrid vertical={false} stroke={GRID_STROKE} strokeDasharray={GRID_DASH} />
            <XAxis dataKey="label" tickLine={false} axisLine={false} tick={AXIS_TICK} interval={0} tickMargin={8} />
            <YAxis tickLine={false} axisLine={false} tick={AXIS_TICK} tickFormatter={(v: number) => format(v)} width={66} allowDecimals={false} />
            <ChartTooltip
              cursor={BAR_CURSOR}
              content={<TooltipBox seriesMap={{ value: { key: "value", label: seriesLabel, color: slotColor(0), format } }} />}
              isAnimationActive={false}
            />
            <Bar dataKey="value" radius={BAR_RADIUS} maxBarSize={48} isAnimationActive={false} onClick={(d: { payload?: ColumnItem }) => pick(d.payload)}>
              {items.map((it) => (
                <Cell key={it.key} fill={barFill(it.color ?? slotColor(0))} className={it.href || onSelect ? "cursor-pointer" : undefined} />
              ))}
            </Bar>
          </ComposedChart>
        </ResponsiveContainer>
      </div>
      {onSelect ? <MarkButtons marks={items.filter((i) => i.value > 0)} label={(i) => `${i.label}: ${format(i.value)}`} onSelect={(i) => onSelect(i)} /> : null}
    </div>
  );
}
