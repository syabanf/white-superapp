"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { z } from "zod";
import { db } from "@/lib/db";
import { fail, isProviderError, ok, type ActionResult } from "@/lib/action-result";
import { parseISODate, previousRange } from "@/lib/dates";
import { getSessionUser, requireClientAccess } from "@/lib/rbac";
import { generateInsight as providerGenerate, type InsightLanguage } from "@/lib/providers/openrouter";
import { buildInsightPayload, toRangeInfo } from "@/features/reports/insights";
import type { InsightModule } from "@/features/reports/queries";
import { t } from "@/i18n/id";
import { rs } from "@/features/reports/strings";

const MODULES = ["OVERVIEW", "SOCIAL", "SEO", "ADS"] as const;

function parseRange(rangeFrom: string, rangeTo: string) {
  const from = parseISODate(rangeFrom);
  const to = parseISODate(rangeTo);
  if (!from || !to || from > to) return null;
  return { from, to };
}

/**
 * Generate (or regenerate) the strategic insight for a module + range and cache it
 * in AiInsight. Accepts the client slug so RBAC runs through requireClientAccess.
 */
export async function generateInsight(
  slug: string,
  module: InsightModule,
  rangeFrom: string,
  rangeTo: string,
  language: InsightLanguage = "id",
): Promise<ActionResult<{ content: string }>> {
  const { client } = await requireClientAccess(slug);
  if (!MODULES.includes(module)) return fail(t.errors.VALIDATION, "VALIDATION");
  const range = parseRange(rangeFrom, rangeTo);
  if (!range) return fail(rs.builder.invalidRange, "VALIDATION");
  const lang: InsightLanguage = language === "en" ? "en" : "id";
  const previous = previousRange(range);

  try {
    const data = await buildInsightPayload(client.id, module, range, previous);
    const { content, model } = await providerGenerate({
      module,
      language: lang,
      clientName: client.name,
      range: toRangeInfo(range, previous),
      data,
    });
    await db.aiInsight.create({
      data: { clientId: client.id, module, dateFrom: range.from, dateTo: range.to, language: lang, model, content },
    });
    revalidatePath(`/clients/${slug}`);
    revalidatePath(`/clients/${slug}/reports`);
    return ok({ content });
  } catch (e) {
    if (isProviderError(e)) return fail(t.errors[e.code] ?? t.errors.UNKNOWN, e.code);
    console.error("generateInsight failed", e);
    return fail(t.errors.UNKNOWN);
  }
}

const createReportSchema = z.object({
  title: z.string().trim().min(1, rs.builder.titleRequired).max(160),
  modules: z.array(z.enum(["SOCIAL", "SEO", "ADS"])).min(1, rs.builder.noModules),
  compare: z.boolean(),
  includeAi: z.boolean(),
  language: z.enum(["id", "en"]),
  from: z.string(),
  to: z.string(),
});

export type CreateReportState = ActionResult<{ id: string }> | null;

/** Create a report from the builder form; on success redirects to the history tab. */
export async function createReport(slug: string, _prev: CreateReportState, formData: FormData): Promise<CreateReportState> {
  const { client } = await requireClientAccess(slug);

  const parsed = createReportSchema.safeParse({
    title: formData.get("title"),
    modules: formData.getAll("modules").map(String),
    compare: formData.get("compare") === "on" || formData.get("compare") === "true",
    includeAi: formData.get("includeAi") === "on" || formData.get("includeAi") === "true",
    language: formData.get("language") ?? "id",
    from: formData.get("from") ?? "",
    to: formData.get("to") ?? "",
  });
  if (!parsed.success) {
    const flat = z.flattenError(parsed.error);
    const firstError = Object.values(flat.fieldErrors).flat()[0] ?? t.errors.VALIDATION;
    return fail(firstError, "VALIDATION", flat.fieldErrors as Record<string, string[]>);
  }
  const input = parsed.data;
  const range = parseRange(input.from, input.to);
  if (!range) return fail(rs.builder.invalidRange, "VALIDATION");
  const previous = previousRange(range);

  let aiSummary: string | null = null;
  if (input.includeAi) {
    try {
      const data = await buildInsightPayload(client.id, "OVERVIEW", range, previous, input.modules);
      const { content, model } = await providerGenerate({
        module: "OVERVIEW",
        language: input.language,
        clientName: client.name,
        range: toRangeInfo(range, previous),
        data,
      });
      aiSummary = content;
      // Cache so the Overview page and insight tab pick it up too.
      await db.aiInsight.create({
        data: { clientId: client.id, module: "OVERVIEW", dateFrom: range.from, dateTo: range.to, language: input.language, model, content },
      });
    } catch (e) {
      if (isProviderError(e)) return fail(t.errors[e.code] ?? t.errors.UNKNOWN, e.code);
      console.error("createReport insight failed", e);
      return fail(t.errors.UNKNOWN);
    }
  }

  let reportId: string;
  try {
    // createdById is a nullable FK — verify the session user still exists (sessions
    // can outlive a re-seeded database) so the insert never trips the constraint.
    const session = await getSessionUser();
    const creator = session ? await db.user.findUnique({ where: { id: session.id }, select: { id: true } }) : null;
    const report = await db.report.create({
      data: {
        clientId: client.id,
        title: input.title,
        dateFrom: range.from,
        dateTo: range.to,
        compare: input.compare,
        modules: input.modules,
        language: input.language,
        aiSummary,
        createdById: creator?.id ?? null,
      },
    });
    reportId = report.id;
  } catch (e) {
    console.error("createReport failed", e);
    return fail(t.errors.UNKNOWN);
  }

  revalidatePath(`/clients/${slug}/reports`);
  redirect(`/clients/${slug}/reports?tab=history&created=${reportId}&from=${input.from}&to=${input.to}`);
}

/** Delete a report (MANAGER/ADMIN only). */
export async function deleteReport(slug: string, reportId: string): Promise<ActionResult<{ id: string }>> {
  const { client, canManage } = await requireClientAccess(slug);
  if (!canManage) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const report = await db.report.findUnique({ where: { id: reportId }, select: { id: true, clientId: true } });
  if (!report || report.clientId !== client.id) return fail(t.errors.NOT_FOUND);
  await db.report.delete({ where: { id: reportId } });
  revalidatePath(`/clients/${slug}/reports`);
  return ok({ id: reportId });
}
