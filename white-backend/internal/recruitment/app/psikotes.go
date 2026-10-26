package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// Psikotes is the HR side of psikotes: the instrument and question bank,
// invitations, answer breakdowns and manual review. Answer keys leave the
// server only through these staff RPCs.
type Psikotes struct{ core }

func NewPsikotes(d Deps) *Psikotes { return &Psikotes{core{d}} }

const (
	msgInstrumentNotFound = "Instrumen tidak ditemukan"
	msgQuestionNotFound   = "Soal tidak ditemukan"
	msgTestNotFound       = "Tes tidak ditemukan"
)

func (p *Psikotes) Instruments(ctx context.Context) ([]domain.Instrument, error) {
	if _, err := staff(ctx, false); err != nil {
		return nil, err
	}
	return p.Store.ListInstruments(ctx)
}

// UpdateInstrument patches an instrument; 404 when missing.
func (p *Psikotes) UpdateInstrument(ctx context.Context, id string, patch domain.InstrumentPatch) (*domain.Instrument, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return nil, err
	}
	if err := patch.Validate(); err != nil {
		return nil, err
	}
	ok, err := p.Store.UpdateInstrument(ctx, id, patch, a.UserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, notFound(msgInstrumentNotFound)
	}
	return p.Store.GetInstrument(ctx, id)
}

func (p *Psikotes) instrument(ctx context.Context, id string) (*domain.Instrument, error) {
	inst, err := p.Store.GetInstrument(ctx, id)
	if err == nil && inst == nil {
		err = notFound(msgInstrumentNotFound)
	}
	return inst, err
}

// Questions lists an instrument's bank with its keys.
func (p *Psikotes) Questions(ctx context.Context, instrumentID string) ([]domain.Question, error) {
	if _, err := staff(ctx, false); err != nil {
		return nil, err
	}
	if _, err := p.instrument(ctx, instrumentID); err != nil {
		return nil, err
	}
	return p.Store.ListQuestions(ctx, instrumentID)
}

// buildQuestion validates the input with the schema of the instrument kind
// (mcq, else forced choice).
func buildQuestion(kind string, in domain.QuestionInput) (domain.Question, error) {
	q := domain.Question{SortOrder: in.SortOrder, IsActive: in.IsActive}
	if kind == domain.KindMcq {
		if err := domain.ValidateMcqQuestion(&in); err != nil {
			return q, err
		}
		q.Body, q.Options, q.CorrectKey = in.Body, in.Options, in.CorrectKey
		return q, nil
	}
	if err := domain.ValidatePapiQuestion(&in); err != nil {
		return q, err
	}
	q.Body, q.Pair = in.Body, &domain.PapiPair{A: in.StatementA, B: in.StatementB}
	return q, nil
}

// CreateQuestion adds a question; drawing instruments have no bank.
func (p *Psikotes) CreateQuestion(ctx context.Context, instrumentID string, in domain.QuestionInput) (*domain.Question, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return nil, err
	}
	inst, err := p.instrument(ctx, instrumentID)
	if err != nil {
		return nil, err
	}
	if inst.Kind == domain.KindDrawing {
		return nil, invalid("Instrumen tes gambar tidak memiliki bank soal — atur instruksi di config")
	}
	q, err := buildQuestion(inst.Kind, in)
	if err != nil {
		return nil, err
	}
	q.ID, q.InstrumentID = newID(), instrumentID
	if err := p.Store.InsertQuestion(ctx, q, a.UserID); err != nil {
		return nil, err
	}
	return p.Store.GetQuestion(ctx, q.ID)
}

// UpdateQuestion replaces a question, validated by its instrument kind.
func (p *Psikotes) UpdateQuestion(ctx context.Context, id string, in domain.QuestionInput) (*domain.Question, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return nil, err
	}
	cur, err := p.Store.GetQuestion(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, notFound(msgQuestionNotFound)
	}
	inst, err := p.instrument(ctx, cur.InstrumentID)
	if err != nil {
		return nil, err
	}
	q, err := buildQuestion(inst.Kind, in)
	if err != nil {
		return nil, err
	}
	q.ID, q.InstrumentID = id, cur.InstrumentID
	if _, err := p.Store.UpdateQuestion(ctx, q, a.UserID); err != nil {
		return nil, err
	}
	return p.Store.GetQuestion(ctx, id)
}

// DeleteQuestion hard-deletes a question.
func (p *Psikotes) DeleteQuestion(ctx context.Context, id string) error {
	if _, err := staff(ctx, true); err != nil {
		return err
	}
	ok, err := p.Store.DeleteQuestion(ctx, id)
	if err == nil && !ok {
		err = notFound(msgQuestionNotFound)
	}
	return err
}

