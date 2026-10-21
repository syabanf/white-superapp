package seodata

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

type dfsServer struct {
	t        *testing.T
	mu       sync.Mutex
	requests map[string][]map[string]any
	handlers map[string]func(w http.ResponseWriter, task map[string]any)
}

func newDfsServer(t *testing.T) (*dfsServer, *DataForSEO) {
	s := &dfsServer{t: t, requests: map[string][]map[string]any{}, handlers: map[string]func(http.ResponseWriter, map[string]any){}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Basic "+base64.StdEncoding.EncodeToString([]byte("login:secret")) {
			t.Errorf("auth header %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		var tasks []map[string]any
		_ = json.Unmarshal(body, &tasks)
		var task map[string]any
		if len(tasks) > 0 {
			task = tasks[0]
		}
		s.mu.Lock()
		s.requests[r.URL.Path] = append(s.requests[r.URL.Path], task)
		h := s.handlers[r.URL.Path]
		s.mu.Unlock()
		if h == nil {
			http.NotFound(w, r)
			return
		}
		h(w, task)
	}))
	t.Cleanup(srv.Close)
	return s, NewDataForSEO("login", "secret", Options{DataForSEOBase: srv.URL, Clock: fixedClock})
}

func ok(result ...any) func(http.ResponseWriter, map[string]any) {
	return func(w http.ResponseWriter, _ map[string]any) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 20000, "tasks": []any{map[string]any{"status_code": 20000, "result": result}}})
	}
}

func TestDataForSEOKeywordIdeas(t *testing.T) {
	s, d := newDfsServer(t)
	item := func(kw string, vol int, cpc float64, kd float64, intent string, months ...int) map[string]any {
		var monthly []any
		for i, v := range months {
			monthly = append(monthly, map[string]any{"year": 2026, "month": 9 - i, "search_volume": v})
		}
		return map[string]any{
			"keyword":            kw,
			"keyword_info":       map[string]any{"search_volume": vol, "cpc": cpc, "competition": 0.4, "monthly_searches": monthly},
			"keyword_properties": map[string]any{"keyword_difficulty": kd},
			"search_intent_info": map[string]any{"main_intent": intent},
			"serp_info":          map[string]any{"serp_item_types": []string{"organic", "video"}},
		}
	}
	s.handlers["/v3/dataforseo_labs/google/keyword_ideas/live"] = ok(map[string]any{"items": []any{
		item("kopi susu", 1000, 0.5, 42.6, "commercial", 900, 950, 1000),
		item("kopi susu murah", 5000, 0.25, 10, "transactional"),
		item("", 1, 0, 0, ""),
	}})
	ideas, err := d.KeywordIdeas(context.Background(), "kopi susu", 2360, "id", domain.ModeIdeas, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(ideas) != 2 || ideas[0].Keyword != "kopi susu murah" {
		t.Fatalf("%+v", ideas)
	}
	second := ideas[1]
	if second.CPC != 8000 || second.Difficulty != 43 || second.Intent != domain.IntentCommercial || second.Competition != 0.4 {
		t.Fatalf("%+v", second)
	}
	// monthly sorted ascending then left-padded to 12 with the first value
	wantTrend := []int32{1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 950, 900}
	if !reflect.DeepEqual(second.Trend, wantTrend) {
		t.Fatalf("trend %v", second.Trend)
	}
	if !reflect.DeepEqual(second.SerpFeatures, []string{"organic", "video"}) {
		t.Fatalf("%v", second.SerpFeatures)
	}
	if len(ideas[0].Trend) != 12 || ideas[0].Trend[0] != 5000 {
		t.Fatalf("flat trend %v", ideas[0].Trend)
	}
	req := s.requests["/v3/dataforseo_labs/google/keyword_ideas/live"][0]
	if req["limit"].(float64) != 50 || req["include_serp_info"] != true || req["include_clickstream_data"] != false {
		t.Fatalf("task %v", req)
	}

	s.handlers["/v3/dataforseo_labs/google/keyword_suggestions/live"] = ok(map[string]any{"items": []any{
		item("cara bikin kopi susu", 300, 0.1, 5, "informational"),
		item("kopi susu enak", 800, 0.1, 5, "informational"),
		item("how to make kopi susu", 100, 0.1, 5, "informational"),
	}})
	questions, err := d.KeywordIdeas(context.Background(), "kopi susu", 2360, "id", domain.ModeQuestions, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(questions) != 2 || questions[0].Keyword != "cara bikin kopi susu" || questions[1].Keyword != "how to make kopi susu" {
		t.Fatalf("%+v", questions)
	}
	qreq := s.requests["/v3/dataforseo_labs/google/keyword_suggestions/live"][0]
	if qreq["keyword"] != "kopi susu" || qreq["filters"] == nil {
		t.Fatalf("task %v", qreq)
	}

	s.handlers["/v3/dataforseo_labs/google/related_keywords/live"] = ok(map[string]any{"items": []any{
		map[string]any{"keyword_data": item("kopi arabika", 700, 0.3, 20, "navigational")},
	}})
	related, err := d.KeywordIdeas(context.Background(), "kopi", 2360, "id", domain.ModeRelated, 0)
	if err != nil || len(related) != 1 || related[0].Intent != domain.IntentNavigational {
		t.Fatalf("%v %+v", err, related)
	}
	if s.requests["/v3/dataforseo_labs/google/related_keywords/live"][0]["depth"].(float64) != 2 {
		t.Fatal("depth")
	}
}

func TestDataForSEOKeywordMetricsFallback(t *testing.T) {
	s, d := newDfsServer(t)
	s.handlers["/v3/dataforseo_labs/google/historical_search_volume/live"] = func(w http.ResponseWriter, _ map[string]any) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 20000, "tasks": []any{map[string]any{"status_code": 40400, "status_message": "no data"}}})
	}
	s.handlers["/v3/keywords_data/google_ads/search_volume/live"] = ok(
		map[string]any{"keyword": "Kopi Susu", "search_volume": 1200, "cpc": 0.1, "competition_index": 55},
	)
	s.handlers["/v3/dataforseo_labs/google/bulk_keyword_difficulty/live"] = ok(map[string]any{"items": []any{
		map[string]any{"keyword": "kopi susu", "keyword_difficulty": 33.4},
		map[string]any{"keyword": "unknown kw", "keyword_difficulty": 7},
	}})
	out, err := d.KeywordMetrics(context.Background(), []string{"kopi susu", "unknown kw"}, 2360, "id")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].Volume != 1200 || out[0].CPC != 1600 || out[0].Competition != 0.55 || out[0].Difficulty != 33 {
		t.Fatalf("%+v", out[0])
	}
	if out[1].Keyword != "unknown kw" || out[1].Volume != 0 || out[1].Difficulty != 7 || len(out[1].Trend) != 12 {
		t.Fatalf("%+v", out[1])
	}
}

