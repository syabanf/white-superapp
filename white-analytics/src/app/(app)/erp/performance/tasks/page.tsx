import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { MonthFilter } from "@/features/erp/shared/period-picker";
import { ErpEmpty } from "@/features/erp/shared/states";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { UrlSelect } from "@/features/erp/performance/components/common";
import { KpiTasksTable, MonitorPanel, TaskBoard } from "@/features/erp/performance/components/department-tasks";
import { SectionError } from "@/features/erp/performance/components/section-error";
import { TaskFormDialog } from "@/features/erp/performance/components/task-form-dialog";
import { isMonth, jakartaToday } from "@/features/erp/performance/lib";
import { getKpiTasks, getMonitor, getTaskMeta, listTaskItems } from "@/features/erp/performance/queries";
import { sp } from "@/features/erp/performance/strings";

export const metadata: Metadata = { title: sp.tasks.title };

const s = sp.tasks;

export default async function DepartmentTasksPage(props: PageProps<"/erp/performance/tasks">) {
  const q = await props.searchParams;
  const { offline } = await requireErp("hr.read");
  const header = <PageHeader eyebrow={sp.common.eyebrow} title={s.title} description={s.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;

  const metaRes = await load(getTaskMeta);
  if (metaRes.error) return <>{header}<SectionError error={metaRes.error} /></>;
  const meta = metaRes.data;

  const tab = sp1(q.tab) ?? "board";
  const departments = meta.departments;
  const defaultDept =
    departments.find((d) => d.id === meta.me.departmentId)?.id ?? departments[0]?.id ?? "";
  const dept = departments.some((d) => d.id === sp1(q.dept)) ? sp1(q.dept)! : defaultDept;
  const view = ["today", "week", "month"].includes(sp1(q.view) ?? "") ? sp1(q.view)! : "week";
  const thisMonth = (meta.today || jakartaToday()).slice(0, 7);
  const month = isMonth(sp1(q.month)) ? sp1(q.month)! : thisMonth;

  return (
    <>
      <PageHeader
        eyebrow={sp.common.eyebrow}
        title={s.title}
        description={s.subtitle}
        actions={meta.teamScope !== "none" && dept ? <TaskFormDialog meta={meta} departmentId={dept} /> : null}
      />
      {meta.teamScope === "none" ? (
        <ErpEmpty title={s.teamNone} />
      ) : (
        <>
          <div className="flex flex-wrap items-center gap-2">
            <UrlTabs
              label={s.title}
              tabs={[
                { value: "board", label: s.tabBoard },
                { value: "monitor", label: s.tabMonitor },
                { value: "kpi", label: s.tabKpi },
              ]}
            />
            <UrlSelect
              param="dept"
              label={sp.common.department}
              fallback={defaultDept}
              options={departments.map((d) => ({ value: d.id, label: d.name }))}
              className="min-w-40"
            />
            {tab === "board" ? (
              <UrlSelect
                param="view"
                label={s.title}
                fallback="week"
                options={[
                  { value: "today", label: s.viewToday },
                  { value: "week", label: s.viewWeek },
                  { value: "month", label: s.viewMonth },
                ]}
              />
            ) : (
              <MonthFilter fallback={thisMonth} />
            )}
          </div>
          {tab === "monitor" ? (
            <MonitorSection month={month} dept={dept} meta={meta} />
          ) : tab === "kpi" ? (
            <KpiSection month={month} dept={dept} meta={meta} />
          ) : (
            <BoardSection view={view} dept={dept} meta={meta} />
          )}
        </>
      )}
    </>
  );
}

type SectionProps = { dept: string; meta: Awaited<ReturnType<typeof getTaskMeta>> };

async function BoardSection({ view, dept, meta }: SectionProps & { view: string }) {
  const res = await load(() => listTaskItems({ scope: "team", view, departmentId: dept }));
  if (res.error) return <SectionError error={res.error} />;
  return <TaskBoard items={res.data} meta={meta} />;
}

async function MonitorSection({ month, dept, meta }: SectionProps & { month: string }) {
  const res = await load(() => getMonitor(month, dept));
  if (res.error) return <SectionError error={res.error} />;
  return <MonitorPanel data={res.data} meta={meta} />;
}

async function KpiSection({ month, dept, meta }: SectionProps & { month: string }) {
  const res = await load(() => getKpiTasks(dept, month));
  if (res.error) return <SectionError error={res.error} />;
  return <KpiTasksTable rows={res.data} meta={meta} />;
}
