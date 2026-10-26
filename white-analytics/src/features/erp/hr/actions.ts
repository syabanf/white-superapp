"use server";

import { revalidatePath } from "next/cache";
import type { z } from "zod";
import { fail, type ActionResult } from "@/lib/action-result";
import { gatewayFetch } from "@/lib/erp/gateway";
import { runGateway, zodFieldErrors } from "@/features/erp/shared/action";
import {
  additionSchema,
  announcementSchema,
  contractSchema,
  departmentSchema,
  employeeSchema,
  holidaySchema,
  leaveSchema,
  loanSchema,
  overtimeSchema,
  positionSchema,
  salarySchema,
  shiftSchema,
  statusOptionSchema,
} from "./schemas";
import { th } from "./strings";
import type {
  AttendanceRecord,
  CalculatePayrollRunResponse,
  Contract,
  ContractActionResult,
  EmployeeProfile,
  Leave,
  LeaveBalance,
  Loan,
  MarkPayrollRunPaidResponse,
  Offboarding,
  OnboardingChecklist,
  Overtime,
  PayrollRun,
  PayrollSettings,
  ScheduleDay,
} from "./types";

const H = "/v1/hr";

/** Calls the gateway and refreshes every HR and ESS page on success. */
async function send<T>(path: string, method: string, body?: unknown): Promise<ActionResult<T>> {
  const res = await runGateway(() =>
    gatewayFetch<T>(path, { method, body: body === undefined ? undefined : JSON.stringify(body) }),
  );
  if (res.ok) {
    revalidatePath("/erp/hr", "layout");
    revalidatePath("/erp/me", "layout");
  }
  return res;
}

function validate<S extends z.ZodType>(
  schema: S,
  input: unknown,
): { data: z.output<S>; error?: never } | { data?: never; error: ActionResult<never> } {
  const parsed = schema.safeParse(input);
  if (parsed.success) return { data: parsed.data };
  return { error: fail(th.checkForm, "VALIDATION", zodFieldErrors(parsed.error)) };
}

// Employees

export async function saveEmployeeAction(id: string | null, input: z.input<typeof employeeSchema>) {
  const v = validate(employeeSchema, input);
  if (v.error) return v.error;
  return id
    ? send<EmployeeProfile>(`${H}/employees/${id}`, "PUT", v.data)
    : send<EmployeeProfile>(`${H}/employees`, "POST", v.data);
}

export async function deleteEmployeeAction(id: string) {
  return send(`${H}/employees/${id}`, "DELETE");
}

export async function linkEmployeeUserAction(id: string, userId: string) {
  return send<EmployeeProfile>(`${H}/employees/${id}:linkUser`, "POST", { userId });
}

export async function deleteDocumentAction(documentId: string) {
  return send(`${H}/documents/${documentId}`, "DELETE");
}

// Master data

export async function saveDepartmentAction(id: string | null, input: z.input<typeof departmentSchema>) {
  const v = validate(departmentSchema, input);
  if (v.error) return v.error;
  return id ? send(`${H}/departments/${id}`, "PUT", v.data) : send(`${H}/departments`, "POST", v.data);
}

export async function deleteDepartmentAction(id: string) {
  return send(`${H}/departments/${id}`, "DELETE");
}

export async function savePositionAction(id: string | null, input: z.input<typeof positionSchema>) {
  const v = validate(positionSchema, input);
  if (v.error) return v.error;
  return id ? send(`${H}/positions/${id}`, "PUT", v.data) : send(`${H}/positions`, "POST", v.data);
}

export async function deletePositionAction(id: string) {
  return send(`${H}/positions/${id}`, "DELETE");
}

export async function saveEmploymentStatusAction(input: z.input<typeof statusOptionSchema>) {
  const v = validate(statusOptionSchema, input);
  if (v.error) return v.error;
  return send(`${H}/employment-statuses/${v.data.code}`, "PUT", v.data);
}

export async function deleteEmploymentStatusAction(code: string) {
  return send(`${H}/employment-statuses/${encodeURIComponent(code)}`, "DELETE");
}

// Schedules, shifts, holidays

export async function setEmployeeScheduleAction(employeeId: string, effectiveFrom: string, days: ScheduleDay[]) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(effectiveFrom)) return fail(th.invalidDate, "VALIDATION");
  return send(`${H}/employees/${employeeId}/schedule`, "PUT", {
    effectiveFrom,
    days: days.map((d) => ({ dayOfWeek: d.dayOfWeek, shiftId: d.shiftId })),
  });
}

