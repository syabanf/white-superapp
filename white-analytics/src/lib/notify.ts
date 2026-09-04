import "server-only";
import { db } from "@/lib/db";

/**
 * In-app notifications (bell in the topbar). Fire-and-forget from actions/cron:
 * `await notify(userIds, { type: "POST_REVIEW", title, body, href, clientId })`.
 * `type` is a free-form upper-snake tag per module (POST_REVIEW, POST_APPROVED,
 * POST_PUBLISHED, POST_FAILED, RANK_DROP, BACKLINK_LOST, ...).
 */
export type NotificationInput = {
  type: string;
  title: string;
  body?: string;
  href?: string;
  clientId?: string;
};

export async function notify(userIds: readonly string[], input: NotificationInput): Promise<number> {
  const ids = Array.from(new Set(userIds)).filter(Boolean);
  if (ids.length === 0) return 0;
  const res = await db.notification.createMany({
    data: ids.map((userId) => ({
      userId,
      clientId: input.clientId,
      type: input.type,
      title: input.title,
      body: input.body ?? "",
      href: input.href,
    })),
  });
  return res.count;
}

/** Everyone who should hear about a client: ADMINs + the client's members (minus `except`). */
export async function clientAudience(clientId: string, except?: string): Promise<string[]> {
  const [admins, members] = await Promise.all([
    db.user.findMany({ where: { role: "ADMIN", isActive: true }, select: { id: true } }),
    db.clientMember.findMany({ where: { clientId, user: { isActive: true } }, select: { userId: true } }),
  ]);
  const ids = new Set<string>([...admins.map((a) => a.id), ...members.map((m) => m.userId)]);
  if (except) ids.delete(except);
  return Array.from(ids);
}

/** Managers/admins only — e.g. "a post is waiting for review". */
export async function clientApprovers(clientId: string, except?: string): Promise<string[]> {
  const [admins, managers] = await Promise.all([
    db.user.findMany({ where: { role: "ADMIN", isActive: true }, select: { id: true } }),
    db.clientMember.findMany({ where: { clientId, role: "MANAGER", user: { isActive: true } }, select: { userId: true } }),
  ]);
  const ids = new Set<string>([...admins.map((a) => a.id), ...managers.map((m) => m.userId)]);
  if (except) ids.delete(except);
  return Array.from(ids);
}
