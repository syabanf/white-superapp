package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// AgencyTZ is where deadlines and "today" are evaluated: Mixist works in
// Jakarta time.
const AgencyTZ = "Asia/Jakarta"

func AgencyLocation() *time.Location {
	loc, err := time.LoadLocation(AgencyTZ)
	if err != nil {
		return time.FixedZone("WIB", 7*3600)
	}
	return loc
}

// Day is the agency calendar day of t as a UTC midnight date.
func Day(t time.Time) time.Time {
	y, m, d := t.In(AgencyLocation()).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// ─── Month ────────────────────────────────────────────────────

// Month is the first day of a calendar month (UTC midnight).
type Month struct{ time.Time }

func MonthOf(t time.Time) Month {
	return Month{time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)}
}

// ParseMonth accepts "YYYY-MM".
func ParseMonth(s string) (Month, bool) {
	t, err := time.Parse("2006-01", strings.TrimSpace(s))
	if err != nil {
		return Month{}, false
	}
	return MonthOf(t), true
}

func (m Month) String() string { return m.Format("2006-01") }

// Last is the last day of the month.
func (m Month) Last() time.Time { return m.AddDate(0, 1, -1) }

func (m Month) Contains(day time.Time) bool {
	return !day.Before(m.Time) && !day.After(m.Last())
}

func (m Month) Next() Month { return Month{m.AddDate(0, 1, 0)} }
func (m Month) Prev() Month { return Month{m.AddDate(0, -1, 0)} }

var monthNames = []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

// Label is "Oktober 2026".
func (m Month) Label() string { return fmt.Sprintf("%s %d", monthNames[m.Month()-1], m.Year()) }

// ─── Enums ────────────────────────────────────────────────────

func IsPlatform(p Platform) bool {
	for _, v := range Platforms {
		if v == p {
			return true
		}
	}
	return false
}

func IsFormat(f Format) bool {
	for _, v := range Formats {
		if v == f {
			return true
		}
	}
	return false
}

func PlatformLabel(p Platform) string {
	switch p {
	case PlatformFacebook:
		return "Facebook"
	case PlatformTikTok:
		return "TikTok"
	}
	return "Instagram"
}

func FormatLabel(f Format) string {
	switch f {
	case FormatCarousel:
		return "Carousel"
	case FormatReel:
		return "Reel"
	case FormatStory:
		return "Story"
	}
	return "Feed"
}

// PublishingFormat maps a content format to the publishing post format:
// carousels are feed posts with several media.
func PublishingFormat(f Format) Format {
	if f == FormatCarousel {
		return FormatFeed
	}
	return f
}

// ─── Plan workflow ────────────────────────────────────────────

// CanEditBrief: the brief and AI generation belong to the draft stage.
func CanEditBrief(s PlanStatus) bool { return s == PlanDraft }

// CanApproveOwn mirrors publishing: a MANAGER may not approve what they
// submitted or wrote; ADMIN may.
func CanApproveOwn(role Role) bool { return role == RoleAdmin }

// ─── Item workflow ────────────────────────────────────────────

// ItemTransitions lists the moves people make by hand. SCHEDULED,
// PUBLISHED and FAILED are reached through scheduling and publish results.
var ItemTransitions = map[ItemStatus][]ItemStatus{
	ItemIdea:        {ItemCopywriting},
	ItemCopywriting: {ItemDesign, ItemReview, ItemIdea},
	ItemDesign:      {ItemReview, ItemCopywriting},
	ItemReview:      {ItemApproved, ItemRevision},
	ItemRevision:    {ItemReview},
	ItemApproved:    {ItemRevision},
	ItemFailed:      {ItemApproved},
}

// IsReviewMove: approving or sending back is the reviewer's call.
func IsReviewMove(from, to ItemStatus) bool {
	return to == ItemApproved && from == ItemReview || to == ItemRevision
}

type MoveCtx struct {
	Role Role
	// IsCopywriter: the caller wrote the caption.
	IsCopywriter bool
}

// CanMove: VIEWER never; a MANAGER may not review their own copy; ADMIN may.
func CanMove(from, to ItemStatus, ctx MoveCtx) bool {
	if ctx.Role == RoleViewer {
		return false
	}
	allowed := false
	for _, s := range ItemTransitions[from] {
		if s == to {
			allowed = true
			break
		}
	}
	if !allowed {
		return false
	}
	if IsReviewMove(from, to) && ctx.Role == RoleManager && ctx.IsCopywriter {
		return false
	}
	return true
}

func AllowedMoves(from ItemStatus, ctx MoveCtx) []ItemStatus {
	out := []ItemStatus{}
	for _, to := range ItemTransitions[from] {
		if CanMove(from, to, ctx) {
			out = append(out, to)
		}
	}
	return out
}

