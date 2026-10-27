package openrouter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/reports/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

func f(v float64) *float64 { return &v }

func strp(s string) *string { return &s }

// Payload ported from tests/reports-mock-insight.test.ts.
func payload() domain.InsightPayload {
	health := int32(82)
	return domain.InsightPayload{
		Social: &domain.InsightSocial{
			Followers: 125_400, FollowerGrowth: 2_600, FollowerGrowthPct: f(2.1), EngagementRate: 4.32, EngagementRateDeltaPct: f(-3.5),
			Posts: 24, Reach: 890_000, ReachDeltaPct: f(5.2),
			TopPosts: []domain.InsightTopPost{{Platform: "INSTAGRAM", Caption: "Kopi single origin Gayo tiba di toko", Type: "Reels", Engagements: 3_590, Likes: 3_210, Comments: 148}},
		},
		Seo: &domain.InsightSeo{
			Clicks: 8_412, ClicksDeltaPct: f(12.3), Impressions: 402_000, ImpressionsDeltaPct: f(8.8), Ctr: 2.09, Position: 9.4,
			PositionDeltaAbs: -0.6, Health: &health,
			TopQueries: []domain.InsightTopQuery{
				{Query: "kopi gayo asli", Clicks: 812, Impressions: 22_400, Ctr: 3.63, Position: 2.4},
				{Query: "beli kopi arabica online", Clicks: 402, Impressions: 31_000, Ctr: 1.3, Position: 6.8},
			},
		},
		Ads: &domain.InsightAds{
			Currency: "IDR", Spend: 45_200_000, SpendDeltaPct: f(10.0), Results: 1_204, ResultsDeltaPct: f(18.2), Cpr: 37_542, CprDeltaPct: f(-8.0),
			Ctr: 1.84, Cpm: 21_400, Roas: f(3.4),
			Campaigns: []domain.InsightCampaign{
				{Name: "Promo Kemerdekaan", Objective: "OUTCOME_SALES", Spend: 15_100_000, Results: 480, ResultType: strp("purchase"), Cpr: 31_458},
				{Name: "Always-on Traffic", Objective: "OUTCOME_TRAFFIC", Spend: 12_400_000, Results: 402, ResultType: strp("link_click"), Cpr: 30_845},
			},
		},
	}
}

func input(module, lang string) domain.InsightInput {
	return domain.InsightInput{
		Module: module, Language: lang, ClientName: "Kopi Nusantara", Data: payload(),
		Range: domain.InsightRange{From: "2026-07-22", To: "2026-08-18", Days: 28, PrevFrom: "2026-06-24", PrevTo: "2026-07-21"},
	}
}

func TestMockSectionsInOrder(t *testing.T) {
	md := GenerateMock(input(domain.ModuleOverview, "id"))
	last := -1
	for _, s := range []string{"## Ringkasan eksekutif", "## Sorotan performa", "## Area perbaikan", "## Rekomendasi 30 hari"} {
		idx := strings.Index(md, s)
		if idx <= last {
			t.Fatalf("section %q out of order (%d after %d)", s, idx, last)
		}
		last = idx
	}
}

func TestMockFiveRecommendations(t *testing.T) {
	md := GenerateMock(input(domain.ModuleOverview, "id"))
	block := strings.SplitN(md, "## Rekomendasi 30 hari", 2)[1]
	var items []string
	for _, l := range strings.Split(block, "\n") {
		if regexp.MustCompile(`^\d+\.\s`).MatchString(strings.TrimSpace(l)) {
			items = append(items, l)
		}
	}
	if len(items) != 5 || !strings.HasPrefix(items[0], "1.") || !strings.HasPrefix(items[4], "5.") {
		t.Fatalf("recommendations %q", items)
	}
	// With nothing but SEO data the pool is padded to five as well.
	in := input(domain.ModuleSeo, "id")
	in.Data = domain.InsightPayload{}
	if n := strings.Count(GenerateMock(in), "\n5. "); n != 1 {
		t.Fatalf("padded recommendations missing")
	}
}

func TestMockRealNumbers(t *testing.T) {
	md := GenerateMock(input(domain.ModuleOverview, "id"))
	for _, want := range []string{"125,4\u00a0rb", "4,32%", "8,4\u00a0rb", "Rp 45,2\u00a0jt", "Kopi Nusantara", "28 hari", "Promo Kemerdekaan"} {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q", want)
		}
	}
	// Always-on Traffic has the lower CPR, so it is the efficient one.
	if !strings.Contains(md, "Kampanye paling efisien: **Always-on Traffic**") || !strings.Contains(md, "Kampanye **Promo Kemerdekaan** paling boros") {
		t.Fatalf("best/worst campaign:\n%s", md)
	}
}

