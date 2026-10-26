import "server-only";
import { gatewayFetch } from "@/lib/erp/gateway";
import { qs } from "@/features/erp/shared/load";
import type {
  DashboardBilling,
  DashboardLeads,
  DashboardPipeline,
  DashboardQuery,
  DashboardSales,
  DashboardSummary,
  DashboardTeam,
} from "./types";

const get = <T>(tab: string, q: DashboardQuery) => gatewayFetch<T>(`/v1/sales/dashboard/${tab}${qs(q)}`);

export const getDashboardSummary = (q: DashboardQuery) => get<DashboardSummary>("summary", q);
export const getDashboardPipeline = (q: DashboardQuery) => get<DashboardPipeline>("pipeline", q);
export const getDashboardSales = (q: DashboardQuery) => get<DashboardSales>("sales", q);
export const getDashboardTeam = (q: DashboardQuery) => get<DashboardTeam>("team", q);
export const getDashboardLeads = (q: DashboardQuery) => get<DashboardLeads>("leads", q);
export const getDashboardBilling = (q: DashboardQuery) => get<DashboardBilling>("billing", q);
