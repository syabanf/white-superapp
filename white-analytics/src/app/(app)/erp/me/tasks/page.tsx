import type { Metadata } from "next";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/dashboard/page-header";
import { canAccess } from "@/components/shell/nav-config";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { MyTasks } from "@/features/erp/performance/components/my-tasks";
import { SectionError } from "@/features/erp/performance/components/section-error";
import { completionSummary } from "@/features/erp/performance/lib";
import { getTaskMeta, listTaskItems } from "@/features/erp/performance/queries";
import { sp } from "@/features/erp/performance/strings";

export const metadata: Metadata = { title: sp.myTasks.title };

export default async function MyTasksPage(props: PageProps<"/erp/me/tasks">) {
  const q = await props.searchParams;
  const { access, offline } = await requireErp("self.service");
  const m = sp.myTasks;
  if (offline)
    return (
      <>
        <PageHeader eyebrow={sp.common.eyebrow} title={m.title} description={m.subtitle} />
        <OfflineNotice message={offline.message} />
      </>
    );

  const view = ["today", "week", "month"].includes(sp1(q.view) ?? "") ? sp1(q.view)! : "today";
  const [metaRes, itemsRes] = await Promise.all([
    load(getTaskMeta),
    load(() => listTaskItems({ scope: "mine", view })),
  ]);
  const summary = itemsRes.data ? completionSummary(itemsRes.data) : null;
  const error = metaRes.error ?? itemsRes.error;

  return (
    <>
      <PageHeader
        eyebrow={sp.common.eyebrow}
        title={m.title}
        description={summary ? m.summary(summary.open, summary.overdue, summary.review, summary.approved) : m.subtitle}
        actions={
          canAccess(access, "hr.read") ? (
            <Button asChild size="sm" variant="secondary">
              <Link href="/erp/performance/tasks">{m.teamBoard}</Link>
            </Button>
          ) : null
        }
      />
      <UrlTabs
        param="view"
        label={m.title}
        tabs={[
          { value: "today", label: sp.tasks.viewToday },
          { value: "week", label: sp.tasks.viewWeek },
          { value: "month", label: sp.tasks.viewMonth },
        ]}
      />
      {error ? <SectionError error={error} /> : <MyTasks items={itemsRes.data!} meta={metaRes.data!} />}
    </>
  );
}
