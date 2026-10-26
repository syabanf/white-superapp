import { toQuery } from "@/features/erp/hr/format";

/** Drill-down URLs into the HR list pages, using their existing params. */
export const hrLinks = {
  attendance: (p: { from: string; to: string; late?: boolean; status?: string }) =>
    `/erp/hr/attendance${toQuery({ tab: "list", from: p.from, to: p.to, late: p.late ? "1" : undefined, status: p.status })}`,
  leave: (p: { from?: string; to?: string; type?: string; status?: string }) => `/erp/hr/leave${toQuery(p)}`,
  leaveApprovals: () => "/erp/hr/leave?tab=approvals",
  overtime: (month: string, status = "approved") => `/erp/hr/overtime${toQuery({ month, status })}`,
  overtimeApprovals: () => "/erp/hr/overtime?tab=approvals",
  employees: (dept?: string) => `/erp/hr/employees${toQuery({ dept })}`,
  employee: (id: string) => `/erp/hr/employees/${id}`,
  contracts: (p: { type?: string; status?: string }) => `/erp/hr/contracts${toQuery(p)}`,
  expiring: () => "/erp/hr/contracts?tab=expiring&days=90",
  payroll: () => "/erp/hr/payroll",
  loans: () => "/erp/hr/payroll?tab=loans&status=approved",
};
