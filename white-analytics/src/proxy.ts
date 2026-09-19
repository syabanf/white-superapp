import NextAuth from "next-auth";
import { NextResponse, type NextRequest } from "next/server";
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
  if (isLoggedIn) {
    const restored = restoreSavedFilters(req);
    if (restored) return restored;
  }
  return NextResponse.next();
});

function restoreSavedFilters(req: NextRequest) {
  const { pathname, searchParams } = req.nextUrl;
  const isPublishingCalendar = /^\/clients\/[^/]+\/publish\/?$/.test(pathname);
  const usesDateRange =
    pathname === "/" ||
    (/^\/clients\/[^/]+(?:\/|$)/.test(pathname) &&
      !/\/settings(?:\/|$)/.test(pathname) &&
      !/\/publish(?:\/|$)/.test(pathname) &&
      !/\/seo\/(?:research|backlinks|competitors)(?:\/|$)/.test(pathname));

  const cookieName = isPublishingCalendar ? "white_calendar_filters" : usesDateRange ? "white_date_range" : null;
  if (!cookieName) return null;
  const keys = isPublishingCalendar ? ["view", "date", "platform", "status"] : ["from", "to", "preset", "compare"];
  if (keys.some((key) => searchParams.has(key))) return null;
  const stored = req.cookies.get(cookieName)?.value;
  if (!stored) return null;

  try {
    const values = JSON.parse(decodeURIComponent(stored)) as Record<string, unknown>;
    const url = req.nextUrl.clone();
    for (const key of keys) {
      const value = values[key];
      if (typeof value === "string" && value && value !== "ALL") url.searchParams.set(key, value);
    }
    if (url.search === req.nextUrl.search) return null;
    return NextResponse.redirect(url);
  } catch {
    return null;
  }
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico|logo.*|.*\\.(?:png|jpg|jpeg|svg|webp|ico|txt|xml)$).*)"],
};