func TestDataForSEOErrorMapping(t *testing.T) {
	s, d := newDfsServer(t)
	cases := []struct {
		name    string
		handler func(http.ResponseWriter, map[string]any)
		want    apperr.Code
	}{
		{"http 401", func(w http.ResponseWriter, _ map[string]any) { w.WriteHeader(401) }, apperr.TokenExpired},
		{"http 402", func(w http.ResponseWriter, _ map[string]any) { w.WriteHeader(402) }, apperr.Permission},
		{"http 403", func(w http.ResponseWriter, _ map[string]any) { w.WriteHeader(403) }, apperr.Permission},
		{"http 429", func(w http.ResponseWriter, _ map[string]any) { w.WriteHeader(429) }, apperr.RateLimit},
		{"http 404", func(w http.ResponseWriter, _ map[string]any) { w.WriteHeader(404) }, apperr.NotFound},
		{"http 500", func(w http.ResponseWriter, _ map[string]any) { w.WriteHeader(500) }, apperr.Unknown},
		{"envelope 40200", func(w http.ResponseWriter, _ map[string]any) {
			_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 40200, "status_message": "Payment Required"})
		}, apperr.Permission},
		{"envelope 40202", func(w http.ResponseWriter, _ map[string]any) {
			_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 40202, "status_message": "Too many"})
		}, apperr.RateLimit},
		{"envelope 50000", func(w http.ResponseWriter, _ map[string]any) {
			_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 50000, "status_message": "Internal"})
		}, apperr.Unknown},
		{"task 40402", func(w http.ResponseWriter, _ map[string]any) {
			_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 20000, "tasks": []any{map[string]any{"status_code": 40402, "status_message": "Not found"}}})
		}, apperr.NotFound},
		{"task 40210", func(w http.ResponseWriter, _ map[string]any) {
			_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 20000, "tasks": []any{map[string]any{"status_code": 40210, "status_message": "No money"}}})
		}, apperr.Permission},
	}
	for _, c := range cases {
		s.handlers["/v3/backlinks/summary/live"] = c.handler
		_, err := d.BacklinkSummary(context.Background(), "x.id")
		e := apperr.As(err)
		if e == nil || e.Code != c.want || e.Provider != providerDataForSEO {
			t.Fatalf("%s: got %v", c.name, err)
		}
	}
	e := apperr.As(func() error {
		s.handlers["/v3/backlinks/summary/live"] = cases[0].handler
		_, err := d.BacklinkSummary(context.Background(), "x.id")
		return err
	}())
	if !e.RequiresReconnect || e.Message != "Kredensial DataForSEO ditolak saat memanggil /v3/backlinks/summary/live. Periksa DATAFORSEO_LOGIN/PASSWORD." {
		t.Fatalf("%+v", e)
	}
}

