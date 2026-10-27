"use client";

import { ChartCard } from "@/components/dashboard/chart-card";
import { StatStrip } from "@/components/dashboard/stat-strip";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { StackedBarChart, type CategoryRow } from "@/components/dashboard/charts/stacked-bar-chart";
import { formatDecimal, formatNumber, formatPercent } from "@/lib/format";
import { candidatesHref, interviewsHref, rate, type DrillContext } from "../../analytics";
import { INTERVIEW_STATUS, INTERVIEW_TYPE, SESSION_STATUS, ra } from "../../strings";
import type { CountByKey, RecruitmentDashboard } from "../../types";
import { AnalyticsTable } from "./table";

const PSIKOTES = "/erp/hr/recruitment/psikotes";
const SESSION_ORDER = ["completed", "in_progress", "sent", "draft", "expired"];

type Interviewer = RecruitmentDashboard["interviewers"][number];

function countOf(list: CountByKey[], key: string): number {
  return list.find((c) => c.key === key)?.count ?? 0;
}

function total(list: CountByKey[]): number {
  return list.reduce((a, b) => a + b.count, 0);
}

export function AssessmentsTab({ data, ctx }: { data: RecruitmentDashboard; ctx: DrillContext }) {
  const s = ra.assessments;
  const psi = data.psikotesStatus;
  const ai = data.aiInterviewStatus;
  const psiDone = countOf(psi, "completed");
  const aiDone = countOf(ai, "completed");
  const scored = data.psikotesScores.reduce((a, b) => a + b.count, 0);
  const avgScore = scored ? data.psikotesScores.reduce((a, b) => a + b.avgScore * b.count, 0) / scored : null;
  const people = data.interviewers;
  const completed = people.reduce((a, b) => a + b.completed, 0);
  const weighted = people.filter((p) => p.avgScore != null && p.completed > 0);
  const weightSum = weighted.reduce((a, b) => a + b.completed, 0);
  const avgCard = weightSum ? weighted.reduce((a, b) => a + b.avgScore! * b.completed, 0) / weightSum : null;
  const sumOf = (pick: (p: Interviewer) => number) => people.reduce((a, b) => a + pick(b), 0);

  const statusBars = (list: CountByKey[]) =>
    SESSION_ORDER.filter((k) => countOf(list, k) > 0).map((k) => ({ label: SESSION_STATUS[k] ?? k, value: countOf(list, k), href: PSIKOTES }));

  const instruments = [...new Set(data.psikotesScores.map((x) => x.instrument))];
  const bands: CategoryRow[] = Array.from({ length: 10 }, (_, i) => {
    const row: CategoryRow = { label: s.band(i * 10), href: PSIKOTES };
    for (const inst of instruments) {
      row[inst] = data.psikotesScores.find((x) => x.instrument === inst && x.bucket === i * 10)?.count ?? 0;
    }
    return row;
  });

  const criteria = [...new Set(data.scorecards.map((c) => c.criterion))];
  const types = [...new Set(data.scorecards.map((c) => c.type))];
  const criteriaRows: CategoryRow[] = criteria.map((c) => {
    const row: CategoryRow = { label: c, href: interviewsHref(ctx, "completed") };
    for (const t of types) row[t] = data.scorecards.find((x) => x.criterion === c && x.type === t)?.avgScore ?? 0;
    return row;
  });

  return (
    <div className="page-stack">
      <StatStrip
        items={[
          { label: s.psikotesInvited, value: formatNumber(total(psi)), href: PSIKOTES },
          {
            label: s.psikotesDone,
            value: rate(psiDone, total(psi)) == null ? "–" : formatPercent(rate(psiDone, total(psi))!, 0),
            caption: s.rate(formatNumber(psiDone), formatNumber(total(psi))),
            href: PSIKOTES,
          },
          { label: s.avgScore, value: avgScore == null ? "–" : formatDecimal(avgScore, 1), href: PSIKOTES },
          {
            label: s.aiDone,
            value: rate(aiDone, total(ai)) == null ? "–" : formatPercent(rate(aiDone, total(ai))!, 0),
            caption: s.rate(formatNumber(aiDone), formatNumber(total(ai))),
            href: candidatesHref(ctx, { status: "interview" }),
          },
          { label: s.interviews, value: formatNumber(completed), href: interviewsHref(ctx, "completed") },
          { label: s.avgScorecard, value: avgCard == null ? "–" : `${formatDecimal(avgCard, 2)} / 5`, href: interviewsHref(ctx, "completed") },
        ]}
      />

      <div className="grid gap-(--gap) lg:grid-cols-2">
        <ChartCard title={s.psikotes} description={s.psikotesHint}>
          <HBarChart items={statusBars(psi)} format={formatNumber} showShare emptyLabel={ra.empty} />
        </ChartCard>
        <ChartCard title={s.ai} description={s.aiHint}>
          <HBarChart items={statusBars(ai)} format={formatNumber} showShare emptyLabel={ra.empty} />
        </ChartCard>
      </div>

      <div className="grid gap-(--gap) xl:grid-cols-2">
        <ChartCard title={s.scores} description={s.scoresHint}>
          {scored ? (
            <StackedBarChart
              data={bands}
              series={instruments.map((i) => ({ key: i, label: i, format: formatNumber }))}
              valueFormat={formatNumber}
              height={240}
            />
          ) : (
            <p className="py-8 text-center text-sm text-muted-foreground">{ra.empty}</p>
          )}
        </ChartCard>
        <ChartCard title={s.criteria} description={s.criteriaHint}>
          <StackedBarChart
            horizontal
            stacked={false}
            categoryWidth={180}
            data={criteriaRows}
            series={types.map((t) => ({ key: t, label: INTERVIEW_TYPE[t] ?? t, format: (v: number) => formatDecimal(v, 2) }))}
            valueFormat={(v) => formatDecimal(v, 0)}
            valueDomain={[0, 5]}
          />
        </ChartCard>
      </div>

      <ChartCard title={s.load} description={s.loadHint}>
        <StackedBarChart
          horizontal
          categoryWidth={180}
          data={people.map((p) => ({
            label: p.name,
            completed: p.completed,
            scheduled: p.scheduled,
            cancelled: p.cancelled,
            href: interviewsHref(ctx),
          }))}
          series={[
            { key: "completed", label: INTERVIEW_STATUS.completed!, format: formatNumber },
            { key: "scheduled", label: INTERVIEW_STATUS.scheduled!, format: formatNumber },
            { key: "cancelled", label: INTERVIEW_STATUS.cancelled!, format: formatNumber, color: "var(--chart-emphasis-muted)" },
          ]}
          valueFormat={formatNumber}
        />
      </ChartCard>

      <ChartCard title={s.table}>
        <AnalyticsTable<Interviewer>
          rows={people}
          rowKey={(r) => `${r.name}-${r.employeeId}`}
          rowHref={() => interviewsHref(ctx)}
          initialSort={{ key: "total", desc: true }}
          columns={[
            { key: "name", header: s.cols.interviewer, cell: (r) => <span className="font-medium">{r.name}</span>, sort: (r) => r.name },
            {
              key: "total",
              header: s.cols.total,
              align: "right",
              cell: (r) => formatNumber(r.completed + r.scheduled + r.cancelled),
              sort: (r) => r.completed + r.scheduled + r.cancelled,
              total: formatNumber(sumOf((p) => p.completed + p.scheduled + p.cancelled)),
            },
            { key: "completed", header: s.cols.completed, align: "right", cell: (r) => formatNumber(r.completed), sort: (r) => r.completed, total: formatNumber(sumOf((p) => p.completed)) },
            { key: "scheduled", header: s.cols.scheduled, align: "right", cell: (r) => formatNumber(r.scheduled), sort: (r) => r.scheduled, total: formatNumber(sumOf((p) => p.scheduled)) },
            { key: "cancelled", header: s.cols.cancelled, align: "right", cell: (r) => formatNumber(r.cancelled), sort: (r) => r.cancelled, total: formatNumber(sumOf((p) => p.cancelled)) },
            { key: "proceed", header: s.cols.proceed, align: "right", cell: (r) => formatNumber(r.proceed), sort: (r) => r.proceed, total: formatNumber(sumOf((p) => p.proceed)) },
            { key: "pool", header: s.cols.pool, align: "right", cell: (r) => formatNumber(r.pool), sort: (r) => r.pool, total: formatNumber(sumOf((p) => p.pool)) },
            { key: "reject", header: s.cols.reject, align: "right", cell: (r) => formatNumber(r.reject), sort: (r) => r.reject, total: formatNumber(sumOf((p) => p.reject)) },
            {
              key: "avg",
              header: s.cols.avg,
              align: "right",
              cell: (r) => (r.avgScore == null ? "–" : formatDecimal(r.avgScore, 2)),
              sort: (r) => r.avgScore ?? -1,
              total: avgCard == null ? "–" : formatDecimal(avgCard, 2),
            },
          ]}
        />
      </ChartCard>
    </div>
  );
}
