// Package openrouter is the chat completion provider behind app.AI. The
// real adapter calls OpenRouter with the key integrations returns; with no
// key the source answers Mock so each use case falls back to its own
// deterministic mock (same input, same output).
package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

const (
	DefaultEndpoint = "https://openrouter.ai/api/v1/chat/completions"
	DefaultModel    = "anthropic/claude-sonnet-4.5"
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
	Referer    string
	Endpoint   string
	HTTPClient *http.Client
}

// Source picks real or mock per call, so saving a key in the setup wizard
// takes effect without a restart.
type Source struct {
	creds CredentialsFunc
	opts  Options
}

func NewSource(creds CredentialsFunc, opts Options) *Source {
	if opts.Endpoint == "" {
		opts.Endpoint = DefaultEndpoint
	}
	if opts.Referer == "" {
		opts.Referer = "http://localhost:3000"
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: timeout}
	}
	return &Source{creds: creds, opts: opts}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string            `json:"model"`
	MaxTokens      int               `json:"max_tokens,omitempty"`
	Temperature    float64           `json:"temperature"`
	Messages       []chatMessage     `json:"messages"`
	ResponseFormat map[string]string `json:"response_format,omitempty"`
}

type chatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func providerErr(code apperr.Code, cause error) error {
	return apperr.FromProvider("openrouter", code, apperr.Message(code), cause)
}

// Complete runs one completion. JSON mode is retried once without
// response_format on HTTP 400, like the NüHabit OpenAI adapter.
func (s *Source) Complete(ctx context.Context, req app.AIRequest) (app.AIResult, error) {
	creds, err := s.creds(ctx)
	if err != nil {
		return app.AIResult{}, err
	}
	if creds.APIKey == "" {
		return app.AIResult{Model: app.MockModel, Mock: true}, nil
	}
	model := creds.Model
	if model == "" {
		model = DefaultModel
	}
	body := chatRequest{Model: model, MaxTokens: req.MaxTokens, Temperature: req.Temperature}
	for _, m := range req.Messages {
		body.Messages = append(body.Messages, chatMessage{Role: m.Role, Content: m.Content})
	}
	if req.JSON {
		body.ResponseFormat = map[string]string{"type": "json_object"}
	}
	content, status, err := s.post(ctx, creds.APIKey, body)
	if status == http.StatusBadRequest && req.JSON {
		body.ResponseFormat = nil
		content, _, err = s.post(ctx, creds.APIKey, body)
	}
	if err != nil {
		return app.AIResult{}, err
	}
	return app.AIResult{Content: content, Model: model}, nil
}

func (s *Source) post(ctx context.Context, key string, body chatRequest) (string, int, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return "", 0, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.opts.Endpoint, bytes.NewReader(raw))
	if err != nil {
		return "", 0, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+key)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("HTTP-Referer", s.opts.Referer)
	httpReq.Header.Set("X-Title", "WHITE Sales")
	res, err := s.opts.HTTPClient.Do(httpReq)
	if err != nil {
		return "", 0, providerErr(apperr.Network, fmt.Errorf("tidak dapat menghubungi OpenRouter: %w", err))
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return "", res.StatusCode, providerErr(apperr.Network, err)
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		code := apperr.Unknown
		switch res.StatusCode {
		case http.StatusUnauthorized:
			code = apperr.TokenExpired
		case http.StatusForbidden:
			code = apperr.Permission
		case http.StatusTooManyRequests:
			code = apperr.RateLimit
		}
		detail := string(data)
		if len(detail) > 300 {
			detail = detail[:300]
		}
		return "", res.StatusCode, providerErr(code, fmt.Errorf("OpenRouter status %d: %s", res.StatusCode, detail))
	}
	var out chatResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return "", res.StatusCode, providerErr(apperr.Unknown, err)
	}
	if len(out.Choices) == 0 || strings.TrimSpace(out.Choices[0].Message.Content) == "" {
		msg := "OpenRouter tidak mengembalikan konten."
		if out.Error != nil && out.Error.Message != "" {
			msg = out.Error.Message
		}
		return "", res.StatusCode, providerErr(apperr.Unknown, errors.New(msg))
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), res.StatusCode, nil
}
