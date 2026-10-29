/**
 * AI studio picker: OpenRouter when a key is configured and the client is not
 * in demo mode, otherwise the deterministic mock. Resolved per call.
 */
import "server-only";
import { inDemoMode } from "@/lib/demo-mode";
import { lazyProvider } from "@/lib/providers/lazy";
import { mockAiStudio } from "./mock";
import { realAiStudio } from "./real";
import type { AiStudioProvider } from "./types";

export function isAiStudioConfigured(): boolean {
  return !inDemoMode() && Boolean(process.env.OPENROUTER_API_KEY);
}

export const aiStudio: AiStudioProvider = lazyProvider(() => (isAiStudioConfigured() ? realAiStudio : mockAiStudio));

export { mockAiStudio };
export * from "./types";
