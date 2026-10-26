import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { UrlDate } from "@/features/erp/performance/components/common";
import { MyRecap } from "@/features/erp/performance/components/my-recap";
import { SectionError } from "@/features/erp/performance/components/section-error";
import { isDate, jakartaToday } from "@/features/erp/performance/lib";
import { getMyDailyRecap, listDailyRecapHistory } from "@/features/erp/performance/queries";
import { sp } from "@/features/erp/performance/strings";

export const metadata: Metadata = { title: sp.myRecap.title };

export default async function MyRecapPage(props: PageProps<"/erp/me/recap">) {
  const q = await props.searchParams;
  const { offline } = await requireErp("self.service");
  const m = sp.myRecap;
  const today = jakartaToday();
  const date = isDate(sp1(q.date)) ? sp1(q.date)! : today;
  const header = (
    <PageHeader eyebrow={sp.common.eyebrow} title={m.title} description={m.subtitle} actions={<UrlDate fallback={today} />} />
  );
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;

  const [recap, history] = await Promise.all([
    load(() => getMyDailyRecap(date)),
    load(() => listDailyRecapHistory("me", 14)),
  ]);
  return (
    <>
      {header}
      {recap.error ? (
        <SectionError error={recap.error} />
      ) : (
        <MyRecap key={date} recap={recap.data} history={history.data ?? []} />
      )}
    </>
  );
}
