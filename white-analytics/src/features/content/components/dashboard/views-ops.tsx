"use client";

import * as React from "react";
import { ChartCard } from "@/components/dashboard/chart-card";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { Heatmap } from "@/components/dashboard/charts/heatmap";
import { slotColor } from "@/components/dashboard/charts/chart-primitives";
import { KpiGrid, KpiTile } from "@/components/dashboard/kpi-tile";
import { Section } from "@/components/dashboard/page-header";
import { monthGrid, monthLabel, weekdayLabel } from "@/features/publishing/time";
import { funnelSteps, publishRate, type DashBucket, type DashPerson } from "@/features/content/dashboard";
import { num } from "@/features/content/lib";
import { c, cd } from "@/features/content/strings";
import type { ItemStatus } from "@/features/content/types";
import type { BoardStage } from "@/features/content/board-filters";
import { formatCompact, formatDecimal, formatNumber, formatPercent } from "@/lib/format";
import { CategoryBars, FunnelBars, MonthCalendars, PeriodChart, type CalendarMonth } from "./charts";
import { useContentDrill } from "./drill";
import { contentBoardHref, periodRange } from "./links";
import { bucketRange } from "@/features/erp/shared/drill/time";
import {
  DetailTable,
  TextLink,
  TextButton,
  bucketLabel,
  overviewHref,
  bucketRangeLabel,
  personName,
  planForMonth,
  spark,
  useDelta,
  type TabProps,
} from "./shared";

const days = (v: number) => cd.days(formatDecimal(v));
const pct = (v: number) => formatPercent(v, 1);

export function SummaryTab({ data, ctx }: TabProps) {
  const cur = data.current;
  const prev = data.previous;
  const k = cur.kpis;
  const p = prev.kpis;
  const d = useDelta(ctx);
  const buckets = cur.buckets ?? [];
  const prevBuckets = prev.buckets ?? [];
  const gran = cur.granularity;
  const drill = useContentDrill(ctx);
  const range = periodRange(ctx);
  const board = (patch: Parameters<typeof contentBoardHref>[1] = {}) => contentBoardHref(ctx, { ...range, ...patch });
  const series = buckets.map((b, i) => ({
    start: b.start,
    label: bucketLabel(b.start, gran),
    planned: b.planned,
    published: b.published,
    previous: ctx.compare ? (prevBuckets[i]?.published ?? null) : null,
  }));
  const steps = funnelSteps(cur.funnel ?? []);

  return (
    <div className="space-y-6">
      <KpiGrid>
        <KpiTile label={cd.kPlanned} value={formatNumber(k.planned)} delta={d(k.planned, p.planned)} previous={formatNumber(p.planned)} spark={spark(buckets, (b) => b.planned)} href={board()} hint={cd.hPlanned} />
        <KpiTile
          label={cd.kPublished}
          value={formatNumber(k.published)}
          delta={d(k.published, p.published)}
          caption={cd.publishRate(pct(publishRate(k)))}
          spark={spark(buckets, (b) => b.published)}
          href={board({ status: ["PUBLISHED"] })}
          hint={cd.hPublished}
        />
        <KpiTile label={cd.kOnTime} value={pct(k.onTimeRate)} delta={d(k.onTimeRate, p.onTimeRate)} previous={pct(p.onTimeRate)} spark={spark(buckets, (b) => b.onTimeRate)} href={board()} hint={cd.hOnTime(k.deadlinesMet, k.deadlinesTotal)} />
        <KpiTile
          href={board({ rounds: "any" })}
          label={cd.kRevisions}
          value={formatDecimal(k.revisionAvg)}
          delta={d(k.revisionAvg, p.revisionAvg)}
          lowerIsBetter
          previous={formatDecimal(p.revisionAvg)}
          spark={spark(buckets, (b) => b.revisionAvg)}
          hint={cd.hRevisions}
        />
        <KpiTile label={cd.kInReview} value={formatNumber(k.inReview)} caption={cd.now} href={board({ status: ["REVIEW", "REVISION"] })} hint={cd.hInReview} />
        <KpiTile label={cd.kOverdue} value={formatNumber(k.overdue)} caption={cd.now} href={board({ overdue: true })} hint={cd.hOverdue} />
        <KpiTile label={cd.engagement} value={formatCompact(num(k.engagements))} delta={d(k.engagements, p.engagements)} previous={formatCompact(num(p.engagements))} spark={spark(buckets, (b) => b.engagements)} href={overviewHref(ctx, {}, "performa")} hint={cd.hEngagement} />
        <KpiTile label={cd.reach} value={formatCompact(num(k.reach))} delta={d(k.reach, p.reach)} previous={formatCompact(num(p.reach))} spark={spark(buckets, (b) => b.reach)} href={overviewHref(ctx, {}, "performa")} hint={cd.hReach} />
      </KpiGrid>

      <div className="grid gap-5 [&>*]:min-w-0 lg:grid-cols-5">
        <ChartCard
          className="lg:col-span-3"
          title={gran === "week" ? cd.publishedPerWeek : cd.publishedPerMonth}
          description={cd.publishedPerDesc}
          table={<BucketTable buckets={buckets} gran={gran} onSelect={(start) => drill(bucketDrill(start, gran))} />}
        >
          <PeriodChart
            data={series}
            onSelect={(row, _i, key) => drill(bucketDrill(String(row.start), gran, key === "planned" ? "planned" : "published"))}
            series={[
              { key: "published", label: cd.published, type: "bar", format: (v) => formatNumber(v) },
              { key: "planned", label: cd.planned, type: "line", color: slotColor(1), format: (v) => formatNumber(v) },
            ]}
            previousKey={ctx.compare ? "previous" : undefined}
            previousLabel={cd.publishedComparison}
            yFormat={(v) => formatNumber(v)}
            summary={cd.publishedSummary(k.published, k.planned)}
          />
        </ChartCard>
        <ChartCard className="lg:col-span-2" title={cd.funnelTitle} description={cd.funnelDesc}>
          <FunnelBars steps={steps} label={(key) => c.itemStatus[key as ItemStatus] ?? key} onSelect={(key) => drill({ ...range, reached: key as ItemStatus })} />
        </ChartCard>
      </div>

      <Section title={cd.perPeriodTable} description={cd.perPeriodTableDesc}>
        <div className="rounded-[1.5rem] bg-card p-5 shadow-(--card-shadow)">
          <BucketTable buckets={buckets} gran={gran} totals={k} onSelect={(start) => drill(bucketDrill(start, gran))} />
        </div>
      </Section>
    </div>
  );
}

