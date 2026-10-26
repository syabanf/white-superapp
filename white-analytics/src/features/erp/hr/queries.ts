import "server-only";
import { gatewayFetch } from "@/lib/erp/gateway";
import { qs } from "@/features/erp/shared/load";
import type {
  Contract,
  DailyRoster,
  Department,
  EmployeeDocument,
  EmployeeProfile,
  EmployeeSchedule,
  EmploymentHistoryEntry,
  EmploymentStatusOption,
  ExpiringContracts,
  Holiday,
  HrisReport,
  LeaveBalance,
  LeaveType,
  ListAnnouncementsResponse,
  ListAttendanceResponse,
  ListEmployeesResponse,
  ListLeavesResponse,
  Loan,
  Offboarding,
  OnboardingChecklist,
  OrgChart,
  Overtime,
  PayrollAddition,
  PayrollRun,
  PayrollRunDetail,
  PayrollSettings,
  Position,
  Salary,
  Shift,
  TeamSchedule,
  UserOption,
} from "./types";

const H = "/v1/hr";

// People and master data

export function listEmployees(p: {
  search?: string;
  departmentId?: string;
  positionId?: string;
  level?: string;
  gender?: string;
  employmentStatus?: string;
  includeInactive?: boolean;
  pageSize?: number;
  pageToken?: string;
}) {
  return gatewayFetch<ListEmployeesResponse>(`${H}/employees${qs(p)}`);
}

/** Every active employee, for pickers (name + id). */
export async function listEmployeeOptions() {
  const res = await listEmployees({ pageSize: 500 });
  return res.employees.map((e) => ({ id: e.id, label: `${e.fullName} · ${e.employeeNo}` }));
}

export const getEmployeeProfile = (id: string) =>
  gatewayFetch<EmployeeProfile>(`${H}/employees/${id}/profile`);
export const listEmploymentHistory = (id: string) =>
  gatewayFetch<{ entries: EmploymentHistoryEntry[] }>(`${H}/employees/${id}/history`).then((r) => r.entries);
export const listEmployeeDocuments = (id: string) =>
  gatewayFetch<{ documents: EmployeeDocument[] }>(`${H}/employees/${id}/documents`).then((r) => r.documents);
export const getEmployeeSchedule = (id: string) =>
  gatewayFetch<EmployeeSchedule>(`${H}/employees/${id}/schedule`);
export const getLeaveBalance = (id: string, year?: number) =>
  gatewayFetch<LeaveBalance>(`${H}/employees/${id}/leave-balance${qs({ year })}`);
export const getOnboarding = (id: string) =>
  gatewayFetch<OnboardingChecklist>(`${H}/employees/${id}/onboarding`);
export const getOffboarding = (id: string) => gatewayFetch<Offboarding>(`${H}/employees/${id}/offboarding`);
export const getOrgChart = () => gatewayFetch<OrgChart>(`${H}/org-chart`);

export const listDepartments = (includeInactive = false) =>
  gatewayFetch<{ departments: Department[] }>(`${H}/departments${qs({ includeInactive })}`).then(
    (r) => r.departments,
  );
export const listPositions = (includeInactive = false) =>
  gatewayFetch<{ positions: Position[] }>(`${H}/positions${qs({ includeInactive })}`).then((r) => r.positions);
export const listEmploymentStatuses = () =>
  gatewayFetch<{ statuses: EmploymentStatusOption[] }>(`${H}/employment-statuses`).then((r) => r.statuses);

/** Identity users for the "link login" picker. */
export const listUserOptions = () =>
  gatewayFetch<{ users: UserOption[] }>(`/v1/users:options`).then((r) => r.users);

// Attendance and schedules

export type AttendanceFilter = {
  employeeId?: string;
  departmentId?: string;
  startDate?: string;
  endDate?: string;
  status?: string;
  onlyLate?: boolean;
  onlyUnvalidated?: boolean;
  page?: number;
  pageSize?: number;
};

export const listAttendance = (f: AttendanceFilter) =>
  gatewayFetch<ListAttendanceResponse>(`${H}/attendance${qs(f)}`);
export const getDailyRoster = (date?: string, departmentId?: string) =>
  gatewayFetch<DailyRoster>(`${H}/attendance/roster${qs({ date, departmentId })}`);
export const listShifts = (includeInactive = true) =>
  gatewayFetch<{ shifts: Shift[] }>(`${H}/shifts${qs({ includeInactive })}`).then((r) => r.shifts);
export const listHolidays = (year?: number) =>
  gatewayFetch<{ holidays: Holiday[] }>(`${H}/holidays${qs({ year })}`).then((r) => r.holidays);
export const getTeamSchedule = (departmentId?: string, weekOf?: string) =>
  gatewayFetch<TeamSchedule>(`${H}/schedule/team${qs({ departmentId, weekOf })}`);

// Leave, overtime, contracts, announcements

export type LeaveFilter = {
  employeeId?: string;
  departmentId?: string;
  status?: string;
  leaveType?: string;
  startDate?: string;
  endDate?: string;
  scope?: string;
  page?: number;
  pageSize?: number;
};

export const listLeaveTypes = () =>
  gatewayFetch<{ types: LeaveType[] }>(`${H}/leave-types`).then((r) => r.types);
export const listLeaves = (f: LeaveFilter) => gatewayFetch<ListLeavesResponse>(`${H}/leaves${qs(f)}`);

export const listOvertime = (f: {
  employeeId?: string;
  departmentId?: string;
  status?: string;
  month?: string;
  scope?: string;
}) =>
  gatewayFetch<{ overtime: Overtime[] }>(`${H}/overtime${qs(f)}`).then((r) => r.overtime);

export const listContracts = (f: { employeeId?: string; status?: string; contractType?: string; search?: string }) =>
  gatewayFetch<{ contracts: Contract[] }>(`${H}/contracts${qs(f)}`).then((r) => r.contracts);
export const listExpiringContracts = (days = 30) =>
  gatewayFetch<ExpiringContracts>(`${H}/contracts:expiring${qs({ days })}`);

export const listAnnouncements = (status?: string) =>
  gatewayFetch<ListAnnouncementsResponse>(`${H}/announcements${qs({ status })}`);

// Payroll

export const getPayrollSettings = () => gatewayFetch<PayrollSettings>(`${H}/payroll/settings`);
export const listSalaries = (f: { employeeId?: string; activeOnly?: boolean }) =>
  gatewayFetch<{ salaries: Salary[] }>(`${H}/salaries${qs(f)}`).then((r) => r.salaries);
export const listPayrollRuns = (f: { year?: number; status?: string }) =>
  gatewayFetch<{ runs: PayrollRun[] }>(`${H}/payroll/runs${qs(f)}`).then((r) => r.runs);
export const getPayrollRun = (id: string) => gatewayFetch<PayrollRunDetail>(`${H}/payroll/runs/${id}`);
export const listPayrollAdditions = (f: { periodMonth?: number; periodYear?: number; employeeId?: string }) =>
  gatewayFetch<{ additions: PayrollAddition[] }>(`${H}/payroll/additions${qs(f)}`).then((r) => r.additions);
export const listLoans = (f: { employeeId?: string; status?: string }) =>
  gatewayFetch<{ loans: Loan[] }>(`${H}/loans${qs(f)}`).then((r) => r.loans);

// Reports

export const getHrisReport = (month: number, year: number) =>
  gatewayFetch<HrisReport>(`${H}/reports/monthly${qs({ month, year })}`);
