package app

import (
	"context"
	"time"
	"unicode/utf8"

	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// Items is the content item workflow: planning fields, versioned
// captions, review moves and comments.
type Items struct{ Deps }

func NewItems(d Deps) *Items { return &Items{d} }

// ItemInput holds the planning fields; dates are YYYY-MM-DD, "" clears.
type ItemInput struct {
	PlannedDate  string
	PublishAt    *time.Time
	Platform     domain.Platform
	Format       domain.Format
	Pillar       string
	Title        string
	VisualBrief  string
	CopywriterID string
	DesignerID   string
	CopyDue      string
	DesignDue    string
	ReviewDue    string
}

type CaptionInput struct {
	Hook     string
	Caption  string
	Hashtags []string
	Note     string
}

func (c CaptionInput) empty() bool {
	return trim(c.Caption) == "" && trim(c.Hook) == "" && len(c.Hashtags) == 0
}

type ItemDetail struct {
	Item       domain.Item
	Revisions  []domain.CaptionRevision
	Comments   []domain.ItemComment
	Activities []domain.ItemActivity
	Tasks      []domain.DesignTask
	Allowed    []domain.ItemStatus
	Names      map[string]string
}

func parseDay(v *validate.Validator, field, s string) *time.Time {
	s = trim(s)
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		v.Add(field, msgDateInvalid)
		return nil
	}
	return &t
}

// applyInput validates the planning fields against the plan month and
// writes them onto the item.
func applyInput(it *domain.Item, in ItemInput, month domain.Month) error {
	v := validate.New()
	planned := parseDay(v, "plannedDate", in.PlannedDate)
	if planned == nil && trim(in.PlannedDate) == "" {
		v.Add("plannedDate", msgDateInvalid)
	}
	if planned != nil && !month.Contains(*planned) {
		v.Add("plannedDate", msgDateOutsideMonth)
	}
	if !domain.IsPlatform(in.Platform) {
		v.Add("platform", msgPlatformInvalid)
	}
	if !domain.IsFormat(in.Format) {
		v.Add("format", msgFormatInvalid)
	}
	v.MaxLen("title", in.Title, 160, msgTitleTooLong)
	v.MaxLen("pillar", in.Pillar, 80, "Pilar terlalu panjang (maksimal 80 karakter).")
	v.MaxLen("visualBrief", in.VisualBrief, 2000, "Brief visual terlalu panjang (maksimal 2.000 karakter).")
	copyDue := parseDay(v, "copyDue", in.CopyDue)
	designDue := parseDay(v, "designDue", in.DesignDue)
	reviewDue := parseDay(v, "reviewDue", in.ReviewDue)
	if v.HasErrors() {
		return v.Err()
	}
	it.PlannedDate = *planned
	it.PublishAt = in.PublishAt
	it.Platform, it.Format = in.Platform, in.Format
	it.Pillar, it.Title, it.VisualBrief = trim(in.Pillar), trim(in.Title), trim(in.VisualBrief)
	it.CopywriterID, it.DesignerID = optID(in.CopywriterID), optID(in.DesignerID)
	it.CopyDue, it.DesignDue, it.ReviewDue = copyDue, designDue, reviewDue
	return nil
}

func validateCaption(c *CaptionInput) error {
	v := validate.New()
	v.MaxLen("caption", c.Caption, 2200, msgCaptionTooLong)
	v.MaxLen("hook", c.Hook, 300, "Hook terlalu panjang (maksimal 300 karakter).")
	v.MaxLen("note", c.Note, 1000, "Catatan terlalu panjang (maksimal 1.000 karakter).")
	c.Hashtags = domain.NormalizeHashtags(c.Hashtags)
	if len(c.Hashtags) > 30 {
		v.Add("hashtags", msgTooManyHashtags)
	}
	if v.HasErrors() {
		return v.Err()
	}
	c.Caption, c.Hook, c.Note = trim(c.Caption), trim(c.Hook), trim(c.Note)
	return nil
}

