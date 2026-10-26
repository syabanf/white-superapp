import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { DateRangeFilter, MonthFilter } from "@/features/erp/shared/period-picker";
import { ErpLoadError } from "@/features/erp/shared/states";
import { ss } from "@/features/erp/shared/strings";
import { ResetFilters, SelectFilter } from "@/features/erp/shared/url-filters";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { listEmployeeOptions } from "@/features/erp/hr/queries";
import { monthParam } from "@/features/erp/hr/perms";
import { InterviewCalendar, InterviewList, ScheduleButton } from "@/features/erp/recruitment/components/interviews";
import { recPerms } from "@/features/erp/recruitment/lib";
import { listCandidateOptions, listInterviews } from "@/features/erp/recruitment/queries";
import { INTERVIEW_STATUS, rs } from "@/features/erp/recruitment/strings";

export const metadata: Metadata = { title: rs.interviews.title };

const iv = rs.interviews;
const VIEWS = [
  { value: "list", label: iv.list },
  { value: "calendar", label: iv.calendar },
];
const none = { data: [] as never[], error: null };

export default async function InterviewsPage(props: PageProps<"/erp/hr/recruitment/interviews">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("recruitment.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const perms = recPerms(access);
  const calendar = sp1(sp.view) === "calendar";
  const { ym, month, year } = monthParam(sp1(sp.month));
  const lastDay = new Date(Date.UTC(year, month, 0)).getUTCDate();

  const filter = calendar
    ? { dateFrom: `${ym}-01`, dateTo: `${ym}-${String(lastDay).padStart(2, "0")}` }
    : { status: sp1(sp.status), dateFrom: sp1(sp.from), dateTo: sp1(sp.to) };
  const [list, candidates, employees] = await Promise.all([
    load(() => listInterviews(filter)),
    perms.write ? load(listCandidateOptions) : Promise.resolve(none),
    perms.write ? load(listEmployeeOptions) : Promise.resolve(none),
  ]);
  const lookups = {
    candidates: candidates.data ?? [],
    employees: (employees.data ?? []).map((e) => ({ value: e.id, label: e.label })),
  };
  // upcoming first in the list view
  const rows = calendar ? list.data : list.data?.slice().sort((a, b) => (b.interviewDate ?? "").localeCompare(a.interviewDate ?? ""));

  return (
    <>
      <PageHeader
        eyebrow={rs.eyebrow}
        title={iv.title}
        description={iv.subtitle}
        actions={perms.write ? <ScheduleButton lookups={lookups} /> : null}
      />
      <div className="flex flex-wrap items-center gap-2">
        <UrlTabs tabs={VIEWS} param="view" label={iv.title} reset={["status", "from", "to"]} />
        {calendar ? (
          <MonthFilter fallback={ym} />
        ) : (
          <>
            <SelectFilter
              param="status"
              label={iv.statusFilter}
              allLabel={ss.allStatus}
              options={Object.entries(INTERVIEW_STATUS).map(([value, label]) => ({ value, label }))}
            />
            <DateRangeFilter />
            <ResetFilters params={["status", "from", "to"]} />
          </>
        )}
      </div>
      {!rows ? (
        <ErpLoadError error={list.error!} />
      ) : calendar ? (
        <InterviewCalendar interviews={rows} month={ym} />
      ) : (
        <InterviewList interviews={rows} canWrite={perms.write} lookups={lookups} />
      )}
    </>
  );
}
