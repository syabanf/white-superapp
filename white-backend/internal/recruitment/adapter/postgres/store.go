// Package postgres implements the recruitment ports on top of sqlc queries.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

type Store struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, q: sqlc.New(pool)}
}

var _ domain.Store = (*Store)(nil)

func (s *Store) WithTx(ctx context.Context, fn func(tx domain.Store) error) error {
	return pg.Tx(ctx, s.pool, func(tx pgx.Tx) error {
		return fn(&Store{pool: s.pool, q: s.q.WithTx(tx)})
	})
}

// wrap turns unique violations into *domain.ErrUnique.
func wrap(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return &domain.ErrUnique{Constraint: pgErr.ConstraintName}
	}
	return err
}

// one maps pgx.ErrNoRows to a nil result.
func one[R any, T any](r R, err error, conv func(R) T) (*T, error) {
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	v := conv(r)
	return &v, nil
}

func each[R any, T any](rows []R, err error, conv func(R) T) ([]T, error) {
	if err != nil {
		return nil, err
	}
	out := make([]T, len(rows))
	for i, r := range rows {
		out[i] = conv(r)
	}
	return out, nil
}

func affected(n int64, err error) (bool, error) { return n > 0, err }

// ─── conversions ─────────────────────────────────────────────

func str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func dayPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(domain.IsoDate)
	return &s
}

func toDay(s *string) *time.Time {
	if s == nil {
		return nil
	}
	t, ok := domain.ParseDay(*s)
	if !ok {
		return nil
	}
	return &t
}

func i32(n int) int32 { return int32(n) }

func i32Ptr(n *int) *int32 {
	if n == nil {
		return nil
	}
	v := int32(*n)
	return &v
}

func intPtr(n *int32) *int {
	if n == nil {
		return nil
	}
	v := int(*n)
	return &v
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("null")
	}
	return b
}

func stringsFrom(b []byte) []string {
	out := []string{}
	_ = json.Unmarshal(b, &out)
	return out
}

func searchPattern(s string) *string {
	if s = strings.TrimSpace(s); s == "" {
		return nil
	}
	p := domain.LikePattern(s)
	return &p
}

// ─── job openings ────────────────────────────────────────────

