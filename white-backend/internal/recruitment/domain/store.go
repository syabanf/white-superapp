package domain

import (
	"context"
	"time"
)

// CandidateFilter narrows the candidate list.
type CandidateFilter struct {
	Statuses     []string
	JobOpeningID string
	Search       string
	Source       string
	DateFrom     *time.Time
	DateTo       *time.Time
	SortUpdated  bool
	Limit        int
	Offset       int
}

// InterviewFilter narrows the interview list.
type InterviewFilter struct {
	CandidateID           string
	Status                string
	InterviewerEmployeeID string
	From                  *time.Time
	To                    *time.Time
}

// Store is every recruitment repository. A nil pointer result means "not
// found"; bool results report whether a conditional write matched.
type Store interface {
	WithTx(ctx context.Context, fn func(tx Store) error) error

	// job openings
	ListJobOpenings(ctx context.Context, status, search string) ([]JobOpening, error)
	GetJobOpening(ctx context.Context, id string) (*JobOpening, error)
	GetJobOpeningBySlug(ctx context.Context, slug string) (*JobOpening, error)
	ListPublicJobs(ctx context.Context, today, search, department string) ([]JobOpening, error)
	InsertJobOpening(ctx context.Context, o JobOpening) error
	UpdateJobOpening(ctx context.Context, o JobOpening) (bool, error)
	DeleteJobOpening(ctx context.Context, id, by string) (bool, error)

	// candidates
	ListCandidates(ctx context.Context, f CandidateFilter) ([]Candidate, int, error)
	GetCandidate(ctx context.Context, id string) (*Candidate, error)
	InsertCandidate(ctx context.Context, c Candidate) error
	UpdateCandidateProfile(ctx context.Context, id string, f CandidateFields, by string) (bool, error)
	SetCandidateStatus(ctx context.Context, id, status string, by *string) (bool, error)
	MarkHiredEvent(ctx context.Context, id string, at time.Time) error
	PendingHiredEvents(ctx context.Context) ([]Candidate, error)
	DeleteCandidate(ctx context.Context, id string) (bool, error)
	SetCandidateCV(ctx context.Context, id string, key, name, contentType *string, by string) (bool, error)
	CandidateAppliedTo(ctx context.Context, email, jobOpeningID string) (bool, error)
	CountByStatus(ctx context.Context, jobOpeningID string) (map[string]int, error)
	CountBySource(ctx context.Context) ([]Count, error)
	Summary(ctx context.Context, now time.Time) (Summary, error)
	DashboardTrend(ctx context.Context, f DashboardFilter, p Period) ([]TrendBucket, error)
	DashboardDetail(ctx context.Context, f DashboardFilter, p Period, now time.Time) (*DashboardDetail, error)

	ListNotes(ctx context.Context, candidateID string) ([]Note, error)
	InsertNote(ctx context.Context, n Note) error
	ListActivities(ctx context.Context, candidateID string) ([]Activity, error)
	InsertActivity(ctx context.Context, a Activity) error

	GetScreening(ctx context.Context, candidateID string) (*Screening, error)
	UpsertScreening(ctx context.Context, s Screening) error
	GetPsikotesSummary(ctx context.Context, candidateID string) (*PsikotesSummary, error)
	UpsertPsikotesSummary(ctx context.Context, s PsikotesSummary) error
	GetAnalysis(ctx context.Context, candidateID string) (*CandidateAnalysis, error)
	UpsertAnalysis(ctx context.Context, a CandidateAnalysis) error

	// psikotes bank
	ListInstruments(ctx context.Context) ([]Instrument, error)
	GetInstrument(ctx context.Context, id string) (*Instrument, error)
	ActiveInstruments(ctx context.Context, ids []string) ([]Instrument, error)
	UpdateInstrument(ctx context.Context, id string, p InstrumentPatch, by string) (bool, error)
	ListQuestions(ctx context.Context, instrumentID string) ([]Question, error)
	GetQuestion(ctx context.Context, id string) (*Question, error)
	QuestionsByIDs(ctx context.Context, ids []string) ([]Question, error)
	InsertQuestion(ctx context.Context, q Question, by string) error
	UpdateQuestion(ctx context.Context, q Question, by string) (bool, error)
	DeleteQuestion(ctx context.Context, id string) (bool, error)
	DrawQuestionIDs(ctx context.Context, instrumentID string, shuffle bool, limit int) ([]string, error)

	// psikotes sessions
	InsertPsikotesSession(ctx context.Context, s PsikotesSession) error
	ListPsikotesSessions(ctx context.Context, candidateID string) ([]PsikotesSession, error)
	PsikotesSessionByToken(ctx context.Context, token string) (*PsikotesSession, error)
	SetPsikotesSessionStatus(ctx context.Context, id, status string) error
	StartPsikotesSession(ctx context.Context, id string, consent bool) (bool, error)
	CompletePsikotesSession(ctx context.Context, id string) (bool, error)
	ListSessionTests(ctx context.Context, sessionID, candidateID string) ([]SessionTest, error)
	GetSessionTest(ctx context.Context, id, sessionID string) (*SessionTest, error)
	BeginTest(ctx context.Context, id string, questionIDs []string) (bool, error)
	MergeAnswers(ctx context.Context, id string, answers map[string]string) (int, bool, error)
	ScoreTest(ctx context.Context, id string, answers map[string]string, score *float64, detail ScoreDetail) (bool, error)
	FinishDrawingTest(ctx context.Context, id string) (bool, error)
	SetTestAttachment(ctx context.Context, id, key, contentType string) (bool, error)
	UnfinishedTestNames(ctx context.Context, sessionID string) ([]string, error)
	ReviewTest(ctx context.Context, id, notes string, by *string, byName string) (bool, error)

	InsertProctorEvent(ctx context.Context, e ProctorEvent) error
	ListProctorEvents(ctx context.Context, sessionType, sessionID string) ([]ProctorEvent, error)
	ProctorTallies(ctx context.Context, sessionType string, sessionIDs []string) (map[string]ProctorTally, error)
	SessionExists(ctx context.Context, sessionType, id string) (bool, error)

	// AI interview
	InsertAiSession(ctx context.Context, s AiSession) error
	ListAiSessions(ctx context.Context, candidateID string) ([]AiSession, error)
	AiSessionByToken(ctx context.Context, token string) (*AiSession, error)
	SetAiSessionStatus(ctx context.Context, id, status string) error
	StartAiSession(ctx context.Context, id string) (bool, error)
	CompleteAiSession(ctx context.Context, id string, summary *InterviewSummary, model *string) error
	ListTurns(ctx context.Context, sessionID, candidateID string) ([]Turn, error)
	GetTurn(ctx context.Context, sessionID, id string) (*Turn, error)
	InsertTurn(ctx context.Context, t Turn) (bool, error)
	AnswerTurn(ctx context.Context, sessionID, id, answer, mode string) (bool, error)

	// scheduled interviews
	ListInterviews(ctx context.Context, f InterviewFilter) ([]Interview, error)
	GetInterview(ctx context.Context, id string) (*Interview, error)
	InsertInterview(ctx context.Context, i Interview) error
	UpdateInterview(ctx context.Context, i Interview) (bool, error)

	// offers
	ListOffers(ctx context.Context, candidateID string) ([]Offer, error)
	GetOffer(ctx context.Context, id string) (*Offer, error)
	OfferByToken(ctx context.Context, token string) (*Offer, error)
	ExpireOpenOffers(ctx context.Context, candidateID string) error
	NextOfferVersion(ctx context.Context, candidateID string) (int, error)
	InsertOffer(ctx context.Context, o Offer) error
	RespondOffer(ctx context.Context, id, status string, note, ip *string) (bool, error)
	RecordManualResponse(ctx context.Context, id, status string, note *string) (bool, error)
	SetOfferStatus(ctx context.Context, id, status string) error
	LatestAcceptedOffer(ctx context.Context, candidateID string) (*Offer, error)
	ExpiringOffers(ctx context.Context, now, until time.Time) ([]Offer, error)
	ExpireLapsed(ctx context.Context, now time.Time) (int, error)

	// plumbing
	HitRateLimit(ctx context.Context, key string, now time.Time, window time.Duration) (int, error)
	PruneRateLimits(ctx context.Context, before time.Time) error
	ClaimReminder(ctx context.Context, kind, refID, day string) (bool, error)
}
