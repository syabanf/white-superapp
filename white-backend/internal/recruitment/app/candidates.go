package app

import (
	"context"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// Candidates is the candidate list, profile, pipeline and their panels.
type Candidates struct{ core }

func NewCandidates(d Deps) *Candidates { return &Candidates{core{d}} }

// File is a stored or rendered file returned inline.
type File struct {
	Name        string
	ContentType string
	Data        []byte
}

// ListQuery is candidateListQuerySchema.
type ListQuery struct {
	Status       string
	JobOpeningID string
	Search       string
	DateFrom     string
	DateTo       string
	Sort         string
	Page         int
	Limit        int
	All          bool
}

// Page is one page of candidates.
type Page struct {
	Candidates []domain.Candidate
	Total      int
	Page       int
	Limit      int
	TotalPages int
}

func (q ListQuery) filter() (domain.CandidateFilter, error) {
	v := validate.New()
	f := domain.CandidateFilter{JobOpeningID: q.JobOpeningID, Search: strings.TrimSpace(q.Search)}
	if q.Status != "" {
		if !domain.IsCandidateStatus(q.Status) {
			v.Add("status", "Status tidak valid")
		}
		f.Statuses = []string{q.Status}
	}
	if domain.UTF16Len(f.Search) > 100 {
		v.Add("search", "Pencarian maksimal 100 karakter")
	}
	for _, d := range []struct {
		field, value string
		dst          **time.Time
	}{{"date_from", q.DateFrom, &f.DateFrom}, {"date_to", q.DateTo, &f.DateTo}} {
		if d.value == "" {
			continue
		}
		t, ok := domain.ParseDay(d.value)
		if !ok {
			v.Add(d.field, "Tanggal tidak valid")
			continue
		}
		*d.dst = &t
	}
	switch q.Sort {
	case "", "created_at":
	case "updated_at":
		f.SortUpdated = true
	default:
		v.Add("sort", "Urutan tidak valid")
	}
	return f, v.Err()
}

func pageOf(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	switch {
	case limit < 1:
		limit = 20
	case limit > 100:
		limit = 100
	}
	return page, limit
}

func (c *Candidates) list(ctx context.Context, f domain.CandidateFilter, page, limit int, all bool) (*Page, error) {
	page, limit = pageOf(page, limit)
	if all {
		page, limit = 1, 10_000
	}
	f.Limit, f.Offset = limit, (page-1)*limit
	rows, total, err := c.Store.ListCandidates(ctx, f)
	if err != nil {
		return nil, err
	}
	if all {
		limit = max(total, 1)
	}
	return &Page{Candidates: rows, Total: total, Page: page, Limit: limit,
		TotalPages: int(math.Ceil(float64(total) / float64(limit)))}, nil
}

// List pages the candidate list (all=true returns every row).
func (c *Candidates) List(ctx context.Context, q ListQuery) (*Page, error) {
	if _, err := staff(ctx, false); err != nil {
		return nil, err
	}
	f, err := q.filter()
	if err != nil {
		return nil, err
	}
	return c.list(ctx, f, q.Page, q.Limit, q.All)
}

// TalentPool lists talent_pool candidates (and the archive when asked).
func (c *Candidates) TalentPool(ctx context.Context, search string, includeArchived bool, page, limit int) (*Page, error) {
	if _, err := staff(ctx, false); err != nil {
		return nil, err
	}
	statuses := []string{"talent_pool"}
	if includeArchived {
		statuses = domain.TalentPoolStatuses
	}
	return c.list(ctx, domain.CandidateFilter{Statuses: statuses, Search: strings.TrimSpace(search), SortUpdated: true}, page, limit, false)
}

// Column is one kanban column.
type Column struct {
	Status     string
	Label      string
	Count      int
	Candidates []domain.Candidate
}

// pipelineCards caps the cards per column; counts stay exact.
const pipelineCards = 100

// Pipeline is the kanban board, one column per status (the archive is left
// to the talent pool page).
func (c *Candidates) Pipeline(ctx context.Context, jobOpeningID, search string) ([]Column, error) {
	if _, err := staff(ctx, false); err != nil {
		return nil, err
	}
	counts, err := c.Store.CountByStatus(ctx, jobOpeningID)
	if err != nil {
		return nil, err
	}
	var out []Column
	for _, status := range domain.CandidateStatuses {
		if status == "archived" {
			continue
		}
		rows, total, err := c.Store.ListCandidates(ctx, domain.CandidateFilter{Statuses: []string{status}, JobOpeningID: jobOpeningID,
			Search: strings.TrimSpace(search), SortUpdated: true, Limit: pipelineCards})
		if err != nil {
			return nil, err
		}
		label, _ := domain.StatusLabel(status)
		n := counts[status]
		if search != "" {
			n = total
		}
		out = append(out, Column{Status: status, Label: label, Count: n, Candidates: rows})
	}
	return out, nil
}

// Summary is the recruitment dashboard.
func (c *Candidates) Summary(ctx context.Context) (*domain.Summary, error) {
	if _, err := staff(ctx, false); err != nil {
		return nil, err
	}
	s, err := c.Store.Summary(ctx, c.now())
	if err != nil {
		return nil, err
	}
	counts, err := c.Store.CountByStatus(ctx, "")
	if err != nil {
		return nil, err
	}
	for _, status := range domain.CandidateStatuses {
		label, _ := domain.StatusLabel(status)
		s.ByStatus = append(s.ByStatus, domain.Count{Key: status, Label: label, Count: counts[status]})
	}
	if s.BySource, err = c.Store.CountBySource(ctx); err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *Candidates) Get(ctx context.Context, id string) (*domain.Candidate, error) {
	if _, err := staff(ctx, false); err != nil {
		return nil, err
	}
	return c.requireCandidate(ctx, id)
}

func (c *Candidates) checkOpening(ctx context.Context, id *string) error {
	if id == nil {
		return nil
	}
	o, err := c.Store.GetJobOpening(ctx, *id)
	if err == nil && o == nil {
		err = apperr.Field("job_opening_id", MsgJobNotFound)
	}
	return err
}

// Create inserts a manual candidate.
func (c *Candidates) Create(ctx context.Context, in domain.CandidateFields) (*domain.Candidate, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return nil, err
	}
	f, err := domain.NormalizeCandidate(in, true)
	if err != nil {
		return nil, err
	}
	if err := c.checkOpening(ctx, f.JobOpeningID); err != nil {
		return nil, err
	}
	cand := candidateFromFields(newID(), f)
	cand.CreatedBy = a.id()
	if err := c.Store.InsertCandidate(ctx, cand); err != nil {
		return nil, err
	}
	return c.Store.GetCandidate(ctx, cand.ID)
}