export async function saveShiftAction(id: string | null, input: z.input<typeof shiftSchema>) {
  const v = validate(shiftSchema, input);
  if (v.error) return v.error;
  return id ? send(`${H}/shifts/${id}`, "PUT", v.data) : send(`${H}/shifts`, "POST", v.data);
}

export async function deleteShiftAction(id: string) {
  return send<{ message: string }>(`${H}/shifts/${id}`, "DELETE");
}

export async function saveHolidayAction(id: string | null, input: z.input<typeof holidaySchema>) {
  const v = validate(holidaySchema, input);
  if (v.error) return v.error;
  return id ? send(`${H}/holidays/${id}`, "PUT", v.data) : send(`${H}/holidays`, "POST", v.data);
}

export async function deleteHolidayAction(id: string) {
  return send(`${H}/holidays/${id}`, "DELETE");
}

// Attendance

export async function validateAttendanceAction(id: string, status: string, notes: string) {
  return send<AttendanceRecord>(`${H}/attendance/${id}:validate`, "POST", { status, notes: notes.trim() });
}

// Leave

export async function createLeaveAction(input: z.input<typeof leaveSchema>) {
  const v = validate(leaveSchema, input);
  if (v.error) return v.error;
  return send<Leave>(`${H}/leaves`, "POST", v.data);
}

export async function decideLeaveAction(id: string, approve: boolean, rejectionReason = "") {
  if (!approve && !rejectionReason.trim()) return fail(th.rejectReasonRequired, "VALIDATION");
  return send<Leave>(`${H}/leaves/${id}:decide`, "POST", { approve, rejectionReason: rejectionReason.trim() });
}

export async function cancelLeaveAction(id: string) {
  return send<Leave>(`${H}/leaves/${id}:cancel`, "POST", {});
}

export async function updateLeaveBalanceAction(
  employeeId: string,
  year: number,
  annualLeaveTotal: number,
  annualLeaveUsed: number,
) {
  if (![annualLeaveTotal, annualLeaveUsed].every((n) => Number.isInteger(n) && n >= 0))
    return fail(th.required, "VALIDATION");
  return send<LeaveBalance>(`${H}/employees/${employeeId}/leave-balance`, "PUT", {
    year,
    annualLeaveTotal,
    annualLeaveUsed,
  });
}

// Overtime

export async function createOvertimeAction(input: z.input<typeof overtimeSchema>) {
  const v = validate(overtimeSchema, input);
  if (v.error) return v.error;
  return send<Overtime>(`${H}/overtime`, "POST", v.data);
}

export async function decideOvertimeAction(id: string, action: "approve" | "reject" | "cancel", rejectionReason = "") {
  if (action === "reject" && !rejectionReason.trim()) return fail(th.rejectReasonRequired, "VALIDATION");
  return send<Overtime>(`${H}/overtime/${id}:decide`, "POST", { action, rejectionReason: rejectionReason.trim() });
}

// Contracts

export async function saveContractAction(id: string | null, input: z.input<typeof contractSchema>) {
  const v = validate(contractSchema, input);
  if (v.error) return v.error;
  return id
    ? send<Contract>(`${H}/contracts/${id}`, "PUT", v.data)
    : send<Contract>(`${H}/contracts`, "POST", v.data);
}

export async function contractActionAction(
  id: string,
  input: { action: string; endDate?: string; reason?: string; signedAt?: string },
) {
  if (input.action === "terminate" && !input.reason?.trim()) return fail(th.rejectReasonRequired, "VALIDATION");
  return send<ContractActionResult>(`${H}/contracts/${id}:action`, "POST", input);
}

export async function deleteContractAction(id: string) {
  return send(`${H}/contracts/${id}`, "DELETE");
}

// Announcements

export async function saveAnnouncementAction(id: string | null, input: z.input<typeof announcementSchema>) {
  const v = validate(announcementSchema, input);
  if (v.error) return v.error;
  const body = {
    ...v.data,
    publishAt: v.data.publishAt || undefined,
    expiresAt: v.data.expiresAt || undefined,
  };
  return id ? send(`${H}/announcements/${id}`, "PUT", body) : send(`${H}/announcements`, "POST", body);
}

