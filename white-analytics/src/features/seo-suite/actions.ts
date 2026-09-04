"use server";

import { revalidatePath } from "next/cache";
import { z } from "zod";
import { db } from "@/lib/db";
import { todayUtc } from "@/lib/dates";
import { fail, ok, type ActionResult } from "@/lib/action-result";
import { seoData } from "@/lib/providers/dataforseo";
import { t } from "@/i18n/id";
import { errorCode, errorMessage, finishJob, guardProperty, startJob } from "./access";
import { parseKeywordLines, parseTags } from "./lib";
import { s } from "./strings";
import { trackRanks } from "./tracker";

const ideaSchema = z.object({
  keyword: z.string().min(1).max(120),
  volume: z.number().int().nonnegative().nullable().optional(),
  difficulty: z.number().min(0).max(100).nullable().optional(),
  cpc: z.number().nonnegative().nullable().optional(),
  intent: z.enum(["INFORMATIONAL", "NAVIGATIONAL", "COMMERCIAL", "TRANSACTIONAL"]).nullable().optional(),
});

const trackSchema = z.object({
  clientId: z.string().min(1),
  propertyId: z.string().min(1),
  ideas: z.array(ideaSchema).min(1).max(100),
  device: z.enum(["DESKTOP", "MOBILE"]).default("MOBILE"),
  tags: z.array(z.string().min(1).max(30)).max(10).default([]),
});

function revalidateSuite(slug: string) {
  revalidatePath(`/clients/${slug}/seo/rank`);
  revalidatePath(`/clients/${slug}/seo/research`);
}

type IdeaInput = z.infer<typeof ideaSchema>;

/** Insert tracked keywords (skip duplicates), then fetch their first positions right away. */
async function insertTracked(property: { id: string; siteUrl: string; locationCode: number; languageCode: string }, ideas: IdeaInput[], device: "DESKTOP" | "MOBILE", tags: string[]): Promise<number> {
  const before = await db.trackedKeyword.findMany({ where: { propertyId: property.id }, select: { id: true } });
  const beforeIds = new Set(before.map((b) => b.id));
  const res = await db.trackedKeyword.createMany({
    data: ideas.map((i) => ({
      propertyId: property.id,
      keyword: i.keyword.trim().toLowerCase(),
      locationCode: property.locationCode,
      languageCode: property.languageCode,
      device,
      tags,
      volume: i.volume ?? null,
      difficulty: i.difficulty == null ? null : Math.round(i.difficulty),
      cpc: i.cpc ?? null,
      intent: i.intent ?? null,
    })),
    skipDuplicates: true,
  });
  if (res.count > 0) {
    const after = await db.trackedKeyword.findMany({ where: { propertyId: property.id }, select: { id: true } });
    const newIds = after.map((a) => a.id).filter((id) => !beforeIds.has(id));
    try {
      await trackRanks(property, { date: todayUtc(), keywordIds: newIds });
    } catch {
      // first positions are a nicety; the daily job will fill them in
    }
  }
  return res.count;
}

/** From the research table: "Lacak" on selected ideas (metrics already known). */
export async function trackKeywords(input: z.infer<typeof trackSchema>): Promise<ActionResult<{ added: number }>> {
  const parsed = trackSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const g = await guardProperty(parsed.data.clientId, parsed.data.propertyId, { manage: true });
  if (!g.ok) return g;
  try {
    const added = await insertTracked(g.data.property, parsed.data.ideas, parsed.data.device, parsed.data.tags);
    revalidateSuite(g.data.property.slug);
    return ok({ added });
  } catch (e) {
    return fail(errorMessage(e), errorCode(e));
  }
}

const addSchema = z.object({
  clientId: z.string().min(1),
  propertyId: z.string().min(1),
  text: z.string().min(1).max(20_000),
  device: z.enum(["DESKTOP", "MOBILE"]).default("MOBILE"),
  tags: z.string().max(500).default(""),
});

