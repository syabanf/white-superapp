package app

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// Portal is the anonymous candidate side: the career page and apply form,
// the psikotes and AI interview runners and the offer response. The public
// job slug or the link token is the only identity; rate limits are keyed to
// the network (apply) or the session.
type Portal struct{ core }

func NewPortal(d Deps) *Portal { return &Portal{core{d}} }

// ─── career page ─────────────────────────────────────────────

// PublicJobs lists published openings that still take applications.
func (p *Portal) PublicJobs(ctx context.Context, search, department string) ([]domain.JobOpening, error) {
	return p.Store.ListPublicJobs(ctx, domain.DayKey(p.now()), strings.TrimSpace(search), strings.TrimSpace(department))
}

func (p *Portal) openJob(ctx context.Context, slug string) (*domain.JobOpening, error) {
	o, err := p.Store.GetJobOpeningBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if o == nil || o.Status == "draft" {
		return nil, notFound(MsgJobNotFound)
	}
	if !o.IsOpenForApplications(p.now()) {
		return nil, conflict(domain.MsgJobClosed)
	}
	return o, nil
}

// PublicJob is one opening of the career page.
func (p *Portal) PublicJob(ctx context.Context, slug string) (*domain.JobOpening, error) {
	return p.openJob(ctx, slug)
}

// Apply is the career form: an applied candidate with an optional CV,
// limited to 5 applications per network every 10 minutes.
func (p *Portal) Apply(ctx context.Context, slug string, a domain.Application, cvName string, cv []byte, ip string) (string, error) {
	if err := p.enforce(ctx, "portal-submit:"+ip, domain.ApplyRateLimit, domain.ApplyRateWindow, domain.MsgApplyTooMany); err != nil {
		return "", err
	}
	job, err := p.openJob(ctx, slug)
	if err != nil {
		return "", err
	}
	f, err := domain.NormalizeApplication(a)
	if err != nil {
		return "", err
	}
	var mime string
	if len(cv) > 0 {
		if mime, err = domain.CheckPortalCV(cvName, cv); err != nil {
			return "", err
		}
	}
	applied, err := p.Store.CandidateAppliedTo(ctx, f.Email, job.ID)
	if err != nil {
		return "", err
	}
	if applied {
		return "", conflict(domain.MsgAlreadyApplied)
	}
	f.JobOpeningID = &job.ID
	cand := candidateFromFields(newID(), f)
	if len(cv) > 0 {
		key, err := p.storeCV(ctx, cand.ID, cvName, cv)
		if err != nil {
			return "", err
		}
		cand.CVKey, cand.CVFileName, cand.CVContentType = &key, domain.SafeFileName(cvName, "cv."+domain.FileExt(cvName)), mime
	}
	err = p.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.InsertCandidate(ctx, cand); err != nil {
			return err
		}
		return p.activity(ctx, tx, cand.ID, "applied", "Lamaran masuk melalui portal karir untuk "+job.Title, nil, "")
	})
	if err != nil {
		p.removeFile(ctx, cand.CVKey)
		return "", err
	}
	p.notify(ctx, Notice{UserIDs: p.recipients(ctx, job.HiringManagerUserID), Type: NotifApplication,
		Title: "Lamaran baru: " + cand.FullName, Body: cand.FullName + " melamar posisi " + job.Title + ".", Href: candidateHref(cand.ID)})
	return cand.ID, nil
}

// ─── link tokens ─────────────────────────────────────────────

func bucketLimit(bucket string) int {
	switch bucket {
	case "upload":
		return domain.UploadRateLimit
	case "answer":
		return domain.AnswerRateLimit
	case "proctor":
		return domain.ProctorRateLimit
	}
	return domain.DefaultRateLimit
}

