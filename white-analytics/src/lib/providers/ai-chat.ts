/**
 * Where text AI requests go. Ollama Cloud when OLLAMA_API_KEY is set, otherwise
 * OpenRouter; both speak the OpenAI chat format. Images always use OpenRouter
 * because Ollama has no image model.
 */
export type AiChatTarget = { label: string; url: string; apiKey: string | undefined; model: string };

const OPENROUTER_URL = "https://openrouter.ai/api/v1/chat/completions";
const OLLAMA_URL = "https://ollama.com/v1/chat/completions";
const DEFAULT_OPENROUTER_MODEL = "anthropic/claude-sonnet-4.5";
const DEFAULT_OLLAMA_MODEL = "deepseek-v4.1-flash";

export function openRouterTarget(model = process.env.OPENROUTER_MODEL || DEFAULT_OPENROUTER_MODEL): AiChatTarget {
  return { label: "OpenRouter", url: OPENROUTER_URL, apiKey: process.env.OPENROUTER_API_KEY, model };
}

export function aiChatTarget(): AiChatTarget {
  const apiKey = process.env.OLLAMA_API_KEY;
  if (!apiKey) return openRouterTarget();
  return { label: "Ollama", url: OLLAMA_URL, apiKey, model: process.env.OLLAMA_MODEL || DEFAULT_OLLAMA_MODEL };
}
