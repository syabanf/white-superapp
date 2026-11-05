import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { AssetDialog, AssetsTable, RecurringDialog, RecurringTable, RunSchedulesButton } from "@/features/erp/finance/components/schedules";
import { RecurringInvoiceDialog, RecurringInvoicesTable } from "@/features/erp/finance/components/recurring-invoices";
import { financePerms } from "@/features/erp/finance/perms";
import { listFixedAssets, listPostableAccounts, listProjectTags, listRecurringInvoices, listRecurringJournals, listTaxCodes } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { matchesActive, matchesText } from "@/features/erp/shared/list-filter";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { UrlTabs } from "@/features/erp/shared/url-tabs";

const s = fs.schedules;
export const metadata: Metadata = { title: s.title };

export default async function SchedulesPage(props: PageProps<"/erp/finance/schedules">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("finance.read");
  const { canApprove } = financePerms(access);
  const header = (actions?: React.ReactNode) => <PageHeader eyebrow={fs.eyebrow} title={s.title} description={s.subtitle} actions={actions} />;
  if (offline) return <>{header()}<OfflineNotice message={offline.message} /></>;

  const [q, status, frequency] = [sp1(sp.q), sp1(sp.status), sp1(sp.frequency)];
  const tab = sp1(sp.tab);
  const assetsTab = tab === "assets";
  const [postable, tagsRes] = await Promise.all([canApprove ? load(listPostableAccounts) : null, load(() => listProjectTags())]);
  const accounts = postable?.data ?? [];
  const tags = tagsRes.data ?? [];
  const tabs = (
    <UrlTabs
      label={s.title}
      tabs={[
        { value: "recurring", label: s.tabRecurring },
        { value: "invoices", label: fs.recurringInvoices.tab },
        { value: "assets", label: s.tabAssets },
      ]}
      reset={["q", "status", "frequency", "kind"]}
    />
  );
  if (tab === "invoices") {
    const kind = sp1(sp.kind);
    const [res, taxes] = await Promise.all([load(listRecurringInvoices), canApprove ? load(() => listTaxCodes(true)) : null]);
    const opts = { accounts, taxes: taxes?.data ?? [], tags };
    return (
      <>
        {header(canApprove ? <><RunSchedulesButton /><RecurringInvoiceDialog opts={opts} /></> : null)}
        {tabs}
        {res.error ? (
          <ErpLoadError error={res.error} />
        ) : (
          <RecurringInvoicesTable
            rows={res.data.filter((r) => matchesText(q, r.name, r.partyName) && matchesActive(status, r.isActive) && (!frequency || r.frequency === frequency) && (!kind || r.kind === kind))}
            opts={opts}
            canApprove={canApprove}
          />
        )}
      </>
    );
  }

  const actions = canApprove ? (
    <>
      <RunSchedulesButton />
      {assetsTab ? <AssetDialog accounts={accounts} /> : <RecurringDialog accounts={accounts} tags={tags} />}
    </>
  ) : null;

  if (assetsTab) {
    const res = await load(listFixedAssets);
    return (
      <>
        {header(actions)}
        {tabs}
        {res.error ? <ErpLoadError error={res.error} /> : <AssetsTable rows={res.data.filter((a) => matchesText(q, a.name) && matchesActive(status, a.isActive))} accounts={accounts} canApprove={canApprove} />}
      </>
    );
  }
  const res = await load(listRecurringJournals);
  return (
    <>
      {header(actions)}
      {tabs}
      {res.error ? <ErpLoadError error={res.error} /> : <RecurringTable rows={res.data.filter((r) => matchesText(q, r.name, r.description) && matchesActive(status, r.isActive) && (!frequency || r.frequency === frequency))} accounts={accounts} tags={tags} canApprove={canApprove} />}
    </>
  );
}
