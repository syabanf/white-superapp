/**
 * Shared by the client server-action files (`actions.ts`, `actions-resources.ts`).
 * Lives outside a "use server" module because those may only export async functions.
 */
import "server-only";
import { revalidatePath } from "next/cache";
import type { z } from "zod";
import { Prisma } from "@/generated/prisma/client";
import { db } from "@/lib/db";
import { getSessionUser } from "@/lib/rbac";

export function zodErrors(err: z.ZodError): Record<string, string[]> {
  const out: Record<string, string[]> = {};
  for (const issue of err.issues) {
    const key = String(issue.path[0] ?? "form");
    (out[key] ??= []).push(issue.message);
  }
  return out;
}

export function isUniqueViolation(e: unknown): boolean {
  return e instanceof Prisma.PrismaClientKnownRequestError && e.code === "P2002";
}

export function emptyToNull(v: string | undefined): string | null {
  const s = (v ?? "").trim();
  return s.length === 0 ? null : s;
}

type ManageCtx = { userId: string; isAdmin: boolean; clientId: string; slug: string };

/** ADMIN, or MANAGER member of the client. Returns null when not allowed. */
export async function manageContext(clientId: string): Promise<ManageCtx | null> {
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

export function revalidateClientPaths(slug: string) {
  revalidatePath("/");
  revalidatePath("/clients");
  revalidatePath(`/clients/${slug}`);
  revalidatePath(`/clients/${slug}/settings`);
}
