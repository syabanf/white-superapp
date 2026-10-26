package app

import (
	"context"
	"errors"
	"sort"
	"time"
	"unicode/utf8"

	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// Plans is the editorial plan (EP) workflow: brief, AI draft, review and
// approval, comments.
type Plans struct{ Deps }

func NewPlans(d Deps) *Plans { return &Plans{d} }

type PlanDetail struct {
	Plan     domain.Plan
	Items    []domain.Item
	Comments []domain.PlanComment
	Names    map[string]string
}

func validateBrief(b *domain.Brief) error {
	v := validate.New()
	v.MaxLen("goals", b.Goals, 2000, "Tujuan terlalu panjang (maksimal 2.000 karakter).")
	v.MaxLen("tone", b.Tone, 500, "Tone terlalu panjang (maksimal 500 karakter).")
	v.MaxLen("notes", b.Notes, 2000, "Catatan terlalu panjang (maksimal 2.000 karakter).")
	pillars := []string{}
	seen := map[string]bool{}
	for _, p := range b.Pillars {
		p = trim(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		v.MaxLen("pillars", p, 80, "Pilar terlalu panjang (maksimal 80 karakter).")
		pillars = append(pillars, p)
	}
	if len(pillars) > 10 {
		v.Add("pillars", msgTooManyPillars)
	}
	freq := []domain.Frequency{}
	byPlatform := map[domain.Platform]bool{}
	for _, f := range b.Frequency {
		if !domain.IsPlatform(f.Platform) {
			v.Add("frequency", msgPlatformInvalid)
			continue
		}
		if f.PerWeek < 0 || f.PerWeek > 21 {
			v.Add("frequency", msgFrequencyInvalid)
			continue
		}
		if byPlatform[f.Platform] || f.PerWeek == 0 {
			continue
		}
		byPlatform[f.Platform] = true
		freq = append(freq, f)
	}
	if v.HasErrors() {
		return v.Err()
	}
	b.Goals, b.Tone, b.Notes, b.Pillars, b.Frequency = trim(b.Goals), trim(b.Tone), trim(b.Notes), pillars, freq
	return nil
}

func (p *Plans) List(ctx context.Context, userID, clientID string) ([]domain.Plan, error) {
	if _, err := p.access(ctx, userID, clientID); err != nil {
		return nil, err
	}
	return p.Store.ListPlans(ctx, clientID)
}

func (p *Plans) Create(ctx context.Context, userID, clientID, month, title string, brief domain.Brief) (*PlanDetail, error) {
	m, ok := domain.ParseMonth(month)
	if !ok {
		return nil, apperr.Field("month", msgMonthInvalid)
	}
	title = trim(title)
	if utf8.RuneCountInString(title) > 160 {
		return nil, apperr.Field("title", msgTitleTooLong)
	}
	if title == "" {
		title = "EP " + m.Label()
	}
	if err := validateBrief(&brief); err != nil {
		return nil, err
	}
	if _, err := p.manage(ctx, userID, clientID); err != nil {
		return nil, err
	}
	plan := domain.Plan{ID: ids.New(), ClientID: clientID, Month: m, Title: title, Status: domain.PlanDraft, Brief: brief, CreatedBy: &userID}
	if err := p.Store.InsertPlan(ctx, plan); err != nil {
		var u *domain.ErrUnique
		if errors.As(err, &u) {
			return nil, apperr.Field("month", msgPlanExists)
		}
		return nil, err
	}
	return p.Get(ctx, userID, clientID, plan.ID)
}

func (p *Plans) Get(ctx context.Context, userID, clientID, planID string) (*PlanDetail, error) {
	if _, err := p.access(ctx, userID, clientID); err != nil {
		return nil, err
	}
	plan, err := ensureFound(p.Store.GetPlan(ctx, clientID, planID))(msgPlanNotFound)
	if err != nil {
		return nil, err
	}
	d := &PlanDetail{Plan: *plan}
	if d.Items, err = p.Store.ListItems(ctx, clientID, domain.ItemFilter{PlanID: planID}); err != nil {
		return nil, err
	}
	for _, it := range d.Items {
		d.Plan.ItemCount++
		if it.Status == domain.ItemPublished {
			d.Plan.PublishedCount++
		}
	}
	if d.Comments, err = p.Store.ListPlanComments(ctx, planID); err != nil {
		return nil, err
	}
	people := make([]*string, 0, len(d.Comments))
	for _, c := range d.Comments {
		people = append(people, c.UserID)
	}
	d.Names = p.userNames(ctx, people...)
	return d, nil
}

// loadForWrite returns the plan and access for a MANAGER/ADMIN.
func (p *Plans) loadForWrite(ctx context.Context, userID, clientID, planID string) (*domain.Plan, *Access, error) {
	a, err := p.manage(ctx, userID, clientID)
	if err != nil {
		return nil, nil, err
	}
	plan, err := ensureFound(p.Store.GetPlan(ctx, clientID, planID))(msgPlanNotFound)
	if err != nil {
		return nil, nil, err
	}
	return plan, a, nil
}

func (p *Plans) Update(ctx context.Context, userID, clientID, planID string, title *string, brief *domain.Brief) (*PlanDetail, error) {
	plan, _, err := p.loadForWrite(ctx, userID, clientID, planID)
	if err != nil {
		return nil, err
	}
	if !domain.CanEditBrief(plan.Status) {
		return nil, apperr.New(apperr.Validation, msgPlanNotDraft)
	}
	if title != nil {
		t := trim(*title)
		if t == "" || utf8.RuneCountInString(t) > 160 {
			return nil, apperr.Field("title", msgTitleTooLong)
		}
		plan.Title = t
	}
	if brief != nil {
		if err := validateBrief(brief); err != nil {
			return nil, err
		}
		plan.Brief = *brief
	}
	plan.UpdatedBy = &userID
	if err := p.Store.UpdatePlan(ctx, *plan); err != nil {
		return nil, err
	}
	return p.Get(ctx, userID, clientID, planID)
}

func (p *Plans) Delete(ctx context.Context, userID, clientID, planID string) error {
	plan, _, err := p.loadForWrite(ctx, userID, clientID, planID)
	if err != nil {
		return err
	}
	if plan.Status != domain.PlanDraft {
		return apperr.New(apperr.Validation, msgPlanNotDraft)
	}
	return p.Store.SoftDeletePlan(ctx, plan.ID, userID)
}

// topPosts reads the client's best posts of the last 90 days (best effort).
func (p *Plans) topPosts(ctx context.Context, clientID string) []TopPost {
	today := domain.Day(p.now())
	posts, err := p.Analytics.Posts(ctx, clientID, nil, today.AddDate(0, 0, -90).Format(time.DateOnly), today.Format(time.DateOnly))
	if err != nil {
		p.log().Info("top posts unavailable", "client", clientID, "err", err)
		return nil
	}
	sort.SliceStable(posts, func(i, j int) bool { return posts[i].Metrics.Engagements() > posts[j].Metrics.Engagements() })
	out := []TopPost{}
	for _, sp := range posts[:min(5, len(posts))] {
		caption := sp.Caption
		if r := []rune(caption); len(r) > 300 {
			caption = string(r[:300])
		}
		out = append(out, TopPost{Caption: caption, MediaType: sp.MediaType, PublishedAt: sp.PublishedAt.Format(time.DateOnly), Engagements: sp.Metrics.Engagements()})
	}
	return out
}

// DefaultDeadlines puts copy three days, design two days and review one
// day before the planned date.
func DefaultDeadlines(planned time.Time) (copyDue, designDue, reviewDue *time.Time) {
	c, d, r := planned.AddDate(0, 0, -3), planned.AddDate(0, 0, -2), planned.AddDate(0, 0, -1)
	return &c, &d, &r
}

// Generate asks the AI for item proposals from the brief and recent top
// posts and adds them as IDEA items.
func (p *Plans) Generate(ctx context.Context, userID, clientID, planID string, replace bool) (*PlanDetail, error) {
	plan, a, err := p.loadForWrite(ctx, userID, clientID, planID)
	if err != nil {
		return nil, err
	}
	if !domain.CanEditBrief(plan.Status) {
		return nil, apperr.New(apperr.Validation, msgPlanNotDraft)
	}
	clientName := a.Name
	if clientName == "" {
		clientName = a.Slug
	}
	drafts, model, err := p.AI.DraftPlan(ctx, PlanPrompt{ClientName: clientName, Month: plan.Month, Brief: plan.Brief, TopPosts: p.topPosts(ctx, clientID)})
	if err != nil {
		return nil, err
	}
	now := p.now()
	plan.AIModel, plan.GeneratedAt, plan.UpdatedBy = &model, &now, &userID
	err = p.Store.WithTx(ctx, func(tx domain.Store) error {
		if replace {
			if err := tx.SoftDeleteIdeaItems(ctx, plan.ID, userID); err != nil {
				return err
			}
		}
		for _, dr := range drafts {
			if !plan.Month.Contains(dr.PlannedDate) || !domain.IsPlatform(dr.Platform) || !domain.IsFormat(dr.Format) {
				continue
			}
			it := domain.Item{ID: ids.New(), PlanID: plan.ID, ClientID: clientID, PlannedDate: dr.PlannedDate, Platform: dr.Platform, Format: dr.Format, Pillar: dr.Pillar, Title: dr.Title, VisualBrief: dr.VisualBrief, Status: domain.ItemIdea, CreatedBy: &userID}
			it.CopyDue, it.DesignDue, it.ReviewDue = DefaultDeadlines(dr.PlannedDate)
			c := CaptionInput{Hook: dr.Hook, Caption: dr.Caption, Hashtags: dr.Hashtags, Note: "Draft AI"}
			if err := validateCaption(&c); err != nil {
				continue
			}
			if err := insertItem(ctx, tx, it, c, userID, "Dibuat dari draft AI ("+model+")"); err != nil {
				return err
			}
		}
		return tx.UpdatePlan(ctx, *plan)
	})
	if err != nil {
		return nil, err
	}
	return p.Get(ctx, userID, clientID, planID)
}

func (p *Plans) Submit(ctx context.Context, userID, clientID, planID string) (*PlanDetail, error) {
	plan, a, err := p.loadForWrite(ctx, userID, clientID, planID)
	if err != nil {
		return nil, err
	}
	if plan.Status != domain.PlanDraft {
		return nil, apperr.New(apperr.Validation, msgPlanNotDraft)
	}
	items, err := p.Store.ListItems(ctx, clientID, domain.ItemFilter{PlanID: planID})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, apperr.New(apperr.Validation, msgPlanEmpty)
	}
	now := p.now()
	plan.Status, plan.SubmittedBy, plan.SubmittedAt, plan.UpdatedBy = domain.PlanInReview, &userID, &now, &userID
	if err := p.Store.UpdatePlan(ctx, *plan); err != nil {
		return nil, err
	}
	p.notify(ctx, NotifyInput{ClientID: clientID, Audience: AudienceClientApprovers, ExceptUserID: userID, Type: NotifPlanReview, Title: "Editorial plan menunggu review: " + plan.Title, Href: planHref(a.Slug, plan.ID)})
	return p.Get(ctx, userID, clientID, planID)
}

func checkNote(note string, required bool) (string, error) {
	note = trim(note)
	if utf8.RuneCountInString(note) > 1000 || (required && note == "") {
		return "", apperr.Field("note", msgNoteRequired)
	}
	return note, nil
}

// review loads an IN_REVIEW plan for a reviewer who is not the submitter
// (unless ADMIN).
func (p *Plans) review(ctx context.Context, userID, clientID, planID string) (*domain.Plan, *Access, error) {
	plan, a, err := p.loadForWrite(ctx, userID, clientID, planID)
	if err != nil {
		return nil, nil, err
	}
	if plan.Status != domain.PlanInReview {
		return nil, nil, apperr.New(apperr.Validation, msgPlanNotInReview)
	}
	if plan.SubmittedBy != nil && *plan.SubmittedBy == userID && !domain.CanApproveOwn(a.Role) {
		return nil, nil, apperr.New(apperr.Unauthorized, msgSelfApprove)
	}
	return plan, a, nil
}

func (p *Plans) Approve(ctx context.Context, userID, clientID, planID, note string) (*PlanDetail, error) {
	note, err := checkNote(note, false)
	if err != nil {
		return nil, err
	}
	plan, a, err := p.review(ctx, userID, clientID, planID)
	if err != nil {
		return nil, err
	}
	now := p.now()
	plan.Status, plan.ApprovedBy, plan.ApprovedAt, plan.ReviewNote, plan.UpdatedBy = domain.PlanApproved, &userID, &now, optID(note), &userID
	if err := p.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.UpdatePlan(ctx, *plan); err != nil {
			return err
		}
		return p.insertNote(ctx, tx, plan.ID, userID, note)
	}); err != nil {
		return nil, err
	}
	p.notifySubmitter(ctx, plan, a, NotifPlanApproved, "Editorial plan disetujui: "+plan.Title, note)
	return p.Get(ctx, userID, clientID, planID)
}

