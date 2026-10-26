import "server-only";
import { redirect } from "next/navigation";
import { requireUser } from "@/lib/rbac";
import { gatewaySession, GatewayError } from "@/lib/erp/gateway";
import { canAccess, type ErpAccess } from "@/components/shell/nav-config";

export type ErpContext = {
  access: ErpAccess;
  /** Set when the Go backend could not be reached; pages show a notice. */
  offline: GatewayError | null;
};

/** Signed-in user plus ERP permissions. Redirects home when perm is missing. */
export async function requireErp(perm?: string): Promise<ErpContext> {
  const user = await requireUser();
  let ctx: ErpContext;
  try {
    const s = await gatewaySession();
    ctx = { access: { role: s?.role ?? user.role, permissions: s?.permissions ?? [] }, offline: null };
  } catch (e) {
    ctx = {
      access: { role: user.role, permissions: [] },
      offline: e instanceof GatewayError ? e : new GatewayError(503, "NETWORK", String(e)),
    };
  }
  if (perm && !canAccess(ctx.access, perm)) redirect("/erp?unauthorized=1");
  return ctx;
}
