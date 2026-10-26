import "server-only";
import { GatewayError } from "@/lib/erp/gateway";

export type Loaded<T> = { data: T; error: null } | { data: null; error: GatewayError };

/** Runs a gateway fetch for a page section; errors come back as values so one section can fail alone. */
export async function load<T>(fn: () => Promise<T>): Promise<Loaded<T>> {
  try {
    return { data: await fn(), error: null };
  } catch (e) {
    const err = e instanceof GatewayError ? e : new GatewayError(500, "UNKNOWN", "Terjadi kesalahan.");
    return { data: null, error: err };
  }
}

/** Builds "?a=1&b=2" from defined, non-empty values (arrays repeat the key). */
export function qs(params: Record<string, string | number | boolean | string[] | null | undefined>): string {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v == null || v === "") continue;
    if (Array.isArray(v)) v.forEach((x) => sp.append(k, x));
    else sp.set(k, String(v));
  }
  const s = sp.toString();
  return s ? `?${s}` : "";
}

/** First value of a search param. */
export function sp1(v: string | string[] | undefined): string | undefined {
  return Array.isArray(v) ? v[0] : v;
}

/** Positive integer search param with a fallback. */
export function spInt(v: string | string[] | undefined, fallback: number): number {
  const n = Number(sp1(v));
  return Number.isInteger(n) && n > 0 ? n : fallback;
}