// Panel is the HR psikotes panel of a candidate: the summary and every
// session with its tests and proctor tally. Tokens go to writers only.
func (p *Psikotes) Panel(ctx context.Context, candidateID string) (*domain.PsikotesSummary, []domain.PsikotesSession, error) {
	a, err := staff(ctx, false)
	if err != nil {
		return nil, nil, err
	}
	if _, err := p.requireCandidate(ctx, candidateID); err != nil {
		return nil, nil, err
	}
	summary, err := p.Store.GetPsikotesSummary(ctx, candidateID)
	if err != nil {
		return nil, nil, err
	}
	sessions, err := p.Store.ListPsikotesSessions(ctx, candidateID)
	if err != nil {
		return nil, nil, err
	}
	tests, err := p.Store.ListSessionTests(ctx, "", candidateID)
	if err != nil {
		return nil, nil, err
	}
	ids := make([]string, len(sessions))
	for i, s := range sessions {
		ids[i] = s.ID
	}
	tallies, err := p.Store.ProctorTallies(ctx, "psikotes", ids)
	if err != nil {
		return nil, nil, err
	}
	bySession := map[string][]domain.SessionTest{}
	for _, t := range tests {
		bySession[t.SessionID] = append(bySession[t.SessionID], t)
	}
	for i := range sessions {
		s := &sessions[i]
		s.Tests, s.Proctor = bySession[s.ID], tallies[s.ID]
		if !a.CanWrite {
			s.Token = ""
		}
	}
	return summary, sessions, nil
}

// Invite creates a psikotes session for the chosen active instruments, its
// tests and the activity in one transaction.
func (p *Psikotes) Invite(ctx context.Context, candidateID string, instrumentIDs []string, expiresDays int) (*domain.PsikotesSession, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return nil, err
	}
	if err := domain.ValidateInvite(instrumentIDs, &expiresDays); err != nil {
		return nil, err
	}
	if _, err := p.requireCandidate(ctx, candidateID); err != nil {
		return nil, err
	}
	instruments, err := p.Store.ActiveInstruments(ctx, instrumentIDs)
	if err != nil {
		return nil, err
	}
	if len(instruments) != len(instrumentIDs) {
		return nil, invalid("Ada instrumen yang tidak ditemukan atau nonaktif")
	}
	now := p.now()
	s := domain.PsikotesSession{ID: newID(), CandidateID: candidateID, Token: newPortalToken(), Status: "sent", InvitedAt: &now,
		ExpiresAt: timePtr(now.Add(time.Duration(expiresDays) * 24 * time.Hour)), CreatedBy: a.id(), CreatedByName: a.Name}
	names := make([]string, len(instruments))
	for i, inst := range instruments {
		names[i] = inst.Name
		s.Tests = append(s.Tests, domain.SessionTest{ID: newID(), InstrumentID: inst.ID, SortOrder: i})
	}
	err = p.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.InsertPsikotesSession(ctx, s); err != nil {
			return err
		}
		return p.staffActivity(ctx, tx, candidateID, "psikotes_invited",
			fmt.Sprintf("Undangan psikotes online dibuat (%s; berlaku %d hari)", strings.Join(names, ", "), expiresDays), a)
	})
	if err != nil {
		return nil, err
	}
	if s.Tests, err = p.Store.ListSessionTests(ctx, s.ID, ""); err != nil {
		return nil, err
	}
	return &s, nil
}

// SaveSummary upserts the psikotes recommendation and its activity.
func (p *Psikotes) SaveSummary(ctx context.Context, s domain.PsikotesSummary) (*domain.PsikotesSummary, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return nil, err
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if _, err := p.requireCandidate(ctx, s.CandidateID); err != nil {
		return nil, err
	}
	s.UpdatedBy, s.UpdatedByName = a.id(), a.Name
	err = p.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.UpsertPsikotesSummary(ctx, s); err != nil {
			return err
		}
		rec := s.Recommendation
		return p.staffActivity(ctx, tx, s.CandidateID, "psikotes_summary_updated",
			"Rekomendasi psikotes disimpan"+domain.RecommendationSuffix(&rec), a)
	})
	if err != nil {
		return nil, err
	}
	return p.Store.GetPsikotesSummary(ctx, s.CandidateID)
}

func (p *Psikotes) test(ctx context.Context, id string) (*domain.SessionTest, error) {
	t, err := p.Store.GetSessionTest(ctx, id, "")
	if err == nil && t == nil {
		err = notFound(msgTestNotFound)
	}
	return t, err
}

