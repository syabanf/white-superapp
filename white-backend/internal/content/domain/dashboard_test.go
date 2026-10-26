package domain

import (
	"testing"
	"time"
)

func date(s string) time.Time {
	t, _ := time.Parse(time.DateOnly, s)
	return t
}

func instant(s string) *time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return &t
}

func TestCompareWindow(t *testing.T) {
	for _, c := range []struct {
		mode, from, to, wantFrom, wantTo string
	}{
		{ComparePrevious, "2026-10-01", "2026-10-28", "2026-09-03", "2026-09-30"},
		{CompareYear, "2026-10-01", "2026-10-28", "2025-10-01", "2025-10-28"},
		{"", "2026-10-26", "2026-10-26", "2026-10-25", "2026-10-25"},
	} {
		f, to := CompareWindow(date(c.from), date(c.to), c.mode)
		if f.Format(time.DateOnly) != c.wantFrom || to.Format(time.DateOnly) != c.wantTo {
			t.Errorf("%s %s..%s: got %s..%s", c.mode, c.from, c.to, f.Format(time.DateOnly), to.Format(time.DateOnly))
		}
	}
}

func TestComputeDashboard(t *testing.T) {
	copyw, design := "u_copy", "u_design"
	due := func(s string) *time.Time { d := date(s); return &d }
	items := []DashItem{
		{ // published on time, one revision
			ID: "a", PlannedDate: date("2026-10-06"), Platform: PlatformInstagram, Format: FormatReel, Pillar: "Promo", Status: ItemPublished, RevisionRound: 1,
			CopywriterID: &copyw, DesignerID: &design, CopyDue: due("2026-10-02"), DesignDue: due("2026-10-04"), ReviewDue: due("2026-10-05"),
			CreatedAt: *instant("2026-09-28T02:00:00Z"), CopyDoneAt: instant("2026-10-01T09:00:00Z"), DesignDoneAt: instant("2026-10-03T07:00:00Z"),
			ApprovedAt: instant("2026-10-05T04:00:00Z"), PublishedAt: instant("2026-10-06T11:00:00Z"), ThumbnailFileID: "f1",
		},
		{ // copy overdue
			ID: "b", PlannedDate: date("2026-10-20"), Platform: PlatformFacebook, Format: FormatFeed, Pillar: "Edukasi", Status: ItemCopywriting,
			CopywriterID: &copyw, DesignerID: &design, CopyDue: due("2026-10-15"), CreatedAt: *instant("2026-10-10T02:00:00Z"),
		},
		{ // in review, not late
			ID: "c", PlannedDate: date("2026-10-28"), Platform: PlatformInstagram, Format: FormatFeed, Pillar: "Promo", Status: ItemRevision, RevisionRound: 3,
			CopywriterID: &copyw, ReviewDue: due("2026-10-27"), CreatedAt: *instant("2026-10-12T02:00:00Z"), CopyDoneAt: instant("2026-10-14T02:00:00Z"),
			DesignDoneAt: instant("2026-10-16T02:00:00Z"),
		},
	}
	tasks := []DashDesignTask{
		{ID: "t1", ItemID: "a", DesignerID: &design, Versions: 2, CreatedAt: *instant("2026-10-01T09:00:00Z"), ApprovedAt: instant("2026-10-03T09:00:00Z"), ChangesRequested: 1},
		{ID: "t2", ItemID: "c", DesignerID: &design, Versions: 1, CreatedAt: *instant("2026-10-14T02:00:00Z"), ApprovedAt: instant("2026-10-15T02:00:00Z")},
	}
	assets := []DashAssetVersion{
		{AssetID: "logo", Name: "Logo", Kind: AssetLogo, CurrentVersion: 2, Version: 2, CreatedAt: *instant("2026-10-02T03:00:00Z")},
		{AssetID: "logo", Name: "Logo", Kind: AssetLogo, CurrentVersion: 2, Version: 1, CreatedAt: *instant("2026-10-01T03:00:00Z")},
		{AssetID: "photo", Name: "Foto", Kind: AssetPhoto, CurrentVersion: 1, Version: 1, CreatedAt: *instant("2026-10-09T03:00:00Z")},
	}
	in := DashboardInput{From: date("2026-10-01"), To: date("2026-10-28"), Items: items, Tasks: tasks, Assets: assets,
		Metrics: map[string]PostMetrics{"a": {Likes: 80, Comments: 10, Shares: 5, Saves: 5, Reach: 2000}}}
	d := ComputeDashboard(in, *instant("2026-10-26T03:00:00Z"))

	k := d.Kpis
	if k.Planned != 3 || k.Published != 1 || k.InReview != 1 || k.Overdue != 1 || k.Engagements != 100 || k.EngagementRate != 5 || k.MeasuredPosts != 1 {
		t.Fatalf("kpis %+v", k)
	}
	if k.RevisionAvg != 1.3 || k.DesignApproved != 2 || k.FirstPassRate != 50 || k.DesignTurnaroundDays != 1.5 || k.AssetUploads != 3 {
		t.Fatalf("kpis %+v", k)
	}
	// a: 4 deadlines met; b: copy due and planned day passed and missed. c: nothing due yet.
	if k.DeadlinesMet != 4 || k.DeadlinesTotal != 6 || k.OnTimeRate != 66.7 {
		t.Fatalf("deadlines %d/%d %.1f", k.DeadlinesMet, k.DeadlinesTotal, k.OnTimeRate)
	}
	if d.Granularity != "week" || len(d.Buckets) != 5 || d.Buckets[0].Start.Format(time.DateOnly) != "2026-09-28" {
		t.Fatalf("buckets %s %d %v", d.Granularity, len(d.Buckets), d.Buckets[0].Start)
	}
	if b := d.Buckets[1]; b.Published != 1 || b.Engagements != 100 || b.Planned != 1 {
		t.Fatalf("week 2 %+v", b)
	}
	// End of week 3 (Oct 18): b in copy, c in review.
	if b := d.Buckets[2]; b.StageCopy != 1 || b.StageReview != 1 || b.StageReady != 0 {
		t.Fatalf("week 3 stages %+v", b)
	}
	want := []int32{3, 3, 2, 2, 1, 1, 1}
	for i, c := range d.Funnel {
		if c.Count != want[i] {
			t.Fatalf("funnel %+v", d.Funnel)
		}
	}
	if d.OverdueByStage[0].Count != 1 || d.RevisionRounds[1].Count != 1 || d.RevisionRounds[3].Count != 1 {
		t.Fatalf("overdue %+v rounds %+v", d.OverdueByStage, d.RevisionRounds)
	}
	if len(d.Posts) != 1 || d.Posts[0].ThumbnailFileID != "f1" || len(d.PostingHeat) != 1 {
		t.Fatalf("posts %+v heat %+v", d.Posts, d.PostingHeat)
	}
	// 11:00Z on a Tuesday is 18:00 WIB.
	if c := d.PostingHeat[0]; c.Weekday != 1 || c.Hour != 18 || c.Engagements != 100 {
		t.Fatalf("heat %+v", c)
	}
	if d.ByPlatform[0].Key != "INSTAGRAM" || d.ByPlatform[0].Planned != 2 || d.ByPillar[0].Engagements != 100 {
		t.Fatalf("breakdowns %+v %+v", d.ByPlatform, d.ByPillar)
	}
	if d.TopAssets[0].AssetID != "logo" || d.TopAssets[0].Uploads != 2 || len(d.AssetUploadsKind) != 2 {
		t.Fatalf("assets %+v %+v", d.TopAssets, d.AssetUploadsKind)
	}
	var cw *DashPerson
	for i := range d.People {
		if d.People[i].UserID == copyw && d.People[i].Role == "COPYWRITER" {
			cw = &d.People[i]
		}
	}
	if cw == nil || cw.Assigned != 3 || cw.Overdue != 1 || cw.OverdueCopy != 1 || cw.Published != 1 {
		t.Fatalf("people %+v", d.People)
	}
	if len(d.Days) != 28 || d.Days[5].Planned != 1 || d.Days[5].Published != 1 {
		t.Fatalf("days %+v", d.Days[5])
	}
}

func TestGranularity(t *testing.T) {
	if Granularity(date("2026-01-01"), date("2026-04-30")) != "week" || Granularity(date("2025-10-27"), date("2026-10-26")) != "month" {
		t.Fatal("granularity")
	}
}
