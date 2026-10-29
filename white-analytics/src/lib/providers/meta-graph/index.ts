/**
 * Pemilih adapter Meta Graph: implementasi nyata bila kredensial ada di env,
 * selain itu mock deterministik (mode demo + DemoBanner di UI).
 */
import "server-only";
import { lazyProvider } from "@/lib/providers/lazy";
import type { MetaGraphProvider } from "./types";
import { mockMetaGraph } from "./mock";
import { realMetaGraph } from "./real";
import { inDemoMode } from "@/lib/demo-mode";

export function isMetaConfigured(): boolean {
  return !inDemoMode() && Boolean(process.env.META_APP_ID && process.env.META_APP_SECRET);
}

export const metaGraph: MetaGraphProvider = lazyProvider(() => (isMetaConfigured() ? realMetaGraph : mockMetaGraph));

export type * from "./types";
