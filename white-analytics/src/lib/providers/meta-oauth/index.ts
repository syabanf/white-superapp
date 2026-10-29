import { ProviderError, type ProviderErrorCode } from "@/lib/action-result";
import { META_DIALOG_BASE, META_GRAPH_BASE } from "@/lib/providers/meta-graph/version";
import { inDemoMode } from "@/lib/demo-mode";

/**
 * Meta (Facebook) OAuth 2.0 helpers — pure fetch-based, no SDK, no env access
 * inside the exchange functions so they stay unit-testable. Errors are normalized
 * into ProviderError("meta", code, message).
 */

const GRAPH_BASE = META_GRAPH_BASE;
const DIALOG_BASE = META_DIALOG_BASE;

/**
 * Semua izin yang dipakai aplikasi, sama dengan checklist App Review di wizard
 * Setup. Izin yang belum disetujui Meta tidak diberikan ke pengguna di luar
 * peran aplikasi; yang benar-benar diberikan dibaca ulang lewat /me/permissions.
 */
export const META_SCOPES = [
  "pages_show_list",
  "pages_read_engagement",
  "read_insights",
  "instagram_basic",
  "instagram_manage_insights",
  "business_management",
  "ads_read",
  "pages_manage_posts",
  "instagram_content_publish",
];

/** Page Facebook yang diizinkan pengguna, beserta akun IG Business yang tertaut. */
export type MetaAsset = {
  pageId: string;
  pageName: string;
  pageUsername: string | null;
  igId: string | null;
  igUsername: string | null;
};

/** Bentuk `Connection.meta` untuk koneksi META. */
export type MetaConnectionMeta = { assets: MetaAsset[]; grantedScopes: string[]; syncedAt: string };

export function parseMetaConnectionMeta(raw: unknown): MetaConnectionMeta | null {
  if (!raw || typeof raw !== "object") return null;
  const o = raw as Partial<MetaConnectionMeta>;
  if (!Array.isArray(o.assets)) return null;
  return { assets: o.assets, grantedScopes: Array.isArray(o.grantedScopes) ? o.grantedScopes : [], syncedAt: o.syncedAt ?? "" };
}

const norm = (v: string | null | undefined) => (v ?? "").trim().replace(/^@+/, "").toLowerCase();

/**
 * Cocokkan username yang diketik admin dengan aset Meta. Instagram dicocokkan
 * dengan username IG; Facebook dengan username Page, Page ID, atau nama Page.
 * Mengembalikan ID numerik yang dibutuhkan Graph API.
 */
export function resolveMetaAsset(
  assets: MetaAsset[],
  platform: "INSTAGRAM" | "FACEBOOK",
  username: string,
): { externalId: string; username: string; displayName: string } | null {
  const u = norm(username);
  if (!u) return null;
  if (platform === "INSTAGRAM") {
    const a = assets.find((x) => x.igId && norm(x.igUsername) === u);
    return a?.igId ? { externalId: a.igId, username: norm(a.igUsername), displayName: a.igUsername ?? a.pageName } : null;
  }
  const a = assets.find((x) => norm(x.pageUsername) === u || x.pageId === u || norm(x.pageName) === u);
  return a ? { externalId: a.pageId, username: norm(a.pageUsername) || a.pageId, displayName: a.pageName } : null;
}

/** ID koneksi demo (dibuat tombol simulasi) — tidak pernah dipanggil ke Meta. */
export function isDemoConnectionId(accountId: string): boolean {
  return accountId.startsWith("demo-");
}

export function metaConfigured(env: Record<string, string | undefined> = process.env): boolean {
  return !inDemoMode() && Boolean(env.META_APP_ID && env.META_APP_SECRET);
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

/** Izin yang benar-benar diberikan pengguna (status "granted"). */
export async function fetchMetaGrantedScopes(accessToken: string): Promise<string[]> {
  const data = (await graphGet("/me/permissions", { access_token: accessToken })) as { data?: { permission?: string; status?: string }[] };
  return (data.data ?? []).filter((p) => p.status === "granted" && p.permission).map((p) => p.permission!);
}

/**
 * Page yang diizinkan pengguna + akun IG Business tertaut (`/me/accounts`).
 * Token Page sengaja tidak disimpan: diambil ulang saat dipakai.
 */
export async function fetchMetaAssets(accessToken: string): Promise<MetaAsset[]> {
  type RawPage = { id: string; name?: string; username?: string; instagram_business_account?: { id?: string; username?: string } };
  const out: MetaAsset[] = [];
  let params: Record<string, string> | null = {
    fields: "id,name,username,instagram_business_account{id,username}",
    limit: "100",
    access_token: accessToken,
  };
  for (let page = 0; params && page < 10; page++) {
    const data = (await graphGet("/me/accounts", params)) as { data?: RawPage[]; paging?: { cursors?: { after?: string }; next?: string } };
    for (const p of data.data ?? []) {
      out.push({
        pageId: p.id,
        pageName: p.name ?? p.id,
        pageUsername: p.username ?? null,
        igId: p.instagram_business_account?.id ?? null,
        igUsername: p.instagram_business_account?.username ?? null,
      });
    }
    const after = data.paging?.next ? data.paging.cursors?.after : undefined;
    params = after ? { ...params, after } : null;
  }
  return out;
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
