package seodata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/httpx"
)

const (
	ApifyBase       = "https://api.apify.com/v2"
	providerApify   = "apify"
	serpActorKey    = "serp"
	serpTimeoutSecs = 180
)

// DefaultActors are the actor ids per job, overridable through APIFY_ACTORS.
var DefaultActors = map[string]string{
	"serp":             "apify/google-search-scraper",
	"instagramProfile": "apify/instagram-profile-scraper",
	"instagramPosts":   "apify/instagram-scraper",
	"tiktok":           "clockworks/tiktok-scraper",
	"facebookPage":     "apify/facebook-pages-scraper",
	"facebookPosts":    "apify/facebook-posts-scraper",
}

var actorIDRe = regexp.MustCompile(`^[\w.-]+/[\w.-]+$`)

// ParseActorOverrides merges `{"serp":"owner/name"}` into the defaults.
// Unknown keys and invalid values are ignored; invalid JSON keeps the defaults.
func ParseActorOverrides(raw string) map[string]string {
	out := map[string]string{}
	for k, v := range DefaultActors {
		out[k] = v
	}
	if strings.TrimSpace(raw) == "" {
		return out
	}
	var o map[string]any
	if err := json.Unmarshal([]byte(raw), &o); err != nil {
		return out
	}
	for k := range DefaultActors {
		if v, ok := o[k].(string); ok && actorIDRe.MatchString(v) {
			out[k] = v
		}
	}
	return out
}

// ActorPath turns owner/name into the owner~name REST form.
func ActorPath(id string) string { return strings.Replace(id, "/", "~", 1) }

var locationCountry = map[int32]string{2360: "id", 2840: "us", 2702: "sg", 2458: "my", 2826: "gb", 2036: "au", 2392: "jp", 2356: "in"}

// CountryForLocation maps a DataForSEO location code to an ISO country; unknown is "id".
func CountryForLocation(locationCode int32) string {
	if c, ok := locationCountry[locationCode]; ok {
		return c
	}
	return "id"
}

// HostOf is the lowercase hostname without a leading www., "" on bad input.
func HostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(wwwRe.ReplaceAllString(u.Hostname(), ""))
}

func DomainMatches(host, target string) bool {
	t := strings.ToLower(wwwRe.ReplaceAllString(target, ""))
	return host == t || strings.HasSuffix(host, "."+t)
}

// ApifySerp runs the Google Search Scraper for rank positions.
type ApifySerp struct {
	http   *httpx.Client
	base   string
	token  string
	actors map[string]string
}

func NewApifySerp(token, actorsJSON, base string) *ApifySerp {
	if base == "" {
		base = ApifyBase
	}
	return &ApifySerp{
		http:   &httpx.Client{HTTP: &http.Client{}, UserAgent: "WHITEAnalytics/1.0"},
		base:   strings.TrimRight(base, "/"),
		token:  token,
		actors: ParseActorOverrides(actorsJSON),
	}
}

func apifyStatusError(status int, body string) error {
	switch status {
	case 401, 403:
		return apperr.FromProvider(providerApify, apperr.TokenExpired, "Token Apify ditolak.", nil)
	case 402:
		e := apperr.FromProvider(providerApify, apperr.Permission, "Kuota/kredit Apify habis (402).", nil)
		e.RequiresReconnect = false
		return e
	case 404:
		return apperr.FromProvider(providerApify, apperr.NotFound, "Actor Apify tidak ditemukan. Periksa override actor.", nil)
	case 408:
		return apperr.FromProvider(providerApify, apperr.Network, "Run Apify melebihi batas waktu sinkron (408).", nil)
	case 429:
		return apperr.FromProvider(providerApify, apperr.RateLimit, "Terlalu banyak permintaan ke Apify (429).", nil)
	}
	if len(body) > 200 {
		body = body[:200]
	}
	return apperr.FromProvider(providerApify, apperr.Unknown, fmt.Sprintf("Apify menjawab %d: %s", status, body), nil)
}

// RunActor calls run-sync-get-dataset-items and returns the dataset items.
// The HTTP deadline is timeoutSecs + 15 s; maxItems 0 means no limit.
func (a *ApifySerp) RunActor(ctx context.Context, actorID string, input any, timeoutSecs, maxItems int) ([]json.RawMessage, error) {
	if a.token == "" {
		e := apperr.FromProvider(providerApify, apperr.Permission, "APIFY_TOKEN belum diatur.", nil)
		e.RequiresReconnect = false
		return nil, e
	}
	if timeoutSecs <= 0 {
		timeoutSecs = 120
	}
	q := url.Values{}
	q.Set("token", a.token)
	q.Set("timeout", strconv.Itoa(timeoutSecs))
	q.Set("memory", "1024")
	q.Set("clean", "true")
	q.Set("format", "json")
	if maxItems > 0 {
		q.Set("limit", strconv.Itoa(maxItems))
	}
	endpoint := fmt.Sprintf("%s/acts/%s/run-sync-get-dataset-items?%s", a.base, ActorPath(actorID), q.Encode())
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSecs+15)*time.Second)
	defer cancel()
	res, err := a.http.PostJSON(ctx, endpoint, input, nil)
	if err != nil {
		return nil, apperr.FromProvider(providerApify, apperr.Network, "Tidak bisa menghubungi Apify: "+err.Error(), err)
	}
	if res.Status < 200 || res.Status >= 300 {
		return nil, apifyStatusError(res.Status, res.Snippet(200))
	}
	var items []json.RawMessage
	if err := res.JSON(&items); err != nil {
		return []json.RawMessage{}, nil
	}
	return items, nil
}

