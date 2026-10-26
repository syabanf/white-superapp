package domain

import (
	"strings"
	"testing"
	"time"
)

func day(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

func dp(s string) *time.Time { t := day(s); return &t }

func TestMonth(t *testing.T) {
	m, ok := ParseMonth("2026-02")
	if !ok || m.String() != "2026-02" || m.Last().Day() != 28 || m.Label() != "Februari 2026" {
		t.Fatalf("%v %v", m, ok)
	}
	if !m.Contains(day("2026-02-28")) || m.Contains(day("2026-03-01")) || m.Contains(day("2026-01-31")) {
		t.Fatal("contains")
	}
	if m.Next().String() != "2026-03" || m.Prev().String() != "2026-01" {
		t.Fatal("next/prev")
	}
	for _, bad := range []string{"", "2026-13", "2026/02", "26-02"} {
		if _, ok := ParseMonth(bad); ok {
			t.Fatal(bad)
		}
	}
	// 23:30 UTC is already the next day in Jakarta.
	if Day(time.Date(2026, 10, 26, 23, 30, 0, 0, time.UTC)) != day("2026-10-27") {
		t.Fatal("agency day")
	}
}

func TestItemTransitions(t *testing.T) {
	mgr := MoveCtx{Role: RoleManager}
	author := MoveCtx{Role: RoleManager, IsCopywriter: true}
	admin := MoveCtx{Role: RoleAdmin, IsCopywriter: true}
	viewer := MoveCtx{Role: RoleViewer}
	cases := []struct {
		from, to ItemStatus
		ctx      MoveCtx
		want     bool
	}{
		{ItemIdea, ItemCopywriting, author, true},
		{ItemIdea, ItemReview, mgr, false},
		{ItemCopywriting, ItemReview, author, true},
		{ItemReview, ItemApproved, author, false},
		{ItemReview, ItemApproved, admin, true},
		{ItemReview, ItemApproved, mgr, true},
		{ItemReview, ItemRevision, author, false},
		{ItemRevision, ItemReview, author, true},
		{ItemApproved, ItemRevision, mgr, true},
		{ItemApproved, ItemScheduled, admin, false},
		{ItemFailed, ItemApproved, author, true},
		{ItemPublished, ItemRevision, admin, false},
		{ItemIdea, ItemCopywriting, viewer, false},
	}
	for _, c := range cases {
		if got := CanMove(c.from, c.to, c.ctx); got != c.want {
			t.Errorf("%s -> %s %+v = %v", c.from, c.to, c.ctx, got)
		}
	}
	if got := AllowedMoves(ItemReview, author); len(got) != 0 {
		t.Fatal(got)
	}
	if !IsLocked(ItemScheduled) || IsLocked(ItemFailed) || !CanSchedule(ItemFailed) || CanSchedule(ItemReview) {
		t.Fatal("lock/schedule")
	}
}

func TestMoveStamps(t *testing.T) {
	now := time.Date(2026, 11, 1, 2, 0, 0, 0, time.UTC)
	it := Item{Status: ItemCopywriting}
	MoveStamps(&it, ItemDesign, "u1", now)
	MoveStamps(&it, ItemReview, "u2", now.Add(time.Hour))
	if it.CopyDoneAt == nil || !it.CopyDoneAt.Equal(now) || it.DesignDoneAt == nil {
		t.Fatalf("%+v", it)
	}
	MoveStamps(&it, ItemRevision, "lead", now)
	if it.RevisionRound != 1 || it.Status != ItemRevision {
		t.Fatalf("%+v", it)
	}
	MoveStamps(&it, ItemReview, "u1", now.Add(2*time.Hour))
	if !it.CopyDoneAt.Equal(now) {
		t.Fatal("copy stamp overwritten")
	}
	MoveStamps(&it, ItemApproved, "lead", now)
	if it.ApprovedBy == nil || *it.ApprovedBy != "lead" {
		t.Fatal("approval")
	}
	MoveStamps(&it, ItemRevision, "lead", now)
	if it.ApprovedAt != nil || it.RevisionRound != 2 {
		t.Fatal("reopen keeps approval")
	}
}

func TestCaptions(t *testing.T) {
	got := NormalizeHashtags([]string{" #Kopi ", "kopi", "kopi susu", "", "##", "#Promo"})
	if strings.Join(got, " ") != "#Kopi #kopisusu #Promo" {
		t.Fatal(got)
	}
	if ComposeCaption(FormatFeed, " Halo ", []string{"#a", "#b"}) != "Halo\n\n#a #b" || ComposeCaption(FormatStory, "x", nil) != "" {
		t.Fatal("compose")
	}
	if ComposeCaption(FormatReel, "", []string{"#a"}) != "#a" || ComposeCaption(FormatCarousel, "x", nil) != "x" {
		t.Fatal("compose edges")
	}
	if PublishingFormat(FormatCarousel) != FormatFeed || PublishingFormat(FormatStory) != FormatStory {
		t.Fatal("publishing format")
	}
}

func TestSizePresets(t *testing.T) {
	if p := DefaultSizePreset(FormatStory); p.Key != "story" || p.Height != 1920 {
		t.Fatal(p)
	}
	if _, ok := FindSizePreset("nope"); ok {
		t.Fatal("unknown preset")
	}
	if DesignStatusAfterReview(DesignInReview, DecisionComment) != DesignInReview || DesignStatusAfterReview(DesignInReview, DecisionApproved) != DesignApproved {
		t.Fatal("review status")
	}
}

func TestDueReminders(t *testing.T) {
	today := day("2026-10-26")
	refs := []ItemRef{
		{ID: "a", Status: ItemCopywriting, CopyDue: dp("2026-10-27")},
		{ID: "b", Status: ItemDesign, DesignDue: dp("2026-10-20"), CopyDue: dp("2026-10-01")},
		{ID: "c", Status: ItemReview, ReviewDue: dp("2026-10-28")},
		{ID: "d", Status: ItemReview},
		{ID: "e", Status: ItemRevision, CopyDue: dp("2026-10-26")},
	}
	got := DueReminders(refs, today)
	if len(got) != 3 || got[0].Item.ID != "a" || got[1].Kind != DeadlineDesign || got[2].Item.ID != "e" {
		t.Fatalf("%+v", got)
	}
}

func TestComputeReport(t *testing.T) {
	m, _ := ParseMonth("2026-11")
	now := time.Date(2026, 12, 2, 3, 0, 0, 0, time.UTC)
	onTime := time.Date(2026, 11, 4, 3, 0, 0, 0, time.UTC)
	late := time.Date(2026, 11, 12, 3, 0, 0, 0, time.UTC)
	items := []Item{
		{ID: "p1", Title: "A", Platform: PlatformInstagram, Format: FormatReel, Status: ItemPublished, PlannedDate: day("2026-11-05"), CopyDue: dp("2026-11-03"), CopyDoneAt: &onTime, PublishedAt: &onTime, RevisionRound: 1, CaptionVersion: 2, ExternalIDs: []string{"x"}},
		{ID: "p2", Title: "B", Platform: PlatformInstagram, Format: FormatFeed, Status: ItemPublished, PlannedDate: day("2026-11-10"), PublishedAt: &late},
		{ID: "p3", Title: "C", Platform: PlatformFacebook, Format: FormatFeed, Status: ItemPublished, PlannedDate: day("2026-11-15"), PublishedAt: &late},
		{ID: "f1", Platform: PlatformFacebook, Format: FormatStory, Status: ItemFailed, PlannedDate: day("2026-11-20"), RevisionRound: 2},
		{ID: "i1", Platform: PlatformTikTok, Format: FormatReel, Status: ItemDesign, PlannedDate: day("2026-12-10")},
	}
	metrics := map[string]PostMetrics{"p1": {Likes: 100, Comments: 10, Reach: 2000}, "p2": {Likes: 5, Reach: 300}, "p3": {Likes: 50, Saves: 5}}
	d := ComputeReport(m, "Kopi", items, metrics, now)
	if d.Planned != 5 || d.Published != 3 || d.Failed != 1 || d.InProgress != 1 {
		t.Fatalf("%+v", d)
	}
	// Deadlines: p1 copy met + publish met, p2 late, p3 late, f1 publish late; i1 not due yet.
	if d.DeadlinesTotal != 5 || d.DeadlinesMet != 2 || d.OnTimeRate != 40 {
		t.Fatalf("%d/%d %.1f", d.DeadlinesMet, d.DeadlinesTotal, d.OnTimeRate)
	}
	if d.RevisionsTotal != 3 || d.RevisionAvg != 0.6 || d.CaptionVersions != 2 {
		t.Fatalf("%+v", d)
	}
	if len(d.PerPlatform) != 3 || d.PerPlatform[0].Key != "INSTAGRAM" || d.PerPlatform[0].Published != 2 {
		t.Fatalf("%+v", d.PerPlatform)
	}
	if !d.EngagementAvailable || d.TotalEngagements != 170 || d.TotalReach != 2300 {
		t.Fatalf("%+v", d)
	}
	if len(d.Top) != 3 || d.Top[0].ItemID != "p1" || d.Top[2].ItemID != "p2" || len(d.Bottom) != 0 {
		t.Fatalf("top %+v bottom %+v", d.Top, d.Bottom)
	}
	empty := ComputeReport(m, "Kopi", nil, nil, now)
	if empty.OnTimeRate != 0 || empty.EngagementAvailable || empty.Top == nil {
		t.Fatalf("%+v", empty)
	}
}

func TestSniffFile(t *testing.T) {
	cases := map[string]string{
		"\x89PNG\r\n\x1a\n....":      "image/png",
		"\xff\xd8\xff\xe0":           "image/jpeg",
		"%PDF-1.7":                   "application/pdf",
		"wOF2xxxx":                   "font/woff2",
		"\x00\x01\x00\x00xxxx":       "font/ttf",
		"PK\x03\x04xxxx":             "application/zip",
		"<?xml?><svg xmlns='x'/>":    "image/svg+xml",
		"\x00\x00\x00\x18ftypmp42xx": "video/mp4",
		"MZ\x90\x00":                 "",
	}
	for in, want := range cases {
		if got, _ := SniffFile([]byte(in)); got != want {
			t.Errorf("%q: %q want %q", in, got, want)
		}
	}
	if !IsMedia("image/png") || IsMedia("application/pdf") || MimeForKey("c/x.woff2") != "font/woff2" {
		t.Fatal("media")
	}
	if CleanFilename(`C:\fakepath\logo.png`) != "logo.png" || CleanFilename("  ") != "file" {
		t.Fatal(CleanFilename(`C:\fakepath\logo.png`))
	}
}
