import type { EmployeeRef, Leave, LeaveBalance } from "@/features/erp/hr/types";

/** GET /v1/hr/me/dashboard (SelfServiceService.GetMyDashboard). */

export type MyDayStatus = "present" | "late" | "remote" | "absent" | "leave" | "holiday" | "off" | "upcoming" | "missing";

export type MyDashboardDay = {
  date: string;
  status: MyDayStatus;
  clockIn: string;
  clockOut: string;
  workHours: number;
  lateMinutes: number;
  note: string;
  overtimeHours: number;
};

export type MyDashboardStats = {
  scheduledDays: number;
  presentDays: number;
  lateDays: number;
  absentDays: number;
  leaveDays: number;
  onTimeRate: number;
  attendanceRate: number;
  workHours: number;
  overtimeHours: number;
  lateMinutes: number;
  avgClockIn: string;
};

export type MyDashboardMonth = {
  month: string;
  onTimeRate: number;
  teamOnTimeRate: number;
  workHours: number;
  overtimeHours: number;
  netPay: string;
  grossPay: string;
  presentDays: number;
  lateDays: number;
  leaveDays: number;
};

export type MyDashboardWeek = { weekStart: string; workHours: number; days: number; targetHours: number };

export type MyPayslipSummary = {
  payslipId: string;
  periodMonth: number;
  periodYear: number;
  baseSalary: string;
  allowances: string;
  overtimePay: string;
  grossSalary: string;
  bpjs: string;
  pph21: string;
  loanDeduction: string;
  otherDeductions: string;
  totalDeductions: string;
  netSalary: string;
  presentDays: number;
  workingDays: number;
};

export type MyPendingRequest = {
  kind: "leave" | "overtime" | "loan";
  id: string;
  type: string;
  date: string;
  endDate: string;
  amount: string;
  status: string;
  createdAt: string;
};

export type MyTeamStatus = "in" | "late" | "remote" | "leave" | "absent" | "not_in" | "off";

export type MyTeamMember = {
  employeeId: string;
  fullName: string;
  positionName: string;
  status: MyTeamStatus;
  clockIn: string;
  note: string;
};

export type MyDashboard = {
  employee: EmployeeRef;
  month: string;
  today: string;
  days: MyDashboardDay[];
  stats: MyDashboardStats;
  previous: MyDashboardStats;
  trend: MyDashboardMonth[];
  weeks: MyDashboardWeek[];
  leaveBalance: LeaveBalance;
  leaveHistory: Leave[];
  latestPayslip?: MyPayslipSummary;
  pending: MyPendingRequest[];
  isManager: boolean;
  teamToday: MyTeamMember[];
};
