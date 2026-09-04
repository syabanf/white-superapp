import "server-only";
import { cache } from "react";
import { redirect } from "next/navigation";
import { auth } from "@/auth";
import { db } from "@/lib/db";
import type { Client, ClientMemberRole, UserRole } from "@/generated/prisma/client";

export type SessionUser = { id: string; email: string; name: string; role: UserRole };

/**
 * Optional session (no redirect), resolved against the database.
 *
 * Sessions are JWTs, so the token alone proves nothing about the user's current
 * state: the account may have been deleted (or the database restored/re-seeded)
 * and deactivating someone in Admin must take effect on their next request, not
 * whenever the token happens to expire. So we re-read the row and treat a
 * missing or inactive user as signed out. One query, cached per request, and it
 * also guarantees any `userId` we later write as a foreign key really exists.
 */
export const getSessionUser = cache(async (): Promise<SessionUser | null> => {
  const session = await auth();
  const id = session?.user?.id;
  if (!id) return null;
  const user = await db.user.findUnique({
    where: { id },
    select: { id: true, email: true, name: true, role: true, isActive: true },
  });
  if (!user || !user.isActive) return null;
  return { id: user.id, email: user.email, name: user.name, role: user.role };
});

/** Current session user (cached per request). Redirects to /login when absent. */
export const requireUser = cache(async (): Promise<SessionUser> => {
  const user = await getSessionUser();
  if (!user) redirect("/login");
  return user;
});

export async function requireAdmin(): Promise<SessionUser> {
  const user = await requireUser();
  if (user.role !== "ADMIN") redirect("/?unauthorized=1");
  return user;
}

export function isAdmin(user: SessionUser): boolean {
  return user.role === "ADMIN";
}

export type ClientAccess = {
  client: Client;
  role: ClientMemberRole | "ADMIN";
  canManage: boolean;
};

/**
 * Load a client by slug and verify the current user may access it.
 * ADMIN → all clients. MEMBER → only clients they are a member of.
 * Calls notFound-ish redirect when missing.
 */
export const requireClientAccess = cache(async (slug: string): Promise<ClientAccess> => {
  const user = await requireUser();
  const client = await db.client.findUnique({ where: { slug } });
  if (!client) redirect("/clients?missing=1");
  if (user.role === "ADMIN") return { client, role: "ADMIN", canManage: true };
  const membership = await db.clientMember.findUnique({
    where: { userId_clientId: { userId: user.id, clientId: client.id } },
  });
  if (!membership) redirect("/?unauthorized=1");
  return { client, role: membership.role, canManage: membership.role === "MANAGER" };
});

/** Clients visible to the current user (for switcher/portfolio). */
export const listAccessibleClients = cache(async () => {
  const user = await requireUser();
  if (user.role === "ADMIN") {
    return db.client.findMany({ orderBy: { name: "asc" } });
  }
  return db.client.findMany({
    where: { members: { some: { userId: user.id } } },
    orderBy: { name: "asc" },
  });
});
