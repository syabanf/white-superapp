import "server-only";
import { fetchApifyUsage } from "@/lib/providers/apify/client";

/**
 * Credential checks for the setup wizard — one cheap call per provider that
 * fails fast on a wrong id/secret. Returns a human message (Indonesian) plus
 * optional detail (e.g. DataForSEO balance).
 */
export type VerifyResult = { ok: boolean; message: string; detail?: Record<string, string | number> };

const TIMEOUT_MS = 12_000;

async function get(url: string, init?: RequestInit): Promise<{ status: number; json: unknown }> {
  const res = await fetch(url, { ...init, signal: AbortSignal.timeout(TIMEOUT_MS), cache: "no-store" });
  let json: unknown = null;
  try {
    json = await res.json();
  } catch {
    json = null;
  }
  return { status: res.status, json };
}

function errMessage(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

/** App access token via client_credentials proves app id + secret match. */
export async function verifyMetaApp(appId: string, appSecret: string): Promise<VerifyResult> {
  try {
    const q = new URLSearchParams({ client_id: appId, client_secret: appSecret, grant_type: "client_credentials" });
    const { status, json } = await get(`https://graph.facebook.com/v21.0/oauth/access_token?${q}`);
    if (status === 200 && json && typeof json === "object" && "access_token" in json) {
      return { ok: true, message: "Kredensial aplikasi Meta valid." };
    }
    const msg = (json as { error?: { message?: string } } | null)?.error?.message;
    return { ok: false, message: msg ? `Meta menolak: ${msg}` : "App ID atau App Secret tidak cocok." };
  } catch (e) {
    return { ok: false, message: `Tidak bisa menghubungi Meta: ${errMessage(e)}` };
  }
}

/**
 * Google has no "check client" endpoint; a bogus code exchange distinguishes
 * `invalid_client` (wrong id/secret) from `invalid_grant` (credentials fine).
 */
export async function verifyGoogleClient(clientId: string, clientSecret: string): Promise<VerifyResult> {
  try {
    const body = new URLSearchParams({
      client_id: clientId,
      client_secret: clientSecret,
      grant_type: "authorization_code",
      code: "white-analytics-probe",
      redirect_uri: "http://localhost/probe",
    });
    const { json } = await get("https://oauth2.googleapis.com/token", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body,
    });
    const err = (json as { error?: string } | null)?.error;
    if (err === "invalid_grant" || err === "redirect_uri_mismatch") return { ok: true, message: "Client ID & secret Google dikenali." };
    if (err === "invalid_client" || err === "unauthorized_client") return { ok: false, message: "Client ID atau Client Secret Google salah." };
    return { ok: false, message: `Respons tak terduga dari Google (${err ?? "tanpa kode"}).` };
  } catch (e) {
    return { ok: false, message: `Tidak bisa menghubungi Google: ${errMessage(e)}` };
  }
}

export async function verifyDataForSeo(login: string, password: string): Promise<VerifyResult> {
  try {
    const auth = Buffer.from(`${login}:${password}`).toString("base64");
    const { status, json } = await get("https://api.dataforseo.com/v3/appendix/user_data", {
      headers: { Authorization: `Basic ${auth}` },
    });
    if (status === 401) return { ok: false, message: "Login atau password DataForSEO salah." };
    const task = (json as { tasks?: { status_code?: number; result?: { money?: { balance?: number } }[] }[] } | null)?.tasks?.[0];
    if (status === 200 && task?.status_code === 20000) {
      const balance = task.result?.[0]?.money?.balance;
      return {
        ok: true,
        message: typeof balance === "number" ? `Terhubung. Saldo akun: $${balance.toFixed(2)}.` : "Terhubung ke DataForSEO.",
        detail: typeof balance === "number" ? { balanceUsd: balance } : undefined,
      };
    }
    return { ok: false, message: `DataForSEO menjawab status ${status}.` };
  } catch (e) {
    return { ok: false, message: `Tidak bisa menghubungi DataForSEO: ${errMessage(e)}` };
  }
}

export async function verifyOpenRouter(apiKey: string): Promise<VerifyResult> {
  try {
    const { status, json } = await get("https://openrouter.ai/api/v1/auth/key", { headers: { Authorization: `Bearer ${apiKey}` } });
    if (status === 200) {
      const label = (json as { data?: { label?: string } } | null)?.data?.label;
      return { ok: true, message: label ? `Kunci OpenRouter valid (${label}).` : "Kunci OpenRouter valid." };
    }
    if (status === 401) return { ok: false, message: "Kunci OpenRouter ditolak." };
    return { ok: false, message: `OpenRouter menjawab status ${status}.` };
  } catch (e) {
    return { ok: false, message: `Tidak bisa menghubungi OpenRouter: ${errMessage(e)}` };
  }
}

export async function verifyApify(token: string): Promise<VerifyResult> {
  try {
    const u = await fetchApifyUsage(token);
    const used = u.monthlyUsageUsd != null ? `$${u.monthlyUsageUsd.toFixed(2)}` : null;
    const limit = u.monthlyLimitUsd != null ? `$${u.monthlyLimitUsd.toFixed(2)}` : null;
    const who = u.username ? ` (${u.username}${u.plan ? `, ${u.plan}` : ""})` : "";
    const usage = used && limit ? ` Pemakaian bulan ini ${used} dari ${limit}.` : "";
    return { ok: true, message: `Token Apify valid${who}.${usage}`, detail: u.monthlyUsageUsd != null ? { monthlyUsageUsd: u.monthlyUsageUsd } : undefined };
  } catch (e) {
    return { ok: false, message: e instanceof Error ? e.message : String(e) };
  }
}
