import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { VendorDialog, VendorsTable } from "@/features/erp/finance/components/vendors";
import { financePerms } from "@/features/erp/finance/perms";
import { listVendors } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";

const s = fs.vendors;
export const metadata: Metadata = { title: s.title };

export default async function VendorsPage(props: PageProps<"/erp/finance/vendors">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("finance.read");
  const { canWrite } = financePerms(access);
  const header = <PageHeader eyebrow={fs.eyebrow} title={s.title} description={s.subtitle} actions={canWrite && !offline ? <VendorDialog /> : null} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  const res = await load(() => listVendors({ q: sp1(sp.q) }));
  return (
    <>
      {header}
      {res.error ? <ErpLoadError error={res.error} /> : <VendorsTable vendors={res.data} canWrite={canWrite} />}
    </>
  );
}
