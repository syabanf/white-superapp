"use server";

import { z } from "zod";
import { fail, ok, type ActionResult } from "@/lib/action-result";
import { encrypt } from "@/lib/crypto";
import { db } from "@/lib/db";
import { GOOGLE_SCOPES } from "@/lib/providers/google-oauth";
import { META_SCOPES } from "@/lib/providers/meta-oauth";
import { t } from "@/i18n/id";
import { tc } from "@/features/clients/strings";
import { SOCIAL_PLATFORMS } from "@/features/clients/constants";
import {
  emptyToNull,
  isUniqueViolation,
  manageContext,
  revalidateClientPaths,
  zodErrors,
} from "@/features/clients/action-helpers";

// ── Connections ──────────────────────────────────────────────

export async function disconnectConnectionAction(clientId: string, connectionId: string): Promise<ActionResult<void>> {
  const ctx = await manageContext(clientId);
  if (!ctx) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const conn = await db.connection.findUnique({ where: { id: connectionId }, select: { clientId: true } });
  if (!conn || conn.clientId !== clientId) return fail(t.errors.NOT_FOUND, "NOT_FOUND");
  await db.connection.delete({ where: { id: connectionId } });
  revalidateClientPaths(ctx.slug);
  return ok(undefined);
}

const providerSchema = z.enum(["META", "GOOGLE"]);

/** Demo mode: create a mock Connection row when real credentials are absent. */
export async function simulateConnectionAction(clientId: string, provider: string): Promise<ActionResult<void>> {
  const ctx = await manageContext(clientId);
  if (!ctx) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const parsed = providerSchema.safeParse(provider);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const p = parsed.data;
  const accountId = p === "META" ? "demo-meta" : "demo-google";
  const payload = {
    displayName: "Akun demo",
    accessTokenEnc: encrypt("mock"),
    refreshTokenEnc: null,
    expiresAt: new Date(Date.now() + 60 * 24 * 60 * 60 * 1000),
    scopes: p === "META" ? META_SCOPES : GOOGLE_SCOPES,
  };
  await db.connection.upsert({
    where: { clientId_provider_accountId: { clientId, provider: p, accountId } },
    create: { clientId, provider: p, accountId, ...payload },
    update: payload,
  });
  revalidateClientPaths(ctx.slug);
  return ok(undefined);
}

// ── Linked resources ─────────────────────────────────────────

const socialAccountSchema = z.object({
  platform: z.enum(SOCIAL_PLATFORMS),
  username: z
    .string()
    .trim()
    .min(1, tc.form.nameRequired)
    .max(60)
    .transform((v) => v.replace(/^@+/, "").toLowerCase()),
  displayName: z.string().trim().max(80).default(""),
});

export async function addSocialAccountAction(
  clientId: string,
  input: { platform: string; username: string; displayName?: string },
): Promise<ActionResult<void>> {
  const ctx = await manageContext(clientId);
  if (!ctx) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const parsed = socialAccountSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION", zodErrors(parsed.error));
  const d = parsed.data;
  const connection = await db.connection.findFirst({ where: { clientId, provider: "META" }, select: { id: true } });
  try {
    await db.socialAccount.create({
      data: {
        clientId,
        platform: d.platform,
        externalId: `${d.platform.toLowerCase()}:${d.username}`,
        username: d.username,
        displayName: d.displayName || d.username,
        isCompetitor: false,
        connectionId: connection?.id ?? null,
      },
    });
  } catch (e) {
    if (isUniqueViolation(e)) return fail(tc.settings.duplicate, "VALIDATION", { username: [tc.settings.duplicate] });
    return fail(t.errors.UNKNOWN);
  }
  revalidateClientPaths(ctx.slug);
  return ok(undefined);
}

