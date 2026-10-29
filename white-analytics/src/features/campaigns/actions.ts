"use server";

import { z } from "zod";
import { revalidatePath } from "next/cache";
import { db } from "@/lib/db";
import { fail, isProviderError, ok, type ActionResult } from "@/lib/action-result";
import { getSessionUser } from "@/lib/rbac";
import { withClientMode } from "@/lib/client-mode";
import { aiStudio, isAiStudioConfigured } from "@/lib/providers/ai-studio";
import { getBrandContext } from "@/features/brand/context";
import {
  AGE_MAX,
  AGE_MIN,
  CREATIVE_SLOTS,
  HEADLINE_MAX,
  MAX_INTERESTS,
  MAX_LOCATIONS,
  OBJECTIVES,
  PLACEMENTS,
  PRIMARY_TEXT_MAX,
  canEditDraft,
  canSubmitDraft,
  dateFromKey,
  draftIssues,
  isDateKey,
  mergeSuggestion,
  parseAudience,
  parseCreatives,
  parsePlacements,
  statusAfterSave,
  todayKey,
  type Audience,
  type Creative,
  type DraftInput,
  type DraftStatus,
  type Placement,
} from "./lib";
import { submitDraftToMeta } from "./submit";
import { c } from "./strings";

// ── access ───────────────────────────────────────────────────

type Access = { userId: string; client: { id: string; slug: string; currency: string; timezone: string; websiteUrl: string | null } };

/** RBAC in the data layer: only ADMIN or the client's MANAGER may write. */
async function requireManager(clientId: string): Promise<Access | null> {
  const user = await getSessionUser();
  if (!user || !clientId) return null;
  const client = await db.client.findUnique({ where: { id: clientId }, select: { id: true, slug: true, currency: true, timezone: true, websiteUrl: true } });
  if (!client) return null;
  if (user.role !== "ADMIN") {
    const m = await db.clientMember.findUnique({ where: { userId_clientId: { userId: user.id, clientId } }, select: { role: true } });
    if (m?.role !== "MANAGER") return null;
  }
  return { userId: user.id, client };
}

async function loadOwned(id: string) {
  if (!id || typeof id !== "string") return null;
  const row = await db.campaignDraft.findUnique({ where: { id } });
  if (!row) return null;
  const access = await requireManager(row.clientId);
  return access ? { row, access } : null;
}

function revalidate(slug: string, id?: string) {
  const base = `/clients/${slug}/ads/campaigns`;
  revalidatePath(base);
  if (id) revalidatePath(`${base}/${id}`);
}

async function currencyFor(clientId: string, adAccountId: string | null, fallback: string) {
  if (!adAccountId) return fallback;
  const a = await db.adAccount.findFirst({ where: { id: adAccountId, clientId }, select: { currency: true } });
  return a?.currency ?? fallback;
}

const issueText = (issues: { message: string }[]) => Array.from(new Set(issues.map((i) => i.message))).join(" ");

// ── schemas ──────────────────────────────────────────────────

const audienceSchema = z.object({
  locations: z.array(z.string().trim().min(1).max(80)).max(MAX_LOCATIONS),
  ageMin: z.number().int().min(AGE_MIN).max(AGE_MAX),
  ageMax: z.number().int().min(AGE_MIN).max(AGE_MAX),
  genders: z.enum(["ALL", "MALE", "FEMALE"]),
  interests: z.array(z.string().trim().min(1).max(80)).max(MAX_INTERESTS),
});
const creativeSchema = z.object({
  headline: z.string().max(HEADLINE_MAX * 2),
  primaryText: z.string().max(PRIMARY_TEXT_MAX * 2),
  cta: z.string().max(40),
  mediaUrl: z.string().max(2000).nullable(),
});
const draftSchema = z.object({
  clientId: z.string().min(1),
  id: z.string().min(1).optional(),
  name: z.string().trim().max(160),
  objective: z.enum(OBJECTIVES),
  dailyBudget: z.number().finite().min(0).max(1e12),
  startDate: z.string().refine(isDateKey),
  endDate: z.string().refine(isDateKey).nullable(),
  adAccountId: z.string().min(1).nullable(),
  audience: audienceSchema,
  placements: z.array(z.enum(PLACEMENTS)).max(PLACEMENTS.length),
  creatives: z.array(creativeSchema).length(CREATIVE_SLOTS),
  aiNotes: z.string().max(4000),
});
export type SaveDraftInput = z.infer<typeof draftSchema>;

