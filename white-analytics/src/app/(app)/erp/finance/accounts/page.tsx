import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { AccountTypesTable } from "@/features/erp/finance/components/account-types";
import { CoaTree } from "@/features/erp/finance/components/coa-tree";
import { JournalMappingsTable } from "@/features/erp/finance/components/journal-mappings";
import { financePerms } from "@/features/erp/finance/perms";
import {
  listAccountTypes,
  listAccounts,
  listJournalEvents,
  listJournalMappings,
  listPostableAccounts,
} from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";

export const metadata: Metadata = { title: fs.accounts.title };

const a = fs.accounts;

export default async function AccountsPage(props: PageProps<"/erp/finance/accounts">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("finance.read");
  const { canWrite } = financePerms(access);
  const tab = sp1(sp.tab) ?? "coa";
  const header = <PageHeader eyebrow={fs.eyebrow} title={a.title} description={a.subtitle} />;
  const tabs = (
    <UrlTabs
      label={a.title}
      tabs={[
        { value: "coa", label: a.tabCoa },
        { value: "types", label: a.tabTypes },
        { value: "mappings", label: a.tabMappings },
      ]}
    />
  );
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;

  let body: React.ReactNode;
  if (tab === "types") {
    const res = await load(() => listAccountTypes(true));
    body = res.error ? <ErpLoadError error={res.error} /> : <AccountTypesTable types={res.data} canWrite={canWrite} />;
  } else if (tab === "mappings") {
    const res = await load(() => Promise.all([listJournalMappings(), listJournalEvents(), listPostableAccounts()]));
    body = res.error ? (
      <ErpLoadError error={res.error} />
    ) : (
      <JournalMappingsTable mappings={res.data[0]} events={res.data[1]} accounts={res.data[2]} canWrite={canWrite} />
    );
  } else {
    const res = await load(() => Promise.all([listAccounts(), listAccountTypes(false)]));
    body = res.error ? (
      <ErpLoadError error={res.error} />
    ) : (
      <CoaTree accounts={res.data[0]} accountTypes={res.data[1]} canWrite={canWrite} />
    );
  }
  return (
    <>
      {header}
      {tabs}
      {body}
    </>
  );
}
