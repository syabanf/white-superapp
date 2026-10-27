"use client";

import { ChartCard } from "@/components/dashboard/chart-card";
import { StatStrip } from "@/components/dashboard/stat-strip";
import { BulletChart } from "@/components/dashboard/charts/bullet-chart";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { slotColor } from "@/components/dashboard/charts/chart-primitives";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { formatDecimal, formatNumber } from "@/lib/format";
import { candidatesHref, mean, type DrillContext } from "../../analytics";
import { JOB_STATUS, kinds, ra } from "../../strings";
import type { OpeningStat, RecruitmentDashboard } from "../../types";
import { AnalyticsTable } from "./table";

function windowDays(o: OpeningStat): number | null {
  if (!o.windowEnd) return null;
  return Math.max(1, Math.round((Date.parse(o.windowEnd) - Date.parse(o.openedOn)) / 86_400_000) + 1);
}

export function OpeningsTab({ data, ctx }: { data: RecruitmentDashboard; ctx: DrillContext }) {
  const s = ra.openings;
  const list = data.openings;
  const filled = list.filter((o) => o.timeToFillDays != null);
  const avgFill = mean(filled.map((o) => o.timeToFillDays!));
  const avgApplicants = mean(list.map((o) => o.applicantsPeriod));
  const sum = (pick: (o: OpeningStat) => number) => list.reduce((a, b) => a + pick(b), 0);
  const byDays = [...list].sort((a, b) => b.daysOpen - a.daysOpen);

  return (
    <div className="page-stack">
      <StatStrip
        items={[
          { label: s.active, value: formatNumber(list.length), caption: `${formatNumber(sum((o) => o.headcount))} kuota` },
          { label: s.filled, value: formatNumber(filled.length), caption: list.length ? `dari ${formatNumber(list.length)} lowongan` : undefined },
          { label: s.avgFill, value: avgFill == null ? "–" : ra.days(formatDecimal(avgFill, 1)), lowerIsBetter: true },
          { label: s.avgApplicants, value: avgApplicants == null ? "–" : formatDecimal(avgApplicants, 1) },
        ]}
      />

      <ChartCard title={s.window} description={s.windowHint}>
        <BulletChart
          rows={byDays.slice(0, 14).map((o) => ({
            key: o.id,
            label: o.title,
            sublabel: `${JOB_STATUS[o.status] ?? o.status} · ${o.department}`,
            value: o.daysOpen,
            rangeMin: windowDays(o) == null ? null : 0,
            rangeMax: windowDays(o),
            marker: o.timeToFillDays ?? null,
            href: candidatesHref(ctx, { job: o.id }),
          }))}
          format={(v) => ra.days(formatNumber(Math.round(v)))}
          rangeLabel={s.windowRange}
          valueLabel={s.windowValue}
          markerLabel={s.windowMarker}
          outsideLabel={s.overdue}
        />
      </ChartCard>

      <div className="grid gap-(--gap) lg:grid-cols-2">
        <ChartCard title={s.applicants} description={s.applicantsHint}>
          <HBarChart
            items={[...list]
              .sort((a, b) => b.applicantsPeriod - a.applicantsPeriod)
              .map((o) => ({ label: o.title, value: o.applicantsPeriod, href: candidatesHref(ctx, { job: o.id }) }))}
            maxItems={12}
            format={formatNumber}
            emptyLabel={ra.empty}
          />
        </ChartCard>
        <ChartCard title={s.fill} description={s.fillHint}>
          <HBarChart
            items={[...filled]
              .sort((a, b) => a.timeToFillDays! - b.timeToFillDays!)
              .map((o) => ({ label: o.title, value: o.timeToFillDays!, href: candidatesHref(ctx, { job: o.id, status: "hired" }) }))}
            maxItems={12}
            color={slotColor(2)}
            format={(v) => ra.days(formatDecimal(v, 1))}
            emptyLabel={ra.empty}
          />
        </ChartCard>
      </div>

      <ChartCard title={s.table}>
        <AnalyticsTable<OpeningStat>
          rows={list}
          rowKey={(r) => r.id}
          rowHref={(r) => candidatesHref(ctx, { job: r.id })}
          initialSort={{ key: "period", desc: true }}
          columns={[
            { key: "title", header: s.cols.opening, cell: (r) => <span className="font-medium">{r.title}</span>, sort: (r) => r.title },
            { key: "dept", header: s.cols.department, cell: (r) => r.department, sort: (r) => r.department },
            {
              key: "status",
              header: s.cols.status,
              cell: (r) => <ErpStatusBadge status={r.status} labels={JOB_STATUS} kinds={kinds.job} />,
              sort: (r) => r.status,
            },
            { key: "hc", header: s.cols.headcount, align: "right", cell: (r) => formatNumber(r.headcount), sort: (r) => r.headcount, total: formatNumber(sum((o) => o.headcount)) },
            { key: "period", header: s.cols.period, align: "right", cell: (r) => formatNumber(r.applicantsPeriod), sort: (r) => r.applicantsPeriod, total: formatNumber(sum((o) => o.applicantsPeriod)) },
            { key: "total", header: s.cols.total, align: "right", cell: (r) => formatNumber(r.applicantsTotal), sort: (r) => r.applicantsTotal, total: formatNumber(sum((o) => o.applicantsTotal)) },
            { key: "hires", header: s.cols.hires, align: "right", cell: (r) => formatNumber(r.hires), sort: (r) => r.hires, total: formatNumber(sum((o) => o.hires)) },
            { key: "open", header: s.cols.daysOpen, align: "right", cell: (r) => formatNumber(r.daysOpen), sort: (r) => r.daysOpen },
            {
              key: "left",
              header: s.cols.daysLeft,
              align: "right",
              cell: (r) => (r.daysToClose == null ? "–" : formatNumber(r.daysToClose)),
              sort: (r) => r.daysToClose ?? Number.MAX_SAFE_INTEGER,
            },
            {
              key: "fill",
              header: s.cols.fill,
              align: "right",
              cell: (r) => (r.timeToFillDays == null ? "–" : ra.days(formatDecimal(r.timeToFillDays, 1))),
              sort: (r) => r.timeToFillDays ?? Number.MAX_SAFE_INTEGER,
            },
          ]}
        />
      </ChartCard>
    </div>
  );
}
