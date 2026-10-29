/**
 * Auth-free cores of report creation and insight generation. The server actions
 * in ./actions.ts run RBAC and form parsing, then call these; the report
 * scheduler and the weekly auto insight (features/ads-automation) call them
 * directly from the cron. Provider errors are thrown (ProviderError) so each
 * caller decides how to surface them.
 */
import "server-only";
import { db } from "@/lib/db";
import { previousRange, type DateRange } from "@/lib/dates";
import { generateInsight as providerGenerate, type InsightLanguage } from "@/lib/providers/openrouter";
import { buildInsightPayload, toRangeInfo } from "@/features/reports/insights";
import type { InsightModule } from "@/features/reports/queries";

type ReportModule = "SOCIAL" | "SEO" | "ADS";

/** Generate an insight for a module + range and cache it in AiInsight. */
export async function generateInsightCore(input: {
  clientId: string;
  clientName: string;
  module: InsightModule;
  range: DateRange;
  language: InsightLanguage;
  modules?: ReportModule[];
}): Promise<{ id: string; content: string; model: string }> {
  const previous = previousRange(input.range);
  const data = await buildInsightPayload(input.clientId, input.module, input.range, previous, input.modules);
  const { content, model } = await providerGenerate({
    module: input.module,
    language: input.language,
    clientName: input.clientName,
    range: toRangeInfo(input.range, previous),
    data,
  });
  const row = await db.aiInsight.create({
    data: {
      clientId: input.clientId,
      module: input.module,
      dateFrom: input.range.from,
      dateTo: input.range.to,
      language: input.language,
      model,
      content,
    },
  });
  return { id: row.id, content, model };
}

/**
 * Create a Report row (optionally with an OVERVIEW AI summary, cached in
 * AiInsight so the Overview page and insight tab pick it up too).
 * `createdById` is verified to exist (sessions can outlive a re-seeded DB).
 */
export async function createReportCore(input: {
  clientId: string;
  clientName: string;
  title: string;
  range: DateRange;
  modules: ReportModule[];
  compare: boolean;
  includeAi: boolean;
  language: InsightLanguage;
  createdById?: string | null;
}): Promise<{ id: string; aiSummary: string | null }> {
  let aiSummary: string | null = null;
  if (input.includeAi) {
    const res = await generateInsightCore({
      clientId: input.clientId,
      clientName: input.clientName,
      module: "OVERVIEW",
      range: input.range,
      language: input.language,
      modules: input.modules,
    });
    aiSummary = res.content;
  }
  const creator = input.createdById
    ? await db.user.findUnique({ where: { id: input.createdById }, select: { id: true } })
    : null;
  const report = await db.report.create({
    data: {
      clientId: input.clientId,
      title: input.title,
      dateFrom: input.range.from,
      dateTo: input.range.to,
      compare: input.compare,
      modules: input.modules,
      language: input.language,
      aiSummary,
      createdById: creator?.id ?? null,
    },
  });
  return { id: report.id, aiSummary };
}
