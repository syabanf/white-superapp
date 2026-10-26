// Package ai drafts editorial plan items and writes content report
// summaries: OpenRouter chat completions when a key is configured in
// integrations, otherwise a deterministic template mock (model "mock").
package ai

import (
	"context"
	"net/http"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/content/app"
	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
)

const (
	DefaultEndpoint = "https://openrouter.ai/api/v1/chat/completions"
	DefaultModel    = "anthropic/claude-sonnet-4.5"
	MockModel       = "mock"
	timeout         = 120 * time.Second
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
}

// provider is what Real and Mock both implement.
type provider interface {
	model() string
	draftPlan(ctx context.Context, in app.PlanPrompt) ([]app.ItemDraft, error)
	summarize(ctx context.Context, data domain.ReportData) (string, error)
}

// Source picks the provider per call, so saving a key in the setup wizard
// switches to OpenRouter without a restart. It implements app.AI.
type Source struct {
	creds CredentialsFunc
	opts  Options
}

func NewSource(creds CredentialsFunc, opts Options) *Source { return &Source{creds: creds, opts: opts} }

func (s *Source) pick(ctx context.Context) (provider, error) {
	creds, err := s.creds(ctx)
	if err != nil {
		return nil, err
	}
	if creds.APIKey == "" {
		return Mock{}, nil
	}
	return NewReal(creds, s.opts), nil
}

func (s *Source) DraftPlan(ctx context.Context, in app.PlanPrompt) ([]app.ItemDraft, string, error) {
	p, err := s.pick(ctx)
	if err != nil {
		return nil, "", err
	}
	items, err := p.draftPlan(ctx, in)
	return items, p.model(), err
}

func (s *Source) SummarizeReport(ctx context.Context, data domain.ReportData) (string, string, error) {
	p, err := s.pick(ctx)
	if err != nil {
		return "", "", err
	}
	md, err := p.summarize(ctx, data)
	return md, p.model(), err
}

// slots returns how many items each platform gets in the month: posts per
// week times the month's weeks, rounded. An empty frequency means three
// Instagram posts a week.
func slots(month domain.Month, freq []domain.Frequency) []domain.Frequency {
	if len(freq) == 0 {
		freq = []domain.Frequency{{Platform: domain.PlatformInstagram, PerWeek: 3}}
	}
	weeks := float64(month.Last().Day()) / 7
	out := make([]domain.Frequency, 0, len(freq))
	for _, f := range freq {
		n := int32(float64(f.PerWeek)*weeks + 0.5)
		if n > 0 {
			out = append(out, domain.Frequency{Platform: f.Platform, PerWeek: n})
		}
	}
	return out
}