// ── save (create / update) ───────────────────────────────────

export async function saveCampaignDraft(raw: SaveDraftInput): Promise<ActionResult<{ id: string; status: DraftStatus; created: boolean }>> {
  const parsed = draftSchema.safeParse(raw);
  if (!parsed.success) return fail(c.errors.invalid, "VALIDATION");
  const input = parsed.data;
  const access = await requireManager(input.clientId);
  if (!access) return fail(c.errors.forbidden, "UNAUTHORIZED");

  const accountCount = await db.adAccount.count({ where: { clientId: input.clientId } });
  if (input.adAccountId) {
    const owns = await db.adAccount.count({ where: { id: input.adAccountId, clientId: input.clientId } });
    if (!owns) return fail(c.errors.adAccount, "VALIDATION");
  }
  const existing = input.id ? await db.campaignDraft.findFirst({ where: { id: input.id, clientId: input.clientId } }) : null;
  if (input.id && !existing) return fail(c.errors.notFound, "NOT_FOUND");
  if (existing && !canEditDraft(existing.status)) return fail(c.errors.locked, "VALIDATION");

  const currency = await currencyFor(input.clientId, input.adAccountId, access.client.currency);
  const draftInput: DraftInput = { ...input, currency };
  const complete = draftIssues(draftInput, { today: todayKey(new Date(), access.client.timezone), requireAdAccount: accountCount > 0 }).length === 0;
  const status = statusAfterSave(existing?.status ?? null, complete);

  const data = {
    name: input.name,
    objective: input.objective,
    dailyBudget: input.dailyBudget,
    startDate: dateFromKey(input.startDate),
    endDate: input.endDate ? dateFromKey(input.endDate) : null,
    adAccountId: input.adAccountId,
    audience: input.audience,
    placements: input.placements,
    creatives: input.creatives,
    aiNotes: input.aiNotes,
    status,
  };
  const row = existing
    ? await db.campaignDraft.update({ where: { id: existing.id }, data })
    : await db.campaignDraft.create({ data: { ...data, clientId: input.clientId, createdById: access.userId } });
  revalidate(access.client.slug, row.id);
  return ok({ id: row.id, status: row.status, created: !existing });
}

// ── mark ready ───────────────────────────────────────────────

export async function markCampaignReady(id: string): Promise<ActionResult<{ status: DraftStatus }>> {
  const loaded = await loadOwned(id);
  if (!loaded) return fail(c.errors.notFound, "NOT_FOUND");
  const { row, access } = loaded;
  if (!canEditDraft(row.status)) return fail(c.errors.locked, "VALIDATION");
  const accountCount = await db.adAccount.count({ where: { clientId: row.clientId } });
  const issues = draftIssues(
    {
      name: row.name,
      objective: row.objective,
      dailyBudget: row.dailyBudget,
      currency: await currencyFor(row.clientId, row.adAccountId, access.client.currency),
      startDate: row.startDate.toISOString().slice(0, 10),
      endDate: row.endDate ? row.endDate.toISOString().slice(0, 10) : null,
      adAccountId: row.adAccountId,
      audience: parseAudience(row.audience),
      placements: parsePlacements(row.placements),
      creatives: parseCreatives(row.creatives),
    },
    { today: todayKey(new Date(), access.client.timezone), requireAdAccount: accountCount > 0 },
  );
  if (issues.length) return fail(issueText(issues), "VALIDATION");
  await db.campaignDraft.update({ where: { id: row.id }, data: { status: "READY" } });
  revalidate(access.client.slug, row.id);
  return ok({ status: "READY" });
}

