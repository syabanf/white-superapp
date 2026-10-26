"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  LabelList,
  ComposedChart,
  Line,
  ResponsiveContainer,
  Scatter,
  ScatterChart,
  Tooltip,
  XAxis,
  YAxis,
  ZAxis,
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
} from "@/components/dashboard/charts/chart-primitives";
import { Tooltip as UiTooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { formatCompact, formatNumber, formatPercent } from "@/lib/format";
import { cd } from "@/features/content/strings";
import { waterfallBars, type WaterfallStep } from "@/features/content/dashboard";

type Row = { label: string; href?: string; [k: string]: number | string | null | undefined };

function Empty({ height }: { height: number }) {
  return (
    <div style={{ height }} className="flex items-center justify-center rounded-xl border border-dashed text-sm text-muted-foreground">
      {cd.noData}
    </div>
  );
}

function useGo() {
  const router = useRouter();
  return React.useCallback((href: unknown) => {
    if (typeof href === "string" && href) router.push(href);
  }, [router]);
}

/**
 * Bars, lines and areas over weekly or monthly buckets on one y axis, with an
 * optional comparison-period line in the muted gray. Clicking a bucket opens
 * its `href` (drill-down).
 */
export function PeriodChart({
  data,
  series,
  height = 260,
  yFormat = formatCompact,
  previousKey,
  previousLabel = cd.comparison,
  stacked,
  summary,
}: {
  data: Row[];
  series: SeriesDef[];
  height?: number;
  yFormat?: (v: number) => string;
  previousKey?: string;
  previousLabel?: string;
  stacked?: boolean;
  summary: string;
}) {
  const go = useGo();
  const resolved = series.map((s, i) => ({ ...s, color: s.color ?? slotColor(i), type: s.type ?? "bar" }));
  const map: Record<string, SeriesDef & { color: string }> = Object.fromEntries(resolved.map((s) => [s.key, s]));
  if (previousKey) map[previousKey] = { key: previousKey, label: previousLabel, color: MUTED_SERIES, format: resolved[0]?.format };
  const legend = previousKey ? [...resolved, { key: previousKey, label: previousLabel, color: MUTED_SERIES, type: "line" as const }] : resolved;
  if (data.length === 0) return <Empty height={height} />;
  const bars = resolved.filter((s) => s.type === "bar");
  const lastBar = bars.at(-1)?.key;
  return (
    <div>
      <div style={{ height }} role="img" aria-label={summary} className="w-full">
        <ResponsiveContainer width="100%" height="100%" minWidth={0} initialDimension={{ width: 560, height }}>
          <ComposedChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: 0 }} barGap={2} barCategoryGap="22%">
            <CartesianGrid vertical={false} stroke={GRID_STROKE} strokeDasharray="3 4" />
            <XAxis dataKey="label" tickLine={false} axisLine={{ stroke: "var(--axis)" }} tick={AXIS_TICK} interval="preserveStartEnd" minTickGap={18} tickMargin={8} />
            <YAxis tickLine={false} axisLine={false} tick={AXIS_TICK} tickFormatter={(v: number) => yFormat(v)} width={48} allowDecimals={false} />
            <Tooltip
              cursor={bars.length ? { fill: "var(--muted)", fillOpacity: 0.5 } : { stroke: CURSOR_STROKE }}
              content={<TooltipBox seriesMap={map} />}
              isAnimationActive={false}
              wrapperStyle={{ outline: "none" }}
            />
            {resolved.map((s) =>
              s.type === "bar" ? (
                <Bar
                  key={s.key}
                  dataKey={s.key}
                  fill={s.color}
                  stackId={stacked ? "s" : undefined}
                  radius={!stacked || s.key === lastBar ? [4, 4, 0, 0] : 0}
                  maxBarSize={28}
                  stroke="var(--card)"
                  strokeWidth={stacked ? 2 : 0}
                  isAnimationActive={false}
                  className={data.some((d) => d.href) ? "cursor-pointer" : undefined}
                  onClick={(e: { payload?: Row }) => go(e?.payload?.href)}
                />
              ) : (
                <Line key={s.key} type="monotone" dataKey={s.key} stroke={s.color} strokeWidth={2} dot={data.length < 16 ? { r: 3, strokeWidth: 0, fill: s.color } : false} activeDot={{ r: 4, strokeWidth: 2, stroke: "var(--card)" }} isAnimationActive={false} connectNulls />
              ),
            )}
            {previousKey ? (
              <Line type="monotone" dataKey={previousKey} stroke={MUTED_SERIES} strokeWidth={2} strokeDasharray="4 3" dot={false} isAnimationActive={false} connectNulls />
            ) : null}
          </ComposedChart>
        </ResponsiveContainer>
      </div>
      {legend.length > 1 ? <SeriesLegend series={legend} className="mt-2" /> : null}
    </div>
  );
}

