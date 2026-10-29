import "server-only";
import { mockGoogleSearchProvider } from "./mock";
import { realGoogleSearchProvider } from "./real";
import type { GoogleSearchProvider } from "./types";
import { inDemoMode } from "@/lib/demo-mode";

export type { Ga4ReportRequest, Ga4ReportRow, GoogleSearchProvider, GscDimension, GscQueryRequest, GscRow } from "./types";

/** True when Google OAuth credentials are configured on the server. */
export function isGoogleConfigured(): boolean {
  return !inDemoMode() && Boolean(process.env.GOOGLE_CLIENT_ID && process.env.GOOGLE_CLIENT_SECRET);
}

/** Real adapter when credentials exist, deterministic mock otherwise (DemoBanner mode). */
export function getGoogleSearchProvider(): GoogleSearchProvider {
  return isGoogleConfigured() ? realGoogleSearchProvider : mockGoogleSearchProvider;
}
