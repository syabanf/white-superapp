/**
 * Publisher adapter picker: real Meta Graph when credentials are present,
 * otherwise the deterministic mock (demo mode + DemoBanner in the UI).
 */
import "server-only";
import { lazyProvider } from "@/lib/providers/lazy";
import type { SocialPublisher } from "./types";
import { mockPublisher } from "./mock";
import { realPublisher } from "./real";
import { inDemoMode } from "@/lib/demo-mode";

export function isPublisherConfigured(): boolean {
  return !inDemoMode() && Boolean(process.env.META_APP_ID && process.env.META_APP_SECRET);
}

export const socialPublisher: SocialPublisher = lazyProvider(() => (isPublisherConfigured() ? realPublisher : mockPublisher));

export { mockPublisher };
export type * from "./types";
