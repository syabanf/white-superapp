import "server-only";
import { gatewayFetch } from "@/lib/erp/gateway";
import { qs } from "@/features/erp/shared/load";
import type { HrDashboard } from "./types";

export type HrDashboardQuery = {
  from: string;
  to: string;
  compare: string;
  departmentId?: string;
  employmentStatus?: string;
  positionId?: string;
};

export const getHrDashboard = (q: HrDashboardQuery) => gatewayFetch<HrDashboard>(`/v1/hr/dashboard${qs(q)}`);
