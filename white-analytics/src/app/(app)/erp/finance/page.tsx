import { Suspense } from "react";
import type { Metadata } from "next";
import { Badge } from "@/components/ui/badge";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { resolvePeriod } from "@/features/erp/shared/dashboard-period";
import { ErpLoadError } from "@/features/erp/shared/states";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { DrillBreadcrumb, type DrillCrumb } from "@/features/erp/shared/drill/drill-breadcrumb";
import { withParams } from "@/features/erp/shared/drill/href";
import { FinanceDrill } from "@/features/erp/finance/dashboard/drill";
import { listAccounts, listCashBankAccounts } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";
import { FinanceFilters } from "@/features/erp/finance/dashboard/filters";
import {
  getFinanceArAp,
  getFinanceCash,
  getFinanceProfitability,
  getFinanceSpend,
  getFinanceSummary,
} from "@/features/erp/finance/dashboard/queries";
import { TabSkeleton } from "@/features/erp/finance/dashboard/skeleton";
import { ds } from "@/features/erp/finance/dashboard/strings";
import { ArApTab, CashTab, ProfitTab, SpendTab, SummaryTab } from "@/features/erp/finance/dashboard/tabs";
import type { DashboardFilters } from "@/features/erp/finance/dashboard/types";

export const metadata: Metadata = { title: fs.dashboard.title };

const TABS = [
  { value: "summary", label: ds.tabs.summary },
  { value: "profit", label: ds.tabs.profit },
  { value: "cash", label: ds.tabs.cash },
  { value: "arap", label: ds.tabs.arap },
  { value: "spend", label: ds.tabs.spend },
];

const PL_TYPES = new Set(["REVENUE", "COGS", "EXPENSE", "OTHER_INCOME", "OTHER_EXPENSE"]);

export default async function FinanceDashboardPage(props: PageProps<"/erp/finance">) {
  const sp = await props.searchParams;
  const { offline } = await requireErp("finance.read");
  const period = resolvePeriod({ preset: sp1(sp.preset), from: sp1(sp.from), to: sp1(sp.to), compare: sp1(sp.compare) }, "ytd");
  const tab = TABS.some((t) => t.value === sp1(sp.tab)) ? sp1(sp.tab)! : "summary";
  const header = (
    <PageHeader
      eyebrow={fs.eyebrow}
      title={fs.dashboard.title}
      description={fs.dashboard.subtitle}
      actions={
        <Badge variant="outline" className="font-normal text-muted-foreground">
          {fs.dashboard.postedOnly}
        </Badge>
      }
    />
  );
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;

  const [accounts, cash] = await Promise.all([load(() => listAccounts()), load(listCashBankAccounts)]);
  const groups = (accounts.data ?? [])
    .filter((a) => !a.isPostable && a.level >= 2 && PL_TYPES.has(a.accountTypeCode))
    .map((a) => ({ value: a.id, label: `${a.codeDisplay} ${a.name}` }));
  const cashAccounts = (cash.data?.accounts ?? []).map((a) => ({ value: a.id, label: a.name }));
  const filters: DashboardFilters = {
    from: period.from,
    to: period.to,
    compare: period.compare,
    accountType: sp1(sp.accountType),
    accountGroup: sp1(sp.accountGroup),
    department: sp1(sp.department),
    cashAccountId: sp1(sp.cashAccount),
  };
  return (
    <>
      {header}
      <div className="flex flex-col gap-3">
        <FinanceFilters period={period} tab={tab} groups={groups} cashAccounts={cashAccounts} />
        <DrillBreadcrumb items={drillTrail(sp, period, groups.find((g) => g.value === filters.accountGroup)?.label, filters.department)} />
        <UrlTabs tabs={TABS} label={ds.tabsLabel} />
      </div>
      <Suspense key={`${tab}|${JSON.stringify(filters)}`} fallback={<TabSkeleton />}>
        <TabBody tab={tab} filters={filters} />
      </Suspense>
      <FinanceDrill period={{ from: period.from, to: period.to }} />
    </>
  );
}

/** "Semua > 1 Mar – 31 Mar 2026 > 6100 Beban > Creative": custom range and dimension drills, each linking up. */
function drillTrail(
  sp: Record<string, string | string[] | undefined>,
  period: { preset: string; label: string },
  group?: string,
  department?: string,
): DrillCrumb[] {
  const items: DrillCrumb[] = [
    { label: ds.drill.all, href: withParams("/erp/finance", sp, { from: null, to: null, accountGroup: null, department: null }, ["drill"]) },
  ];
  if (period.preset === "custom") items.push({ label: period.label, href: withParams("/erp/finance", sp, { accountGroup: null, department: null }, ["drill"]) });
  if (group) items.push({ label: group, href: withParams("/erp/finance", sp, { department: null }, ["drill"]) });
  if (department) items.push({ label: department });
  return items;
}

async function TabBody({ tab, filters }: { tab: string; filters: DashboardFilters }) {
  switch (tab) {
    case "profit": {
      const r = await load(() => getFinanceProfitability(filters));
      return r.error ? <ErpLoadError error={r.error} /> : <ProfitTab data={r.data} filters={filters} />;
    }
    case "cash": {
      const r = await load(() => getFinanceCash(filters));
      return r.error ? <ErpLoadError error={r.error} /> : <CashTab data={r.data} filters={filters} />;
    }
    case "arap": {
      const r = await load(() => getFinanceArAp(filters));
      return r.error ? <ErpLoadError error={r.error} /> : <ArApTab data={r.data} filters={filters} />;
    }
    case "spend": {
      const r = await load(() => getFinanceSpend(filters));
      return r.error ? <ErpLoadError error={r.error} /> : <SpendTab data={r.data} filters={filters} />;
    }
    default: {
      const r = await load(() => getFinanceSummary(filters));
      return r.error ? <ErpLoadError error={r.error} /> : <SummaryTab data={r.data} filters={filters} />;
    }
  }
}
