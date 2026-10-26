package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/content/app"
	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

func november() domain.Month { m, _ := domain.ParseMonth("2026-11"); return m }

func TestMockDraftIsDeterministicAndInsideTheMonth(t *testing.T) {
	in := app.PlanPrompt{ClientName: "Kopi Senja", Month: november(), Brief: domain.Brief{Pillars: []string{"Edukasi", "Promo"}, Frequency: []domain.Frequency{{Platform: domain.PlatformInstagram, PerWeek: 3}, {Platform: domain.PlatformTikTok, PerWeek: 1}}}}
	a, _ := Mock{}.draftPlan(context.Background(), in)
	b, _ := Mock{}.draftPlan(context.Background(), in)
	if len(a) != 13+4 {
		t.Fatal(len(a))
	}
	for i := range a {
		if a[i].Title != b[i].Title || a[i].Caption != b[i].Caption {
			t.Fatal("not deterministic")
		}
		if !in.Month.Contains(a[i].PlannedDate) || strings.Contains(a[i].Title, "%!") {
			t.Fatalf("%+v", a[i])
		}
		if a[i].Platform == domain.PlatformTikTok && a[i].Format != domain.FormatReel {
			t.Fatal("tiktok must be reels")
		}
	}
	if a[0].Hashtags[0] != "#KopiSenja" || a[0].Pillar != "Edukasi" || a[1].Pillar != "Promo" {
		t.Fatalf("%+v", a[0])
	}
}

func TestMockSummaryHasSections(t *testing.T) {
	md, _ := Mock{}.summarize(context.Background(), domain.ReportData{Month: "2026-11", ClientName: "Kopi", Planned: 10, Published: 8, OnTimeRate: 72.5, RevisionAvg: 1.6})
	for _, want := range []string{"## Ringkasan", "## Sorotan", "## Rekomendasi", "72,5%", "8 dari 10", "Data engagement belum tersedia"} {
		if !strings.Contains(md, want) {
			t.Fatalf("missing %q in\n%s", want, md)
		}
	}
}

func TestParseDraftDropsInvalidItems(t *testing.T) {
	content := "```json\n{\"items\":[" +
		`{"date":"2026-11-03","platform":"instagram","format":"reel","title":"A","hashtags":["#a"]},` +
		`{"date":"2026-12-01","platform":"INSTAGRAM","format":"FEED","title":"bulan lain"},` +
		`{"date":"2026-11-04","platform":"LINKEDIN","format":"FEED","title":"platform lain"}` +
		"]}\n```"
	got, err := ParseDraft(content, november())
	if err != nil || len(got) != 1 || got[0].Platform != domain.PlatformInstagram || got[0].Format != domain.FormatReel {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err := ParseDraft("maaf, tidak bisa", november()); !apperr.Is(err, apperr.Unknown) {
		t.Fatal(err)
	}
}

func TestSourcePicksRealWithKey(t *testing.T) {
	var gotAuth, gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		var req chatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		gotModel = req.Model
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"items\":[{\"date\":\"2026-11-02\",\"platform\":\"FACEBOOK\",\"format\":\"FEED\",\"title\":\"Halo\"}]}"}}]}`))
	}))
	defer srv.Close()
	src := NewSource(func(context.Context) (Credentials, error) { return Credentials{APIKey: "sk", Model: "x/model"}, nil }, Options{Endpoint: srv.URL})
	items, model, err := src.DraftPlan(context.Background(), app.PlanPrompt{ClientName: "Kopi", Month: november()})
	if err != nil || model != "x/model" || len(items) != 1 || gotAuth != "Bearer sk" || gotModel != "x/model" {
		t.Fatalf("%v %s %+v %s", err, model, items, gotAuth)
	}

	mock := NewSource(func(context.Context) (Credentials, error) { return Credentials{}, nil }, Options{})
	if _, model, _ := mock.SummarizeReport(context.Background(), domain.ReportData{Month: "2026-11"}); model != MockModel {
		t.Fatal(model)
	}
}

func TestRealMapsHTTPErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	_, err := NewReal(Credentials{APIKey: "sk"}, Options{Endpoint: srv.URL}).summarize(context.Background(), domain.ReportData{})
	if !apperr.Is(err, apperr.RateLimit) {
		t.Fatal(err)
	}
}