/**
 * Drill for one period bucket: published items by publish day (what the bars
 * count) or planned items by planned day (what the planned line counts).
 */
function bucketDrill(start: string, gran: "week" | "month", by: "planned" | "published" = "planned") {
  const { from, to } = bucketRange(start, gran);
  return by === "published"
    ? { dateFrom: from, dateTo: to, dateBy: "published" as const, status: ["PUBLISHED" as ItemStatus] }
    : { dateFrom: from, dateTo: to };
}

function BucketTable({
  buckets,
  gran,
  totals,
  onSelect,
}: {
  buckets: DashBucket[];
  gran: "week" | "month";
  totals?: TabProps["data"]["current"]["kpis"];
  onSelect: (start: string) => void;
}) {
  return (
    <DetailTable
      rows={buckets}
      rowKey={(b) => b.start}
      showTotals={!!totals}
      columns={[
        { key: "start", label: cd.period, value: (b) => b.start, render: (b) => <TextButton onClick={() => onSelect(b.start)}>{bucketRangeLabel(b.start, gran)}</TextButton> },
        { key: "planned", label: cd.planned, align: "right", value: (b) => b.planned, total: totals ? formatNumber(totals.planned) : undefined },
        { key: "published", label: cd.published, align: "right", value: (b) => b.published, total: totals ? formatNumber(totals.published) : undefined },
        { key: "onTime", label: cd.kOnTime, align: "right", value: (b) => b.onTimeRate, render: (b) => (b.deadlinesTotal ? pct(b.onTimeRate) : "–"), total: totals ? pct(totals.onTimeRate) : undefined },
        { key: "rev", label: cd.kRevisions, align: "right", value: (b) => b.revisionAvg, render: (b) => formatDecimal(b.revisionAvg), total: totals ? formatDecimal(totals.revisionAvg) : undefined },
        { key: "eng", label: cd.engagement, align: "right", value: (b) => num(b.engagements), render: (b) => formatNumber(num(b.engagements)), total: totals ? formatNumber(num(totals.engagements)) : undefined },
      ]}
    />
  );
}

