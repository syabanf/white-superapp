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

	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

type Real struct {
	apiKey   string
	model    string
	referer  string
	endpoint string
	client   *http.Client
}

func NewReal(creds Credentials, opts Options) *Real {
	r := &Real{apiKey: creds.APIKey, model: creds.Model, referer: opts.Referer, endpoint: opts.Endpoint, client: opts.HTTPClient}
	if r.model == "" {
		r.model = DefaultModel
	}
	if r.referer == "" {
		r.referer = "http://localhost:3000"
	}
	if r.endpoint == "" {
		r.endpoint = DefaultEndpoint
	}
	if r.client == nil {
		r.client = &http.Client{Timeout: timeout}
	}
	return r
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	Messages    []chatMessage `json:"messages"`
}

type chatResponse struct {
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

func (r *Real) chat(ctx context.Context, system, user string, temperature float64, maxTokens int) (string, error) {
	body, err := json.Marshal(chatRequest{Model: r.model, MaxTokens: maxTokens, Temperature: temperature,
		Messages: []chatMessage{{Role: "system", Content: system}, {Role: "user", Content: user}}})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", r.referer)
	req.Header.Set("X-Title", "WHITE ERP")
	res, err := r.client.Do(req)
	if err != nil {
		return "", providerErr(apperr.Network, fmt.Errorf("tidak dapat menghubungi OpenRouter: %w", err))
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return "", providerErr(apperr.Network, err)
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
		return "", providerErr(code, fmt.Errorf("OpenRouter mengembalikan status %d: %s", res.StatusCode, domain.JSSlice(string(raw), 300)))
	}
	var out chatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", providerErr(apperr.Unknown, fmt.Errorf("respons OpenRouter tidak dapat dibaca: %w", err))
	}
	if len(out.Choices) == 0 || strings.TrimSpace(out.Choices[0].Message.Content) == "" {
		m := "OpenRouter tidak mengembalikan konten."
		if out.Error != nil && out.Error.Message != "" {
			m = out.Error.Message
		}
		return "", providerErr(apperr.Unknown, errors.New(m))
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}

// ParseJSONObject reads the first JSON object of a reply, tolerating a
// fenced code block or surrounding prose.
func ParseJSONObject(text string) (map[string]any, error) {
	t := strings.TrimSpace(text)
	if i := strings.Index(t, "{"); i >= 0 {
		if j := strings.LastIndex(t, "}"); j > i {
			t = t[i : j+1]
		}
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(t), &out); err != nil {
		return nil, err
	}
	if out == nil {
		return nil, errors.New("respons bukan objek JSON")
	}
	return out, nil
}

// InterviewStep is generateNextInterviewQuestion's call (temperature 0.4).
func (r *Real) InterviewStep(ctx context.Context, t domain.Transcript) (*domain.AIStep, error) {
	text, err := r.chat(ctx, domain.QuestionPrompt, domain.InterviewStepPrompt(t.CandidateName, t.PositionTitle, t.MaxQuestions, t.Turns), 0.4, 300)
	if err != nil {
		return nil, err
	}
	raw, err := ParseJSONObject(text)
	if err != nil {
		return nil, err
	}
	step := &domain.AIStep{}
	step.Action, _ = raw["action"].(string)
	if s, ok := raw["topic"].(string); ok {
		step.Topic = &s
	}
	if s, ok := raw["question"].(string); ok {
		step.Question = &s
	}
	return step, nil
}

// SummarizeInterview is summarizeInterview with its normalization.
func (r *Real) SummarizeInterview(ctx context.Context, t domain.Transcript) (domain.InterviewSummary, error) {
	text, err := r.chat(ctx, domain.SummaryPrompt, domain.SummaryInput(t.CandidateName, t.PositionTitle, t.Turns), 0.4, 1200)
	if err != nil {
		return domain.InterviewSummary{}, err
	}
	raw, err := ParseJSONObject(text)
	if err != nil {
		return domain.InterviewSummary{}, err
	}
	return domain.NormalizeSummary(raw), nil
}

// AnalyzeCandidate is analyzeCvWithDeepseek over the candidate profile
// (temperature 0.2).
func (r *Real) AnalyzeCandidate(ctx context.Context, in domain.AnalysisInput) (domain.CandidateAnalysis, error) {
	job := "DESKRIPSI PEKERJAAN YANG DILAMAR: (tidak tersedia)"
	if in.JobContext != "" {
		job = "DESKRIPSI PEKERJAAN YANG DILAMAR:\n" + in.JobContext
	}
	text, err := r.chat(ctx, domain.AnalysisPrompt, job+"\n\nPROFIL KANDIDAT:\n"+domain.JSSlice(in.Profile, 12_000), 0.2, 900)
	if err != nil {
		return domain.CandidateAnalysis{}, err
	}
	raw, err := ParseJSONObject(text)
	if err != nil {
		return domain.CandidateAnalysis{}, errors.New("respons AI bukan JSON valid")
	}
	return domain.NormalizeAnalysis(raw), nil
}
