import type { Metadata } from "next";
import { Toolbar } from "@/components/dashboard/layout";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { MonthFilter } from "@/features/erp/shared/period-picker";
import { ResetFilters, SelectFilter } from "@/features/erp/shared/url-filters";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { UrlDate, UrlSelect } from "@/features/erp/performance/components/common";
import { MonthlyRecaps, TeamDailyRecaps } from "@/features/erp/performance/components/recaps";
import { SectionError } from "@/features/erp/performance/components/section-error";
import { isDate, isMonth, jakartaToday, splitMonth } from "@/features/erp/performance/lib";
import { getTaskMeta, listMonthlyRecaps, listTeamDailyRecaps } from "@/features/erp/performance/queries";
import { sp } from "@/features/erp/performance/strings";

export const metadata: Metadata = { title: sp.recaps.title };

const c = sp.recaps;

export default async function RecapsPage(props: PageProps<"/erp/performance/recaps">) {
  const q = await props.searchParams;
  const { offline } = await requireErp("hr.read");
  const header = <PageHeader eyebrow={sp.common.eyebrow} title={c.title} description={c.subtitle} />;
  if (offline)
    return (
      <>
        {header}
        <OfflineNotice message={offline.message} />
      </>
    );

  const meta = await load(getTaskMeta);
  const departments = meta.data?.departments ?? [];
  const tab = sp1(q.tab) ?? "daily";
  const today = jakartaToday();
  const thisMonth = today.slice(0, 7);
  const dept = sp1(q.dept);
  const scope = sp1(q.scope) === "department" ? "department" : "employee";

  return (
    <>
      {header}
      <Toolbar>
        <UrlTabs
          label={c.title}
          reset={["date", "month", "scope"]}
          tabs={[
            { value: "daily", label: c.tabDaily },
            { value: "monthly", label: c.tabMonthly },
          ]}
        />
        <SelectFilter
          param="dept"
          label={sp.common.department}
          allLabel={sp.common.allDepartments}
          options={departments.map((d) => ({ value: d.id, label: d.name }))}
        />
        {tab === "monthly" ? (
          <>
            <UrlSelect
              param="scope"
              label={sp.improvements.scope}
              fallback="employee"
              options={[
                { value: "employee", label: c.scopeEmployee },
                { value: "department", label: c.scopeDepartment },
              ]}
            />
            <MonthFilter fallback={thisMonth} />
          </>
        ) : (
          <UrlDate fallback={today} />
        )}
        <ResetFilters params={["dept", "date", "month", "scope"]} />
      </Toolbar>
      {tab === "monthly" ? (
        <MonthlySection
          month={isMonth(sp1(q.month)) ? sp1(q.month)! : thisMonth}
          scope={scope}
          dept={dept}
          meta={meta.data}
        />
      ) : (
        <DailySection date={isDate(sp1(q.date)) ? sp1(q.date)! : today} dept={dept} />
      )}
    </>
  );
}

async function DailySection({ date, dept }: { date: string; dept?: string }) {
  const res = await load(() => listTeamDailyRecaps(date, dept));
  if (res.error) return <SectionError error={res.error} />;
  return <TeamDailyRecaps recaps={res.data} />;
}

async function MonthlySection({
  month,
  scope,
  dept,
  meta,
}: {
  month: string;
  scope: "employee" | "department";
  dept?: string;
  meta: Awaited<ReturnType<typeof getTaskMeta>> | null;
}) {
  const { year, month: m } = splitMonth(month);
  const res = await load(() =>
    listMonthlyRecaps({ periodYear: year, periodMonth: m, scope, departmentId: dept }),
  );
  if (res.error) return <SectionError error={res.error} />;
  const me = meta?.me;
  const canGenerate = !!me && (me.hrWrite || me.headOf.length > 0 || me.directReports.length > 0);
  return (
    <MonthlyRecaps
      recaps={res.data}
      canGenerate={canGenerate}
      people={meta?.members ?? []}
      departments={(meta?.departments ?? []).filter((d) => me?.hrWrite || me?.headOf.includes(d.id))}
      scope={scope}
      period={{ periodYear: year, periodMonth: m }}
    />
  );
}