// insertItem writes a new item with its first caption version (when
// given) and the CREATED activity. Callers run it inside a transaction.
func insertItem(ctx context.Context, tx domain.Store, it domain.Item, c CaptionInput, userID, note string) error {
	if !c.empty() {
		it.Hook, it.Caption, it.Hashtags = c.Hook, c.Caption, c.Hashtags
		it.CaptionVersion = 1
	}
	if err := tx.InsertItem(ctx, it); err != nil {
		return err
	}
	if it.CaptionVersion == 1 {
		if err := tx.InsertRevision(ctx, domain.CaptionRevision{ID: ids.New(), ItemID: it.ID, Version: 1, Caption: c.Caption, Hook: c.Hook, Hashtags: c.Hashtags, Note: c.Note, CreatedBy: optID(userID)}); err != nil {
			return err
		}
	}
	to := string(it.Status)
	return tx.InsertActivity(ctx, domain.ItemActivity{ID: ids.New(), ItemID: it.ID, UserID: optID(userID), Type: domain.ActCreated, ToStatus: &to, Note: optID(note)})
}

func (s *Items) List(ctx context.Context, userID, clientID string, f domain.ItemFilter) ([]domain.Item, error) {
	if _, err := s.access(ctx, userID, clientID); err != nil {
		return nil, err
	}
	return s.Store.ListItems(ctx, clientID, f)
}

func (s *Items) Create(ctx context.Context, userID, clientID, planID string, in ItemInput, c CaptionInput) (*ItemDetail, error) {
	if _, err := s.manage(ctx, userID, clientID); err != nil {
		return nil, err
	}
	plan, err := ensureFound(s.Store.GetPlan(ctx, clientID, planID))(msgPlanNotFound)
	if err != nil {
		return nil, err
	}
	it := domain.Item{ID: ids.New(), PlanID: plan.ID, ClientID: clientID, Status: domain.ItemIdea, CreatedBy: &userID, Hashtags: []string{}}
	if err := applyInput(&it, in, plan.Month); err != nil {
		return nil, err
	}
	if err := validateCaption(&c); err != nil {
		return nil, err
	}
	if err := s.Store.WithTx(ctx, func(tx domain.Store) error { return insertItem(ctx, tx, it, c, userID, "") }); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, clientID, it.ID)
}

// load returns the item and the caller's access; missing access reads as
// a missing item.
func (s *Items) load(ctx context.Context, userID, clientID, itemID string, write bool) (*domain.Item, *Access, error) {
	a, err := s.access(ctx, userID, clientID)
	if err != nil {
		return nil, nil, apperr.New(apperr.NotFound, msgItemNotFound)
	}
	it, err := ensureFound(s.Store.GetItem(ctx, clientID, itemID))(msgItemNotFound)
	if err != nil {
		return nil, nil, err
	}
	if write && !a.canWrite() {
		return nil, nil, apperr.New(apperr.Unauthorized, "")
	}
	return it, a, nil
}

func (s *Items) Get(ctx context.Context, userID, clientID, itemID string) (*ItemDetail, error) {
	it, a, err := s.load(ctx, userID, clientID, itemID, false)
	if err != nil {
		return nil, err
	}
	d := &ItemDetail{Item: *it}
	if d.Revisions, err = s.Store.ListRevisions(ctx, it.ID); err != nil {
		return nil, err
	}
	if d.Comments, err = s.Store.ListItemComments(ctx, it.ID); err != nil {
		return nil, err
	}
	if d.Activities, err = s.Store.ListActivities(ctx, it.ID); err != nil {
		return nil, err
	}
	if d.Tasks, err = s.Store.ListDesignTasks(ctx, clientID, it.ID, "", ""); err != nil {
		return nil, err
	}
	d.Allowed = domain.AllowedMoves(it.Status, moveCtx(it, a))
	var people []*string
	for _, r := range d.Revisions {
		people = append(people, r.CreatedBy)
	}
	for _, c := range d.Comments {
		people = append(people, c.UserID)
	}
	for _, act := range d.Activities {
		people = append(people, act.UserID)
	}
	d.Names = s.userNames(ctx, people...)
	return d, nil
}