// openLink expires a lapsed link, then applies the bucket's rate limit.
func (p *Portal) openLink(ctx context.Context, kind domain.PortalKind, bucket, id string, status *string, expires, issued *time.Time,
	expire func(ctx context.Context, id, status string) error) error {
	if kind.ShouldExpire(*status, expires, issued, p.now()) {
		if err := expire(ctx, id, "expired"); err != nil {
			return err
		}
		*status = "expired"
	}
	return p.enforce(ctx, kind.Name+"_session_"+bucket+"_"+id, bucketLimit(bucket), domain.RateWindow, domain.TooManyRequests)
}

// missing is the store error, or 404 with the portal's message.
func missing(err error, kind domain.PortalKind) error {
	if err != nil {
		return err
	}
	return notFound(kind.NotFound)
}

func (p *Portal) psikotes(ctx context.Context, token, bucket string) (*domain.PsikotesSession, error) {
	kind := domain.PsikotesPortal
	if !domain.IsPortalToken(token) {
		return nil, notFound(kind.NotFound)
	}
	s, err := p.Store.PsikotesSessionByToken(ctx, token)
	if err != nil || s == nil {
		return nil, missing(err, kind)
	}
	return s, p.openLink(ctx, kind, bucket, s.ID, &s.Status, s.ExpiresAt, s.InvitedAt, p.Store.SetPsikotesSessionStatus)
}

func (p *Portal) aiSession(ctx context.Context, token, bucket string) (*domain.AiSession, error) {
	kind := domain.InterviewPortal
	if !domain.IsPortalToken(token) {
		return nil, notFound(kind.NotFound)
	}
	s, err := p.Store.AiSessionByToken(ctx, token)
	if err != nil || s == nil {
		return nil, missing(err, kind)
	}
	return s, p.openLink(ctx, kind, bucket, s.ID, &s.Status, s.ExpiresAt, s.InvitedAt, p.Store.SetAiSessionStatus)
}

func (p *Portal) offer(ctx context.Context, token string) (*domain.Offer, error) {
	kind := domain.OfferPortal
	if !domain.IsPortalToken(token) {
		return nil, notFound(kind.NotFound)
	}
	o, err := p.Store.OfferByToken(ctx, token)
	if err != nil || o == nil {
		return nil, missing(err, kind)
	}
	return o, p.openLink(ctx, kind, "get", o.ID, &o.Status, o.ExpiresAt, o.SentAt, p.Store.SetOfferStatus)
}

func requireInProgress(status, message string) error {
	if status != "in_progress" {
		return conflict(message)
	}
	return nil
}

// ─── psikotes runner ─────────────────────────────────────────

// PsikotesPortal is the portal summary of a session (never questions).
func (p *Portal) PsikotesPortal(ctx context.Context, token string) (*domain.PsikotesSession, error) {
	s, err := p.psikotes(ctx, token, "get")
	if err != nil {
		return nil, err
	}
	if s.Tests, err = p.Store.ListSessionTests(ctx, s.ID, ""); err != nil {
		return nil, err
	}
	return s, nil
}

// StartPsikotes records the camera consent and starts the session.
func (p *Portal) StartPsikotes(ctx context.Context, token string, consent bool) (*domain.PsikotesSession, error) {
	s, err := p.psikotes(ctx, token, "start")
	if err != nil {
		return nil, err
	}
	switch s.Status {
	case "expired":
		return nil, conflict("Link tes sudah kedaluwarsa")
	case "completed":
		return nil, conflict("Sesi tes sudah selesai")
	}
	if _, err := p.Store.StartPsikotesSession(ctx, s.ID, consent); err != nil {
		return nil, err
	}
	return p.PsikotesPortal(ctx, token)
}

// TestRun is a started test: its questions without keys and the saved
// answers.
type TestRun struct {
	Test      domain.SessionTest
	Questions []domain.Question
	Saved     map[string]string
}

func (p *Portal) requireTest(ctx context.Context, sessionID, testID string) (*domain.SessionTest, error) {
	t, err := p.Store.GetSessionTest(ctx, testID, sessionID)
	if err == nil && t == nil {
		err = notFound(msgTestNotFound)
	}
	return t, err
}

