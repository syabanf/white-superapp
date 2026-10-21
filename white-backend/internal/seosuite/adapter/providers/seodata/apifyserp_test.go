package seodata

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

func TestApifyConfigHelpers(t *testing.T) {
	if !reflect.DeepEqual(ParseActorOverrides(""), DefaultActors) || !reflect.DeepEqual(ParseActorOverrides("not json"), DefaultActors) {
		t.Fatal("defaults")
	}
	got := ParseActorOverrides(`{"serp":"me/my-serp","bogus":"x/y","tiktok":"invalid actor"}`)
	want := map[string]string{}
	for k, v := range DefaultActors {
		want[k] = v
	}
	want["serp"] = "me/my-serp"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%v", got)
	}
	if ActorPath("apify/google-search-scraper") != "apify~google-search-scraper" {
		t.Fatal("actorPath")
	}
	if CountryForLocation(2360) != "id" || CountryForLocation(999) != "id" || CountryForLocation(2840) != "us" {
		t.Fatal("country")
	}
	if !DomainMatches("shop.kopinusantara.id", "kopinusantara.id") || DomainMatches("notkopinusantara.id", "kopinusantara.id") {
		t.Fatal("domainMatches")
	}
	if HostOf("https://www.Tomoro.id/menu") != "tomoro.id" || HostOf("::bad") != "" {
		t.Fatal("hostOf")
	}
}

func rawItems(t *testing.T, v any) []json.RawMessage {
	t.Helper()
	b, _ := json.Marshal(v)
	var out []json.RawMessage
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestNormalizeSerpItems(t *testing.T) {
	items := rawItems(t, []any{map[string]any{
		"searchQuery": map[string]any{"term": "kopi susu"},
		"organicResults": []any{
			map[string]any{"url": "https://www.tomoro.id/menu", "position": 1},
			map[string]any{"url": "https://kopinusantara.id/kopi-susu", "position": 2},
			map[string]any{"url": "https://www.youtube.com/watch?v=1"},
		},
		"peopleAlsoAsk": []any{map[string]any{"q": 1}},
		"paidResults":   []any{},
	}})
	out := NormalizeSerpItems(items, []domain.SerpRankRequest{{Keyword: "Kopi Susu", Device: domain.DeviceMobile}, {Keyword: "missing", Device: domain.DeviceMobile}}, "kopinusantara.id")
	if len(out) != 2 || out[0].Position == nil || *out[0].Position != 2 || *out[0].URL != "https://kopinusantara.id/kopi-susu" {
		t.Fatalf("%+v", out)
	}
	if out[0].Top10[0] != (domain.TopResult{Domain: "tomoro.id", Position: 1, URL: "https://www.tomoro.id/menu"}) || out[0].Top10[2].Position != 3 {
		t.Fatalf("%+v", out[0].Top10)
	}
	if !reflect.DeepEqual(out[0].SerpFeatures, []string{"people_also_ask", "video"}) {
		t.Fatalf("%v", out[0].SerpFeatures)
	}
	if out[1].Position != nil || len(out[1].Top10) != 0 || out[1].Keyword != "missing" {
		t.Fatalf("%+v", out[1])
	}
}

func TestApifyRunSync(t *testing.T) {
	var gotPath, gotQuery string
	var gotBody map[string]any
	status := 200
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		if status != 200 {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":"nope"}`))
			return
		}
		_, _ = w.Write([]byte(`[{"searchQuery":{"term":"kopi susu"},"organicResults":[{"url":"https://kopinusantara.id/a","position":1}]},{"searchQuery":{"term":"villa bali"},"organicResults":[{"url":"https://airbnb.com/x"}]}]`))
	}))
	defer srv.Close()
	a := NewApifySerp("tok", `{"serp":"me/serp"}`, srv.URL)
	out, err := a.SerpRanks(context.Background(), []domain.SerpRankRequest{
		{Keyword: "kopi susu", Device: domain.DeviceMobile},
		{Keyword: "villa bali", Device: domain.DeviceMobile},
	}, 2360, "id", "kopinusantara.id")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/acts/me~serp/run-sync-get-dataset-items" {
		t.Fatal(gotPath)
	}
	for _, part := range []string{"token=tok", "timeout=180", "memory=1024", "clean=true", "format=json"} {
		if !strings.Contains(gotQuery, part) {
			t.Fatalf("query %s lacks %s", gotQuery, part)
		}
	}
	if gotBody["queries"] != "kopi susu\nvilla bali" || gotBody["mobileResults"] != true || gotBody["countryCode"] != "id" || gotBody["resultsPerPage"].(float64) != 100 {
		t.Fatalf("%v", gotBody)
	}
	if *out[0].Position != 1 || out[1].Position != nil || len(out[1].Top10) != 1 || out[1].Top10[0].Position != 1 {
		t.Fatalf("%+v", out)
	}

	for code, want := range map[int]apperr.Code{401: apperr.TokenExpired, 402: apperr.Permission, 404: apperr.NotFound, 408: apperr.Network, 429: apperr.RateLimit, 500: apperr.Unknown} {
		status = code
		_, err := a.RunActor(context.Background(), "me/serp", map[string]any{}, 10, 0)
		e := apperr.As(err)
		if e.Code != want || e.Provider != providerApify {
			t.Fatalf("%d -> %v", code, err)
		}
		if code == 402 && e.RequiresReconnect {
			t.Fatal("402 must not ask to reconnect")
		}
		if code == 500 && !strings.Contains(e.Message, "Apify menjawab 500") {
			t.Fatal(e.Message)
		}
	}
	status = 200
	if _, err := NewApifySerp("", "", srv.URL).RunActor(context.Background(), "me/serp", nil, 10, 0); !apperr.Is(err, apperr.Permission) {
		t.Fatalf("missing token: %v", err)
	}
}
