import { parseActorOverrides } from "./normalize";
import type { ApifyActors } from "./types";
import { inDemoMode } from "@/lib/demo-mode";

/** Env-only reads (hydrated from the setup wizard) — safe on the server at call time. */
export function isApifyConfigured(): boolean {
  return !inDemoMode() && Boolean(process.env.APIFY_TOKEN);
}

export function getApifyActors(): ApifyActors {
  return parseActorOverrides(process.env.APIFY_ACTORS);
}
