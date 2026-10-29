/**
 * Meta Ads writer picker. Unlike the read adapters this is NOT chosen by env
 * alone: a real write additionally needs a non-demo client, an ad account on a
 * real Meta connection, and the `ads_management` scope on that connection
 * (granted only after Meta App Review). Anything less → the simulation mock.
 *
 * The caller (features/campaigns/submit.ts) gathers the gate facts from the
 * database and runs inside `withClientMode`, so `inDemoMode()` is accurate.
 */
import "server-only";
import { inDemoMode } from "@/lib/demo-mode";
import { mockAdsWriter } from "./mock";
import { realAdsWriter } from "./real";
import { writerGate } from "./payload";
import type { MetaAdsWriter, WriterGate } from "./types";

/** Meta app credentials present and the current client is not in demo mode. */
export function isAdsWriterConfigured(): boolean {
  return !inDemoMode() && Boolean(process.env.META_APP_ID && process.env.META_APP_SECRET);
}

/** Evaluate the gate for one ad account connection. */
export function adsWriterGate(conn: { realConnection: boolean; scopes: string[] }): WriterGate {
  return writerGate({
    metaConfigured: Boolean(process.env.META_APP_ID && process.env.META_APP_SECRET),
    demoClient: inDemoMode(),
    realConnection: conn.realConnection,
    scopes: conn.scopes,
  });
}

export function pickAdsWriter(gate: WriterGate): MetaAdsWriter {
  return gate.mode === "real" ? realAdsWriter : mockAdsWriter;
}

export { mockAdsWriter, realAdsWriter, writerGate };
export * from "./types";
