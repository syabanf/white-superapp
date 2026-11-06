import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { todayWib } from "@/features/erp/hr/perms";
import { listProjectOptions, listTimesheet } from "@/features/erp/hr/workplace/queries";
import { tw, weekRange } from "@/features/erp/hr/workplace/strings";
import { MyTimesheet } from "@/features/erp/hr/workplace/timesheet";
import { tm } from "@/features/erp/me/strings";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";

const s = tw.timesheet;
export const metadata: Metadata = { title: s.myTitle };

export default async function MyTimesheetPage(props: PageProps<"/erp/me/timesheet">) {
  const sp = await props.searchParams;
  const { offline } = await requireErp("self.service");
  const header = <PageHeader eyebrow={tm.eyebrow} title={s.myTitle} description={s.mySubtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  const today = todayWib();
  const week = weekRange(sp1(sp.week), today);
  const [entries, projects] = await Promise.all([load(() => listTimesheet({ from: week.from, to: week.to })), load(listProjectOptions)]);
  return (
    <>
      {header}
      {entries.data ? (
        <MyTimesheet entries={entries.data} projects={projects.data ?? []} today={today} {...week} />
      ) : (
        <ErpError message={entries.error.message} />
      )}
    </>
  );
}
