import { createHmac } from "node:crypto";
import { describe, expect, it } from "vitest";
import {
  OAUTH_STATE_MAX_AGE_MS,
  randomNonce,
  signOAuthState,
  verifyOAuthState,
} from "@/lib/providers/oauth-state";

const SECRET = "test-secret-please-rotate";

/** Sign an arbitrary payload with the same algorithm (to test payload-shape validation). */
function forgeSigned(payload: unknown): string {
  const body = Buffer.from(JSON.stringify(payload), "utf8").toString("base64url");
  const sig = createHmac("sha256", SECRET).update(body).digest("base64url");
  return `${body}.${sig}`;
}

describe("oauth state sign/verify", () => {
  it("round-trips a valid payload", () => {
    const now = 1_750_000_000_000;
    const state = signOAuthState({ slug: "kopi-nusantara", nonce: "abc123" }, SECRET, now);
    const out = verifyOAuthState(state, SECRET, { now: now + 1000 });
    expect(out).toEqual({ slug: "kopi-nusantara", nonce: "abc123", ts: now });
  });

  it("rejects a tampered signature", () => {
    const state = signOAuthState({ slug: "kopi-nusantara", nonce: "abc123" }, SECRET);
    const tampered = state.slice(0, -2) + (state.endsWith("aa") ? "bb" : "aa");
    expect(verifyOAuthState(tampered, SECRET)).toBeNull();
  });

  it("rejects a tampered body (slug swap)", () => {
    const now = Date.now();
    const state = signOAuthState({ slug: "kopi-nusantara", nonce: "abc123" }, SECRET, now);
    const [, sig] = state.split(".");
    const forgedBody = Buffer.from(JSON.stringify({ slug: "klien-lain", nonce: "abc123", ts: now }), "utf8").toString("base64url");
    expect(verifyOAuthState(`${forgedBody}.${sig}`, SECRET)).toBeNull();
  });

  it("rejects when verified with a different secret", () => {
    const state = signOAuthState({ slug: "kopi-nusantara", nonce: "abc123" }, SECRET);
    expect(verifyOAuthState(state, "another-secret")).toBeNull();
  });

  it("expires after the max age (default 10 minutes)", () => {
    const now = 1_750_000_000_000;
    const state = signOAuthState({ slug: "kopi-nusantara", nonce: "n" }, SECRET, now);
    expect(verifyOAuthState(state, SECRET, { now: now + OAUTH_STATE_MAX_AGE_MS - 1 })).not.toBeNull();
    expect(verifyOAuthState(state, SECRET, { now: now + OAUTH_STATE_MAX_AGE_MS + 1 })).toBeNull();
  });

  it("honors a custom maxAgeMs", () => {
    const now = 1_750_000_000_000;
    const state = signOAuthState({ slug: "kopi", nonce: "n" }, SECRET, now);
    expect(verifyOAuthState(state, SECRET, { now: now + 5_000, maxAgeMs: 4_000 })).toBeNull();
    expect(verifyOAuthState(state, SECRET, { now: now + 3_000, maxAgeMs: 4_000 })).not.toBeNull();
  });

  it("rejects timestamps too far in the future (clock skew guard)", () => {
    const now = 1_750_000_000_000;
    const state = signOAuthState({ slug: "kopi", nonce: "n" }, SECRET, now + 120_000);
    expect(verifyOAuthState(state, SECRET, { now })).toBeNull();
  });

  it("rejects malformed input", () => {
    expect(verifyOAuthState(null, SECRET)).toBeNull();
    expect(verifyOAuthState(undefined, SECRET)).toBeNull();
    expect(verifyOAuthState("", SECRET)).toBeNull();
    expect(verifyOAuthState("no-dot-here", SECRET)).toBeNull();
    expect(verifyOAuthState("a.", SECRET)).toBeNull();
    expect(verifyOAuthState(".b", SECRET)).toBeNull();
    expect(verifyOAuthState("not-base64.not-a-mac", SECRET)).toBeNull();
  });

  it("rejects payloads with invalid fields", () => {
    const now = Date.now();
    expect(verifyOAuthState(forgeSigned({ slug: "ok-slug", nonce: "n", ts: now }), SECRET, { now })).not.toBeNull();
    expect(verifyOAuthState(forgeSigned({ slug: "", nonce: "n", ts: now }), SECRET, { now })).toBeNull();
    expect(verifyOAuthState(forgeSigned({ slug: "UPPER CASE", nonce: "n", ts: now }), SECRET, { now })).toBeNull();
    expect(verifyOAuthState(forgeSigned({ slug: "ok-slug", nonce: "", ts: now }), SECRET, { now })).toBeNull();
    expect(verifyOAuthState(forgeSigned({ slug: "ok-slug", nonce: "n", ts: "not-a-number" }), SECRET, { now })).toBeNull();
    expect(verifyOAuthState(forgeSigned(["array"]), SECRET, { now })).toBeNull();
  });

  it("requires a secret to sign", () => {
    expect(() => signOAuthState({ slug: "a", nonce: "b" }, "")).toThrow();
  });

  it("generates distinct nonces", () => {
    const a = randomNonce();
    const b = randomNonce();
    expect(a).not.toEqual(b);
    expect(a).toMatch(/^[0-9a-f]{32}$/);
  });
});