func jobFrom(r sqlc.RecruitmentJobOpening, candidates, hired int32) domain.JobOpening {
	return domain.JobOpening{ID: r.ID, Title: r.Title, Slug: r.Slug, DepartmentID: r.DepartmentID, Department: r.Department,
		Location: r.Location, EmploymentType: r.EmploymentType, WorkMode: r.WorkMode, Headcount: int(r.Headcount),
		Description: str(r.Description), Requirements: str(r.Requirements), Benefits: str(r.Benefits),
		SalaryMin: r.SalaryMin, SalaryMax: r.SalaryMax, HiringManagerUserID: r.HiringManagerUserID, Status: r.Status,
		PublishedAt: r.PublishedAt, ClosingDate: dayPtr(r.ClosingDate), CandidateCount: int(candidates), HiredCount: int(hired),
		CreatedBy: r.CreatedBy, UpdatedBy: r.UpdatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func (s *Store) ListJobOpenings(ctx context.Context, status, search string) ([]domain.JobOpening, error) {
	rows, err := s.q.ListJobOpenings(ctx, sqlc.ListJobOpeningsParams{Status: nilIfEmpty(status), Search: searchPattern(search)})
	return each(rows, err, func(r sqlc.ListJobOpeningsRow) domain.JobOpening {
		return jobFrom(r.RecruitmentJobOpening, r.CandidateCount, r.HiredCount)
	})
}

func (s *Store) GetJobOpening(ctx context.Context, id string) (*domain.JobOpening, error) {
	r, err := s.q.GetJobOpening(ctx, id)
	return one(r, err, func(r sqlc.GetJobOpeningRow) domain.JobOpening {
		return jobFrom(r.RecruitmentJobOpening, r.CandidateCount, r.HiredCount)
	})
}

func (s *Store) GetJobOpeningBySlug(ctx context.Context, slug string) (*domain.JobOpening, error) {
	r, err := s.q.GetJobOpeningBySlug(ctx, slug)
	return one(r, err, func(r sqlc.GetJobOpeningBySlugRow) domain.JobOpening { return jobFrom(r.RecruitmentJobOpening, 0, 0) })
}

func (s *Store) ListPublicJobs(ctx context.Context, today, search, department string) ([]domain.JobOpening, error) {
	day, _ := domain.ParseDay(today)
	rows, err := s.q.ListPublicJobs(ctx, sqlc.ListPublicJobsParams{Today: day, Search: searchPattern(search), Department: nilIfEmpty(department)})
	return each(rows, err, func(r sqlc.ListPublicJobsRow) domain.JobOpening { return jobFrom(r.RecruitmentJobOpening, 0, 0) })
}

func (s *Store) InsertJobOpening(ctx context.Context, o domain.JobOpening) error {
	return wrap(s.q.InsertJobOpening(ctx, sqlc.InsertJobOpeningParams{ID: o.ID, Title: o.Title, Slug: o.Slug, DepartmentID: o.DepartmentID,
		Department: o.Department, Location: o.Location, EmploymentType: o.EmploymentType, WorkMode: o.WorkMode,
		Headcount: i32(o.Headcount), Description: nilIfEmpty(o.Description), Requirements: nilIfEmpty(o.Requirements),
		Benefits: nilIfEmpty(o.Benefits), SalaryMin: o.SalaryMin, SalaryMax: o.SalaryMax, HiringManagerUserID: o.HiringManagerUserID,
		Status: o.Status, PublishedAt: o.PublishedAt, ClosingDate: toDay(o.ClosingDate), CreatedBy: o.CreatedBy}))
}

func (s *Store) UpdateJobOpening(ctx context.Context, o domain.JobOpening) (bool, error) {
	n, err := s.q.UpdateJobOpening(ctx, sqlc.UpdateJobOpeningParams{ID: o.ID, Title: o.Title, Slug: o.Slug, DepartmentID: o.DepartmentID,
		Department: o.Department, Location: o.Location, EmploymentType: o.EmploymentType, WorkMode: o.WorkMode,
		Headcount: i32(o.Headcount), Description: nilIfEmpty(o.Description), Requirements: nilIfEmpty(o.Requirements),
		Benefits: nilIfEmpty(o.Benefits), SalaryMin: o.SalaryMin, SalaryMax: o.SalaryMax, HiringManagerUserID: o.HiringManagerUserID,
		Status: o.Status, PublishedAt: o.PublishedAt, ClosingDate: toDay(o.ClosingDate), UpdatedBy: o.UpdatedBy})
	return affected(n, wrap(err))
}

func (s *Store) DeleteJobOpening(ctx context.Context, id, by string) (bool, error) {
	return affected(s.q.DeleteJobOpening(ctx, sqlc.DeleteJobOpeningParams{ID: id, UpdatedBy: nilIfEmpty(by)}))
}

// ─── candidates ──────────────────────────────────────────────

func candidateFrom(r sqlc.RecruitmentCandidate, title string) domain.Candidate {
	return domain.Candidate{ID: r.ID, FullName: r.FullName, Email: r.Email, Phone: r.Phone, Domicile: r.Domicile, Source: r.Source,
		JobOpeningID: r.JobOpeningID, JobOpeningTitle: title, Status: r.Status, Notes: str(r.Notes),
		LastExperience: str(r.LastExperience), LastEducation: str(r.LastEducation), Availability: str(r.Availability),
		ExpectedSalary: r.ExpectedSalary, CVKey: r.CvKey, CVFileName: str(r.CvFileName), CVContentType: str(r.CvContentType),
		HiredAt: r.HiredAt, HiredEventAt: r.HiredEventAt, CreatedBy: r.CreatedBy, UpdatedBy: r.UpdatedBy,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func (s *Store) ListCandidates(ctx context.Context, f domain.CandidateFilter) ([]domain.Candidate, int, error) {
	statuses := f.Statuses
	if statuses == nil {
		statuses = []string{}
	}
	rows, err := s.q.ListCandidates(ctx, sqlc.ListCandidatesParams{Statuses: statuses, JobOpeningID: nilIfEmpty(f.JobOpeningID),
		Search: searchPattern(f.Search), DateFrom: f.DateFrom, DateTo: f.DateTo, SortUpdated: f.SortUpdated,
		Lim: i32(f.Limit), Off: i32(f.Offset)})
	if err != nil {
		return nil, 0, err
	}
	out := make([]domain.Candidate, len(rows))
	total := 0
	for i, r := range rows {
		out[i] = candidateFrom(r.RecruitmentCandidate, r.JobOpeningTitle)
		total = int(r.Total)
	}
	return out, total, nil
}

func (s *Store) GetCandidate(ctx context.Context, id string) (*domain.Candidate, error) {
	r, err := s.q.GetCandidate(ctx, id)
	return one(r, err, func(r sqlc.GetCandidateRow) domain.Candidate {
		return candidateFrom(r.RecruitmentCandidate, r.JobOpeningTitle)
	})
}

func (s *Store) InsertCandidate(ctx context.Context, c domain.Candidate) error {
	return wrap(s.q.InsertCandidate(ctx, sqlc.InsertCandidateParams{ID: c.ID, FullName: c.FullName, Email: c.Email, Phone: c.Phone,
		Domicile: c.Domicile, Source: c.Source, JobOpeningID: c.JobOpeningID, Status: c.Status, Notes: nilIfEmpty(c.Notes),
		LastExperience: nilIfEmpty(c.LastExperience), LastEducation: nilIfEmpty(c.LastEducation),
		Availability: nilIfEmpty(c.Availability), ExpectedSalary: c.ExpectedSalary, CvKey: c.CVKey,
		CvFileName: nilIfEmpty(c.CVFileName), CvContentType: nilIfEmpty(c.CVContentType), CreatedBy: c.CreatedBy}))
}

func (s *Store) UpdateCandidateProfile(ctx context.Context, id string, f domain.CandidateFields, by string) (bool, error) {
	return affected(s.q.UpdateCandidateProfile(ctx, sqlc.UpdateCandidateProfileParams{ID: id, FullName: f.FullName, Email: f.Email,
		Phone: f.Phone, Domicile: f.Domicile, Source: f.Source, JobOpeningID: f.JobOpeningID, Notes: nilIfEmpty(f.Notes),
		LastExperience: nilIfEmpty(f.LastExperience), LastEducation: nilIfEmpty(f.LastEducation),
		Availability: nilIfEmpty(f.Availability), ExpectedSalary: f.ExpectedSalary, UpdatedBy: nilIfEmpty(by)}))
}

func (s *Store) SetCandidateStatus(ctx context.Context, id, status string, by *string) (bool, error) {
	return affected(s.q.SetCandidateStatus(ctx, sqlc.SetCandidateStatusParams{ID: id, Status: status, UpdatedBy: by}))
}

func (s *Store) MarkHiredEvent(ctx context.Context, id string, at time.Time) error {
	return s.q.MarkHiredEvent(ctx, sqlc.MarkHiredEventParams{ID: id, At: &at})
}

func (s *Store) PendingHiredEvents(ctx context.Context) ([]domain.Candidate, error) {
	rows, err := s.q.PendingHiredEvents(ctx)
	return each(rows, err, func(r sqlc.PendingHiredEventsRow) domain.Candidate {
		return candidateFrom(r.RecruitmentCandidate, r.JobOpeningTitle)
	})
}

func (s *Store) DeleteCandidate(ctx context.Context, id string) (bool, error) {
	return affected(s.q.DeleteCandidate(ctx, id))
}

func (s *Store) SetCandidateCV(ctx context.Context, id string, key, name, contentType *string, by string) (bool, error) {
	return affected(s.q.SetCandidateCV(ctx, sqlc.SetCandidateCVParams{ID: id, CvKey: key, CvFileName: name, CvContentType: contentType,
		UpdatedBy: nilIfEmpty(by)}))
}

func (s *Store) CandidateAppliedTo(ctx context.Context, email, jobOpeningID string) (bool, error) {
	return s.q.CandidateAppliedTo(ctx, sqlc.CandidateAppliedToParams{Email: email, JobOpeningID: &jobOpeningID})
}

func (s *Store) CountByStatus(ctx context.Context, jobOpeningID string) (map[string]int, error) {
	rows, err := s.q.CountCandidatesByStatus(ctx, nilIfEmpty(jobOpeningID))
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, r := range rows {
		out[r.Status] = int(r.N)
	}
	return out, nil
}

func (s *Store) CountBySource(ctx context.Context) ([]domain.Count, error) {
	rows, err := s.q.CountCandidatesBySource(ctx)
	return each(rows, err, func(r sqlc.CountCandidatesBySourceRow) domain.Count {
		return domain.Count{Key: r.Source, Label: domain.SourceLabel(r.Source), Count: int(r.N)}
	})
}

func (s *Store) Summary(ctx context.Context, now time.Time) (domain.Summary, error) {
	r, err := s.q.SummaryCounts(ctx, sqlc.SummaryCountsParams{Active: domain.ActiveStatuses, Since30: now.AddDate(0, 0, -30),
		Since90: now.AddDate(0, 0, -90), Now: now, Next7: now.AddDate(0, 0, 7)})
	if err != nil {
		return domain.Summary{}, err
	}
	out := domain.Summary{OpenJobs: int(r.OpenJobs), ActiveCandidates: int(r.ActiveCandidates), AppliedLast30: int(r.AppliedLast30),
		HiredLast90: int(r.HiredLast90), InterviewsNext7: int(r.InterviewsNext7), OpenOffers: int(r.OpenOffers)}
	if r.AvgDaysToHire >= 0 {
		avg := float64(int(r.AvgDaysToHire*10+0.5)) / 10
		out.AvgDaysToHire = &avg
	}
	return out, nil
}

func (s *Store) ListNotes(ctx context.Context, candidateID string) ([]domain.Note, error) {
	rows, err := s.q.ListNotes(ctx, candidateID)
	return each(rows, err, func(r sqlc.RecruitmentCandidateNote) domain.Note {
		return domain.Note{ID: r.ID, CandidateID: r.CandidateID, Content: r.Content, CreatedBy: r.CreatedBy,
			CreatedByName: str(r.CreatedByName), CreatedAt: r.CreatedAt}
	})
}

func (s *Store) InsertNote(ctx context.Context, n domain.Note) error {
	return s.q.InsertNote(ctx, sqlc.InsertNoteParams{ID: n.ID, CandidateID: n.CandidateID, Content: n.Content, CreatedBy: n.CreatedBy,
		CreatedByName: nilIfEmpty(n.CreatedByName), CreatedAt: n.CreatedAt})
}

func (s *Store) ListActivities(ctx context.Context, candidateID string) ([]domain.Activity, error) {
	rows, err := s.q.ListActivities(ctx, candidateID)
	return each(rows, err, func(r sqlc.RecruitmentCandidateActivity) domain.Activity {
		return domain.Activity{ID: r.ID, CandidateID: r.CandidateID, Type: r.ActivityType, Description: r.Description,
			CreatedBy: r.CreatedBy, CreatedByName: str(r.CreatedByName), CreatedAt: r.CreatedAt}
	})
}

func (s *Store) InsertActivity(ctx context.Context, a domain.Activity) error {
	return s.q.InsertActivity(ctx, sqlc.InsertActivityParams{ID: a.ID, CandidateID: a.CandidateID, ActivityType: a.Type,
		Description: a.Description, CreatedBy: a.CreatedBy, CreatedByName: nilIfEmpty(a.CreatedByName), CreatedAt: a.CreatedAt})
}

func (s *Store) GetScreening(ctx context.Context, candidateID string) (*domain.Screening, error) {
	r, err := s.q.GetScreening(ctx, candidateID)
	return one(r, err, func(r sqlc.RecruitmentCandidateScreening) domain.Screening {
		return domain.Screening{CandidateID: r.CandidateID, Contacted: r.Contacted, Interested: r.Interested,
			AvailabilityNote: str(r.AvailabilityNote), ConfirmedSalary: r.ConfirmedSalary, WillingShift: r.WillingShift,
			WillingPlacement: r.WillingPlacement, Notes: str(r.Notes), Recommendation: str(r.Recommendation),
			UpdatedBy: r.UpdatedBy, UpdatedByName: str(r.UpdatedByName), UpdatedAt: r.UpdatedAt}
	})
}

func (s *Store) UpsertScreening(ctx context.Context, x domain.Screening) error {
	return s.q.UpsertScreening(ctx, sqlc.UpsertScreeningParams{CandidateID: x.CandidateID, Contacted: x.Contacted,
		Interested: x.Interested, AvailabilityNote: nilIfEmpty(x.AvailabilityNote), ConfirmedSalary: x.ConfirmedSalary,
		WillingShift: x.WillingShift, WillingPlacement: x.WillingPlacement, Notes: nilIfEmpty(x.Notes),
		Recommendation: nilIfEmpty(x.Recommendation), UpdatedBy: x.UpdatedBy, UpdatedByName: nilIfEmpty(x.UpdatedByName)})
}

func (s *Store) GetPsikotesSummary(ctx context.Context, candidateID string) (*domain.PsikotesSummary, error) {
	r, err := s.q.GetPsikotesSummary(ctx, candidateID)
	return one(r, err, func(r sqlc.RecruitmentCandidatePsikotesSummary) domain.PsikotesSummary {
		return domain.PsikotesSummary{CandidateID: r.CandidateID, Recommendation: str(r.Recommendation), Notes: str(r.Notes),
			UpdatedBy: r.UpdatedBy, UpdatedByName: str(r.UpdatedByName), UpdatedAt: r.UpdatedAt}
	})
}

func (s *Store) UpsertPsikotesSummary(ctx context.Context, x domain.PsikotesSummary) error {
	return s.q.UpsertPsikotesSummary(ctx, sqlc.UpsertPsikotesSummaryParams{CandidateID: x.CandidateID,
		Recommendation: nilIfEmpty(x.Recommendation), Notes: nilIfEmpty(x.Notes), UpdatedBy: x.UpdatedBy,
		UpdatedByName: nilIfEmpty(x.UpdatedByName)})
}

func (s *Store) GetAnalysis(ctx context.Context, candidateID string) (*domain.CandidateAnalysis, error) {
	r, err := s.q.GetAnalysis(ctx, candidateID)
	return one(r, err, func(r sqlc.RecruitmentCandidateAiAnalysis) domain.CandidateAnalysis {
		return domain.CandidateAnalysis{CandidateID: r.CandidateID, Summary: r.Summary, MatchScore: int(r.MatchScore),
			MatchReason: r.MatchReason, Strengths: stringsFrom(r.Strengths), Gaps: stringsFrom(r.Gaps), JobContext: r.JobContext,
			Model: r.Model, UpdatedAt: r.UpdatedAt}
	})
}

func (s *Store) UpsertAnalysis(ctx context.Context, a domain.CandidateAnalysis) error {
	return s.q.UpsertAnalysis(ctx, sqlc.UpsertAnalysisParams{CandidateID: a.CandidateID, Summary: a.Summary, MatchScore: i32(a.MatchScore),
		MatchReason: a.MatchReason, Strengths: mustJSON(nonNil(a.Strengths)), Gaps: mustJSON(nonNil(a.Gaps)),
		JobContext: a.JobContext, Model: a.Model})
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// ─── psikotes bank ───────────────────────────────────────────

func instrumentFrom(r sqlc.RecruitmentPsikotesInstrument, active int32) domain.Instrument {
	return domain.Instrument{ID: r.ID, Code: r.Code, Name: r.Name, Kind: r.Kind, DurationSeconds: int(r.DurationSeconds),
		QuestionCount: intPtr(r.QuestionCount), Shuffle: r.Shuffle, Instructions: r.Instructions, IsActive: r.IsActive,
		SortOrder: int(r.SortOrder), ActiveQuestions: int(active)}
}

func (s *Store) ListInstruments(ctx context.Context) ([]domain.Instrument, error) {
	rows, err := s.q.ListInstruments(ctx)
	return each(rows, err, func(r sqlc.ListInstrumentsRow) domain.Instrument {
		return instrumentFrom(r.RecruitmentPsikotesInstrument, r.ActiveQuestions)
	})
}

func (s *Store) GetInstrument(ctx context.Context, id string) (*domain.Instrument, error) {
	r, err := s.q.GetInstrument(ctx, id)
	return one(r, err, func(r sqlc.GetInstrumentRow) domain.Instrument {
		return instrumentFrom(r.RecruitmentPsikotesInstrument, r.ActiveQuestions)
	})
}

func (s *Store) ActiveInstruments(ctx context.Context, ids []string) ([]domain.Instrument, error) {
	rows, err := s.q.ActiveInstruments(ctx, ids)
	return each(rows, err, func(r sqlc.ActiveInstrumentsRow) domain.Instrument {
		return instrumentFrom(r.RecruitmentPsikotesInstrument, 0)
	})
}

func (s *Store) UpdateInstrument(ctx context.Context, id string, p domain.InstrumentPatch, by string) (bool, error) {
	return affected(s.q.UpdateInstrument(ctx, sqlc.UpdateInstrumentParams{ID: id, Name: p.Name, IsActive: p.IsActive,
		DurationSeconds: i32Ptr(p.DurationSeconds), QuestionCount: i32Ptr(p.QuestionCount), ClearQuestionCount: p.ClearQuestionCount,
		Shuffle: p.Shuffle, Instructions: p.Instructions, UpdatedBy: nilIfEmpty(by)}))
}

type answerKey struct {
	Correct string `json:"correct"`
}

func questionFrom(r sqlc.RecruitmentPsikotesQuestion) domain.Question {
	q := domain.Question{ID: r.ID, InstrumentID: r.InstrumentID, Body: r.Body, SortOrder: int(r.SortOrder), IsActive: r.IsActive}
	if raw := strings.TrimSpace(string(r.Options)); strings.HasPrefix(raw, "{") {
		var pair domain.PapiPair
		if json.Unmarshal(r.Options, &pair) == nil {
			q.Pair = &pair
		}
	} else {
		q.Options = []domain.McqOption{}
		_ = json.Unmarshal(r.Options, &q.Options)
	}
	var key answerKey
	if len(r.AnswerKey) > 0 && json.Unmarshal(r.AnswerKey, &key) == nil {
		q.CorrectKey = key.Correct
	}
	return q
}

// questionColumns renders options and answer_key as NüHabit stored them.
func questionColumns(q domain.Question) (options, key []byte) {
	if q.Pair != nil {
		return mustJSON(q.Pair), nil
	}
	opts := q.Options
	if opts == nil {
		opts = []domain.McqOption{}
	}
	if q.CorrectKey != "" {
		key = mustJSON(answerKey{Correct: q.CorrectKey})
	}
	return mustJSON(opts), key
}

func (s *Store) ListQuestions(ctx context.Context, instrumentID string) ([]domain.Question, error) {
	rows, err := s.q.ListQuestions(ctx, instrumentID)
	return each(rows, err, questionFrom)
}

func (s *Store) GetQuestion(ctx context.Context, id string) (*domain.Question, error) {
	r, err := s.q.GetQuestion(ctx, id)
	return one(r, err, questionFrom)
}

func (s *Store) QuestionsByIDs(ctx context.Context, ids []string) ([]domain.Question, error) {
	if ids == nil {
		ids = []string{}
	}
	rows, err := s.q.QuestionsByIDs(ctx, ids)
	return each(rows, err, questionFrom)
}

func (s *Store) InsertQuestion(ctx context.Context, q domain.Question, by string) error {
	opts, key := questionColumns(q)
	return s.q.InsertQuestion(ctx, sqlc.InsertQuestionParams{ID: q.ID, InstrumentID: q.InstrumentID, Body: q.Body, Options: opts,
		AnswerKey: key, SortOrder: i32(q.SortOrder), IsActive: q.IsActive, CreatedBy: nilIfEmpty(by)})
}

func (s *Store) UpdateQuestion(ctx context.Context, q domain.Question, by string) (bool, error) {
	opts, key := questionColumns(q)
	return affected(s.q.UpdateQuestion(ctx, sqlc.UpdateQuestionParams{ID: q.ID, Body: q.Body, Options: opts, AnswerKey: key,
		SortOrder: i32(q.SortOrder), IsActive: q.IsActive, UpdatedBy: nilIfEmpty(by)}))
}

func (s *Store) DeleteQuestion(ctx context.Context, id string) (bool, error) {
	return affected(s.q.DeleteQuestion(ctx, id))
}

func (s *Store) DrawQuestionIDs(ctx context.Context, instrumentID string, shuffle bool, limit int) ([]string, error) {
	return s.q.DrawQuestionIDs(ctx, sqlc.DrawQuestionIDsParams{InstrumentID: instrumentID, Shuffle: shuffle, Lim: i32(limit)})
}

// ─── psikotes sessions ───────────────────────────────────────

func psikotesSessionFrom(r sqlc.RecruitmentPsikotesSession, name, position string) domain.PsikotesSession {
	return domain.PsikotesSession{ID: r.ID, CandidateID: r.CandidateID, CandidateName: name, PositionTitle: position, Token: r.Token,
		Status: r.Status, WebcamConsent: r.WebcamConsent, InvitedAt: r.InvitedAt, ExpiresAt: r.ExpiresAt, StartedAt: r.StartedAt,
		CompletedAt: r.CompletedAt, CreatedByName: str(r.CreatedByName)}
}

func (s *Store) InsertPsikotesSession(ctx context.Context, x domain.PsikotesSession) error {
	if err := s.q.InsertPsikotesSession(ctx, sqlc.InsertPsikotesSessionParams{ID: x.ID, CandidateID: x.CandidateID, Token: x.Token,
		Status: x.Status, InvitedAt: x.InvitedAt, ExpiresAt: x.ExpiresAt, CreatedBy: x.CreatedBy, CreatedByName: nilIfEmpty(x.CreatedByName)}); err != nil {
		return err
	}
	for _, t := range x.Tests {
		if err := s.q.InsertSessionTest(ctx, sqlc.InsertSessionTestParams{ID: t.ID, SessionID: x.ID, InstrumentID: t.InstrumentID,
			SortOrder: i32(t.SortOrder)}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ListPsikotesSessions(ctx context.Context, candidateID string) ([]domain.PsikotesSession, error) {
	rows, err := s.q.ListPsikotesSessions(ctx, candidateID)
	return each(rows, err, func(r sqlc.ListPsikotesSessionsRow) domain.PsikotesSession {
		return psikotesSessionFrom(r.RecruitmentPsikotesSession, r.CandidateName, r.PositionTitle)
	})
}

func (s *Store) PsikotesSessionByToken(ctx context.Context, token string) (*domain.PsikotesSession, error) {
	r, err := s.q.PsikotesSessionByToken(ctx, token)
	return one(r, err, func(r sqlc.PsikotesSessionByTokenRow) domain.PsikotesSession {
		return psikotesSessionFrom(r.RecruitmentPsikotesSession, r.CandidateName, r.PositionTitle)
	})
}

func (s *Store) SetPsikotesSessionStatus(ctx context.Context, id, status string) error {
	return s.q.SetPsikotesSessionStatus(ctx, sqlc.SetPsikotesSessionStatusParams{ID: id, Status: status})
}

func (s *Store) StartPsikotesSession(ctx context.Context, id string, consent bool) (bool, error) {
	return affected(s.q.StartPsikotesSession(ctx, sqlc.StartPsikotesSessionParams{ID: id, WebcamConsent: &consent}))
}

func (s *Store) CompletePsikotesSession(ctx context.Context, id string) (bool, error) {
	return affected(s.q.CompletePsikotesSession(ctx, id))
}

// testRow is the shared shape of ListSessionTestsRow and GetSessionTestRow.
type testRow = sqlc.GetSessionTestRow

func testFrom(r testRow) domain.SessionTest {
	t := r.RecruitmentPsikotesSessionTest
	out := domain.SessionTest{ID: t.ID, SessionID: t.SessionID, CandidateID: r.CandidateID, InstrumentID: t.InstrumentID,
		InstrumentCode: r.InstrumentCode, InstrumentName: r.InstrumentName, InstrumentKind: r.InstrumentKind,
		DurationSeconds: int(r.DurationSeconds), QuestionCount: intPtr(r.QuestionCount), Shuffle: r.Shuffle,
		Instructions: r.Instructions, Status: t.Status, QuestionIDs: stringsFrom(t.QuestionIds), Answers: map[string]string{},
		Score: t.Score, AttachmentKey: t.AttachmentKey, AttachmentContentType: str(t.AttachmentContentType),
		ReviewNotes: str(t.ReviewNotes), ReviewedByName: str(t.ReviewedByName), SortOrder: int(t.SortOrder),
		StartedAt: t.StartedAt, CompletedAt: t.CompletedAt}
	_ = json.Unmarshal(t.Answers, &out.Answers)
	if len(t.ScoreDetail) > 0 && string(t.ScoreDetail) != "null" {
		var d domain.ScoreDetail
		if json.Unmarshal(t.ScoreDetail, &d) == nil {
			out.ScoreDetail = &d
		}
	}
	return out
}

func (s *Store) ListSessionTests(ctx context.Context, sessionID, candidateID string) ([]domain.SessionTest, error) {
	rows, err := s.q.ListSessionTests(ctx, sqlc.ListSessionTestsParams{SessionID: nilIfEmpty(sessionID), CandidateID: nilIfEmpty(candidateID)})
	return each(rows, err, func(r sqlc.ListSessionTestsRow) domain.SessionTest { return testFrom(testRow(r)) })
}

func (s *Store) GetSessionTest(ctx context.Context, id, sessionID string) (*domain.SessionTest, error) {
	r, err := s.q.GetSessionTest(ctx, sqlc.GetSessionTestParams{ID: id, SessionID: nilIfEmpty(sessionID)})
	return one(r, err, testFrom)
}

func (s *Store) BeginTest(ctx context.Context, id string, questionIDs []string) (bool, error) {
	return affected(s.q.BeginTest(ctx, sqlc.BeginTestParams{ID: id, QuestionIds: mustJSON(nonNil(questionIDs))}))
}

func (s *Store) MergeAnswers(ctx context.Context, id string, answers map[string]string) (int, bool, error) {
	n, err := s.q.MergeAnswers(ctx, sqlc.MergeAnswersParams{ID: id, Answers: mustJSON(answers)})
	if pg.IsNoRows(err) {
		return 0, false, nil
	}
	return int(n), err == nil, err
}

func (s *Store) ScoreTest(ctx context.Context, id string, answers map[string]string, score *float64, detail domain.ScoreDetail) (bool, error) {
	return affected(s.q.ScoreTest(ctx, sqlc.ScoreTestParams{ID: id, Answers: mustJSON(answers), Score: score, ScoreDetail: mustJSON(detail)}))
}

func (s *Store) FinishDrawingTest(ctx context.Context, id string) (bool, error) {
	return affected(s.q.FinishDrawingTest(ctx, id))
}

func (s *Store) SetTestAttachment(ctx context.Context, id, key, contentType string) (bool, error) {
	return affected(s.q.SetTestAttachment(ctx, sqlc.SetTestAttachmentParams{ID: id, AttachmentKey: &key, AttachmentContentType: &contentType}))
}

func (s *Store) UnfinishedTestNames(ctx context.Context, sessionID string) ([]string, error) {
	return s.q.UnfinishedTestNames(ctx, sessionID)
}

func (s *Store) ReviewTest(ctx context.Context, id, notes string, by *string, byName string) (bool, error) {
	return affected(s.q.ReviewTest(ctx, sqlc.ReviewTestParams{ID: id, ReviewNotes: &notes, ReviewedBy: by, ReviewedByName: nilIfEmpty(byName)}))
}

func (s *Store) InsertProctorEvent(ctx context.Context, e domain.ProctorEvent) error {
	meta := e.Meta
	if meta == nil {
		meta = map[string]string{}
	}
	return s.q.InsertProctorEvent(ctx, sqlc.InsertProctorEventParams{ID: e.ID, SessionType: e.SessionType, SessionID: e.SessionID,
		EventType: e.EventType, Meta: mustJSON(meta), CreatedAt: e.CreatedAt})
}

func (s *Store) ListProctorEvents(ctx context.Context, sessionType, sessionID string) ([]domain.ProctorEvent, error) {
	rows, err := s.q.ListProctorEvents(ctx, sqlc.ListProctorEventsParams{SessionType: sessionType, SessionID: sessionID})
	return each(rows, err, func(r sqlc.RecruitmentProctorEvent) domain.ProctorEvent {
		e := domain.ProctorEvent{ID: r.ID, SessionType: r.SessionType, SessionID: r.SessionID, EventType: r.EventType,
			Meta: map[string]string{}, CreatedAt: r.CreatedAt}
		_ = json.Unmarshal(r.Meta, &e.Meta)
		return e
	})
}

func (s *Store) ProctorTallies(ctx context.Context, sessionType string, sessionIDs []string) (map[string]domain.ProctorTally, error) {
	out := map[string]domain.ProctorTally{}
	if len(sessionIDs) == 0 {
		return out, nil
	}
	rows, err := s.q.ProctorCounts(ctx, sqlc.ProctorCountsParams{SessionType: sessionType, SessionIds: sessionIDs})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		t := out[r.SessionID]
		t.Flags += int(r.N)
		out[r.SessionID] = t
	}
	return out, nil
}

func (s *Store) SessionExists(ctx context.Context, sessionType, id string) (bool, error) {
	return s.q.SessionExists(ctx, sqlc.SessionExistsParams{SessionType: sessionType, ID: id})
}

// ─── AI interview ────────────────────────────────────────────

func aiSessionFrom(r sqlc.RecruitmentInterviewAiSession, name, position string) domain.AiSession {
	out := domain.AiSession{ID: r.ID, CandidateID: r.CandidateID, CandidateName: name, PositionTitle: position, Token: r.Token,
		Status: r.Status, WebcamConsent: r.WebcamConsent, MaxQuestions: int(r.MaxQuestions), SummaryModel: str(r.SummaryModel),
		InvitedAt: r.InvitedAt, ExpiresAt: r.ExpiresAt, StartedAt: r.StartedAt, CompletedAt: r.CompletedAt,
		CreatedByName: str(r.CreatedByName)}
	if len(r.AiSummary) > 0 && string(r.AiSummary) != "null" {
		var sum domain.InterviewSummary
		if json.Unmarshal(r.AiSummary, &sum) == nil {
			out.Summary = &sum
		}
	}
	return out
}

func (s *Store) InsertAiSession(ctx context.Context, x domain.AiSession) error {
	return s.q.InsertAiSession(ctx, sqlc.InsertAiSessionParams{ID: x.ID, CandidateID: x.CandidateID, Token: x.Token, Status: x.Status,
		MaxQuestions: i32(x.MaxQuestions), InvitedAt: x.InvitedAt, ExpiresAt: x.ExpiresAt, CreatedBy: x.CreatedBy, CreatedByName: nilIfEmpty(x.CreatedByName)})
}

func (s *Store) ListAiSessions(ctx context.Context, candidateID string) ([]domain.AiSession, error) {
	rows, err := s.q.ListAiSessions(ctx, candidateID)
	return each(rows, err, func(r sqlc.ListAiSessionsRow) domain.AiSession {
		return aiSessionFrom(r.RecruitmentInterviewAiSession, r.CandidateName, r.PositionTitle)
	})
}

func (s *Store) AiSessionByToken(ctx context.Context, token string) (*domain.AiSession, error) {
	r, err := s.q.AiSessionByToken(ctx, token)
	return one(r, err, func(r sqlc.AiSessionByTokenRow) domain.AiSession {
		return aiSessionFrom(r.RecruitmentInterviewAiSession, r.CandidateName, r.PositionTitle)
	})
}

func (s *Store) SetAiSessionStatus(ctx context.Context, id, status string) error {
	return s.q.SetAiSessionStatus(ctx, sqlc.SetAiSessionStatusParams{ID: id, Status: status})
}

func (s *Store) StartAiSession(ctx context.Context, id string) (bool, error) {
	return affected(s.q.StartAiSession(ctx, id))
}

func (s *Store) CompleteAiSession(ctx context.Context, id string, summary *domain.InterviewSummary, model *string) error {
	var raw []byte
	if summary != nil {
		raw = mustJSON(summary)
	}
	return s.q.CompleteAiSession(ctx, sqlc.CompleteAiSessionParams{ID: id, AiSummary: raw, SummaryModel: model})
}

func turnFrom(r sqlc.RecruitmentInterviewAiTurn) domain.Turn {
	return domain.Turn{ID: r.ID, SessionID: r.SessionID, TurnNo: int(r.TurnNo), Topic: r.Topic, Question: r.Question,
		Answer: r.AnswerTranscript, AnswerMode: r.AnswerMode, AskedAt: r.AskedAt, AnsweredAt: r.AnsweredAt}
}

func (s *Store) ListTurns(ctx context.Context, sessionID, candidateID string) ([]domain.Turn, error) {
	rows, err := s.q.ListTurns(ctx, sqlc.ListTurnsParams{SessionID: nilIfEmpty(sessionID), CandidateID: nilIfEmpty(candidateID)})
	return each(rows, err, turnFrom)
}

func (s *Store) GetTurn(ctx context.Context, sessionID, id string) (*domain.Turn, error) {
	r, err := s.q.GetTurn(ctx, sqlc.GetTurnParams{ID: id, SessionID: sessionID})
	return one(r, err, turnFrom)
}

func (s *Store) InsertTurn(ctx context.Context, t domain.Turn) (bool, error) {
	return affected(s.q.InsertTurn(ctx, sqlc.InsertTurnParams{ID: t.ID, SessionID: t.SessionID, TurnNo: i32(t.TurnNo), Topic: t.Topic,
		Question: t.Question, AnswerTranscript: t.Answer, AnswerMode: t.AnswerMode, AskedAt: t.AskedAt, AnsweredAt: t.AnsweredAt}))
}

func (s *Store) AnswerTurn(ctx context.Context, sessionID, id, answer, mode string) (bool, error) {
	return affected(s.q.AnswerTurn(ctx, sqlc.AnswerTurnParams{ID: id, SessionID: sessionID, AnswerTranscript: &answer, AnswerMode: &mode}))
}

// ─── scheduled interviews ────────────────────────────────────

func interviewFrom(r sqlc.RecruitmentInterview, name, title string) domain.Interview {
	out := domain.Interview{ID: r.ID, CandidateID: r.CandidateID, CandidateName: name, JobOpeningTitle: title,
		InterviewerEmployeeID: r.InterviewerEmployeeID, InterviewerUserID: r.InterviewerUserID, InterviewerName: r.InterviewerName,
		InterviewDate: r.InterviewDate, DurationMinutes: int(r.DurationMinutes), Type: r.Type, Mode: r.Mode,
		MeetingLink: str(r.MeetingLink), Location: str(r.Location), Status: r.Status, Scorecard: []domain.ScoreItem{},
		OverallScore: r.OverallScore, Recommendation: str(r.Recommendation), Notes: str(r.Notes), CancelReason: str(r.CancelReason),
		CreatedBy: r.CreatedBy, UpdatedBy: r.UpdatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
	_ = json.Unmarshal(r.Scorecard, &out.Scorecard)
	return out
}

func (s *Store) ListInterviews(ctx context.Context, f domain.InterviewFilter) ([]domain.Interview, error) {
	rows, err := s.q.ListInterviews(ctx, sqlc.ListInterviewsParams{CandidateID: nilIfEmpty(f.CandidateID), Status: nilIfEmpty(f.Status),
		InterviewerEmployeeID: nilIfEmpty(f.InterviewerEmployeeID), FromAt: f.From, ToAt: f.To})
	return each(rows, err, func(r sqlc.ListInterviewsRow) domain.Interview {
		return interviewFrom(r.RecruitmentInterview, r.CandidateName, r.JobOpeningTitle)
	})
}

func (s *Store) GetInterview(ctx context.Context, id string) (*domain.Interview, error) {
	r, err := s.q.GetInterview(ctx, id)
	return one(r, err, func(r sqlc.GetInterviewRow) domain.Interview {
		return interviewFrom(r.RecruitmentInterview, r.CandidateName, r.JobOpeningTitle)
	})
}

func scorecardJSON(items []domain.ScoreItem) []byte {
	if items == nil {
		items = []domain.ScoreItem{}
	}
	return mustJSON(items)
}

func (s *Store) InsertInterview(ctx context.Context, i domain.Interview) error {
	return s.q.InsertInterview(ctx, sqlc.InsertInterviewParams{ID: i.ID, CandidateID: i.CandidateID,
		InterviewerEmployeeID: i.InterviewerEmployeeID, InterviewerUserID: i.InterviewerUserID, InterviewerName: i.InterviewerName,
		InterviewDate: i.InterviewDate, DurationMinutes: i32(i.DurationMinutes), Type: i.Type, Mode: i.Mode,
		MeetingLink: nilIfEmpty(i.MeetingLink), Location: nilIfEmpty(i.Location), Status: i.Status, Scorecard: scorecardJSON(i.Scorecard),
		OverallScore: i.OverallScore, Recommendation: nilIfEmpty(i.Recommendation), Notes: nilIfEmpty(i.Notes), CreatedBy: i.CreatedBy})
}

func (s *Store) UpdateInterview(ctx context.Context, i domain.Interview) (bool, error) {
	return affected(s.q.UpdateInterview(ctx, sqlc.UpdateInterviewParams{ID: i.ID, InterviewerEmployeeID: i.InterviewerEmployeeID,
		InterviewerUserID: i.InterviewerUserID, InterviewerName: i.InterviewerName, InterviewDate: i.InterviewDate,
		DurationMinutes: i32(i.DurationMinutes), Type: i.Type, Mode: i.Mode, MeetingLink: nilIfEmpty(i.MeetingLink),
		Location: nilIfEmpty(i.Location), Status: i.Status, Scorecard: scorecardJSON(i.Scorecard), OverallScore: i.OverallScore,
		Recommendation: nilIfEmpty(i.Recommendation), Notes: nilIfEmpty(i.Notes), CancelReason: nilIfEmpty(i.CancelReason),
		UpdatedBy: i.UpdatedBy}))
}

// ─── offers ──────────────────────────────────────────────────

func offerFrom(r sqlc.RecruitmentCandidateOffer, name, email string) domain.Offer {
	return domain.Offer{ID: r.ID, CandidateID: r.CandidateID, CandidateName: name, CandidateEmail: email, Version: int(r.Version),
		Token: r.Token, Status: r.Status, PositionTitle: str(r.PositionTitle), BaseSalary: r.BaseSalary, Benefits: stringsFrom(r.Benefits),
		StartDate: dayPtr(r.StartDate), Notes: str(r.Notes), ResponseNote: str(r.ResponseNote), RespondedAt: r.RespondedAt,
		ResponseIP: r.ResponseIp, ResponseSource: str(r.ResponseSource), SentAt: r.SentAt, ExpiresAt: r.ExpiresAt,
		CreatedBy: r.CreatedBy, CreatedByName: str(r.CreatedByName), CreatedAt: r.CreatedAt}
}

func (s *Store) ListOffers(ctx context.Context, candidateID string) ([]domain.Offer, error) {
	rows, err := s.q.ListOffers(ctx, candidateID)
	return each(rows, err, func(r sqlc.ListOffersRow) domain.Offer {
		return offerFrom(r.RecruitmentCandidateOffer, r.CandidateName, r.CandidateEmail)
	})
}

func (s *Store) GetOffer(ctx context.Context, id string) (*domain.Offer, error) {
	r, err := s.q.GetOffer(ctx, id)
	return one(r, err, func(r sqlc.GetOfferRow) domain.Offer {
		return offerFrom(r.RecruitmentCandidateOffer, r.CandidateName, r.CandidateEmail)
	})
}

func (s *Store) OfferByToken(ctx context.Context, token string) (*domain.Offer, error) {
	r, err := s.q.OfferByToken(ctx, token)
	return one(r, err, func(r sqlc.OfferByTokenRow) domain.Offer {
		return offerFrom(r.RecruitmentCandidateOffer, r.CandidateName, r.CandidateEmail)
	})
}

func (s *Store) ExpireOpenOffers(ctx context.Context, candidateID string) error {
	return s.q.ExpireOpenOffers(ctx, candidateID)
}

func (s *Store) NextOfferVersion(ctx context.Context, candidateID string) (int, error) {
	n, err := s.q.NextOfferVersion(ctx, candidateID)
	return int(n), err
}

func (s *Store) InsertOffer(ctx context.Context, o domain.Offer) error {
	benefits := o.Benefits
	if benefits == nil {
		benefits = []string{}
	}
	return wrap(s.q.InsertOffer(ctx, sqlc.InsertOfferParams{ID: o.ID, CandidateID: o.CandidateID, Version: i32(o.Version), Token: o.Token,
		Status: o.Status, PositionTitle: nilIfEmpty(o.PositionTitle), BaseSalary: o.BaseSalary, Benefits: mustJSON(benefits),
		StartDate: toDay(o.StartDate), Notes: nilIfEmpty(o.Notes), ResponseNote: nilIfEmpty(o.ResponseNote), RespondedAt: o.RespondedAt,
		ResponseSource: nilIfEmpty(o.ResponseSource), SentAt: o.SentAt, ExpiresAt: o.ExpiresAt, CreatedBy: o.CreatedBy,
		CreatedByName: nilIfEmpty(o.CreatedByName)}))
}

func (s *Store) RespondOffer(ctx context.Context, id, status string, note, ip *string) (bool, error) {
	return affected(s.q.RespondOffer(ctx, sqlc.RespondOfferParams{ID: id, Status: status, ResponseNote: note, ResponseIp: ip}))
}

func (s *Store) RecordManualResponse(ctx context.Context, id, status string, note *string) (bool, error) {
	return affected(s.q.RecordManualResponse(ctx, sqlc.RecordManualResponseParams{ID: id, Status: status, ResponseNote: note}))
}

func (s *Store) SetOfferStatus(ctx context.Context, id, status string) error {
	return s.q.SetOfferStatus(ctx, sqlc.SetOfferStatusParams{ID: id, Status: status})
}

func (s *Store) LatestAcceptedOffer(ctx context.Context, candidateID string) (*domain.Offer, error) {
	r, err := s.q.LatestAcceptedOffer(ctx, candidateID)
	return one(r, err, func(r sqlc.LatestAcceptedOfferRow) domain.Offer {
		return offerFrom(r.RecruitmentCandidateOffer, r.CandidateName, r.CandidateEmail)
	})
}

func (s *Store) ExpiringOffers(ctx context.Context, now, until time.Time) ([]domain.Offer, error) {
	rows, err := s.q.ExpiringOffers(ctx, sqlc.ExpiringOffersParams{Now: now, Until: until})
	return each(rows, err, func(r sqlc.ExpiringOffersRow) domain.Offer {
		return offerFrom(r.RecruitmentCandidateOffer, r.CandidateName, r.CandidateEmail)
	})
}

func (s *Store) ExpireLapsed(ctx context.Context, now time.Time) (int, error) {
	total := 0
	for _, fn := range []func(context.Context, time.Time) (int64, error){s.q.ExpireLapsedOffers, s.q.ExpireLapsedPsikotes, s.q.ExpireLapsedAiSessions} {
		n, err := fn(ctx, now)
		if err != nil {
			return total, err
		}
		total += int(n)
	}
	return total, nil
}

// ─── plumbing ────────────────────────────────────────────────

func (s *Store) HitRateLimit(ctx context.Context, key string, now time.Time, window time.Duration) (int, error) {
	n, err := s.q.HitRateLimit(ctx, sqlc.HitRateLimitParams{Key: key, Now: now, WindowSeconds: window.Seconds()})
	return int(n), err
}

func (s *Store) PruneRateLimits(ctx context.Context, before time.Time) error {
	return s.q.PruneRateLimits(ctx, before)
}

func (s *Store) ClaimReminder(ctx context.Context, kind, refID, day string) (bool, error) {
	d, _ := domain.ParseDay(day)
	return affected(s.q.ClaimReminder(ctx, sqlc.ClaimReminderParams{Kind: kind, RefID: refID, Day: d}))
}
