import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { ProjectTagsTable, TagDialog } from "@/features/erp/finance/components/project-tags";
import { financePerms } from "@/features/erp/finance/perms";
import { AllocationDialog, AllocationsTable } from "@/features/erp/finance/components/controls";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { listProjectAllocations, listProjectTags } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { matchesActive, matchesText } from "@/features/erp/shared/list-filter";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";

const s = fs.projects;
export const metadata: Metadata = { title: s.title };

export default async function ProjectsPage(props: PageProps<"/erp/finance/projects">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("finance.read");
  const { canWrite, canApprove } = financePerms(access);
  const header = <PageHeader eyebrow={fs.eyebrow} title={s.title} description={s.subtitle} actions={canWrite && !offline ? <TagDialog /> : null} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  const tabs = <UrlTabs label={s.title} tabs={[{ value: "tags", label: fs.controls.tabTags }, { value: "allocations", label: fs.controls.tabAllocations }]} reset={["q", "client", "status"]} />;
  const res = await load(() => listProjectTags());
  if (sp1(sp.tab) === "allocations") {
    const year = new Date().getFullYear();
    const allocations = await load(() => listProjectAllocations({ dateFrom: `${year}-01-01`, dateTo: `${year}-12-31` }));
    return (
      <>
        <PageHeader eyebrow={fs.eyebrow} title={s.title} description={s.subtitle} actions={canApprove && res.data ? <AllocationDialog tags={res.data} /> : null} />
        {tabs}
        {allocations.error ? <ErpLoadError error={allocations.error} /> : <AllocationsTable rows={allocations.data} canApprove={canApprove} />}
      </>
    );
  }
  const [q, client, status] = [sp1(sp.q), sp1(sp.client), sp1(sp.status)];
  const clients = [...new Set((res.data ?? []).map((t) => t.clientName).filter(Boolean))].sort();
  return (
    <>
      {header}
      {tabs}
      {res.error ? (
        <ErpLoadError error={res.error} />
      ) : (
        <ProjectTagsTable tags={res.data.filter((t) => matchesText(q, t.name, t.clientName) && matchesActive(status, t.isActive) && (!client || t.clientName === client))} clients={clients} canWrite={canWrite} />
      )}
    </>
  );
}
