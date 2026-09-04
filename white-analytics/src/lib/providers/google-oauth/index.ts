import { ProviderError, type ProviderErrorCode } from "@/lib/action-result";

/**
 * Google OAuth 2.0 helpers — pure fetch-based (no SDK), errors normalized into
 * ProviderError("google", code, message). Scopes cover Search Console (read),
 * GA4 Data API (read) and basic identity.
 */

const AUTH_BASE = "https://accounts.google.com/o/oauth2/v2/auth";
const TOKEN_URL = "https://oauth2.googleapis.com/token";
const USERINFO_URL = "https://openidconnect.googleapis.com/v1/userinfo";

export const GOOGLE_SCOPES = [
  "https://www.googleapis.com/auth/webmasters.readonly",
  "https://www.googleapis.com/auth/analytics.readonly",
  "openid",
  "email",
];

export function googleConfigured(env: Record<string, string | undefined> = process.env): boolean {
  return Boolean(env.GOOGLE_CLIENT_ID && env.GOOGLE_CLIENT_SECRET);
}

export function buildGoogleAuthUrl(opts: { clientId: string; redirectUri: string; state: string; scopes?: string[] }): string {
  const url = new URL(AUTH_BASE);
  url.searchParams.set("client_id", opts.clientId);
  url.searchParams.set("redirect_uri", opts.redirectUri);
  url.searchParams.set("response_type", "code");
  url.searchParams.set("access_type", "offline");
  url.searchParams.set("prompt", "consent");
  url.searchParams.set("include_granted_scopes", "true");
  url.searchParams.set("scope", (opts.scopes ?? GOOGLE_SCOPES).join(" "));
  url.searchParams.set("state", opts.state);
  return url.toString();
}

export type GoogleTokens = {
  accessToken: string;
  refreshToken: string | null;
  expiresIn: number | null;
  scopes: string[];
};

/** Exchange the authorization code for access + refresh tokens. */
export async function exchangeGoogleCode(opts: { clientId: string; clientSecret: string; redirectUri: string; code: string }): Promise<GoogleTokens> {
  const body = new URLSearchParams({
    grant_type: "authorization_code",
    client_id: opts.clientId,
    client_secret: opts.clientSecret,
    redirect_uri: opts.redirectUri,
    code: opts.code,
  });
  let res: Response;
  try {
    res = await fetch(TOKEN_URL, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body,
      cache: "no-store",
    });
  } catch (cause) {
    throw new ProviderError("google", "NETWORK", "Tidak dapat menghubungi server token Google", { cause });
  }
  let data: unknown = null;
  try {
    data = await res.json();
  } catch {
    // handled below
  }
  if (!res.ok) throw mapGoogleError(res.status, data);
  const rec = data as { access_token?: unknown; refresh_token?: unknown; expires_in?: unknown; scope?: unknown };
  if (typeof rec.access_token !== "string" || rec.access_token.length === 0) {
    throw new ProviderError("google", "UNKNOWN", "Token tidak ditemukan pada respons Google");
  }
  return {
    accessToken: rec.access_token,
    refreshToken: typeof rec.refresh_token === "string" ? rec.refresh_token : null,
    expiresIn: typeof rec.expires_in === "number" ? rec.expires_in : null,
    scopes: typeof rec.scope === "string" ? rec.scope.split(" ").filter(Boolean) : [],
  };
}

/** OpenID userinfo — email is used as the Connection displayName. */
export async function fetchGoogleUserinfo(accessToken: string): Promise<{ sub: string; email: string | null; name: string | null }> {
  let res: Response;
  try {
    res = await fetch(USERINFO_URL, { headers: { Authorization: `Bearer ${accessToken}` }, cache: "no-store" });
  } catch (cause) {
    throw new ProviderError("google", "NETWORK", "Tidak dapat menghubungi userinfo Google", { cause });
  }
  let data: unknown = null;
  try {
    data = await res.json();
  } catch {
    // handled below
  }
  if (!res.ok) throw mapGoogleError(res.status, data);
  const rec = data as { sub?: unknown; email?: unknown; name?: unknown };
  if (typeof rec.sub !== "string") throw new ProviderError("google", "UNKNOWN", "Respons userinfo tidak valid");
  return {
    sub: rec.sub,
    email: typeof rec.email === "string" ? rec.email : null,
    name: typeof rec.name === "string" ? rec.name : null,
  };
}

function mapGoogleError(status: number, body: unknown): ProviderError {
  const rec = body as { error?: unknown; error_description?: unknown } | null;
  const errCode = typeof rec?.error === "string" ? rec.error : (rec?.error as { status?: string } | undefined)?.status;
  const message =
    (typeof rec?.error_description === "string" && rec.error_description) ||
    (typeof (rec?.error as { message?: string } | undefined)?.message === "string" && (rec?.error as { message: string }).message) ||
    `Google OAuth error (HTTP ${status})`;
  let mapped: ProviderErrorCode = "UNKNOWN";
  if (status === 429 || errCode === "RESOURCE_EXHAUSTED") mapped = "RATE_LIMIT";
  else if (errCode === "invalid_grant" || status === 401) mapped = "TOKEN_EXPIRED";
  else if (status === 403 || errCode === "access_denied" || errCode === "PERMISSION_DENIED") mapped = "PERMISSION";
  else if (status === 404) mapped = "NOT_FOUND";
  return new ProviderError("google", mapped, message);
}

/** Refresh an access token with the stored refresh token (Google rotates neither). */
export async function refreshGoogleAccessToken(opts: { clientId: string; clientSecret: string; refreshToken: string }): Promise<{ accessToken: string; expiresIn: number | null }> {
  const body = new URLSearchParams({
    client_id: opts.clientId,
    client_secret: opts.clientSecret,
    refresh_token: opts.refreshToken,
    grant_type: "refresh_token",
  });
  const res = await fetch("https://oauth2.googleapis.com/token", {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body,
    signal: AbortSignal.timeout(15_000),
    cache: "no-store",
  });
  const data = (await res.json().catch(() => ({}))) as { access_token?: unknown; expires_in?: unknown; error?: unknown };
  if (!res.ok || typeof data.access_token !== "string") {
    const { ProviderError } = await import("@/lib/action-result");
    const code = data.error === "invalid_grant" ? "TOKEN_EXPIRED" : "PERMISSION";
    throw new ProviderError("google", code, `Google menolak refresh token (${String(data.error ?? res.status)}).`);
  }
  return { accessToken: data.access_token, expiresIn: typeof data.expires_in === "number" ? data.expires_in : null };
}
