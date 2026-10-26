import type { Metadata } from "next";
import Link from "next/link";
import { UserPlus } from "lucide-react";
import { PageHeader } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { EmployeesTable } from "@/features/erp/hr/components/employees-table";
import { DepartmentsPanel, PositionsPanel, StatusesPanel } from "@/features/erp/hr/components/masters";
import { OrgChartTree } from "@/features/erp/hr/components/org-chart";
import { UrlTabs } from "@/features/erp/hr/components/ui";
import { hrPerms } from "@/features/erp/hr/perms";
import {
  getOrgChart,
  listDepartments,
  listEmployeeOptions,
  listEmployees,
  listEmploymentStatuses,
  listPositions,
} from "@/features/erp/hr/queries";
import { th } from "@/features/erp/hr/strings";

export const metadata: Metadata = { title: th.employees.title };

const TABS = Object.entries(th.employees.tabs).map(([value, label]) => ({ value, label }));

export default async function EmployeesPage(props: PageProps<"/erp/hr/employees">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("hr.read");
  const perms = hrPerms(access);
  const tab = TABS.some((t) => t.value === sp1(sp.tab)) ? sp1(sp.tab)! : "list";
  const e = th.employees;

  return (
    <>
      <PageHeader
        eyebrow={th.eyebrow}
        title={e.title}
        description={e.subtitle}
        actions={
          perms.write ? (
            <Button size="sm" asChild>
              <Link href="/erp/hr/employees/new">
                <UserPlus className="size-4" /> {e.newEmployee}
              </Link>
            </Button>
          ) : null
        }
      />
      {offline ? <OfflineNotice message={offline.message} /> : <UrlTabs tabs={TABS} value={tab} />}
      {offline ? null : <TabBody tab={tab} sp={sp} canWrite={perms.write} />}
    </>
  );
}

async function TabBody({
  tab,
  sp,
  canWrite,
}: {
  tab: string;
  sp: Record<string, string | string[] | undefined>;
  canWrite: boolean;
}) {
  if (tab === "org") {
    const res = await load(getOrgChart);
    return res.data ? <OrgChartTree chart={res.data} /> : <ErpError message={res.error.message} />;
  }
  if (tab === "departments") {
    const [deps, emps] = await Promise.all([load(() => listDepartments(true)), load(listEmployeeOptions)]);
    if (!deps.data) return <ErpError message={deps.error.message} />;
    return <DepartmentsPanel departments={deps.data} employees={emps.data ?? []} canWrite={canWrite} />;
  }
  if (tab === "positions") {
    const [pos, deps] = await Promise.all([load(() => listPositions(true)), load(() => listDepartments())]);
    if (!pos.data) return <ErpError message={pos.error.message} />;
    return <PositionsPanel positions={pos.data} departments={deps.data ?? []} canWrite={canWrite} />;
  }
  if (tab === "statuses") {
    const res = await load(listEmploymentStatuses);
    return res.data ? <StatusesPanel statuses={res.data} canWrite={canWrite} /> : <ErpError message={res.error.message} />;
  }
  const [list, deps] = await Promise.all([
    load(() =>
      listEmployees({
        search: sp1(sp.q),
        departmentId: sp1(sp.dept),
        includeInactive: sp1(sp.inactive) === "1",
        pageSize: 500,
      }),
    ),
    load(() => listDepartments()),
  ]);
  if (!list.data) return <ErpError message={list.error.message} />;
  return (
    <EmployeesTable
      employees={list.data.employees}
      departments={(deps.data ?? []).map((d) => ({ value: d.id, label: d.name }))}
    />
  );
}
