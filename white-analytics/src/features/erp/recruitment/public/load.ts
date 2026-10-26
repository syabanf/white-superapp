import "server-only";
import { GatewayError } from "@/lib/erp/gateway";
import { publicFetch } from "@/lib/erp/public";

export type PublicLoad<T> = { data: T; error: null } | { data: null; error: "not-found" | "offline" | "error"; message: string };

/** Fetches a public recruitment resource; 404 and outages come back as values for the page to render. */
export async function loadPublic<T>(path: string): Promise<PublicLoad<T>> {
  try {
    return { data: await publicFetch<T>(path), error: null };
  } catch (e) {
    const err = e instanceof GatewayError ? e : new GatewayError(500, "UNKNOWN", "Terjadi kesalahan.");
    const kind = err.status === 404 || err.code === "NOT_FOUND" ? "not-found" : err.code === "NETWORK" ? "offline" : "error";
    return { data: null, error: kind, message: err.message };
  }
}
