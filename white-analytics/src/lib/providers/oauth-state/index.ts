import { createHmac, randomBytes, timingSafeEqual } from "node:crypto";

/**
 * Signed OAuth `state` parameter: base64url(JSON{slug,nonce,ts}) + "." + HMAC-SHA256.
 * Pure (no server-only / env access) so it is unit-testable — callers pass the secret
 * (AUTH_SECRET). The nonce is mirrored in an httpOnly cookie and compared on callback.
 */

export type OAuthState = { slug: string; nonce: string; ts: number };

/** Default validity window for an OAuth round-trip. */
export const OAUTH_STATE_MAX_AGE_MS = 10 * 60 * 1000;

export function randomNonce(): string {
  return randomBytes(16).toString("hex");
}

function mac(body: string, secret: string): string {
  return createHmac("sha256", secret).update(body).digest("base64url");
}

export function signOAuthState(payload: { slug: string; nonce: string }, secret: string, now: number = Date.now()): string {
  if (!secret) throw new Error("OAuth state secret is missing (AUTH_SECRET)");
  const body = Buffer.from(JSON.stringify({ slug: payload.slug, nonce: payload.nonce, ts: now }), "utf8").toString("base64url");
  return `${body}.${mac(body, secret)}`;
}

/**
 * Verify signature + expiry. Returns the payload, or null on any failure
 * (tampering, malformed input, expired, future timestamp beyond clock skew).
 */
export function verifyOAuthState(
  state: string | null | undefined,
  secret: string,
  opts: { maxAgeMs?: number; now?: number } = {},
): OAuthState | null {
  if (!state || !secret) return null;
  const idx = state.lastIndexOf(".");
  if (idx <= 0 || idx === state.length - 1) return null;
  const body = state.slice(0, idx);
  const sig = Buffer.from(state.slice(idx + 1));
  const expected = Buffer.from(mac(body, secret));
  if (sig.length !== expected.length || !timingSafeEqual(sig, expected)) return null;

  let parsed: unknown;
  try {
    parsed = JSON.parse(Buffer.from(body, "base64url").toString("utf8"));
  } catch {
    return null;
  }
  if (typeof parsed !== "object" || parsed === null) return null;
  const { slug, nonce, ts } = parsed as Record<string, unknown>;
  if (typeof slug !== "string" || slug.length === 0 || !/^[a-z0-9-]+$/.test(slug)) return null;
  if (typeof nonce !== "string" || nonce.length === 0) return null;
  if (typeof ts !== "number" || !Number.isFinite(ts)) return null;

  const now = opts.now ?? Date.now();
  const maxAgeMs = opts.maxAgeMs ?? OAUTH_STATE_MAX_AGE_MS;
  if (ts > now + 60_000) return null; // future timestamp beyond small clock skew
  if (now - ts > maxAgeMs) return null;
  return { slug, nonce, ts };
}
