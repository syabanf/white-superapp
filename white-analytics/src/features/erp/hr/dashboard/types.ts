/** GET /v1/hr/dashboard (HrReportService.GetHrDashboard). Money is a decimal string. */

export type HrDashKpis = {
  headcount: number;
  joiners: number;
  leavers: number;
  turnoverRate: number;
  attendanceRecords: number;
  attendanceRate: number;
  lateCount: number;
  lateRate: number;
  avgLateMinutes: number;
  absentCount: number;
  avgWorkHours: number;
  leaveDays: number;
  overtimeHours: number;
  overtimeCost: string;
  payrollGross: string;
  payrollNet: string;
  payrollEmployerCost: string;
  payrollEmployees: number;
};

export type HrDashBucket = {
  date: string;
  headcount: number;
  joiners: number;
  leavers: number;
  attendanceRecords: number;
  present: number;
  late: number;
  absent: number;
  attendanceRate: number;
  lateRate: number;
  avgWorkHours: number;
  leaveDays: number;
  overtimeHours: number;
  overtimeCost: string;
  turnoverRate: number;
  payrollGross: string;
  payrollNet: string;
  payrollPph21: string;
  payrollBpjsEmployee: string;
  payrollBpjsEmployer: string;
  payrollEmployees: number;
};

export type HrDashDeptMonth = {
  month: string;
  departmentId: string;
  headcount: number;
  attendanceRecords: number;
  attendanceRate: number;
  late: number;
  leaveDays: number;
  overtimeHours: number;
  overtimeCost: string;
  payrollGross: string;
};

export type HrDashDept = {
  departmentId: string;
  name: string;
  headcount: number;
  joiners: number;
  leavers: number;
  attendanceRecords: number;
  attendanceRate: number;
  late: number;
  lateRate: number;
  lateMinutes: number;
  absent: number;
  avgWorkHours: number;
  leaveDays: number;
  overtimeHours: number;
  overtimeCost: string;
  payrollGross: string;
  payrollEmployerCost: string;
  leaveQuota: number;
  leaveUsed: number;
};

export type HrDashSlice = { key: string; label: string; value: number; count: number };
export type HrDashCell = { row: string; col: number; value: number; total: number };
export type HrDashTypeMonth = { month: string; type: string; days: number };

export type HrDashEmployee = {
  employeeId: string;
  fullName: string;
  departmentName: string;
  positionName: string;
  attendanceRecords: number;
  late: number;
  lateMinutes: number;
  absent: number;
  attendanceRate: number;
  overtimeHours: number;
  overtimeCost: string;
};

export type HrDashPending = {
  kind: "leave" | "overtime";
  id: string;
  employeeId: string;
  fullName: string;
  departmentName: string;
  type: string;
  date: string;
  amount: number;
  status: string;
  ageDays: number;
};

export type HrDashExpiry = {
  kind: "contract" | "probation";
  employeeId: string;
  fullName: string;
  departmentName: string;
  contractId: string;
  contractNumber: string;
  contractType: string;
  endDate: string;
  daysLeft: number;
};

export type HrDashLevel = {
  level: string;
  employees: number;
  avgBase: string;
  avgGross: string;
  minGross: string;
  maxGross: string;
};

export type HrDashLoan = {
  loanId: string;
  employeeId: string;
  fullName: string;
  departmentName: string;
  loanType: string;
  principalAmount: string;
  remainingBalance: string;
  monthlyInstallment: string;
};

export type HrDashboard = {
  from: string;
  to: string;
  compareFrom: string;
  compareTo: string;
  bucket: "day" | "week" | "month";
  payrollVisible: boolean;
  current: HrDashKpis;
  previous: HrDashKpis;
  buckets: HrDashBucket[];
  previousBuckets: HrDashBucket[];
  months: HrDashBucket[];
  deptMonths: HrDashDeptMonth[];
  departments: HrDashDept[];
  lateByWeekday: HrDashCell[];
  lateByDay: HrDashCell[];
  absenceReasons: HrDashSlice[];
  clockInOffsets: HrDashSlice[];
  topLate: HrDashEmployee[];
  topOvertime: HrDashEmployee[];
  leaveByType: HrDashSlice[];
  leaveTypeMonths: HrDashTypeMonth[];
  pendingAging: HrDashSlice[];
  pending: HrDashPending[];
  byStatus: HrDashSlice[];
  byPosition: HrDashSlice[];
  byLevel: HrDashSlice[];
  byContractType: HrDashSlice[];
  byGender: HrDashSlice[];
  ageBands: HrDashSlice[];
  tenureBands: HrDashSlice[];
  expiring: HrDashExpiry[];
  salaryByLevel: HrDashLevel[];
  loans: HrDashLoan[];
  loansOutstanding: string;
  pendingCount: number;
};
