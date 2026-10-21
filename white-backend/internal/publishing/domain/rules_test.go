package domain

import (
	"strings"
	"testing"
	"time"
)

var now = time.Date(2026, 9, 2, 3, 0, 0, 0, time.UTC)

func msgs(issues []Issue) []string {
	out := make([]string, 0, len(issues))
	for _, i := range issues {
		out = append(out, i.Message)
	}
	return out
}

func has(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func strp(s string) *string { return &s }

func TestCaptionHelpers(t *testing.T) {
	if n := CountHashtags("#promo #kopi_lokal test#no\n#baris"); n != 3 {
		t.Fatalf("hashtags = %d", n)
	}
	if CountHashtags("tanpa tagar") != 0 {
		t.Fatal("expected 0")
	}
	if CaptionFor("master", nil) != "master" || CaptionFor("master", strp("   ")) != "master" || CaptionFor("master", strp("khusus")) != "khusus" {
		t.Fatal("captionFor")
	}
}

func TestValidatePostInstagram(t *testing.T) {
	ig := func(media ...MediaKind) []Issue {
		return ValidatePost(DraftInput{Body: "hi", Targets: []DraftTarget{{Platform: PlatformInstagram}}, Media: media}, ValidateOptions{Full: true, Now: now})
	}
	if !has(msgs(ig()), MsgIGNeedsMedia) {
		t.Fatal("needs media")
	}
	if len(ig(MediaImage)) != 0 || len(ig(MediaVideo)) != 0 || len(ig(MediaImage, MediaImage, MediaImage)) != 0 {
		t.Fatal("valid IG posts rejected")
	}
	if !has(msgs(ig(MediaImage, MediaVideo)), MsgIGCarouselVideo) {
		t.Fatal("carousel video")
	}
	eleven := make([]MediaKind, 11)
	for i := range eleven {
		eleven[i] = MediaImage
	}
	if len(ig(eleven...)) == 0 {
		t.Fatal("11 images accepted")
	}
}

func TestValidatePostTikTokFacebook(t *testing.T) {
	tt := func(media ...MediaKind) []Issue {
		return ValidatePost(DraftInput{Body: "x", Targets: []DraftTarget{{Platform: PlatformTikTok}}, Media: media}, ValidateOptions{Full: true, Now: now})
	}
	if len(tt(MediaVideo)) != 0 {
		t.Fatal("one video rejected")
	}
	if tt(MediaImage)[0].Message != MsgTikTokOneVideo || len(tt(MediaVideo, MediaVideo)) == 0 {
		t.Fatal("tiktok rule")
	}
	fb := ValidatePost(DraftInput{Body: "teks saja", Targets: []DraftTarget{{Platform: PlatformFacebook}}}, ValidateOptions{Full: true, Now: now})
	if len(fb) != 0 {
		t.Fatalf("fb text only: %v", msgs(fb))
	}
	empty := ValidatePost(DraftInput{Body: "", Targets: []DraftTarget{{Platform: PlatformFacebook}}}, ValidateOptions{Full: true, Now: now})
	if empty[0].Message != MsgEmptyPost {
		t.Fatal("empty post")
	}
	fbVideo := ValidatePost(DraftInput{Body: "x", Targets: []DraftTarget{{Platform: PlatformFacebook}}, Media: []MediaKind{MediaVideo, MediaImage}}, ValidateOptions{Full: true, Now: now})
	if !has(msgs(fbVideo), MsgFBOneVideo) {
		t.Fatal("fb one video")
	}
}

func TestValidatePostLimits(t *testing.T) {
	long := strings.Repeat("a", PlatformLimits[PlatformInstagram].Caption+1)
	r := ValidatePost(DraftInput{Body: long, Targets: []DraftTarget{{Platform: PlatformInstagram}, {Platform: PlatformFacebook}}, Media: []MediaKind{MediaImage}}, ValidateOptions{Full: true, Now: now})
	igCount, fbCount := 0, 0
	for _, i := range r {
		switch i.Scope {
		case "INSTAGRAM":
			igCount++
		case "FACEBOOK":
			fbCount++
		}
	}
	if igCount != 1 || fbCount != 0 {
		t.Fatalf("ig=%d fb=%d", igCount, fbCount)
	}
	if r[0].Message != "Caption melebihi batas 2.200 karakter." {
		t.Fatalf("message %q", r[0].Message)
	}
	if MsgCaptionTooLong(63206) != "Caption melebihi batas 63.206 karakter." {
		t.Fatal(MsgCaptionTooLong(63206))
	}
	var tags []string
	for i := 0; i < 31; i++ {
		tags = append(tags, "#t"+strings.Repeat("x", i%3))
	}
	tagged := ValidatePost(DraftInput{Body: strings.Join(tags, " "), Targets: []DraftTarget{{Platform: PlatformInstagram}}, Media: []MediaKind{MediaImage}}, ValidateOptions{Full: true, Now: now})
	if tagged[0].Message != MsgTooManyHashtags(30) {
		t.Fatalf("hashtags: %v", msgs(tagged))
	}
	fixed := ValidatePost(DraftInput{Body: long, Targets: []DraftTarget{{Platform: PlatformInstagram, BodyOverride: strp("pendek")}}, Media: []MediaKind{MediaImage}}, ValidateOptions{Full: true, Now: now})
	if len(fixed) != 0 {
		t.Fatal("override not honoured")
	}
}

func TestValidatePostDraftAndSchedule(t *testing.T) {
	if len(ValidatePost(DraftInput{}, ValidateOptions{})) != 0 {
		t.Fatal("empty draft rejected")
	}
	r := ValidatePost(DraftInput{LinkURL: strp("bukan url")}, ValidateOptions{})
	if r[0].Message != MsgInvalidLink {
		t.Fatal("invalid link")
	}
	long := strings.Repeat("x", 2201)
	if ValidatePost(DraftInput{FirstComment: &long}, ValidateOptions{})[0].Message != MsgFirstCommentTooLong {
		t.Fatal("first comment")
	}
	base := DraftInput{Body: "x", Targets: []DraftTarget{{Platform: PlatformFacebook}}}
	if ValidatePost(base, ValidateOptions{Full: true, RequireSchedule: true, Now: now})[0].Message != MsgScheduleRequired {
		t.Fatal("schedule required")
	}
	soon := now.Add(4 * time.Minute)
	base.ScheduledAt = &soon
	if ValidatePost(base, ValidateOptions{Full: true, RequireSchedule: true, Now: now})[0].Message != MsgScheduleTooSoon {
		t.Fatal("too soon")
	}
	ok := now.Add(5 * time.Minute)
	base.ScheduledAt = &ok
	if len(ValidatePost(base, ValidateOptions{Full: true, RequireSchedule: true, Now: now})) != 0 {
		t.Fatal("5 min rejected")
	}
}

func TestIssuesToMessage(t *testing.T) {
	got := IssuesToMessage("Perbaiki dulu isu berikut:", []Issue{{"post", "body", MsgEmptyPost}, {"INSTAGRAM", "media", MsgIGNeedsMedia}, {"INSTAGRAM", "media", MsgIGNeedsMedia}})
	want := "Perbaiki dulu isu berikut: Isi caption atau tambahkan media. Instagram: Instagram membutuhkan minimal satu gambar atau video."
	if got != want {
		t.Fatalf("%q", got)
	}
}

func TestStateMachine(t *testing.T) {
	admin := TransitionCtx{Role: RoleAdmin, IsAuthor: true}
	managerAuthor := TransitionCtx{Role: RoleManager, IsAuthor: true}
	manager := TransitionCtx{Role: RoleManager}
	viewer := TransitionCtx{Role: RoleViewer}

	cases := []struct {
		from, to PostStatus
		ctx      TransitionCtx
		want     bool
	}{
		{StatusDraft, StatusInReview, manager, true},
		{StatusInReview, StatusApproved, manager, true},
		{StatusApproved, StatusScheduled, manager, true},
		{StatusScheduled, StatusPublishing, manager, true},
		{StatusDraft, StatusScheduled, admin, false},
		{StatusPublished, StatusDraft, admin, false},
		{StatusInReview, StatusApproved, managerAuthor, false},
		{StatusInReview, StatusRejected, managerAuthor, false},
		{StatusInReview, StatusDraft, managerAuthor, true},
		{StatusInReview, StatusApproved, admin, true},
		{StatusFailed, StatusPublishing, manager, true},
		{StatusInReview, StatusApproved, viewer, false},
	}
	for _, c := range cases {
		if got := CanTransition(c.from, c.to, c.ctx); got != c.want {
			t.Errorf("%s -> %s (%+v) = %v", c.from, c.to, c.ctx, got)
		}
	}
	if len(AllowedTransitions(StatusInReview, viewer)) != 0 || len(AllowedTransitions(StatusPublishing, admin)) != 0 {
		t.Fatal("allowed transitions")
	}
	if !CanEdit(StatusApproved, RoleManager) || CanEdit(StatusScheduled, RoleManager) || CanEdit(StatusDraft, RoleViewer) {
		t.Fatal("canEdit")
	}
	if StatusAfterEdit(StatusApproved) != StatusDraft || StatusAfterEdit(StatusDraft) != StatusDraft || StatusAfterEdit(StatusScheduled) != StatusScheduled {
		t.Fatal("statusAfterEdit")
	}
	if !CanReschedule(StatusScheduled, RoleAdmin) || CanReschedule(StatusPublished, RoleAdmin) {
		t.Fatal("canReschedule")
	}
	if CanDelete(StatusPublishing, RoleAdmin) || !CanDelete(StatusPublished, RoleManager) || CanDelete(StatusDraft, RoleViewer) {
		t.Fatal("canDelete")
	}
}

func TestResolvePostOutcome(t *testing.T) {
	if ResolvePostOutcome([]TargetOutcome{{StatusPublished, 1}, {StatusPublished, 2}}, 3, false) != StatusPublished {
		t.Fatal("all published")
	}
	if ResolvePostOutcome([]TargetOutcome{{StatusPublished, 1}, {StatusFailed, 1}}, 3, false) != StatusScheduled {
		t.Fatal("retry")
	}
	if ResolvePostOutcome([]TargetOutcome{{StatusFailed, 3}}, 3, false) != StatusFailed {
		t.Fatal("exhausted")
	}
	if ResolvePostOutcome([]TargetOutcome{{StatusFailed, 1}}, 3, true) != StatusFailed {
		t.Fatal("manual")
	}
	if ResolvePostOutcome(nil, 3, false) != StatusFailed {
		t.Fatal("empty")
	}
}

func TestUTM(t *testing.T) {
	got := BuildUTMURL("https://kopi.id/promo?ref=x&utm_source=old", UTM{"source": "instagram", "medium": "social", "campaign": "sep"})
	if got != "https://kopi.id/promo?ref=x&utm_source=instagram&utm_medium=social&utm_campaign=sep" {
		t.Fatal(got)
	}
	if BuildUTMURL("https://kopi.id/", UTM{}) != "https://kopi.id/" || BuildUTMURL("bukan url", UTM{"source": "x"}) != "bukan url" {
		t.Fatal("left alone")
	}
	if BuildUTMURL("  https://kopi.id ", UTM{"source": "ig"}) != "https://kopi.id/?utm_source=ig" {
		t.Fatal(BuildUTMURL("  https://kopi.id ", UTM{"source": "ig"}))
	}
	parsed := ParseUTM(map[string]any{"source": " ig ", "medium": "", "junk": 1})
	if len(parsed) != 1 || parsed["source"] != "ig" {
		t.Fatalf("%v", parsed)
	}
	if len(ParseUTM(nil)) != 0 {
		t.Fatal("nil")
	}
	d := DefaultUTM(PlatformFacebook)
	if d["source"] != "facebook" || d["medium"] != "social" || DefaultUTM("")["source"] != "social" {
		t.Fatal("default utm")
	}
}

func TestTabsAndPresentation(t *testing.T) {
	counts := map[PostStatus]int32{StatusDraft: 2, StatusRejected: 1, StatusScheduled: 3, StatusPublishing: 1, StatusPublished: 4}
	if TabCount(TabDraft, counts) != 3 || TabCount(TabScheduled, counts) != 4 || TabCount(TabAll, counts) != 11 || TabCount(TabFailed, counts) != 0 {
		t.Fatal("tab counts")
	}
	if !IsPostsTab("IN_REVIEW") || IsPostsTab("REJECTED") {
		t.Fatal("isPostsTab")
	}
	if Truncate("a  b\n c", 90) != "a b c" || Truncate("abcdefghij", 5) != "abcd…" {
		t.Fatal(Truncate("abcdefghij", 5))
	}
	p := ContentPost{ID: "id1", Body: strings.Repeat("b", 50)}
	if NotificationTitle(p) != strings.Repeat("b", 40) || NotificationTitle(ContentPost{ID: "id1"}) != "id1" || NotificationTitle(ContentPost{Title: "T"}) != "T" {
		t.Fatal("notification title")
	}
	sched := now.Add(time.Hour)
	if !(ContentPost{ScheduledAt: &sched, CreatedAt: now}).Instant().Equal(sched) {
		t.Fatal("instant")
	}
}
