"use server";

import { revalidatePath } from "next/cache";
import type { z } from "zod";
import { fail, type ActionResult } from "@/lib/action-result";
import { gatewayFetch } from "@/lib/erp/gateway";
import { runGateway, zodFieldErrors } from "@/features/erp/shared/action";
import { leaveSchema, loanSchema, overtimeSchema } from "@/features/erp/hr/schemas";
import { th } from "@/features/erp/hr/strings";
import type { AttendanceRecord, GeoLocation, Leave, Loan, Overtime } from "@/features/erp/hr/types";
import { tm } from "./strings";

const M = "/v1/hr/me";
// base64 of a 2 MB image is about 2.67 MB of text.
const MAX_PHOTO_CHARS = Math.ceil((2 * 1024 * 1024 * 4) / 3) + 4;

async function send<T>(path: string, body: unknown): Promise<ActionResult<T>> {
  const res = await runGateway(() => gatewayFetch<T>(path, { method: "POST", body: JSON.stringify(body) }));
  if (res.ok) {
    revalidatePath("/erp/me", "layout");
    revalidatePath("/erp/hr", "layout");
  }
  return res;
}

function invalid(err: z.ZodError) {
  return fail<never>(th.checkForm, "VALIDATION", zodFieldErrors(err));
}

type ClockInput = { photo?: string; location?: GeoLocation | null; notes?: string };

function clockBody(input: ClockInput) {
  if (input.photo && input.photo.length > MAX_PHOTO_CHARS) return null;
  return {
    photo: input.photo || undefined,
    location: input.location ?? undefined,
    notes: input.notes?.trim() || undefined,
  };
}

/** `photo` is raw base64 (no data: prefix), as the proto `bytes` field expects. */
export async function myClockInAction(input: ClockInput) {
  const body = clockBody(input);
  if (!body) return fail<AttendanceRecord>(tm.clock.photoTooLarge, "VALIDATION");
  return send<AttendanceRecord>(`${M}/attendance:clockIn`, body);
}

export async function myClockOutAction(attendanceId: string, input: ClockInput) {
  const body = clockBody(input);
  if (!body) return fail<AttendanceRecord>(tm.clock.photoTooLarge, "VALIDATION");
  return send<AttendanceRecord>(`${M}/attendance/${attendanceId}:clockOut`, body);
}

export async function createMyLeaveAction(input: z.input<typeof leaveSchema>) {
  const parsed = leaveSchema.safeParse(input);
  if (!parsed.success) return invalid(parsed.error);
  const { leaveType, startDate, endDate, reason } = parsed.data;
  return send<Leave>(`${M}/leaves`, { leaveType, startDate, endDate, reason });
}

export async function cancelMyLeaveAction(id: string) {
  return send<Leave>(`${M}/leaves/${id}:cancel`, {});
}

export async function createMyOvertimeAction(input: z.input<typeof overtimeSchema>) {
  const parsed = overtimeSchema.safeParse(input);
  if (!parsed.success) return invalid(parsed.error);
  const { date, startTime, endTime, reason } = parsed.data;
  return send<Overtime>(`${M}/overtime`, { date, startTime, endTime, reason });
}

/** cancel my own request, or approve / reject a company assignment. */
export async function decideMyOvertimeAction(id: string, action: "approve" | "reject" | "cancel", reason = "") {
  if (action === "reject" && !reason.trim()) return fail<Overtime>(th.rejectReasonRequired, "VALIDATION");
  return send<Overtime>(`${M}/overtime/${id}:decide`, { action, rejectionReason: reason.trim() });
}

export async function createMyLoanAction(input: z.input<typeof loanSchema>) {
  const parsed = loanSchema.safeParse(input);
  if (!parsed.success) return invalid(parsed.error);
  const { loanType, principalAmount, tenorMonths, purpose } = parsed.data;
  return send<Loan>(`${M}/loans`, { loanType, principalAmount, tenorMonths, purpose });
}

export async function cancelMyLoanAction(id: string) {
  return send<Loan>(`${M}/loans/${id}:cancel`, {});
}

/** Opening an announcement marks it read. */
export async function readMyAnnouncementAction(id: string) {
  const res = await runGateway(() => gatewayFetch(`${M}/announcements/${id}`));
  if (res.ok) revalidatePath("/erp/me", "layout");
  return res;
}
