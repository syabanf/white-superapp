// Package openrouter runs the recruitment AI (interview questions and
// summary, candidate analysis): the OpenRouter chat completion when a key
// is configured in integrations, otherwise (or when the call fails) the
// deterministic Indonesian mock (model "mock").
package openrouter

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
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
// switches to OpenRouter without a restart. It implements app.AI.
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
	if s.creds == nil {
		return nil
	}
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

// InterviewStep is nil without a key or on failure; the interview then
// follows the topic order.
func (s *Source) InterviewStep(ctx context.Context, t domain.Transcript) *domain.AIStep {
	r := s.real(ctx)
	if r == nil {
		return nil
	}
	step, err := r.InterviewStep(ctx, t)
	if err != nil {
		s.fallback("interview-step", err)
		return nil
	}
	return step
}

func (s *Source) SummarizeInterview(ctx context.Context, t domain.Transcript) (domain.InterviewSummary, string, error) {
	if r := s.real(ctx); r != nil {
		sum, err := r.SummarizeInterview(ctx, t)
		if err == nil {
			return sum, r.model, nil
		}
		s.fallback("interview-summary", err)
	}
	return MockSummary(t), MockModel, nil
}

func (s *Source) AnalyzeCandidate(ctx context.Context, in domain.AnalysisInput) (domain.CandidateAnalysis, string, error) {
	if r := s.real(ctx); r != nil {
		a, err := r.AnalyzeCandidate(ctx, in)
		if err == nil {
			return a, r.model, nil
		}
		s.fallback("candidate-analysis", err)
	}
	return MockAnalysis(in), MockModel, nil
}