func TestDataForSEOSerpRanks(t *testing.T) {
	s, d := newDfsServer(t)
	s.handlers["/v3/serp/google/organic/live/regular"] = func(w http.ResponseWriter, task map[string]any) {
		kw := task["keyword"].(string)
		items := []any{
			map[string]any{"type": "people_also_ask"},
			map[string]any{"type": "organic", "rank_group": 1, "rank_absolute": 2, "domain": "www.tomoro.id", "url": "https://www.tomoro.id/a"},
			map[string]any{"type": "organic", "rank_group": 2, "rank_absolute": 3, "domain": "shop.kopinusantara.id", "url": "https://shop.kopinusantara.id/" + kw},
			map[string]any{"type": "organic", "rank_group": 3, "rank_absolute": 4, "domain": "kopinusantara.id", "url": "https://kopinusantara.id/b"},
		}
		ok(map[string]any{"item_types": []string{"organic", "people_also_ask", "video"}, "items": items})(w, task)
	}
	reqs := []domain.SerpRankRequest{}
	for i := 0; i < 7; i++ {
		reqs = append(reqs, domain.SerpRankRequest{Keyword: fmt.Sprintf("kw%d", i), Device: domain.DeviceDesktop})
	}
	out, err := d.SerpRanks(context.Background(), reqs, 2360, "id", "kopinusantara.id")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 7 || len(s.requests["/v3/serp/google/organic/live/regular"]) != 7 {
		t.Fatalf("%d results, %d calls", len(out), len(s.requests["/v3/serp/google/organic/live/regular"]))
	}
	r := out[3]
	if r.Keyword != "kw3" || r.Position == nil || *r.Position != 2 || *r.URL != "https://shop.kopinusantara.id/kw3" {
		t.Fatalf("%+v", r)
	}
	if !reflect.DeepEqual(r.SerpFeatures, []string{"people_also_ask", "video"}) || len(r.Top10) != 3 || r.Top10[0].Domain != "tomoro.id" {
		t.Fatalf("%+v", r)
	}
	task := s.requests["/v3/serp/google/organic/live/regular"][0]
	if task["device"] != "desktop" || task["depth"].(float64) != 100 {
		t.Fatalf("task %v", task)
	}
}

