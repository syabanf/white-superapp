package domain

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// ─── Platform limits ──────────────────────────────────────────

type PlatformLimit struct {
	Caption int
	// Hashtags is 0 when the platform has no hashtag limit.
	Hashtags  int
	MediaMin  int
	MediaMax  int
	VideoOnly bool
}

var PlatformLimits = map[Platform]PlatformLimit{
	PlatformInstagram: {Caption: 2200, Hashtags: 30, MediaMin: 1, MediaMax: 10},
	PlatformFacebook:  {Caption: 63206, MediaMin: 0, MediaMax: 10},
	PlatformTikTok:    {Caption: 2200, MediaMin: 1, MediaMax: 1, VideoOnly: true},
}

const (
	MinScheduleLead    = 5 * time.Minute
	MaxPublishAttempts = 3
)

var hashtagRe = regexp.MustCompile(`(^|\s)#[\p{L}\p{N}_]+`)

func CountHashtags(text string) int {
	return len(hashtagRe.FindAllStringIndex(text, -1))
}

// CaptionFor returns the override when it is non-blank, else the master caption.
func CaptionFor(master string, override *string) string {
	if override != nil && strings.TrimSpace(*override) != "" {
		return *override
	}
	return master
}

// ─── Validation ───────────────────────────────────────────────

// Validation messages (strings.ts V). Keep them byte-identical.
const (
	MsgNoTargets           = "Pilih minimal satu akun tujuan."
	MsgEmptyPost           = "Isi caption atau tambahkan media."
	MsgInvalidLink         = "Tautan harus berupa URL http(s) yang valid."
	MsgScheduleRequired    = "Tentukan jadwal terbit."
	MsgScheduleTooSoon     = "Jadwal minimal 5 menit dari sekarang."
	MsgIGNeedsMedia        = "Instagram membutuhkan minimal satu gambar atau video."
	MsgIGCarousel          = "Carousel Instagram berisi 2–10 gambar."
	MsgIGCarouselVideo     = "Carousel Instagram hanya boleh berisi gambar."
	MsgTikTokOneVideo      = "TikTok membutuhkan tepat satu video."
	MsgFBOneVideo          = "Facebook hanya menerima satu video per post."
	MsgStoryOneMedia       = "Story membutuhkan tepat satu gambar atau video."
	MsgStoryTikTok         = "TikTok belum mendukung story."
	MsgReelOneVideo        = "Reel membutuhkan tepat satu video."
	MsgFirstCommentTooLong = "Komentar pertama melebihi 2.200 karakter."
)

func MsgCaptionTooLong(limit int) string {
	return fmt.Sprintf("Caption melebihi batas %s karakter.", formatID(limit))
}

func MsgTooManyHashtags(limit int) string { return fmt.Sprintf("Maksimal %d hashtag.", limit) }

func MsgTooManyMedia(max int) string { return fmt.Sprintf("Maksimal %d media.", max) }

