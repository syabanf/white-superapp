// Package openrouter writes the CFO memo: the real OpenRouter chat
// completion when a key is configured, otherwise a deterministic template
// (model "mock"). Credentials come from integrations
// GetProviderCredentials(OPENROUTER), picked per call so saving a key
// switches to the real adapter without a restart.
package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

const (
	DefaultEndpoint = "https://openrouter.ai/api/v1/chat/completions"
	DefaultModel    = "anthropic/claude-sonnet-4.5"
	MockModel       = "mock"
)

// Credentials: the secret is the API key, the public id an optional model.
type Credentials struct {
	APIKey string
	Model  string
}

type CredentialsFunc func(ctx context.Context) (Credentials, error)

type Options struct {
	Referer    string
	Endpoint   string
	HTTPClient *http.Client
}

// Source implements app.InsightGenerator.
type Source struct {
	creds CredentialsFunc
	opts  Options
}

func NewSource(creds CredentialsFunc, opts Options) *Source { return &Source{creds: creds, opts: opts} }

func (s *Source) Generate(ctx context.Context, in app.InsightInput) (string, string, error) {
	creds, err := s.creds(ctx)
	if err != nil || creds.APIKey == "" {
		// No key (or integrations unreachable): the deterministic memo.
		return Mock(in), MockModel, nil
	}
	model := creds.Model
	if model == "" {
		model = DefaultModel
	}
	content, err := complete(ctx, s.opts, creds.APIKey, model, in)
	return content, model, err
}

// SystemPrompt is the CFO brief; headings are fixed in Indonesian.
const SystemPrompt = `Kamu adalah CFO berpengalaman di sebuah digital agency Indonesia yang menulis memo keuangan untuk direksi.
Aturan wajib:
- Tulis dalam markdown sederhana (heading "##", bullet "-", daftar bernomor, **bold**). Tanpa tabel, tanpa HTML.
- Gunakan hanya angka NYATA dari data JSON. Jangan mengarang angka. Nominal dalam Rupiah.
- Nada: lugas, berbasis data, fokus pada arus kas dan kesehatan bisnis.
- Jangan menyebut AI, model bahasa, atau proses pembuatan memo ini. Tanpa salam pembuka atau penutup.
Struktur output WAJIB memakai heading berikut, dalam urutan ini:
## Ringkasan eksekutif
(1 paragraf: kondisi keuangan periode ini dibanding periode pembanding)
## Kesehatan keuangan
(bullet: profitabilitas, likuiditas, leverage, efisiensi penagihan, dengan angka rasio)
## Risiko
(bullet: risiko terbesar, misalnya runway, piutang menumpuk, beban yang naik tajam)
## Rekomendasi 30 hari
(TEPAT 5 butir bernomor, masing-masing aksi konkret yang bisa dieksekusi dalam 30 hari)`

