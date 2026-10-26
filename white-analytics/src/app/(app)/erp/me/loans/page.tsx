import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { MyLoansTable } from "@/features/erp/me/components/lists";
import { listMyLoans } from "@/features/erp/me/queries";
import { tm } from "@/features/erp/me/strings";

export const metadata: Metadata = { title: tm.loans.title };

export default async function MyLoansPage() {
  const { offline } = await requireErp("self.service");
  const header = <PageHeader eyebrow={tm.eyebrow} title={tm.loans.title} description={tm.loans.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  const res = await load(listMyLoans);
  return (
    <>
      {header}
      {res.data ? <MyLoansTable loans={res.data} /> : <ErpError message={res.error.message} />}
    </>
  );
}