// runningTest loads the session (bucket), requires it in progress and loads
// the test.
func (p *Portal) runningTest(ctx context.Context, token, testID, bucket string) (*domain.SessionTest, error) {
	s, err := p.psikotes(ctx, token, bucket)
	if err != nil {
		return nil, err
	}
	if err := requireInProgress(s.Status, "Sesi sudah berakhir"); err != nil {
		return nil, err
	}
	return p.requireTest(ctx, s.ID, testID)
}

// StartTest draws the questions once and returns them without keys.
func (p *Portal) StartTest(ctx context.Context, token, testID string) (*TestRun, error) {
	s, err := p.psikotes(ctx, token, "test_start")
	if err != nil {
		return nil, err
	}
	if err := requireInProgress(s.Status, "Sesi belum dimulai atau sudah berakhir"); err != nil {
		return nil, err
	}
	t, err := p.requireTest(ctx, s.ID, testID)
	if err != nil {
		return nil, err
	}
	if t.Status != "pending" && t.Status != "in_progress" {
		return nil, conflict("Tes ini sudah diselesaikan")
	}
	if t.Status == "pending" {
		ids := []string{}
		if t.InstrumentKind != domain.KindDrawing {
			mcq := t.InstrumentKind == domain.KindMcq
			if ids, err = p.Store.DrawQuestionIDs(ctx, t.InstrumentID, mcq && t.Shuffle, domain.DrawLimit(t.InstrumentKind, t.QuestionCount)); err != nil {
				return nil, err
			}
			if len(ids) == 0 {
				return nil, conflict("Bank soal instrumen ini kosong — hubungi HR")
			}
		}
		// conditional on status: of two racing starts only one draws
		if _, err := p.Store.BeginTest(ctx, t.ID, ids); err != nil {
			return nil, err
		}
		if t, err = p.requireTest(ctx, s.ID, testID); err != nil {
			return nil, err
		}
	}
	run := &TestRun{Test: *t, Questions: []domain.Question{}, Saved: t.Answers}
	if t.InstrumentKind == domain.KindDrawing || len(t.QuestionIDs) == 0 {
		return run, nil
	}
	qs, err := p.drawnQuestions(ctx, t.QuestionIDs)
	if err != nil {
		return nil, err
	}
	for _, q := range qs {
		run.Questions = append(run.Questions, q.Sanitized())
	}
	return run, nil
}

// drawnQuestions loads a test's questions in drawing order (deleted ones
// drop out), so the candidate view and the score snapshot share an order.
func (p *Portal) drawnQuestions(ctx context.Context, ids []string) ([]domain.Question, error) {
	qs, err := p.Store.QuestionsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := map[string]domain.Question{}
	for _, q := range qs {
		byID[q.ID] = q
	}
	out := make([]domain.Question, 0, len(qs))
	for _, id := range ids {
		if q, ok := byID[id]; ok {
			out = append(out, q)
		}
	}
	return out, nil
}

func (p *Portal) assertTestRunning(t *domain.SessionTest) error {
	if t.Status != "in_progress" {
		return conflict("Tes tidak sedang berjalan")
	}
	if domain.PastAnswerGrace(t.StartedAt, t.DurationSeconds, p.now()) {
		return conflict("Waktu tes sudah habis")
	}
	return nil
}

const msgBadPayload = "Payload tidak valid"

// SaveAnswers autosaves drawn answers; returns how many are stored.
func (p *Portal) SaveAnswers(ctx context.Context, token, testID string, answers map[string]string) (int, error) {
	t, err := p.runningTest(ctx, token, testID, "answers")
	if err != nil {
		return 0, err
	}
	if err := p.assertTestRunning(t); err != nil {
		return 0, err
	}
	clean, ok := domain.CleanAnswers(answers)
	if !ok {
		return 0, invalid(msgBadPayload)
	}
	n, ok, err := p.Store.MergeAnswers(ctx, t.ID, domain.PickDrawnAnswers(t.QuestionIDs, clean))
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, conflict("Tes tidak sedang berjalan")
	}
	return n, nil
}