// UserMessage carries the period and the JSON payload.
func UserMessage(in app.InsightInput) (string, error) {
	var data bytes.Buffer
	enc := json.NewEncoder(&data)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(in.Data); err != nil {
		return "", err
	}
	lang := "Tulis seluruh isi dalam Bahasa Indonesia."
	if in.Language == "en" {
		lang = "Write the body in English, but KEEP the section headings exactly as specified (in Indonesian)."
	}
	return strings.Join([]string{
		fmt.Sprintf("Periode analisa: %s s.d. %s.", in.From, in.To),
		"Catatan data: margin dan pertumbuhan dalam persen; rasio dalam kali; DSO/DPO dalam hari; runway dalam bulan; nominal dalam Rupiah.",
		lang, "", "Data keuangan (JSON):", strings.TrimSuffix(data.String(), "\n"),
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

func providerErr(code apperr.Code, err error) error {
	return apperr.FromProvider("openrouter", code, err.Error(), err)
}

func complete(ctx context.Context, opts Options, apiKey, model string, in app.InsightInput) (string, error) {
	user, err := UserMessage(in)
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(chatRequest{Model: model, MaxTokens: 2000, Temperature: 0.4, Messages: []chatMessage{{"system", SystemPrompt}, {"user", user}}})
	if err != nil {
		return "", err
	}
	endpoint, referer, client := opts.Endpoint, opts.Referer, opts.HTTPClient
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	if referer == "" {
		referer = "http://localhost:3000"
	}
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", referer)
	req.Header.Set("X-Title", "Mixist ERP")
	res, err := client.Do(req)
	if err != nil {
		return "", providerErr(apperr.Network, fmt.Errorf("tidak dapat menghubungi OpenRouter (timeout/jaringan): %w", err))
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
	if len(out.Choices) > 0 {
		if content := strings.TrimSpace(out.Choices[0].Message.Content); content != "" {
			return content, nil
		}
	}
	msg := "OpenRouter tidak mengembalikan konten."
	if out.Error != nil && out.Error.Message != "" {
		msg = out.Error.Message
	}
	return "", providerErr(apperr.Unknown, errors.New(msg))
}

// ─── Mock ───────────────────────────────────────────────────

func lookup(m map[string]any, path ...string) (float64, bool) {
	var cur any = m
	for _, p := range path {
		mm, ok := cur.(map[string]any)
		if !ok {
			return 0, false
		}
		cur = mm[p]
	}
	f, ok := cur.(float64)
	return f, ok
}

func rp(v float64) string { return "Rp" + domain.FormatRupiah(domain.FromFloat(v)) }

func ratio(m map[string]any, key, unit string) string {
	v, ok := lookup(m, "rasio", key, "value")
	if !ok {
		return "tidak tersedia"
	}
	switch unit {
	case "pct":
		return fmt.Sprintf("%.1f%%", v)
	case "days":
		return fmt.Sprintf("%.0f hari", v)
	case "months":
		return fmt.Sprintf("%.1f bulan", v)
	}
	return fmt.Sprintf("%.2fx", v)
}

// Mock fills a template with the payload numbers: same input, same output.
func Mock(in app.InsightInput) string {
	d := in.Data
	rev, _ := lookup(d, "ringkasan", "pendapatan")
	net, _ := lookup(d, "ringkasan", "labaBersih")
	cash, _ := lookup(d, "ringkasan", "kas")
	ar, _ := lookup(d, "ringkasan", "piutang")
	prevRev, hasPrev := lookup(d, "pembanding", "ringkasan", "pendapatan")
	trend := "tanpa data pembanding"
	if hasPrev && prevRev != 0 {
		pct := (rev - prevRev) / math.Abs(prevRev) * 100
		switch {
		case pct > 1:
			trend = fmt.Sprintf("naik %.1f%% dari periode sebelumnya", pct)
		case pct < -1:
			trend = fmt.Sprintf("turun %.1f%% dari periode sebelumnya", -pct)
		default:
			trend = "relatif stabil dibanding periode sebelumnya"
		}
	}
	result := "laba bersih " + rp(net)
	if net < 0 {
		result = "rugi bersih " + rp(-net)
	}
	runway, hasRunway := lookup(d, "rasio", "runway_months", "value")
	runwayRisk := "- Kas tidak tergerus pada periode ini, runway belum menjadi isu."
	if hasRunway {
		runwayRisk = fmt.Sprintf("- Dengan burn rate saat ini, runway kas sekitar %.1f bulan.", runway)
		if runway < 6 {
			runwayRisk += " Ini di bawah batas aman 6 bulan."
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## Ringkasan eksekutif\nPeriode %s s.d. %s mencatat pendapatan %s, %s, dengan %s. Saldo kas dan bank akhir periode %s.\n\n", in.From, in.To, rp(rev), trend, result, rp(cash))
	fmt.Fprintf(&b, "## Kesehatan keuangan\n- Profitabilitas: gross margin %s, operating margin %s, net margin %s.\n", ratio(d, "gross_margin", "pct"), ratio(d, "operating_margin", "pct"), ratio(d, "net_margin", "pct"))
	fmt.Fprintf(&b, "- Likuiditas: current ratio %s, quick ratio %s, cash ratio %s.\n", ratio(d, "current_ratio", "ratio"), ratio(d, "quick_ratio", "ratio"), ratio(d, "cash_ratio", "ratio"))
	fmt.Fprintf(&b, "- Leverage: debt to equity %s.\n", ratio(d, "debt_to_equity", "ratio"))
	fmt.Fprintf(&b, "- Efisiensi: DSO %s, DPO %s, piutang berjalan %s.\n\n", ratio(d, "dso", "days"), ratio(d, "dpo", "days"), rp(ar))
	fmt.Fprintf(&b, "## Risiko\n%s\n- Piutang %s perlu dipantau agar DSO tidak memanjang.\n", runwayRisk, rp(ar))
	if top, ok := d["bebanTerbesar"].([]map[string]any); ok && len(top) > 0 {
		if v, ok := top[0]["nilai"].(float64); ok {
			fmt.Fprintf(&b, "- Beban terbesar adalah %v sebesar %s.\n", top[0]["akun"], rp(v))
		}
	}
	b.WriteString("\n## Rekomendasi 30 hari\n")
	b.WriteString("1. Tagih piutang yang lewat jatuh tempo dan kirim pengingat otomatis ke klien.\n")
	b.WriteString("2. Tinjau tiga pos beban terbesar dan tetapkan batas anggaran bulanan.\n")
	b.WriteString("3. Jadwalkan pembayaran vendor mendekati jatuh tempo untuk menjaga kas.\n")
	b.WriteString("4. Perbarui proyeksi arus kas mingguan untuk 13 minggu ke depan.\n")
	b.WriteString("5. Evaluasi harga retainer klien dengan margin terendah.\n")
	return b.String()
}
