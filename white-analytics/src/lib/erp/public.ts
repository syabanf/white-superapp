import "server-only";
import { headers } from "next/headers";
import { GatewayError, gatewayUrl } from "@/lib/erp/gateway";

/** Visitor IP from the incoming request, for the gateway's per-IP rate limits. */
async function visitorIp(): Promise<string | null> {
  const h = await headers();
  return h.get("x-forwarded-for")?.split(",")[0]?.trim() || h.get("x-real-ip") || null;
}

/**
 * Unauthenticated JSON call to the gateway's public sales routes
 * (`/v1/public/...`) from server components. Never sends a token; forwards
 * the visitor IP as x-forwarded-for. Throws GatewayError like gatewayFetch.
 */
export async function publicFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  if (!/^\/(v1|api)\/(recruitment\/)?public\//.test(path))
    throw new GatewayError(404, "NOT_FOUND", "Endpoint tidak ditemukan.");
  const ip = await visitorIp();
  let res: Response;
  try {
    res = await fetch(`${gatewayUrl()}${path}`, {
      ...init,
      headers: {
        "Content-Type": "application/json",
        ...init.headers,
        ...(ip ? { "X-Forwarded-For": ip } : {}),
      },
      cache: "no-store",
    });
  } catch {
    throw new GatewayError(503, "NETWORK", "Layanan sedang tidak dapat dihubungi. Coba lagi beberapa saat.");
  }
  const body = await res.json().catch(() => ({}));
  if (!res.ok) throw new GatewayError(res.status, body.code ?? "UNKNOWN", body.error ?? "Terjadi kesalahan.", body.fieldErrors);
  return body as T;
}
