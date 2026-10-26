// Package openrouter generates the strategic insight markdown: the real
// OpenRouter chat completion when a key is configured, otherwise the
// deterministic template mock (model "mock").
package openrouter

import (
	"context"
	"net/http"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/reports/domain"
)

const (
	DefaultEndpoint = "https://openrouter.ai/api/v1/chat/completions"
	DefaultModel    = "anthropic/claude-sonnet-4.5"
	MockModel       = "mock"
	timeout         = 90 * time.Second
)

// Provider writes one insight. Model is the id stored on ai_insights rows.
type Provider interface {
	Model() string
	Generate(ctx context.Context, in domain.InsightInput) (string, error)
}

// Credentials come from integrations GetProviderCredentials(OPENROUTER):
// the secret is the API key, the public id an optional model override.
type Credentials struct {
	APIKey string
	Model  string
}

type Options struct {
	// Referer is sent as HTTP-Referer (APP_BASE_URL).
	Referer string
	// Endpoint overrides the API URL (tests).
	Endpoint   string
	HTTPClient *http.Client
}

// Pick returns the real adapter when an API key is configured.
func Pick(creds Credentials, opts Options) Provider {
	if creds.APIKey == "" {
		return Mock{}
	}
	return NewReal(creds, opts)
}

// CredentialsFunc fetches the current OPENROUTER credentials (cached by the caller).
type CredentialsFunc func(ctx context.Context) (Credentials, error)

// Source picks the provider per call so saving a key in the setup wizard
// switches to the real adapter without a restart.
type Source struct {
	creds CredentialsFunc
	opts  Options
}

func NewSource(creds CredentialsFunc, opts Options) *Source {
	return &Source{creds: creds, opts: opts}
}

// Generate returns the markdown and the model id to store.
func (s *Source) Generate(ctx context.Context, in domain.InsightInput) (string, string, error) {
	creds, err := s.creds(ctx)
	if err != nil {
		return "", "", err
	}
	p := Pick(creds, s.opts)
	content, err := p.Generate(ctx, in)
	if err != nil {
		return "", "", err
	}
	return content, p.Model(), nil
}