// FinishTest scores a test (drawing tests wait for review). flush is an
// optional last batch of answers; an invalid one is ignored.
func (p *Portal) FinishTest(ctx context.Context, token, testID string, flush map[string]string) (string, string, error) {
	t, err := p.runningTest(ctx, token, testID, "test_finish")
	if err != nil {
		return "", "", err
	}
	if domain.IsTerminalTest(t.Status) {
		return t.Status, "Tes sudah selesai", nil
	}
	if t.Status != "in_progress" {
		return "", "", conflict("Tes belum dimulai")
	}
	if t.InstrumentKind == domain.KindDrawing {
		// past the deadline finish is accepted without a drawing so the
		// session cannot hang; HR sees there is no attachment
		if t.AttachmentKey == nil && !domain.PastAnswerGrace(t.StartedAt, t.DurationSeconds, p.now()) {
			return "", "", invalid("Unggah hasil gambar terlebih dahulu")
		}
		if _, err := p.Store.FinishDrawingTest(ctx, t.ID); err != nil {
			return "", "", err
		}
		return "perlu_review", "Tes selesai — menunggu review HR", nil
	}
	final := map[string]string{}
	for k, v := range t.Answers {
		final[k] = v
	}
	if clean, ok := domain.CleanAnswers(flush); ok {
		for k, v := range domain.PickDrawnAnswers(t.QuestionIDs, clean) {
			final[k] = v
		}
	}
	qs, err := p.drawnQuestions(ctx, t.QuestionIDs)
	if err != nil {
		return "", "", err
	}
	var score *float64
	var detail domain.ScoreDetail
	if t.InstrumentKind == domain.KindMcq {
		mq := make([]domain.McqQuestion, len(qs))
		for i, q := range qs {
			mq[i] = domain.McqQuestion{ID: q.ID}
			if q.CorrectKey != "" {
				key := q.CorrectKey
				mq[i].Correct = &key
			}
		}
		n, d := domain.ScoreMcq(mq, final)
		f := float64(n)
		score, detail = &f, domain.McqScoreDetail(d)
	} else {
		pq := make([]domain.PapiQuestion, len(qs))
		for i, q := range qs {
			pq[i] = domain.PapiQuestion{ID: q.ID}
			if q.Pair != nil {
				pq[i].ScaleA, pq[i].ScaleB = q.Pair.A.Scale, q.Pair.B.Scale
			}
		}
		detail = domain.PapiScoreDetail(domain.ScorePapi(pq, final))
	}
	// conditional on status: of two racing finishes only one writes a score
	if _, err := p.Store.ScoreTest(ctx, t.ID, final, score, detail); err != nil {
		return "", "", err
	}
	return "selesai", "Tes selesai", nil
}

// UploadDrawing stores the photo of a drawing test (jpg, png, webp; 8 MB).
func (p *Portal) UploadDrawing(ctx context.Context, token, testID, name string, data []byte) error {
	t, err := p.runningTest(ctx, token, testID, "upload")
	if err != nil {
		return err
	}
	if t.InstrumentKind != domain.KindDrawing {
		return invalid("Tes ini tidak menerima unggahan gambar")
	}
	if err := p.assertTestRunning(t); err != nil {
		return err
	}
	mime, err := domain.CheckDrawing(name, data)
	if err != nil {
		return err
	}
	ext := map[string]string{domain.MimeJPEG: "jpg", domain.MimePNG: "png", domain.MimeWEBP: "webp"}[mime]
	key := "psikotes/" + t.SessionID + "/" + t.ID + "-" + newID() + "." + ext
	if err := p.Files.Put(ctx, key, data); err != nil {
		return apperr.Wrap(apperr.Unknown, "Upload gagal", err)
	}
	ok, err := p.Store.SetTestAttachment(ctx, t.ID, key, mime)
	if err != nil || !ok {
		p.removeFile(ctx, &key)
		if err != nil {
			return err
		}
		return conflict("Tes tidak sedang berjalan")
	}
	// a re-upload must not orphan the previous file
	p.removeFile(ctx, t.AttachmentKey)
	return nil
}

