import "server-only";
import { cache } from "react";
import { db } from "@/lib/db";

export type NotificationRow = {
  id: string;
  type: string;
  title: string;
  body: string;
  href: string | null;
  /** ISO timestamp */
  createdAt: string;
  read: boolean;
};

export type NotificationFeed = { items: NotificationRow[]; unread: number };

/** Latest notifications for the bell (per request cache). */
export const getNotificationFeed = cache(async (userId: string, limit = 20): Promise<NotificationFeed> => {
  const [rows, unread] = await Promise.all([
    db.notification.findMany({
      where: { userId },
      orderBy: { createdAt: "desc" },
      take: limit,
      select: { id: true, type: true, title: true, body: true, href: true, createdAt: true, readAt: true },
    }),
    db.notification.count({ where: { userId, readAt: null } }),
  ]);
  return {
    unread,
    items: rows.map((r) => ({
      id: r.id,
      type: r.type,
      title: r.title,
      body: r.body,
      href: r.href,
      createdAt: r.createdAt.toISOString(),
      read: r.readAt !== null,
    })),
  };
});
