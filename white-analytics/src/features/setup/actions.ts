"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { z } from "zod";
import { getSessionUser } from "@/lib/rbac";
import { fail, ok, type ActionResult } from "@/lib/action-result";
import { clearIntegration, getIntegrationStatuses, markSetup, saveIntegration } from "@/lib/integrations";
import { verifyApify, verifyDataForSeo, verifyGoogleClient, verifyMetaApp, verifyOpenRouter, type VerifyResult } from "@/lib/providers/verify";
import { INTEGRATIONS, INTEGRATION_KEYS, META_REVIEW_STATUSES, type IntegrationKey } from "@/features/setup/integrations";
import { t } from "@/i18n/id";
import { ts } from "./strings";

const keySchema = z.enum(INTEGRATION_KEYS);
const credentialSchema = z.object({
  key: keySchema,
  publicId: z.string().trim().max(300).optional(),
  secret: z.string().trim().max(2000).optional(),
  verify: z.boolean().default(false),
});

async function requireAdminAction() {
  const user = await getSessionUser();
  if (!user) return null;
  if (user.role !== "ADMIN") return null;
  return user;
}

function revalidateSetup() {
  revalidatePath("/setup");
  revalidatePath("/", "layout");
}

async function runVerify(key: IntegrationKey, publicId: string | null, secret: string | null): Promise<VerifyResult> {
  if (!secret) return { ok: false, message: t.errors.VALIDATION };
  switch (key) {
    case "META":
      return publicId ? verifyMetaApp(publicId, secret) : { ok: false, message: ts.metaAppId };
    case "GOOGLE":
      return publicId ? verifyGoogleClient(publicId, secret) : { ok: false, message: ts.googleClientId };
    case "DATAFORSEO":
      return publicId ? verifyDataForSeo(publicId, secret) : { ok: false, message: ts.dfsLogin };
    case "APIFY":
      return verifyApify(secret);
    case "OPENROUTER":
      return verifyOpenRouter(secret);
  }
}

export type SaveCredentialResult = { verified: VerifyResult | null };

export async function saveCredentialAction(input: z.input<typeof credentialSchema>): Promise<ActionResult<SaveCredentialResult>> {
  const user = await requireAdminAction();
  if (!user) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const parsed = credentialSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const { key, publicId, secret, verify } = parsed.data;
  const def = INTEGRATIONS[key];

  await saveIntegration(key, {
    publicId: publicId === undefined ? undefined : publicId,
    secret: secret ? secret : undefined, // blank keeps the stored secret
    updatedById: user.id,
  });

  let verified: VerifyResult | null = null;
  if (verify) {
    // Effective values after save: env now carries DB values (or the deployed env fallback).
    const effPublic = process.env[def.publicEnv] ?? null;
    const effSecret = process.env[def.secretEnv] ?? null;
    verified = await runVerify(key, effPublic, effSecret);
    await saveIntegration(key, { verifiedAt: verified.ok ? new Date() : null });
  }
  revalidateSetup();
  return ok({ verified });
}

export async function clearCredentialAction(key: IntegrationKey): Promise<ActionResult> {
  const user = await requireAdminAction();
  if (!user) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const parsed = keySchema.safeParse(key);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  await clearIntegration(parsed.data);
  revalidateSetup();
  return ok(undefined);
}

const reviewSchema = z.object({
  permissions: z.record(z.string().max(80), z.enum(META_REVIEW_STATUSES)),
  appLive: z.boolean(),
  businessVerified: z.boolean(),
});

export async function saveMetaReviewAction(input: z.input<typeof reviewSchema>): Promise<ActionResult> {
  const user = await requireAdminAction();
  if (!user) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const parsed = reviewSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const statuses = await getIntegrationStatuses();
  const prev = (statuses.META.meta && typeof statuses.META.meta === "object" ? statuses.META.meta : {}) as Record<string, unknown>;
  await saveIntegration("META", { meta: { ...prev, ...parsed.data }, updatedById: user.id });
  revalidateSetup();
  return ok(undefined);
}

export async function finishSetupAction(): Promise<void> {
  const user = await requireAdminAction();
  if (!user) redirect("/login");
  await markSetup("completedAt");
  revalidateSetup();
  redirect("/clients/new");
}

export async function skipSetupAction(): Promise<void> {
  const user = await requireAdminAction();
  if (!user) redirect("/login");
  await markSetup("skippedAt");
  revalidateSetup();
  redirect("/");
}