func moveCtx(it *domain.Item, a *Access) domain.MoveCtx {
	return domain.MoveCtx{Role: a.Role, IsCopywriter: it.CopywriterID != nil && *it.CopywriterID == a.UserID}
}

func (s *Items) Update(ctx context.Context, userID, clientID, itemID string, in ItemInput) (*ItemDetail, error) {
	it, _, err := s.load(ctx, userID, clientID, itemID, true)
	if err != nil {
		return nil, err
	}
	if domain.IsLocked(it.Status) {
		return nil, apperr.New(apperr.Validation, msgItemLocked)
	}
	plan, err := ensureFound(s.Store.GetPlan(ctx, clientID, it.PlanID))(msgPlanNotFound)
	if err != nil {
		return nil, err
	}
	if err := applyInput(it, in, plan.Month); err != nil {
		return nil, err
	}
	it.UpdatedBy = &userID
	err = s.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.UpdateItem(ctx, *it); err != nil {
			return err
		}
		return tx.InsertActivity(ctx, domain.ItemActivity{ID: ids.New(), ItemID: it.ID, UserID: &userID, Type: domain.ActEdited})
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, clientID, itemID)
}

func (s *Items) Delete(ctx context.Context, userID, clientID, itemID string) error {
	it, _, err := s.load(ctx, userID, clientID, itemID, true)
	if err != nil {
		return err
	}
	if domain.IsLocked(it.Status) {
		return apperr.New(apperr.Validation, msgItemLocked)
	}
	return s.Store.SoftDeleteItem(ctx, it.ID, userID)
}

// Move applies one workflow step with its comment, audit row and notifications.
func (s *Items) Move(ctx context.Context, userID, clientID, itemID string, to domain.ItemStatus, note string) (*ItemDetail, error) {
	it, a, err := s.load(ctx, userID, clientID, itemID, true)
	if err != nil {
		return nil, err
	}
	from := it.Status
	if !domain.CanMove(from, to, moveCtx(it, a)) {
		if domain.IsReviewMove(from, to) && a.Role == domain.RoleManager {
			return nil, apperr.New(apperr.Unauthorized, msgSelfApprove)
		}
		return nil, apperr.New(apperr.Unauthorized, msgMoveDenied)
	}
	note = trim(note)
	if utf8.RuneCountInString(note) > 1000 || (to == domain.ItemRevision && note == "") {
		return nil, apperr.Field("note", msgNoteRequired)
	}
	if to == domain.ItemApproved && from == domain.ItemReview && it.Caption == "" && it.Format != domain.FormatStory {
		return nil, apperr.New(apperr.Validation, msgCaptionRequired)
	}
	now := s.now()
	domain.MoveStamps(it, to, userID, now)
	it.UpdatedBy = &userID
	err = s.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.UpdateItem(ctx, *it); err != nil {
			return err
		}
		f, t := string(from), string(to)
		if err := tx.InsertActivity(ctx, domain.ItemActivity{ID: ids.New(), ItemID: it.ID, UserID: &userID, Type: domain.ActMoved, FromStatus: &f, ToStatus: &t, Note: optID(note)}); err != nil {
			return err
		}
		if note == "" {
			return nil
		}
		kind := domain.DecisionComment
		switch to {
		case domain.ItemRevision:
			kind = domain.DecisionChangesRequested
		case domain.ItemApproved:
			kind = domain.DecisionApproved
		}
		return tx.InsertItemComment(ctx, domain.ItemComment{ID: ids.New(), ItemID: it.ID, UserID: &userID, Body: note, Kind: kind, Round: it.RevisionRound})
	})
	if err != nil {
		return nil, err
	}
	s.notifyMove(ctx, it, a, to, note)
	return s.Get(ctx, userID, clientID, itemID)
}

