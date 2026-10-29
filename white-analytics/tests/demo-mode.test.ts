import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { inDemoMode, runInDemoMode } from "@/lib/demo-mode";
import { isApifyConfigured } from "@/lib/providers/apify/config";
import { metaConfigured } from "@/lib/providers/meta-oauth";
import { googleConfigured } from "@/lib/providers/google-oauth";
import { isOpenRouterConfigured } from "@/lib/providers/openrouter";

const KEYS = ["APIFY_TOKEN", "META_APP_ID", "META_APP_SECRET", "GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET", "OPENROUTER_API_KEY"] as const;
let saved: Record<string, string | undefined> = {};

beforeEach(() => {
  saved = Object.fromEntries(KEYS.map((k) => [k, process.env[k]]));
  for (const k of KEYS) process.env[k] = "real-credential";
});
afterEach(() => {
  for (const k of KEYS) {
    if (saved[k] === undefined) delete process.env[k];
    else process.env[k] = saved[k];
  }
});

describe("demo mode per client", () => {
  it("uses real credentials outside demo mode", () => {
    expect(inDemoMode()).toBe(false);
    expect(isApifyConfigured()).toBe(true);
    expect(metaConfigured()).toBe(true);
  });

  it("reports every provider as unconfigured inside demo mode", () => {
    runInDemoMode(true, () => {
      expect(inDemoMode()).toBe(true);
      expect(isApifyConfigured()).toBe(false);
      expect(metaConfigured()).toBe(false);
      expect(googleConfigured()).toBe(false);
      expect(isOpenRouterConfigured()).toBe(false);
    });
  });

  it("keeps the context across awaits and leaves it afterwards", async () => {
    await runInDemoMode(true, async () => {
      await new Promise((r) => setTimeout(r, 5));
      expect(isApifyConfigured()).toBe(false);
    });
    expect(isApifyConfigured()).toBe(true);
  });

  it("does nothing for non-demo clients", () => {
    runInDemoMode(false, () => expect(isApifyConfigured()).toBe(true));
  });
});
