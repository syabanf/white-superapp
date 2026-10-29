/**
 * Apify picker: real actors when APIFY_TOKEN is set (hydrated from the setup
 * wizard), deterministic mock otherwise. `publicSocial` resolves per call.
 */
import "server-only";
import { lazyProvider } from "@/lib/providers/lazy";
import { apifyPublicSocial } from "./social-real";
import { mockPublicSocial } from "./social-mock";
import type { PublicSocialProvider } from "./types";

import { isApifyConfigured } from "./config";
export { isApifyConfigured, getApifyActors } from "./config";
export { apifySerpRanks } from "./serp";
export { fetchApifyUsage } from "./client";
export const publicSocial: PublicSocialProvider = lazyProvider(() => (isApifyConfigured() ? apifyPublicSocial : mockPublicSocial));
export type * from "./types";
