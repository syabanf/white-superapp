import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { canAccess } from "@/components/shell/nav-config";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { FeedbackCycleDetail, FeedbackCycles, MyAssignments } from "@/features/erp/performance/components/feedback";
import { SectionError } from "@/features/erp/performance/components/section-error";
import {
  getTaskMeta,
  listAssignments,
  listFeedbackCycles,
  listMyAssignments,
  listSummaries,
} from "@/features/erp/performance/queries";
import { sp } from "@/features/erp/performance/strings";

export const metadata: Metadata = { title: sp.feedback.title };

const f = sp.feedback;

export default async function FeedbackPage(props: PageProps<"/erp/performance/feedback">) {
  const q = await props.searchParams;
  const { access, offline } = await requireErp("hr.read");
  const tab = sp1(q.tab) ?? "cycles";
  return (
    <>
      <PageHeader eyebrow={sp.common.eyebrow} title={f.title} description={f.subtitle} />
      {offline ? (
        <OfflineNotice message={offline.message} />
      ) : (
        <>
          <UrlTabs
            label={f.title}
            reset={["cycle"]}
            tabs={[
              { value: "cycles", label: f.tabCycles },
              { value: "mine", label: f.tabMine },
            ]}
          />
          {tab === "mine" ? (
            <MineSection />
          ) : (
            <CyclesSection cycleId={sp1(q.cycle)} canWrite={canAccess(access, "hr.write")} />
          )}
        </>
      )}
    </>
  );
}

async function MineSection() {
  const res = await load(listMyAssignments);
  if (res.error) return <SectionError error={res.error} />;
  return <MyAssignments assignments={res.data} />;
}

async function CyclesSection({ cycleId, canWrite }: { cycleId?: string; canWrite: boolean }) {
  const res = await load(() => listFeedbackCycles());
  if (res.error) return <SectionError error={res.error} />;
  const cycle = res.data.find((c) => c.id === cycleId);
  if (!cycle) return <FeedbackCycles cycles={res.data} canWrite={canWrite} />;
  const [assignments, summaries, meta] = await Promise.all([
    load(() => listAssignments({ cycleId: cycle.id })),
    load(() => listSummaries(cycle.id)),
    load(getTaskMeta),
  ]);
  return (
    <>
      <FeedbackCycles cycles={res.data} active={cycle.id} canWrite={canWrite} />
      {assignments.error ? (
        <SectionError error={assignments.error} />
      ) : (
        <FeedbackCycleDetail
          cycle={cycle}
          assignments={assignments.data}
          summaries={summaries.data ?? []}
          people={meta.data?.members ?? []}
          canWrite={canWrite}
        />
      )}
    </>
  );
}
