package seodata

import (
	"context"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
)

var fixedClock = clock.Fixed{T: time.Date(2026, 10, 21, 8, 0, 0, 0, time.UTC)}

func TestMockKeywordIdeasDeterministic(t *testing.T) {
	m := NewMock(fixedClock)
	ctx := context.Background()
	a, _ := m.KeywordIdeas(ctx, "kopi susu", 2360, "id", domain.ModeIdeas, 30)
	b, _ := m.KeywordIdeas(ctx, "kopi susu", 2360, "id", domain.ModeIdeas, 30)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("ideas differ between calls")
	}
	if len(a) <= 10 || len(a) > 30 {
		t.Fatalf("got %d ideas", len(a))
	}
	for i, idea := range a {
		if len(idea.Trend) != 12 || idea.Difficulty < 0 || idea.Difficulty > 100 || idea.CPC < 500 {
			t.Fatalf("idea %+v", idea)
		}
		if i > 0 && a[i-1].Volume < idea.Volume {
			t.Fatal("not sorted by volume")
		}
	}
	q, _ := m.KeywordIdeas(ctx, "kopi susu", 2360, "id", domain.ModeQuestions, 30)
	re := regexp.MustCompile(`^(cara|apa itu|bagaimana)`)
	found := false
	for _, idea := range q {
		if re.MatchString(idea.Keyword) {
			found = true
		}
	}
	if !found {
		t.Fatal("no question-shaped ideas")
	}
}

func TestMockSerpRanks(t *testing.T) {
	m := NewMock(fixedClock)
	ctx := context.Background()
	req := []domain.SerpRankRequest{{Keyword: "kopi susu gula aren", Device: domain.DeviceMobile}}
	r1, _ := m.SerpRanks(ctx, req, 2360, "id", "kopinusantara.id")
	r2, _ := m.SerpRanks(ctx, req, 2360, "id", "kopinusantara.id")
	if !reflect.DeepEqual(r1, r2) || len(r1[0].Top10) != 10 {
		t.Fatalf("%+v", r1)
	}
	if r1[0].Position != nil && *r1[0].Position <= 10 {
		var inTop *int32
		for _, top := range r1[0].Top10 {
			if top.Domain == "kopinusantara.id" {
				v := top.Position
				inTop = &v
			}
		}
		if inTop == nil || *inTop != *r1[0].Position {
			t.Fatalf("target missing from top10: %+v", r1[0])
		}
	}
	// Brand keywords rank 1-3 on every day.
	for d := 0; d < 60; d++ {
		day := dates.AddDays(dates.UTC(2026, 1, 1), d)
		for _, dev := range []domain.Device{domain.DeviceMobile, domain.DeviceDesktop} {
			pos := MockPositionOn("kopi nusantara menu", RankKey("kopinusantara.id", dev), day)
			if pos == nil || *pos < 1 || *pos > 3 {
				t.Fatalf("brand keyword day %d device %s position %v", d, dev, pos)
			}
		}
	}
	if RankKey("x.id", domain.DeviceDesktop) != "x.id#d" || RankKey("x.id", domain.DeviceMobile) != "x.id" {
		t.Fatal("rankKey")
	}
	if a, b := MockPositionOn("kopi susu", "x.id", dates.UTC(2026, 5, 1)), MockPositionOn("kopi susu", "x.id", dates.UTC(2026, 5, 1)); !reflect.DeepEqual(a, b) {
		t.Fatal("position not stable")
	}
}

func TestMockDomainsAndBacklinks(t *testing.T) {
	m := NewMock(fixedClock)
	ctx := context.Background()
	own, _ := m.DomainKeywords(ctx, "kopinusantara.id", 2360, "id", 100)
	comp, _ := m.DomainKeywords(ctx, "kopikenangan.com", 2360, "id", 100)
	gap := domain.KeywordGapOf(own, comp)
	if len(gap.Shared) <= 5 || len(gap.Missing) == 0 {
		t.Fatalf("shared %d missing %d", len(gap.Shared), len(gap.Missing))
	}
	list, _ := m.Backlinks(ctx, "kopinusantara.id", 0, domain.BacklinksAll)
	var toxic, lost bool
	for _, b := range list {
		toxic = toxic || b.SpamScore >= 60
		lost = lost || b.IsLost
	}
	if !toxic || !lost || len(list) < 70 || len(list) > 120 {
		t.Fatalf("toxic %v lost %v n %d", toxic, lost, len(list))
	}
	lostOnly, _ := m.Backlinks(ctx, "kopinusantara.id", 0, domain.BacklinksLost)
	for _, b := range lostOnly {
		if !b.IsLost {
			t.Fatal("lost filter")
		}
	}
	sum, _ := m.BacklinkSummary(ctx, "kopinusantara.id")
	if sum.Backlinks <= 0 || sum.DomainRank < 18 || sum.ToxicShare < 0 || sum.ToxicShare > 1 {
		t.Fatalf("%+v", sum)
	}
	o, _ := m.DomainOverview(ctx, "kopinusantara.id", 2360, "id")
	if o.Top3 > o.Top10 || o.Top10 > o.Top100 || o.Backlinks != sum.Backlinks {
		t.Fatalf("%+v", o)
	}
	cs, _ := m.Competitors(ctx, "kopinusantara.id", 2360, "id")
	if len(cs) != 6 || cs[0].Intersections < cs[len(cs)-1].Intersections {
		t.Fatalf("%+v", cs)
	}
	metrics, _ := m.KeywordMetrics(ctx, []string{" Kopi Susu ", "villa bali"}, 2360, "id")
	if len(metrics) != 2 || metrics[0].Keyword != "kopi susu" {
		t.Fatalf("%+v", metrics)
	}
}

func TestPick(t *testing.T) {
	p := NewPicker(Options{Clock: fixedClock})
	if _, mode := p.Pick(Credentials{}); mode != ModeMock {
		t.Fatal(mode)
	}
	if _, mode := p.Pick(Credentials{ApifyToken: "t"}); mode != ModeApify {
		t.Fatal(mode)
	}
	if _, mode := p.Pick(Credentials{DataForSEOLogin: "l", DataForSEOPassword: "p", ApifyToken: "t"}); mode != ModeDataForSEO {
		t.Fatal(mode)
	}
	if _, mode := p.Pick(Credentials{DataForSEOLogin: "l"}); mode != ModeMock {
		t.Fatal("login without password is not configured")
	}
}
