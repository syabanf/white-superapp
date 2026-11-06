import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { PoMatchTable } from "@/features/erp/finance/components/controls";
import { financePerms } from "@/features/erp/finance/perms";
import { listInvoices, listPoMatches } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";

const s = fs.controls;
export const metadata: Metadata = { title: s.poTitle };

export default async function PoMatchingPage() {
  const { access, offline } = await requireErp("finance.read");
  const { canWrite, canApprove } = financePerms(access);
  const header = <PageHeader eyebrow={fs.eyebrow} title={s.poTitle} description={s.poSubtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  const res = await load(() => Promise.all([listPoMatches(), listInvoices("AP", { status: "DRAFT", pageSize: 200 })]));
  return (
    <>
      {header}
      {res.error ? <ErpLoadError error={res.error} /> : <PoMatchTable rows={res.data[0]} drafts={res.data[1].invoices ?? []} canWrite={canWrite} canApprove={canApprove} />}
    </>
  );
}
