"use client";

import Link from "next/link";
import { ChartCard } from "@/components/dashboard/chart-card";
import { StatStrip } from "@/components/dashboard/stat-strip";
import { BulletChart } from "@/components/dashboard/charts/bullet-chart";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { TimeSeriesChart } from "@/components/dashboard/charts/time-series-chart";
import { slotColor } from "@/components/dashboard/charts/chart-primitives";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { formatRupiah } from "@/features/erp/shared/money";
import { computeDelta } from "@/lib/metrics";
import { formatCurrency, formatDate, formatDecimal, formatNumber, formatPercent } from "@/lib/format";
import { candidateHref, candidatesHref, mean, responseBuckets, type DrillContext } from "../../analytics";
import { useRecruitmentDrill, weekRange } from "./drill";
import { OFFER_STATUS, kinds, ra } from "../../strings";
import type { DashboardOffer, RecruitmentDashboard } from "../../types";
import { AnalyticsTable } from "./table";
import { WeekTable } from "./week-table";

const rp = (v: number) => formatCurrency(v, "IDR", { compact: true });
const num = (v: string | null | undefined) => (v == null || v === "" ? null : Number(v));

export function OffersTab({ data, ctx }: { data: RecruitmentDashboard; ctx: DrillContext }) {
  const s = ra.offers;
  const k = data.kpis;
  const p = data.previousKpis;
  const offers = data.offers;
  const responses = offers.filter((o) => o.responseDays != null).map((o) => o.responseDays!);
  const avgResponse = mean(responses);
  const negotiating = offers.filter((o) => o.status === "negotiating").length;
  const buckets = responseBuckets(offers);
  const drill = useRecruitmentDrill(ctx);
  const weeks = data.trend.map((t) => ({
    date: t.start,
    accepted: t.offersAccepted,
    declined: t.offersDeclined,
    expired: t.offersExpired,
    negotiating: t.offersNegotiating,
  }));

  return (
    <div className="page-stack">
      <StatStrip
        items={[
          { label: s.sent, value: formatNumber(k.offersSent), delta: computeDelta(k.offersSent, p.offersSent), previous: formatNumber(p.offersSent), href: candidatesHref(ctx, { status: "offer" }) },
          { label: s.accepted, value: formatNumber(k.offersAccepted), delta: computeDelta(k.offersAccepted, p.offersAccepted), previous: formatNumber(p.offersAccepted), href: candidatesHref(ctx, { status: "hired" }) },
          {
            label: s.declined,
            value: formatNumber(k.offersDeclined),
            delta: computeDelta(k.offersDeclined, p.offersDeclined),
            previous: formatNumber(p.offersDeclined),
            lowerIsBetter: true,
            href: "#tabel-offer",
          },
          {
            label: s.acceptance,
            value: k.offerAcceptanceRate == null ? "–" : formatPercent(k.offerAcceptanceRate, 0),
            delta: k.offerAcceptanceRate != null && p.offerAcceptanceRate != null ? computeDelta(k.offerAcceptanceRate, p.offerAcceptanceRate) : null,
          },
          { label: s.responseTime, value: avgResponse == null ? "–" : ra.days(formatDecimal(avgResponse, 1)), lowerIsBetter: true, href: "#tabel-offer" },
          { label: s.negotiating, value: formatNumber(negotiating), href: "#tabel-offer" },
        ]}
      />

      <ChartCard
        title={s.outcomes}
        description={s.outcomesHint}
        table={
          <WeekTable
            rows={weeks.map((w) => ({ date: w.date, values: [w.accepted, w.declined, w.expired, w.negotiating] }))}
            headers={[OFFER_STATUS.accepted!, OFFER_STATUS.declined!, OFFER_STATUS.expired!, OFFER_STATUS.negotiating!]}
          />
        }
      >
        <TimeSeriesChart
          data={weeks}
          stacked
          series={[
            { key: "accepted", label: OFFER_STATUS.accepted!, type: "bar", color: slotColor(2), format: formatNumber },
            { key: "declined", label: OFFER_STATUS.declined!, type: "bar", color: slotColor(1), format: formatNumber },
            { key: "expired", label: OFFER_STATUS.expired!, type: "bar", color: "var(--chart-emphasis-muted)", format: formatNumber },
            { key: "negotiating", label: OFFER_STATUS.negotiating!, type: "bar", color: slotColor(3), format: formatNumber },
          ]}
          yFormat={formatNumber}
          height={250}
          onSelect={(pt) => drill("offer", { ...weekRange(String(pt.date)), week: 1, l: ra.drill.week(formatDate(String(pt.date))) })}
        />
      </ChartCard>

      <div className="grid gap-(--gap) xl:grid-cols-3">
        <ChartCard className="xl:col-span-2" title={s.salary} description={s.salaryHint}>
          <BulletChart
            rows={offers.slice(0, 14).map((o) => ({
              key: o.candidateId,
              label: o.candidateName,
              sublabel: o.positionTitle,
              value: Number(o.baseSalary),
              rangeMin: num(o.salaryMin),
              rangeMax: num(o.salaryMax),
              marker: num(o.expectedSalary),
              href: candidateHref(o.candidateId, "offers"),
              detail: <div>{formatRupiah(o.baseSalary)}</div>,
            }))}
            format={rp}
            rangeLabel={s.salaryRange}
            valueLabel={s.salaryValue}
            markerLabel={s.salaryMarker}
            outsideLabel={s.outside}
          />
        </ChartCard>
        <ChartCard title={s.responseDist} description={s.responseDistHint}>
          <HBarChart
            items={s.buckets.map((label, i) => ({ label, value: buckets[i] ?? 0, key: String(i) }))}
            onSelect={(it) => drill("offer", { bucket: it.key, l: `${s.responseDist} · ${it.label}` })}
            format={formatNumber}
            showShare
            emptyLabel={ra.empty}
          />
        </ChartCard>
      </div>

      <div id="tabel-offer" className="scroll-mt-20">
      <ChartCard title={s.table}>
        <AnalyticsTable<DashboardOffer>
          rows={offers}
          rowKey={(r) => r.candidateId}
          rowHref={(r) => candidateHref(r.candidateId, "offers")}
          initialSort={{ key: "sent", desc: true }}
          columns={[
            {
              key: "name",
              header: s.cols.candidate,
              cell: (r) => (
                <Link href={candidateHref(r.candidateId, "offers")} className="font-medium hover:underline" onClick={(e) => e.stopPropagation()}>
                  {r.candidateName}
                </Link>
              ),
              sort: (r) => r.candidateName,
            },
            { key: "pos", header: s.cols.position, cell: (r) => r.positionTitle, sort: (r) => r.positionTitle },
            {
              key: "status",
              header: s.cols.status,
              cell: (r) => <ErpStatusBadge status={r.status} labels={OFFER_STATUS} kinds={kinds.offer} />,
              sort: (r) => r.status,
            },
            {
              key: "salary",
              header: s.cols.salary,
              align: "right",
              cell: (r) => formatRupiah(r.baseSalary),
              sort: (r) => Number(r.baseSalary),
              total: formatRupiah(offers.reduce((a, b) => a + Number(b.baseSalary), 0)),
            },
            {
              key: "range",
              header: s.cols.range,
              cell: (r) => (r.salaryMin && r.salaryMax ? `${rp(Number(r.salaryMin))} – ${rp(Number(r.salaryMax))}` : "–"),
            },
            { key: "exp", header: s.cols.expected, align: "right", cell: (r) => (r.expectedSalary ? formatRupiah(r.expectedSalary) : "–") },
            { key: "sent", header: s.cols.sent, cell: (r) => formatDate(r.sentAt), sort: (r) => r.sentAt },
            {
              key: "resp",
              header: s.cols.response,
              align: "right",
              cell: (r) => (r.responseDays == null ? "–" : ra.days(formatDecimal(r.responseDays, 1))),
              sort: (r) => r.responseDays ?? Number.MAX_SAFE_INTEGER,
            },
            { key: "ver", header: s.cols.versions, align: "right", cell: (r) => formatNumber(r.versions), sort: (r) => r.versions },
          ]}
        />
      </ChartCard>
      </div>
    </div>
  );
}
