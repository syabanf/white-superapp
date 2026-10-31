/**
 * JSON shapes of white.hr.v1 as grpc-gateway emits them (camelCase). Money
 * is a decimal string, days are "YYYY-MM-DD", timestamps are RFC 3339 or
 * null. Proto3 `optional` fields are omitted when unset, so they are `?`.
 */

export type EmploymentStatusEnum =
  | "EMPLOYMENT_STATUS_UNSPECIFIED"
  | "ACTIVE"
  | "PROBATION"
  | "ON_LEAVE"
  | "RESIGNED"
  | "TERMINATED";

export type Employee = {
  id: string;
  userId?: string;
  employeeNo: string;
  fullName: string;
  email: string;
  phone: string;
  departmentId?: string;
  departmentName: string;
  positionId?: string;
  positionName: string;
  managerId?: string;
  status: EmploymentStatusEnum;
  joinDate: string;
  endDate?: string;
  createdAt: string | null;
  managerUserId?: string;
};

export type EmployeeRef = { id: string; fullName: string; employeeNo: string; departmentName: string };

export type ListEmployeesResponse = { employees: Employee[]; nextPageToken: string; total: number };

export type Department = {
  id: string;
  code: string;
  name: string;
  parentId?: string;
  headEmployeeId?: string;
  isActive: boolean;
};

export type Position = {
  id: string;
  title: string;
  departmentId?: string;
  departmentName: string;
  level: string;
  salaryMin: string;
  salaryMax: string;
  isActive: boolean;
};

export type EmploymentStatusOption = { code: string; name: string; isActive: boolean; sortOrder: number };

export type EmployeeProfile = {
  employee: Employee;
  employmentStatus: string;
  employmentStatusName: string;
  managerName: string;
  nik: string;
  npwp: string;
  bpjsTkNo: string;
  bpjsKesNo: string;
  birthPlace: string;
  birthDate?: string;
  gender: string;
  maritalStatus: string;
  religion: string;
  address: string;
  probationEndDate?: string;
  bankName: string;
  bankAccountNo: string;
  bankAccountName: string;
  emergencyContactName: string;
  emergencyContactPhone: string;
  emergencyContactRelation: string;
  isActive: boolean;
  updatedAt: string | null;
  userEmail: string;
};

/** Body of CreateEmployee / UpdateEmployee (every field a string). */
export type EmployeeInput = {
  employeeNo: string;
  fullName: string;
  email: string;
  phone: string;
  departmentId: string;
  positionId: string;
  managerId: string;
  employmentStatus: string;
  joinDate: string;
  endDate: string;
  probationEndDate: string;
  nik: string;
  npwp: string;
  bpjsTkNo: string;
  bpjsKesNo: string;
  birthPlace: string;
  birthDate: string;
  gender: string;
  maritalStatus: string;
  religion: string;
  address: string;
  bankName: string;
  bankAccountNo: string;
  bankAccountName: string;
  emergencyContactName: string;
  emergencyContactPhone: string;
  emergencyContactRelation: string;
  userId: string;
};

export type EmploymentHistoryEntry = {
  id: string;
  changeType: string;
  notes: string;
  prevEmploymentStatus: string;
  newEmploymentStatus: string;
  prevDepartment: string;
  newDepartment: string;
  prevPosition: string;
  newPosition: string;
  createdAt: string | null;
};

export type EmployeeDocument = {
  id: string;
  employeeId: string;
  docType: string;
  name: string;
  contentType: string;
  sizeBytes: string;
  createdAt: string | null;
};

export type OrgNode = {
  employeeId: string;
  fullName: string;
  positionName: string;
  departmentName: string;
  reports: OrgNode[];
};

export type OrgChart = { roots: OrgNode[]; total: number };

export type GeoLocation = { latitude: number; longitude: number; accuracy: number; address: string };

export type AttendanceRecord = {
  id: string;
  employee: EmployeeRef;
  date: string;
  clockIn: string | null;
  clockOut: string | null;
  clockInLocation?: GeoLocation;
  clockOutLocation?: GeoLocation;
  hasClockInPhoto: boolean;
  hasClockOutPhoto: boolean;
  workHours: number;
  status: string;
  isLate: boolean;
  lateMinutes: number;
  overtimeHours: number;
  shiftId: string;
  shiftName: string;
  scheduledStart: string | null;
  scheduledEnd: string | null;
  isValidated: boolean;
  validatedByName: string;
  validatedAt: string | null;
  validationNotes: string;
  notes: string;
};

export type ListAttendanceResponse = { records: AttendanceRecord[]; total: number };

export type RosterEntry = {
  employee: EmployeeRef;
  shiftId: string;
  shiftName: string;
  shiftStart: string;
  shiftEnd: string;
  status: string;
  isOverdue: boolean;
  attendanceId: string;
  clockIn: string | null;
  clockOut: string | null;
  lateMinutes: number;
  leaveType: string;
  attendanceStatus: string;
  attendanceNotes: string;
};

export type DailyRoster = {
  date: string;
  holidayName: string;
  entries: RosterEntry[];
  summary: Record<string, number>;
};