// IsLocked: scheduled and published items no longer change here.
func IsLocked(s ItemStatus) bool { return s == ItemScheduled || s == ItemPublished }

// CanSchedule: approved items, and failed ones being retried.
func CanSchedule(s ItemStatus) bool { return s == ItemApproved || s == ItemFailed }

// MoveStamps applies the milestone timestamps a move records the first
// time it happens (copy done, design done, approval).
func MoveStamps(it *Item, to ItemStatus, userID string, now time.Time) {
	from := it.Status
	if from == ItemCopywriting && (to == ItemDesign || to == ItemReview) && it.CopyDoneAt == nil {
		it.CopyDoneAt = &now
	}
	if from == ItemDesign && to == ItemReview && it.DesignDoneAt == nil {
		it.DesignDoneAt = &now
	}
	switch to {
	case ItemApproved:
		if from == ItemReview {
			it.ApprovedAt = &now
			it.ApprovedBy = &userID
		}
	case ItemRevision:
		it.RevisionRound++
		it.ApprovedAt, it.ApprovedBy = nil, nil
	}
	it.Status = to
}

// ─── Captions ─────────────────────────────────────────────────

// NormalizeHashtags trims, adds the leading "#", drops blanks, spaces
// inside a tag and duplicates (case-insensitive), keeping order.
func NormalizeHashtags(tags []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, t := range tags {
		t = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(t), "#"))
		t = strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, t)
		if t == "" {
			continue
		}
		key := strings.ToLower(t)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, "#"+t)
	}
	return out
}

// ComposeCaption is the text sent to publishing: caption, a blank line,
// then the hashtags. Stories carry no caption.
func ComposeCaption(format Format, caption string, hashtags []string) string {
	if format == FormatStory {
		return ""
	}
	caption = strings.TrimSpace(caption)
	tags := strings.Join(hashtags, " ")
	switch {
	case tags == "":
		return caption
	case caption == "":
		return tags
	}
	return caption + "\n\n" + tags
}

// ─── Design ───────────────────────────────────────────────────

type SizePreset struct {
	Key    string
	Label  string
	Width  int32
	Height int32
	Format Format
}

// SizePresets are the canvas sizes designers start from, per format.
var SizePresets = []SizePreset{
	{"feed-portrait", "Feed portrait 4:5", 1080, 1350, FormatFeed},
	{"feed-square", "Feed persegi 1:1", 1080, 1080, FormatFeed},
	{"carousel-portrait", "Carousel portrait 4:5", 1080, 1350, FormatCarousel},
	{"carousel-square", "Carousel persegi 1:1", 1080, 1080, FormatCarousel},
	{"reel-cover", "Reel / video 9:16", 1080, 1920, FormatReel},
	{"story", "Story 9:16", 1080, 1920, FormatStory},
}

func FindSizePreset(key string) (SizePreset, bool) {
	for _, p := range SizePresets {
		if p.Key == key {
			return p, true
		}
	}
	return SizePreset{}, false
}

// DefaultSizePreset is the first preset of the format.
func DefaultSizePreset(f Format) SizePreset {
	for _, p := range SizePresets {
		if p.Format == f {
			return p
		}
	}
	return SizePresets[0]
}

// DesignStatusAfterReview maps a reviewer decision to the task status.
func DesignStatusAfterReview(current DesignStatus, decision string) DesignStatus {
	switch decision {
	case DecisionApproved:
		return DesignApproved
	case DecisionChangesRequested:
		return DesignChangesRequested
	}
	return current
}

// ─── Deadlines ────────────────────────────────────────────────

// DueReminders returns the deadline that matters for the item's current
// stage when it is due tomorrow or already overdue on `today`.
func DueReminders(items []ItemRef, today time.Time) []Reminder {
	tomorrow := today.AddDate(0, 0, 1)
	var out []Reminder
	for _, it := range items {
		var kind string
		var due *time.Time
		switch it.Status {
		case ItemIdea, ItemCopywriting, ItemRevision:
			kind, due = DeadlineCopy, it.CopyDue
		case ItemDesign:
			kind, due = DeadlineDesign, it.DesignDue
		case ItemReview:
			kind, due = DeadlineReview, it.ReviewDue
		}
		if due == nil || due.After(tomorrow) {
			continue
		}
		out = append(out, Reminder{Item: it, Kind: kind, Due: *due})
	}
	return out
}

func DeadlineLabel(kind string) string {
	switch kind {
	case DeadlineDesign:
		return "desain"
	case DeadlineReview:
		return "review"
	}
	return "copywriting"
}
