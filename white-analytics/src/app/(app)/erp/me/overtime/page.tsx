import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { MonthFilter } from "@/features/erp/shared/period-picker";
import { ErpError } from "@/features/erp/shared/states";
import { monthParam } from "@/features/erp/hr/perms";
import { MyOvertimeTable } from "@/features/erp/me/components/lists";
import { listMyOvertime } from "@/features/erp/me/queries";
import { tm } from "@/features/erp/me/strings";

export const metadata: Metadata = { title: tm.overtime.title };

export default async function MyOvertimePage(props: PageProps<"/erp/me/overtime">) {
  const sp = await props.searchParams;
  const { offline } = await requireErp("self.service");
  const header = <PageHeader eyebrow={tm.eyebrow} title={tm.overtime.title} description={tm.overtime.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  const { ym } = monthParam(sp1(sp.month));
  const res = await load(() => listMyOvertime(ym));
  return (
    <>
      {header}
      <MonthFilter fallback={ym} />
      {res.data ? <MyOvertimeTable rows={res.data} /> : <ErpError message={res.error.message} />}
    </>
  );
}