func TestMockModuleSections(t *testing.T) {
	seo := GenerateMock(input(domain.ModuleSeo, "id"))
	if !strings.Contains(seo, "## Peluang kata kunci") || strings.Contains(seo, "## Alokasi anggaran") || !strings.Contains(seo, "beli kopi arabica online") {
		t.Fatalf("SEO insight:\n%s", seo)
	}
	ads := GenerateMock(input(domain.ModuleAds, "id"))
	if !strings.Contains(ads, "## Alokasi anggaran") || strings.Contains(ads, "## Peluang kata kunci") || !strings.Contains(ads, "Always-on Traffic") {
		t.Fatalf("ADS insight:\n%s", ads)
	}
	if !strings.Contains(ads, "**Promo Kemerdekaan** — Rp 15,1\u00a0jt (54,9% dari total), 480 hasil, CPR Rp 31.458.") {
		t.Fatalf("budget line:\n%s", ads)
	}
	ov := GenerateMock(input(domain.ModuleOverview, "id"))
	if strings.Contains(ov, "## Peluang kata kunci") || strings.Contains(ov, "## Alokasi anggaran") {
		t.Fatal("OVERVIEW has no module section")
	}
}

func TestMockDeterministicAndNeverMentionsAI(t *testing.T) {
	a, b := GenerateMock(input(domain.ModuleOverview, "id")), GenerateMock(input(domain.ModuleOverview, "id"))
	if a != b {
		t.Fatal("not deterministic")
	}
	if regexp.MustCompile(`\bai\b|kecerdasan buatan|model bahasa|language model`).MatchString(strings.ToLower(a)) {
		t.Fatal("mentions AI")
	}
}

func TestMockEnglishKeepsIndonesianHeadings(t *testing.T) {
	md := GenerateMock(input(domain.ModuleSeo, "en"))
	for _, want := range []string{"## Ringkasan eksekutif", "## Peluang kata kunci", "Over the last 28 days"} {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestPick(t *testing.T) {
	if Pick(Credentials{}, Options{}).Model() != MockModel {
		t.Fatal("no key picks the mock")
	}
	if Pick(Credentials{APIKey: "k"}, Options{}).Model() != DefaultModel {
		t.Fatal("default model")
	}
	if Pick(Credentials{APIKey: "k", Model: "openai/gpt-5"}, Options{}).Model() != "openai/gpt-5" {
		t.Fatal("model override from the public id")
	}
}

func TestRealRequestAndResponse(t *testing.T) {
	var got struct {
		header http.Header
		body   chatRequest
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.header = r.Header.Clone()
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &got.body)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"  ## Ringkasan eksekutif\nIsi  "}}]}`))
	}))
	defer srv.Close()

	src := NewSource(func(context.Context) (Credentials, error) { return Credentials{APIKey: "sk-test"}, nil }, Options{Endpoint: srv.URL, Referer: "https://app.mixist.id"})
	content, model, err := src.Generate(context.Background(), input(domain.ModuleSeo, "en"))
	if err != nil {
		t.Fatal(err)
	}
	if content != "## Ringkasan eksekutif\nIsi" || model != DefaultModel {
		t.Fatalf("content %q model %q", content, model)
	}
	h := got.header
	if h.Get("Authorization") != "Bearer sk-test" || h.Get("HTTP-Referer") != "https://app.mixist.id" || h.Get("X-Title") != "Mixist Analytics" {
		t.Fatalf("headers %v", h)
	}
	b := got.body
	if b.Model != DefaultModel || b.MaxTokens != 2500 || b.Temperature != 0.4 || len(b.Messages) != 2 {
		t.Fatalf("body %+v", b)
	}
	if b.Messages[0].Content != SystemPrompts[domain.ModuleSeo] || !strings.Contains(b.Messages[0].Content, "## Peluang kata kunci") {
		t.Fatal("system prompt")
	}
	user := b.Messages[1].Content
	for _, want := range []string{
		"Klien: Kopi Nusantara\n",
		"Periode analisa: 2026-07-22 s.d. 2026-08-18 (28 hari), dibandingkan 2026-06-24 s.d. 2026-07-21.",
		"Write the body text in English, but KEEP the section headings exactly as specified (in Indonesian).",
		"Data KPI (JSON):\n{\"social\":{\"followers\":125400,",
	} {
		if !strings.Contains(user, want) {
			t.Errorf("user message lacks %q:\n%s", want, user)
		}
	}
}

func TestRealStatusMapping(t *testing.T) {
	cases := map[int]apperr.Code{401: apperr.TokenExpired, 403: apperr.Permission, 429: apperr.RateLimit, 404: apperr.NotFound, 500: apperr.Unknown}
	for status, code := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(strings.Repeat("x", 400)))
		}))
		_, err := NewReal(Credentials{APIKey: "k"}, Options{Endpoint: srv.URL}).Generate(context.Background(), input(domain.ModuleOverview, "id"))
		srv.Close()
		e := apperr.As(err)
		if e.Code != code || e.Provider != "openrouter" || e.Message != apperr.Message(code) {
			t.Errorf("status %d: %+v", status, e)
		}
	}
	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"   "}}]}`))
	}))
	defer empty.Close()
	_, err := NewReal(Credentials{APIKey: "k"}, Options{Endpoint: empty.URL}).Generate(context.Background(), input(domain.ModuleOverview, "id"))
	if apperr.As(err).Code != apperr.Unknown {
		t.Fatalf("empty content: %v", err)
	}
	_, err = NewReal(Credentials{APIKey: "k"}, Options{Endpoint: "http://127.0.0.1:1"}).Generate(context.Background(), input(domain.ModuleOverview, "id"))
	if apperr.As(err).Code != apperr.Network {
		t.Fatalf("network: %v", err)
	}
}
