import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { AssetDialog, AssetsTable, RecurringDialog, RecurringTable, RunSchedulesButton } from "@/features/erp/finance/components/schedules";
import { financePerms } from "@/features/erp/finance/perms";
import { listFixedAssets, listPostableAccounts, listRecurringJournals } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";
import { OfflineNotice } from "@/features/erp/offline-notice";
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

  const assetsTab = sp1(sp.tab) === "assets";
  const postable = canApprove ? await load(listPostableAccounts) : null;
  const accounts = postable?.data ?? [];
  const tabs = (
    <UrlTabs
      label={s.title}
      tabs={[
        { value: "recurring", label: s.tabRecurring },
        { value: "assets", label: s.tabAssets },
      ]}
    />
  );
  const actions = canApprove ? (
    <>
      <RunSchedulesButton />
      {assetsTab ? <AssetDialog accounts={accounts} /> : <RecurringDialog accounts={accounts} />}
    </>
  ) : null;

  if (assetsTab) {
    const res = await load(listFixedAssets);
    return (
      <>
        {header(actions)}
        {tabs}
        {res.error ? <ErpLoadError error={res.error} /> : <AssetsTable rows={res.data} canApprove={canApprove} />}
      </>
    );
  }
  const res = await load(listRecurringJournals);
  return (
    <>
      {header(actions)}
      {tabs}
      {res.error ? <ErpLoadError error={res.error} /> : <RecurringTable rows={res.data} canApprove={canApprove} />}
    </>
  );
}
