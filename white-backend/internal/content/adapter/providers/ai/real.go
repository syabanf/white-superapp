package ai

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

	"github.com/syabanf/white-superapp/white-backend/internal/content/app"
	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

const planSystem = `Kamu adalah content strategist senior di sebuah digital agency Indonesia yang menyusun editorial plan (EP) bulanan untuk klien.
Aturan wajib:
- Jawab HANYA dengan satu objek JSON valid, tanpa teks lain dan tanpa blok kode.
- Bentuk: {"items":[{"date":"YYYY-MM-DD","platform":"INSTAGRAM|FACEBOOK|TIKTOK","format":"FEED|CAROUSEL|REEL|STORY","pillar":"...","title":"...","hook":"...","caption":"...","hashtags":["#..."],"visualBrief":"..."}]}
- Semua tanggal berada di bulan yang diminta dan tersebar merata.
- Jumlah item per platform mengikuti kuota yang diberikan. TikTok hanya REEL. STORY tanpa caption panjang.
- Gunakan pilar konten dari brief secara bergiliran. Tulis dalam Bahasa Indonesia dengan tone dari brief.
- Caption maksimal 2.200 karakter, hashtag maksimal 10 per item, judul maksimal 100 karakter.
- Pelajari pola konten terbaik klien (topPosts) bila ada, tanpa menyalin captionnya.`

const reportSystem = `Kamu adalah kepala tim konten di sebuah digital agency Indonesia yang menulis evaluasi bulanan untuk klien.
Aturan wajib:
- Tulis dalam markdown sederhana (heading "##", bullet "-", daftar bernomor, **bold**). Tanpa tabel, tanpa HTML.
- Gunakan angka NYATA dari data JSON; jangan mengarang angka. Bila engagementAvailable false, katakan data engagement belum tersedia.
- Jangan menyebut AI atau proses pembuatan analisa ini. Jangan menyapa pembaca.
Struktur WAJIB:
## Ringkasan
(1 paragraf: rencana vs tayang, ketepatan waktu, revisi)
## Sorotan
(bullet: konten terbaik dan terlemah dengan angka)
## Rekomendasi
(TEPAT 5 butir bernomor, aksi konkret untuk bulan depan)`

type Real struct {
	apiKey   string
	modelID  string
	referer  string
	endpoint string
	client   *http.Client
}

func NewReal(creds Credentials, opts Options) *Real {
	r := &Real{apiKey: creds.APIKey, modelID: creds.Model, referer: opts.Referer, endpoint: opts.Endpoint, client: opts.HTTPClient}
	if r.modelID == "" {
		r.modelID = DefaultModel
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

func (r *Real) model() string { return r.modelID }

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

func jsonText(v any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

func (r *Real) complete(ctx context.Context, system, user string, maxTokens int) (string, error) {
	body, err := json.Marshal(chatRequest{Model: r.modelID, MaxTokens: maxTokens, Temperature: 0.5, Messages: []chatMessage{{Role: "system", Content: system}, {Role: "user", Content: user}}})
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
	req.Header.Set("X-Title", "WHITE Content")
	res, err := r.client.Do(req)
	if err != nil {
		return "", providerErr(apperr.Network, fmt.Errorf("tidak dapat menghubungi OpenRouter (timeout/jaringan): %w", err))
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return "", providerErr(apperr.Network, err)
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		detail := string(raw)
		if len(detail) > 300 {
			detail = detail[:300]
		}
		code := apperr.Unknown
		switch res.StatusCode {
		case http.StatusUnauthorized:
			code = apperr.TokenExpired
		case http.StatusForbidden:
			code = apperr.Permission
		case http.StatusTooManyRequests:
			code = apperr.RateLimit
		}
		return "", providerErr(code, fmt.Errorf("OpenRouter mengembalikan status %d: %s", res.StatusCode, detail))
	}
	var out chatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", providerErr(apperr.Unknown, fmt.Errorf("respons OpenRouter tidak dapat dibaca: %w", err))
	}
	var content string
	if len(out.Choices) > 0 {
		content = strings.TrimSpace(out.Choices[0].Message.Content)
	}
	if content == "" {
		msg := "OpenRouter tidak mengembalikan konten."
		if out.Error != nil && out.Error.Message != "" {
			msg = out.Error.Message
		}
		return "", providerErr(apperr.Unknown, errors.New(msg))
	}
	return content, nil
}

type draftJSON struct {
	Items []struct {
		Date        string   `json:"date"`
		Platform    string   `json:"platform"`
		Format      string   `json:"format"`
		Pillar      string   `json:"pillar"`
		Title       string   `json:"title"`
		Hook        string   `json:"hook"`
		Caption     string   `json:"caption"`
		Hashtags    []string `json:"hashtags"`
		VisualBrief string   `json:"visualBrief"`
	} `json:"items"`
}

// ParseDraft reads the model's JSON (code fences and surrounding text are
// tolerated) and drops items with an invalid date, platform or format.
func ParseDraft(content string, month domain.Month) ([]app.ItemDraft, error) {
	start, end := strings.Index(content, "{"), strings.LastIndex(content, "}")
	if start < 0 || end <= start {
		return nil, providerErr(apperr.Unknown, errors.New("draft EP bukan JSON"))
	}
	var parsed draftJSON
	if err := json.Unmarshal([]byte(content[start:end+1]), &parsed); err != nil {
		return nil, providerErr(apperr.Unknown, fmt.Errorf("draft EP tidak dapat dibaca: %w", err))
	}
	out := make([]app.ItemDraft, 0, len(parsed.Items))
	for _, it := range parsed.Items {
		day, err := time.Parse(time.DateOnly, strings.TrimSpace(it.Date))
		platform, format := domain.Platform(strings.ToUpper(it.Platform)), domain.Format(strings.ToUpper(it.Format))
		if err != nil || !month.Contains(day) || !domain.IsPlatform(platform) || !domain.IsFormat(format) {
			continue
		}
		out = append(out, app.ItemDraft{PlannedDate: day, Platform: platform, Format: format, Pillar: it.Pillar, Title: it.Title, Hook: it.Hook, Caption: it.Caption, Hashtags: it.Hashtags, VisualBrief: it.VisualBrief})
	}
	return out, nil
}

func (r *Real) draftPlan(ctx context.Context, in app.PlanPrompt) ([]app.ItemDraft, error) {
	quota := map[string]int32{}
	for _, s := range slots(in.Month, in.Brief.Frequency) {
		quota[string(s.Platform)] = s.PerWeek
	}
	payload, err := jsonText(map[string]any{
		"client": in.ClientName, "month": in.Month.String(), "goals": in.Brief.Goals, "pillars": in.Brief.Pillars,
		"tone": in.Brief.Tone, "notes": in.Brief.Notes, "itemsPerPlatform": quota, "topPosts": in.TopPosts,
	})
	if err != nil {
		return nil, err
	}
	content, err := r.complete(ctx, planSystem, "Susun editorial plan untuk "+in.Month.Label()+".\n\nBrief (JSON):\n"+payload, 8000)
	if err != nil {
		return nil, err
	}
	return ParseDraft(content, in.Month)
}

func (r *Real) summarize(ctx context.Context, data domain.ReportData) (string, error) {
	payload, err := jsonText(data)
	if err != nil {
		return "", err
	}
	return r.complete(ctx, reportSystem, "Data laporan konten (JSON):\n"+payload, 2500)
}
