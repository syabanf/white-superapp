import "server-only";
import { cache } from "react";
import { createHmac } from "node:crypto";
import { cookies } from "next/headers";
import { db } from "@/lib/db";
import type { Client } from "@/generated/prisma/client";

/**
 * Public share access helpers — no session involved. A dashboard is viewable when
 * `shareEnabled` and, if a PIN is set, the visitor carries the HMAC cookie issued by
 * `verifySharePin`. The cookie value is derived from the PIN hash, so rotating the PIN
 * invalidates every previously issued cookie.
 */

function shareSecret(): string {
  return process.env.AUTH_SECRET ?? process.env.ENCRYPTION_KEY ?? "white-share-dev-secret";
}

export function shareCookieName(clientId: string): string {
  return `share_${clientId}`;
}

export function shareCookieValue(pinHash: string): string {
  return createHmac("sha256", shareSecret()).update(pinHash).digest("hex");
}

/** Client for a public share slug (null when unknown). Cached per request. */
export const getShareClient = cache(async (slug: string): Promise<Client | null> => {
  return db.client.findUnique({ where: { slug } });
});

/** True when no PIN is required or the visitor's cookie matches the current PIN hash. */
export async function isShareUnlocked(client: Pick<Client, "id" | "sharePinHash">): Promise<boolean> {
  if (!client.sharePinHash) return true;
  const store = await cookies();
  const cookie = store.get(shareCookieName(client.id));
  return cookie?.value === shareCookieValue(client.sharePinHash);
}

export type ShareContext =
  | { status: "not-found" }
  | { status: "disabled"; client: Client }
  | { status: "locked"; client: Client }
  | { status: "ok"; client: Client };

/** One-stop state resolution shared by the layout, the page and the PDF route. */
export const getShareContext = cache(async (slug: string): Promise<ShareContext> => {
  const client = await getShareClient(slug);
  if (!client) return { status: "not-found" };
  if (!client.shareEnabled) return { status: "disabled", client };
  if (!(await isShareUnlocked(client))) return { status: "locked", client };
  return { status: "ok", client };
});
