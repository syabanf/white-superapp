// Package aichat picks where text AI requests go. Every service's provider
// speaks the OpenAI chat format, so OLLAMA_API_KEY switches them all to Ollama
// Cloud; without it they keep the OpenRouter credentials from integrations.
package aichat

import "github.com/syabanf/white-superapp/white-backend/pkg/config"

const (
	ollamaEndpoint     = "https://ollama.com/v1/chat/completions"
	defaultOllamaModel = "deepseek-v4.1-flash"
)

// Ollama returns the key and model when Ollama is configured.
func Ollama() (apiKey, model string, ok bool) {
	apiKey = config.Env("OLLAMA_API_KEY", "")
	return apiKey, config.Env("OLLAMA_MODEL", defaultOllamaModel), apiKey != ""
}

// Endpoint overrides a provider's URL; empty keeps its OpenRouter default.
func Endpoint() string {
	if _, _, ok := Ollama(); ok {
		return ollamaEndpoint
	}
	return ""
}
