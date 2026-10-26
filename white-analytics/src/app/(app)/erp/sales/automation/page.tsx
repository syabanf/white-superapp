import type { Metadata } from "next";
import { ShieldAlert } from "lucide-react";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpEmpty, ErpError } from "@/features/erp/shared/states";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { CustomFieldsManager, ScoringRules, StagesEditor, WorkflowRules } from "@/features/erp/sales/components/automation";
import { salesCaps } from "@/features/erp/sales/lib";
import { listCustomFields, listPipelines, listScoringRules, listWorkflowRules, listWorkflowRuns } from "@/features/erp/sales/queries";
import { ownerOptions } from "@/features/erp/sales/refs";
import { ss } from "@/features/erp/sales/strings";

const a = ss.automation;
export const metadata: Metadata = { title: a.title };

export default async function AutomationPage(props: PageProps<"/erp/sales/automation">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("sales.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const caps = salesCaps(access);
  const header = <PageHeader eyebrow={ss.eyebrow} title={a.title} description={a.subtitle} />;
  if (!caps.admin)
    return (
      <>
        {header}
        <ErpEmpty icon={<ShieldAlert className="size-5" />} title={ss.adminOnly} />
      </>
    );
  const tab = sp1(sp.tab) ?? "workflows";

  let body: React.ReactNode;
  if (tab === "scoring") {
    const rules = await load(listScoringRules);
    body = rules.data ? <ScoringRules rules={rules.data} /> : <ErpError message={rules.error?.message} />;
  } else if (tab === "stages") {
    const pipelines = await load(listPipelines);
    body = pipelines.data ? <StagesEditor pipelines={pipelines.data} /> : <ErpError message={pipelines.error?.message} />;
  } else if (tab === "fields") {
    const fields = await load(() => listCustomFields());
    body = fields.data ? <CustomFieldsManager fields={fields.data} /> : <ErpError message={fields.error?.message} />;
  } else {
    const runsFor = sp1(sp.runs);
    const [rules, owners, runs] = await Promise.all([
      load(listWorkflowRules),
      ownerOptions(),
      runsFor ? load(() => listWorkflowRuns(runsFor)) : null,
    ]);
    body = rules.data ? (
      <WorkflowRules rules={rules.data} owners={owners} runs={runsFor && runs?.data ? { ruleId: runsFor, items: runs.data } : undefined} />
    ) : (
      <ErpError message={rules.error?.message} />
    );
  }

  return (
    <>
      {header}
      <UrlTabs
        label={a.title}
        tabs={[
          { value: "workflows", label: a.tabs.workflows },
          { value: "scoring", label: a.tabs.scoring },
          { value: "stages", label: a.tabs.stages },
          { value: "fields", label: a.tabs.fields },
        ]}
        reset={["runs"]}
      />
      {body}
    </>
  );
}