/** Stage cycle → the funnel status an item reaches once that stage is done. */
const STAGE_DONE = { COPY: "DESIGN", DESIGN: "REVIEW", REVIEW: "APPROVED", PUBLISH: "PUBLISHED" } as const satisfies Record<string, ItemStatus>;
const WIP_STAGE: Record<string, BoardStage> = { stageCopy: "COPY", stageDesign: "DESIGN", stageReview: "REVIEW", stageReady: "PUBLISH" };

const STAGE_SERIES = [
  { key: "stageCopy", label: "Copy", color: slotColor(0) },
  { key: "stageDesign", label: "Desain", color: slotColor(1) },
  { key: "stageReview", label: "Review", color: slotColor(2) },
  { key: "stageReady", label: "Siap tayang", color: slotColor(3) },
] as const;

export function ProductionTab({ data, ctx }: TabProps) {
  const cur = data.current;
  const gran = cur.granularity;
  const buckets = cur.buckets ?? [];
  const cycles = cur.stageCycles ?? [];
  const people = cur.people ?? [];
  const copywriters = people.filter((p) => p.role === "COPYWRITER");
  const designers = people.filter((p) => p.role === "DESIGNER");
  const overdueOwners = people.filter((p) => p.overdue > 0).toSorted((a, b) => b.overdue - a.overdue);
  const drill = useContentDrill(ctx);
  const range = periodRange(ctx);
  const workload = (rows: DashPerson[], color: string) =>
    rows.map((p) => ({
      key: p.userId,
      label: personName(ctx, p.userId),
      value: p.assigned,
      secondary: cd.workloadSecondary(p.published, p.overdue),
      color,
    }));
  const openPerson = (item: { key?: string }) => item.key && drill({ ...range, assignee: item.key });

  return (
    <div className="space-y-6">
      <div className="grid gap-5 [&>*]:min-w-0 lg:grid-cols-2">
        <ChartCard
          title={cd.cycleTitle}
          description={cd.cycleDesc}
          table={
            <DetailTable
              rows={cycles}
              rowKey={(r) => r.stage}
              showTotals={false}
              minWidth={320}
              columns={[
                { key: "stage", label: cd.stage, value: (r) => cd.stage4[r.stage] },
                { key: "avg", label: cd.avg, align: "right", value: (r) => r.avgDays, render: (r) => days(r.avgDays) },
                { key: "median", label: cd.median, align: "right", value: (r) => r.medianDays, render: (r) => days(r.medianDays) },
                { key: "count", label: cd.items, align: "right", value: (r) => r.count },
              ]}
            />
          }
        >
          <HBarChart
            items={cycles.map((r) => ({ key: r.stage, label: cd.stage4[r.stage], value: r.avgDays, secondary: cd.medianOf(days(r.medianDays)) }))}
            format={days}
            emptyLabel={cd.noData}
            onSelect={(it) => drill({ ...range, reached: STAGE_DONE[it.key as keyof typeof STAGE_DONE] })}
          />
          <p className="mt-4 text-xs text-muted-foreground">{cd.cycleTotal(days(cur.kpis.cycleDaysAvg))}</p>
        </ChartCard>
        <ChartCard title={cd.wipTitle} description={cd.wipDesc}>
          <PeriodChart
            onSelect={(row, _i, key) => {
              const { from, to } = bucketRange(String(row.start), gran);
              drill({ dateFrom: from, dateTo: to, stage: key ? WIP_STAGE[key] : undefined });
            }}
            data={buckets.map((b) => ({ start: b.start, label: bucketLabel(b.start, gran), stageCopy: b.stageCopy, stageDesign: b.stageDesign, stageReview: b.stageReview, stageReady: b.stageReady }))}
            series={STAGE_SERIES.map((s) => ({ ...s, type: "bar" as const, format: (v: number) => formatNumber(v) }))}
            stacked
            yFormat={(v) => formatNumber(v)}
            summary={cd.wipSummary}
          />
        </ChartCard>
      </div>

      <div className="grid gap-5 [&>*]:min-w-0 lg:grid-cols-3">
        <ChartCard title={cd.overdueStageTitle} description={cd.overdueStageDesc}>
          <HBarChart
            items={(cur.overdueByStage ?? []).map((r) => ({ key: r.key, label: cd.stage4[r.key as "COPY"] ?? r.key, value: r.count, color: "var(--status-critical)" }))}
            format={(v) => formatNumber(v)}
            emptyLabel={cd.noData}
            onSelect={(it) => drill({ ...range, overdue: true, stage: it.key as BoardStage })}
          />
        </ChartCard>
        <ChartCard title={cd.overdueOwnerTitle} description={cd.overdueOwnerDesc}>
          {overdueOwners.length ? (
            <HBarChart
              items={overdueOwners.map((p) => ({
                key: p.userId,
                label: `${personName(ctx, p.userId)} · ${cd.role[p.role]}`,
                value: p.overdue,
                secondary: cd.overdueSplit(p.overdueCopy, p.overdueDesign, p.overdueReview),
                color: "var(--status-critical)",
              }))}
              format={(v) => formatNumber(v)}
              onSelect={(it) => it.key && drill({ ...range, overdue: true, assignee: it.key })}
            />
          ) : (
            <p className="py-6 text-center text-sm text-muted-foreground">{cd.noOverdue}</p>
          )}
        </ChartCard>
        <ChartCard title={cd.roundsTitle} description={cd.roundsDesc}>
          <CategoryBars
            data={(cur.revisionRounds ?? []).map((r) => ({ key: r.key, label: cd.roundsLabel(r.key), value: r.count }))}
            onSelect={(r) => r.key && drill({ ...range, rounds: r.key })}
            label={cd.items}
            height={200}
            summary={cd.roundsTitle}
          />
        </ChartCard>
      </div>

      <div className="grid gap-5 [&>*]:min-w-0 lg:grid-cols-2">
        <ChartCard title={cd.workloadCopy} description={cd.workloadDesc}>
          <HBarChart items={workload(copywriters, slotColor(0))} format={(v) => formatNumber(v)} maxItems={8} emptyLabel={cd.noData} onSelect={openPerson} />
        </ChartCard>
        <ChartCard title={cd.workloadDesign} description={cd.workloadDesc}>
          <HBarChart items={workload(designers, slotColor(1))} format={(v) => formatNumber(v)} maxItems={8} emptyLabel={cd.noData} onSelect={openPerson} />
        </ChartCard>
      </div>

      <Section title={cd.peopleTable} description={cd.peopleTableDesc}>
        <div className="rounded-[1.5rem] bg-card p-5 shadow-(--card-shadow)">
          <DetailTable
            rows={people}
            rowKey={(p) => `${p.userId}-${p.role}`}
            initialSort={{ key: "assigned", desc: true }}
            columns={[
              { key: "name", label: cd.person, value: (p) => personName(ctx, p.userId), render: (p) => <TextLink href={contentBoardHref(ctx, { ...range, assignee: p.userId })}>{personName(ctx, p.userId)}</TextLink> },
              { key: "role", label: cd.roleCol, value: (p) => cd.role[p.role] },
              { key: "assigned", label: cd.assigned, align: "right", value: (p) => p.assigned, total: formatNumber(cur.kpis.planned) },
              { key: "published", label: cd.published, align: "right", value: (p) => p.published, total: formatNumber(cur.kpis.published) },
              { key: "overdue", label: cd.kOverdue, align: "right", value: (p) => p.overdue, total: formatNumber(cur.kpis.overdue) },
              { key: "onTime", label: cd.kOnTime, align: "right", value: (p) => p.onTimeRate, render: (p) => pct(p.onTimeRate), total: pct(cur.kpis.onTimeRate) },
              { key: "rev", label: cd.kRevisions, align: "right", value: (p) => p.revisionAvg, render: (p) => formatDecimal(p.revisionAvg), total: formatDecimal(cur.kpis.revisionAvg) },
            ]}
          />
        </div>
      </Section>
    </div>
  );
}

