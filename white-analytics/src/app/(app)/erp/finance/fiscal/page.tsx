import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { FiscalYears } from "@/features/erp/finance/components/fiscal-years";
import { financePerms } from "@/features/erp/finance/perms";
import { listFiscalYears } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";

export const metadata: Metadata = { title: fs.fiscal.title };

export default async function FiscalPage() {
  const { access, offline } = await requireErp("finance.read");
  const header = <PageHeader eyebrow={fs.eyebrow} title={fs.fiscal.title} description={fs.fiscal.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  const res = await load(listFiscalYears);
  return (
    <>
      {header}
      {res.error ? <ErpLoadError error={res.error} /> : <FiscalYears years={res.data} perms={financePerms(access)} />}
    </>
  );
}
