import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1, spInt } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { AccountsTable, ContactsTable } from "@/features/erp/sales/components/accounts";
import { LeadsTable } from "@/features/erp/sales/components/leads-table";
import { salesCaps } from "@/features/erp/sales/lib";
import { listAccounts, listContacts, listCustomFields, listLeads } from "@/features/erp/sales/queries";
import { ownerOptions } from "@/features/erp/sales/refs";
import { ss } from "@/features/erp/sales/strings";

const s = ss.leads;
export const metadata: Metadata = { title: s.title };

export default async function LeadsPage(props: PageProps<"/erp/sales/leads">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("sales.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const caps = salesCaps(access);
  const tab = sp1(sp.tab) ?? "leads";
  const page = spInt(sp.page, 1);
  const pageSize = spInt(sp.pageSize, 20);

  const [owners, accounts] = await Promise.all([ownerOptions(), load(() => listAccounts({ q: sp1(sp.aq), limit: 500 }))]);
  const accountOptions = (accounts.data ?? []).map((a) => ({ value: a.id, label: a.name }));

  let body: React.ReactNode;
  if (tab === "accounts") {
    body = accounts.data ? (
      <AccountsTable accounts={accounts.data} owners={owners} caps={caps} />
    ) : (
      <ErpError message={accounts.error?.message} />
    );
  } else if (tab === "contacts") {
    const contacts = await load(() => listContacts({ q: sp1(sp.cq), limit: 500 }));
    body = contacts.data ? (
      <ContactsTable contacts={contacts.data} accounts={accountOptions} caps={caps} />
    ) : (
      <ErpError message={contacts.error?.message} />
    );
  } else {
    const [leads, fields] = await Promise.all([
      load(() =>
        listLeads({
          q: sp1(sp.q),
          status: sp1(sp.status),
          temperature: sp1(sp.temperature),
          source: sp1(sp.source),
          ownerUserId: sp1(sp.owner),
          page,
          pageSize,
        }),
      ),
      load(() => listCustomFields("lead")),
    ]);
    body = leads.data ? (
      <LeadsTable
        leads={leads.data.leads}
        pagination={{ page, pageSize, total: leads.data.total }}
        owners={owners}
        accounts={accountOptions}
        customFields={fields.data ?? []}
        caps={caps}
      />
    ) : (
      <ErpError message={leads.error?.message} />
    );
  }

  return (
    <>
      <PageHeader eyebrow={ss.eyebrow} title={s.title} description={s.subtitle} />
      <UrlTabs
        label={s.title}
        tabs={[
          { value: "leads", label: s.tabs.leads },
          { value: "accounts", label: s.tabs.accounts },
          { value: "contacts", label: s.tabs.contacts },
        ]}
        reset={["q", "status", "temperature", "source", "owner", "page", "aq", "cq"]}
      />
      {body}
    </>
  );
}