export type Shift = {
  id: string;
  name: string;
  startTime: string;
  endTime: string;
  breakMinutes: number;
  lateToleranceMinutes: number;
  isOvernight: boolean;
  isActive: boolean;
  sortOrder: number;
};

export type ScheduleDay = {
  dayOfWeek: number;
  shiftId: string;
  shiftName: string;
  effectiveFrom: string;
  effectiveTo?: string;
};

export type EmployeeSchedule = { employeeId: string; current: ScheduleDay[]; history: ScheduleDay[] };

export type TeamScheduleDay = {
  date: string;
  status: string;
  shiftName: string;
  startTime: string;
  endTime: string;
};

export type TeamSchedule = {
  weekStart: string;
  rows: { employee: EmployeeRef; days: TeamScheduleDay[] }[];
};

export type Holiday = {
  id: string;
  holidayDate: string;
  name: string;
  type: string;
  deductsLeave: boolean;
  status: string;
  note: string;
};

export type LeaveType = { code: string; name: string; usesAnnualQuota: boolean };

export type Leave = {
  id: string;
  employee: EmployeeRef;
  leaveType: string;
  startDate: string;
  endDate: string;
  totalDays: number;
  reason: string;
  status: string;
  managerApprovedByName: string;
  managerApprovedAt: string | null;
  approvedByName: string;
  approvedAt: string | null;
  rejectionReason: string;
  createdAt: string | null;
  excludedHolidays: string[];
  canDecide: boolean;
  canCancel: boolean;
};

export type ListLeavesResponse = { leaves: Leave[]; total: number };

export type LeaveBalance = {
  employeeId: string;
  year: number;
  annualLeaveTotal: number;
  annualLeaveUsed: number;
  annualLeaveRemaining: number;
  sickLeaveUsed: number;
  unpaidLeaveUsed: number;
  otherLeaveUsed: number;
  pendingDays: number;
};

export type Overtime = {
  id: string;
  employee: EmployeeRef;
  date: string;
  startTime: string;
  endTime: string;
  hours: number;
  source: string;
  status: string;
  reason: string;
  requestedByName: string;
  decidedByName: string;
  decidedAt: string | null;
  rejectionReason: string;
  createdAt: string | null;
};

export type Contract = {
  id: string;
  contractNumber: string;
  employee: EmployeeRef;
  contractType: string;
  status: string;
  startDate: string;
  endDate?: string;
  probationEndDate?: string;
  parentContractId?: string;
  sequence: number;
  positionTitle: string;
  departmentName: string;
  workLocation: string;
  baseSalary: string;
  signedAt?: string;
  compensationAmount: string;
  terminatedReason: string;
  notes: string;
  createdAt: string | null;
  daysLeft?: number;
};

export type ContractActionResult = { message: string; contract?: Contract; compensationAmount: string };

export type ExpiringContracts = {
  days: number;
  contracts: Contract[];
  probations: { employee: EmployeeRef; probationEndDate: string; daysLeft: number }[];
  withoutContract: EmployeeRef[];
};

export type Announcement = {
  id: string;
  title: string;
  body: string;
  tags: string[];
  status: string;
  isPinned: boolean;
  targetScope: string;
  departmentIds: string[];
  publishAt: string | null;
  expiresAt: string | null;
  createdByName: string;
  createdAt: string | null;
  isRead: boolean;
  readCount: number;
};

export type ListAnnouncementsResponse = { announcements: Announcement[]; unread: number };

export type OnboardingTask = {
  id: string;
  employeeId: string;
  taskName: string;
  category: string;
  priority: number;
  dueDate?: string;
  isCompleted: boolean;
  completedAt: string | null;
  completedByName: string;
  notes: string;
};

export type OnboardingChecklist = { employeeId: string; tasks: OnboardingTask[]; completed: number; total: number };

export type Offboarding = {
  id: string;
  employeeId: string;
  resignationType: string;
  status: string;
  resignationDate: string;
  lastWorkingDay: string;
  reason: string;
  exitInterviewNotes: string;
  clearanceHr: boolean;
  clearanceIt: boolean;
  clearanceFinance: boolean;
  clearanceManager: boolean;
  notes: string;
  createdAt: string | null;
  completedAt: string | null;
};

export type PayrollSettings = {
  bpjsTkJhtEmployee: string;
  bpjsTkJhtEmployer: string;
  bpjsTkJpEmployee: string;
  bpjsTkJpEmployer: string;
  bpjsTkJkk: string;
  bpjsTkJkm: string;
  bpjsKesEmployee: string;
  bpjsKesEmployer: string;
  bpjsKesMaxUpah: string;
  taperaEmployee: string;
  taperaEmployer: string;
  ptkp: Record<string, string>;
  pph21BracketLimits: string[];
  overtimeMultiplier: string;
  overtimeMultiplierHoliday: string;
  overtimeHourlyDivisor: string;
  lateDeductionMode: string;
  lateDeductionAmount: string;
  loanMaxInstallmentPercent: string;
  loanMaxActivePerEmployee: number;
  payrollDay: number;
  pph21Method: string;
  updatedAt: string | null;
};