/** Vertical bars for a small categorical distribution (one series, one hue). */
export function CategoryBars({
  data,
  label,
  height = 220,
  color = slotColor(0),
  format = formatNumber,
  summary,
}: {
  data: { label: string; value: number; href?: string }[];
  label: string;
  height?: number;
  color?: string;
  format?: (v: number) => string;
  summary: string;
}) {
  const go = useGo();
  if (data.every((d) => d.value === 0)) return <Empty height={height} />;
  const map = { value: { key: "value", label, color, format } };
  return (
    <div style={{ height }} role="img" aria-label={summary}>
      <ResponsiveContainer width="100%" height="100%" minWidth={0} initialDimension={{ width: 360, height }}>
        <BarChart data={data} margin={{ top: 16, right: 8, bottom: 0, left: 0 }}>
          <CartesianGrid vertical={false} stroke={GRID_STROKE} strokeDasharray="3 4" />
          <XAxis dataKey="label" tickLine={false} axisLine={{ stroke: "var(--axis)" }} tick={AXIS_TICK} tickMargin={8} interval={0} />
          <YAxis tickLine={false} axisLine={false} tick={AXIS_TICK} width={36} allowDecimals={false} />
          <Tooltip cursor={{ fill: "var(--muted)", fillOpacity: 0.5 }} content={<TooltipBox seriesMap={map} />} isAnimationActive={false} />
          <Bar
            dataKey="value"
            fill={color}
            radius={[4, 4, 0, 0]}
            maxBarSize={44}
            isAnimationActive={false}
            label={{ position: "top", fill: "var(--muted-foreground)", fontSize: 11 }}
            className={data.some((d) => d.href) ? "cursor-pointer" : undefined}
            onClick={(e: { payload?: { href?: string } }) => go(e?.payload?.href)}
          />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}

/** Planned (muted) next to published (accent) per category. */
export function GroupedBars({
  data,
  height = 220,
  summary,
}: {
  data: { label: string; planned: number; published: number }[];
  height?: number;
  summary: string;
}) {
  if (data.length === 0) return <Empty height={height} />;
  const series = [
    { key: "planned", label: cd.planned, color: MUTED_SERIES, type: "bar" as const, format: (v: number) => formatNumber(v) },
    { key: "published", label: cd.published, color: slotColor(0), type: "bar" as const, format: (v: number) => formatNumber(v) },
  ];
  const map = Object.fromEntries(series.map((s) => [s.key, s]));
  return (
    <div>
      <div style={{ height }} role="img" aria-label={summary}>
        <ResponsiveContainer width="100%" height="100%" minWidth={0} initialDimension={{ width: 360, height }}>
          <BarChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: 0 }} barGap={2} barCategoryGap="28%">
            <CartesianGrid vertical={false} stroke={GRID_STROKE} strokeDasharray="3 4" />
            <XAxis dataKey="label" tickLine={false} axisLine={{ stroke: "var(--axis)" }} tick={AXIS_TICK} tickMargin={8} interval={0} />
            <YAxis tickLine={false} axisLine={false} tick={AXIS_TICK} width={32} allowDecimals={false} />
            <Tooltip cursor={{ fill: "var(--muted)", fillOpacity: 0.5 }} content={<TooltipBox seriesMap={map} />} isAnimationActive={false} />
            {series.map((s) => (
              <Bar key={s.key} dataKey={s.key} fill={s.color} radius={[4, 4, 0, 0]} maxBarSize={28} isAnimationActive={false} />
            ))}
          </BarChart>
        </ResponsiveContainer>
      </div>
      <SeriesLegend series={series} className="mt-2" />
    </div>
  );
}

/**
 * Bridge from planned to published: totals stand on the baseline, each loss
 * floats down from the running total. One axis, values labelled on the bars.
 */
