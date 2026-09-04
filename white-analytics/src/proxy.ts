import NextAuth from "next-auth";
import { NextResponse } from "next/server";
import { authConfig } from "@/auth.config";

const { auth } = NextAuth(authConfig);

/**
 * Optimistic auth check (cookie only — no DB). Real authorization happens in the
 * data-access layer (`src/lib/rbac.ts`).
 */
export const proxy = auth((req) => {
  const isLoggedIn = !!req.auth?.user;
  const { pathname, search } = req.nextUrl;

  const isPublic =
    pathname === "/login" ||
    pathname.startsWith("/share/") ||
    pathname.startsWith("/api/auth") ||
    pathname.startsWith("/api/cron") ||
    pathname.startsWith("/api/share");

  if (!isLoggedIn && !isPublic) {
    const url = new URL("/login", req.nextUrl);
    const callback = pathname + search;
    if (callback !== "/") url.searchParams.set("callbackUrl", callback);
    return NextResponse.redirect(url);
  }
  if (isLoggedIn && pathname === "/login") {
    return NextResponse.redirect(new URL("/", req.nextUrl));
  }
  return NextResponse.next();
});

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico|logo.*|.*\\.(?:png|jpg|jpeg|svg|webp|ico|txt|xml)$).*)"],
};
