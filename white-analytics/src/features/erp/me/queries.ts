import "server-only";
import { gatewayFetch } from "@/lib/erp/gateway";
import { qs } from "@/features/erp/shared/load";
import type {
  LeaveBalance,
  ListAnnouncementsResponse,
  ListAttendanceResponse,
  ListLeavesResponse,
  Loan,
  MyApprovals,
  MyHome,
  Overtime,
  Payslip,
  TeamSchedule,
} from "@/features/erp/hr/types";
import type { MyDashboard } from "./types";

const M = "/v1/hr/me";

export const getMyHome = () => gatewayFetch<MyHome>(`${M}/home`);
export const listMyAttendance = (month?: string) =>
  gatewayFetch<ListAttendanceResponse>(`${M}/attendance${qs({ month })}`);
export const listMyLeaves = (year?: number, status?: string) =>
  gatewayFetch<ListLeavesResponse>(`${M}/leaves${qs({ year, status })}`).then((r) => r.leaves);
export const getMyLeaveBalance = (year?: number) => gatewayFetch<LeaveBalance>(`${M}/leave-balance${qs({ year })}`);
export const listMyOvertime = (month?: string) =>
  gatewayFetch<{ overtime: Overtime[] }>(`${M}/overtime${qs({ month })}`).then((r) => r.overtime);
export const listMyLoans = () => gatewayFetch<{ loans: Loan[] }>(`${M}/loans`).then((r) => r.loans);
export const listMyPayslips = (year?: number) =>
  gatewayFetch<{ payslips: Payslip[] }>(`${M}/payslips${qs({ year })}`).then((r) => r.payslips);
export const listMyAnnouncements = () => gatewayFetch<ListAnnouncementsResponse>(`${M}/announcements`);
export const getMyApprovals = () => gatewayFetch<MyApprovals>(`${M}/approvals`);
export const getMyTeamSchedule = (weekOf?: string) => gatewayFetch<TeamSchedule>(`${M}/team-schedule${qs({ weekOf })}`);
export const getMyDashboard = (month?: string) => gatewayFetch<MyDashboard>(`${M}/dashboard${qs({ month })}`);
