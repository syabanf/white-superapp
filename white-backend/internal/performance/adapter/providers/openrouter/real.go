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

	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

const baseRules = `Kamu adalah asisten HR di sebuah digital agency Indonesia bernama WHITE.
Aturan wajib:
- Tulis dalam Bahasa Indonesia yang ringkas, hangat, dan profesional.
- Gunakan hanya fakta dan angka dari data JSON; jangan mengarang.
- Jangan menyebut AI, model bahasa, atau proses pembuatan teks ini.
- Tanpa tabel, tanpa HTML, tanpa salam pembuka dan penutup.`

const dailyPrompt = baseRules + `
Tugas: tulis rekap kerja harian satu karyawan dalam 2 sampai 4 kalimat: apa yang selesai, apa yang masih berjalan, kepatuhan logbook, dan rangkum catatan karyawan bila ada.`

const monthlyPrompt = baseRules + `
Tugas: tulis narasi rekap kerja bulanan (2 paragraf pendek). Paragraf 1: capaian utama dengan angka (tugas selesai vs rencana, ketepatan waktu, skor KPI, logbook, kehadiran). Paragraf 2: area yang perlu diperhatikan bulan depan.`

const suggestPrompt = baseRules + `
Tugas: usulkan maksimal 5 perbaikan konkret untuk bulan depan berdasarkan data rekap bulanan.
Balas HANYA dengan JSON array, tanpa teks lain, format: [{"title": "...", "description": "..."}].
title maksimal 80 karakter dan diawali kata kerja; description 1 sampai 2 kalimat yang menjelaskan alasan dan langkah.`

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

// payload renders the user turn: a context line and the JSON data.
func payload(header string, data any) (string, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(data); err != nil {
		return "", err
	}
	return header + "\n\nData (JSON):\n" + strings.TrimSpace(b.String()), nil
}

func (r *Real) chat(ctx context.Context, system, user string, maxTokens int) (string, error) {
	body, err := json.Marshal(chatRequest{Model: r.model, MaxTokens: maxTokens, Temperature: 0.4,
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
		detail := string(raw)
		if len(detail) > 300 {
			detail = detail[:300]
		}
		return "", providerErr(code, fmt.Errorf("OpenRouter mengembalikan status %d: %s", res.StatusCode, detail))
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

func (r *Real) DailySummary(ctx context.Context, in domain.DailyRecapInput) (string, error) {
	user, err := payload(fmt.Sprintf("Karyawan: %s\nTanggal: %s", in.EmployeeName, in.Date),
		map[string]any{"tugas": in.Items.Tasks, "logbook": in.Items.Logbook, "catatanKaryawan": in.Notes})
	if err != nil {
		return "", err
	}
	return r.chat(ctx, dailyPrompt, user, 400)
}

func monthlyData(in domain.MonthlyRecapInput) map[string]any {
	return map[string]any{"cakupan": in.Scope, "metrik": in.Metrics, "capaian": in.Accomplishments, "konteks": in.Signals}
}

func (r *Real) MonthlyNarrative(ctx context.Context, in domain.MonthlyRecapInput) (string, error) {
	user, err := payload(fmt.Sprintf("Subjek: %s (%s)\nPeriode: %s", in.SubjectName, in.DepartmentName, in.Period.Label()), monthlyData(in))
	if err != nil {
		return "", err
	}
	return r.chat(ctx, monthlyPrompt, user, 900)
}

// ParseSuggestions reads the JSON array, tolerating a fenced code block.
func ParseSuggestions(text string) ([]domain.ImprovementSuggestion, error) {
	t := strings.TrimSpace(text)
	if i := strings.Index(t, "["); i >= 0 {
		if j := strings.LastIndex(t, "]"); j > i {
			t = t[i : j+1]
		}
	}
	var out []domain.ImprovementSuggestion
	if err := json.Unmarshal([]byte(t), &out); err != nil {
		return nil, err
	}
	clean := out[:0]
	for _, s := range out {
		s.Title, s.Description = strings.TrimSpace(s.Title), strings.TrimSpace(s.Description)
		if s.Title != "" && len(clean) < domain.MaxSuggestions {
			clean = append(clean, s)
		}
	}
	if len(clean) == 0 {
		return nil, errors.New("tidak ada saran yang terbaca")
	}
	return clean, nil
}

func (r *Real) SuggestImprovements(ctx context.Context, in domain.MonthlyRecapInput) ([]domain.ImprovementSuggestion, error) {
	user, err := payload(fmt.Sprintf("Subjek: %s (%s)\nPeriode: %s", in.SubjectName, in.DepartmentName, in.Period.Label()), monthlyData(in))
	if err != nil {
		return nil, err
	}
	text, err := r.chat(ctx, suggestPrompt, user, 700)
	if err != nil {
		return nil, err
	}
	return ParseSuggestions(text)
}
