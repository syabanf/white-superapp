"use server";

import { revalidatePath } from "next/cache";
import { z } from "zod";
import { db } from "@/lib/db";
import { getSessionUser } from "@/lib/rbac";
import { fail, ok, type ActionResult } from "@/lib/action-result";
import { t } from "@/i18n/id";

export async function markNotificationRead(id: string): Promise<ActionResult> {
  const user = await getSessionUser();
  if (!user) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const parsed = z.string().min(1).safeParse(id);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  // Scoped to the owner — a foreign id is a silent no-op, never an error leak.
  await db.notification.updateMany({ where: { id: parsed.data, userId: user.id, readAt: null }, data: { readAt: new Date() } });
  revalidatePath("/", "layout");
  return ok(undefined);
}

export async function markAllNotificationsRead(): Promise<ActionResult<{ count: number }>> {
  const user = await getSessionUser();
  if (!user) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const res = await db.notification.updateMany({ where: { userId: user.id, readAt: null }, data: { readAt: new Date() } });
  revalidatePath("/", "layout");
  return ok({ count: res.count });
}
