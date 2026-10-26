package pdf

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/reports/domain"
)

func TestSafe(t *testing.T) {
	cases := [][2]string{
		{"−4,0%", "-4,0%"},
		{"1,3 rb", "1,3 rb"},
		{"a b c", "a b c"},
		{"a b c", "a\nb\nc"},
		{"Kopi 🚀 enak", "Kopi  enak"},
		{"€ “kutip” — ŽŸ…", "€ “kutip” — ŽŸ…"},
		{"日本", ""},
	}
	for _, c := range cases {
		if got := Safe(c[0]); got != c[1] {
			t.Errorf("Safe(%q) = %q, want %q", c[0], got, c[1])
		}
	}
}

func sample(summary string, modules int) domain.PdfData {
	good, bad := true, false
	all := []domain.PdfModule{
		{
			Key: "SOCIAL", Title: "Social Media",
			Kpis: []domain.PdfKpi{
				{Label: "Followers", Value: "125,4 rb", Delta: "+30,0%", DeltaGood: &good},
				{Label: "Engagement rate", Value: "4,32%", Delta: "−4,0%", DeltaGood: &bad},
				{Label: "Jangkauan", Value: "890 rb"},
				{Label: "Postingan", Value: "24", Delta: "0,0%"},
			},
			Chart: &domain.PdfChart{Title: "Pertumbuhan followers per hari (net)", MaxLabel: "+25/hari", Points: []domain.ChartPoint{{Date: "10 Agu", Value: 10}, {Date: "11 Agu", Value: -5}, {Date: "12 Agu", Value: 25}}},
			Tables: []domain.Table{{
				Title:   "Postingan teratas",
				Columns: []domain.Column{{Label: "Platform", Width: 14}, {Label: "Caption", Width: 44}, {Label: "Tipe", Width: 12}, {Label: "Suka", Width: 10, Align: "right"}, {Label: "Komentar", Width: 10, Align: "right"}, {Label: "Interaksi", Width: 10, Align: "right"}},
				Rows:    [][]string{{"Instagram", "Kopi single origin Gayo tiba di toko, stok terbatas untuk minggu ini saja 🚀", "Reels", "3.210", "148", "3.590"}},
			}},
			Sections: []domain.PdfSection{{Title: "Performa publikasi", Kpis: []domain.PdfKpi{{Label: "Postingan terbit", Value: "3"}}}},
		},
		{Key: "SEO", Title: "SEO", Kpis: []domain.PdfKpi{{Label: "Klik", Value: "8,4 rb"}}},
		{Key: "ADS", Title: "Meta Ads", Kpis: []domain.PdfKpi{{Label: "Belanja iklan", Value: "Rp 45,2 jt"}}},
	}
	return domain.PdfData{
		Lang: "id", Title: "Laporan Performa: Kopi Nusantara", ClientName: "Kopi Nusantara",
		PeriodLabel: "10 Agu – 12 Agu 2026", ComparedLabel: "vs periode sebelumnya: 7 Agu – 9 Agu 2026",
		GeneratedLabel: "19 Agu 2026, 14.05", Modules: all[:modules], AiSummary: summary,
	}
}

func pages(t *testing.T, pdf []byte) int {
	t.Helper()
	return len(regexp.MustCompile(`/Type /Page\b`).FindAll(pdf, -1))
}

func TestRenderPageCount(t *testing.T) {
	summary := "## Ringkasan eksekutif\nKlik **naik 12%**.\n- satu\n1. dua"
	cases := []struct {
		summary string
		modules int
		want    int
	}{
		{summary, 3, 1 + 3 + 1 + 1},
		{"", 3, 1 + 3 + 1},
		{"", 0, 2},
		{summary, 1, 4},
	}
	for _, c := range cases {
		out, err := Render(sample(c.summary, c.modules))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(out, []byte("%PDF")) {
			t.Fatal("not a PDF")
		}
		if got := pages(t, out); got != c.want {
			t.Errorf("modules=%d summary=%v: %d pages, want %d", c.modules, c.summary != "", got, c.want)
		}
	}
}

func TestRenderLongContentOverflows(t *testing.T) {
	long := strings.Repeat("- Poin panjang dengan **angka 12%** yang terus berlanjut sampai baris berikutnya.\n", 80)
	data := sample(long, 1)
	rows := make([][]string, 60)
	for i := range rows {
		rows[i] = []string{"Instagram", "Caption", "Foto", "1", "2", "3"}
	}
	data.Modules[0].Tables[0].Rows = rows
	out, err := Render(data)
	if err != nil {
		t.Fatal(err)
	}
	n := pages(t, out)
	if n <= 4 {
		t.Fatalf("long content should overflow, got %d pages", n)
	}
}