func TestDataForSEOBacklinksAndDomains(t *testing.T) {
	s, d := newDfsServer(t)
	s.handlers["/v3/backlinks/summary/live"] = ok(map[string]any{"backlinks": 1000, "referring_domains": 120, "rank": 347, "referring_links_attributes": map[string]any{"nofollow": 300}, "backlinks_spam_score": 25})
	s.handlers["/v3/backlinks/timeseries_new_lost_summary/live"] = ok(map[string]any{"items": []any{
		map[string]any{"new_backlinks": 40, "lost_backlinks": 10},
		map[string]any{"new_backlinks": 20, "lost_backlinks": 5},
	}})
	sum, err := d.BacklinkSummary(context.Background(), "kopinusantara.id")
	if err != nil {
		t.Fatal(err)
	}
	want := domain.BacklinkSummary{Backlinks: 1000, ReferringDomains: 120, Dofollow: 700, Nofollow: 300, DomainRank: 35, NewLast30: 60, LostLast30: 15, ToxicShare: 0.25}
	if sum != want {
		t.Fatalf("%+v", sum)
	}
	if got := s.requests["/v3/backlinks/timeseries_new_lost_summary/live"][0]["date_from"]; got != "2026-09-21" {
		t.Fatalf("date_from %v", got)
	}

	s.handlers["/v3/backlinks/backlinks/live"] = ok(map[string]any{"items": []any{
		map[string]any{"url_from": "https://a.com/x", "domain_from": "a.com", "url_to": "https://kopinusantara.id/", "anchor": nil, "dofollow": false, "domain_from_rank": 456, "backlink_spam_score": 70, "first_seen": "2026-01-02 10:00:00 +00:00", "last_seen": "2026-09-01 10:00:00 +00:00", "is_lost": true},
	}})
	links, err := d.Backlinks(context.Background(), "kopinusantara.id", 0, domain.BacklinksLost)
	if err != nil || len(links) != 1 {
		t.Fatal(err, links)
	}
	l := links[0]
	if l.Anchor != "" || l.Dofollow || l.DomainRank != 46 || l.SpamScore != 70 || l.FirstSeen != "2026-01-02" || l.LastSeen != "2026-09-01" || !l.IsLost {
		t.Fatalf("%+v", l)
	}
	body := s.requests["/v3/backlinks/backlinks/live"][0]
	if body["backlinks_status_type"] != "all" || body["limit"].(float64) != 200 || body["filters"] == nil {
		t.Fatalf("%v", body)
	}

	s.handlers["/v3/dataforseo_labs/google/domain_rank_overview/live"] = ok(map[string]any{"items": []any{
		map[string]any{"metrics": map[string]any{"organic": map[string]any{"count": 500, "etv": 1234.6, "estimated_paid_traffic_cost": 2.5, "pos_1": 3, "pos_2_3": 7, "pos_4_10": 20}}},
	}})
	o, err := d.DomainOverview(context.Background(), "kopinusantara.id", 2360, "id")
	if err != nil {
		t.Fatal(err)
	}
	if o.OrganicKeywords != 500 || o.OrganicTraffic != 1235 || o.OrganicCost != 40000 || o.Top3 != 10 || o.Top10 != 30 || o.Top100 != 500 || o.Backlinks != 1000 || o.DomainRank != 35 {
		t.Fatalf("%+v", o)
	}

	s.handlers["/v3/dataforseo_labs/google/ranked_keywords/live"] = ok(map[string]any{"items": []any{
		map[string]any{"keyword_data": map[string]any{"keyword": "kopi", "keyword_info": map[string]any{"search_volume": 900}}, "ranked_serp_element": map[string]any{"serp_item": map[string]any{"rank_group": 4, "url": "https://kopinusantara.id/kopi"}}},
		map[string]any{"keyword_data": map[string]any{"keyword": "susu"}},
		map[string]any{"keyword_data": map[string]any{"keyword": ""}},
	}})
	kws, err := d.DomainKeywords(context.Background(), "kopinusantara.id", 2360, "id", 0)
	if err != nil || len(kws) != 2 || kws[0].Position != 4 || kws[1].Position != 100 || kws[1].Volume != 0 {
		t.Fatalf("%v %+v", err, kws)
	}

	s.handlers["/v3/dataforseo_labs/google/competitors_domain/live"] = ok(map[string]any{"items": []any{
		map[string]any{"domain": "kopinusantara.id", "intersections": 999},
		map[string]any{"domain": "kopikenangan.com", "intersections": 120, "metrics": map[string]any{"organic": map[string]any{"etv": 5000.4}}},
	}})
	cs, err := d.Competitors(context.Background(), "kopinusantara.id", 2360, "id")
	if err != nil || len(cs) != 1 || cs[0].Domain != "kopikenangan.com" || cs[0].Intersections != 120 || cs[0].OrganicTraffic != 5000 {
		t.Fatalf("%v %+v", err, cs)
	}
	if s.requests["/v3/dataforseo_labs/google/competitors_domain/live"][0]["exclude_top_domains"] != true {
		t.Fatal("exclude_top_domains")
	}
}
