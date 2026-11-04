import { afterEach, describe, expect, it, vi } from "vitest";
import { aiChatTarget, openRouterTarget } from "@/lib/providers/ai-chat";

afterEach(() => vi.unstubAllEnvs());

describe("aiChatTarget", () => {
  it("keeps OpenRouter without an Ollama key", () => {
    vi.stubEnv("OLLAMA_API_KEY", "");
    vi.stubEnv("OPENROUTER_API_KEY", "or-key");
    vi.stubEnv("OPENROUTER_MODEL", "some/model");
    expect(aiChatTarget()).toMatchObject({ label: "OpenRouter", apiKey: "or-key", model: "some/model" });
  });

  it("switches text to Ollama and defaults to DeepSeek", () => {
    vi.stubEnv("OLLAMA_API_KEY", "ol-key");
    vi.stubEnv("OLLAMA_MODEL", "");
    expect(aiChatTarget()).toMatchObject({ label: "Ollama", url: "https://ollama.com/v1/chat/completions", apiKey: "ol-key", model: "deepseek-v4.1-flash" });
  });

  it("leaves image calls on OpenRouter while Ollama is on", () => {
    vi.stubEnv("OLLAMA_API_KEY", "ol-key");
    vi.stubEnv("OPENROUTER_API_KEY", "or-key");
    expect(openRouterTarget("image/model")).toMatchObject({ label: "OpenRouter", apiKey: "or-key", model: "image/model" });
  });
});
