import type { Metadata } from "next";
import { KpiTile } from "@/components/dashboard/kpi-tile";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { MonthFilter } from "@/features/erp/shared/period-picker";
import { ErpError } from "@/features/erp/shared/states";
import { TeamScheduleGrid } from "@/features/erp/hr/components/schedules";
import { UrlTabs } from "@/features/erp/hr/components/ui";
import { sumHours } from "@/features/erp/hr/format";
import { monthParam } from "@/features/erp/hr/perms";
import { th } from "@/features/erp/hr/strings";
import { MyAttendanceTable } from "@/features/erp/me/components/lists";
import { getMyTeamSchedule, listMyAttendance } from "@/features/erp/me/queries";
import { tm } from "@/features/erp/me/strings";
import { formatNumber } from "@/lib/format";

export const metadata: Metadata = { title: tm.attendance.title };

const TABS = Object.entries(tm.attendance.tabs).map(([value, label]) => ({ value, label }));

export default async function MyAttendancePage(props: PageProps<"/erp/me/attendance">) {
  const sp = await props.searchParams;
  const { offline } = await requireErp("self.service");
  const t = tm.attendance;
  const team = sp1(sp.tab) === "team";
  const header = <PageHeader eyebrow={tm.eyebrow} title={t.title} description={t.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;

  if (team) {
    const res = await load(() => getMyTeamSchedule(sp1(sp.week)));
    return (
      <>
        {header}
        <UrlTabs tabs={TABS} value="team" />
        {res.data ? <TeamScheduleGrid data={res.data} /> : <ErpError message={res.error.message} />}
      </>
    );
  }

  const { ym } = monthParam(sp1(sp.month));
  const res = await load(() => listMyAttendance(ym));
  const rows = res.data?.records ?? [];
  return (
    <>
      {header}
      <UrlTabs tabs={TABS} value="history" />
      <MonthFilter fallback={ym} />
      {res.data ? (
        <>
          <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
            <KpiTile size="sm" label={t.present} value={formatNumber(rows.filter((r) => r.clockIn).length)} />
            <KpiTile size="sm" label={t.late} value={formatNumber(rows.filter((r) => r.isLate).length)} />
            <KpiTile size="sm" label={t.hours} value={th.hours(sumHours(rows.map((r) => r.workHours)))} />
            <KpiTile size="sm" label={t.overtime} value={th.hours(sumHours(rows.map((r) => r.overtimeHours)))} />
          </div>
          <MyAttendanceTable records={rows} />
        </>
      ) : (
        <ErpError message={res.error.message} />
      )}
    </>
  );
}
