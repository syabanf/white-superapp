package pdf

import (
	"bytes"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
)

func TestRenderProducesPDF(t *testing.T) {
	m, _ := domain.ParseMonth("2026-09")
	pub := time.Date(2026, 9, 3, 4, 0, 0, 0, time.UTC)
	post := domain.ReportPost{ItemID: "i1", Title: "Promo kopi — edisi “spesial”", Platform: domain.PlatformInstagram, Format: domain.FormatReel, PlannedDate: "2026-09-03", PublishedAt: &pub, HasMetrics: true, Engagements: 1234, Reach: 56789}
	r := domain.Report{Month: m, UpdatedAt: pub, Summary: "## Ringkasan\nBagus **sekali**.\n\n## Rekomendasi\n- satu\n1. dua", Data: domain.ReportData{
		ClientName: "Kopi Senja", Planned: 10, Published: 8, OnTimeRate: 87.5, EngagementAvailable: true,
		PerPlatform: []domain.Breakdown{{Key: "INSTAGRAM", Planned: 10, Published: 8}}, Posts: []domain.ReportPost{post}, Top: []domain.ReportPost{post},
	}}
	out, err := Renderer{}.Render(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) || len(out) < 1000 {
		t.Fatalf("not a pdf (%d bytes)", len(out))
	}
}

func TestHelpers(t *testing.T) {
	if thousands(1234567) != "1.234.567" || thousands(-1000) != "-1.000" || thousands(12) != "12" {
		t.Fatal("thousands")
	}
	if safe("a—b “c” ✓") != "a-b \"c\" " {
		t.Fatalf("%q", safe("a—b “c” ✓"))
	}
}
