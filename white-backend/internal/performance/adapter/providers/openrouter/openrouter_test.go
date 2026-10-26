package openrouter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
)

func f(v float64) *float64 { return &v }

func monthly() domain.MonthlyRecapInput {
	return domain.MonthlyRecapInput{Scope: domain.ScopeEmployee, SubjectName: "Rani", DepartmentName: "Social Media",
		Period:          domain.Period{Year: 2026, Month: 9},
		Metrics:         domain.MonthlyMetrics{TasksPlanned: 20, TasksDone: 15, TasksOnTime: 10, TasksOverdue: 2, CompletionRate: f(0.75), OnTimeRate: f(0.6667), KPIScore: f(72.5)},
		Accomplishments: []string{"Posting konten harian klien (15x)"}}
}

func TestMockDailySummary(t *testing.T) {
	if got := MockDailySummary(domain.DailyRecapInput{EmployeeName: "Rani"}); got != "Belum ada aktivitas tugas atau logbook yang tercatat hari ini." {
		t.Fatal(got)
	}
	got := MockDailySummary(domain.DailyRecapInput{EmployeeName: "Rani", Notes: "Meeting klien kopi.", Items: domain.DailyRecapItems{
		Tasks:   []domain.RecapTask{{Title: "Posting", Completed: true}, {Title: "Laporan", Progress: 40}},
		Logbook: []domain.RecapLogbook{{Title: "Checklist", Completion: 75, Status: "submitted"}}}})
	want := "Hari ini Rani menyelesaikan 1 tugas: Posting. Masih berjalan: Laporan (40%). Logbook Checklist terisi 75% (status submitted). Catatan: Meeting klien kopi."
	if got != want {
		t.Fatalf("%q", got)
	}
}

func TestMockMonthlyAndSuggestions(t *testing.T) {
	text := MockMonthlyNarrative(monthly())
	if !strings.Contains(text, "menyelesaikan 15 dari 20 tugas terjadwal (75%)") || !strings.Contains(text, "Skor KPI tercatat 72,5") ||
		!strings.Contains(text, "2 tugas masih terlambat") {
		t.Fatal(text)
	}
	s := MockSuggestions(monthly())
	if len(s) != 4 || s[0].Title != "Tuntaskan 2 tugas yang terlambat" || s[3].Title != "Susun rencana peningkatan KPI bersama atasan" {
		t.Fatalf("%+v", s)
	}
	healthy := domain.MonthlyRecapInput{Metrics: domain.MonthlyMetrics{OnTimeRate: f(0.95), CompletionRate: f(0.95)}}
	if s := MockSuggestions(healthy); len(s) != 1 || !strings.HasPrefix(s[0].Title, "Dokumentasikan") {
		t.Fatalf("%+v", s)
	}
}

func TestParseSuggestions(t *testing.T) {
	got, err := ParseSuggestions("Berikut saran:\n```json\n[{\"title\":\" Kirim laporan Jumat \",\"description\":\"x\"},{\"title\":\"\"}]\n```")
	if err != nil || len(got) != 1 || got[0].Title != "Kirim laporan Jumat" {
		t.Fatalf("%v %+v", err, got)
	}
	if _, err := ParseSuggestions("tidak ada"); err == nil {
		t.Fatal("garbage")
	}
}

func TestRealAndFallback(t *testing.T) {
	var gotAuth, gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		var req chatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		gotModel = req.Model
		content := "Rekap singkat."
		if strings.Contains(req.Messages[0].Content, "JSON array") {
			content = `[{"title":"Rapikan jadwal posting","description":"Supaya tepat waktu."}]`
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": content}}}})
	}))
	defer srv.Close()
	src := NewSource(func(context.Context) (Credentials, error) { return Credentials{APIKey: "k", Model: "m/x"}, nil },
		Options{Endpoint: srv.URL})
	text, model, err := src.DailySummary(context.Background(), domain.DailyRecapInput{EmployeeName: "Rani", Date: "2026-10-26"})
	if err != nil || text != "Rekap singkat." || model != "m/x" || gotAuth != "Bearer k" || gotModel != "m/x" {
		t.Fatalf("%q %q %v %q", text, model, err, gotAuth)
	}
	list, _, err := src.SuggestImprovements(context.Background(), monthly())
	if err != nil || len(list) != 1 || list[0].Title != "Rapikan jadwal posting" {
		t.Fatalf("%v %+v", err, list)
	}

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTooManyRequests) }))
	defer failing.Close()
	src = NewSource(func(context.Context) (Credentials, error) { return Credentials{APIKey: "k"}, nil }, Options{Endpoint: failing.URL})
	text, model, err = src.MonthlyNarrative(context.Background(), monthly())
	if err != nil || model != MockModel || text != MockMonthlyNarrative(monthly()) {
		t.Fatalf("fallback %q %v", model, err)
	}
	noCreds := NewSource(func(context.Context) (Credentials, error) { return Credentials{}, errors.New("down") }, Options{})
	if _, model, _ := noCreds.DailySummary(context.Background(), domain.DailyRecapInput{}); model != MockModel {
		t.Fatal(model)
	}
}
