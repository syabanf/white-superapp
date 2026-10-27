// Package app holds the recruitment use cases: job openings, candidates
// and their pipeline, psikotes, AI and scheduled interviews, offers, the
// public portals and the scheduled reminders.
package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Directory is the slice of the hr EmployeeService this service reads.
type Directory interface {
	Departments(ctx context.Context) ([]domain.Department, error)
	// Employee is nil when the employee does not exist.
	Employee(ctx context.Context, id string) (*domain.Employee, error)
	Employees(ctx context.Context) ([]domain.Employee, error)
}

// Notice is one in-app notification (identity NotificationService.Notify).
type Notice struct {
	UserIDs []string
	Type    string
	Title   string
	Body    string
	Href    string
}

// Notifier delivers notices. WhatsApp and email can be added behind it.
type Notifier interface {
	Notify(ctx context.Context, n Notice) error
}

// Audience lists the active identity users holding a permission (admins
// included).
type Audience interface {
	PermissionHolders(ctx context.Context, perm string) ([]string, error)
}

// AI is OpenRouter when a key is configured, the deterministic mock
// otherwise. The model id is stored with each result.
type AI interface {
	// InterviewStep is nil when no AI is configured or the call failed; the
	// interview then falls back to the topic order.
	InterviewStep(ctx context.Context, t domain.Transcript) *domain.AIStep
	SummarizeInterview(ctx context.Context, t domain.Transcript) (domain.InterviewSummary, string, error)
	AnalyzeCandidate(ctx context.Context, in domain.AnalysisInput) (domain.CandidateAnalysis, string, error)
}

