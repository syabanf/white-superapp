import { toQuery } from "@/features/erp/hr/format";

type People = { dept?: string; employee?: string };

export type EmployeeQuery = {
  dept?: string;
  position?: string;
  level?: string;
  gender?: string;
  status?: string;
  inactive?: boolean;
};

/** Drill-down URLs into the HR list pages, carrying the dashboard filters they support. */
export const hrLinks = {
  attendance: (p: { from: string; to: string; late?: boolean; status?: string } & People) =>
    `/erp/hr/attendance${toQuery({
      tab: "list",
      from: p.from,
      to: p.to,
      late: p.late ? "1" : undefined,
      status: p.status,
      dept: p.dept,
      employee: p.employee,
    })}`,
  leave: (p: { from?: string; to?: string; type?: string; status?: string } & People) => `/erp/hr/leave${toQuery(p)}`,
  leaveApprovals: () => "/erp/hr/leave?tab=approvals",
  leaveBalances: () => "/erp/hr/leave?tab=balances",
  overtime: (month: string, p: { status?: string } & People = {}) =>
    `/erp/hr/overtime${toQuery({ month, status: p.status ?? "approved", dept: p.dept, employee: p.employee })}`,
  overtimeApprovals: () => "/erp/hr/overtime?tab=approvals",
  employees: (p: EmployeeQuery = {}) => `/erp/hr/employees${toQuery({ ...p, inactive: p.inactive ? "1" : undefined })}`,
  employee: (id: string) => `/erp/hr/employees/${id}`,
  contracts: (p: { type?: string; status?: string }) => `/erp/hr/contracts${toQuery(p)}`,
  expiring: () => "/erp/hr/contracts?tab=expiring&days=90",
  payroll: (year?: string | number) => `/erp/hr/payroll${toQuery({ year })}`,
  payrollRun: (id: string, p: People = {}) => `/erp/hr/payroll/runs/${id}${toQuery(p)}`,
  salaries: (p: People = {}) => `/erp/hr/payroll${toQuery({ tab: "salaries", ...p })}`,
  loans: () => "/erp/hr/payroll?tab=loans&status=approved",
};
