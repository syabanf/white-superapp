"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { hash } from "bcryptjs";
import { z } from "zod";
import { Prisma } from "@/generated/prisma/client";
import { fail, ok, type ActionResult } from "@/lib/action-result";
import { encrypt } from "@/lib/crypto";
import { db } from "@/lib/db";
import { getSessionUser } from "@/lib/rbac";
import { GOOGLE_SCOPES } from "@/lib/providers/google-oauth";
import { META_SCOPES } from "@/lib/providers/meta-oauth";
import { t } from "@/i18n/id";
import { tc } from "@/features/clients/strings";
import {
  CLIENT_CURRENCIES,
  CLIENT_TIMEZONES,
  MEMBER_ROLES,
  SHARE_MODULE_KEYS,
  SOCIAL_PLATFORMS,
} from "@/features/clients/constants";

// ── Helpers ──────────────────────────────────────────────────

function zodErrors(err: z.ZodError): Record<string, string[]> {
  const out: Record<string, string[]> = {};
  for (const issue of err.issues) {
    const key = String(issue.path[0] ?? "form");
    (out[key] ??= []).push(issue.message);
  }
  return out;
}

function isUniqueViolation(e: unknown): boolean {
  return e instanceof Prisma.PrismaClientKnownRequestError && e.code === "P2002";
}

function emptyToNull(v: string | undefined): string | null {
  const s = (v ?? "").trim();
  return s.length === 0 ? null : s;
}

type ManageCtx = { userId: string; isAdmin: boolean; clientId: string; slug: string };

/** ADMIN, or MANAGER member of the client. Returns null when not allowed. */
async function manageContext(clientId: string): Promise<ManageCtx | null> {
  const user = await getSessionUser();
  if (!user) return null;
  const client = await db.client.findUnique({ where: { id: clientId }, select: { slug: true } });
  if (!client) return null;
  if (user.role === "ADMIN") return { userId: user.id, isAdmin: true, clientId, slug: client.slug };
  const membership = await db.clientMember.findUnique({
    where: { userId_clientId: { userId: user.id, clientId } },
    select: { role: true },
  });
  if (membership?.role !== "MANAGER") return null;
  return { userId: user.id, isAdmin: false, clientId, slug: client.slug };
}

function revalidateClientPaths(slug: string) {
  revalidatePath("/");
  revalidatePath("/clients");
  revalidatePath(`/clients/${slug}`);
  revalidatePath(`/clients/${slug}/settings`);
}

// ── Client CRUD ──────────────────────────────────────────────

const clientFormSchema = z.object({
  name: z.string().trim().min(1, tc.form.nameRequired).max(80),
  slug: z
    .string()
    .trim()
    .toLowerCase()
    .min(2, tc.form.slugRequired)
    .max(60)
    .regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/, tc.form.slugInvalid),
  description: z.string().trim().max(500).default(""),
  industry: z.string().trim().max(60).default(""),
  websiteUrl: z.union([z.literal(""), z.string().trim().url(tc.form.websiteInvalid)]).default(""),
  currency: z.enum(CLIENT_CURRENCIES),
  timezone: z.enum(CLIENT_TIMEZONES),
});

export type ClientFormInput = z.input<typeof clientFormSchema>;

export async function createClientAction(input: ClientFormInput): Promise<ActionResult<{ slug: string }>> {
  const user = await getSessionUser();
  if (!user || user.role !== "ADMIN") return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const parsed = clientFormSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION", zodErrors(parsed.error));
  const d = parsed.data;
  const exists = await db.client.findUnique({ where: { slug: d.slug }, select: { id: true } });
  if (exists) return fail(t.clients.slugTaken, "VALIDATION", { slug: [t.clients.slugTaken] });
  try {
    await db.client.create({
      data: {
        name: d.name,
        slug: d.slug,
        description: d.description,
        industry: emptyToNull(d.industry),
        websiteUrl: emptyToNull(d.websiteUrl),
        currency: d.currency,
        timezone: d.timezone,
        createdById: user.id,
      },
    });
  } catch (e) {
    if (isUniqueViolation(e)) return fail(t.clients.slugTaken, "VALIDATION", { slug: [t.clients.slugTaken] });
    return fail(t.errors.UNKNOWN);
  }
  revalidatePath("/");
  revalidatePath("/clients");
  redirect(`/clients/${d.slug}`);
}

