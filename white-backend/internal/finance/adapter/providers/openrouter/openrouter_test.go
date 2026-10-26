package openrouter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

var input = app.InsightInput{From: "2026-07-01", To: "2026-09-30", Language: "id", Data: map[string]any{
	"ringkasan":  map[string]any{"pendapatan": 1_350_000_000.0, "labaBersih": 140_000_000.0, "kas": 1_000_000_000.0, "piutang": 210_000_000.0},
	"pembanding": map[string]any{"ringkasan": map[string]any{"pendapatan": 1_200_000_000.0}},
	"rasio":      map[string]any{"gross_margin": map[string]any{"value": 77.5}, "runway_months": map[string]any{"value": 4.2}},
}}

func TestMockIsDeterministic(t *testing.T) {
	a, b := Mock(input), Mock(input)
	if a != b {
		t.Fatal("mock must be deterministic")
	}
	for _, want := range []string{"## Ringkasan eksekutif", "## Kesehatan keuangan", "## Risiko", "## Rekomendasi 30 hari", "naik 12.5%", "Rp1.350.000.000", "gross margin 77.5%", "4.2 bulan. Ini di bawah batas aman 6 bulan."} {
		if !strings.Contains(a, want) {
			t.Errorf("mock lacks %q:\n%s", want, a)
		}
	}
}

func TestSourcePicksRealWithKey(t *testing.T) {
	var got chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"## Ringkasan eksekutif\nmemo"}}]}`))
	}))
	defer srv.Close()
	creds := func(context.Context) (Credentials, error) { return Credentials{APIKey: "sk-test"}, nil }
	content, model, err := NewSource(creds, Options{Endpoint: srv.URL}).Generate(context.Background(), input)
	if err != nil || model != DefaultModel || !strings.Contains(content, "memo") {
		t.Fatalf("%q %q %v", content, model, err)
	}
	if got.Messages[0].Content != SystemPrompt || !strings.Contains(got.Messages[1].Content, "Periode analisa: 2026-07-01 s.d. 2026-09-30") {
		t.Fatalf("prompt %+v", got.Messages)
	}
	none := func(context.Context) (Credentials, error) { return Credentials{}, nil }
	if _, model, _ := NewSource(none, Options{}).Generate(context.Background(), input); model != MockModel {
		t.Fatal("no key uses the mock")
	}
	bad := func(context.Context) (Credentials, error) { return Credentials{APIKey: "wrong"}, nil }
	if _, _, err := NewSource(bad, Options{Endpoint: srv.URL}).Generate(context.Background(), input); !apperr.Is(err, apperr.TokenExpired) {
		t.Fatalf("401 maps to TOKEN_EXPIRED: %v", err)
	}
}
