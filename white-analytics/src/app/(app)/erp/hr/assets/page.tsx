import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { hrPerms, todayWib } from "@/features/erp/hr/perms";
import { listEmployeeOptions } from "@/features/erp/hr/queries";
import { th } from "@/features/erp/hr/strings";
import { AssetsAdmin } from "@/features/erp/hr/workplace/assets";
import { listAssets } from "@/features/erp/hr/workplace/queries";
import { tw } from "@/features/erp/hr/workplace/strings";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";

export const metadata: Metadata = { title: tw.assets.title };

export default async function AssetsPage() {
  const { access, offline } = await requireErp("hr.read");
  const header = <PageHeader eyebrow={th.eyebrow} title={tw.assets.title} description={tw.assets.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  const [assets, employees] = await Promise.all([load(() => listAssets()), load(listEmployeeOptions)]);
  return (
    <>
      {header}
      {assets.data ? (
        <AssetsAdmin assets={assets.data} employees={employees.data ?? []} canWrite={hrPerms(access).write} today={todayWib()} />
      ) : (
        <ErpError message={assets.error.message} />
      )}
    </>
  );
}