export type Salary = {
  id: string;
  employee: EmployeeRef;
  baseSalary: string;
  fixedAllowance: string;
  variableAllowance: string;
  transportAllowance: string;
  mealAllowance: string;
  housingAllowance: string;
  ptkpStatus: string;
  isTaxable: boolean;
  bpjsTkEnrolled: boolean;
  bpjsKesEnrolled: boolean;
  taperaEnrolled: boolean;
  effectiveDate: string;
  endDate?: string;
  isActive: boolean;
  notes: string;
};

export type PayrollRun = {
  id: string;
  runName: string;
  periodMonth: number;
  periodYear: number;
  status: string;
  totalEmployees: number;
  totalGross: string;
  totalDeductions: string;
  totalNet: string;
  totalBpjsEmployee: string;
  totalBpjsEmployer: string;
  totalPph21: string;
  needsRecalculation: boolean;
  calculatedAt: string | null;
  approvedAt: string | null;
  approvedByName: string;
  paidAt: string | null;
  notes: string;
  createdAt: string | null;
};

export type LoanInstallment = {
  loanId: string;
  loanType: string;
  installmentNo: number;
  tenorMonths: number;
  amount: string;
  remainingBefore: string;
  remainingAfter: string;
  label: string;
};

export type Payslip = {
  id: string;
  runId: string;
  periodMonth: number;
  periodYear: number;
  runStatus: string;
  employee: EmployeeRef;
  baseSalary: string;
  fixedAllowance: string;
  variableAllowance: string;
  transportAllowance: string;
  mealAllowance: string;
  housingAllowance: string;
  overtimePay: string;
  thr: string;
  bonus: string;
  grossSalary: string;
  bpjsTkJhtDeduction: string;
  bpjsTkJpDeduction: string;
  bpjsKesDeduction: string;
  taperaDeduction: string;
  pph21Deduction: string;
  unpaidLeaveDeduction: string;
  lateDeduction: string;
  loanDeduction: string;
  totalDeductions: string;
  netSalary: string;
  bpjsTkJhtEmployer: string;
  bpjsTkJpEmployer: string;
  bpjsTkJkkEmployer: string;
  bpjsTkJkmEmployer: string;
  bpjsKesEmployer: string;
  taperaEmployer: string;
  taxableIncome: string;
  ptkpAmount: string;
  pph21Annual: string;
  workingDays: number;
  presentDays: number;
  lateDays: number;
  unpaidLeaveDays: number;
  overtimeHours: number;
  prorateFactor: number;
  loanDetails: LoanInstallment[];
};

export type PayrollRunDetail = { run: PayrollRun; payslips: Payslip[] };

export type CalculatePayrollRunResponse = {
  run: PayrollRun;
  skipped: { employeeId: string; fullName: string; reason: string }[];
  message: string;
};

export type MarkPayrollRunPaidResponse = { run: PayrollRun; settledLoans: number; message: string };

export type PayrollAddition = {
  id: string;
  employee: EmployeeRef;
  periodMonth: number;
  periodYear: number;
  amount: string;
  label: string;
  createdAt: string | null;
};

export type LoanScheduleRow = {
  installmentNo: number;
  periodMonth: number;
  periodYear: number;
  amount: string;
  remainingAfter: string;
  isPaid: boolean;
};

export type Loan = {
  id: string;
  employee: EmployeeRef;
  loanType: string;
  principalAmount: string;
  interestRate: string;
  tenorMonths: number;
  monthlyInstallment: string;
  remainingBalance: string;
  paidAmount: string;
  status: string;
  isActive: boolean;
  purpose: string;
  firstInstallmentMonth: number;
  firstInstallmentYear: number;
  approvedByName: string;
  approvedAt: string | null;
  rejectionReason: string;
  createdAt: string | null;
  schedule: LoanScheduleRow[];
};

export type TodayShift = { name: string; startTime: string; endTime: string; lateToleranceMinutes: number };

export type MyHome = {
  employee: EmployeeRef;
  today: string;
  todayShift?: TodayShift;
  todayAttendance?: AttendanceRecord;
  week: TeamScheduleDay[];
  leaveBalance: LeaveBalance;
  unreadAnnouncements: number;
  pendingApprovals: number;
  holidayName: string;
};

export type MyApprovals = { leaves: Leave[]; overtime: Overtime[] };

export type LabelCount = { label: string; count: number };

export type HrisReport = {
  month: number;
  year: number;
  totalActive: number;
  newHires: number;
  turnoverCount: number;
  turnoverRate: number;
  byStatus: LabelCount[];
  byDepartment: LabelCount[];
  monthlyTrend: LabelCount[];
  attendanceRecords: number;
  presentCount: number;
  absentCount: number;
  lateCount: number;
  presentRate: number;
  lateRate: number;
  avgWorkHours: number;
  dailyTrend: { date: string; present: number; absent: number; late: number }[];
  leaveApprovedCount: number;
  leavePendingCount: number;
  leaveTotalDays: number;
  leaveByType: { type: string; days: number }[];
};

export type UserOption = { id: string; name: string; email: string };
