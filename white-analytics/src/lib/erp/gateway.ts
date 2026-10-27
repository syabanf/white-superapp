import "server-only";
import { getSessionUser } from "@/lib/rbac";

/**
 * Bridge to the Go gateway (white-backend) for the ERP workspace. The
 * browser keeps its Auth.js session; this server trades the session user's
 * id plus INTERNAL_API_SECRET for a 15-minute access token and caches it per
 * user until shortly before it expires.
 */

type Cached = { token: string; expiresAt: number; permissions: string[]; role: string };

const globalCache = globalThis as unknown as { __erpTokens?: Map<string, Cached> };
const tokens = (globalCache.__erpTokens ??= new Map<string, Cached>());
const REFRESH_SKEW_MS = 60_000;

export function gatewayUrl(): string {
  return (process.env.GATEWAY_URL ?? "http://127.0.0.1:8380").replace(/\/$/, "");
}

export class GatewayError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
    public fieldErrors?: Record<string, string[]>,
  ) {
    super(message);
  }
}

/**
 * Upper bounds for gateway calls, so a slow or unreachable backend shows the
 * offline notice instead of holding the page render open until TCP gives up.
 */
export const GATEWAY_TIMEOUT = { session: 5_000, data: 15_000, transfer: 60_000 } as const;

async function mintToken(userId: string): Promise<Cached> {
  const secret = process.env.INTERNAL_API_SECRET;
  if (!secret) throw new GatewayError(503, "UNKNOWN", "INTERNAL_API_SECRET belum diset.");
  let res: Response;
  try {
    res = await fetch(`${gatewayUrl()}/internal/session`, {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-Internal-Secret": secret },
      body: JSON.stringify({ userId }),
      cache: "no-store",
      signal: AbortSignal.timeout(GATEWAY_TIMEOUT.session),
    });
  } catch {
    throw new GatewayError(503, "NETWORK", "Layanan ERP tidak dapat dihubungi. Pastikan backend berjalan.");
  }
  const body = await res.json().catch(() => ({}));
  if (!res.ok)
    throw new GatewayError(res.status, body.code ?? "UNKNOWN", body.error ?? "Gagal membuka sesi ERP.");
  return {
    token: body.accessToken,
    expiresAt: new Date(body.expiresAt).getTime(),
    permissions: body.user?.permissions ?? [],
    role: body.user?.role ?? "MEMBER",
  };
}

/** Access token for the signed-in user; null when nobody is signed in. */
export async function gatewaySession(): Promise<(Cached & { userId: string }) | null> {
  const user = await getSessionUser();
  if (!user) return null;
  let entry = tokens.get(user.id);
  if (!entry || entry.expiresAt - Date.now() < REFRESH_SKEW_MS) {
    entry = await mintToken(user.id);
    tokens.set(user.id, entry);
  }
  return { ...entry, userId: user.id };
}

/** Typed JSON call from server components and actions. */
export async function gatewayFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const session = await gatewaySession();
  if (!session) throw new GatewayError(401, "UNAUTHENTICATED", "Silakan masuk terlebih dahulu.");
  let res: Response;
  try {
    res = await fetch(`${gatewayUrl()}${path}`, {
      ...init,
      headers: {
        "Content-Type": "application/json",
        ...init.headers,
        Authorization: `Bearer ${session.token}`,
      },
      cache: "no-store",
      signal: init.signal ?? AbortSignal.timeout(GATEWAY_TIMEOUT.data),
    });
  } catch {
    throw new GatewayError(503, "NETWORK", "Layanan ERP tidak dapat dihubungi. Pastikan backend berjalan.");
  }
  const body = await res.json().catch(() => ({}));
  if (!res.ok)
    throw new GatewayError(
      res.status,
      body.code ?? "UNKNOWN",
      body.error ?? "Terjadi kesalahan.",
      body.fieldErrors,
    );
  return body as T;
}

/** Drops a user's cached token so role or permission changes apply now. */
export function forgetGatewayToken(userId: string) {
  tokens.delete(userId);
}

/**
 * Mirrors a user row the admin pages wrote into identity (same id and
 * bcrypt hash) so they can open the ERP. Best effort: a backend outage must
 * not block marketing user management, so failures only log.
 */
export async function syncUserToGateway(
  user:
    | { id: string; email: string; name: string; passwordHash: string; role: string; isActive: boolean }
    | { id: string; deleted: true },
): Promise<void> {
  const secret = process.env.INTERNAL_API_SECRET;
  if (!secret) return;
  forgetGatewayToken(user.id);
  try {
    const res = await fetch(`${gatewayUrl()}/internal/users/sync`, {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-Internal-Secret": secret },
      body: JSON.stringify(user),
      cache: "no-store",
      signal: AbortSignal.timeout(GATEWAY_TIMEOUT.session),
    });
    if (!res.ok) console.warn("[erp] sync user failed", user.id, res.status, await res.text());
  } catch (e) {
    console.warn("[erp] sync user failed", user.id, e);
  }
}
