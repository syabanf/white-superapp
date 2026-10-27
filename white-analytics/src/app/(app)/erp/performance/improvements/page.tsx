import type { Metadata } from "next";
import { Toolbar } from "@/components/dashboard/layout";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ResetFilters, SelectFilter } from "@/features/erp/shared/url-filters";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { ImprovementsTable, RetrosPanel } from "@/features/erp/performance/components/improvements";
import { SectionError } from "@/features/erp/performance/components/section-error";
import { getTaskMeta, listImprovements, listRetros } from "@/features/erp/performance/queries";
import { sp } from "@/features/erp/performance/strings";

export const metadata: Metadata = { title: sp.improvements.title };

const im = sp.improvements;
const options = (map: Record<string, string>) => Object.entries(map).map(([value, label]) => ({ value, label }));

export default async function ImprovementsPage(props: PageProps<"/erp/performance/improvements">) {
  const q = await props.searchParams;
  const { offline } = await requireErp("hr.read");
  const header = <PageHeader eyebrow={sp.common.eyebrow} title={im.title} description={im.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;

  const meta = await load(getTaskMeta);
  const departments = meta.data?.departments ?? [];
  const people = meta.data?.members ?? [];
  const tab = sp1(q.tab) ?? "items";
  const dept = sp1(q.dept);

  return (
    <>
      {header}
      <Toolbar>
        <UrlTabs
          label={im.title}
          reset={["status", "source", "scope"]}
          tabs={[
            { value: "items", label: im.tabItems },
            { value: "retros", label: im.tabRetros },
          ]}
        />
        <SelectFilter
          param="dept"
          label={sp.common.department}
          allLabel={sp.common.allDepartments}
          options={departments.map((d) => ({ value: d.id, label: d.name }))}
        />
      </Toolbar>
      {tab === "retros" ? (
        <RetrosSection
          dept={dept}
          departments={departments.filter((d) => meta.data?.me.hrWrite || meta.data?.me.headOf.includes(d.id))}
          people={people}
          defaultDept={dept ?? meta.data?.me.departmentId ?? ""}
        />
      ) : (
        <ItemsSection
          filters={{ departmentId: dept, status: sp1(q.status), source: sp1(q.source), scope: sp1(q.scope) }}
          people={people}
          departments={departments}
        />
      )}
    </>
  );
}

async function ItemsSection({
  filters,
  people,
  departments,
}: {
  filters: { departmentId?: string; status?: string; source?: string; scope?: string };
  people: Awaited<ReturnType<typeof getTaskMeta>>["members"];
  departments: Awaited<ReturnType<typeof getTaskMeta>>["departments"];
}) {
  const res = await load(() => listImprovements(filters));
  if (res.error) return <SectionError error={res.error} />;
  return (
    <ImprovementsTable
      items={res.data}
      people={people}
      departments={departments}
      toolbar={
        <>
          <SelectFilter param="status" label={sp.common.status} allLabel={sp.common.all} options={options(im.statuses)} />
          <SelectFilter param="source" label={im.source} allLabel={sp.common.all} options={options(im.sources)} />
          <SelectFilter param="scope" label={im.scope} allLabel={sp.common.all} options={options(im.scopes)} />
          <ResetFilters params={["status", "source", "scope"]} />
        </>
      }
    />
  );
}

async function RetrosSection({
  dept,
  departments,
  people,
  defaultDept,
}: {
  dept?: string;
  departments: Awaited<ReturnType<typeof getTaskMeta>>["departments"];
  people: Awaited<ReturnType<typeof getTaskMeta>>["members"];
  defaultDept: string;
}) {
  const res = await load(() => listRetros(dept));
  if (res.error) return <SectionError error={res.error} />;
  return <RetrosPanel retros={res.data} departments={departments} people={people} defaultDept={defaultDept} />;
}
