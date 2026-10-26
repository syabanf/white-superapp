import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1, spInt } from "@/features/erp/shared/load";
import { MonthFilter } from "@/features/erp/shared/period-picker";
import { ErpError } from "@/features/erp/shared/states";
import { LoansTable } from "@/features/erp/hr/components/loans";
import { AdditionsTable, SalariesTable, SettingsForm } from "@/features/erp/hr/components/payroll-data";
import { RunsTable } from "@/features/erp/hr/components/payroll-runs";
import { UrlTabs } from "@/features/erp/hr/components/ui";
import { hrPerms, monthParam } from "@/features/erp/hr/perms";
import {
  getPayrollSettings,
  listEmployeeOptions,
  listLoans,
  listPayrollAdditions,
  listPayrollRuns,
  listSalaries,
} from "@/features/erp/hr/queries";
import { th } from "@/features/erp/hr/strings";

export const metadata: Metadata = { title: th.payroll.title };

const TABS = Object.entries(th.payroll.tabs).map(([value, label]) => ({ value, label }));

type SP = Record<string, string | string[] | undefined>;

export default async function PayrollPage(props: PageProps<"/erp/hr/payroll">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("hr.payroll");
  const p = th.payroll;
  const tab = TABS.some((t) => t.value === sp1(sp.tab)) ? sp1(sp.tab)! : "runs";
  return (
    <>
      <PageHeader eyebrow={th.eyebrow} title={p.title} description={p.subtitle} />
      {offline ? (
        <OfflineNotice message={offline.message} />
      ) : (
        <>
          <UrlTabs tabs={TABS} value={tab} />
          <Body tab={tab} sp={sp} perms={hrPerms(access)} />
        </>
      )}
    </>
  );
}

async function Body({ tab, sp, perms }: { tab: string; sp: SP; perms: ReturnType<typeof hrPerms> }) {
  switch (tab) {
    case "salaries": {
      const [list, emps] = await Promise.all([
        load(() => listSalaries({ activeOnly: sp1(sp.all) !== "1" })),
        load(listEmployeeOptions),
      ]);
      return list.data ? (
        <SalariesTable salaries={list.data} employees={emps.data ?? []} canWrite={perms.payroll} />
      ) : (
        <ErpError message={list.error.message} />
      );
    }
    case "additions": {
      const { ym, month, year } = monthParam(sp1(sp.month));
      const [list, emps] = await Promise.all([
        load(() => listPayrollAdditions({ periodMonth: month, periodYear: year })),
        load(listEmployeeOptions),
      ]);
      return (
        <>
          <MonthFilter fallback={ym} />
          {list.data ? (
            <AdditionsTable additions={list.data} employees={emps.data ?? []} canWrite={perms.payroll} />
          ) : (
            <ErpError message={list.error.message} />
          )}
        </>
      );
    }
    case "loans": {
      const [list, emps] = await Promise.all([load(() => listLoans({ status: sp1(sp.status) })), load(listEmployeeOptions)]);
      return list.data ? (
        <LoansTable loans={list.data} employees={emps.data ?? []} canWrite={perms.payroll} canDecide={perms.approve} />
      ) : (
        <ErpError message={list.error.message} />
      );
    }
    case "settings": {
      const res = await load(getPayrollSettings);
      return res.data ? <SettingsForm settings={res.data} canWrite={perms.payroll} /> : <ErpError message={res.error.message} />;
    }
    default: {
      const year = spInt(sp.year, new Date().getFullYear());
      const res = await load(() => listPayrollRuns({ year }));
      return res.data ? <RunsTable runs={res.data} year={year} canWrite={perms.payroll} /> : <ErpError message={res.error.message} />;
    }
  }
}
