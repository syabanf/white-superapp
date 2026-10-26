import { cp } from "../strings";

export class PortalError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message);
  }
}

/** JSON call from the candidate portals through the token-less proxy (`/api/public-erp` + path). */
export async function portalFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  let res: Response;
  try {
    res = await fetch(`/api/public-erp${path}`, { ...init, headers: { "Content-Type": "application/json", ...init.headers } });
  } catch {
    throw new PortalError(503, "NETWORK", cp.offline);
  }
  const body = await res.json().catch(() => ({}));
  if (!res.ok) throw new PortalError(res.status, body.code ?? "UNKNOWN", body.error ?? cp.error);
  return body as T;
}

export const portalPost = <T>(path: string, body: unknown = {}) => portalFetch<T>(path, { method: "POST", body: JSON.stringify(body) });

export const errorMessage = (e: unknown) => (e instanceof Error ? e.message : cp.error);
