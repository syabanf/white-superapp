"use server";

import { revalidatePath } from "next/cache";
import { z } from "zod";
import { db } from "@/lib/db";
import { withClientMode } from "@/lib/client-mode";
import { fail, isProviderError, ok, type ActionResult } from "@/lib/action-result";
import { getSessionUser, requireClientAccess } from "@/lib/rbac";
import { t } from "@/i18n/id";
import { BACKFILL_DAYS, detectAdAnomalies } from "./detect";
import { computeNextRun } from "./lib";
import { runScheduleNow } from "./schedules";
import { aa } from "./strings";

/** Managers/admins only. Returns the client or null. */
async function manageable(slug: string) {
  const { client, canManage } = await requireClientAccess(slug);
  return canManage ? client : null;
}

function revalidateAll(slug: string) {
  revalidatePath(`/clients/${slug}/ads`);
  revalidatePath(`/clients/${slug}/reports`);
}

// ── Alerts ────────────────────────────────────────────────────

/** "Periksa sekarang": run detection over the last 7 days on demand. */
async function checkAdAnomaliesInner(slug: string): Promise<ActionResult<{ created: number; found: number }>> {
  const client = await manageable(slug);
  if (!client) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  try {
    const r = await detectAdAnomalies({ clientId: client.id, lookbackDays: BACKFILL_DAYS });
    revalidateAll(slug);
    return ok({ created: r.created, found: r.found });
  } catch (e) {
    console.error("checkAdAnomalies failed", e);
    return fail(t.errors.UNKNOWN);
  }
}

/** Resolve an alert (and its repeats for the same rule + campaign). */
async function resolveAdAlertInner(slug: string, alertId: string): Promise<ActionResult<{ count: number }>> {
  const client = await manageable(slug);
  if (!client) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const alert = await db.adAlert.findUnique({ where: { id: alertId }, select: { clientId: true, rule: true, campaignId: true } });
  if (!alert || alert.clientId !== client.id) return fail(t.errors.NOT_FOUND);
  const res = await db.adAlert.updateMany({
    where: { clientId: client.id, rule: alert.rule, campaignId: alert.campaignId, resolvedAt: null },
    data: { resolvedAt: new Date() },
  });
  revalidatePath(`/clients/${slug}/ads`);
  return ok({ count: res.count });
}

// ── Report schedules ──────────────────────────────────────────

const scheduleSchema = z.object({
  name: z.string().trim().min(1, aa.schedules.nameRequired).max(120),
  cadence: z.enum(["WEEKLY", "MONTHLY"]),
  modules: z.array(z.enum(["SOCIAL", "SEO", "ADS"])).min(1, aa.schedules.noModules),
  withInsight: z.boolean(),
});
export type ScheduleInput = z.input<typeof scheduleSchema>;

type Parsed = { valid: true; data: z.output<typeof scheduleSchema> } | { valid: false; error: ActionResult<never> };

function parseSchedule(input: unknown): Parsed {
  const parsed = scheduleSchema.safeParse(input);
  if (parsed.success) return { valid: true, data: parsed.data };
  const flat = z.flattenError(parsed.error);
  const first = Object.values(flat.fieldErrors).flat()[0] ?? t.errors.VALIDATION;
  return { valid: false, error: fail<never>(first, "VALIDATION", flat.fieldErrors as Record<string, string[]>) };
}

async function createReportScheduleInner(slug: string, input: ScheduleInput): Promise<ActionResult<{ id: string }>> {
  const client = await manageable(slug);
  if (!client) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const p = parseSchedule(input);
  if (!p.valid) return p.error;
  const session = await getSessionUser();
  const row = await db.reportSchedule.create({
    data: { clientId: client.id, ...p.data, nextRunAt: computeNextRun(p.data.cadence, new Date()), createdById: session?.id ?? null },
  });
  revalidatePath(`/clients/${slug}/reports`);
  return ok({ id: row.id });
}

async function updateReportScheduleInner(slug: string, id: string, input: ScheduleInput): Promise<ActionResult<{ id: string }>> {
  const client = await manageable(slug);
  if (!client) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const p = parseSchedule(input);
  if (!p.valid) return p.error;
  const existing = await db.reportSchedule.findUnique({ where: { id }, select: { clientId: true, cadence: true } });
  if (!existing || existing.clientId !== client.id) return fail(t.errors.NOT_FOUND);
  await db.reportSchedule.update({
    where: { id },
    data: { ...p.data, ...(existing.cadence !== p.data.cadence ? { nextRunAt: computeNextRun(p.data.cadence, new Date()) } : {}) },
  });
  revalidatePath(`/clients/${slug}/reports`);
  return ok({ id });
}