// FinishPsikotes closes the session once every test is terminal; only the
// request that wins the transition logs the activity.
func (p *Portal) FinishPsikotes(ctx context.Context, token string) (string, string, error) {
	s, err := p.psikotes(ctx, token, "finish")
	if err != nil {
		return "", "", err
	}
	switch s.Status {
	case "completed":
		return "completed", "Sesi sudah selesai", nil
	case "in_progress":
	default:
		return "", "", conflict("Sesi tidak sedang berjalan")
	}
	remaining, err := p.Store.UnfinishedTestNames(ctx, s.ID)
	if err != nil {
		return "", "", err
	}
	if len(remaining) > 0 {
		return "", "", invalid("Masih ada tes yang belum selesai: " + strings.Join(remaining, ", "))
	}
	won := false
	err = p.Store.WithTx(ctx, func(tx domain.Store) error {
		var err error
		if won, err = tx.CompletePsikotesSession(ctx, s.ID); err != nil || !won {
			return err
		}
		return p.activity(ctx, tx, s.CandidateID, "psikotes_completed", "Kandidat menyelesaikan seluruh rangkaian psikotes online", nil, "Sistem")
	})
	if err != nil {
		return "", "", err
	}
	if !won {
		return "completed", "Sesi sudah selesai", nil
	}
	p.notifyCandidateEvent(ctx, s.CandidateID, NotifPsikotesDone, "Psikotes selesai: ", " menyelesaikan seluruh tes psikotes online.")
	return "completed", "Seluruh tes selesai — terima kasih", nil
}

func (p *Portal) notifyCandidateEvent(ctx context.Context, candidateID, typ, titlePrefix, bodySuffix string) {
	cand, err := p.Store.GetCandidate(ctx, candidateID)
	if err != nil || cand == nil {
		return
	}
	p.notify(ctx, Notice{UserIDs: p.recipients(ctx, p.hiringManager(ctx, cand)), Type: typ,
		Title: titlePrefix + cand.FullName, Body: cand.FullName + bodySuffix, Href: candidateHref(cand.ID)})
}

// RecordProctorEvent stores a proctoring flag of a running session.
func (p *Portal) RecordProctorEvent(ctx context.Context, sessionType, token, eventType string, meta map[string]string) (*domain.ProctorEvent, error) {
	var sessionID, status string
	allowed := domain.PsikotesProctorEvents
	if sessionType == "psikotes" {
		s, err := p.psikotes(ctx, token, "proctor")
		if err != nil {
			return nil, err
		}
		sessionID, status = s.ID, s.Status
	} else {
		s, err := p.aiSession(ctx, token, "proctor")
		if err != nil {
			return nil, err
		}
		sessionID, status, allowed = s.ID, s.Status, domain.InterviewProctorEvents
	}
	if err := requireInProgress(status, "Sesi tidak sedang berjalan"); err != nil {
		return nil, err
	}
	if !slices.Contains(allowed, eventType) {
		return nil, apperr.Field("event_type", "Jenis event tidak valid")
	}
	if err := domain.ValidateProctorMeta(meta); err != nil {
		return nil, err
	}
	e := domain.ProctorEvent{ID: newID(), SessionType: sessionType, SessionID: sessionID, EventType: eventType, Meta: meta, CreatedAt: p.now()}
	return &e, p.Store.InsertProctorEvent(ctx, e)
}

// ─── AI interview runner ─────────────────────────────────────

// InterviewPortal is the session with its turns (never the AI summary).
func (p *Portal) InterviewPortal(ctx context.Context, token string) (*domain.AiSession, *domain.Turn, error) {
	s, err := p.aiSession(ctx, token, "get")
	if err != nil {
		return nil, nil, err
	}
	if s.Turns, err = p.Store.ListTurns(ctx, s.ID, ""); err != nil {
		return nil, nil, err
	}
	s.Summary = nil
	return s, activeTurn(s.Turns), nil
}

