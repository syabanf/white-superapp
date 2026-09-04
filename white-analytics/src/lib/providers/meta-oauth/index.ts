import { ProviderError, type ProviderErrorCode } from "@/lib/action-result";

/**
 * Meta (Facebook) OAuth 2.0 helpers — pure fetch-based, no SDK, no env access
 * inside the exchange functions so they stay unit-testable. Errors are normalized
 * into ProviderError("meta", code, message).
 */

const FB_VERSION = "v21.0";
const GRAPH_BASE = `https://graph.facebook.com/${FB_VERSION}`;
const DIALOG_BASE = `https://www.facebook.com/${FB_VERSION}/dialog/oauth`;

export const META_SCOPES = [
  "pages_show_list",
  "pages_read_engagement",
  "instagram_basic",
  "instagram_manage_insights",
  "business_management",
  "ads_read",
];

export function metaConfigured(env: Record<string, string | undefined> = process.env): boolean {
  return Boolean(env.META_APP_ID && env.META_APP_SECRET);
}

export function buildMetaAuthUrl(opts: { appId: string; redirectUri: string; state: string; scopes?: string[] }): string {
  const url = new URL(DIALOG_BASE);
  url.searchParams.set("client_id", opts.appId);
  url.searchParams.set("redirect_uri", opts.redirectUri);
  url.searchParams.set("state", opts.state);
  url.searchParams.set("response_type", "code");
  url.searchParams.set("scope", (opts.scopes ?? META_SCOPES).join(","));
  return url.toString();
}

export type MetaToken = { accessToken: string; expiresIn: number | null };

/** Exchange the authorization code for a short-lived user token. */
export async function exchangeMetaCode(opts: { appId: string; appSecret: string; redirectUri: string; code: string }): Promise<MetaToken> {
  const data = await graphGet("/oauth/access_token", {
    client_id: opts.appId,
    client_secret: opts.appSecret,
    redirect_uri: opts.redirectUri,
    code: opts.code,
  });
  return toToken(data);
}

/** Upgrade a short-lived token to a long-lived (~60 days) token. */
export async function exchangeMetaLongLivedToken(opts: { appId: string; appSecret: string; accessToken: string }): Promise<MetaToken> {
  const data = await graphGet("/oauth/access_token", {
    grant_type: "fb_exchange_token",
    client_id: opts.appId,
    client_secret: opts.appSecret,
    fb_exchange_token: opts.accessToken,
  });
  return toToken(data);
}

/** Identity of the token owner — used as Connection.accountId / displayName. */
export async function fetchMetaMe(accessToken: string): Promise<{ id: string; name: string }> {
  const data = await graphGet("/me", { fields: "id,name", access_token: accessToken });
  const rec = data as { id?: unknown; name?: unknown };
  if (typeof rec.id !== "string") throw new ProviderError("meta", "UNKNOWN", "Respons /me tidak valid");
  return { id: rec.id, name: typeof rec.name === "string" ? rec.name : rec.id };
}

// ── internals ────────────────────────────────────────────────

function toToken(data: unknown): MetaToken {
  const rec = data as { access_token?: unknown; expires_in?: unknown };
  if (typeof rec.access_token !== "string" || rec.access_token.length === 0) {
    throw new ProviderError("meta", "UNKNOWN", "Token tidak ditemukan pada respons Meta");
  }
  return { accessToken: rec.access_token, expiresIn: typeof rec.expires_in === "number" ? rec.expires_in : null };
}

async function graphGet(path: string, params: Record<string, string>): Promise<unknown> {
  const url = new URL(`${GRAPH_BASE}${path}`);
  for (const [k, v] of Object.entries(params)) url.searchParams.set(k, v);
  let res: Response;
  try {
    res = await fetch(url, { cache: "no-store" });
  } catch (cause) {
    throw new ProviderError("meta", "NETWORK", "Tidak dapat menghubungi Graph API", { cause });
  }
  let body: unknown = null;
  try {
    body = await res.json();
  } catch {
    // fall through — handled below
  }
  if (!res.ok) throw mapMetaError(res.status, body);
  if (body == null) throw new ProviderError("meta", "UNKNOWN", "Respons Graph API kosong");
  return body;
}

function mapMetaError(status: number, body: unknown): ProviderError {
  const err = (body as { error?: { message?: string; code?: number; type?: string } } | null)?.error;
  const message = err?.message ?? `Graph API error (HTTP ${status})`;
  const code = err?.code;
  let mapped: ProviderErrorCode = "UNKNOWN";
  if (status === 429 || code === 4 || code === 17 || code === 32 || code === 613) mapped = "RATE_LIMIT";
  else if (code === 190 || err?.type === "OAuthException") mapped = "TOKEN_EXPIRED";
  else if (status === 403 || code === 10 || (code != null && code >= 200 && code <= 299)) mapped = "PERMISSION";
  else if (status === 404 || code === 803) mapped = "NOT_FOUND";
  return new ProviderError("meta", mapped, message);
}
