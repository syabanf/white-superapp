import { type NextRequest } from "next/server";
import { appBaseUrl, checkClientManage, GOOGLE_NONCE_COOKIE, nonceCookieOptions, redirectTo } from "@/app/api/connections/_lib/access";
import { buildGoogleAuthUrl, googleConfigured } from "@/lib/providers/google-oauth";
import { randomNonce, signOAuthState } from "@/lib/providers/oauth-state";

/** GET /api/connections/google/start?client=<slug> — kick off Google OAuth consent. */
export async function GET(req: NextRequest) {
  const slug = req.nextUrl.searchParams.get("client") ?? "";
  const access = await checkClientManage(slug);
  if (access === "unauthenticated") return redirectTo(req, "/login");
  if (!access) return redirectTo(req, "/clients?missing=1");

  const settingsPath = `/clients/${access.slug}/settings`;
  if (!googleConfigured()) {
    return redirectTo(req, `${settingsPath}?demo=1&provider=google`);
  }

  const nonce = randomNonce();
  const state = signOAuthState({ slug: access.slug, nonce }, process.env.AUTH_SECRET ?? "");
  const authUrl = buildGoogleAuthUrl({
    clientId: process.env.GOOGLE_CLIENT_ID!,
    redirectUri: `${appBaseUrl(req)}/api/connections/google/callback`,
    state,
  });
  const res = redirectTo(req, authUrl);
  res.cookies.set(GOOGLE_NONCE_COOKIE, nonce, nonceCookieOptions());
  return res;
}