func activeTurn(turns []domain.Turn) *domain.Turn {
	for i := range turns {
		if turns[i].AnsweredAt == nil {
			return &turns[i]
		}
	}
	return nil
}

func (p *Portal) transcript(s *domain.AiSession, turns []domain.Turn) domain.Transcript {
	return domain.Transcript{CandidateName: s.CandidateName, PositionTitle: s.PositionTitle,
		MaxQuestions: domain.ClampMaxQuestions(s.MaxQuestions), Turns: turns}
}

// nextQuestion is generateNextInterviewQuestion: the static first
// question, gaji forced into the last slot, the AI in between with the
// topic order as fallback.
func (p *Portal) nextQuestion(ctx context.Context, s *domain.AiSession, turns []domain.Turn) domain.NextQ {
	t := p.transcript(s, turns)
	q, needAI := domain.PlanNextQuestion(s.CandidateName, s.PositionTitle, t.MaxQuestions, turns)
	if !needAI {
		return q
	}
	return domain.ResolveNextQuestion(turns, p.AI.InterviewStep(ctx, t))
}

// StartInterview requires the camera consent; a running session returns
// its active question.
func (p *Portal) StartInterview(ctx context.Context, token string, consent bool) (*domain.Turn, error) {
	s, err := p.aiSession(ctx, token, "start")
	if err != nil {
		return nil, err
	}
	switch s.Status {
	case "completed", "expired":
		return nil, conflict("Sesi interview sudah berakhir")
	case "in_progress":
		turns, err := p.Store.ListTurns(ctx, s.ID, "")
		if err != nil {
			return nil, err
		}
		return activeTurn(turns), nil
	}
	if !consent {
		return nil, apperr.Field("webcam_consent", "Interview mewajibkan kamera aktif — izinkan kamera untuk memulai")
	}
	first := p.nextQuestion(ctx, s, nil)
	err = p.Store.WithTx(ctx, func(tx domain.Store) error {
		if _, err := tx.StartAiSession(ctx, s.ID); err != nil {
			return err
		}
		_, err := tx.InsertTurn(ctx, domain.Turn{ID: newID(), SessionID: s.ID, TurnNo: 1, Topic: first.Topic, Question: first.Question, AskedAt: p.now()})
		return err
	})
	if err != nil {
		return nil, err
	}
	turns, err := p.Store.ListTurns(ctx, s.ID, "")
	if err != nil {
		return nil, err
	}
	return activeTurn(turns), nil
}

var errAnswered = errors.New("answered")

// AnswerInterview stores a text answer, then asks the next question or
// closes the interview with a best-effort AI summary. done reports the end.
func (p *Portal) AnswerInterview(ctx context.Context, token, turnID, text string) (*domain.Turn, bool, error) {
	s, err := p.aiSession(ctx, token, "answer")
	if err != nil {
		return nil, false, err
	}
	if err := requireInProgress(s.Status, "Sesi tidak sedang berjalan"); err != nil {
		return nil, false, err
	}
	turn, err := p.Store.GetTurn(ctx, s.ID, turnID)
	if err != nil {
		return nil, false, err
	}
	if turn == nil {
		return nil, false, notFound("Pertanyaan tidak ditemukan")
	}
	if turn.AnsweredAt != nil {
		return nil, false, conflict("Pertanyaan ini sudah dijawab")
	}
	answer, ok := domain.ValidateAnswer(text)
	if !ok {
		return nil, false, apperr.Field("answer_text", "Jawaban kosong")
	}
	if ok, err := p.Store.AnswerTurn(ctx, s.ID, turnID, answer, "text"); err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, conflict("Pertanyaan ini sudah dijawab")
	}
	turns, err := p.Store.ListTurns(ctx, s.ID, "")
	if err != nil {
		return nil, false, err
	}
	next := p.nextQuestion(ctx, s, turns)
	if next.Done() {
		return nil, true, p.closeInterview(ctx, s, turns)
	}
	if _, err := p.Store.InsertTurn(ctx, domain.Turn{ID: newID(), SessionID: s.ID, TurnNo: turn.TurnNo + 1, Topic: next.Topic,
		Question: next.Question, AskedAt: p.now()}); err != nil {
		return nil, false, err
	}
	// a parallel request may have inserted it first: return the active turn
	if turns, err = p.Store.ListTurns(ctx, s.ID, ""); err != nil {
		return nil, false, err
	}
	return activeTurn(turns), false, nil
}

