import "server-only";
import { gatewayFetch } from "@/lib/erp/gateway";
import { qs } from "@/features/erp/shared/load";
import type {
  DashboardFilters,
  FinanceArAp,
  FinanceCash,
  FinanceProfitability,
  FinanceSpend,
  FinanceSummary,
  RatioTrend,
} from "./types";

const D = "/v1/finance/dashboard";

export const getFinanceSummary = (f: DashboardFilters) => gatewayFetch<FinanceSummary>(`${D}/summary${qs(f)}`);
export const getFinanceProfitability = (f: DashboardFilters) => gatewayFetch<FinanceProfitability>(`${D}/profitability${qs(f)}`);
export const getFinanceCash = (f: DashboardFilters) => gatewayFetch<FinanceCash>(`${D}/cash${qs(f)}`);
export const getFinanceArAp = (f: DashboardFilters) => gatewayFetch<FinanceArAp>(`${D}/ar-ap${qs(f)}`);
export const getFinanceSpend = (f: DashboardFilters) => gatewayFetch<FinanceSpend>(`${D}/expenses${qs(f)}`);
export const getRatioTrend = (to: string, months = 12) => gatewayFetch<RatioTrend>(`/v1/finance/analysis/trend${qs({ to, months })}`);
