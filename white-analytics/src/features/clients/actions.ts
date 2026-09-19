"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { hash } from "bcryptjs";
import { z } from "zod";
import { fail, ok, type ActionResult } from "@/lib/action-result";
import { db } from "@/lib/db";
import { getSessionUser } from "@/lib/rbac";
import { t } from "@/i18n/id";
import { tc } from "@/features/clients/strings";
import {
  CLIENT_CURRENCIES,
  CLIENT_TIMEZONES,
  MEMBER_ROLES,
  SHARE_MODULE_KEYS,
} from "@/features/clients/constants";
import {
  emptyToNull,
  isUniqueViolation,
  manageContext,
  revalidateClientPaths,
  zodErrors,
} from "@/features/clients/action-helpers";

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