func (p *Plans) RequestChanges(ctx context.Context, userID, clientID, planID, note string) (*PlanDetail, error) {
	note, err := checkNote(note, true)
	if err != nil {
		return nil, err
	}
	plan, a, err := p.review(ctx, userID, clientID, planID)
	if err != nil {
		return nil, err
	}
	plan.Status, plan.ReviewNote, plan.UpdatedBy = domain.PlanDraft, &note, &userID
	if err := p.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.UpdatePlan(ctx, *plan); err != nil {
			return err
		}
		return p.insertNote(ctx, tx, plan.ID, userID, note)
	}); err != nil {
		return nil, err
	}
	p.notifySubmitter(ctx, plan, a, NotifPlanChanges, "Editorial plan perlu revisi: "+plan.Title, note)
	return p.Get(ctx, userID, clientID, planID)
}

func (p *Plans) insertNote(ctx context.Context, tx domain.Store, planID, userID, note string) error {
	if note == "" {
		return nil
	}
	return tx.InsertPlanComment(ctx, domain.PlanComment{ID: ids.New(), PlanID: planID, UserID: &userID, Body: note})
}

func (p *Plans) notifySubmitter(ctx context.Context, plan *domain.Plan, a *Access, typ, title, body string) {
	if plan.SubmittedBy == nil || *plan.SubmittedBy == a.UserID {
		return
	}
	p.notify(ctx, NotifyInput{UserIDs: []string{*plan.SubmittedBy}, ClientID: plan.ClientID, Type: typ, Title: title, Body: body, Href: planHref(a.Slug, plan.ID)})
}

// Comment is open to every member, VIEWER included.
func (p *Plans) Comment(ctx context.Context, userID, clientID, planID, body string) (*domain.PlanComment, error) {
	body = trim(body)
	if n := utf8.RuneCountInString(body); n < 1 || n > 2000 {
		return nil, apperr.Field("body", msgCommentRequired)
	}
	if _, err := p.access(ctx, userID, clientID); err != nil {
		return nil, err
	}
	plan, err := ensureFound(p.Store.GetPlan(ctx, clientID, planID))(msgPlanNotFound)
	if err != nil {
		return nil, err
	}
	c := domain.PlanComment{ID: ids.New(), PlanID: plan.ID, UserID: &userID, Body: body, CreatedAt: p.now()}
	if err := p.Store.InsertPlanComment(ctx, c); err != nil {
		return nil, err
	}
	return &c, nil
}
