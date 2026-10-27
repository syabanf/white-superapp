import type { Metadata } from "next";
import { Toolbar } from "@/components/dashboard/layout";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1, spInt } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { SelectFilter } from "@/features/erp/shared/url-filters";
import { ContractsTable, ExpiringPanel } from "@/features/erp/hr/components/contracts";
import { UrlTabs } from "@/features/erp/hr/components/ui";
import { hrPerms } from "@/features/erp/hr/perms";
import { listContracts, listEmployeeOptions, listExpiringContracts } from "@/features/erp/hr/queries";
import { th } from "@/features/erp/hr/strings";

export const metadata: Metadata = { title: th.contracts.title };

const TABS = Object.entries(th.contracts.tabs).map(([value, label]) => ({ value, label }));

export default async function ContractsPage(props: PageProps<"/erp/hr/contracts">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("hr.read");
  const c = th.contracts;
  const tab = sp1(sp.tab) === "expiring" ? "expiring" : "list";
  const header = <PageHeader eyebrow={th.eyebrow} title={c.title} description={c.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;

  if (tab === "expiring") {
    const days = Math.min(90, spInt(sp.days, 30));
    const res = await load(() => listExpiringContracts(days));
    return (
      <>
        {header}
        <UrlTabs tabs={TABS} value={tab} />
        <Toolbar label={c.window}>
          <span className="text-sm text-muted-foreground">{c.window}</span>
          <SelectFilter
            param="days"
            label={c.window}
            allLabel={c.windowDays(30)}
            options={[7, 14, 60, 90].map((d) => ({ value: String(d), label: c.windowDays(d) }))}
          />
        </Toolbar>
        {res.data ? <ExpiringPanel data={res.data} /> : <ErpError message={res.error.message} />}
      </>
    );
  }

  const [list, employees] = await Promise.all([
    load(() =>
      listContracts({ status: sp1(sp.status), contractType: sp1(sp.type), search: sp1(sp.q) }),
    ),
    load(listEmployeeOptions),
  ]);
  return (
    <>
      {header}
      <UrlTabs tabs={TABS} value={tab} />
      {list.data ? (
        <ContractsTable contracts={list.data} employees={employees.data ?? []} canWrite={hrPerms(access).write} />
      ) : (
        <ErpError message={list.error.message} />
      )}
    </>
  );
}
