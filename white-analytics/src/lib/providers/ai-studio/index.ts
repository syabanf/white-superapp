/**
 * AI studio picker: the real adapter when a text AI key is configured and the client is not
 * in demo mode, otherwise the deterministic mock. Resolved per call.
 */
import "server-only";
import { inDemoMode } from "@/lib/demo-mode";
import { aiChatTarget } from "@/lib/providers/ai-chat";
import { lazyProvider } from "@/lib/providers/lazy";
import { mockAiStudio } from "./mock";
import { realAiStudio } from "./real";
import type { AiStudioProvider } from "./types";

export function isAiStudioConfigured(): boolean {
  return !inDemoMode() && Boolean(aiChatTarget().apiKey);
}

export const aiStudio: AiStudioProvider = lazyProvider(() => (isAiStudioConfigured() ? realAiStudio : mockAiStudio));

export { mockAiStudio };
export * from "./types";
