import { describe, it, expect, beforeAll } from "vitest";

// server-only module import is fine under vitest (node env) once the guard is stubbed
beforeAll(() => {
  process.env.ENCRYPTION_KEY = "ZGV2LW9ubHktZW5jcnlwdGlvbi1rZXktMzJieXRlcyEhIQ==";
});

describe("crypto", () => {
  it("encrypts and decrypts round-trip, with a fresh IV each time", async () => {
    const { encrypt, decrypt, randomPin } = await import("@/lib/crypto");
    const secret = "EAAB-long-lived-token-123";
    const a = encrypt(secret);
    const b = encrypt(secret);
    expect(a).not.toBe(b); // random IV
    expect(decrypt(a)).toBe(secret);
    expect(decrypt(b)).toBe(secret);
    expect(randomPin()).toMatch(/^\d{6}$/);
  });

  it("fails to decrypt tampered payloads (auth tag)", async () => {
    const { encrypt, decrypt } = await import("@/lib/crypto");
    const enc = encrypt("rahasia");
    const buf = Buffer.from(enc, "base64");
    buf[buf.length - 1] ^= 0xff;
    expect(() => decrypt(buf.toString("base64"))).toThrow();
  });
});
