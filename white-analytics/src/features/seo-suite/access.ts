/**
 * Shared guards for SEO suite server actions: session + client membership (MANAGER for writes),
 * property lookup, and a SyncJob-based cooldown so "Perbarui" cannot hammer the provider.
 */
import "server-only";
import { db } from "@/lib/db";
import { getSessionUser } from "@/lib/rbac";
import { fail, isProviderError, type ActionResult } from "@/lib/action-result";
import { t } from "@/i18n/id";
import { s } from "./strings";

export const REFRESH_COOLDOWN_MS = 10 * 60 * 1000;

export type Guarded = { userId: string; property: { id: string; siteUrl: string; locationCode: number; languageCode: string; clientId: string; slug: string } };

export async function guardProperty(clientId: string, propertyId: string, opts: { manage?: boolean } = {}): Promise<ActionResult<Guarded>> {
  const user = await getSessionUser();
  if (!user) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  if (user.role !== "ADMIN") {
    const m = await db.clientMember.findUnique({ where: { userId_clientId: { userId: user.id, clientId } }, select: { role: true } });
    if (!m) return fail(s.unauthorized, "UNAUTHORIZED");
    if (opts.manage && m.role !== "MANAGER") return fail(s.viewer, "UNAUTHORIZED");
  }
  const property = await db.seoProperty.findFirst({
    where: { id: propertyId, clientId },
    select: { id: true, siteUrl: true, locationCode: true, languageCode: true, clientId: true, client: { select: { slug: true } } },
  });
  if (!property) return fail(s.noProperty, "NOT_FOUND");
  return { ok: true, data: { userId: user.id, property: { ...property, slug: property.client.slug } } };
}

/** Start a SyncJob unless one of `kind` ran within the cooldown. Returns the job id or a fail(). */
export async function startJob(clientId: string, kind: string, message?: string): Promise<ActionResult<string>> {
  const last = await db.syncJob.findFirst({ where: { clientId, kind }, orderBy: { startedAt: "desc" }, select: { startedAt: true } });
  if (last && Date.now() - last.startedAt.getTime() < REFRESH_COOLDOWN_MS) return fail(s.refreshThrottled);
  const job = await db.syncJob.create({ data: { clientId, kind, status: "RUNNING", message } });
  return { ok: true, data: job.id };
}

export async function finishJob(jobId: string, outcome: { ok: true; message: string } | { ok: false; error: string }): Promise<void> {
  await db.syncJob.update({
    where: { id: jobId },
    data: outcome.ok ? { status: "SUCCESS", finishedAt: new Date(), message: outcome.message } : { status: "FAILED", finishedAt: new Date(), error: outcome.error.slice(0, 900) },
  });
}

export function errorMessage(e: unknown): string {
  if (isProviderError(e)) return e.message;
  if (e instanceof Error) return e.message;
  return t.errors.UNKNOWN;
}

export function errorCode(e: unknown) {
  return isProviderError(e) ? e.code : undefined;
}
