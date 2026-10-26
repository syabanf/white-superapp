import "server-only";
import { cache } from "react";
import { db } from "@/lib/db";
import { requireAdmin } from "@/lib/rbac";
import { gatewayFetch } from "@/lib/erp/gateway";

export type AdminUserRow = {
  id: string;
  name: string;
  email: string;
  role: "ADMIN" | "MEMBER";
  isActive: boolean;
  createdAt: string;
  /** ERP permission keys from identity; null when the backend is unreachable. */
  erpPermissions: string[] | null;
  clients: { id: string; name: string; slug: string; role: "MANAGER" | "VIEWER" }[];
};

/** All users incl. their client memberships — ADMIN only (RBAC in the data layer). */
export const listUsersWithClients = cache(async (): Promise<AdminUserRow[]> => {
  await requireAdmin();
  const users = await db.user.findMany({
    orderBy: { createdAt: "asc" },
    include: {
      memberships: {
        include: { client: { select: { id: true, name: true, slug: true } } },
        orderBy: { client: { name: "asc" } },
      },
    },
  });
  const erp = await loadErpPermissions();
  return users.map((u) => ({
    id: u.id,
    name: u.name,
    email: u.email,
    role: u.role,
    isActive: u.isActive,
    createdAt: u.createdAt.toISOString(),
    erpPermissions: erp ? (erp.get(u.id) ?? []) : null,
    clients: u.memberships.map((m) => ({
      id: m.client.id,
      name: m.client.name,
      slug: m.client.slug,
      role: m.role,
    })),
  }));
});

/** Clients available for the access checklist in the user dialog — ADMIN only. */
export const listClientOptions = cache(async (): Promise<{ id: string; name: string; slug: string }[]> => {
  await requireAdmin();
  return db.client.findMany({ orderBy: { name: "asc" }, select: { id: true, name: true, slug: true } });
});

type GatewayUsers = { users: { user: { id: string; permissions?: string[] } }[] };

async function loadErpPermissions(): Promise<Map<string, string[]> | null> {
  try {
    const res = await gatewayFetch<GatewayUsers>("/v1/users");
    return new Map(res.users.map((u) => [u.user.id, u.user.permissions ?? []]));
  } catch {
    return null;
  }
}

export type ErpPermissionOption = { key: string; label: string };

/** Permission catalog from identity; empty when the backend is unreachable. */
export const listErpPermissionOptions = cache(async (): Promise<ErpPermissionOption[]> => {
  await requireAdmin();
  try {
    const res = await gatewayFetch<{ permissions: ErpPermissionOption[] }>("/v1/permissions");
    return res.permissions;
  } catch {
    return [];
  }
});
