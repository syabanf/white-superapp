"use client";

import { ChartCard } from "@/components/dashboard/chart-card";
import { KpiGrid } from "@/components/dashboard/kpi-tile";
import { MUTED_SERIES, slotColor } from "@/components/dashboard/charts/chart-primitives";
import { Funnel, LinkedTrendChart, PairedBars, StackedRows } from "./charts";
import { href, int, pct, score, scoreDomain } from "./lib";
import { ds } from "./strings";
import type { DashReviews } from "./types";
import { MetricTile, SortTable } from "./widgets";

const r = ds.reviews;
const STEPS = ["total", "selfAssessed", "rated", "signed", "finalized"] as const;

export function ReviewsTab({ data }: { data: DashReviews }) {
  const c = data.cycle;
  const reviews = href("/erp/performance/reviews", { tab: "cycles", cycle: c.cycleId || undefined });
  const feedback = href("/erp/performance/feedback", { cycle: data.feedbackCycleId || undefined });
  const hasPrevRatings = data.ratings.some((x) => x.previous != null);
  const ratings = data.ratings.map((x) => ({ key: x.key, value: x.value, previous: x.previous ?? null }));
  const one = (v: number) => v.toLocaleString("id-ID", { maximumFractionDigits: 1 });

  return (
    <div className="flex flex-col gap-4 sm:gap-5">
      <KpiGrid cols={4}>
        <MetricTile
          label={r.score}
          hint={r.scoreHint}
          metric={data.reviewScore}
          format={score}
          href={reviews}
        />
        <MetricTile
          label={r.steps.finalized}
          metric={{ value: c.total ? c.finalized / c.total : undefined, spark: [] }}
          format={pct}
          href={reviews}
        />
        <MetricTile
          label={r.feedbackScore}
          hint={r.feedbackScoreHint}
          metric={data.feedbackScore}
          format={(v) => (v == null ? "–" : one(v))}
          href={feedback}
        />
        <MetricTile
          label={r.feedbackCompletion}
          metric={data.feedbackCompletion}
          format={pct}
          href={feedback}
        />
      </KpiGrid>

      <div className="grid gap-4 sm:gap-5 lg:grid-cols-2">
        <ChartCard title={c.cycleId ? `${r.cycle} · ${c.name}` : r.cycle} description={r.cycleHint}>
          {c.cycleId ? (
            <Funnel
              relative="first"
              steps={STEPS.map((k) => ({ label: r.steps[k], value: c[k], href: reviews }))}
            />
          ) : (
            <p className="py-10 text-center text-sm text-muted-foreground">{r.noCycle}</p>
          )}
        </ChartCard>
        <ChartCard
          title={r.ratings}
          description={r.ratingsHint}
          table={
            <SortTable
              rows={ratings}
              rowKey={(x) => x.key}
              columns={[
                { key: "k", label: "Kategori", render: (x) => x.key },
                { key: "v", label: ds.current, align: "right", render: (x) => int(x.value) },
                { key: "p", label: ds.previous, align: "right", render: (x) => int(x.previous) },
              ]}
            />
          }
        >
          <LinkedTrendChart
            data={ratings}
            xKey="key"
            xFormat={(v) => v}
            yFormat={(v) => int(v)}
            series={[
              { key: "value", label: r.cycleLabel(c.name), type: "bar", color: slotColor(0) },
              ...(hasPrevRatings
                ? [{ key: "previous", label: ds.previous, type: "bar" as const, color: MUTED_SERIES }]
                : []),
            ]}
            hrefFor={() => reviews}
            summary={r.ratings}
          />
        </ChartCard>
      </div>

      <ChartCard title={r.cycles}>
        <LinkedTrendChart
          data={data.cycles.map((x) => ({ name: x.name, score: x.avgScore ?? null, id: x.cycleId }))}
          xKey="name"
          xFormat={(v) => v}
          yFormat={(v) => score(v)}
          yDomain={scoreDomain(data.cycles.map((x) => x.avgScore))}
          height={220}
          series={[{ key: "score", label: r.score, color: slotColor(0), format: (v: number) => score(v) }]}
          hrefFor={(row) => href("/erp/performance/reviews", { tab: "cycles", cycle: String(row.id) })}
          summary={r.cycles}
        />
      </ChartCard>

      <div className="grid gap-4 sm:gap-5 lg:grid-cols-2">
        <ChartCard
          title={
            data.feedbackCycleName
              ? `${r.feedbackCategories} · ${data.feedbackCycleName}`
              : r.feedbackCategories
          }
          description={r.feedbackCategoriesHint}
        >
          {data.feedbackCycleId ? (
            <PairedBars
              rows={data.feedbackCategories.map((x) => ({
                label: x.name,
                value: x.score,
                previous: x.previous,
                href: feedback,
              }))}
              max={100}
              format={one}
              currentLabel={data.feedbackCycleName}
              previousLabel={data.previousFeedbackCycleName || undefined}
            />
          ) : (
            <p className="py-10 text-center text-sm text-muted-foreground">{r.noFeedback}</p>
          )}
        </ChartCard>
        <ChartCard title={r.feedbackProgress} description={r.feedbackProgressHint}>
          <StackedRows
            segments={[
              { key: "done", label: ds.logbook.sent, color: slotColor(0) },
              { key: "open", label: "Belum", color: MUTED_SERIES },
            ]}
            rows={data.feedbackProgress.map((p) => ({
              label: `${r.relationships[p.relationship] ?? p.relationship} · ${pct(p.total ? p.submitted / p.total : undefined)}`,
              href: feedback,
              values: { done: p.submitted, open: p.total - p.submitted },
            }))}
          />
        </ChartCard>
      </div>

      <ChartCard title={r.departments}>
        <SortTable
          rows={data.departments}
          rowKey={(d) => d.departmentId}
          rowHref={() => reviews}
          initialSort={{ key: "avg", desc: true }}
          columns={[
            {
              key: "n",
              label: ds.filters.department,
              render: (d) => d.departmentName,
              sort: (d) => d.departmentName,
            },
            {
              key: "r",
              label: r.reviewsCol,
              align: "right",
              render: (d) => int(d.reviews),
              sort: (d) => d.reviews,
              total: int(data.departments.reduce((a, d) => a + d.reviews, 0)),
            },
            {
              key: "f",
              label: r.finalCol,
              align: "right",
              render: (d) => int(d.finalized),
              sort: (d) => d.finalized,
              total: int(data.departments.reduce((a, d) => a + d.finalized, 0)),
            },
            {
              key: "avg",
              label: r.avgCol,
              align: "right",
              render: (d) => score(d.avgScore),
              sort: (d) => d.avgScore,
            },
            {
              key: "f360",
              label: r.avg360Col,
              align: "right",
              render: (d) => (d.avg360 == null ? "–" : one(d.avg360)),
              sort: (d) => d.avg360,
            },
          ]}
        />
      </ChartCard>
    </div>
  );
}