async function toggleReportScheduleInner(slug: string, id: string, enabled: boolean): Promise<ActionResult<{ enabled: boolean }>> {
  const client = await manageable(slug);
  if (!client) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const existing = await db.reportSchedule.findUnique({ where: { id }, select: { clientId: true, cadence: true, nextRunAt: true } });
  if (!existing || existing.clientId !== client.id) return fail(t.errors.NOT_FOUND);
  const now = new Date();
  // Re-enabling after a long pause continues from the next period, not a stale one.
  const nextRunAt = enabled && existing.nextRunAt < now ? computeNextRun(existing.cadence, now) : existing.nextRunAt;
  await db.reportSchedule.update({ where: { id }, data: { enabled: Boolean(enabled), nextRunAt } });
  revalidatePath(`/clients/${slug}/reports`);
  return ok({ enabled: Boolean(enabled) });
}

async function deleteReportScheduleInner(slug: string, id: string): Promise<ActionResult<{ id: string }>> {
  const client = await manageable(slug);
  if (!client) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const existing = await db.reportSchedule.findUnique({ where: { id }, select: { clientId: true } });
  if (!existing || existing.clientId !== client.id) return fail(t.errors.NOT_FOUND);
  await db.reportSchedule.delete({ where: { id } });
  revalidatePath(`/clients/${slug}/reports`);
  return ok({ id });
}

/** "Jalankan sekarang": generate this schedule's report immediately (demo affordance). */
async function runReportScheduleNowInner(slug: string, id: string): Promise<ActionResult<{ reportId: string; withAi: boolean }>> {
  const client = await manageable(slug);
  if (!client) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const existing = await db.reportSchedule.findUnique({ where: { id }, select: { clientId: true } });
  if (!existing || existing.clientId !== client.id) return fail(t.errors.NOT_FOUND);
  try {
    const run = await runScheduleNow(id);
    if (!run) return fail(t.errors.NOT_FOUND);
    revalidatePath(`/clients/${slug}/reports`);
    return ok({ reportId: run.reportId, withAi: run.withAi });
  } catch (e) {
    if (isProviderError(e)) return fail(t.errors[e.code] ?? t.errors.UNKNOWN, e.code);
    console.error("runReportScheduleNow failed", e);
    return fail(t.errors.UNKNOWN);
  }
}

// ── Mode demo per klien: klien isDemo selalu memakai adapter mock ──

export async function checkAdAnomalies(...args: Parameters<typeof checkAdAnomaliesInner>): ReturnType<typeof checkAdAnomaliesInner> {
  return withClientMode({ slug: args[0] }, () => checkAdAnomaliesInner(...args));
}
export async function resolveAdAlert(...args: Parameters<typeof resolveAdAlertInner>): ReturnType<typeof resolveAdAlertInner> {
  return withClientMode({ slug: args[0] }, () => resolveAdAlertInner(...args));
}
export async function createReportSchedule(...args: Parameters<typeof createReportScheduleInner>): ReturnType<typeof createReportScheduleInner> {
  return withClientMode({ slug: args[0] }, () => createReportScheduleInner(...args));
}
export async function updateReportSchedule(...args: Parameters<typeof updateReportScheduleInner>): ReturnType<typeof updateReportScheduleInner> {
  return withClientMode({ slug: args[0] }, () => updateReportScheduleInner(...args));
}
export async function toggleReportSchedule(...args: Parameters<typeof toggleReportScheduleInner>): ReturnType<typeof toggleReportScheduleInner> {
  return withClientMode({ slug: args[0] }, () => toggleReportScheduleInner(...args));
}
export async function deleteReportSchedule(...args: Parameters<typeof deleteReportScheduleInner>): ReturnType<typeof deleteReportScheduleInner> {
  return withClientMode({ slug: args[0] }, () => deleteReportScheduleInner(...args));
}
export async function runReportScheduleNow(...args: Parameters<typeof runReportScheduleNowInner>): ReturnType<typeof runReportScheduleNowInner> {
  return withClientMode({ slug: args[0] }, () => runReportScheduleNowInner(...args));
}
