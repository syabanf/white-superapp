import { mockInsightProvider } from "./mock";
import { realInsightProvider } from "./real";
import type { GenerateInsightInput, InsightProvider } from "./types";

export * from "./types";
export { generateMockInsight } from "./mock";

/** True when a real OpenRouter key is available. */
export function isOpenRouterConfigured(): boolean {
  return Boolean(process.env.OPENROUTER_API_KEY);
}

/** Picks the real adapter when configured, otherwise the deterministic mock. */
export function getInsightProvider(): InsightProvider {
  return isOpenRouterConfigured() ? realInsightProvider : mockInsightProvider;
}

/** Convenience: generate with the active provider. Returns content + the model id to store. */
export async function generateInsight(input: GenerateInsightInput): Promise<{ content: string; model: string }> {
  const provider = getInsightProvider();
  const content = await provider.generateInsight(input);
  return { content, model: provider.model };
}
