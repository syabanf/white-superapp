"use server";

import { revalidatePath } from "next/cache";
import { z } from "zod";
import { db } from "@/lib/db";
import { todayUtc } from "@/lib/dates";
import { fail, ok, type ActionResult } from "@/lib/action-result";
import { MAX_COMPETITOR_DOMAINS, normalizeDomain, siteHost } from "@/lib/metrics/seo-suite";
import { t } from "@/i18n/id";
import { errorCode, errorMessage, finishJob, guardProperty, startJob } from "./access";
import { s } from "./strings";
import { refreshBacklinks as refreshBacklinksCore, refreshDomains as refreshDomainsCore } from "./tracker";

const propSchema = z.object({ clientId: z.string().min(1), propertyId: z.string().min(1) });

// ── Backlink ──────────────────────────────────────────────────

export async function refreshBacklinks(input: z.infer<typeof propSchema>): Promise<ActionResult<{ added: number; lost: number }>> {
  const parsed = propSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const g = await guardProperty(parsed.data.clientId, parsed.data.propertyId);
  if (!g.ok) return g;
  const job = await startJob(parsed.data.clientId, "SEO_SUITE_BACKLINKS", g.data.property.siteUrl);
  if (!job.ok) return job;
  try {
    const res = await refreshBacklinksCore(g.data.property, todayUtc());
    await finishJob(job.data, { ok: true, message: `+${res.added} / −${res.lost}` });
    revalidatePath(`/clients/${g.data.property.slug}/seo/backlinks`);
    return ok(res);
  } catch (e) {
    await finishJob(job.data, { ok: false, error: errorMessage(e) });
    return fail(errorMessage(e), errorCode(e));
  }
}

// ── Kompetitor domain ─────────────────────────────────────────

const addSchema = propSchema.extend({ domain: z.string().min(1).max(253) });

export async function addCompetitorDomain(input: z.infer<typeof addSchema>): Promise<ActionResult<{ id: string; domain: string }>> {
  const parsed = addSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const domain = normalizeDomain(parsed.data.domain);
  if (!domain) return fail(s.domainInvalid, "VALIDATION");
  const g = await guardProperty(parsed.data.clientId, parsed.data.propertyId, { manage: true });
  if (!g.ok) return g;
  const p = g.data.property;
  if (domain === siteHost(p.siteUrl)) return fail(s.domainInvalid, "VALIDATION");
  const [count, existing] = await Promise.all([
    db.competitorDomain.count({ where: { propertyId: p.id } }),
    db.competitorDomain.findUnique({ where: { propertyId_domain: { propertyId: p.id, domain } }, select: { id: true } }),
  ]);
  if (existing) return fail(s.domainExists, "VALIDATION");
  if (count >= MAX_COMPETITOR_DOMAINS) return fail(s.domainMax, "VALIDATION");
  const row = await db.competitorDomain.create({ data: { propertyId: p.id, domain } });
  try {
    await refreshDomainsCore(p, todayUtc(), [domain]);
  } catch {
    // benchmark row will be filled by the next refresh / daily job
  }
  revalidatePath(`/clients/${p.slug}/seo/competitors`);
  revalidatePath(`/clients/${p.slug}/seo/research`);
  return ok({ id: row.id, domain });
}

const removeSchema = z.object({ clientId: z.string().min(1), id: z.string().min(1) });

export async function removeCompetitorDomain(input: z.infer<typeof removeSchema>): Promise<ActionResult<void>> {
  const parsed = removeSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const row = await db.competitorDomain.findUnique({ where: { id: parsed.data.id }, select: { propertyId: true, domain: true } });
  if (!row) return fail(t.errors.NOT_FOUND, "NOT_FOUND");
  const g = await guardProperty(parsed.data.clientId, row.propertyId, { manage: true });
  if (!g.ok) return g;
  await db.$transaction([
    db.competitorDomain.delete({ where: { id: parsed.data.id } }),
    db.domainSnapshot.deleteMany({ where: { propertyId: row.propertyId, domain: row.domain } }),
  ]);
  revalidatePath(`/clients/${g.data.property.slug}/seo/competitors`);
  revalidatePath(`/clients/${g.data.property.slug}/seo/research`);
  return ok(undefined);
}

export async function refreshDomains(input: z.infer<typeof propSchema>): Promise<ActionResult<{ domains: number }>> {
  const parsed = propSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const g = await guardProperty(parsed.data.clientId, parsed.data.propertyId);
  if (!g.ok) return g;
  const job = await startJob(parsed.data.clientId, "SEO_SUITE_DOMAINS", g.data.property.siteUrl);
  if (!job.ok) return job;
  try {
    const domains = await refreshDomainsCore(g.data.property, todayUtc());
    await finishJob(job.data, { ok: true, message: `${domains} domain` });
    revalidatePath(`/clients/${g.data.property.slug}/seo/competitors`);
    return ok({ domains });
  } catch (e) {
    await finishJob(job.data, { ok: false, error: errorMessage(e) });
    return fail(errorMessage(e), errorCode(e));
  }
}