// AnswerItem is one line of an answer breakdown.
type AnswerItem struct {
	Question  domain.Question
	Found     bool
	Given     *string
	IsCorrect *bool
}

// Answers is the HR breakdown of a test: MCQ in the order of the score
// snapshot (deleted questions have an empty body), PAPI over the active
// bank.
func (p *Psikotes) Answers(ctx context.Context, testID string) (string, []AnswerItem, error) {
	if _, err := staff(ctx, false); err != nil {
		return "", nil, err
	}
	t, err := p.test(ctx, testID)
	if err != nil {
		return "", nil, err
	}
	switch t.InstrumentKind {
	case domain.KindDrawing:
		return "", nil, invalid("Tes gambar tidak memiliki rincian soal")
	case domain.KindMcq:
		items := []AnswerItem{}
		if t.ScoreDetail == nil || len(t.ScoreDetail.PerQuestion) == 0 {
			return domain.KindMcq, items, nil
		}
		ids := make([]string, len(t.ScoreDetail.PerQuestion))
		for i, pq := range t.ScoreDetail.PerQuestion {
			ids[i] = pq.ID
		}
		qs, err := p.Store.QuestionsByIDs(ctx, ids)
		if err != nil {
			return "", nil, err
		}
		byID := map[string]domain.Question{}
		for _, q := range qs {
			byID[q.ID] = q
		}
		for _, pq := range t.ScoreDetail.PerQuestion {
			q, found := byID[pq.ID]
			q.ID, q.CorrectKey = pq.ID, pq.CorrectKey
			correct := pq.IsCorrect
			items = append(items, AnswerItem{Question: q, Found: found, Given: pq.Given, IsCorrect: &correct})
		}
		return domain.KindMcq, items, nil
	}
	bank, err := p.Store.ListQuestions(ctx, t.InstrumentID)
	if err != nil {
		return "", nil, err
	}
	items := []AnswerItem{}
	for _, q := range bank {
		if !q.IsActive {
			continue
		}
		var given *string
		if g, ok := t.Answers[q.ID]; ok && (g == "a" || g == "b") {
			given = &g
		}
		items = append(items, AnswerItem{Question: q, Found: true, Given: given})
	}
	return domain.KindForcedChoice, items, nil
}

// Review stores HR's review of a projective test with its activity.
func (p *Psikotes) Review(ctx context.Context, testID, notes string) (*domain.SessionTest, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return nil, err
	}
	notes, err = domain.ValidateReview(notes)
	if err != nil {
		return nil, err
	}
	t, err := p.test(ctx, testID)
	if err != nil {
		return nil, err
	}
	if !domain.IsReviewable(t.Status) {
		return nil, conflict("Tes ini tidak dalam antrian review manual")
	}
	err = p.Store.WithTx(ctx, func(tx domain.Store) error {
		if _, err := tx.ReviewTest(ctx, testID, notes, a.id(), a.Name); err != nil {
			return err
		}
		return p.staffActivity(ctx, tx, t.CandidateID, "psikotes_reviewed", "Hasil tes "+t.InstrumentName+" direview manual", a)
	})
	if err != nil {
		return nil, err
	}
	return p.test(ctx, testID)
}

// Attachment is the uploaded drawing of a test.
func (p *Psikotes) Attachment(ctx context.Context, testID string) (*File, error) {
	if _, err := staff(ctx, false); err != nil {
		return nil, err
	}
	t, err := p.test(ctx, testID)
	if err != nil {
		return nil, err
	}
	if t.AttachmentKey == nil {
		return nil, notFound("Tes ini tidak punya gambar terunggah")
	}
	data, err := p.Files.Get(ctx, *t.AttachmentKey)
	if err != nil {
		return nil, apperr.Wrap(apperr.NotFound, "Berkas gambar tidak ditemukan di storage", err)
	}
	name := fileSlug(t.InstrumentCode, "gambar") + "-" + t.ID + "." + domain.FileExt(*t.AttachmentKey)
	return &File{Name: name, ContentType: t.AttachmentContentType, Data: data}, nil
}

// ProctorEvents lists a session's proctoring flags; 404 when missing.
func (p *Psikotes) ProctorEvents(ctx context.Context, sessionType, sessionID string) ([]domain.ProctorEvent, error) {
	if _, err := staff(ctx, false); err != nil {
		return nil, err
	}
	if sessionType != "psikotes" && sessionType != "interview" {
		return nil, notFound(MsgSessionNotFound)
	}
	ok, err := p.Store.SessionExists(ctx, sessionType, sessionID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, notFound(MsgSessionNotFound)
	}
	return p.Store.ListProctorEvents(ctx, sessionType, sessionID)
}