// closeInterview completes the session; a failing AI must not leave it
// hanging, so the summary is best effort.
func (p *Portal) closeInterview(ctx context.Context, s *domain.AiSession, turns []domain.Turn) error {
	var summary *domain.InterviewSummary
	var model *string
	if sum, m, err := p.AI.SummarizeInterview(ctx, p.transcript(s, turns)); err != nil {
		p.log().Error("interview summary failed", "session", s.ID, "err", err)
	} else {
		summary, model = &sum, &m
	}
	err := p.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.CompleteAiSession(ctx, s.ID, summary, model); err != nil {
			return err
		}
		return p.activity(ctx, tx, s.CandidateID, "interview_ai_completed", domain.AiCompletedDescription(len(turns), summary != nil), nil, "")
	})
	if err != nil {
		return err
	}
	p.notifyCandidateEvent(ctx, s.CandidateID, NotifAiInterviewDone, "Interview AI selesai: ", " menyelesaikan interview AI.")
	return nil
}

// ─── offer portal ────────────────────────────────────────────

// OfferPortal is offerForCandidate: HR's internal notes stay out.
func (p *Portal) OfferPortal(ctx context.Context, token string) (*domain.Offer, error) {
	o, err := p.offer(ctx, token)
	if err != nil {
		return nil, err
	}
	o.Notes = ""
	return o, nil
}

// RespondOffer records accept, negotiate or decline with time and IP;
// accept and decline are final, and accepting hires the candidate.
func (p *Portal) RespondOffer(ctx context.Context, token, action, note, ip string) (*domain.Offer, error) {
	o, err := p.offer(ctx, token)
	if err != nil {
		return nil, err
	}
	if !domain.IsOfferOpen(o.Status) {
		return nil, conflict(domain.MsgOfferClosed)
	}
	status, notePtr, err := domain.ValidateOfferResponse(action, note)
	if err != nil {
		return nil, err
	}
	var ipPtr *string
	if ip != "" {
		ipPtr = &ip
	}
	err = p.Store.WithTx(ctx, func(tx domain.Store) error {
		ok, err := tx.RespondOffer(ctx, o.ID, status, notePtr, ipPtr)
		if err != nil {
			return err
		}
		if !ok {
			return errAnswered
		}
		if err := p.activity(ctx, tx, o.CandidateID, "offer_response", domain.OfferResponseDescription(action, o.Version, notePtr), nil, ""); err != nil {
			return err
		}
		if status == "accepted" {
			return p.hireOnAccept(ctx, tx, o.CandidateID, nil, "Sistem")
		}
		return nil
	})
	if errors.Is(err, errAnswered) {
		return nil, conflict(domain.MsgOfferClosed)
	}
	if err != nil {
		return nil, err
	}
	p.afterOfferResponse(ctx, *o, status, deref(notePtr))
	return p.OfferPortal(ctx, token)
}

// OfferLetter renders the offer letter for the candidate.
func (p *Portal) OfferLetter(ctx context.Context, token string) (*File, error) {
	o, err := p.offer(ctx, token)
	if err != nil {
		return nil, err
	}
	return p.letter(ctx, *o)
}

// Company is the employer name the portals show.
func (p *Portal) Company() string { return p.company() }
