"use server";

import { revalidatePath } from "next/cache";
import { z } from "zod";
import { db } from "@/lib/db";
import { getSessionUser } from "@/lib/rbac";
import { fail, ok, type ActionResult } from "@/lib/action-result";
import { t } from "@/i18n/id";
import { s } from "./strings";
import { syncSeoForClient } from "./sync";
import { AUDIT_COOLDOWN_MS, runAuditCore } from "./audit-core";

const inputSchema = z.object({
  clientId: z.string().min(1),
  propertyId: z.string().min(1),
  url: z.string().url().optional(),
});

async function guard(clientId: string, propertyId: string) {
  const user = await getSessionUser();
  if (!user) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  if (user.role !== "ADMIN") {
    const membership = await db.clientMember.findUnique({
      where: { userId_clientId: { userId: user.id, clientId } },
      select: { userId: true },
    });
    if (!membership) return fail(s.auditUnauthorized, "UNAUTHORIZED");
  }
  const property = await db.seoProperty.findFirst({ where: { id: propertyId, clientId }, include: { client: { select: { slug: true } } } });
  if (!property) return fail(s.auditNoProperty, "NOT_FOUND");
  return ok(property);
}

export type RunAuditResult = { message: string; audits: number; pagesCrawled: number; issues: number };

/**
 * Run a real site audit: PageSpeed Insights (mobile + desktop) and the internal
 * cheerio crawler, concurrently, within a ~60s wall-clock budget.
 */
export async function runAudit(clientId: string, propertyId: string, url?: string): Promise<ActionResult<RunAuditResult>> {
  const parsed = inputSchema.safeParse({ clientId, propertyId, url });
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const g = await guard(parsed.data.clientId, parsed.data.propertyId);
  if (!g.ok) return g;
  const property = g.data;

  const lastAudit = await db.seoAudit.findFirst({ where: { propertyId: property.id }, orderBy: { runAt: "desc" }, select: { runAt: true } });
  if (lastAudit && Date.now() - lastAudit.runAt.getTime() < AUDIT_COOLDOWN_MS) {
    return fail(s.auditThrottled);
  }

  const res = await runAuditCore(property.id, parsed.data.url);

  const slug = property.client.slug;
  revalidatePath(`/clients/${slug}/seo/audit`);
  revalidatePath(`/clients/${slug}/seo`);

  if (res.allFailed) {
    return fail(`${s.auditAllFailed}: ${res.errorParts[0] ?? t.errors.UNKNOWN}`);
  }
  const headline = res.errorParts.length > 0 ? s.auditPartial : s.auditDone;
  return ok({
    message: `${headline}: ${res.parts.join(" · ")}${res.errorParts.length ? ` — ${res.errorParts[0]}` : ""}`,
    audits: res.audits,
    pagesCrawled: res.pagesCrawled,
    issues: res.issues,
  });
}

const cadenceSchema = z.object({
  clientId: z.string().min(1),
  propertyId: z.string().min(1),
  cadence: z.enum(["NONE", "WEEKLY", "DAILY"]),
});

/** "Jadwal audit": how often the daily cron re-runs the audit for this property. */
export async function setAuditCadence(input: z.infer<typeof cadenceSchema>): Promise<ActionResult<{ cadence: "NONE" | "WEEKLY" | "DAILY" }>> {
  const parsed = cadenceSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const g = await guard(parsed.data.clientId, parsed.data.propertyId);
  if (!g.ok) return g;
  await db.seoProperty.update({ where: { id: g.data.id }, data: { auditCadence: parsed.data.cadence } });
  revalidatePath(`/clients/${g.data.client.slug}/seo/audit`);
  return ok({ cadence: parsed.data.cadence });
}


/** "Sinkronkan" on the SEO overview: real GSC/GA4 pull when the property has a Google connection, otherwise a no-op demo log. */
export async function syncSeo(clientId: string): Promise<ActionResult<{ mode: "real" | "demo"; daily: number; ga4: number }>> {
  const user = await getSessionUser();
  if (!user) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const parsed = z.string().min(1).safeParse(clientId);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  if (user.role !== "ADMIN") {
    const membership = await db.clientMember.findUnique({ where: { userId_clientId: { userId: user.id, clientId: parsed.data } }, select: { role: true } });
    if (!membership) return fail(s.auditUnauthorized, "UNAUTHORIZED");
  }
  const client = await db.client.findUnique({ where: { id: parsed.data }, select: { slug: true } });
  if (!client) return fail(t.errors.NOT_FOUND, "NOT_FOUND");
  const r = await syncSeoForClient(parsed.data, { days: 90 });
  revalidatePath(`/clients/${client.slug}/seo`, "layout");
  revalidatePath(`/clients/${client.slug}`);
  const first = r.results[0];
  if (!first) return fail(s.auditNoProperty, "NOT_FOUND");
  if (first.error) return fail(first.error);
  return ok({ mode: first.mode, daily: r.results.reduce((a, x) => a + x.daily, 0), ga4: r.results.reduce((a, x) => a + x.ga4, 0) });
}
