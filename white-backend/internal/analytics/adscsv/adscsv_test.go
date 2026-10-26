package adscsv

import (
	"strings"
	"testing"
)

func s(v string) *string { return &v }

func TestParseNumber(t *testing.T) {
	cases := map[string]*float64{}
	want := func(in string, v float64) { cases[in] = &v }
	want("1.234.567,89", 1234567.89)
	want("1,234,567.89", 1234567.89)
	want("1.234", 1234)
	want("1,234", 1234)
	want("12,5", 12.5)
	want("12.5", 12.5)
	want("Rp 1.500", 1500)
	want("-3,5", -3.5)
	cases["--"] = nil
	cases["–"] = nil
	cases["1.2.3"] = nil
	cases["1,2,3"] = nil
	cases[""] = nil
	for in, w := range cases {
		got := ParseNumber(s(in))
		if (got == nil) != (w == nil) || (got != nil && *got != *w) {
			t.Errorf("%q: got %v want %v", in, deref(got), deref(w))
		}
	}
}

func deref(f *float64) any {
	if f == nil {
		return nil
	}
	return *f
}

func TestParseDate(t *testing.T) {
	cases := map[string]string{
		"2026-9-1":    "2026-09-01",
		"2026-09-01T": "2026-09-01",
		"05/09/2026":  "2026-09-05",
		"09/25/2026":  "2026-09-25",
		"5.9.2026":    "2026-09-05",
		"31/02/2026":  "",
		"garbage":     "",
	}
	for in, w := range cases {
		got := ParseDate(s(in))
		if (got == nil && w != "") || (got != nil && *got != w) {
			t.Errorf("%q: got %v want %q", in, got, w)
		}
	}
}

func TestNormalizers(t *testing.T) {
	for in, w := range map[string]string{"Pria": "male", "perempuan": "female", "F": "female", "x": "unknown"} {
		if g := NormalizeGender(s(in)); g == nil || *g != w {
			t.Errorf("gender %q -> %v", in, g)
		}
	}
	if NormalizeGender(s("Semua")) != nil || normalizeAge(s("all")) != nil {
		t.Fatal("all means no breakdown")
	}
	if a := normalizeAge(s("Tidak diketahui")); *a != "unknown" {
		t.Fatal("age unknown")
	}
	for in, w := range map[string]string{
		"actions:offsite_conversion.fb_pixel_purchase": "purchase",
		"Percakapan dimulai":                           "messaging_conversation_started",
		"Prospek":                                      "lead",
		"Link clicks":                                  "link_click",
		"ThruPlay":                                     "thruplay",
		"Jangkauan":                                    "reach",
		"Tayangan":                                     "impressions",
		"actions:custom.Some Event":                    "some_event",
	} {
		if r := NormalizeResultType(s(in)); r == nil || *r != w {
			t.Errorf("result %q -> %v", in, r)
		}
	}
}

const fixture = "\ufeffNama kampanye,Nama set iklan,Nama iklan,Hari,Jumlah yang dibelanjakan (IDR),Tayangan,Jangkauan,Klik tautan,Hasil,Indikator hasil,Usia,Jenis kelamin\n" +
	"Promo Kopi,Broad,Video A,01/09/2026,\"150.000\",\"12.000\",8000,240,24,Link clicks,,\n" +
	"Promo Kopi,Broad,Video A,02/09/2026,\"100.000\",\"10.000\",7000,200,20,Link clicks,,\n" +
	"Promo Kopi,,,,\"50.000\",4000,3000,80,8,Link clicks,25-34,Wanita\n" +
	",,,,\"300.000\",,,,,,,\n" +
	"Promo Kopi,Broad,Video A,,\"1\",1,1,1,1,,,\n"

func TestParseFixture(t *testing.T) {
	r := Parse(fixture)
	if len(r.Rows) != 3 {
		t.Fatalf("rows %d %+v", len(r.Rows), r.Warnings)
	}
	d := r.Rows[0]
	if d.Kind != "daily" || *d.Date != "2026-09-01" || d.Spend != 150000 || d.Impressions != 12000 || *d.ResultType != "link_click" {
		t.Fatalf("%+v", d)
	}
	demo := r.Rows[2]
	if demo.Kind != "demographic" || *demo.Gender != "female" || *demo.Age != "25-34" || demo.Date != nil {
		t.Fatalf("%+v", demo)
	}
	if len(r.Warnings) != 1 || !strings.Contains(r.Warnings[0], "1 baris harian dilewati") {
		t.Fatalf("warnings %v", r.Warnings)
	}
}

func TestParseMissingColumns(t *testing.T) {
	r := Parse("Foo,Bar\n1,2\n")
	if len(r.Rows) != 0 || !strings.HasPrefix(r.Warnings[0], "Kolom wajib tidak ditemukan: Campaign name / Nama kampanye, Amount spent") {
		t.Fatalf("%+v", r)
	}
	if det := DetectColumns([]string{"Campaign name", "Amount spent (USD)"}); det[FSpend] != "Amount spent (USD)" {
		t.Fatal("prefix match for spend")
	}
}