/** Bulk textarea → enrich with provider metrics → tracked keywords. */
export async function addTrackedKeywords(input: z.infer<typeof addSchema>): Promise<ActionResult<{ added: number }>> {
  const parsed = addSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const keywords = parseKeywordLines(parsed.data.text, 100);
  if (keywords.length === 0) return fail(s.keywordsInvalid, "VALIDATION");
  const g = await guardProperty(parsed.data.clientId, parsed.data.propertyId, { manage: true });
  if (!g.ok) return g;
  const p = g.data.property;
  try {
    let ideas: IdeaInput[] = keywords.map((keyword) => ({ keyword }));
    try {
      const metrics = await seoData.keywordMetrics(keywords, p.locationCode, p.languageCode);
      const by = new Map(metrics.map((m) => [m.keyword.toLowerCase(), m]));
      ideas = keywords.map((keyword) => {
        const m = by.get(keyword);
        return m ? { keyword, volume: m.volume, difficulty: m.difficulty, cpc: m.cpc, intent: m.intent } : { keyword };
      });
    } catch {
      // metrics are optional; keywords are still tracked
    }
    const added = await insertTracked(p, ideas, parsed.data.device, parseTags(parsed.data.tags));
    revalidateSuite(p.slug);
    return ok({ added });
  } catch (e) {
    return fail(errorMessage(e), errorCode(e));
  }
}

const keywordIdSchema = z.object({ clientId: z.string().min(1), keywordId: z.string().min(1) });

export async function deleteTrackedKeyword(input: z.infer<typeof keywordIdSchema>): Promise<ActionResult<void>> {
  const parsed = keywordIdSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const kw = await db.trackedKeyword.findUnique({ where: { id: parsed.data.keywordId }, select: { propertyId: true } });
  if (!kw) return fail(t.errors.NOT_FOUND, "NOT_FOUND");
  const g = await guardProperty(parsed.data.clientId, kw.propertyId, { manage: true });
  if (!g.ok) return g;
  await db.trackedKeyword.delete({ where: { id: parsed.data.keywordId } });
  revalidateSuite(g.data.property.slug);
  return ok(undefined);
}

const tagsSchema = keywordIdSchema.extend({ tags: z.string().max(500) });

export async function updateKeywordTags(input: z.infer<typeof tagsSchema>): Promise<ActionResult<{ tags: string[] }>> {
  const parsed = tagsSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const kw = await db.trackedKeyword.findUnique({ where: { id: parsed.data.keywordId }, select: { propertyId: true } });
  if (!kw) return fail(t.errors.NOT_FOUND, "NOT_FOUND");
  const g = await guardProperty(parsed.data.clientId, kw.propertyId, { manage: true });
  if (!g.ok) return g;
  const tags = parseTags(parsed.data.tags);
  await db.trackedKeyword.update({ where: { id: parsed.data.keywordId }, data: { tags } });
  revalidateSuite(g.data.property.slug);
  return ok({ tags });
}

const refreshSchema = z.object({ clientId: z.string().min(1), propertyId: z.string().min(1) });

/** "Perbarui sekarang": re-check every tracked keyword (10-minute cooldown per client). */
export async function refreshRanks(input: z.infer<typeof refreshSchema>): Promise<ActionResult<{ updated: number }>> {
  const parsed = refreshSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const g = await guardProperty(parsed.data.clientId, parsed.data.propertyId);
  if (!g.ok) return g;
  const job = await startJob(parsed.data.clientId, "SEO_SUITE_RANKS", g.data.property.siteUrl);
  if (!job.ok) return job;
  try {
    const { updated } = await trackRanks(g.data.property, { date: todayUtc() });
    await finishJob(job.data, { ok: true, message: `${updated} kata kunci` });
    revalidateSuite(g.data.property.slug);
    return ok({ updated });
  } catch (e) {
    await finishJob(job.data, { ok: false, error: errorMessage(e) });
    return fail(errorMessage(e), errorCode(e));
  }
}
