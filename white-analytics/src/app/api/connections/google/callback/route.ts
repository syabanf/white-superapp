import { type NextRequest } from "next/server";
import { appBaseUrl, checkClientManage, GOOGLE_NONCE_COOKIE, redirectTo } from "@/app/api/connections/_lib/access";
import { isProviderError } from "@/lib/action-result";
import { encrypt } from "@/lib/crypto";
import { db } from "@/lib/db";
import { exchangeGoogleCode, fetchGoogleUserinfo, GOOGLE_SCOPES, googleConfigured } from "@/lib/providers/google-oauth";
import { verifyOAuthState } from "@/lib/providers/oauth-state";

/** GET /api/connections/google/callback — code exchange + Connection upsert. */
export async function GET(req: NextRequest) {
  const sp = req.nextUrl.searchParams;
  const state = verifyOAuthState(sp.get("state"), process.env.AUTH_SECRET ?? "");
  if (!state) return redirectTo(req, "/clients?missing=1");

  const settingsPath = `/clients/${state.slug}/settings`;
  const nonce = req.cookies.get(GOOGLE_NONCE_COOKIE)?.value;
  if (!nonce || nonce !== state.nonce) return redirectTo(req, `${settingsPath}?error=oauth_state`);

  const access = await checkClientManage(state.slug);
  if (access === "unauthenticated") return redirectTo(req, "/login");
  if (!access) return redirectTo(req, "/clients?missing=1");
  if (!googleConfigured()) return redirectTo(req, `${settingsPath}?demo=1&provider=google`);

  const code = sp.get("code");
  if (sp.get("error") || !code) return redirectTo(req, `${settingsPath}?error=oauth_denied`);

  try {
    const tokens = await exchangeGoogleCode({
      clientId: process.env.GOOGLE_CLIENT_ID!,
      clientSecret: process.env.GOOGLE_CLIENT_SECRET!,
      redirectUri: `${appBaseUrl(req)}/api/connections/google/callback`,
      code,
    });
    const userinfo = await fetchGoogleUserinfo(tokens.accessToken);
    const expiresAt = tokens.expiresIn ? new Date(Date.now() + tokens.expiresIn * 1000) : null;

    const refreshTokenEnc = tokens.refreshToken ? encrypt(tokens.refreshToken) : null;
    const payload = {
      displayName: userinfo.email ?? userinfo.name ?? "Akun Google",
      accessTokenEnc: encrypt(tokens.accessToken),
      expiresAt,
      scopes: tokens.scopes.length > 0 ? tokens.scopes : GOOGLE_SCOPES,
    };
    await db.connection.upsert({
      where: { clientId_provider_accountId: { clientId: access.clientId, provider: "GOOGLE", accountId: userinfo.sub } },
      create: { clientId: access.clientId, provider: "GOOGLE", accountId: userinfo.sub, refreshTokenEnc, ...payload },
      // keep the previously stored refresh token when Google does not return a new one
      update: refreshTokenEnc ? { refreshTokenEnc, ...payload } : payload,
    });

    const res = redirectTo(req, `${settingsPath}?connected=google`);
    res.cookies.delete(GOOGLE_NONCE_COOKIE);
    return res;
  } catch (e) {
    const code = isProviderError(e) ? e.code : "UNKNOWN";
    const res = redirectTo(req, `${settingsPath}?error=${code}`);
    res.cookies.delete(GOOGLE_NONCE_COOKIE);
    return res;
  }
}
