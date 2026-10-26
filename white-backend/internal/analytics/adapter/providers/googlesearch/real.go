package googlesearch

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/httpx"
	"github.com/syabanf/white-superapp/white-backend/pkg/providers/google"
)

const (
	providerName = "google"
	timeout      = 30 * time.Second
)

type Real struct {
	HTTP    *httpx.Client
	GSCBase string
	GA4Base string
}

func NewReal() *Real {
	return &Real{HTTP: httpx.New(timeout), GSCBase: google.GSCBase, GA4Base: google.GA4Base}
}

func (r *Real) post(ctx context.Context, rawURL, token string, body any, api string, out any) error {
	res, err := r.HTTP.PostJSON(ctx, rawURL, body, map[string]string{"Authorization": "Bearer " + token})
	if err != nil {
		return apperr.FromProvider(providerName, apperr.Network, fmt.Sprintf("Gangguan jaringan saat memanggil %s. Periksa koneksi dan coba lagi.", api), err)
	}
	if res.Status >= 400 {
		mapped := google.MapError(providerName, res.Status, res.Body)
		detail := mapped.Message
		mapped.Message = messageFor(mapped.Code, api)
		if detail != "" && !strings.HasPrefix(detail, "HTTP ") {
			mapped.Message += " (" + detail + ")"
		}
		return mapped
	}
	if err := res.JSON(out); err != nil {
		return apperr.FromProvider(providerName, apperr.Unknown, fmt.Sprintf("Respons %s tidak bisa dibaca.", api), err)
	}
	return nil
}

func messageFor(code apperr.Code, api string) string {
	switch code {
	case apperr.TokenExpired:
		return fmt.Sprintf("Token Google kedaluwarsa saat memanggil %s. Hubungkan ulang akun Google.", api)
	case apperr.Permission:
		return fmt.Sprintf("Akun Google tidak punya izin untuk %s. Periksa akses properti.", api)
	case apperr.RateLimit:
		return fmt.Sprintf("Batas kuota %s tercapai. Coba lagi beberapa menit.", api)
	case apperr.NotFound:
		return fmt.Sprintf("Properti tidak ditemukan di %s.", api)
	}
	return fmt.Sprintf("Terjadi kesalahan saat memanggil %s.", api)
}

type rawGsc struct {
	Rows []struct {
		Keys        []string `json:"keys"`
		Clicks      float64  `json:"clicks"`
		Impressions float64  `json:"impressions"`
		CTR         float64  `json:"ctr"`
		Position    float64  `json:"position"`
	} `json:"rows"`
}

func (r *Real) QuerySearchAnalytics(ctx context.Context, token string, req GscQuery) ([]GscRow, error) {
	rowLimit := req.RowLimit
	if rowLimit == 0 {
		rowLimit = 5000
	}
	body := map[string]any{"startDate": req.StartDate, "endDate": req.EndDate, "dimensions": req.Dimensions, "rowLimit": rowLimit, "startRow": req.StartRow}
	var out rawGsc
	// encodeURIComponent: ":" and "/" must be escaped too.
	u := r.GSCBase + "/sites/" + strings.ReplaceAll(url.QueryEscape(req.SiteURL), "+", "%20") + "/searchAnalytics/query"
	if err := r.post(ctx, u, token, body, "Google Search Console", &out); err != nil {
		return nil, err
	}
	rows := make([]GscRow, 0, len(out.Rows))
	for _, x := range out.Rows {
		keys := x.Keys
		if keys == nil {
			keys = []string{}
		}
		rows = append(rows, GscRow{Keys: keys, Clicks: x.Clicks, Impressions: x.Impressions, CTR: x.CTR * 100, Position: x.Position})
	}
	return rows, nil
}

type rawGa4 struct {
	Rows []struct {
		DimensionValues []struct {
			Value string `json:"value"`
		} `json:"dimensionValues"`
		MetricValues []struct {
			Value string `json:"value"`
		} `json:"metricValues"`
	} `json:"rows"`
}

func (r *Real) RunGa4Report(ctx context.Context, token string, req Ga4Request) ([]Ga4Row, error) {
	prop := Ga4PropertyName(req.PropertyID)
	limit := req.Limit
	if limit == 0 {
		limit = 10000
	}
	metrics := make([]map[string]string, 0, len(req.Metrics))
	for _, m := range req.Metrics {
		metrics = append(metrics, map[string]string{"name": m})
	}
	dims := make([]map[string]string, 0, len(req.Dimensions))
	for _, d := range req.Dimensions {
		dims = append(dims, map[string]string{"name": d})
	}
	body := map[string]any{
		"dateRanges": []map[string]string{{"startDate": req.StartDate, "endDate": req.EndDate}},
		"metrics":    metrics,
		"dimensions": dims,
		"limit":      limit,
	}
	var out rawGa4
	if err := r.post(ctx, r.GA4Base+"/"+prop+":runReport", token, body, "Google Analytics 4", &out); err != nil {
		return nil, err
	}
	rows := make([]Ga4Row, 0, len(out.Rows))
	for _, x := range out.Rows {
		row := Ga4Row{Dimensions: make([]string, 0, len(x.DimensionValues)), Metrics: make([]float64, 0, len(x.MetricValues))}
		for _, d := range x.DimensionValues {
			row.Dimensions = append(row.Dimensions, d.Value)
		}
		for _, m := range x.MetricValues {
			f, _ := strconv.ParseFloat(m.Value, 64)
			row.Metrics = append(row.Metrics, f)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// Ga4PropertyName adds the "properties/" prefix when missing.
func Ga4PropertyName(id string) string {
	if strings.HasPrefix(id, "properties/") {
		return id
	}
	return "properties/" + id
}