func candidateFromFields(id string, f domain.CandidateFields) domain.Candidate {
	return domain.Candidate{ID: id, FullName: f.FullName, Email: f.Email, Phone: f.Phone, Domicile: f.Domicile, Source: f.Source,
		JobOpeningID: f.JobOpeningID, Status: f.Status, Notes: f.Notes, LastExperience: f.LastExperience,
		LastEducation: f.LastEducation, Availability: f.Availability, ExpectedSalary: f.ExpectedSalary}
}

// Update patches the profile fields that were sent (status moves go
// through MoveStage).
func (c *Candidates) Update(ctx context.Context, id string, p domain.CandidatePatch) (*domain.Candidate, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return nil, err
	}
	cand, err := c.requireCandidate(ctx, id)
	if err != nil {
		return nil, err
	}
	f, err := domain.NormalizeCandidate(p.Apply(cand.Fields()), false)
	if err != nil {
		return nil, err
	}
	if err := c.checkOpening(ctx, f.JobOpeningID); err != nil {
		return nil, err
	}
	if _, err := c.Store.UpdateCandidateProfile(ctx, id, f, a.UserID); err != nil {
		return nil, err
	}
	return c.Store.GetCandidate(ctx, id)
}

// Delete removes the candidate and every panel (CASCADE), then their CV
// and drawings on disk.
func (c *Candidates) Delete(ctx context.Context, id string) error {
	a, err := staff(ctx, true)
	if err != nil {
		return err
	}
	cand, err := c.requireCandidate(ctx, id)
	if err != nil {
		return err
	}
	tests, err := c.Store.ListSessionTests(ctx, "", id)
	if err != nil {
		return err
	}
	if _, err := c.Store.DeleteCandidate(ctx, id); err != nil {
		return err
	}
	keys := []*string{cand.CVKey}
	for _, t := range tests {
		keys = append(keys, t.AttachmentKey)
	}
	for _, k := range keys {
		if k != nil {
			if err := c.Files.Delete(ctx, *k); err != nil {
				c.log().Warn("candidate file not removed", "key", *k, "err", err)
			}
		}
	}
	c.log().Info("candidate deleted", "candidate", id, "by", a.UserID)
	return nil
}

