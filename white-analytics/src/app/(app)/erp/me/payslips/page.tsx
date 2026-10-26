import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, spInt } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { SelectFilter } from "@/features/erp/shared/url-filters";
import { MyPayslips } from "@/features/erp/me/components/lists";
import { listMyPayslips } from "@/features/erp/me/queries";
import { tm } from "@/features/erp/me/strings";

export const metadata: Metadata = { title: tm.payslips.title };

export default async function MyPayslipsPage(props: PageProps<"/erp/me/payslips">) {
  const sp = await props.searchParams;
  const { offline } = await requireErp("self.service");
  const header = <PageHeader eyebrow={tm.eyebrow} title={tm.payslips.title} description={tm.payslips.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  const now = new Date().getFullYear();
  const year = spInt(sp.year, now);
  const res = await load(() => listMyPayslips(year));
  return (
    <>
      {header}
      <div>
        <SelectFilter
          param="year"
          label={tm.payslips.year}
          allLabel={String(now)}
          options={[now - 1, now - 2].map((y) => ({ value: String(y), label: String(y) }))}
        />
      </div>
      {res.data ? <MyPayslips payslips={res.data} /> : <ErpError message={res.error.message} />}
    </>
  );
}