export async function removeSocialAccountAction(clientId: string, accountId: string): Promise<ActionResult<void>> {
  const ctx = await manageContext(clientId);
  if (!ctx) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const account = await db.socialAccount.findUnique({ where: { id: accountId }, select: { clientId: true, isCompetitor: true } });
  if (!account || account.clientId !== clientId || account.isCompetitor) return fail(t.errors.NOT_FOUND, "NOT_FOUND");
  await db.socialAccount.delete({ where: { id: accountId } });
  revalidateClientPaths(ctx.slug);
  return ok(undefined);
}

const seoPropertySchema = z.object({
  siteUrl: z
    .string()
    .trim()
    .min(4)
    .refine((v) => {
      if (/^sc-domain:[a-z0-9.-]+$/i.test(v)) return true;
      try {
        const u = new URL(v);
        return u.protocol === "http:" || u.protocol === "https:";
      } catch {
        return false;
      }
    }, tc.form.websiteInvalid),
  ga4PropertyId: z.string().trim().max(30).default(""),
});

export async function addSeoPropertyAction(
  clientId: string,
  input: { siteUrl: string; ga4PropertyId?: string },
): Promise<ActionResult<void>> {
  const ctx = await manageContext(clientId);
  if (!ctx) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const parsed = seoPropertySchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION", zodErrors(parsed.error));
  const connection = await db.connection.findFirst({ where: { clientId, provider: "GOOGLE" }, select: { id: true } });
  try {
    await db.seoProperty.create({
      data: {
        clientId,
        siteUrl: parsed.data.siteUrl,
        ga4PropertyId: emptyToNull(parsed.data.ga4PropertyId),
        connectionId: connection?.id ?? null,
      },
    });
  } catch (e) {
    if (isUniqueViolation(e)) return fail(tc.settings.duplicate, "VALIDATION", { siteUrl: [tc.settings.duplicate] });
    return fail(t.errors.UNKNOWN);
  }
  revalidateClientPaths(ctx.slug);
  return ok(undefined);
}

export async function removeSeoPropertyAction(clientId: string, propertyId: string): Promise<ActionResult<void>> {
  const ctx = await manageContext(clientId);
  if (!ctx) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const property = await db.seoProperty.findUnique({ where: { id: propertyId }, select: { clientId: true } });
  if (!property || property.clientId !== clientId) return fail(t.errors.NOT_FOUND, "NOT_FOUND");
  await db.seoProperty.delete({ where: { id: propertyId } });
  revalidateClientPaths(ctx.slug);
  return ok(undefined);
}

const adAccountSchema = z.object({
  externalId: z.string().trim().min(2).max(40),
  name: z.string().trim().min(1, tc.form.nameRequired).max(80),
});

export async function addAdAccountAction(clientId: string, input: { externalId: string; name: string }): Promise<ActionResult<void>> {
  const ctx = await manageContext(clientId);
  if (!ctx) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const parsed = adAccountSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION", zodErrors(parsed.error));
  const [client, connection] = await Promise.all([
    db.client.findUnique({ where: { id: clientId }, select: { currency: true } }),
    db.connection.findFirst({ where: { clientId, provider: "META" }, select: { id: true } }),
  ]);
  try {
    await db.adAccount.create({
      data: {
        clientId,
        externalId: parsed.data.externalId,
        name: parsed.data.name,
        currency: client?.currency ?? "IDR",
        connectionId: connection?.id ?? null,
      },
    });
  } catch (e) {
    if (isUniqueViolation(e)) return fail(tc.settings.duplicate, "VALIDATION", { externalId: [tc.settings.duplicate] });
    return fail(t.errors.UNKNOWN);
  }
  revalidateClientPaths(ctx.slug);
  return ok(undefined);
}

export async function removeAdAccountAction(clientId: string, adAccountId: string): Promise<ActionResult<void>> {
  const ctx = await manageContext(clientId);
  if (!ctx) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const account = await db.adAccount.findUnique({ where: { id: adAccountId }, select: { clientId: true } });
  if (!account || account.clientId !== clientId) return fail(t.errors.NOT_FOUND, "NOT_FOUND");
  await db.adAccount.delete({ where: { id: adAccountId } });
  revalidateClientPaths(ctx.slug);
  return ok(undefined);
}
