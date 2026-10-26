import type { Metadata } from "next";
import { Suspense } from "react";
import { redirect } from "next/navigation";
import { PageHeader } from "@/components/dashboard/page-header";
import { canAccess } from "@/components/shell/nav-config";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { resolvePeriod } from "@/features/erp/shared/dashboard-period";
import { load, sp1 } from "@/features/erp/shared/load";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { SectionError } from "@/features/erp/performance/components/section-error";
import { DashboardFilterBar } from "@/features/erp/performance/dashboard/filter-bar";
import { ImprovementsTab } from "@/features/erp/performance/dashboard/improvements-tab";
import { KpiTab } from "@/features/erp/performance/dashboard/kpi-tab";
import { LogbookTab } from "@/features/erp/performance/dashboard/logbook-tab";
import { ReviewsTab } from "@/features/erp/performance/dashboard/reviews-tab";
import { TabSkeleton } from "@/features/erp/performance/dashboard/skeleton";
import { ds } from "@/features/erp/performance/dashboard/strings";
import { SummaryTab } from "@/features/erp/performance/dashboard/summary-tab";
import { TasksTab } from "@/features/erp/performance/dashboard/tasks-tab";
import type {
  DashContext,
  DashImprovements,
  DashKpi,
  DashLogbook,
  DashQuery,
  DashReviews,
  DashSummary,
  DashTasks,
} from "@/features/erp/performance/dashboard/types";
import { getDashboard } from "@/features/erp/performance/queries";
import { DashboardTrail, type TrailLevel } from "@/features/erp/sales/dashboard/trail";
import { sp } from "@/features/erp/performance/strings";

export const metadata: Metadata = { title: ds.title };

const TABS = ["summary", "kpi", "tasks", "logbook", "reviews", "improvements"] as const;
type Tab = (typeof TABS)[number];

export default async function PerformanceDashboardPage(props: PageProps<"/erp/performance">) {
  const q = await props.searchParams;
  // hr.read sees the company and teams; self.service alone sees "me".
  const { access, offline } = await requireErp();
  if (!canAccess(access, "hr.read") && !canAccess(access, "self.service")) redirect("/erp?unauthorized=1");
  const header = <PageHeader eyebrow={sp.common.eyebrow} title={ds.title} description={ds.subtitle} />;
  if (offline)
    return (
      <>
        {header}
        <OfflineNotice message={offline.message} />
      </>
    );

  const period = resolvePeriod(
    { preset: sp1(q.preset), from: sp1(q.from), to: sp1(q.to), compare: sp1(q.compare) },
    "thisMonth",
  );
  const query: DashQuery = {
    from: period.from,
    to: period.to,
    compare: period.compare,
    departmentId: sp1(q.dept),
    employeeId: sp1(q.emp),
    scope: sp1(q.scope),
  };
  const ctx = await load(() => getDashboard<DashContext>("context", query));
  if (ctx.error)
    return (
      <>
        {header}
        <SectionError error={ctx.error} />
      </>
    );
  const tab: Tab = TABS.find((t) => t === sp1(q.tab)) ?? "summary";
  const trail: TrailLevel[] = [
    period.preset === "custom" ? { label: period.label, params: ["preset", "from", "to"] } : null,
    query.departmentId
      ? { label: ctx.data.departments.find((d) => d.id === query.departmentId)?.name ?? query.departmentId, params: ["dept", "emp"] }
      : null,
    query.employeeId ? { label: ctx.data.people.find((p) => p.id === query.employeeId)?.name ?? query.employeeId, params: ["emp"] } : null,
  ].filter((l): l is TrailLevel => l != null);

  return (
    <>
      {header}
      <DashboardFilterBar context={ctx.data} preset={period.preset} />
      <DashboardTrail levels={trail} />
      <UrlTabs label={ds.title} tabs={TABS.map((t) => ({ value: t, label: ds.tabs[t] }))} />
      <Suspense key={`${tab}|${JSON.stringify(query)}`} fallback={<TabSkeleton />}>
        <TabSection tab={tab} query={query} />
      </Suspense>
    </>
  );
}

async function TabSection({ tab, query }: { tab: Tab; query: DashQuery }) {
  const dept = query.departmentId;
  switch (tab) {
    case "kpi": {
      const r = await load(() => getDashboard<DashKpi>("kpi", query));
      return r.error ? <SectionError error={r.error} /> : <KpiTab data={r.data} />;
    }
    case "tasks": {
      const r = await load(() => getDashboard<DashTasks>("tasks", query));
      return r.error ? <SectionError error={r.error} /> : <TasksTab data={r.data} dept={dept} />;
    }
    case "logbook": {
      const r = await load(() => getDashboard<DashLogbook>("logbook", query));
      return r.error ? <SectionError error={r.error} /> : <LogbookTab data={r.data} dept={dept} />;
    }
    case "reviews": {
      const r = await load(() => getDashboard<DashReviews>("reviews", query));
      return r.error ? <SectionError error={r.error} /> : <ReviewsTab data={r.data} />;
    }
    case "improvements": {
      const r = await load(() => getDashboard<DashImprovements>("improvements", query));
      return r.error ? <SectionError error={r.error} /> : <ImprovementsTab data={r.data} dept={dept} />;
    }
    default: {
      const r = await load(() => getDashboard<DashSummary>("summary", query));
      return r.error ? <SectionError error={r.error} /> : <SummaryTab data={r.data} dept={dept} />;
    }
  }
}
