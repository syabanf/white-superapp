package domain

import (
	"testing"
	"time"
)

func TestBestPostingTimes(t *testing.T) {
	cells := []HeatCell{
		{Day: 1, Hour: 9, Posts: 5, AvgEngagement: 100},
		{Day: 1, Hour: 18, Posts: 5, AvgEngagement: 90},
		{Day: 3, Hour: 12, Posts: 3, AvgEngagement: 80},
		{Day: 5, Hour: 20, Posts: 1, AvgEngagement: 999}, // below minPosts
	}
	best := BestPostingTimes(cells, 3, 2)
	want := [][2]int{{1, 9}, {3, 12}, {1, 18}}
	for i, b := range best {
		if b.Day != want[i][0] || b.Hour != want[i][1] {
			t.Fatalf("%+v", best)
		}
	}
	def := BestPostingTimes(nil, 3, 2)
	if len(def) != 3 || def[0] != DefaultBestTimes[0] || def[2] != DefaultBestTimes[2] {
		t.Fatalf("%+v", def)
	}
	one := BestPostingTimes(cells[:1], 3, 2)
	if one[0].Score == 0 || one[1] != DefaultBestTimes[0] || one[2] != DefaultBestTimes[1] {
		t.Fatalf("top-up: %+v", one)
	}
}

func TestPostingHeatmap(t *testing.T) {
	// 2026-09-02 10:00 WIB (Wednesday, day 2).
	at := time.Date(2026, 9, 2, 3, 0, 0, 0, time.UTC)
	cells := PostingHeatmap([]EngagedPost{{at, 10}, {at, 30}, {at.Add(24 * time.Hour), 5}}, "Asia/Jakarta")
	if len(cells) != 168 {
		t.Fatal(len(cells))
	}
	c := cells[2*24+10]
	if c.Day != 2 || c.Hour != 10 || c.Posts != 2 || c.AvgEngagement != 20 {
		t.Fatalf("%+v", c)
	}
	if cells[3*24+10].Posts != 1 {
		t.Fatal("thursday")
	}
}

func TestCountsAndFailureRate(t *testing.T) {
	counts := PostStatusCounts([]PostStatus{StatusDraft, StatusDraft, StatusPublished, "BOGUS"})
	if counts[StatusDraft] != 2 || counts[StatusFailed] != 0 || len(counts) != 8 {
		t.Fatalf("%v", counts)
	}
	if r := FailureRate([]PostStatus{StatusPublished, StatusFailed, StatusScheduled}); r == nil || *r != 50 {
		t.Fatal("rate")
	}
	if FailureRate([]PostStatus{StatusDraft}) != nil {
		t.Fatal("nil rate")
	}
}

func TestSummarizePublishing(t *testing.T) {
	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	e1, e2 := int64(10), int64(50)
	rows := []PublishedTarget{
		{PostID: "a", Platform: PlatformInstagram, PublishedAt: base},
		{PostID: "a", Platform: PlatformFacebook, PublishedAt: base.Add(time.Hour)},
		{PostID: "b", Platform: PlatformInstagram, PublishedAt: base.Add(2 * time.Hour)},
	}
	s := SummarizePublishing(rows, 2, 5)
	if s.Posts != 2 || s.Failed != 2 || s.Mode != "recent" || len(s.Top) != 3 || s.Top[0].PostID != "b" {
		t.Fatalf("%+v", s)
	}
	if s.PerPlatform[0].Platform != PlatformInstagram || s.PerPlatform[0].Count != 2 || s.PerPlatform[1].Platform != PlatformFacebook {
		t.Fatalf("%+v", s.PerPlatform)
	}
	rows[0].Engagements = &e1
	rows[2].Engagements = &e2
	s = SummarizePublishing(rows, 0, 1)
	if s.Mode != "engagement" || len(s.Top) != 1 || s.Top[0].PostID != "b" {
		t.Fatalf("%+v", s)
	}
}
