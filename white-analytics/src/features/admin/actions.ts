"use server";

import { revalidatePath } from "next/cache";
import { hash } from "bcryptjs";
import { z } from "zod";
import { Prisma } from "@/generated/prisma/client";
import { fail, ok, type ActionResult } from "@/lib/action-result";
import { db } from "@/lib/db";
import { getSessionUser } from "@/lib/rbac";
import { t } from "@/i18n/id";
import { ta } from "@/features/admin/strings";
import { MEMBER_ROLES } from "@/features/clients/constants";
import { gatewayFetch, GatewayError, syncUserToGateway } from "@/lib/erp/gateway";

/** Pushes the saved row to identity so the user can open the ERP. */
async function mirrorUser(userId: string) {
  const u = await db.user.findUnique({
    where: { id: userId },
    select: { id: true, email: true, name: true, passwordHash: true, role: true, isActive: true },
  });
  if (u) await syncUserToGateway(u);
}

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

async function requireAdminActor() {
  const user = await getSessionUser();
  if (!user || user.role !== "ADMIN") return null;
  return user;
}

const membershipSchema = z.object({
  clientId: z.string().min(1),
  role: z.enum(MEMBER_ROLES),
});

const baseUserSchema = z.object({
  name: z.string().trim().min(1, ta.nameRequired).max(80),
  email: z.string().trim().toLowerCase().email(ta.emailInvalid),
  role: z.enum(["ADMIN", "MEMBER"]),
  isActive: z.boolean(),
  memberships: z.array(membershipSchema).default([]),
});

const createUserSchema = baseUserSchema.extend({
  password: z.string().min(8, ta.passwordMin).max(100),
});

const updateUserSchema = baseUserSchema.extend({
  password: z.union([z.literal(""), z.string().min(8, ta.passwordMin).max(100)]).default(""),
});

export type UserFormInput = z.input<typeof updateUserSchema>;

export async function createUserAction(input: UserFormInput): Promise<ActionResult<{ id: string }>> {
  const actor = await requireAdminActor();
  if (!actor) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const parsed = createUserSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION", zodErrors(parsed.error));
  const d = parsed.data;
  const exists = await db.user.findUnique({ where: { email: d.email }, select: { id: true } });
  if (exists) return fail(t.admin.emailTaken, "VALIDATION", { email: [t.admin.emailTaken] });
  try {
    const passwordHash = await hash(d.password, 10);
    const user = await db.user.create({
      data: {
        name: d.name,
        email: d.email,
        passwordHash,
        role: d.role,
        isActive: d.isActive,
        memberships: { create: d.memberships.map((m) => ({ clientId: m.clientId, role: m.role })) },
      },
    });
    await mirrorUser(user.id);
    revalidatePath("/admin/users");
    return ok({ id: user.id });
  } catch (e) {
    if (isUniqueViolation(e)) return fail(t.admin.emailTaken, "VALIDATION", { email: [t.admin.emailTaken] });
    return fail(t.errors.UNKNOWN);
  }
}

export async function updateUserAction(userId: string, input: UserFormInput): Promise<ActionResult<void>> {
  const actor = await requireAdminActor();
  if (!actor) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const parsed = updateUserSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION", zodErrors(parsed.error));
  const d = parsed.data;
  const target = await db.user.findUnique({ where: { id: userId }, select: { id: true } });
  if (!target) return fail(t.errors.NOT_FOUND, "NOT_FOUND");
  // Safety: an admin cannot deactivate or demote their own account.
  if (userId === actor.id && (!d.isActive || d.role !== "ADMIN")) {
    return fail(ta.cannotEditSelf, "VALIDATION");
  }
  const emailOwner = await db.user.findUnique({ where: { email: d.email }, select: { id: true } });
  if (emailOwner && emailOwner.id !== userId)
    return fail(t.admin.emailTaken, "VALIDATION", { email: [t.admin.emailTaken] });
  try {
    await db.$transaction(async (tx) => {
      await tx.user.update({
        where: { id: userId },
        data: {
          name: d.name,
          email: d.email,
          role: d.role,
          isActive: d.isActive,
          ...(d.password === "" ? {} : { passwordHash: await hash(d.password, 10) }),
        },
      });
      await tx.clientMember.deleteMany({ where: { userId } });
      if (d.memberships.length > 0) {
        await tx.clientMember.createMany({
          data: d.memberships.map((m) => ({ userId, clientId: m.clientId, role: m.role })),
          skipDuplicates: true,
        });
      }
    });
    await mirrorUser(userId);
    revalidatePath("/admin/users");
    return ok(undefined);
  } catch (e) {
    if (isUniqueViolation(e)) return fail(t.admin.emailTaken, "VALIDATION", { email: [t.admin.emailTaken] });
    return fail(t.errors.UNKNOWN);
  }
}

/** Quick toggle from the users table. */
export async function setUserActiveAction(userId: string, isActive: boolean): Promise<ActionResult<void>> {
  const actor = await requireAdminActor();
  if (!actor) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  if (userId === actor.id && !isActive) return fail(ta.cannotEditSelf, "VALIDATION");
  try {
    await db.user.update({ where: { id: userId }, data: { isActive } });
  } catch {
    return fail(t.errors.NOT_FOUND, "NOT_FOUND");
  }
  await mirrorUser(userId);
  revalidatePath("/admin/users");
  return ok(undefined);
}

export async function deleteUserAction(userId: string): Promise<ActionResult<void>> {
  const actor = await requireAdminActor();
  if (!actor) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  if (userId === actor.id) return fail(t.admin.cannotDeleteSelf, "VALIDATION");
  const target = await db.user.findUnique({ where: { id: userId }, select: { id: true } });
  if (!target) return fail(t.errors.NOT_FOUND, "NOT_FOUND");
  await db.user.delete({ where: { id: userId } }); // memberships cascade; clients/reports keep with SetNull
  await syncUserToGateway({ id: userId, deleted: true });
  revalidatePath("/admin/users");
  return ok(undefined);
}

const permissionsSchema = z.array(z.string().regex(/^[a-z]+\.(\*|[a-z]+)$/)).max(40);

/** Replaces a user's ERP permissions in identity (ADMIN only). */
export async function setErpPermissionsAction(
  userId: string,
  permissions: string[],
): Promise<ActionResult<void>> {
  const actor = await requireAdminActor();
  if (!actor) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const parsed = permissionsSchema.safeParse(permissions);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  try {
    // The mirror may not exist yet for users created before the bridge.
    await mirrorUser(userId);
    await gatewayFetch(`/v1/users/${encodeURIComponent(userId)}/permissions`, {
      method: "PUT",
      body: JSON.stringify({ permissions: parsed.data }),
    });
  } catch (e) {
    if (e instanceof GatewayError) return fail(e.message, e.code === "VALIDATION" ? "VALIDATION" : undefined);
    return fail(t.errors.UNKNOWN);
  }
  revalidatePath("/admin/users");
  return ok(undefined);
}