export async function updateClientAction(clientId: string, input: ClientFormInput): Promise<ActionResult<{ slug: string }>> {
  const ctx = await manageContext(clientId);
  if (!ctx) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const parsed = clientFormSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION", zodErrors(parsed.error));
  const d = parsed.data;
  if (d.slug !== ctx.slug) {
    const exists = await db.client.findUnique({ where: { slug: d.slug }, select: { id: true } });
    if (exists) return fail(t.clients.slugTaken, "VALIDATION", { slug: [t.clients.slugTaken] });
  }
  try {
    await db.client.update({
      where: { id: clientId },
      data: {
        name: d.name,
        slug: d.slug,
        description: d.description,
        industry: emptyToNull(d.industry),
        websiteUrl: emptyToNull(d.websiteUrl),
        currency: d.currency,
        timezone: d.timezone,
      },
    });
  } catch (e) {
    if (isUniqueViolation(e)) return fail(t.clients.slugTaken, "VALIDATION", { slug: [t.clients.slugTaken] });
    return fail(t.errors.UNKNOWN);
  }
  revalidateClientPaths(ctx.slug);
  if (d.slug !== ctx.slug) revalidateClientPaths(d.slug);
  return ok({ slug: d.slug });
}

export async function deleteClientAction(clientId: string): Promise<ActionResult<void>> {
  const user = await getSessionUser();
  if (!user || user.role !== "ADMIN") return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const client = await db.client.findUnique({ where: { id: clientId }, select: { slug: true } });
  if (!client) return fail(t.errors.NOT_FOUND, "NOT_FOUND");
  await db.client.delete({ where: { id: clientId } }); // relations cascade via Prisma schema
  revalidatePath("/");
  revalidatePath("/clients");
  return ok(undefined);
}

// ── Members ──────────────────────────────────────────────────

const memberSchema = z.object({
  userId: z.string().min(1),
  role: z.enum(MEMBER_ROLES),
});

export async function addMemberAction(clientId: string, input: { userId: string; role: string }): Promise<ActionResult<void>> {
  const ctx = await manageContext(clientId);
  if (!ctx) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const parsed = memberSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION", zodErrors(parsed.error));
  const user = await db.user.findUnique({ where: { id: parsed.data.userId }, select: { id: true } });
  if (!user) return fail(t.errors.NOT_FOUND, "NOT_FOUND");
  await db.clientMember.upsert({
    where: { userId_clientId: { userId: parsed.data.userId, clientId } },
    create: { userId: parsed.data.userId, clientId, role: parsed.data.role },
    update: { role: parsed.data.role },
  });
  revalidateClientPaths(ctx.slug);
  return ok(undefined);
}

export async function updateMemberRoleAction(clientId: string, input: { userId: string; role: string }): Promise<ActionResult<void>> {
  const ctx = await manageContext(clientId);
  if (!ctx) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const parsed = memberSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION", zodErrors(parsed.error));
  try {
    await db.clientMember.update({
      where: { userId_clientId: { userId: parsed.data.userId, clientId } },
      data: { role: parsed.data.role },
    });
  } catch {
    return fail(t.errors.NOT_FOUND, "NOT_FOUND");
  }
  revalidateClientPaths(ctx.slug);
  return ok(undefined);
}

export async function removeMemberAction(clientId: string, userId: string): Promise<ActionResult<void>> {
  const ctx = await manageContext(clientId);
  if (!ctx) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  try {
    await db.clientMember.delete({ where: { userId_clientId: { userId, clientId } } });
  } catch {
    return fail(t.errors.NOT_FOUND, "NOT_FOUND");
  }
  revalidateClientPaths(ctx.slug);
  return ok(undefined);
}

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

// ── Share settings ───────────────────────────────────────────

const shareSchema = z.object({
  shareEnabled: z.boolean(),
  pin: z.union([z.literal(""), z.string().regex(/^\d{6}$/, tc.settings.pinInvalid)]),
  modules: z.array(z.enum(SHARE_MODULE_KEYS)).min(1, tc.settings.modulesMin),
});

export type ShareSettingsInput = z.input<typeof shareSchema>;

export async function updateShareSettingsAction(clientId: string, input: ShareSettingsInput): Promise<ActionResult<void>> {
  const ctx = await manageContext(clientId);
  if (!ctx) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const parsed = shareSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION", zodErrors(parsed.error));
  const d = parsed.data;
  const sharePinHash = d.pin === "" ? null : await hash(d.pin, 10);
  await db.client.update({
    where: { id: clientId },
    data: { shareEnabled: d.shareEnabled, sharePinHash, shareModules: [...d.modules] },
  });
  revalidateClientPaths(ctx.slug);
  revalidatePath(`/share/${ctx.slug}`);
  return ok(undefined);
}
