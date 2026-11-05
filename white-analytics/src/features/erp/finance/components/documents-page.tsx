import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1, spInt } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { financePerms } from "../perms";
import { listDocuments, listVendors } from "../queries";
import { fs } from "../strings";
import type { DocumentKind } from "../types";
import { DocumentDialog, DocumentsTable } from "./documents";

const s = fs.documents;
type SearchParams = Record<string, string | string[] | undefined>;

/** One list page for two document pairs: PR and PO, quotation and invoice. */
export async function DocumentsListPage({ kinds, title, subtitle, sp }: { kinds: [DocumentKind, DocumentKind]; title: string; subtitle: string; sp: SearchParams }) {
  const { access, offline } = await requireErp("finance.read");
  const { canWrite } = financePerms(access);
  const kind = kinds.find((k) => k === sp1(sp.tab)) ?? kinds[0];
  const vendors = await load(() => listVendors({ activeOnly: true }));
  const header = <PageHeader eyebrow={fs.eyebrow} title={title} description={subtitle} actions={canWrite && !offline ? <DocumentDialog kind={kind} vendors={vendors.data ?? []} /> : null} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  const page = spInt(sp.page, 1);
  const pageSize = spInt(sp.pageSize, 25);
  const res = await load(() => listDocuments({ kind, status: sp1(sp.status), q: sp1(sp.q), page, pageSize }));
  return (
    <>
      {header}
      <UrlTabs label={title} tabs={kinds.map((value) => ({ value, label: s.kinds[value] }))} reset={["page", "status", "q"]} />
      {res.error ? <ErpLoadError error={res.error} /> : <DocumentsTable documents={res.data.documents ?? []} kind={kind} page={{ page, pageSize, total: res.data.total }} />}
    </>
  );
}
