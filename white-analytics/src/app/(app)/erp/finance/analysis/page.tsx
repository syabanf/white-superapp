import { Suspense } from "react";
import type { Metadata } from "next";
import { Skeleton } from "@/components/ui/skeleton";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { DashboardFilterBar } from "@/features/erp/shared/dashboard-filter-bar";
import { resolvePeriod } from "@/features/erp/shared/dashboard-period";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { AnalysisView, InsightPanel } from "@/features/erp/finance/components/analysis";
import { getFinancialAnalysis, listFinancialInsights } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";
import { getRatioTrend } from "@/features/erp/finance/dashboard/queries";
import { RatioTrends } from "@/features/erp/finance/dashboard/ratio-trends";

export const metadata: Metadata = { title: fs.analysis.title };

export default async function FinanceAnalysisPage(props: PageProps<"/erp/finance/analysis">) {
  const sp = await props.searchParams;
  const { offline } = await requireErp("finance.read");
  const period = resolvePeriod({ preset: sp1(sp.preset), from: sp1(sp.from), to: sp1(sp.to), compare: sp1(sp.compare) }, "thisQuarter");
  const header = <PageHeader eyebrow={fs.eyebrow} title={fs.analysis.title} description={fs.analysis.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  const [analysis, insights] = await Promise.all([
    load(() => getFinancialAnalysis(period.from, period.to, period.compare)),
    load(() => listFinancialInsights(20)),
  ]);
  const range = analysis.data?.current;
  return (
    <>
      {header}
      <DashboardFilterBar period={period} />
      {analysis.error ? <ErpLoadError error={analysis.error} /> : <AnalysisView data={analysis.data} />}
      <Suspense key={period.to} fallback={<Skeleton className="h-96 rounded-xl" />}>
        <Trends to={period.to} />
      </Suspense>
      {range ? (
        insights.error ? (
          <ErpLoadError error={insights.error} compact />
        ) : (
          <InsightPanel from={range.from} to={range.to} insights={insights.data} />
        )
      ) : null}
    </>
  );
}

async function Trends({ to }: { to: string }) {
  const r = await load(() => getRatioTrend(to, 12));
  return r.error ? <ErpLoadError error={r.error} compact /> : <RatioTrends data={r.data} />;
}
