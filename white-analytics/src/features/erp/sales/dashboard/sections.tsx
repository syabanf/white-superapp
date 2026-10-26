import "server-only";
import { Skeleton } from "@/components/ui/skeleton";
import { load } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { BillingView } from "./components/billing-view";
import { LeadsView } from "./components/leads-view";
import { PipelineView } from "./components/pipeline-view";
import { SalesView } from "./components/sales-view";
import { SummaryView } from "./components/summary-view";
import { TeamView } from "./components/team-view";
import type { DashTab } from "./params";
import {
  getDashboardBilling,
  getDashboardLeads,
  getDashboardPipeline,
  getDashboardSales,
  getDashboardSummary,
  getDashboardTeam,
} from "./queries";
import type { DashboardQuery } from "./types";

/** Fetches and renders one tab; streamed behind Suspense by the page. */
export async function DashboardSection({ tab, query }: { tab: DashTab; query: DashboardQuery }) {
  switch (tab) {
    case "pipeline": {
      const r = await load(() => getDashboardPipeline(query));
      return r.data ? <PipelineView data={r.data} query={query} /> : <ErpError message={r.error.message} />;
    }
    case "penjualan": {
      const r = await load(() => getDashboardSales(query));
      return r.data ? <SalesView data={r.data} query={query} /> : <ErpError message={r.error.message} />;
    }
    case "tim": {
      const r = await load(() => getDashboardTeam(query));
      return r.data ? <TeamView data={r.data} query={query} /> : <ErpError message={r.error.message} />;
    }
    case "lead": {
      const r = await load(() => getDashboardLeads(query));
      return r.data ? <LeadsView data={r.data} query={query} /> : <ErpError message={r.error.message} />;
    }
    case "tagihan": {
      const r = await load(() => getDashboardBilling(query));
      return r.data ? <BillingView data={r.data} query={query} /> : <ErpError message={r.error.message} />;
    }
    default: {
      const r = await load(() => getDashboardSummary(query));
      return r.data ? <SummaryView data={r.data} query={query} /> : <ErpError message={r.error.message} />;
    }
  }
}

/** Mirrors a tab: KPI row, a chart grid and a table. */
export function DashboardSkeleton() {
  return (
    <div className="flex flex-col gap-4" aria-busy>
      <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-[118px] rounded-xl" />
        ))}
      </div>
      <div className="grid gap-4 xl:grid-cols-3">
        <Skeleton className="h-80 rounded-xl xl:col-span-2" />
        <Skeleton className="h-80 rounded-xl" />
      </div>
      <Skeleton className="h-64 rounded-xl" />
    </div>
  );
}
