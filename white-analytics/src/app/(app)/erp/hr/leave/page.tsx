import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1, spInt } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { BalancesTable, LeaveTable } from "@/features/erp/hr/components/leave";
import { UrlTabs } from "@/features/erp/hr/components/ui";
import { hrPerms } from "@/features/erp/hr/perms";
import { getLeaveBalance, listEmployeeOptions, listLeaveTypes, listLeaves } from "@/features/erp/hr/queries";
import { th } from "@/features/erp/hr/strings";

export const metadata: Metadata = { title: th.leave.title };

const TABS = Object.entries(th.leave.tabs).map(([value, label]) => ({ value, label }));

export default async function LeavePage(props: PageProps<"/erp/hr/leave">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("hr.read");
  const perms = hrPerms(access);
  const l = th.leave;
  const tab = TABS.some((t) => t.value === sp1(sp.tab)) ? sp1(sp.tab)! : "requests";
  const header = <PageHeader eyebrow={th.eyebrow} title={l.title} description={l.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;

  const [types, employees] = await Promise.all([load(listLeaveTypes), load(listEmployeeOptions)]);
  let body: React.ReactNode;

  if (tab === "balances") {
    const year = new Date().getFullYear();
    // No bulk endpoint: one balance read per active employee.
    const rows = await Promise.all(
      (employees.data ?? []).map(async (employee) => {
        const res = await load(() => getLeaveBalance(employee.id, year));
        return res.data ? { employee, balance: res.data } : null;
      }),
    );
    body = <BalancesTable rows={rows.filter((r) => r !== null)} />;
  } else {
    const approvals = tab === "approvals";
    const page = spInt(sp.page, 1);
    const pageSize = spInt(sp.pageSize, 20);
    const filters = { status: sp1(sp.status), type: sp1(sp.type), from: sp1(sp.from), to: sp1(sp.to) };
    const res = await load(() =>
      listLeaves(
        approvals
          ? { scope: "approvals", status: "pending", pageSize: 100 }
          : {
              status: filters.status,
              leaveType: filters.type,
              startDate: filters.from,
              endDate: filters.to,
              page,
              pageSize,
            },
      ),
    );
    body = res.data ? (
      <LeaveTable
        leaves={res.data.leaves}
        page={approvals ? undefined : { page, pageSize, total: res.data.total }}
        types={types.data ?? []}
        employees={employees.data ?? []}
        canWrite={perms.write}
        filters={filters}
        approvals={approvals}
      />
    ) : (
      <ErpError message={res.error.message} />
    );
  }

  return (
    <>
      {header}
      <UrlTabs tabs={TABS} value={tab} />
      {body}
    </>
  );
}