export async function deleteAnnouncementAction(id: string) {
  return send(`${H}/announcements/${id}`, "DELETE");
}

// Onboarding and offboarding

export async function applyOnboardingTemplateAction(employeeId: string) {
  return send<OnboardingChecklist>(`${H}/employees/${employeeId}/onboarding:applyTemplate`, "POST", {});
}

export async function addOnboardingTaskAction(
  employeeId: string,
  input: { taskName: string; category: string; priority: number; dueDate: string },
) {
  if (!input.taskName.trim()) return fail(th.required, "VALIDATION", { taskName: [th.required] });
  return send(`${H}/employees/${employeeId}/onboarding`, "POST", { ...input, taskName: input.taskName.trim() });
}

export async function updateOnboardingTaskAction(taskId: string, input: { isCompleted?: boolean; notes?: string }) {
  return send(`${H}/onboarding/${taskId}`, "PUT", input);
}

export async function deleteOnboardingTaskAction(taskId: string) {
  return send(`${H}/onboarding/${taskId}`, "DELETE");
}

export async function startOffboardingAction(
  employeeId: string,
  input: { resignationType: string; resignationDate: string; lastWorkingDay: string; reason: string },
) {
  if (!input.resignationDate || !input.lastWorkingDay) return fail(th.invalidDate, "VALIDATION");
  if (input.lastWorkingDay < input.resignationDate) return fail(th.endBeforeStart, "VALIDATION");
  return send<Offboarding>(`${H}/employees/${employeeId}/offboarding`, "POST", input);
}

export async function updateOffboardingAction(
  employeeId: string,
  input: { status?: string; clearanceType?: string; cleared?: boolean; exitInterviewNotes?: string; notes?: string },
) {
  return send<Offboarding>(`${H}/employees/${employeeId}/offboarding`, "PUT", input);
}

// Payroll

export async function updatePayrollSettingsAction(input: PayrollSettings) {
  return send<PayrollSettings>(`${H}/payroll/settings`, "PUT", { ...input, updatedAt: undefined });
}

export async function saveSalaryAction(id: string | null, input: z.input<typeof salarySchema>) {
  const v = validate(salarySchema, input);
  if (v.error) return v.error;
  return id ? send(`${H}/salaries/${id}`, "PUT", v.data) : send(`${H}/salaries`, "POST", v.data);
}

export async function deactivateSalaryAction(id: string) {
  return send(`${H}/salaries/${id}`, "DELETE");
}

export async function createPayrollRunAction(periodMonth: number, periodYear: number, runName: string) {
  if (periodMonth < 1 || periodMonth > 12 || periodYear < 2000) return fail(th.required, "VALIDATION");
  return send<PayrollRun>(`${H}/payroll/runs`, "POST", { periodMonth, periodYear, runName: runName.trim() });
}

export async function calculatePayrollRunAction(id: string, includeThr: boolean) {
  return send<CalculatePayrollRunResponse>(`${H}/payroll/runs/${id}:calculate`, "POST", { includeThr });
}

export async function approvePayrollRunAction(id: string) {
  return send<PayrollRun>(`${H}/payroll/runs/${id}:approve`, "POST", {});
}

export async function markPayrollRunPaidAction(id: string, notes: string) {
  return send<MarkPayrollRunPaidResponse>(`${H}/payroll/runs/${id}:markPaid`, "POST", { notes: notes.trim() });
}

export async function deletePayrollRunAction(id: string) {
  return send(`${H}/payroll/runs/${id}`, "DELETE");
}

export async function createPayrollAdditionAction(input: z.input<typeof additionSchema>) {
  const v = validate(additionSchema, input);
  if (v.error) return v.error;
  return send(`${H}/payroll/additions`, "POST", v.data);
}

export async function deletePayrollAdditionAction(id: string) {
  return send(`${H}/payroll/additions/${id}`, "DELETE");
}

export async function createLoanAction(input: z.input<typeof loanSchema>) {
  const v = validate(loanSchema, input);
  if (v.error) return v.error;
  return send<Loan>(`${H}/loans`, "POST", v.data);
}

export async function decideLoanAction(id: string, approve: boolean, rejectionReason = "") {
  if (!approve && !rejectionReason.trim()) return fail(th.rejectReasonRequired, "VALIDATION");
  return send<Loan>(`${H}/loans/${id}:decide`, "POST", { approve, rejectionReason: rejectionReason.trim() });
}
