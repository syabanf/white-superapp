// Package openrouter writes the recap texts: the OpenRouter chat
// completion when a key is configured in integrations, otherwise (or when
// the call fails) the deterministic Indonesian mock (model "mock").
package openrouter

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
)

const (
	DefaultEndpoint = "https://openrouter.ai/api/v1/chat/completions"
	DefaultModel    = "anthropic/claude-sonnet-4.5"
	MockModel       = "mock"
	timeout         = 60 * time.Second
)

// Credentials come from integrations GetProviderCredentials(OPENROUTER):
// the secret is the API key, the public id an optional model override.
type Credentials struct {
	APIKey string
	Model  string
}

// CredentialsFunc fetches the current credentials (cached by the caller).
type CredentialsFunc func(ctx context.Context) (Credentials, error)

type Options struct {
	// Referer is sent as HTTP-Referer (APP_BASE_URL).
	Referer string
	// Endpoint overrides the API URL (tests).
	Endpoint   string
	HTTPClient *http.Client
	Logger     *slog.Logger
}

// Source picks the adapter per call so saving a key in the setup wizard
// switches to OpenRouter without a restart. It implements app.AIWriter.
type Source struct {
	creds CredentialsFunc
	opts  Options
}

func NewSource(creds CredentialsFunc, opts Options) *Source {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Source{creds: creds, opts: opts}
}

// real returns the OpenRouter adapter, nil when no key is configured or
// the credentials cannot be read (the mock is used then).
func (s *Source) real(ctx context.Context) *Real {
	creds, err := s.creds(ctx)
	if err != nil {
		s.opts.Logger.Warn("openrouter credentials unavailable, using mock", "err", err)
		return nil
	}
	if creds.APIKey == "" {
		return nil
	}
	return NewReal(creds, s.opts)
}

func (s *Source) fallback(what string, err error) {
	s.opts.Logger.Warn("openrouter failed, using mock", "what", what, "err", err)
}

func (s *Source) DailySummary(ctx context.Context, in domain.DailyRecapInput) (string, string, error) {
	if r := s.real(ctx); r != nil {
		text, err := r.DailySummary(ctx, in)
		if err == nil {
			return text, r.model, nil
		}
		s.fallback("daily", err)
	}
	return MockDailySummary(in), MockModel, nil
}

func (s *Source) MonthlyNarrative(ctx context.Context, in domain.MonthlyRecapInput) (string, string, error) {
	if r := s.real(ctx); r != nil {
		text, err := r.MonthlyNarrative(ctx, in)
		if err == nil {
			return text, r.model, nil
		}
		s.fallback("monthly", err)
	}
	return MockMonthlyNarrative(in), MockModel, nil
}

func (s *Source) SuggestImprovements(ctx context.Context, in domain.MonthlyRecapInput) ([]domain.ImprovementSuggestion, string, error) {
	if r := s.real(ctx); r != nil {
		list, err := r.SuggestImprovements(ctx, in)
		if err == nil {
			return list, r.model, nil
		}
		s.fallback("suggestions", err)
	}
	return MockSuggestions(in), MockModel, nil
}
