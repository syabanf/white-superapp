package pdf

import (
	"bytes"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
)

func TestRenderers(t *testing.T) {
	now := time.Date(2026, 10, 26, 3, 0, 0, 0, time.UTC)
	start := "2026-11-02"
	score := 80.0
	salary := int64(7_500_000)
	offer := domain.Offer{ID: "o1", CandidateName: "Dinda Pratiwi", Version: 2, Status: "accepted", PositionTitle: "SEO Analyst",
		BaseSalary: domain.Rupiah(8_000_000), Benefits: []string{"BPJS Kesehatan", "Laptop kerja"}, StartDate: &start,
		SentAt: &now, ExpiresAt: &now, RespondedAt: &now, CreatedByName: "Nadia (HR)"}
	report := domain.PipelineReport{CompanyName: "Mixist",
		Candidate:     domain.Candidate{FullName: "Dinda Pratiwi", Status: "offer", Source: "portal", ExpectedSalary: &salary, CreatedAt: now},
		PsikotesTests: []domain.SessionTest{{InstrumentName: "Aritmatika", Status: "selesai", Score: &score}},
		AiInterviews:  []domain.AiSession{{Status: "completed", Summary: &domain.InterviewSummary{Summary: "Jawaban rinci — relevan", RedFlags: []string{"x"}}}},
		Offers:        []domain.Offer{offer},
		Activities:    []domain.Activity{{Description: "Tahap diubah: Interview → Offer", CreatedAt: now}}}
	for name, render := range map[string]func() ([]byte, error){
		"report": func() ([]byte, error) { return Renderer{}.PipelineReport(report, now) },
		"letter": func() ([]byte, error) {
			return Renderer{}.OfferLetter(domain.OfferLetter{Offer: offer, CompanyName: "Mixist", IssuedAt: now})
		},
	} {
		b, err := render()
		if err != nil || !bytes.HasPrefix(b, []byte("%PDF-")) || len(b) < 1000 {
			t.Fatalf("%s: %v (%d bytes)", name, err, len(b))
		}
	}
	if got := LetterNumber("Mixist", 3, now); got != "OL-0003/MIXIST/X/2026" {
		t.Fatal(got)
	}
	if safe("A → B — “c”") != `A -> B - "c"` {
		t.Fatal(safe("A → B — “c”"))
	}
}
