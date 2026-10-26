package domain

import (
	"reflect"
	"testing"
)

func TestParseMarkdownLite(t *testing.T) {
	md := "## Ringkasan eksekutif\r\nKlik **naik 12%** bulan ini.\n\n- satu\n  * dua\n• tiga\n1. pertama\n2) kedua\n#### bukan heading\n**Bold** awal"
	got := ParseMarkdownLite(md)
	want := []Block{
		{Kind: BlockHeading, Spans: []Span{{Text: "Ringkasan eksekutif"}}},
		{Kind: BlockParagraph, Spans: []Span{{Text: "Klik "}, {Text: "naik 12%", Bold: true}, {Text: " bulan ini."}}},
		{Kind: BlockBullet, Spans: []Span{{Text: "satu"}}},
		{Kind: BlockBullet, Spans: []Span{{Text: "dua"}}},
		{Kind: BlockBullet, Spans: []Span{{Text: "tiga"}}},
		{Kind: BlockNumbered, Number: "1", Spans: []Span{{Text: "pertama"}}},
		{Kind: BlockNumbered, Number: "2", Spans: []Span{{Text: "kedua"}}},
		{Kind: BlockParagraph, Spans: []Span{{Text: "#### bukan heading"}}},
		{Kind: BlockParagraph, Spans: []Span{{Text: "Bold", Bold: true}, {Text: " awal"}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

func TestInlineBoldKeepsUnclosedMarkers(t *testing.T) {
	if got := InlineBold("a ** b"); !reflect.DeepEqual(got, []Span{{Text: "a ** b"}}) {
		t.Fatalf("got %+v", got)
	}
}