// ── delete ───────────────────────────────────────────────────

export async function deleteCampaignDraft(id: string): Promise<ActionResult<{ slug: string }>> {
  const loaded = await loadOwned(id);
  if (!loaded) return fail(c.errors.notFound, "NOT_FOUND");
  await db.campaignDraft.delete({ where: { id: loaded.row.id } });
  revalidate(loaded.access.client.slug);
  return ok({ slug: loaded.access.client.slug });
}

// ── AI suggest ───────────────────────────────────────────────

const suggestSchema = z.object({
  clientId: z.string().min(1),
  objective: z.enum(OBJECTIVES),
  brief: z.string().max(1000),
  dailyBudget: z.number().finite().min(0).max(1e12),
  adAccountId: z.string().min(1).nullable(),
  audience: audienceSchema,
  placements: z.array(z.enum(PLACEMENTS)),
  creatives: z.array(creativeSchema).length(CREATIVE_SLOTS),
});

export type SuggestResult = { audience: Audience; placements: Placement[]; creatives: Creative[]; aiNotes: string; mock: boolean };

/**
 * Ask the AI studio for audience/placements/3 creatives/notes, then merge into
 * what the user already typed (nothing they wrote is overwritten). Not
 * persisted: the builder marks itself dirty and the user saves.
 */
export async function suggestCampaignWithAi(raw: z.infer<typeof suggestSchema>): Promise<ActionResult<SuggestResult>> {
  const parsed = suggestSchema.safeParse(raw);
  if (!parsed.success) return fail(c.errors.invalid, "VALIDATION");
  const input = parsed.data;
  const access = await requireManager(input.clientId);
  if (!access) return fail(c.errors.forbidden, "UNAUTHORIZED");
  try {
    return await withClientMode({ id: input.clientId }, async () => {
      const [brand, currency] = await Promise.all([
        getBrandContext(input.clientId),
        currencyFor(input.clientId, input.adAccountId, access.client.currency),
      ]);
      const suggestion = await aiStudio.suggestCampaign({ brand, objective: input.objective, offer: input.brief, dailyBudget: input.dailyBudget, currency });
      const merged = mergeSuggestion({ audience: input.audience, placements: input.placements, creatives: input.creatives }, suggestion, input.objective);
      return ok({ ...merged, mock: !isAiStudioConfigured() });
    });
  } catch (e) {
    if (isProviderError(e)) return fail(e.message, e.code);
    console.error("[campaigns] AI suggest failed", e);
    return fail(c.errors.ai);
  }
}

// ── submit ───────────────────────────────────────────────────

export async function submitCampaignDraft(id: string): Promise<ActionResult<{ externalId: string; simulated: boolean }>> {
  const loaded = await loadOwned(id);
  if (!loaded) return fail(c.errors.notFound, "NOT_FOUND");
  const { row, access } = loaded;
  if (!canSubmitDraft(row.status)) return fail(c.errors.notReady, "VALIDATION");
  try {
    const result = await withClientMode({ id: row.clientId }, () => submitDraftToMeta(row, access.client));
    // Conditional update: a double click cannot submit twice.
    const updated = await db.campaignDraft.updateMany({
      where: { id: row.id, status: "READY" },
      data: { status: "SUBMITTED", externalId: result.externalId, simulated: result.simulated, submittedAt: new Date() },
    });
    if (!updated.count) return fail(c.errors.notReady, "VALIDATION");
    revalidate(access.client.slug, row.id);
    return ok({ externalId: result.externalId, simulated: result.simulated });
  } catch (e) {
    if (isProviderError(e)) return fail(`${c.errors.submit} ${e.message}`, e.code);
    console.error("[campaigns] submit failed", e);
    return fail(c.errors.submit);
  }
}
