package openrouter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
)

func sp(s string) *string { return &s }

func transcript() domain.Transcript {
	return domain.Transcript{CandidateName: "Dinda", PositionTitle: "SEO Analyst", MaxQuestions: 5, Turns: []domain.Turn{
		{TurnNo: 1, Topic: sp("pembuka"), Question: "Perkenalkan diri", Answer: sp("Saya Dinda, dua tahun mengelola SEO on-page dan riset kata kunci dengan Ahrefs.")},
		{TurnNo: 2, Topic: sp("keahlian"), Question: "Keahlian?", Answer: sp("Google Analytics dan Search Console setiap hari, juga laporan di Excel.")},
		{TurnNo: 3, Topic: sp("motivasi"), Question: "Motivasi?", Answer: sp("Suka")},
		{TurnNo: 4, Topic: sp("gaji"), Question: "Gaji?", Answer: sp("Sekitar Rp 7 juta sampai 8 juta per bulan.")},
	}}
}

func TestMockSummary(t *testing.T) {
	s := MockSummary(transcript())
	if !strings.HasPrefix(s.Summary, "Dinda menjawab 4 dari 4 pertanyaan interview AI untuk posisi SEO Analyst.") {
		t.Fatal(s.Summary)
	}
	if strings.Join(s.Skills, ",") != "SEO,Google Analytics,Search Console,Ahrefs,Excel" {
		t.Fatal(s.Skills)
	}
	if !s.Salary.Mentioned || s.Salary.Value == nil || *s.Salary.Value != "Rp 7 juta sampai 8 juta" {
		t.Fatalf("%+v", s.Salary)
	}
	if len(s.RedFlags) != 1 || s.RedFlags[0] != "Jawaban #3 sangat singkat" {
		t.Fatal(s.RedFlags)
	}
	// 40 + 8*3 long answers + 3*5 skills - 10*1 flag
	if s.Relevance.Score != 69 || s.Relevance.Conclusion != "cukup_relevan" {
		t.Fatalf("%+v", s.Relevance)
	}
	if again := MockSummary(transcript()); again.Summary != s.Summary || again.Relevance != s.Relevance {
		t.Fatal("the mock is deterministic")
	}
	empty := MockSummary(domain.Transcript{CandidateName: "X"})
	if empty.Salary.Mentioned || empty.Salary.Note != "Topik gaji belum tertanya." || len(empty.FollowUps) != 4 {
		t.Fatalf("%+v", empty)
	}
}

func TestMockAnalysis(t *testing.T) {
	job := "Posisi: SEO Analyst\nPersyaratan: Menguasai Google Analytics, Search Console dan Ahrefs"
	a := MockAnalysis(domain.AnalysisInput{Profile: "Nama: Dinda\nPendidikan terakhir: S1 Ilmu Komunikasi\nPengalaman terakhir: SEO specialist, Ahrefs dan Google Analytics", JobContext: job})
	if a.MatchScore < 60 || len(a.Strengths) != 2 || len(a.Gaps) != 0 {
		t.Fatalf("%+v", a)
	}
	bare := MockAnalysis(domain.AnalysisInput{Profile: "Nama: Budi"})
	if bare.MatchScore != 35 || bare.MatchReason != "Deskripsi pekerjaan tidak tersedia, skor dinilai dari kelengkapan profil." || len(bare.Gaps) != 3 {
		t.Fatalf("%+v", bare)
	}
}

func TestParseJSONObject(t *testing.T) {
	got, err := ParseJSONObject("Berikut:\n```json\n{\"action\":\"ask\",\"question\":\"Q\"}\n```")
	if err != nil || got["action"] != "ask" {
		t.Fatal(got, err)
	}
	if _, err := ParseJSONObject("tidak ada"); err == nil {
		t.Fatal("garbage")
	}
}

func server(t *testing.T, status int, content string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer k" {
			t.Errorf("auth header %q", r.Header.Get("Authorization"))
		}
		var req chatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Messages) != 2 || req.Model != "m" {
			t.Errorf("request %+v", req)
		}
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": content}}}})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func source(endpoint string, err error) *Source {
	return NewSource(func(context.Context) (Credentials, error) { return Credentials{APIKey: "k", Model: "m"}, err },
		Options{Endpoint: endpoint})
}

func TestRealInterviewStepAndSummary(t *testing.T) {
	srv := server(t, 200, `{"action":"ask","topic":"keahlian","question":"Tools SEO apa yang Anda pakai?"}`)
	step := source(srv.URL, nil).InterviewStep(context.Background(), transcript())
	if step == nil || step.Action != "ask" || *step.Topic != "keahlian" || *step.Question != "Tools SEO apa yang Anda pakai?" {
		t.Fatalf("%+v", step)
	}
	srv = server(t, 200, `{"ringkasan":"Ok","relevansi":{"skor":101.4,"kesimpulan":"aneh","alasan":"x"},"keahlian":["SEO"],"ekspektasi_gaji":{"disebutkan":true,"nilai":"8 juta","catatan":""},"red_flags":[],"perhatikan_saat_interview_lanjutan":["a","b"]}`)
	sum, model, err := source(srv.URL, nil).SummarizeInterview(context.Background(), transcript())
	if err != nil || model != "m" || sum.Relevance.Score != 100 || sum.Relevance.Conclusion != "cukup_relevan" || *sum.Salary.Value != "8 juta" ||
		sum.Limitations != domain.DefaultLimitations {
		t.Fatalf("%v %s %+v", err, model, sum)
	}
}

func TestSourceFallsBackToMock(t *testing.T) {
	srv := server(t, 500, "")
	s := source(srv.URL, nil)
	if s.InterviewStep(context.Background(), transcript()) != nil {
		t.Fatal("a failing step is nil so the topic order takes over")
	}
	if _, model, err := s.SummarizeInterview(context.Background(), transcript()); err != nil || model != MockModel {
		t.Fatal(model, err)
	}
	if _, model, _ := source("", errors.New("integrations down")).AnalyzeCandidate(context.Background(), domain.AnalysisInput{}); model != MockModel {
		t.Fatal(model)
	}
	noKey := NewSource(func(context.Context) (Credentials, error) { return Credentials{}, nil }, Options{})
	if noKey.InterviewStep(context.Background(), transcript()) != nil {
		t.Fatal("no key, no AI step")
	}
}