// SerpRanks runs one actor call per device with every keyword as a query.
func (a *ApifySerp) SerpRanks(ctx context.Context, items []domain.SerpRankRequest, locationCode int32, languageCode, targetDomain string) ([]domain.RankResult, error) {
	if len(items) == 0 {
		return []domain.RankResult{}, nil
	}
	byDevice := map[domain.Device][]domain.SerpRankRequest{}
	var order []domain.Device
	for _, it := range items {
		if _, seen := byDevice[it.Device]; !seen {
			order = append(order, it.Device)
		}
		byDevice[it.Device] = append(byDevice[it.Device], it)
	}
	found := map[string]domain.RankResult{}
	for _, device := range order {
		reqs := byDevice[device]
		queries := make([]string, 0, len(reqs))
		for _, r := range reqs {
			queries = append(queries, r.Keyword)
		}
		raw, err := a.RunActor(ctx, a.actors[serpActorKey], map[string]any{
			"queries":                  strings.Join(queries, "\n"),
			"countryCode":              CountryForLocation(locationCode),
			"languageCode":             languageCode,
			"maxPagesPerQuery":         1,
			"resultsPerPage":           100,
			"mobileResults":            device == domain.DeviceMobile,
			"includeUnfilteredResults": false,
			"saveHtml":                 false,
			"saveHtmlToKeyValueStore":  false,
		}, serpTimeoutSecs, 0)
		if err != nil {
			return nil, err
		}
		for _, r := range NormalizeSerpItems(raw, reqs, targetDomain) {
			found[string(r.Device)+":"+strings.ToLower(r.Keyword)] = r
		}
	}
	out := make([]domain.RankResult, 0, len(items))
	for _, it := range items {
		if r, ok := found[string(it.Device)+":"+strings.ToLower(it.Keyword)]; ok {
			out = append(out, r)
			continue
		}
		out = append(out, emptyRank(it))
	}
	return out, nil
}

func emptyRank(req domain.SerpRankRequest) domain.RankResult {
	return domain.RankResult{Keyword: req.Keyword, Device: req.Device, SerpFeatures: []string{}, Top10: []domain.TopResult{}}
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	}
	return ""
}

func asNumber(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case string:
		n, err := strconv.ParseFloat(regexp.MustCompile(`[^\d.-]`).ReplaceAllString(t, ""), 64)
		if err != nil {
			return 0
		}
		return n
	}
	return 0
}

func nonEmptyList(v any) bool {
	l, ok := v.([]any)
	return ok && len(l) > 0
}

// NormalizeSerpItems maps Google Search Scraper items (one per query page)
// to one RankResult per request, in request order. A query without an item
// yields position nil and an empty top 10.
func NormalizeSerpItems(raw []json.RawMessage, requests []domain.SerpRankRequest, targetDomain string) []domain.RankResult {
	byTerm := map[string]map[string]any{}
	for _, r := range raw {
		var v any
		if err := json.Unmarshal(r, &v); err != nil {
			continue
		}
		it := asMap(v)
		term := asString(asMap(it["searchQuery"])["term"])
		if term == "" {
			term = asString(it["query"])
		}
		if term == "" {
			term = asString(it["term"])
		}
		if term == "" {
			continue
		}
		key := strings.ToLower(term)
		if _, seen := byTerm[key]; !seen {
			byTerm[key] = it
		}
	}
	out := make([]domain.RankResult, 0, len(requests))
	for _, req := range requests {
		it, ok := byTerm[strings.ToLower(req.Keyword)]
		if !ok {
			out = append(out, emptyRank(req))
			continue
		}
		res := emptyRank(req)
		organicRaw, _ := it["organicResults"].([]any)
		type organic struct {
			url      string
			host     string
			position int32
		}
		organics := make([]organic, 0, len(organicRaw))
		for i, o := range organicRaw {
			r := asMap(o)
			u := asString(r["url"])
			pos := int32(asNumber(r["position"]))
			if pos == 0 {
				pos = int32(i + 1)
			}
			organics = append(organics, organic{url: u, host: HostOf(u), position: pos})
		}
		hasVideo := false
		for i, o := range organics {
			if res.Position == nil && DomainMatches(o.host, targetDomain) {
				p, u := o.position, o.url
				res.Position, res.URL = &p, &u
			}
			if i < 10 {
				res.Top10 = append(res.Top10, domain.TopResult{Domain: o.host, Position: o.position, URL: o.url})
			}
			if o.host == "youtube.com" {
				hasVideo = true
			}
		}
		if nonEmptyList(it["paidResults"]) {
			res.SerpFeatures = append(res.SerpFeatures, "ads")
		}
		if nonEmptyList(it["peopleAlsoAsk"]) {
			res.SerpFeatures = append(res.SerpFeatures, "people_also_ask")
		}
		if nonEmptyList(it["relatedQueries"]) {
			res.SerpFeatures = append(res.SerpFeatures, "related_searches")
		}
		if ao := it["aiOverview"]; ao != nil && ao != false && ao != "" {
			res.SerpFeatures = append(res.SerpFeatures, "ai_overview")
		}
		if nonEmptyList(it["paidProducts"]) {
			res.SerpFeatures = append(res.SerpFeatures, "shopping")
		}
		if hasVideo {
			res.SerpFeatures = append(res.SerpFeatures, "video")
		}
		out = append(out, res)
	}
	return out
}
