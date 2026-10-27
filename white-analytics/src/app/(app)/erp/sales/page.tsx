import { Suspense } from "react";
import type { Metadata } from "next";
import Link from "next/link";
import { BadgePercent, Target } from "lucide-react";
import { PageHeader } from "@/components/dashboard/page-header";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { ActivityList } from "@/features/erp/sales/components/activities";
import { DealRows, Panel } from "@/features/erp/sales/components/blocks";
import { SalesDashboardFilters } from "@/features/erp/sales/dashboard/components/filters";
import { DASH_TABS, readDashboardParams } from "@/features/erp/sales/dashboard/params";
import { DashboardSection } from "@/features/erp/sales/dashboard/sections";
import { DashboardSkeleton } from "@/features/erp/sales/dashboard/skeleton";
import { dd } from "@/features/erp/sales/dashboard/strings";
import { DashboardTrail, type TrailLevel } from "@/features/erp/sales/dashboard/trail";
import { salesCaps } from "@/features/erp/sales/lib";
import { listActivities, listApprovals, listDeals, listPipelines } from "@/features/erp/sales/queries";
import { ownerOptions } from "@/features/erp/sales/refs";
import { LEAD_SOURCE, SERVICE_LINE, ss } from "@/features/erp/sales/strings";

export const metadata: Metadata = { title: dd.title };

export default async function SalesDashboardPage(props: PageProps<"/erp/sales">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("sales.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const caps = salesCaps(access);
  const { period, query, tab } = readDashboardParams(sp);
  const [pipelines, owners, approvals] = await Promise.all([
    load(() => listPipelines()),
    ownerOptions(),
    caps.approve ? load(() => listApprovals({ view: "mine", status: "pending" })) : null,
  ]);
  if (pipelines.error?.code === "NETWORK") return <OfflineNotice message={pipelines.error.message} />;
  const pendingApprovals = approvals?.data?.filter((r) => r.canDecide).length ?? 0;
  const trail: TrailLevel[] = [
    period.preset === "custom" ? { label: period.label, params: ["preset", "from", "to"] } : null,
    query.pipelineId ? { label: pipelines.data?.find((p) => p.id === query.pipelineId)?.name ?? query.pipelineId, params: ["pipeline"] } : null,
    query.ownerUserId ? { label: owners.find((o) => o.value === query.ownerUserId)?.label ?? query.ownerUserId, params: ["owner"] } : null,
    query.serviceLine ? { label: SERVICE_LINE[query.serviceLine] ?? query.serviceLine, params: ["line"] } : null,
    query.source ? { label: LEAD_SOURCE[query.source] ?? query.source, params: ["source"] } : null,
  ].filter((l): l is TrailLevel => l != null);

  return (
    <>
      <PageHeader
        eyebrow={ss.eyebrow}
        title={dd.title}
        description={dd.subtitle}
        actions={
          <Button size="sm" variant="outline" asChild>
            <Link href="/erp/sales/reports">
              <Target className="size-3.5" /> {dd.forecastLink}
            </Link>
          </Button>
        }
      />
      {pendingApprovals ? (
        <Alert className="rounded-xl">
          <BadgePercent className="size-4" />
          <AlertDescription>
            <Link href="/erp/sales/quotations?tab=approvals" className="font-medium hover:underline">
              {dd.summary.approvals(pendingApprovals)}
            </Link>
          </AlertDescription>
        </Alert>
      ) : null}
      <SalesDashboardFilters
        period={period}
        pipelines={(pipelines.data ?? []).map((p) => ({ value: p.id, label: p.name }))}
        owners={owners}
      />
      <DashboardTrail levels={trail} />
      <UrlTabs label={dd.title} tabs={DASH_TABS.map((t) => ({ value: t, label: dd.tabs[t] }))} />
      <Suspense key={`${tab}|${JSON.stringify(query)}`} fallback={<DashboardSkeleton />}>
        <DashboardSection tab={tab} query={query} />
      </Suspense>
      {tab === "ringkasan" ? (
        <Suspense fallback={null}>
          <MyWork canWrite={caps.write} owners={owners} />
        </Suspense>
      ) : null}
    </>
  );
}

/** The signed-in user's open tasks and the latest wins. */
async function MyWork({ canWrite, owners }: { canWrite: boolean; owners: { value: string; label: string }[] }) {
  const d = ss.dashboard;
  const [tasks, wins] = await Promise.all([
    load(() => listActivities({ mine: true, openOnly: true, limit: 8 })),
    load(() => listDeals({ status: "won", pageSize: 5 })),
  ]);
  return (
    <div className="grid gap-(--gap) lg:grid-cols-2">
      <Panel title={d.myTasks}>
        <ActivityList activities={tasks.data ?? []} owners={owners} canWrite={canWrite} showSubject emptyLabel={d.noTasks} />
      </Panel>
      <Panel title={d.recentWins} flush>
        <DealRows deals={wins.data?.deals ?? []} empty={d.noWins} />
      </Panel>
    </div>
  );
}
