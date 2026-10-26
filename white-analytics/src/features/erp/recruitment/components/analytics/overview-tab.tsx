"use client";

import { Briefcase, Clock, FileSignature, FileX, Handshake, UserCheck, Users, Workflow } from "lucide-react";
import { ChartCard } from "@/components/dashboard/chart-card";
import { KpiGrid, KpiTile } from "@/components/dashboard/kpi-tile";
import { FunnelChart } from "@/components/dashboard/charts/funnel-chart";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { TimeSeriesChart } from "@/components/dashboard/charts/time-series-chart";
import { computeDelta } from "@/lib/metrics";
import { formatDateShort, formatDecimal, formatNumber, formatPercent } from "@/lib/format";
import { useDrillParams } from "@/features/erp/shared/drill";
import { acceptanceSeries, candidatesHref, dashboardHref, overlay, type DrillContext } from "../../analytics";
import { useRecruitmentDrill, weekRange } from "./drill";
import { CANDIDATE_STATUS, STAGE_ORDER, ra } from "../../strings";
import type { OpeningStat, RecruitmentDashboard } from "../../types";
import { AnalyticsTable } from "./table";
import { WeekTable } from "./week-table";

export function OverviewTab({ data, ctx }: { data: RecruitmentDashboard; ctx: DrillContext }) {
  const k = data.kpis;
  const p = data.previousKpis;
  const t = data.trend;
  const o = ra.overview;
  const days = (v: number | null | undefined) => (v == null ? "–" : ra.days(formatDecimal(v, 1)));
  const lost = k.offersDeclined + k.offersExpired;
  const prevLost = p.offersDeclined + p.offersExpired;

  const apps = overlay(t, data.previousTrend, "applications");
  const drill = useRecruitmentDrill(ctx);
  const { apply } = useDrillParams();
  const week = (start: string) => drill("cand", { ...weekRange(start), week: 1, l: ra.drill.week(formatDateShort(start)) });
  const flow = t.map((x) => ({ date: x.start, offers: x.offersSent, hires: x.hires }));
  const statusCounts = new Map<string, number>();
  for (const r of data.openingStages) statusCounts.set(r.status, (statusCounts.get(r.status) ?? 0) + r.count);
  const composition = STAGE_ORDER.filter((s) => statusCounts.get(s)).map((s) => ({
    label: CANDIDATE_STATUS[s] ?? s,
    value: statusCounts.get(s) ?? 0,
    key: s,
  }));
  const totalApplicants = data.openings.reduce((a, b) => a + b.applicantsPeriod, 0);
  const totalHires = data.openings.reduce((a, b) => a + b.hires, 0);

  return (
    <div className="flex flex-col gap-4 sm:gap-5">
      <KpiGrid>
        <KpiTile
          label={ra.kpi.openPositions}
          hint={ra.kpi.openPositionsHint}
          icon={<Briefcase />}
          value={formatNumber(k.openPositions)}
          delta={computeDelta(k.openPositions, p.openPositions)}
          previous={formatNumber(p.openPositions)}
          caption={ra.kpi.headcount(k.openHeadcount)}
          spark={t.map((x) => x.openPositions)}
          href="/erp/hr/recruitment/jobs?status=published"
        />
        <KpiTile
          label={ra.kpi.applicants}
          hint={ra.kpi.applicantsHint}
          icon={<Users />}
          value={formatNumber(k.applicants)}
          delta={computeDelta(k.applicants, p.applicants)}
          previous={formatNumber(p.applicants)}
          spark={t.map((x) => x.applications)}
          href={candidatesHref(ctx)}
        />
        <KpiTile
          label={ra.kpi.inProcess}
          hint={ra.kpi.inProcessHint}
          icon={<Workflow />}
          value={formatNumber(k.inProcess)}
          delta={computeDelta(k.inProcess, p.inProcess)}
          previous={formatNumber(p.inProcess)}
          spark={t.map((x) => x.inProcess)}
          href={`/erp/hr/recruitment/pipeline${ctx.job ? `?job=${ctx.job}` : ""}`}
        />
        <KpiTile
          label={ra.kpi.offersSent}
          icon={<FileSignature />}
          value={formatNumber(k.offersSent)}
          delta={computeDelta(k.offersSent, p.offersSent)}
          previous={formatNumber(p.offersSent)}
          spark={t.map((x) => x.offersSent)}
          href={candidatesHref(ctx, { status: "offer" })}
        />
        <KpiTile
          label={ra.kpi.hires}
          hint={ra.kpi.hiresHint}
          icon={<UserCheck />}
          value={formatNumber(k.hires)}
          delta={computeDelta(k.hires, p.hires)}
          previous={formatNumber(p.hires)}
          spark={t.map((x) => x.hires)}
          href={candidatesHref(ctx, { status: "hired" })}
        />
        <KpiTile
          label={ra.kpi.timeToHire}
          hint={ra.kpi.timeToHireHint}
          icon={<Clock />}
          value={days(k.avgDaysToHire)}
          lowerIsBetter
          delta={k.avgDaysToHire != null && p.avgDaysToHire != null ? computeDelta(k.avgDaysToHire, p.avgDaysToHire) : null}
          previous={days(p.avgDaysToHire)}
          spark={t.map((x) => x.avgDaysToHire ?? 0)}
          href={candidatesHref(ctx, { status: "hired" })}
        />
        <KpiTile
          label={ra.kpi.acceptance}
          hint={ra.kpi.acceptanceHint}
          icon={<Handshake />}
          value={k.offerAcceptanceRate == null ? "–" : formatPercent(k.offerAcceptanceRate, 0)}
          delta={
            k.offerAcceptanceRate != null && p.offerAcceptanceRate != null
              ? computeDelta(k.offerAcceptanceRate, p.offerAcceptanceRate)
              : null
          }
          previous={p.offerAcceptanceRate == null ? "–" : formatPercent(p.offerAcceptanceRate, 0)}
          spark={acceptanceSeries(t)}
          href={dashboardHref(ctx, { tab: "offer" })}
        />
        <KpiTile
          label={ra.kpi.offersLost}
          hint={ra.kpi.offersLostHint}
          icon={<FileX />}
          value={formatNumber(lost)}
          lowerIsBetter
          delta={computeDelta(lost, prevLost)}
          previous={formatNumber(prevLost)}
          spark={t.map((x) => x.offersDeclined + x.offersExpired)}
          href={dashboardHref(ctx, { tab: "offer" })}
        />
      </KpiGrid>

      <div className="grid gap-4 sm:gap-5 xl:grid-cols-3">
        <ChartCard
          className="xl:col-span-2"
          title={o.applications}
          description={o.applicationsHint}
          table={<WeekTable rows={apps.map((a) => ({ date: a.date, values: [a.value, a.previous] }))} headers={[o.applicationsSeries, ra.previous]} />}
        >
          <TimeSeriesChart
            data={apps}
            series={[{ key: "value", label: o.applicationsSeries, type: "bar", format: formatNumber }]}
            previousKey="previous"
            yFormat={formatNumber}
            height={260}
            onSelect={(p) => week(String(p.date))}
            summary={`${o.applications}: ${formatNumber(k.applicants)} lamaran dalam ${t.length} minggu.`}
          />
        </ChartCard>
        <ChartCard title={o.funnel} description={o.funnelHint}>
          <FunnelChart
            steps={data.funnel.map((f) => ({
              key: f.stage,
              label: CANDIDATE_STATUS[f.stage] ?? f.label,
              value: f.reached,
              exited: f.exited,
            }))}
            onSelect={(s) => drill("cand", { status: s.key, l: s.label })}
          />
        </ChartCard>
      </div>

      <div className="grid gap-4 sm:gap-5 xl:grid-cols-3">
        <ChartCard
          className="xl:col-span-2"
          title={o.flow}
          description={o.flowHint}
          table={<WeekTable rows={flow.map((f) => ({ date: f.date, values: [f.offers, f.hires] }))} headers={[ra.kpi.offersSent, ra.kpi.hires]} />}
        >
          <TimeSeriesChart
            data={flow}
            series={[
              { key: "offers", label: ra.kpi.offersSent, type: "bar", format: formatNumber },
              { key: "hires", label: ra.kpi.hires, type: "bar", format: formatNumber },
            ]}
            yFormat={formatNumber}
            height={240}
            onSelect={(p) => {
              const r = weekRange(String(p.date));
              apply({ preset: "custom", from: r.from, to: r.to });
            }}
            selectLabel={o.flowSelect}
          />
        </ChartCard>
        <ChartCard title={o.composition} description={o.compositionHint}>
          <HBarChart items={composition} format={formatNumber} showShare emptyLabel={ra.empty} onSelect={(it) => drill("cand", { status: it.key, l: it.label })} />
        </ChartCard>
      </div>

      <ChartCard title={o.topOpenings}>
        <AnalyticsTable<OpeningStat>
          rows={data.openings}
          rowKey={(r) => r.id}
          rowHref={(r) => candidatesHref(ctx, { job: r.id })}
          initialSort={{ key: "period", desc: true }}
          limit={8}
          columns={[
            { key: "title", header: ra.openings.cols.opening, cell: (r) => <span className="font-medium">{r.title}</span>, sort: (r) => r.title },
            { key: "dept", header: ra.openings.cols.department, cell: (r) => r.department, sort: (r) => r.department },
            {
              key: "period",
              header: ra.openings.cols.period,
              align: "right",
              cell: (r) => formatNumber(r.applicantsPeriod),
              sort: (r) => r.applicantsPeriod,
              total: formatNumber(totalApplicants),
            },
            { key: "hires", header: ra.openings.cols.hires, align: "right", cell: (r) => formatNumber(r.hires), sort: (r) => r.hires, total: formatNumber(totalHires) },
            { key: "days", header: ra.openings.cols.daysOpen, align: "right", cell: (r) => formatNumber(r.daysOpen), sort: (r) => r.daysOpen },
          ]}
        />
      </ChartCard>
    </div>
  );
}
