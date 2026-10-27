import type { Metadata } from "next";
import { Toolbar } from "@/components/dashboard/layout";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1, spInt } from "@/features/erp/shared/load";
import { DateRangeFilter } from "@/features/erp/shared/period-picker";
import { SelectFilter } from "@/features/erp/shared/url-filters";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { UrlDate } from "@/features/erp/performance/components/common";
import { ChecklistTab, HistoryTab, SummaryTab, TemplatesTab } from "@/features/erp/performance/components/logbook";
import { SectionError } from "@/features/erp/performance/components/section-error";
import { isDate, jakartaToday } from "@/features/erp/performance/lib";
import {
  getLogbookMe,
  getLogbookSummary,
  getTaskMeta,
  listLogbookEntries,
  listLogbookTemplates,
} from "@/features/erp/performance/queries";
import { sp } from "@/features/erp/performance/strings";

export const metadata: Metadata = { title: sp.logbook.title };

const l = sp.logbook;

export default async function LogbookPage(props: PageProps<"/erp/performance/logbook">) {
  const q = await props.searchParams;
  const { offline } = await requireErp("hr.read");
  const header = <PageHeader eyebrow={sp.common.eyebrow} title={l.title} description={l.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;

  const [meRes, metaRes] = await Promise.all([load(getLogbookMe), load(getTaskMeta)]);
  if (meRes.error) return <>{header}<SectionError error={meRes.error} /></>;
  const me = meRes.data;
  const departments = metaRes.data?.departments ?? [];
  const deptNames = Object.fromEntries(departments.map((d) => [d.id, d.name]));
  // Only full-access users pick a department; the service scopes everyone else.
  const dept = me.isFullAccess ? sp1(q.dept) : undefined;
  const tab = sp1(q.tab) ?? "checklist";
  const today = jakartaToday();
  const canManage = me.me.hrWrite || me.me.headOf.length > 0;

  return (
    <>
      <PageHeader
        eyebrow={sp.common.eyebrow}
        title={l.title}
        description={me.isFullAccess ? l.subtitle : `${l.subtitle} ${sp.common.department}: ${me.departmentName}.`}
      />
      <Toolbar>
        <UrlTabs
          label={l.title}
          reset={["date", "status", "from", "to"]}
          tabs={[
            { value: "checklist", label: l.tabChecklist },
            { value: "history", label: l.tabHistory },
            { value: "templates", label: l.tabTemplates },
            { value: "summary", label: l.tabSummary },
          ]}
        />
        {me.isFullAccess ? (
          <SelectFilter
            param="dept"
            label={sp.common.department}
            allLabel={sp.common.allDepartments}
            options={departments.map((d) => ({ value: d.id, label: d.name }))}
          />
        ) : null}
        {tab === "checklist" ? <UrlDate fallback={today} /> : null}
        {tab === "summary" ? <DateRangeFilter /> : null}
      </Toolbar>
      {!me.me.employeeId && !me.isFullAccess ? <SectionNote text={sp.common.unlinked} /> : null}
      {tab === "history" ? (
        <HistorySection
          dept={dept}
          status={sp1(q.status)}
          from={sp1(q.from)}
          to={sp1(q.to)}
          page={spInt(q.page, 1)}
          pageSize={spInt(q.pageSize, 20)}
          canReview={me.canReview}
          deptNames={deptNames}
        />
      ) : tab === "templates" ? (
        <TemplatesSection
          dept={dept}
          departments={departments.filter((d) => me.me.hrWrite || me.me.headOf.includes(d.id))}
          defaultDept={me.me.departmentId ?? ""}
          canManage={canManage}
          deptNames={deptNames}
        />
      ) : tab === "summary" ? (
        <SummarySection dept={dept} from={sp1(q.from)} to={sp1(q.to)} />
      ) : (
        <ChecklistSection
          dept={dept}
          date={isDate(sp1(q.date)) ? sp1(q.date)! : today}
          canReview={me.canReview}
          deptNames={deptNames}
        />
      )}
    </>
  );
}

function SectionNote({ text }: { text: string }) {
  return <p className="rounded-xl bg-muted/60 px-4 py-3 text-sm">{text}</p>;
}

async function ChecklistSection({
  dept,
  date,
  canReview,
  deptNames,
}: {
  dept?: string;
  date: string;
  canReview: boolean;
  deptNames: Record<string, string>;
}) {
  const [templates, entries] = await Promise.all([
    load(() => listLogbookTemplates(dept)),
    load(() => listLogbookEntries({ departmentId: dept, date, limit: 50 })),
  ]);
  const err = templates.error ?? entries.error;
  if (err) return <SectionError error={err} />;
  return (
    <ChecklistTab
      key={date}
      templates={templates.data!.filter((t) => t.isActive)}
      entries={entries.data!.entries}
      date={date}
      canReview={canReview}
      deptNames={deptNames}
    />
  );
}

async function HistorySection(p: {
  dept?: string;
  status?: string;
  from?: string;
  to?: string;
  page: number;
  pageSize: number;
  canReview: boolean;
  deptNames: Record<string, string>;
}) {
  const res = await load(() =>
    listLogbookEntries({ departmentId: p.dept, status: p.status, from: p.from, to: p.to, page: p.page, limit: p.pageSize }),
  );
  if (res.error) return <SectionError error={res.error} />;
  return (
    <HistoryTab
      entries={res.data.entries}
      pagination={{ page: p.page, pageSize: p.pageSize, total: res.data.count }}
      canReview={p.canReview}
      deptNames={p.deptNames}
      toolbar={
        <div className="flex flex-wrap gap-2">
          <SelectFilter
            param="status"
            label={sp.common.status}
            allLabel={sp.common.all}
            options={Object.entries(l.entryStatus).map(([value, label]) => ({ value, label }))}
          />
          <DateRangeFilter />
        </div>
      }
    />
  );
}

async function TemplatesSection(p: {
  dept?: string;
  departments: Awaited<ReturnType<typeof getTaskMeta>>["departments"];
  defaultDept: string;
  canManage: boolean;
  deptNames: Record<string, string>;
}) {
  const res = await load(() => listLogbookTemplates(p.dept));
  if (res.error) return <SectionError error={res.error} />;
  return (
    <TemplatesTab
      templates={res.data}
      departments={p.departments}
      defaultDept={p.defaultDept}
      canManage={p.canManage}
      deptNames={p.deptNames}
    />
  );
}

async function SummarySection({ dept, from, to }: { dept?: string; from?: string; to?: string }) {
  const res = await load(() => getLogbookSummary({ departmentId: dept, from, to }));
  if (res.error) return <SectionError error={res.error} />;
  return <SummaryTab rows={res.data} />;
}