// MoveStage moves a candidate to status with an audited activity; the same
// status is a no-op. Reaching hired publishes the hire event.
func (c *Candidates) MoveStage(ctx context.Context, id, status string) (string, *string, string, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return "", nil, "", err
	}
	if !domain.IsCandidateStatus(status) {
		return "", nil, "", apperr.Field("status", "Status tidak valid")
	}
	cand, err := c.requireCandidate(ctx, id)
	if err != nil {
		return "", nil, "", err
	}
	if cand.Status == status {
		return status, nil, "Status tidak berubah", nil
	}
	err = c.Store.WithTx(ctx, func(tx domain.Store) error {
		if _, err := tx.SetCandidateStatus(ctx, id, status, a.id()); err != nil {
			return err
		}
		return c.staffActivity(ctx, tx, id, "status_change", domain.StageChangeDescription(cand.Status, status), a)
	})
	if err != nil {
		return "", nil, "", err
	}
	c.afterStatusChange(ctx, id, status)
	label, _ := domain.StatusLabel(status)
	prev := cand.Status
	return status, &prev, "Kandidat dipindahkan ke " + label, nil
}

func (c *Candidates) Notes(ctx context.Context, id string) ([]domain.Note, error) {
	if _, err := staff(ctx, false); err != nil {
		return nil, err
	}
	if _, err := c.requireCandidate(ctx, id); err != nil {
		return nil, err
	}
	return c.Store.ListNotes(ctx, id)
}

// AddNote stores an internal note and its activity atomically.
func (c *Candidates) AddNote(ctx context.Context, id, content string) (*domain.Note, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return nil, err
	}
	content, err = domain.ValidateNote(content)
	if err != nil {
		return nil, err
	}
	if _, err := c.requireCandidate(ctx, id); err != nil {
		return nil, err
	}
	note := domain.Note{ID: newID(), CandidateID: id, Content: content, CreatedBy: a.id(), CreatedByName: a.Name, CreatedAt: c.now()}
	err = c.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.InsertNote(ctx, note); err != nil {
			return err
		}
		return c.staffActivity(ctx, tx, id, "note_added", "Catatan internal ditambahkan", a)
	})
	return &note, err
}

func (c *Candidates) Activities(ctx context.Context, id string) ([]domain.Activity, error) {
	if _, err := staff(ctx, false); err != nil {
		return nil, err
	}
	if _, err := c.requireCandidate(ctx, id); err != nil {
		return nil, err
	}
	return c.Store.ListActivities(ctx, id)
}

func (c *Candidates) Screening(ctx context.Context, id string) (*domain.Screening, error) {
	if _, err := staff(ctx, false); err != nil {
		return nil, err
	}
	if _, err := c.requireCandidate(ctx, id); err != nil {
		return nil, err
	}
	return c.Store.GetScreening(ctx, id)
}