func (s *Items) notifyMove(ctx context.Context, it *domain.Item, a *Access, to domain.ItemStatus, note string) {
	in := NotifyInput{ClientID: it.ClientID, ExceptUserID: a.UserID, Href: itemHref(a.Slug, it.ID), Body: note}
	others := func(ids ...*string) []string {
		var out []string
		for _, id := range ids {
			if id != nil && *id != a.UserID {
				out = append(out, *id)
			}
		}
		return out
	}
	switch to {
	case domain.ItemReview:
		in.Audience, in.Type, in.Title = AudienceClientApprovers, NotifItemReview, "Konten menunggu review: "+it.Title
	case domain.ItemRevision:
		in.UserIDs, in.Type, in.Title = others(it.CopywriterID, it.DesignerID), NotifItemRevision, "Konten perlu revisi: "+it.Title
	case domain.ItemApproved:
		in.UserIDs, in.Type, in.Title = others(it.CopywriterID, it.DesignerID), NotifItemApproved, "Konten disetujui: "+it.Title
	case domain.ItemDesign:
		in.UserIDs, in.Type, in.Title = others(it.DesignerID), NotifItemDesign, "Konten siap didesain: "+it.Title
	default:
		return
	}
	s.notify(ctx, in)
}

// Revise stores a new caption version. Revising approved copy sends the
// item back to review.
func (s *Items) Revise(ctx context.Context, userID, clientID, itemID string, c CaptionInput) (*ItemDetail, error) {
	it, _, err := s.load(ctx, userID, clientID, itemID, true)
	if err != nil {
		return nil, err
	}
	if domain.IsLocked(it.Status) {
		return nil, apperr.New(apperr.Validation, msgItemLocked)
	}
	if err := validateCaption(&c); err != nil {
		return nil, err
	}
	from := it.Status
	it.Hook, it.Caption, it.Hashtags = c.Hook, c.Caption, c.Hashtags
	it.CaptionVersion++
	it.UpdatedBy = &userID
	if from == domain.ItemApproved {
		it.Status = domain.ItemReview
		it.ApprovedAt, it.ApprovedBy = nil, nil
	}
	err = s.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.UpdateItem(ctx, *it); err != nil {
			return err
		}
		if err := tx.InsertRevision(ctx, domain.CaptionRevision{ID: ids.New(), ItemID: it.ID, Version: it.CaptionVersion, Caption: c.Caption, Hook: c.Hook, Hashtags: c.Hashtags, Note: c.Note, CreatedBy: &userID}); err != nil {
			return err
		}
		act := domain.ItemActivity{ID: ids.New(), ItemID: it.ID, UserID: &userID, Type: domain.ActRevised, Note: optID(c.Note)}
		if it.Status != from {
			f, t := string(from), string(it.Status)
			act.FromStatus, act.ToStatus = &f, &t
		}
		return tx.InsertActivity(ctx, act)
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, clientID, itemID)
}

// Comment is open to every member, VIEWER included.
func (s *Items) Comment(ctx context.Context, userID, clientID, itemID, body string) (*domain.ItemComment, error) {
	body = trim(body)
	if n := utf8.RuneCountInString(body); n < 1 || n > 2000 {
		return nil, apperr.Field("body", msgCommentRequired)
	}
	it, a, err := s.load(ctx, userID, clientID, itemID, false)
	if err != nil {
		return nil, err
	}
	c := domain.ItemComment{ID: ids.New(), ItemID: it.ID, UserID: &userID, Body: body, Kind: domain.DecisionComment, Round: it.RevisionRound, CreatedAt: s.now()}
	if err := s.Store.InsertItemComment(ctx, c); err != nil {
		return nil, err
	}
	if it.CopywriterID != nil && *it.CopywriterID != userID {
		s.notify(ctx, NotifyInput{UserIDs: []string{*it.CopywriterID}, ClientID: clientID, Type: NotifItemComment, Title: "Komentar baru: " + it.Title, Body: body, Href: itemHref(a.Slug, it.ID)})
	}
	return &c, nil
}