export function WaterfallChart({ steps, height = 240, summary }: { steps: WaterfallStep[]; height?: number; summary: string }) {
  const data = waterfallBars(steps);
  const map = { value: { key: "value", label: cd.items, color: slotColor(0), format: (v: number) => formatNumber(v) } };
  return (
    <div style={{ height }} role="img" aria-label={summary}>
      <ResponsiveContainer width="100%" height="100%" minWidth={0} initialDimension={{ width: 420, height }}>
        <BarChart data={data} margin={{ top: 18, right: 8, bottom: 0, left: 0 }} barCategoryGap="24%">
          <CartesianGrid vertical={false} stroke={GRID_STROKE} strokeDasharray="3 4" />
          <XAxis dataKey="label" tickLine={false} axisLine={{ stroke: "var(--axis)" }} tick={AXIS_TICK} tickMargin={8} interval={0} />
          <YAxis tickLine={false} axisLine={false} tick={AXIS_TICK} width={32} allowDecimals={false} />
          <Tooltip cursor={{ fill: "var(--muted)", fillOpacity: 0.5 }} content={<TooltipBox seriesMap={map} />} isAnimationActive={false} />
          <Bar dataKey="base" stackId="w" fill="transparent" isAnimationActive={false} />
          <Bar dataKey="value" stackId="w" radius={[4, 4, 4, 4]} maxBarSize={48} isAnimationActive={false}>
            <LabelList dataKey="value" position="top" fill="var(--muted-foreground)" fontSize={11} />
            {data.map((d) => (
              <Cell key={d.label} fill={d.kind === "total" ? slotColor(0) : "var(--status-critical)"} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}

/** Funnel as horizontal bars: share of the first step, conversion from the previous one. */
export function FunnelBars({
  steps,
  label,
}: {
  steps: { key: string; count: number; ofFirst: number; ofPrev: number; href?: string }[];
  label: (key: string) => string;
}) {
  if (!steps.length || steps[0]!.count === 0) return <Empty height={200} />;
  return (
    <ol className="space-y-2">
      {steps.map((s, i) => {
        const body = (
          <div className="grid grid-cols-[6.5rem_minmax(0,1fr)_auto] items-center gap-3 text-xs sm:grid-cols-[8rem_minmax(0,1fr)_auto]">
            <span className="truncate text-foreground/90">{label(s.key)}</span>
            <div className="h-6 rounded-r-[4px] bg-muted/60">
              <div
                className="flex h-6 items-center justify-end rounded-r-[4px] pr-2 text-[11px] font-medium text-white"
                style={{ width: `${Math.max(s.ofFirst, 4)}%`, background: i === steps.length - 1 ? "var(--status-good)" : slotColor(0) }}
              >
                {s.ofFirst >= 18 ? formatNumber(s.count) : null}
              </div>
            </div>
            <span className="w-14 text-right tabular text-muted-foreground">
              {s.ofFirst < 18 ? <span className="mr-1 font-medium text-foreground">{formatNumber(s.count)}</span> : null}
              {i === 0 ? "" : formatPercent(s.ofPrev, 0)}
            </span>
          </div>
        );
        return (
          <li key={s.key}>
            <UiTooltip>
              <TooltipTrigger asChild>
                {s.href ? (
                  <a href={s.href} className="block rounded-md hover:bg-muted/40">
                    {body}
                  </a>
                ) : (
                  <div>{body}</div>
                )}
              </TooltipTrigger>
              <TooltipContent className="text-xs">
                {label(s.key)} · {formatNumber(s.count)} {cd.items.toLowerCase()} · {cd.ofPlanned(formatPercent(s.ofFirst, 0))}
                {i > 0 ? ` · ${cd.fromPrevStep(formatPercent(s.ofPrev, 0))}` : ""}
              </TooltipContent>
            </UiTooltip>
          </li>
        );
      })}
    </ol>
  );
}

export type ScatterPoint = { x: number; y: number; title: string; group: string; href: string };

/** Reach (x) against engagement (y) per post, one hue per group (format). */
export function PostScatter({
  points,
  groups,
  groupLabel,
  height = 280,
  summary,
}: {
  points: ScatterPoint[];
  groups: string[];
  groupLabel: (g: string) => string;
  height?: number;
  summary: string;
}) {
  const go = useGo();
  if (points.length === 0) return <Empty height={height} />;
  const legend = groups.map((g, i) => ({ key: g, label: groupLabel(g), color: slotColor(i), type: "bar" as const }));
  return (
    <div>
      <div style={{ height }} role="img" aria-label={summary}>
        <ResponsiveContainer width="100%" height="100%" minWidth={0} initialDimension={{ width: 520, height }}>
          <ScatterChart margin={{ top: 8, right: 12, bottom: 4, left: 0 }}>
            <CartesianGrid stroke={GRID_STROKE} strokeDasharray="3 4" />
            <XAxis type="number" dataKey="x" name={cd.reach} tickLine={false} axisLine={{ stroke: "var(--axis)" }} tick={AXIS_TICK} tickFormatter={(v: number) => formatCompact(v)} />
            <YAxis type="number" dataKey="y" name={cd.engagement} tickLine={false} axisLine={false} tick={AXIS_TICK} tickFormatter={(v: number) => formatCompact(v)} width={44} />
            <ZAxis range={[64, 64]} />
            <Tooltip
              cursor={{ stroke: CURSOR_STROKE, strokeDasharray: "3 3" }}
              isAnimationActive={false}
              content={({ active, payload }) => {
                const p = active ? (payload?.[0]?.payload as ScatterPoint | undefined) : undefined;
                if (!p) return null;
                return (
                  <div className="max-w-64 rounded-lg border bg-popover px-3 py-2 text-xs shadow-lg">
                    <p className="mb-1 truncate font-medium">{p.title}</p>
                    <p className="text-muted-foreground">
                      {groupLabel(p.group)} · {cd.reach} <span className="font-semibold text-foreground tabular">{formatNumber(p.x)}</span> · {cd.engagement}{" "}
                      <span className="font-semibold text-foreground tabular">{formatNumber(p.y)}</span>
                    </p>
                  </div>
                );
              }}
            />
            {groups.map((g, i) => (
              <Scatter
                key={g}
                data={points.filter((p) => p.group === g)}
                fill={slotColor(i)}
                stroke="var(--card)"
                strokeWidth={2}
                isAnimationActive={false}
                className="cursor-pointer"
                onClick={(e: { payload?: ScatterPoint }) => go(e?.payload?.href)}
              />
            ))}
          </ScatterChart>
        </ResponsiveContainer>
      </div>
      <SeriesLegend series={legend} className="mt-2" />
    </div>
  );
}

const SEQ = ["--seq-100", "--seq-200", "--seq-300", "--seq-400", "--seq-500", "--seq-600", "--seq-700"];

export type CalendarMonth = { key: string; label: string; href?: string; weeks: ({ date: string; planned: number; published: number } | null)[][] };

/**
 * Month calendars as small multiples. Cell shade = the chosen measure
 * (sequential, one hue); days outside the window stay blank.
 */
export function MonthCalendars({ months, weekdays }: { months: CalendarMonth[]; weekdays: string[] }) {
  const [measure, setMeasure] = React.useState<"published" | "planned">("published");
  const max = Math.max(1, ...months.flatMap((m) => m.weeks.flat().map((d) => (d ? d[measure] : 0))));
  if (months.length === 0) return <Empty height={200} />;
  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="inline-flex rounded-full bg-muted p-0.5 text-xs" role="radiogroup" aria-label={cd.calendarMeasure}>
          {(["published", "planned"] as const).map((m) => (
            <button
              key={m}
              type="button"
              role="radio"
              aria-checked={measure === m}
              onClick={() => setMeasure(m)}
              className={cn("rounded-full px-3 py-1", measure === m ? "bg-card font-medium shadow-sm" : "text-muted-foreground")}
            >
              {m === "published" ? cd.published : cd.planned}
            </button>
          ))}
        </div>
        <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
          <span>0</span>
          {SEQ.map((s) => (
            <span key={s} className="h-2 w-4 rounded-[2px]" style={{ background: `var(${s})` }} />
          ))}
          <span>{formatNumber(max)}</span>
        </div>
      </div>
      <div className="grid grid-cols-2 gap-x-4 gap-y-5 md:grid-cols-3 xl:grid-cols-4">
        {months.map((m) => (
          <figure key={m.key} className="min-w-0">
            <figcaption className="mb-1.5 text-xs font-medium">
              {m.href ? (
                <a href={m.href} className="hover:underline">
                  {m.label}
                </a>
              ) : (
                m.label
              )}
            </figcaption>
            <div className="grid grid-cols-7 gap-[3px]">
              {weekdays.map((w) => (
                <span key={w} className="text-center text-[10px] text-muted-foreground">
                  {w}
                </span>
              ))}
              {m.weeks.flat().map((d, i) => {
                if (!d) return <span key={i} className="aspect-square" />;
                const v = d[measure];
                const step = v > 0 ? Math.min(SEQ.length - 1, Math.floor((v / max) * (SEQ.length - 1)) + 1) : -1;
                const bg = step >= 0 ? `var(${SEQ[step]})` : "var(--muted)";
                const dark = step >= 3;
                return (
                  <UiTooltip key={d.date}>
                    <TooltipTrigger asChild>
                      <span
                        role="img"
                        aria-label={`${d.date}: ${d.planned} ${cd.planned.toLowerCase()}, ${d.published} ${cd.published.toLowerCase()}`}
                        className={cn("flex aspect-square items-center justify-center rounded-[4px] text-[10px] tabular", dark ? "text-white" : "text-muted-foreground")}
                        style={{ background: bg }}
                      >
                        {Number(d.date.slice(8))}
                      </span>
                    </TooltipTrigger>
                    <TooltipContent className="text-xs">
                      <span className="font-medium">{d.date}</span> · {cd.planned} {d.planned} · {cd.published} {d.published}
                    </TooltipContent>
                  </UiTooltip>
                );
              })}
            </div>
          </figure>
        ))}
      </div>
    </div>
  );
}