// SaveScreening upserts the screening result and its activity atomically.
func (c *Candidates) SaveScreening(ctx context.Context, s domain.Screening) (*domain.Screening, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return nil, err
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if _, err := c.requireCandidate(ctx, s.CandidateID); err != nil {
		return nil, err
	}
	s.UpdatedBy, s.UpdatedByName = a.id(), a.Name
	err = c.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.UpsertScreening(ctx, s); err != nil {
			return err
		}
		rec := s.Recommendation
		return c.staffActivity(ctx, tx, s.CandidateID, "screening_updated", "Hasil screening disimpan"+domain.RecommendationSuffix(&rec), a)
	})
	if err != nil {
		return nil, err
	}
	return c.Store.GetScreening(ctx, s.CandidateID)
}

// UploadCV stores a CV (pdf, doc, docx, jpg, png; 10 MB) and replaces the
// previous file.
func (c *Candidates) UploadCV(ctx context.Context, id, name string, data []byte) (*domain.Candidate, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return nil, err
	}
	cand, err := c.requireCandidate(ctx, id)
	if err != nil {
		return nil, err
	}
	mime, err := domain.CheckCV(name, data)
	if err != nil {
		return nil, err
	}
	key, err := c.storeCV(ctx, id, name, data)
	if err != nil {
		return nil, err
	}
	fileName := domain.SafeFileName(name, "cv."+domain.FileExt(name))
	if _, err := c.Store.SetCandidateCV(ctx, id, &key, &fileName, &mime, a.UserID); err != nil {
		return nil, err
	}
	c.removeFile(ctx, cand.CVKey)
	return c.Store.GetCandidate(ctx, id)
}

func (c core) storeCV(ctx context.Context, candidateID, name string, data []byte) (string, error) {
	key := "cv/" + candidateID + "/" + newID() + "." + domain.FileExt(name)
	if err := c.Files.Put(ctx, key, data); err != nil {
		c.log().Error("cv upload failed", "err", err)
		return "", apperr.Wrap(apperr.Unknown, "Upload gagal", err)
	}
	return key, nil
}

func (c core) removeFile(ctx context.Context, key *string) {
	if key == nil {
		return
	}
	if err := c.Files.Delete(ctx, *key); err != nil {
		c.log().Warn("file not removed", "key", *key, "err", err)
	}
}

func (c *Candidates) DeleteCV(ctx context.Context, id string) (*domain.Candidate, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return nil, err
	}
	cand, err := c.requireCandidate(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err := c.Store.SetCandidateCV(ctx, id, nil, nil, nil, a.UserID); err != nil {
		return nil, err
	}
	c.removeFile(ctx, cand.CVKey)
	return c.Store.GetCandidate(ctx, id)
}

func (c *Candidates) DownloadCV(ctx context.Context, id string) (*File, error) {
	if _, err := staff(ctx, false); err != nil {
		return nil, err
	}
	cand, err := c.requireCandidate(ctx, id)
	if err != nil {
		return nil, err
	}
	if cand.CVKey == nil {
		return nil, notFound("Kandidat belum memiliki lampiran CV")
	}
	data, err := c.Files.Get(ctx, *cand.CVKey)
	if err != nil {
		return nil, apperr.Wrap(apperr.NotFound, "File CV tidak ditemukan di storage", err)
	}
	return &File{Name: cand.CVFileName, ContentType: cand.CVContentType, Data: data}, nil
}

func (c *Candidates) Analysis(ctx context.Context, id string) (*domain.CandidateAnalysis, error) {
	if _, err := staff(ctx, false); err != nil {
		return nil, err
	}
	if _, err := c.requireCandidate(ctx, id); err != nil {
		return nil, err
	}
	return c.Store.GetAnalysis(ctx, id)
}

