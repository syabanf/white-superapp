package aichat

import "testing"

func TestOllamaSwitch(t *testing.T) {
	t.Setenv("OLLAMA_API_KEY", "")
	t.Setenv("OLLAMA_MODEL", "")
	if _, _, ok := Ollama(); ok {
		t.Fatal("Ollama should be off without a key")
	}
	if got := Endpoint(); got != "" {
		t.Fatalf("Endpoint without a key = %q, want empty", got)
	}

	t.Setenv("OLLAMA_API_KEY", "test-key")
	key, model, ok := Ollama()
	if !ok || key != "test-key" || model != defaultOllamaModel {
		t.Fatalf("Ollama() = %q, %q, %v", key, model, ok)
	}
	if got := Endpoint(); got != ollamaEndpoint {
		t.Fatalf("Endpoint with a key = %q", got)
	}

	t.Setenv("OLLAMA_MODEL", "deepseek-v4-pro:0813")
	if _, model, _ := Ollama(); model != "deepseek-v4-pro:0813" {
		t.Fatalf("model override ignored: %q", model)
	}
}
