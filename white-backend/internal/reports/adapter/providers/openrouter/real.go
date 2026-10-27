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

	"github.com/syabanf/white-superapp/white-backend/internal/reports/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

const baseRules = `Kamu adalah kepala strategi (senior strategist) di sebuah digital agency Indonesia yang menulis analisa performa untuk klien.
Aturan wajib:
- Tulis dalam format markdown sederhana (heading "##", bullet "-", daftar bernomor, **bold**). Tanpa tabel, tanpa HTML.
- Gunakan angka-angka NYATA dari data JSON yang diberikan — jangan pernah mengarang angka.
- Nada: tajam, membumi, berorientasi aksi; seperti konsultan senior menulis untuk pemilik bisnis.
- Jangan pernah menyebut AI, model bahasa, otomasi, atau proses pembuatan analisa ini.
- Jangan menyapa pembaca dan jangan menambahkan penutup basa-basi.`

const sectionCore = `Struktur output WAJIB memakai heading berikut, dalam Bahasa Indonesia dan urutan ini:
## Ringkasan eksekutif
(1 paragraf padat: kondisi umum periode ini vs periode sebelumnya)
## Sorotan performa
(bullet — pencapaian terkuat dengan angka)
## Area perbaikan
(bullet — kelemahan atau risiko dengan angka)`

const sectionRecs = `## Rekomendasi 30 hari
(TEPAT 5 butir bernomor 1.–5., masing-masing satu aksi konkret yang bisa dieksekusi dalam 30 hari)`

// SystemPrompts are copied from lib/providers/openrouter/real.ts.
var SystemPrompts = map[string]string{
	domain.ModuleOverview: baseRules + `

Fokus: ringkasan lintas kanal (social media, SEO organik, dan Meta Ads) — hubungkan ketiganya menjadi satu narasi funnel: jangkauan → trafik → konversi.
` + sectionCore + "\n" + sectionRecs,
	domain.ModuleSocial: baseRules + `

Fokus: performa social media organik (Instagram/Facebook/TikTok) — pertumbuhan followers, engagement rate, jangkauan, dan pola konten yang bekerja (lihat topPosts).
` + sectionCore + "\n" + sectionRecs,
	domain.ModuleSeo: baseRules + `

Fokus: performa pencarian organik (Google Search Console) — klik, impresi, CTR, posisi rata-rata, kesehatan situs, dan kata kunci (topQueries).
` + sectionCore + `
## Peluang kata kunci
(bullet — kata kunci posisi 4–15 dengan impresi tinggi yang layak didorong ke top 3, sebutkan posisi & impresinya)
` + sectionRecs,
	domain.ModuleAds: baseRules + `

Fokus: performa Meta Ads — belanja, hasil, biaya per hasil, CTR/CPM/ROAS, dan perbandingan antar kampanye (campaigns).
` + sectionCore + `
## Alokasi anggaran
(bullet — usulan realokasi anggaran antar kampanye berdasarkan efisiensi biaya per hasil, dengan angka)
` + sectionRecs,
}

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

func (r *Real) Model() string { return r.model }

// UserMessage is the user turn: context lines, then the JSON payload.
func UserMessage(in domain.InsightInput) (string, error) {
	// JSON.stringify does not escape <, > and &.
	var data bytes.Buffer
	enc := json.NewEncoder(&data)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(in.Data); err != nil {
		return "", err
	}
	languageLine := "Tulis seluruh isi dalam Bahasa Indonesia."
	if in.Language == "en" {
		languageLine = "Write the body text in English, but KEEP the section headings exactly as specified (in Indonesian)."
	}
	return strings.Join([]string{
		"Klien: " + in.ClientName,
		fmt.Sprintf("Periode analisa: %s s.d. %s (%d hari), dibandingkan %s s.d. %s.", in.Range.From, in.Range.To, in.Range.Days, in.Range.PrevFrom, in.Range.PrevTo),
		"Catatan data: nilai *DeltaPct adalah perubahan % vs periode pembanding; engagementRate/ctr dalam persen; cpr/cpm dalam mata uang akun.",
		languageLine,
		"",
		"Data KPI (JSON):",
		strings.TrimSuffix(data.String(), "\n"),
	}, "\n"), nil
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

func (r *Real) Generate(ctx context.Context, in domain.InsightInput) (string, error) {
	user, err := UserMessage(in)
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(chatRequest{
		Model: r.model, MaxTokens: 2500, Temperature: 0.4,
		Messages: []chatMessage{{Role: "system", Content: SystemPrompts[in.Module]}, {Role: "user", Content: user}},
	})
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
	req.Header.Set("X-Title", "Mixist Analytics")
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
		case http.StatusNotFound:
			code = apperr.NotFound
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