// Analyze scores the candidate profile against their opening and stores
// the result (one analysis per candidate).
func (c *Candidates) Analyze(ctx context.Context, id string) (*domain.CandidateAnalysis, error) {
	if _, err := staff(ctx, true); err != nil {
		return nil, err
	}
	cand, err := c.requireCandidate(ctx, id)
	if err != nil {
		return nil, err
	}
	var opening *domain.JobOpening
	if cand.JobOpeningID != nil {
		if opening, err = c.Store.GetJobOpening(ctx, *cand.JobOpeningID); err != nil {
			return nil, err
		}
	}
	jobContext := domain.JobContext(opening)
	result, model, err := c.AI.AnalyzeCandidate(ctx, domain.AnalysisInput{CandidateName: cand.FullName,
		Profile: domain.CandidateProfile(*cand), JobContext: jobContext})
	if err != nil {
		return nil, apperr.Wrap(apperr.Unknown, "Analisis kandidat gagal", err)
	}
	result.CandidateID, result.JobContext, result.Model = id, jobContext, model
	if err := c.Store.UpsertAnalysis(ctx, result); err != nil {
		return nil, err
	}
	return c.Store.GetAnalysis(ctx, id)
}

// PipelineReport renders the candidate's pipeline report PDF, available
// from the Offer stage.
func (c *Candidates) PipelineReport(ctx context.Context, id string) (*File, error) {
	if _, err := staff(ctx, false); err != nil {
		return nil, err
	}
	cand, err := c.requireCandidate(ctx, id)
	if err != nil {
		return nil, err
	}
	if !slices.Contains([]string{"offer", "hired"}, cand.Status) {
		return nil, conflict("Laporan pipeline tersedia mulai tahap Offer")
	}
	r := domain.PipelineReport{Candidate: *cand, CompanyName: c.company()}
	if r.Analysis, err = c.Store.GetAnalysis(ctx, id); err != nil {
		return nil, err
	}
	if r.Screening, err = c.Store.GetScreening(ctx, id); err != nil {
		return nil, err
	}
	if r.PsikotesTests, err = c.Store.ListSessionTests(ctx, "", id); err != nil {
		return nil, err
	}
	if r.PsikotesSummary, err = c.Store.GetPsikotesSummary(ctx, id); err != nil {
		return nil, err
	}
	if r.AiInterviews, err = aiSessionsWithTurns(ctx, c.Store, id); err != nil {
		return nil, err
	}
	if r.Interviews, err = c.Store.ListInterviews(ctx, domain.InterviewFilter{CandidateID: id}); err != nil {
		return nil, err
	}
	if r.Offers, err = c.Store.ListOffers(ctx, id); err != nil {
		return nil, err
	}
	if r.Activities, err = c.Store.ListActivities(ctx, id); err != nil {
		return nil, err
	}
	pdf, err := c.PDF.PipelineReport(r, c.now())
	if err != nil {
		return nil, err
	}
	return &File{Name: "laporan-pipeline-" + fileSlug(cand.FullName, "kandidat") + ".pdf", ContentType: domain.MimePDF, Data: pdf}, nil
}

// fileSlug is the slug part of a download name (60 characters).
func fileSlug(name, fallback string) string {
	s := domain.JSSlice(domain.Slugify(name), 60)
	if s == "" {
		return fallback
	}
	return s
}

// aiSessionsWithTurns lists a candidate's AI interviews with their turns.
func aiSessionsWithTurns(ctx context.Context, store domain.Store, candidateID string) ([]domain.AiSession, error) {
	sessions, err := store.ListAiSessions(ctx, candidateID)
	if err != nil {
		return nil, err
	}
	turns, err := store.ListTurns(ctx, "", candidateID)
	if err != nil {
		return nil, err
	}
	bySession := map[string][]domain.Turn{}
	for _, t := range turns {
		bySession[t.SessionID] = append(bySession[t.SessionID], t)
	}
	for i := range sessions {
		sessions[i].Turns = bySession[sessions[i].ID]
	}
	return sessions, nil
}
