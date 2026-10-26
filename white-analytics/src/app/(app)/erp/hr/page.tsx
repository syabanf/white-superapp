import type { Metadata } from "next";
import { Suspense } from "react";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { DownloadButton } from "@/features/erp/shared/download-button";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { DrillBreadcrumb, type DrillCrumb } from "@/features/erp/shared/drill/drill-breadcrumb";
import { withParams } from "@/features/erp/shared/drill/href";
import { rangeLabel } from "@/features/erp/hr/dashboard/format";
import { HrFilterBar } from "@/features/erp/hr/components/dashboard/filter-bar";
import { HrDashboardTab, type HrDashTab } from "@/features/erp/hr/components/dashboard/dashboard-tabs";
import { DashboardBodySkeleton } from "@/features/erp/hr/components/dashboard/skeleton";
import { compareRange, resolveHrRange, type HrRange } from "@/features/erp/hr/dashboard/range";
import { getHrDashboard } from "@/features/erp/hr/dashboard/queries";
import { thd } from "@/features/erp/hr/dashboard/strings";
import { hrPerms, todayWib } from "@/features/erp/hr/perms";
import { listDepartments, listEmploymentStatuses, listLeaveTypes, listPositions } from "@/features/erp/hr/queries";
import { th } from "@/features/erp/hr/strings";

export const metadata: Metadata = { title: thd.title };

type Dims = { departmentId?: string; employmentStatus?: string; positionId?: string };

export default async function HrDashboardPage(props: PageProps<"/erp/hr">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("hr.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const perms = hrPerms(access);
  const range = resolveHrRange(
    { preset: sp1(sp.preset), from: sp1(sp.from), to: sp1(sp.to), compare: sp1(sp.compare) },
    todayWib(),
  );
  const cmp = compareRange(range);
  const dims: Dims = { departmentId: sp1(sp.dept), employmentStatus: sp1(sp.status), positionId: sp1(sp.position) };
  const tabs = Object.entries(thd.tabs)
    .filter(([v]) => v !== "payroll" || perms.payroll)
    .map(([value, label]) => ({ value, label }));
  const tab = (tabs.find((t) => t.value === sp1(sp.tab))?.value ?? "summary") as HrDashTab;

  const [depts, positions, statuses, leaveTypes] = await Promise.all([
    load(() => listDepartments()),
    load(() => listPositions()),
    load(listEmploymentStatuses),
    load(listLeaveTypes),
  ]);
  const leaveNames = Object.fromEntries((leaveTypes.data ?? []).map((t) => [t.code, t.name]));
  const [year, month] = range.to.split("-").map(Number);

  return (
    <>
      <PageHeader
        eyebrow={th.eyebrow}
        title={thd.title}
        description={thd.subtitle}
        actions={
          <DownloadButton href={`/api/erp/v1/hr/reports/monthly:export?month=${month}&year=${year}`}>{thd.exportReport}</DownloadButton>
        }
      />
      <HrFilterBar
        range={range}
        compareFrom={cmp.from}
        compareTo={cmp.to}
        departments={(depts.data ?? []).map((d) => ({ value: d.id, label: d.name }))}
        statuses={(statuses.data ?? []).filter((s) => s.isActive).map((s) => ({ value: s.code, label: s.name }))}
        positions={(positions.data ?? []).map((p) => ({ value: p.id, label: p.title }))}
      />
      <DrillBreadcrumb
        items={drillTrail(sp, range, (depts.data ?? []).find((d) => d.id === dims.departmentId)?.name)}
      />
      <UrlTabs tabs={tabs} label={thd.title} />
      <Suspense key={JSON.stringify([range, dims, tab])} fallback={<DashboardBodySkeleton />}>
        <DashboardBody tab={tab} range={range} dims={dims} leaveTypes={leaveNames} />
      </Suspense>
    </>
  );
}

/**
 * "Semua > 1 Okt – 31 Okt 2026 > Creative": a zoomed range (`back` holds the
 * range it came from) and the department filter, each level linking up.
 */
function drillTrail(sp: Record<string, string | string[] | undefined>, range: HrRange, deptName?: string): DrillCrumb[] {
  const back = sp1(sp.back)?.split("_");
  const top = back?.length === 2 ? { from: back[0], to: back[1], back: null } : {};
  const items: DrillCrumb[] = [{ label: thd.drill.all, href: withParams("/erp/hr", sp, { ...top, dept: null }, ["drill"]) }];
  if (back) items.push({ label: rangeLabel(range.from, range.to), href: withParams("/erp/hr", sp, { dept: null }, ["drill"]) });
  if (deptName) items.push({ label: deptName });
  return items;
}

async function DashboardBody({
  tab,
  range,
  dims,
  leaveTypes,
}: {
  tab: HrDashTab;
  range: HrRange;
  dims: Dims;
  leaveTypes: Record<string, string>;
}) {
  const res = await load(() => getHrDashboard({ from: range.from, to: range.to, compare: range.compare, ...dims }));
  if (!res.data) {
    return res.error.code === "NETWORK" ? <OfflineNotice message={res.error.message} /> : <ErpError message={res.error.message} />;
  }
  return <HrDashboardTab tab={tab} data={res.data} leaveTypes={leaveTypes} />;
}
