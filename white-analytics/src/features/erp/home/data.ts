import "server-only";
import { cache } from "react";
import { load } from "@/features/erp/shared/load";
import { getFinanceArAp, getFinanceSpend, getFinanceSummary } from "@/features/erp/finance/dashboard/queries";
import { getHrDashboard } from "@/features/erp/hr/dashboard/queries";
import { getDashboard as getPerformanceDashboard } from "@/features/erp/performance/queries";
import type { DashSummary } from "@/features/erp/performance/dashboard/types";
import { getDashboardPipeline, getDashboardSummary } from "@/features/erp/sales/dashboard/queries";
import { getDashboard as getRecruitmentDashboard } from "@/features/erp/recruitment/queries";

/**
 * Module dashboard loads for the ERP home. Each is memoized per request, so a
 * module section and the attention strip share one gateway call, and each
 * returns errors as values, so one failing module never breaks the page.
 * Arguments stay primitive because cache() keys on identity.
 */
type Compare = "previous" | "year";

export const financeSummary = cache((from: string, to: string, compare: Compare) =>
  load(() => getFinanceSummary({ from, to, compare })),
);
export const financeArAp = cache((from: string, to: string, compare: Compare) =>
  load(() => getFinanceArAp({ from, to, compare })),
);
export const financeSpend = cache((from: string, to: string, compare: Compare) =>
  load(() => getFinanceSpend({ from, to, compare })),
);
export const hrDashboard = cache((from: string, to: string, compare: Compare) =>
  load(() => getHrDashboard({ from, to, compare })),
);
export const performanceSummary = cache((from: string, to: string, compare: Compare, scope: "me" | "") =>
  load(() => getPerformanceDashboard<DashSummary>("summary", { from, to, compare, scope: scope || undefined })),
);
export const salesSummary = cache((from: string, to: string, compare: Compare) =>
  load(() => getDashboardSummary({ from, to, compare })),
);
export const salesPipeline = cache((from: string, to: string, compare: Compare) =>
  load(() => getDashboardPipeline({ from, to, compare })),
);
export const recruitmentDashboard = cache((from: string, to: string, compare: Compare) =>
  load(() => getRecruitmentDashboard({ from, to, compare })),
);
