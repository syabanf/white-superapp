import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { canAccess } from "@/components/shell/nav-config";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1, spInt } from "@/features/erp/shared/load";
import { MonthFilter } from "@/features/erp/shared/period-picker";
import { currentMonth } from "@/features/erp/shared/month";
import { ErpLoadError } from "@/features/erp/shared/states";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { ClaimsTable, InboxList } from "@/features/erp/finance/components/claims";
import { BatchesTable, MonthlySummaryView, PoliciesTable } from "@/features/erp/finance/components/reimburse-admin";
import { financePerms } from "@/features/erp/finance/perms";
import {
  getMonthlySummary,
  listApprovalInbox,
  listClaims,
  listPaymentBatches,
  listPolicies,
  listPostableAccounts,
} from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";
import { listDepartments } from "@/features/erp/hr/queries";

export const metadata: Metadata = { title: fs.reimburse.title };

const rb = fs.reimburse;

/**
 * Finance users get every tab. Managers without finance.read reach the page
 * for their approval inbox only (the backend scopes the inbox to their reports).
 */
export default async function ReimbursementsPage(props: PageProps<"/erp/finance/reimbursements">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp();
  const isFinance = canAccess(access, "finance.read");
  const perms = financePerms(access);
  const header = <PageHeader eyebrow={fs.eyebrow} title={rb.title} description={rb.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;

  const inbox = await load(listApprovalInbox);
  if (!isFinance) {
    if (!inbox.data?.claims.length) redirect("/erp?unauthorized=1");
    return (
      <>
        {header}
        <InboxList claims={inbox.data.claims} />
      </>
    );
  }

  const inboxCount = inbox.data?.claims.length ?? 0;
  const tab = sp1(sp.tab) ?? "inbox";
  const tabs = (
    <UrlTabs
      label={rb.title}
      tabs={[
        { value: "inbox", label: inboxCount ? `${rb.tabInbox} (${inboxCount})` : rb.tabInbox },
        { value: "all", label: rb.tabAll },
        { value: "batches", label: rb.tabBatches },
        { value: "summary", label: rb.tabSummary },
        { value: "policies", label: rb.tabPolicies },
      ]}
      reset={["page", "q", "status", "category", "month", "department", "from", "to"]}
    />
  );

  let body: React.ReactNode;
  if (tab === "all") {
    const page = spInt(sp.page, 1);
    const pageSize = spInt(sp.pageSize, 20);
    const department = sp1(sp.department);
    const [res, accounts, deps] = await Promise.all([
      load(() =>
        listClaims({
          search: sp1(sp.q),
          status: sp1(sp.status),
          category: sp1(sp.category),
          month: sp1(sp.month),
          department,
          dateFrom: sp1(sp.from),
          dateTo: sp1(sp.to),
          page,
          pageSize,
        }),
      ),
      perms.canApprove ? load(listPostableAccounts) : null,
      load(() => listDepartments()),
    ]);
    const departments = (deps.data ?? []).map((d) => d.name);
    if (department && !departments.includes(department)) departments.push(department);
    body = res.error ? (
      <ErpLoadError error={res.error} />
    ) : (
      <ClaimsTable
        claims={res.data.claims}
        totalAmount={res.data.totalAmount}
        page={{ page, pageSize, total: res.data.total }}
        canPay={perms.canApprove}
        accounts={accounts?.data ?? []}
        departments={departments}
      />
    );
  } else if (tab === "batches") {
    const res = await load(() => listPaymentBatches(100));
    body = res.error ? <ErpLoadError error={res.error} /> : <BatchesTable batches={res.data} />;
  } else if (tab === "summary") {
    const month = sp1(sp.month) ?? currentMonth();
    const res = await load(() => getMonthlySummary(month));
    body = (
      <>
        <MonthFilter fallback={month} />
        {res.error ? <ErpLoadError error={res.error} /> : <MonthlySummaryView summary={res.data} />}
      </>
    );
  } else if (tab === "policies") {
    const res = await load(() => Promise.all([listPolicies(), perms.canApprove ? listPostableAccounts() : Promise.resolve([])]));
    body = res.error ? (
      <ErpLoadError error={res.error} />
    ) : (
      <PoliciesTable policies={res.data[0]} accounts={res.data[1]} canEdit={perms.canApprove} />
    );
  } else {
    body = inbox.error ? <ErpLoadError error={inbox.error} /> : <InboxList claims={inbox.data.claims} />;
  }
  return (
    <>
      {header}
      {tabs}
      {body}
    </>
  );
}
