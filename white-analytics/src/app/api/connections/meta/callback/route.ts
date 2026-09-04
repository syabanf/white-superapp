import { type NextRequest } from "next/server";
import { appBaseUrl, checkClientManage, META_NONCE_COOKIE, redirectTo } from "@/app/api/connections/_lib/access";
import { isProviderError } from "@/lib/action-result";
import { encrypt } from "@/lib/crypto";
import { db } from "@/lib/db";
import { exchangeMetaCode, exchangeMetaLongLivedToken, fetchMetaMe, META_SCOPES, metaConfigured } from "@/lib/providers/meta-oauth";
import { verifyOAuthState } from "@/lib/providers/oauth-state";

/** GET /api/connections/meta/callback — code exchange + Connection upsert. */
export async function GET(req: NextRequest) {
  const sp = req.nextUrl.searchParams;
  const state = verifyOAuthState(sp.get("state"), process.env.AUTH_SECRET ?? "");
  if (!state) return redirectTo(req, "/clients?missing=1");

  const settingsPath = `/clients/${state.slug}/settings`;
  const nonce = req.cookies.get(META_NONCE_COOKIE)?.value;
  if (!nonce || nonce !== state.nonce) return redirectTo(req, `${settingsPath}?error=oauth_state`);

  const access = await checkClientManage(state.slug);
  if (access === "unauthenticated") return redirectTo(req, "/login");
  if (!access) return redirectTo(req, "/clients?missing=1");
  if (!metaConfigured()) return redirectTo(req, `${settingsPath}?demo=1&provider=meta`);

  const code = sp.get("code");
  if (sp.get("error") || !code) return redirectTo(req, `${settingsPath}?error=oauth_denied`);

  try {
    const redirectUri = `${appBaseUrl(req)}/api/connections/meta/callback`;
    const appId = process.env.META_APP_ID!;
    const appSecret = process.env.META_APP_SECRET!;
    const shortLived = await exchangeMetaCode({ appId, appSecret, redirectUri, code });
    const longLived = await exchangeMetaLongLivedToken({ appId, appSecret, accessToken: shortLived.accessToken });
    const me = await fetchMetaMe(longLived.accessToken);
    const expiresAt = longLived.expiresIn ? new Date(Date.now() + longLived.expiresIn * 1000) : null;

    const payload = {
      displayName: me.name,
      accessTokenEnc: encrypt(longLived.accessToken),
      refreshTokenEnc: null,
      expiresAt,
      scopes: META_SCOPES,
    };
    await db.connection.upsert({
      where: { clientId_provider_accountId: { clientId: access.clientId, provider: "META", accountId: me.id } },
      create: { clientId: access.clientId, provider: "META", accountId: me.id, ...payload },
      update: payload,
    });

    const res = redirectTo(req, `${settingsPath}?connected=meta`);
    res.cookies.delete(META_NONCE_COOKIE);
    return res;
  } catch (e) {
    const code = isProviderError(e) ? e.code : "UNKNOWN";
    const res = redirectTo(req, `${settingsPath}?error=${code}`);
    res.cookies.delete(META_NONCE_COOKIE);
    return res;
  }
}
