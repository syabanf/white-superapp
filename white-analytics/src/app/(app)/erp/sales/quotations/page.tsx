import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { SelectFilter } from "@/features/erp/shared/url-filters";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { CatalogSettings } from "@/features/erp/sales/components/catalog";
import { ApprovalsInbox, QuotationsTable } from "@/features/erp/sales/components/quotations";
import { salesCaps } from "@/features/erp/sales/lib";
import {
  listApprovalRules,
  listApprovals,
  listCatalog,
  listDeals,
  listQuotations,
  listRetainerDiscounts,
} from "@/features/erp/sales/queries";
import { ownerOptions } from "@/features/erp/sales/refs";
import { ss } from "@/features/erp/sales/strings";

const s = ss.quotations;
export const metadata: Metadata = { title: s.title };

export default async function QuotationsPage(props: PageProps<"/erp/sales/quotations">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("sales.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const caps = salesCaps(access);
  const tab = sp1(sp.tab) ?? "list";

  let body: React.ReactNode;
  if (tab === "approvals") {
    const view = sp1(sp.view) ?? "mine";
    const status = sp1(sp.astatus) ?? "pending";
    const [requests, owners] = await Promise.all([load(() => listApprovals({ view, status })), ownerOptions()]);
    body = (
      <>
        <div className="flex flex-wrap gap-2">
          <SelectFilter
            param="view"
            label={ss.approvals.mine}
            allLabel={ss.approvals.mine}
            options={[{ value: "all", label: ss.approvals.all }]}
          />
          <SelectFilter
            param="astatus"
            label={s.status}
            allLabel={ss.approvals.statusFilter.pending}
            options={["approved", "rejected", "all"].map((v) => ({ value: v, label: ss.approvals.statusFilter[v] }))}
          />
        </div>
        {requests.data ? <ApprovalsInbox requests={requests.data} owners={owners} /> : <ErpError message={requests.error?.message} />}
      </>
    );
  } else if (tab === "catalog") {
    const [catalog, retainer, rules, owners] = await Promise.all([
      load(() => listCatalog(caps.admin)),
      load(listRetainerDiscounts),
      load(listApprovalRules),
      ownerOptions(),
    ]);
    body = catalog.data ? (
      <CatalogSettings
        catalog={catalog.data}
        retainer={retainer.data ?? []}
        rules={rules.data ?? []}
        owners={owners}
        admin={caps.admin}
      />
    ) : (
      <ErpError message={catalog.error?.message} />
    );
  } else {
    const [quotations, deals] = await Promise.all([
      load(() => listQuotations({ status: sp1(sp.status), limit: 500 })),
      load(() => listDeals({ status: "open", pageSize: 500 })),
    ]);
    const dealOptions = (deals.data?.deals ?? []).map((d) => ({ value: d.id, label: `${d.title} · ${d.clientName}` }));
    body = quotations.data ? (
      <QuotationsTable quotations={quotations.data} caps={caps} deals={dealOptions} />
    ) : (
      <ErpError message={quotations.error?.message} />
    );
  }

  return (
    <>
      <PageHeader eyebrow={ss.eyebrow} title={s.title} description={s.subtitle} />
      <UrlTabs
        label={s.title}
        tabs={[
          { value: "list", label: s.tabs.list },
          { value: "approvals", label: s.tabs.approvals },
          { value: "catalog", label: s.tabs.catalog },
        ]}
        reset={["status", "view", "astatus"]}
      />
      {body}
    </>
  );
}
