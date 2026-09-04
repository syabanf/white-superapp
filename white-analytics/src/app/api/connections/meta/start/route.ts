import { type NextRequest } from "next/server";
import { appBaseUrl, checkClientManage, META_NONCE_COOKIE, nonceCookieOptions, redirectTo } from "@/app/api/connections/_lib/access";
import { buildMetaAuthUrl, metaConfigured } from "@/lib/providers/meta-oauth";
import { randomNonce, signOAuthState } from "@/lib/providers/oauth-state";

/** GET /api/connections/meta/start?client=<slug> — kick off the Meta OAuth dialog. */
export async function GET(req: NextRequest) {
  const slug = req.nextUrl.searchParams.get("client") ?? "";
  const access = await checkClientManage(slug);
  if (access === "unauthenticated") return redirectTo(req, "/login");
  if (!access) return redirectTo(req, "/clients?missing=1");

  const settingsPath = `/clients/${access.slug}/settings`;
  if (!metaConfigured()) {
    return redirectTo(req, `${settingsPath}?demo=1&provider=meta`);
  }

  const nonce = randomNonce();
  const state = signOAuthState({ slug: access.slug, nonce }, process.env.AUTH_SECRET ?? "");
  const authUrl = buildMetaAuthUrl({
    appId: process.env.META_APP_ID!,
    redirectUri: `${appBaseUrl(req)}/api/connections/meta/callback`,
    state,
  });
  const res = redirectTo(req, authUrl);
  res.cookies.set(META_NONCE_COOKIE, nonce, nonceCookieOptions());
  return res;
}
