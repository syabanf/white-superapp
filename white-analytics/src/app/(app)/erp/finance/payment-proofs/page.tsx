import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { PaymentProofsTable } from "@/features/erp/finance/components/payment-proofs";
import { financePerms } from "@/features/erp/finance/perms";
import { listPaymentProofs } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";

const s = fs.proofs;
export const metadata: Metadata = { title: s.title };

export default async function PaymentProofsPage(props: PageProps<"/erp/finance/payment-proofs">) {
  const { access, offline } = await requireErp("finance.read");
  const { canWrite } = financePerms(access);
  const sp = await props.searchParams;
  // Pending by default; "all" shows every status.
  const status = sp.status === "all" ? "" : "PENDING";
  const header = <PageHeader eyebrow={fs.eyebrow} title={s.title} description={s.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  const res = await load(() => listPaymentProofs(status));
  return (
    <>
      {header}
      {res.error ? <ErpLoadError error={res.error} /> : <PaymentProofsTable rows={res.data} canWrite={canWrite} status={status} />}
    </>
  );
}
