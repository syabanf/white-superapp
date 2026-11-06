import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { UrlTabs } from "@/features/erp/hr/components/ui";
import { hrPerms, todayWib } from "@/features/erp/hr/perms";
import { th } from "@/features/erp/hr/strings";
import { listTimesheet } from "@/features/erp/hr/workplace/queries";
import { tw, weekRange } from "@/features/erp/hr/workplace/strings";
import { TimesheetQueue } from "@/features/erp/hr/workplace/timesheet";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";

const s = tw.timesheet;
export const metadata: Metadata = { title: s.title };

export default async function TimesheetAdminPage(props: PageProps<"/erp/hr/timesheet">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("hr.read");
  const header = <PageHeader eyebrow={th.eyebrow} title={s.title} description={s.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  const today = todayWib();
  const queue = sp1(sp.tab) !== "all";
  // The queue looks back a quarter; "all" shows one week.
  const week = weekRange(sp1(sp.week), today);
  const from = queue ? new Date(Date.parse(`${today}T00:00:00Z`) - 92 * 86_400_000).toISOString().slice(0, 10) : week.from;
  const res = await load(() => listTimesheet({ from, to: queue ? today : week.to, status: queue ? "submitted" : undefined }));
  return (
    <>
      {header}
      <UrlTabs tabs={[{ value: "queue", label: s.queue }, { value: "all", label: s.all }]} value={queue ? "queue" : "all"} />
      {res.data ? <TimesheetQueue entries={res.data} canApprove={hrPerms(access).approve} queue={queue} /> : <ErpError message={res.error.message} />}
    </>
  );
}