// formatID renders an integer with id-ID thousands separators (2.200).
func formatID(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	head := len(s) % 3
	if head > 0 {
		b.WriteString(s[:head])
	}
	for i := head; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte('.')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

type DraftTarget struct {
	Platform     Platform
	BodyOverride *string
}

type DraftInput struct {
	// Format empty is FEED.
	Format       PostFormat
	Body         string
	Targets      []DraftTarget
	Media        []MediaKind
	ScheduledAt  *time.Time
	LinkURL      *string
	FirstComment *string
}

type ValidateOptions struct {
	// Full checks everything needed to publish; false is the draft level
	// (hard limits only).
	Full            bool
	RequireSchedule bool
	Now             time.Time
}

// Issue scope is "post" or a platform name.
type Issue struct {
	Scope   string
	Field   string
	Message string
}

func IsHTTPURL(v string) bool {
	u, err := url.Parse(v)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https")
}

func validatePlatform(platform Platform, caption string, media []MediaKind, full bool) []Issue {
	lim := PlatformLimits[platform]
	scope := string(platform)
	var out []Issue
	if utf8.RuneCountInString(caption) > lim.Caption {
		out = append(out, Issue{scope, "body", MsgCaptionTooLong(lim.Caption)})
	}
	if lim.Hashtags > 0 && CountHashtags(caption) > lim.Hashtags {
		out = append(out, Issue{scope, "body", MsgTooManyHashtags(lim.Hashtags)})
	}
	if len(media) > lim.MediaMax {
		out = append(out, Issue{scope, "media", MsgTooManyMedia(lim.MediaMax)})
	}
	if !full {
		return out
	}
	videos := 0
	for _, m := range media {
		if m == MediaVideo {
			videos++
		}
	}
	switch platform {
	case PlatformInstagram:
		if len(media) == 0 {
			out = append(out, Issue{scope, "media", MsgIGNeedsMedia})
		} else if len(media) > 1 {
			if len(media) > 10 {
				out = append(out, Issue{scope, "media", MsgIGCarousel})
			}
			if videos > 0 {
				out = append(out, Issue{scope, "media", MsgIGCarouselVideo})
			}
		}
	case PlatformTikTok:
		if len(media) != 1 || videos != 1 {
			out = append(out, Issue{scope, "media", MsgTikTokOneVideo})
		}
	case PlatformFacebook:
		if videos > 0 && len(media) > 1 {
			out = append(out, Issue{scope, "media", MsgFBOneVideo})
		}
	}
	return out
}

// validateFormat adds the STORY and REEL shape rules (publish level only).
func validateFormat(format PostFormat, platform Platform, media []MediaKind) []Issue {
	scope := string(platform)
	switch format {
	case FormatStory:
		if platform == PlatformTikTok {
			return []Issue{{scope, "format", MsgStoryTikTok}}
		}
		if len(media) != 1 {
			return []Issue{{scope, "media", MsgStoryOneMedia}}
		}
	case FormatReel:
		if len(media) != 1 || media[0] != MediaVideo {
			return []Issue{{scope, "media", MsgReelOneVideo}}
		}
	}
	return nil
}

// ValidatePost mirrors lib.ts validatePost. An empty result means ok.
func ValidatePost(in DraftInput, opts ValidateOptions) []Issue {
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	var issues []Issue
	if in.LinkURL != nil {
		link := strings.TrimSpace(*in.LinkURL)
		if link != "" && !IsHTTPURL(link) {
			issues = append(issues, Issue{"post", "linkUrl", MsgInvalidLink})
		}
	}
	if in.FirstComment != nil && utf8.RuneCountInString(*in.FirstComment) > 2200 {
		issues = append(issues, Issue{"post", "firstComment", MsgFirstCommentTooLong})
	}
	if opts.Full {
		if len(in.Targets) == 0 {
			issues = append(issues, Issue{"post", "targets", MsgNoTargets})
		}
		if strings.TrimSpace(in.Body) == "" && len(in.Media) == 0 {
			issues = append(issues, Issue{"post", "body", MsgEmptyPost})
		}
	}
	if opts.RequireSchedule {
		if in.ScheduledAt == nil {
			issues = append(issues, Issue{"post", "scheduledAt", MsgScheduleRequired})
		} else if in.ScheduledAt.Before(now.Add(MinScheduleLead)) {
			issues = append(issues, Issue{"post", "scheduledAt", MsgScheduleTooSoon})
		}
	}
	seen := map[string]bool{}
	for _, t := range in.Targets {
		caption := CaptionFor(in.Body, t.BodyOverride)
		key := string(t.Platform) + ":" + caption
		if seen[key] {
			continue
		}
		seen[key] = true
		issues = append(issues, validatePlatform(t.Platform, caption, in.Media, opts.Full)...)
		if opts.Full {
			issues = append(issues, validateFormat(in.Format, t.Platform, in.Media)...)
		}
	}
	return issues
}

// IssuesToMessage joins unique messages after a prefix; platform-scoped
// messages are prefixed with the platform name (access.ts issuesToMessage).
func IssuesToMessage(prefix string, issues []Issue) string {
	seen := map[string]bool{}
	var parts []string
	for _, i := range issues {
		msg := i.Message
		if i.Scope != "post" {
			msg = PlatformLabel(Platform(i.Scope)) + ": " + msg
		}
		if seen[msg] {
			continue
		}
		seen[msg] = true
		parts = append(parts, msg)
	}
	return prefix + " " + strings.Join(parts, " ")
}

// ─── State machine ────────────────────────────────────────────

// Transitions lists human-driven moves. PUBLISHING to PUBLISHED, FAILED or
// SCHEDULED is system-only.
var Transitions = map[PostStatus][]PostStatus{
	StatusDraft:      {StatusInReview, StatusApproved},
	StatusInReview:   {StatusApproved, StatusRejected, StatusDraft},
	StatusRejected:   {StatusDraft, StatusInReview},
	StatusApproved:   {StatusScheduled, StatusPublishing, StatusDraft},
	StatusScheduled:  {StatusApproved, StatusPublishing},
	StatusPublishing: {},
	StatusPublished:  {},
	StatusFailed:     {StatusPublishing, StatusScheduled, StatusDraft},
}

type TransitionCtx struct {
	Role     PostRole
	IsAuthor bool
}

// CanTransition: VIEWER never; a MANAGER may not approve or reject their own
// post; ADMIN may.
func CanTransition(from, to PostStatus, ctx TransitionCtx) bool {
	if ctx.Role == RoleViewer {
		return false
	}
	if !containsStatus(Transitions[from], to) {
		return false
	}
	if (to == StatusApproved || to == StatusRejected) && ctx.Role == RoleManager && ctx.IsAuthor {
		return false
	}
	return true
}

func AllowedTransitions(from PostStatus, ctx TransitionCtx) []PostStatus {
	out := []PostStatus{}
	for _, to := range Transitions[from] {
		if CanTransition(from, to, ctx) {
			out = append(out, to)
		}
	}
	return out
}

var EditableStatuses = []PostStatus{StatusDraft, StatusRejected, StatusInReview, StatusApproved}

func CanEdit(status PostStatus, role PostRole) bool {
	return role != RoleViewer && containsStatus(EditableStatuses, status)
}

// StatusAfterEdit: editing a reviewed post sends it back to DRAFT.
func StatusAfterEdit(status PostStatus) PostStatus {
	if containsStatus(EditableStatuses, status) {
		return StatusDraft
	}
	return status
}

var ReschedulableStatuses = []PostStatus{StatusDraft, StatusApproved, StatusScheduled}

func CanReschedule(status PostStatus, role PostRole) bool {
	return role != RoleViewer && containsStatus(ReschedulableStatuses, status)
}

func CanDelete(status PostStatus, role PostRole) bool {
	return role != RoleViewer && status != StatusPublishing
}

func containsStatus(xs []PostStatus, x PostStatus) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// ─── Publish outcome ──────────────────────────────────────────

type TargetOutcome struct {
	Status   PostStatus
	Attempts int32
}

// ResolvePostOutcome: PUBLISHED when every target published, FAILED when a
// failed target exhausted its attempts (or the run is manual), otherwise
// SCHEDULED so the next tick retries. No targets at all is FAILED.
func ResolvePostOutcome(targets []TargetOutcome, maxAttempts int32, manual bool) PostStatus {
	if len(targets) > 0 {
		all := true
		for _, t := range targets {
			if t.Status != StatusPublished {
				all = false
				break
			}
		}
		if all {
			return StatusPublished
		}
	}
	var failed []TargetOutcome
	for _, t := range targets {
		if t.Status == StatusFailed {
			failed = append(failed, t)
		}
	}
	if len(failed) == 0 {
		if len(targets) == 0 {
			return StatusFailed
		}
		return StatusScheduled
	}
	if manual {
		return StatusFailed
	}
	for _, t := range failed {
		if t.Attempts >= maxAttempts {
			return StatusFailed
		}
	}
	return StatusScheduled
}

// ─── UTM ──────────────────────────────────────────────────────

// UTM keeps only non-blank, trimmed values keyed by source, medium,
// campaign, term, content.
type UTM map[string]string

var UTMKeys = []string{"source", "medium", "campaign", "term", "content"}

// ParseUTM reads a stored JSON object defensively.
func ParseUTM(raw map[string]any) UTM {
	out := UTM{}
	for _, k := range UTMKeys {
		if v, ok := raw[k].(string); ok && strings.TrimSpace(v) != "" {
			out[k] = strings.TrimSpace(v)
		}
	}
	return out
}

// Normalize drops unknown keys and blank values.
func (u UTM) Normalize() UTM {
	out := UTM{}
	for _, k := range UTMKeys {
		if v := strings.TrimSpace(u[k]); v != "" {
			out[k] = v
		}
	}
	return out
}

func (u UTM) Has() bool {
	for _, k := range UTMKeys {
		if u[k] != "" {
			return true
		}
	}
	return false
}

func DefaultUTM(platform Platform) UTM {
	source := "social"
	if platform != "" {
		source = strings.ToLower(string(platform))
	}
	return UTM{"source": source, "medium": "social"}
}

// BuildUTMURL appends utm_* params, keeping existing params in place and
// overwriting existing utm_* values (URLSearchParams.set semantics). An
// invalid or non-http link is returned trimmed and unchanged.
func BuildUTMURL(link string, utm UTM) string {
	trimmed := strings.TrimSpace(link)
	if trimmed == "" || !utm.Has() || !IsHTTPURL(trimmed) {
		return trimmed
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return trimmed
	}
	type kv struct{ k, v string }
	var params []kv
	for _, part := range strings.Split(u.RawQuery, "&") {
		if part == "" {
			continue
		}
		k, v, _ := strings.Cut(part, "=")
		dk, err := url.QueryUnescape(k)
		if err != nil {
			dk = k
		}
		dv, err := url.QueryUnescape(v)
		if err != nil {
			dv = v
		}
		params = append(params, kv{dk, dv})
	}
	for _, k := range UTMKeys {
		v := strings.TrimSpace(utm[k])
		if v == "" {
			continue
		}
		name := "utm_" + k
		replaced := false
		kept := params[:0]
		for _, p := range params {
			if p.k != name {
				kept = append(kept, p)
				continue
			}
			if !replaced {
				kept = append(kept, kv{name, v})
				replaced = true
			}
		}
		params = kept
		if !replaced {
			params = append(params, kv{name, v})
		}
	}
	var b strings.Builder
	for i, p := range params {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(url.QueryEscape(p.k))
		b.WriteByte('=')
		b.WriteString(url.QueryEscape(p.v))
	}
	u.RawQuery = b.String()
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String()
}

// ─── Posts tabs ───────────────────────────────────────────────

type PostsTab string

const (
	TabAll       PostsTab = "ALL"
	TabDraft     PostsTab = "DRAFT"
	TabInReview  PostsTab = "IN_REVIEW"
	TabApproved  PostsTab = "APPROVED"
	TabScheduled PostsTab = "SCHEDULED"
	TabPublished PostsTab = "PUBLISHED"
	TabFailed    PostsTab = "FAILED"
)

var PostsTabs = []PostsTab{TabAll, TabDraft, TabInReview, TabApproved, TabScheduled, TabPublished, TabFailed}

// TabStatuses maps a tab to the statuses it shows; nil means every status.
var TabStatuses = map[PostsTab][]PostStatus{
	TabAll:       nil,
	TabDraft:     {StatusDraft, StatusRejected},
	TabInReview:  {StatusInReview},
	TabApproved:  {StatusApproved},
	TabScheduled: {StatusScheduled, StatusPublishing},
	TabPublished: {StatusPublished},
	TabFailed:    {StatusFailed},
}

func IsPostsTab(v string) bool {
	for _, t := range PostsTabs {
		if string(t) == v {
			return true
		}
	}
	return false
}

// TabIncludes reports whether a tab shows posts of a status.
func TabIncludes(tab PostsTab, status PostStatus) bool {
	statuses, ok := TabStatuses[tab]
	if !ok || statuses == nil {
		return true
	}
	return containsStatus(statuses, status)
}

func TabCount(tab PostsTab, counts map[PostStatus]int32) int32 {
	var n int32
	for s, c := range counts {
		if TabIncludes(tab, s) {
			n += c
		}
	}
	return n
}

// ─── Presentation helpers ─────────────────────────────────────

var spaceRe = regexp.MustCompile(`\s+`)

// Truncate mirrors lib/format.ts truncate on collapsed whitespace.
func Truncate(text string, max int) string {
	text = spaceRe.ReplaceAllString(text, " ")
	if utf8.RuneCountInString(text) <= max {
		return text
	}
	runes := []rune(text)
	return strings.TrimRight(string(runes[:max-1]), " \t\n") + "…"
}

// NotificationTitle is title, else the first 40 characters of the body, else the id.
func NotificationTitle(p ContentPost) string {
	if p.Title != "" {
		return p.Title
	}
	runes := []rune(p.Body)
	if len(runes) > 40 {
		runes = runes[:40]
	}
	if s := string(runes); s != "" {
		return s
	}
	return p.ID
}
