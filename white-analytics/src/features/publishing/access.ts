import "server-only";
import { revalidatePath } from "next/cache";
import { db } from "@/lib/db";
import { getSessionUser } from "@/lib/rbac";
import type { PostRole, ValidationIssue } from "@/features/publishing/lib";

export type PublishingAccess = {
  userId: string;
  userName: string;
  role: PostRole;
  clientId: string;
  slug: string;
  timezone: string;
};

/** RBAC in the data layer: session user + membership → role for the publishing state machine. */
export async function resolvePublishingAccess(clientId: string): Promise<PublishingAccess | null> {
  const user = await getSessionUser();
  if (!user) return null;
  const client = await db.client.findUnique({ where: { id: clientId }, select: { slug: true, timezone: true } });
  if (!client) return null;
  const base = { userId: user.id, userName: user.name, clientId, slug: client.slug, timezone: client.timezone };
  if (user.role === "ADMIN") return { ...base, role: "ADMIN" };
  const membership = await db.clientMember.findUnique({
    where: { userId_clientId: { userId: user.id, clientId } },
    select: { role: true },
  });
  if (!membership) return null;
  return { ...base, role: membership.role === "MANAGER" ? "MANAGER" : "VIEWER" };
}

export function revalidatePublishing(slug: string, postId?: string) {
  const base = `/clients/${slug}/publish`;
  revalidatePath(base);
  revalidatePath(`${base}/posts`);
  revalidatePath(`${base}/media`);
  if (postId) revalidatePath(`${base}/posts/${postId}`);
}

export function issuesToMessage(prefix: string, issues: ValidationIssue[]): string {
  const unique = Array.from(new Set(issues.map((i) => (i.scope === "post" ? i.message : `${platformName(i.scope)}: ${i.message}`))));
  return `${prefix} ${unique.join(" ")}`;
}

function platformName(p: string): string {
  return p === "INSTAGRAM" ? "Instagram" : p === "FACEBOOK" ? "Facebook" : p === "TIKTOK" ? "TikTok" : p;
}
