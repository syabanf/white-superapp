import { Suspense } from "react";
import type { Metadata } from "next";
import Link from "next/link";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { formatPeriod, resolvePeriod, todayWib, addDaysIso, type ResolvedPeriod } from "@/features/erp/shared/dashboard-period";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpEmpty, ErpLoadError } from "@/features/erp/shared/states";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import type { DrillContext } from "@/features/erp/recruitment/analytics";
import { AssessmentsTab } from "@/features/erp/recruitment/components/analytics/assessments-tab";
import { RecruitmentFilters } from "@/features/erp/recruitment/components/analytics/filters";
import { OffersTab } from "@/features/erp/recruitment/components/analytics/offers-tab";
import { OpeningsTab } from "@/features/erp/recruitment/components/analytics/openings-tab";
import { OverviewTab } from "@/features/erp/recruitment/components/analytics/overview-tab";
import { PipelineTab } from "@/features/erp/recruitment/components/analytics/pipeline-tab";
import { RecruitmentDrillSheet } from "@/features/erp/recruitment/components/analytics/drill";
import { AnalyticsSkeleton } from "@/features/erp/recruitment/components/analytics/skeleton";
import { SourcesTab } from "@/features/erp/recruitment/components/analytics/sources-tab";
import { getDashboard, listInterviews, listJobOpenings } from "@/features/erp/recruitment/queries";
import { DashboardTrail, type TrailLevel } from "@/features/erp/sales/dashboard/trail";
import { CANDIDATE_STATUS, SOURCE, INTERVIEW_MODE, INTERVIEW_TYPE, JOB_STATUS, ra, rs } from "@/features/erp/recruitment/strings";
import { formatDateTime } from "@/lib/format";

export const metadata: Metadata = { title: ra.title };

const TABS = {
  ringkasan: OverviewTab,
  pipeline: PipelineTab,
  sumber: SourcesTab,
  lowongan: OpeningsTab,
  asesmen: AssessmentsTab,
  offer: OffersTab,
} as const;

type Tab = keyof typeof TABS;

export default async function RecruitmentDashboardPage(props: PageProps<"/erp/hr/recruitment">) {
  const { offline } = await requireErp("recruitment.read");
  if (offline) return <OfflineNotice message={offline.message} />;

  const sp = await props.searchParams;
  const period = resolvePeriod({ preset: sp1(sp.preset), from: sp1(sp.from), to: sp1(sp.to), compare: sp1(sp.compare) });
  const requested = sp1(sp.tab);
  const tab: Tab = requested && requested in TABS ? (requested as Tab) : "ringkasan";
  const ctx: DrillContext = {
    from: period.from,
    to: period.to,
    compare: period.compare,
    job: sp1(sp.job),
    department: sp1(sp.department),
    source: sp1(sp.source),
    stage: sp1(sp.stage),
  };

  const jobs = await load(() => listJobOpenings());
  const openings = jobs.data ?? [];
  const jobOptions = openings.map((j) => ({
    value: j.id,
    label: `${j.title} · ${JOB_STATUS[j.status] ?? j.status}${j.publishedAt ? ` ${j.publishedAt.slice(0, 7)}` : ""}`,
  }));
  const departments = [...new Set(openings.map((j) => j.department).filter(Boolean))].sort();
  const trail: TrailLevel[] = [
    period.preset === "custom" ? { label: period.label, params: ["preset", "from", "to"] } : null,
    ctx.department ? { label: ctx.department, params: ["department"] } : null,
    ctx.job ? { label: openings.find((j) => j.id === ctx.job)?.title ?? ctx.job, params: ["job"] } : null,
    ctx.source ? { label: SOURCE[ctx.source] ?? ctx.source, params: ["source"] } : null,
    ctx.stage ? { label: CANDIDATE_STATUS[ctx.stage] ?? ctx.stage, params: ["stage"] } : null,
  ].filter((l): l is TrailLevel => l != null);

  return (
    <>
      <PageHeader eyebrow={rs.eyebrow} title={ra.title} description={ra.subtitle} />
      <RecruitmentFilters period={period} jobs={jobOptions} departments={departments} />
      <DashboardTrail levels={trail} />
      {/* the outer box scrolls so the first tab stays reachable on phones */}
      <div className="-mx-1 overflow-x-auto px-1 scrollbar-thin">
        <UrlTabs tabs={ra.tabs} label={ra.tabsLabel} className="w-max" />
      </div>
      <Suspense key={JSON.stringify(ctx)} fallback={<AnalyticsSkeleton />}>
        <DashboardBody tab={tab} period={period} ctx={ctx} />
      </Suspense>
      {tab === "ringkasan" ? (
        <Suspense fallback={null}>
          <UpcomingInterviews />
        </Suspense>
      ) : null}
    </>
  );
}

async function DashboardBody({ tab, period, ctx }: { tab: Tab; period: ResolvedPeriod; ctx: DrillContext }) {
  const res = await load(() =>
    getDashboard({
      from: period.from,
      to: period.to,
      compare: period.compare,
      jobOpeningId: ctx.job,
      department: ctx.department,
      source: ctx.source,
      stage: ctx.stage,
    }),
  );
  if (!res.data) return <ErpLoadError error={res.error} />;
  const d = res.data;
  const Body = TABS[tab];
  return (
    <div className="flex flex-col gap-3">
      <p className="text-xs text-muted-foreground tabular">
        {ra.periodCaption(formatPeriod(d.period.from, d.period.to), formatPeriod(d.comparePeriod.from, d.comparePeriod.to), d.compare)}
      </p>
      <Body data={d} ctx={ctx} />
      <RecruitmentDrillSheet ctx={ctx} offers={d.offers} />
    </div>
  );
}

async function UpcomingInterviews() {
  const today = todayWib();
  const upcoming = await load(() => listInterviews({ status: "scheduled", dateFrom: today, dateTo: addDaysIso(today, 7) }));
  const o = ra.overview;
  return (
    <Section
      title={o.upcoming}
      actions={
        <Button variant="ghost" size="sm" asChild>
          <Link href="/erp/hr/recruitment/interviews">{o.seeAll}</Link>
        </Button>
      }
    >
      {!upcoming.data ? (
        <ErpLoadError error={upcoming.error} compact />
      ) : upcoming.data.length === 0 ? (
        <ErpEmpty compact title={rs.interviews.empty} />
      ) : (
        <Card className="divide-y p-0">
          {upcoming.data.slice(0, 8).map((iv) => (
            <Link
              key={iv.id}
              href={`/erp/hr/recruitment/candidates/${iv.candidateId}?tab=interviews`}
              className="flex flex-wrap items-center justify-between gap-2 px-4 py-3 text-sm hover:bg-accent/40"
            >
              <span className="min-w-0">
                <span className="font-medium">{iv.candidateName}</span>
                <span className="text-muted-foreground"> · {iv.jobOpeningTitle || rs.none}</span>
              </span>
              <span className="text-xs text-muted-foreground tabular">
                {formatDateTime(iv.interviewDate)} · {INTERVIEW_TYPE[iv.type] ?? iv.type} · {INTERVIEW_MODE[iv.mode] ?? iv.mode}
              </span>
            </Link>
          ))}
        </Card>
      )}
    </Section>
  );
}