// FileStore keeps CVs and drawings.
type FileStore interface {
	Put(ctx context.Context, key string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

// PDFRenderer renders the pipeline report and the offer letter.
type PDFRenderer interface {
	PipelineReport(r domain.PipelineReport, now time.Time) ([]byte, error)
	OfferLetter(l domain.OfferLetter) ([]byte, error)
}

// Deps is shared by every use-case struct.
type Deps struct {
	Store     domain.Store
	Directory Directory
	Notifier  Notifier
	AI        AI
	Files     FileStore
	PDF       PDFRenderer
	Events    events.Publisher
	Clock     clock.Clock
	Logger    *slog.Logger
	// Audience resolves the recruiters: holders of recruitment.write. Nil
	// leaves only Recruiters.
	Audience Audience
	// Recruiters are extra identity users notified about recruitment events
	// (RECRUITMENT_NOTIFY_USER_IDS), unioned with Audience; the opening's
	// hiring manager is added per notice.
	Recruiters []string
	// CompanyName prints on offers and reports.
	CompanyName string
}

type core struct{ Deps }

func (c core) now() time.Time {
	if c.Clock == nil {
		return time.Now().UTC()
	}
	return c.Clock.Now()
}

func (c core) log() *slog.Logger {
	if c.Logger == nil {
		return slog.Default()
	}
	return c.Logger
}

func (c core) company() string {
	if c.CompanyName == "" {
		return "Mixist"
	}
	return c.CompanyName
}

func newID() string { return ids.New() }

// newPortalToken is crypto.randomBytes(32).toString("hex").
func newPortalToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ─── staff ───────────────────────────────────────────────────

// Actor is the staff member a change is attributed to.
type Actor struct {
	UserID string
	Name   string
	// CanWrite also decides whether portal tokens are shown (TOKEN_ROLES).
	CanWrite bool
}

func (a *Actor) id() *string { return &a.UserID }

// staff resolves the caller: recruitment.read for reads, recruitment.write
// for writes (write implies read).
func staff(ctx context.Context, write bool) (*Actor, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	a := &Actor{UserID: p.UserID, Name: p.Name, CanWrite: p.Can(authn.PermRecruitWrite)}
	if write && !a.CanWrite {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	if !a.CanWrite && !p.Can(authn.PermRecruitRead) {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	return a, nil
}

// Messages shared by several use cases.
const (
	MsgCandidateNotFound = "Kandidat tidak ditemukan"
	MsgJobNotFound       = "Lowongan tidak ditemukan"
	MsgSessionNotFound   = "Sesi tidak ditemukan"
)

func notFound(msg string) error { return apperr.New(apperr.NotFound, msg) }

func conflict(msg string) error { return apperr.New(apperr.Conflict, msg) }

func invalid(msg string) error { return apperr.New(apperr.Validation, msg) }

func (c core) requireCandidate(ctx context.Context, id string) (*domain.Candidate, error) {
	if id == "" {
		return nil, invalid("ID kandidat tidak valid")
	}
	cand, err := c.Store.GetCandidate(ctx, id)
	if err != nil {
		return nil, err
	}
	if cand == nil {
		return nil, notFound(MsgCandidateNotFound)
	}
	return cand, nil
}

// activity writes a pipeline history line. by nil with name "" is an
// anonymous portal action; name "Sistem" marks the service itself.
func (c core) activity(ctx context.Context, store domain.Store, candidateID, typ, desc string, by *string, name string) error {
	return store.InsertActivity(ctx, domain.Activity{ID: newID(), CandidateID: candidateID, Type: typ, Description: desc,
		CreatedBy: by, CreatedByName: name, CreatedAt: c.now()})
}

func (c core) staffActivity(ctx context.Context, store domain.Store, candidateID, typ, desc string, a *Actor) error {
	return c.activity(ctx, store, candidateID, typ, desc, a.id(), a.Name)
}

// ─── notifications ───────────────────────────────────────────

// Notification types.
const (
	NotifApplication       = "RECRUIT_APPLICATION"
	NotifInterviewSchedule = "RECRUIT_INTERVIEW_SCHEDULED"
	NotifInterviewReminder = "RECRUIT_INTERVIEW_REMINDER"
	NotifOfferResponse     = "RECRUIT_OFFER_RESPONSE"
	NotifOfferExpiring     = "RECRUIT_OFFER_EXPIRING"
	NotifPsikotesDone      = "RECRUIT_PSIKOTES_DONE"
	NotifAiInterviewDone   = "RECRUIT_AI_INTERVIEW_DONE"
	NotifHired             = "RECRUIT_HIRED"
)

// Links the notices open (Next.js routes of the ERP workspace).
const hrefInterviews = "/erp/recruitment/interviews"

func candidateHref(id string) string { return "/erp/recruitment/candidates/" + id }

// recipients are the recruiters (recruitment.write holders and
// RECRUITMENT_NOTIFY_USER_IDS) plus extra users, deduplicated. An identity
// outage falls back to the configured ids.
func (c core) recipients(ctx context.Context, extra ...*string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(id string) {
		if id != "" && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	for _, id := range c.Recruiters {
		add(id)
	}
	if c.Audience != nil {
		holders, err := c.Audience.PermissionHolders(ctx, authn.PermRecruitWrite)
		if err != nil {
			c.log().Warn("recruiter lookup failed", "err", err)
		}
		for _, id := range holders {
			add(id)
		}
	}
	for _, id := range extra {
		if id != nil {
			add(*id)
		}
	}
	return out
}

// hiringManager is the opening's hiring manager user (nil when none).
func (c core) hiringManager(ctx context.Context, cand *domain.Candidate) *string {
	if cand.JobOpeningID == nil {
		return nil
	}
	o, err := c.Store.GetJobOpening(ctx, *cand.JobOpeningID)
	if err != nil || o == nil {
		return nil
	}
	return o.HiringManagerUserID
}

// notify sends best effort: delivery failures are logged, never returned.
func (c core) notify(ctx context.Context, n Notice) {
	if len(n.UserIDs) == 0 || c.Notifier == nil {
		return
	}
	if err := c.Notifier.Notify(ctx, n); err != nil {
		c.log().Warn("notify failed", "type", n.Type, "err", err)
	}
}

// ─── portal rate limits ──────────────────────────────────────

// enforce is enforceRateLimit: RATE_LIMIT with the given message once key
// passed limit hits in the window.
func (c core) enforce(ctx context.Context, key string, limit int, window time.Duration, message string) error {
	hits, err := c.Store.HitRateLimit(ctx, key, c.now(), window)
	if err != nil {
		return err
	}
	if hits > limit {
		return apperr.New(apperr.RateLimit, message)
	}
	return nil
}

// ─── hire publication ────────────────────────────────────────

// publishHired sends white.recruitment.candidate.hired for a candidate who
// reached hired and marks it sent. A failed publish is retried by the
// daily tick.
func (c core) publishHired(ctx context.Context, cand domain.Candidate) {
	if c.Events == nil || cand.HiredEventAt != nil {
		return
	}
	var opening *domain.JobOpening
	if cand.JobOpeningID != nil {
		opening, _ = c.Store.GetJobOpening(ctx, *cand.JobOpeningID)
	}
	offer, err := c.Store.LatestAcceptedOffer(ctx, cand.ID)
	if err != nil {
		c.log().Warn("hired event: offer lookup failed", "candidate", cand.ID, "err", err)
		return
	}
	hiredAt := c.now()
	if cand.HiredAt != nil {
		hiredAt = *cand.HiredAt
	}
	if err := c.Events.Publish(ctx, events.RecruitmentCandidateHired, domain.HiredPayload(cand, opening, offer, hiredAt)); err != nil {
		c.log().Warn("hired event: publish failed", "candidate", cand.ID, "err", err)
		return
	}
	if err := c.Store.MarkHiredEvent(ctx, cand.ID, c.now()); err != nil {
		c.log().Warn("hired event: mark failed", "candidate", cand.ID, "err", err)
	}
}

// afterStatusChange publishes the hire and notifies recruiters when a
// candidate became hired.
func (c core) afterStatusChange(ctx context.Context, candidateID, status string) {
	if status != "hired" {
		return
	}
	cand, err := c.Store.GetCandidate(ctx, candidateID)
	if err != nil || cand == nil {
		return
	}
	c.publishHired(ctx, *cand)
	c.notify(ctx, Notice{UserIDs: c.recipients(ctx, c.hiringManager(ctx, cand)), Type: NotifHired,
		Title: "Kandidat diterima: " + cand.FullName,
		Body:  "Kandidat siap dibuatkan data karyawan di HR.", Href: candidateHref(cand.ID)})
}

// ─── small helpers ───────────────────────────────────────────

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func timePtr(t time.Time) *time.Time { return &t }
