import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { OvertimeTable } from "@/features/erp/hr/components/overtime";
import { UrlTabs } from "@/features/erp/hr/components/ui";
import { hrPerms, monthParam } from "@/features/erp/hr/perms";
import { listDepartments, listEmployeeOptions, listOvertime } from "@/features/erp/hr/queries";
import { th } from "@/features/erp/hr/strings";

export const metadata: Metadata = { title: th.overtime.title };

const TABS = Object.entries(th.overtime.tabs).map(([value, label]) => ({ value, label }));

export default async function OvertimePage(props: PageProps<"/erp/hr/overtime">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("hr.read");
  const perms = hrPerms(access);
  const o = th.overtime;
  const approvals = sp1(sp.tab) === "approvals";
  const { ym } = monthParam(sp1(sp.month));
  const header = <PageHeader eyebrow={th.eyebrow} title={o.title} description={o.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  const [rows, employees, deps] = await Promise.all([
    load(() =>
      listOvertime(
        approvals
          ? { scope: "approvals", status: "pending" }
          : { month: ym, status: sp1(sp.status), departmentId: sp1(sp.dept), employeeId: sp1(sp.employee) },
      ),
    ),
    load(listEmployeeOptions),
    load(() => listDepartments()),
  ]);
  return (
    <>
      {header}
      <UrlTabs tabs={TABS} value={approvals ? "approvals" : "all"} />
      {rows.data ? (
        <OvertimeTable
          rows={rows.data}
          employees={employees.data ?? []}
          departments={(deps.data ?? []).map((d) => ({ value: d.id, label: d.name }))}
          canWrite={perms.write}
          canDecide={perms.approve}
          month={ym}
          approvals={approvals}
        />
      ) : (
        <ErpError message={rows.error.message} />
      )}
    </>
  );
}
