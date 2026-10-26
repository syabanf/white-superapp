import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1, spInt } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { AttendanceTable, RosterView } from "@/features/erp/hr/components/attendance";
import { HolidaysPanel, ShiftsPanel, TeamScheduleGrid } from "@/features/erp/hr/components/schedules";
import { UrlTabs } from "@/features/erp/hr/components/ui";
import { hrPerms, todayWib } from "@/features/erp/hr/perms";
import {
  getDailyRoster,
  getTeamSchedule,
  listAttendance,
  listDepartments,
  listEmployeeOptions,
  listHolidays,
  listShifts,
} from "@/features/erp/hr/queries";
import { th } from "@/features/erp/hr/strings";

export const metadata: Metadata = { title: th.attendance.title };

const TABS = Object.entries(th.attendance.tabs).map(([value, label]) => ({ value, label }));

type SP = Record<string, string | string[] | undefined>;

export default async function AttendancePage(props: PageProps<"/erp/hr/attendance">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("hr.read");
  const a = th.attendance;
  const tab = TABS.some((t) => t.value === sp1(sp.tab)) ? sp1(sp.tab)! : "roster";
  return (
    <>
      <PageHeader eyebrow={th.eyebrow} title={a.title} description={a.subtitle} />
      {offline ? (
        <OfflineNotice message={offline.message} />
      ) : (
        <>
          <UrlTabs tabs={TABS} value={tab} />
          <Body tab={tab} sp={sp} perms={hrPerms(access)} />
        </>
      )}
    </>
  );
}

async function departmentOptions() {
  const res = await load(() => listDepartments());
  return (res.data ?? []).map((d) => ({ value: d.id, label: d.name }));
}

async function Body({ tab, sp, perms }: { tab: string; sp: SP; perms: ReturnType<typeof hrPerms> }) {
  switch (tab) {
    case "list": {
      const page = spInt(sp.page, 1);
      const pageSize = spInt(sp.pageSize, 20);
      const filters = { from: sp1(sp.from), to: sp1(sp.to), status: sp1(sp.status) };
      const [res, deps, employees] = await Promise.all([
        load(() =>
          listAttendance({
            startDate: filters.from,
            endDate: filters.to,
            status: filters.status,
            departmentId: sp1(sp.dept),
            employeeId: sp1(sp.employee),
            onlyLate: sp1(sp.late) === "1" || undefined,
            onlyUnvalidated: sp1(sp.unvalidated) === "1" || undefined,
            page,
            pageSize,
          }),
        ),
        departmentOptions(),
        load(listEmployeeOptions),
      ]);
      if (!res.data) return <ErpError message={res.error.message} />;
      return (
        <AttendanceTable
          records={res.data.records}
          page={{ page, pageSize, total: res.data.total }}
          canApprove={perms.approve}
          filters={filters}
          departments={deps}
          employees={employees.data ?? []}
        />
      );
    }
    case "team": {
      const [res, deps] = await Promise.all([
        load(() => getTeamSchedule(sp1(sp.dept), sp1(sp.week))),
        departmentOptions(),
      ]);
      return res.data ? <TeamScheduleGrid data={res.data} departments={deps} /> : <ErpError message={res.error.message} />;
    }
    case "shifts": {
      const res = await load(() => listShifts(true));
      return res.data ? <ShiftsPanel shifts={res.data} canWrite={perms.write} /> : <ErpError message={res.error.message} />;
    }
    case "holidays": {
      const year = spInt(sp.year, new Date().getFullYear());
      const res = await load(() => listHolidays(year));
      return res.data ? (
        <HolidaysPanel holidays={res.data} year={new Date().getFullYear()} canWrite={perms.write} />
      ) : (
        <ErpError message={res.error.message} />
      );
    }
    default: {
      const today = todayWib();
      const [res, deps] = await Promise.all([
        load(() => getDailyRoster(sp1(sp.date) ?? today, sp1(sp.dept))),
        departmentOptions(),
      ]);
      return res.data ? (
        <RosterView roster={res.data} today={today} departments={deps} />
      ) : (
        <ErpError message={res.error.message} />
      );
    }
  }
}
