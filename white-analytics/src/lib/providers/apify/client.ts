import "server-only";
import { ProviderError } from "@/lib/action-result";
import { actorPath } from "./normalize";
import type { ApifyUsage } from "./types";

const BASE = "https://api.apify.com/v2";

function token(): string {
  const t = process.env.APIFY_TOKEN;
  if (!t) throw new ProviderError("apify", "PERMISSION", "APIFY_TOKEN belum diatur.", { requiresReconnect: false });
  return t;
}

function mapStatus(status: number, body: string): ProviderError {
  if (status === 401 || status === 403) return new ProviderError("apify", "TOKEN_EXPIRED", "Token Apify ditolak.");
  if (status === 402) return new ProviderError("apify", "PERMISSION", "Kuota/kredit Apify habis (402).", { requiresReconnect: false });
  if (status === 404) return new ProviderError("apify", "NOT_FOUND", "Actor Apify tidak ditemukan — periksa override actor.");
  if (status === 408) return new ProviderError("apify", "NETWORK", "Run Apify melebihi batas waktu sinkron (408).");
  if (status === 429) return new ProviderError("apify", "RATE_LIMIT", "Terlalu banyak permintaan ke Apify (429).");
  return new ProviderError("apify", "UNKNOWN", `Apify menjawab ${status}: ${body.slice(0, 200)}`);
}

/**
 * Run an actor synchronously and return its default dataset items.
 * Keeps within Apify's 300 s sync window; callers batch accordingly.
 */
export async function runActor<T = unknown>(
  actorId: string,
  input: Record<string, unknown>,
  opts: { timeoutSecs?: number; memoryMb?: number; maxItems?: number } = {},
): Promise<T[]> {
  const q = new URLSearchParams({
    token: token(),
    timeout: String(opts.timeoutSecs ?? 120),
    memory: String(opts.memoryMb ?? 1024),
    clean: "true",
    format: "json",
  });
  if (opts.maxItems) q.set("limit", String(opts.maxItems));
  let res: Response;
  try {
    res = await fetch(`${BASE}/acts/${actorPath(actorId)}/run-sync-get-dataset-items?${q}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
      signal: AbortSignal.timeout(((opts.timeoutSecs ?? 120) + 15) * 1000),
      cache: "no-store",
    });
  } catch (e) {
    throw new ProviderError("apify", "NETWORK", `Tidak bisa menghubungi Apify: ${e instanceof Error ? e.message : String(e)}`, { cause: e });
  }
  if (!res.ok) throw mapStatus(res.status, await res.text().catch(() => ""));
  const data: unknown = await res.json().catch(() => []);
  return Array.isArray(data) ? (data as T[]) : [];
}

/** Token check + monthly usage for the setup wizard. */
export async function fetchApifyUsage(tok: string): Promise<ApifyUsage> {
  const get = async (path: string) => {
    const res = await fetch(`${BASE}${path}?token=${encodeURIComponent(tok)}`, { signal: AbortSignal.timeout(12_000), cache: "no-store" });
    if (!res.ok) throw mapStatus(res.status, await res.text().catch(() => ""));
    return (await res.json()) as { data?: Record<string, unknown> };
  };
  const me = (await get("/users/me")).data ?? {};
  let usage: Record<string, unknown> = {};
  try {
    usage = (await get("/users/me/limits")).data ?? {};
  } catch {
    /* limits endpoint is optional */
  }
  const current = (usage.current ?? {}) as Record<string, unknown>;
  const limits = (usage.limits ?? {}) as Record<string, unknown>;
  const plan = me.plan && typeof me.plan === "object" ? ((me.plan as Record<string, unknown>).id as string | undefined) : undefined;
  return {
    username: typeof me.username === "string" ? me.username : null,
    plan: plan ?? null,
    monthlyUsageUsd: typeof current.monthlyUsageUsd === "number" ? current.monthlyUsageUsd : null,
    monthlyLimitUsd: typeof limits.monthlyUsageUsd === "number" ? limits.monthlyUsageUsd : null,
  };
}
