// Package pagespeed calls PageSpeed Insights v5. There is no mock: without
// an API key the call is keyless and quota limited.
package pagespeed

import (
	"context"
	"encoding/json"
	"math"
	"net/url"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/httpx"
)

const (
	providerName    = "pagespeed"
	DefaultEndpoint = "https://www.googleapis.com/pagespeedonline/v5/runPagespeed"
	defaultTimeout  = 60 * time.Second
)

var categories = []string{"performance", "seo", "accessibility", "best-practices"}

type Metrics struct {
	LcpMs        *float64
	InpMs        *float64
	Cls          *float64
	FcpMs        *float64
	TtfbMs       *float64
	TbtMs        *float64
	SpeedIndexMs *float64
}

type Scores struct {
	Performance   int
	Seo           int
	Accessibility int
	BestPractices int
}

type Result struct {
	Strategy string
	FinalURL string
	Scores   Scores
	Metrics  Metrics
	// Raw keeps category scores and the metric audits for seo_audits.raw.
	Raw []byte
}

// metricAuditIDs per metric; the first present audit wins.
var metricAuditIDs = map[string][]string{
	"lcpMs":        {"largest-contentful-paint"},
	"inpMs":        {"interaction-to-next-paint", "experimental-interaction-to-next-paint"},
	"cls":          {"cumulative-layout-shift"},
	"fcpMs":        {"first-contentful-paint"},
	"ttfbMs":       {"server-response-time"},
	"tbtMs":        {"total-blocking-time"},
	"speedIndexMs": {"speed-index"},
}

type rawAudit struct {
	NumericValue *float64 `json:"numericValue,omitempty"`
	DisplayValue *string  `json:"displayValue,omitempty"`
}

type rawResponse struct {
	LighthouseResult *struct {
		FinalDisplayedURL *string                             `json:"finalDisplayedUrl"`
		FinalURL          *string                             `json:"finalUrl"`
		Categories        map[string]struct{ Score *float64 } `json:"categories"`
		Audits            map[string]rawAudit                 `json:"audits"`
	} `json:"lighthouseResult"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type Client struct {
	Endpoint string
	APIKey   string
	HTTP     *httpx.Client
}

func New(apiKey string) *Client {
	return &Client{Endpoint: DefaultEndpoint, APIKey: apiKey, HTTP: httpx.New(defaultTimeout)}
}

// Run audits one URL with the given strategy ("mobile" or "desktop").
func (c *Client) Run(ctx context.Context, target, strategy string) (*Result, error) {
	q := url.Values{"url": {target}, "strategy": {strategy}}
	for _, cat := range categories {
		q.Add("category", cat)
	}
	if c.APIKey != "" {
		q.Set("key", c.APIKey)
	}
	res, err := c.HTTP.Get(ctx, httpx.WithQuery(c.Endpoint, q), nil)
	if err != nil {
		msg := "Gangguan jaringan saat memanggil PageSpeed Insights."
		if httpx.IsTimeout(err) {
			msg = "PageSpeed Insights tidak merespons dalam 60 detik. Coba lagi."
		}
		return nil, apperr.FromProvider(providerName, apperr.Network, msg, err)
	}
	var raw rawResponse
	if err := json.Unmarshal(res.Body, &raw); err != nil {
		return nil, apperr.FromProvider(providerName, apperr.Unknown, "Respons PageSpeed Insights tidak valid.", err)
	}
	if res.Status >= 400 || raw.LighthouseResult == nil {
		code := apperr.Unknown
		base := "PageSpeed Insights gagal menganalisis URL ini."
		switch res.Status {
		case 429:
			code, base = apperr.RateLimit, "Kuota PageSpeed Insights tercapai. Coba lagi beberapa menit."
		case 403:
			code, base = apperr.Permission, "Akses PageSpeed Insights ditolak. Periksa PAGESPEED_API_KEY."
		case 404:
			code = apperr.NotFound
		}
		if raw.Error != nil && raw.Error.Message != "" {
			base += " (" + raw.Error.Message + ")"
		}
		return nil, apperr.FromProvider(providerName, code, base, nil)
	}
	lh := raw.LighthouseResult
	cat := func(id string) int {
		if c, ok := lh.Categories[id]; ok && c.Score != nil {
			return int(math.Round(*c.Score * 100))
		}
		return 0
	}
	metric := func(key string) *float64 {
		for _, id := range metricAuditIDs[key] {
			if a, ok := lh.Audits[id]; ok && a.NumericValue != nil && !math.IsNaN(*a.NumericValue) && !math.IsInf(*a.NumericValue, 0) {
				v := *a.NumericValue
				return &v
			}
		}
		return nil
	}
	rawCats := map[string]int{}
	for _, id := range categories {
		rawCats[id] = cat(id)
	}
	rawAudits := map[string]rawAudit{}
	for _, idList := range metricAuditIDs {
		for _, id := range idList {
			if a, ok := lh.Audits[id]; ok {
				rawAudits[id] = a
			}
		}
	}
	rawJSON, _ := json.Marshal(map[string]any{"categories": rawCats, "audits": rawAudits})
	final := target
	if lh.FinalDisplayedURL != nil {
		final = *lh.FinalDisplayedURL
	} else if lh.FinalURL != nil {
		final = *lh.FinalURL
	}
	return &Result{
		Strategy: strategy, FinalURL: final,
		Scores:  Scores{Performance: cat("performance"), Seo: cat("seo"), Accessibility: cat("accessibility"), BestPractices: cat("best-practices")},
		Metrics: Metrics{LcpMs: metric("lcpMs"), InpMs: metric("inpMs"), Cls: metric("cls"), FcpMs: metric("fcpMs"), TtfbMs: metric("ttfbMs"), TbtMs: metric("tbtMs"), SpeedIndexMs: metric("speedIndexMs")},
		Raw:     rawJSON,
	}, nil
}
