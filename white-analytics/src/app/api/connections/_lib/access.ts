import "server-only";
import { NextResponse, type NextRequest } from "next/server";
import { auth } from "@/auth";
import { db } from "@/lib/db";

/** Base URL used to build the OAuth redirect_uri (must match provider app config). */
export function appBaseUrl(req: NextRequest): string {
  const env = process.env.NEXT_PUBLIC_APP_URL;
  return (env && env.replace(/\/$/, "")) || req.nextUrl.origin;
}

/** Redirect helper relative to the current request origin (keeps dev port). */
export function redirectTo(req: NextRequest, path: string): NextResponse {
  return NextResponse.redirect(new URL(path, req.nextUrl));
}

export type ConnectAccess = { clientId: string; slug: string; userId: string };

/**
 * A connection may only be created/managed by an ADMIN or a MANAGER member of
 * the client. Returns null when unauthenticated / not found / not allowed.
 */
export async function checkClientManage(slug: string): Promise<ConnectAccess | "unauthenticated" | null> {
  if (!slug || !/^[a-z0-9-]+$/.test(slug)) return null;
  const session = await auth();
  const user = session?.user;
  if (!user?.id) return "unauthenticated";
  const client = await db.client.findUnique({ where: { slug }, select: { id: true } });
  if (!client) return null;
  if (user.role === "ADMIN") return { clientId: client.id, slug, userId: user.id };
  const membership = await db.clientMember.findUnique({
    where: { userId_clientId: { userId: user.id, clientId: client.id } },
    select: { role: true },
  });
  if (membership?.role !== "MANAGER") return null;
  return { clientId: client.id, slug, userId: user.id };
}

export const META_NONCE_COOKIE = "meta_oauth_nonce";
export const GOOGLE_NONCE_COOKIE = "google_oauth_nonce";

export function nonceCookieOptions() {
  return {
    httpOnly: true,
    sameSite: "lax" as const,
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: 600,
  };
}