export function CalendarTab({ data, ctx }: TabProps) {
  const cur = data.current;
  const dayMap = new Map((cur.days ?? []).map((d) => [d.date, d]));
  const monthKeys = [...new Set((cur.days ?? []).map((d) => d.date.slice(0, 7)))].slice(-12);
  const months: CalendarMonth[] = monthKeys.map((m) => {
    const grid = monthGrid(`${m}-01`).filter((week) => week.some((d) => d.startsWith(m)));
    const plan = planForMonth(ctx, `${m}-01`);
    return {
      key: m,
      label: monthLabel(m),
      href: plan ? `${ctx.base}/plans/${plan}` : undefined,
      weeks: grid.map((week) => week.map((d) => (d.startsWith(m) ? (dayMap.get(d) ?? null) : null))),
    };
  });
  const weekdays = [0, 1, 2, 3, 4, 5, 6].map((i) => weekdayLabel(i));
  const heat = cur.postingHeat ?? [];
  const byWeekday = [0, 1, 2, 3, 4, 5, 6].map((w) => {
    const cells = heat.filter((h) => h.weekday === w);
    const posts = cells.reduce((s, h) => s + h.posts, 0);
    const eng = cells.reduce((s, h) => s + num(h.engagements), 0);
    const best = cells.toSorted((a, b) => num(b.engagements) / b.posts - num(a.engagements) / a.posts)[0];
    return { weekday: w, label: weekdayLabel(w, true), posts, eng, avg: posts ? eng / posts : 0, bestHour: best ? `${String(best.hour).padStart(2, "0")}:00` : "–" };
  });
  const drill = useContentDrill(ctx);
  const published = { ...periodRange(ctx), dateBy: "published" as const, status: ["PUBLISHED" as ItemStatus] };
  const totalPosts = byWeekday.reduce((s, r) => s + r.posts, 0);
  const totalEng = byWeekday.reduce((s, r) => s + r.eng, 0);

  return (
    <div className="space-y-6">
      <ChartCard title={cd.calendarTitle} description={cd.calendarDesc}>
        <MonthCalendars
          months={months}
          weekdays={weekdays}
          onSelectDay={(date, measure) =>
            drill(measure === "published" ? { dateFrom: date, dateTo: date, dateBy: "published", status: ["PUBLISHED"] } : { dateFrom: date, dateTo: date })
          }
        />
      </ChartCard>
      <ChartCard title={cd.postingHeatTitle} description={cd.postingHeatDesc}>
        <Heatmap
          cells={heat.map((h) => ({ row: h.weekday, col: h.hour, value: h.posts ? num(h.engagements) / h.posts : 0, count: h.posts }))}
          rows={7}
          cols={24}
          rowLabels={[0, 1, 2, 3, 4, 5, 6].map((i) => weekdayLabel(i))}
          colLabels={Array.from({ length: 24 }, (_, h) => String(h).padStart(2, "0"))}
          valueLabel={cd.avgEngagement}
          countLabel={cd.posts}
          format={(v) => formatNumber(Math.round(v))}
          cellSize={18}
          onSelect={(cell) => drill({ ...published, weekday: cell.row, hour: cell.col })}
        />
      </ChartCard>
      <Section title={cd.weekdayTable} description={cd.weekdayTableDesc}>
        <div className="rounded-[1.5rem] bg-card p-5 shadow-(--card-shadow)">
          <DetailTable
            rows={byWeekday}
            rowKey={(r) => String(r.weekday)}
            columns={[
              { key: "day", label: cd.weekday, value: (r) => r.weekday, render: (r) => <TextButton onClick={() => drill({ ...published, weekday: r.weekday })}>{r.label}</TextButton> },
              { key: "posts", label: cd.posts, align: "right", value: (r) => r.posts, total: formatNumber(totalPosts) },
              { key: "eng", label: cd.engagement, align: "right", value: (r) => r.eng, render: (r) => formatNumber(r.eng), total: formatNumber(totalEng) },
              { key: "avg", label: cd.avgEngagement, align: "right", value: (r) => r.avg, render: (r) => formatNumber(Math.round(r.avg)), total: formatNumber(totalPosts ? Math.round(totalEng / totalPosts) : 0) },
              { key: "best", label: cd.bestHour, align: "right", value: (r) => r.bestHour },
            ]}
          />
        </div>
      </Section>
    </div>
  );
}
