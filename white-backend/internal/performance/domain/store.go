package domain

import (
	"context"
	"time"
)

// ErrUnique is a unique-constraint violation.
type ErrUnique struct{ Constraint string }

func (e *ErrUnique) Error() string { return "unique violation: " + e.Constraint }

// KPIStore persists indicators, weights, targets, snapshots, scorecards.
type KPIStore interface {
	ListIndicators(ctx context.Context, includeInactive bool) ([]Indicator, error)
	GetIndicator(ctx context.Context, id string) (*Indicator, error)
	GetIndicatorByCode(ctx context.Context, code string) (*Indicator, error)
	UpsertIndicator(ctx context.Context, in Indicator) error
	ListDepartmentWeights(ctx context.Context) ([]DepartmentWeight, error)
	DeleteDepartmentWeights(ctx context.Context, departmentID string) error
	InsertDepartmentWeight(ctx context.Context, w DepartmentWeight) error
	ListTargets(ctx context.Context) ([]KPITarget, error)
	InsertTarget(ctx context.Context, t KPITarget, by string) error
	DeleteTarget(ctx context.Context, id string) (bool, error)
	UpsertSnapshot(ctx context.Context, s Snapshot) error
	ListSnapshots(ctx context.Context, employeeID string, p Period) ([]Snapshot, error)
	ListManualSnapshots(ctx context.Context, p Period) ([]Snapshot, error)
	// UpsertDraftScorecard writes a draft; a final scorecard is untouched
	// (false).
	UpsertDraftScorecard(ctx context.Context, s Scorecard) (bool, error)
	GetScorecard(ctx context.Context, id string) (*Scorecard, error)
	GetScorecardFor(ctx context.Context, employeeID string, p Period) (*Scorecard, error)
	ListScorecards(ctx context.Context, p Period, employeeIDs []string) ([]Scorecard, error)
	ScorecardHistory(ctx context.Context, employeeID string, limit int) ([]Scorecard, error)
	QuarterScorecards(ctx context.Context, employeeIDs []string, year int, months []int) ([]Scorecard, error)
	SetScorecardStatus(ctx context.Context, id, status string, reviewedBy *string, at *time.Time) error
	FinalScorecardEmployees(ctx context.Context, p Period) ([]string, error)
	RecentScoreAverages(ctx context.Context, employeeIDs []string) ([]ScoreAverage, error)
	TaskCounts(ctx context.Context, span DaySpan) ([]TaskCount, error)
	LogbookStats(ctx context.Context, span DaySpan) (map[string]LogbookStats, error)
}

// ReviewStore persists quarterly reviews.
type ReviewStore interface {
	ListReviewCycles(ctx context.Context) ([]ReviewCycle, error)
	GetReviewCycle(ctx context.Context, id string) (*ReviewCycle, error)
	ReviewCycleExists(ctx context.Context, year, quarter int) (bool, error)
	InsertReviewCycle(ctx context.Context, c ReviewCycle) error
	SetReviewCycleStatus(ctx context.Context, id, status string) error
	ListBehavioralStandards(ctx context.Context) ([]BehavioralStandard, error)
	ListPerformanceCategories(ctx context.Context) ([]Category, error)
	InsertReview(ctx context.Context, r Review) error
	InsertReviewItem(ctx context.Context, it ReviewItem) error
	ListReviewsByCycle(ctx context.Context, cycleID string) ([]Review, error)
	ListOpenCycleReviews(ctx context.Context) ([]Review, error)
	GetReview(ctx context.Context, id string) (*Review, error)
	UpdateReview(ctx context.Context, r Review) error
	ListReviewItems(ctx context.Context, reviewID string) ([]ReviewItem, error)
	UpdateReviewItem(ctx context.Context, it ReviewItem) error
}

// FeedbackStore persists 360 feedback.
type FeedbackStore interface {
	ListFeedbackCategories(ctx context.Context) ([]FeedbackCategory, error)
	ListFeedbackCycles(ctx context.Context, status string) ([]FeedbackCycle, error)
	GetFeedbackCycle(ctx context.Context, id string) (*FeedbackCycle, error)
	InsertFeedbackCycle(ctx context.Context, c FeedbackCycle) error
	UpdateFeedbackCycle(ctx context.Context, c FeedbackCycle) error
	DeleteFeedbackCycle(ctx context.Context, id string) error
	// InsertAssignment returns false when the pair already exists.
	InsertAssignment(ctx context.Context, a FeedbackAssignment) (bool, error)
	ListAssignments(ctx context.Context, cycleID, employeeID, reviewerID, status string) ([]FeedbackAssignment, error)
	ListRemindableAssignments(ctx context.Context) ([]FeedbackAssignment, error)
	GetAssignment(ctx context.Context, id string) (*FeedbackAssignment, error)
	UpdateAssignment(ctx context.Context, a FeedbackAssignment) error
	DeleteAssignment(ctx context.Context, id string) error
	UpsertResponse(ctx context.Context, r FeedbackResponse) error
	ListResponses(ctx context.Context, assignmentID string) ([]FeedbackResponse, error)
	CycleRatings(ctx context.Context, cycleID string) ([]CycleRating, error)
	UpsertFeedbackSummary(ctx context.Context, s FeedbackSummary) error
	ListFeedbackSummaries(ctx context.Context, cycleID, employeeID string) ([]FeedbackSummary, error)
	GetFeedbackSummary(ctx context.Context, id string) (*FeedbackSummary, error)
	UpdateFeedbackSummary(ctx context.Context, s FeedbackSummary) error
}

// TaskStore persists department tasks.
type TaskStore interface {
	ActiveTaskSchedules(ctx context.Context, departmentIDs []string) ([]TaskScheduleRow, error)
	ActiveTaskDepartments(ctx context.Context) ([]string, error)
	InsertOccurrences(ctx context.Context, taskIDs, dates []string) error
	InsertTask(ctx context.Context, t Task) error
	InsertSubtask(ctx context.Context, taskID string, s Subtask, order int) error
	GetTask(ctx context.Context, id string) (*Task, error)
	UpdateTask(ctx context.Context, t Task) error
	QueryItems(ctx context.Context, q ItemQuery) ([]TaskItem, error)
	GetOccurrence(ctx context.Context, id string) (*Occurrence, error)
	SaveOccurrence(ctx context.Context, o Occurrence) error
	CountSubtasks(ctx context.Context, taskID string) (int, error)
	SubtaskExists(ctx context.Context, taskID, subtaskID string) (bool, error)
	ListSubtaskStates(ctx context.Context, occurrenceID, taskID string) ([]SubtaskState, error)
	UpsertOccurrenceItem(ctx context.Context, occurrenceID, subtaskID string, checked bool, at *time.Time, by *string) error
	SubtaskProgress(ctx context.Context, occurrenceID, taskID string) (total, done float64, err error)
	InsertTaskUpdate(ctx context.Context, u TaskUpdate) error
	ListTaskUpdates(ctx context.Context, occurrenceID string, limit int) ([]TaskUpdate, error)
	InsertAttachment(ctx context.Context, a TaskAttachment) error
	ListAttachments(ctx context.Context, occurrenceID string) ([]TaskAttachment, error)
	GetAttachment(ctx context.Context, id string) (*TaskAttachment, error)
	DeleteAttachment(ctx context.Context, id string) error
	RecordContribution(ctx context.Context, occurrenceID, taskID string, p Period) error
	DropContribution(ctx context.Context, occurrenceID string) error
	GetContribution(ctx context.Context, occurrenceID string) (*TaskContribution, error)
	ContributionTotals(ctx context.Context, departmentID string, span DaySpan) (map[string]float64, error)
	EmployeeContributions(ctx context.Context, employeeID string, span DaySpan) ([]ContributionSummary, error)
	DepartmentKPIIndicators(ctx context.Context, departmentID string) ([]KPIIndicatorWeight, error)
	IndicatorDepartments(ctx context.Context) (map[string][]string, error)
	ReminderLines(ctx context.Context, upTo string) ([]ReminderLine, error)
	RecapOccurrences(ctx context.Context, employeeID string, from, to time.Time) ([]RecapOccurrence, error)
	RecapNotes(ctx context.Context, employeeID string, from, to time.Time) ([]RecapNote, error)
	MonthTasks(ctx context.Context, employeeIDs []string, span DaySpan) (map[string][]MonthTask, error)
}

// LogbookStore persists logbook templates and entries.
type LogbookStore interface {
	ListTemplates(ctx context.Context, departmentID string, includeInactive bool) ([]LogbookTemplate, error)
	GetTemplate(ctx context.Context, id string) (*LogbookTemplate, error)
	InsertTemplate(ctx context.Context, t LogbookTemplate) error
	InsertTemplateItem(ctx context.Context, it LogbookTemplateItem) error
	CountTemplateEntries(ctx context.Context, templateID string) (int, error)
	ArchiveTemplate(ctx context.Context, id string) error
	DeleteTemplate(ctx context.Context, id string) error
	InsertEntry(ctx context.Context, e LogbookEntry) error
	CopyTemplateItems(ctx context.Context, entryID, templateID string) error
	GetEntry(ctx context.Context, id string) (*LogbookEntry, error)
	GetEntryItem(ctx context.Context, id string) (*LogbookEntryItem, error)
	ListEntries(ctx context.Context, f EntryFilter) ([]LogbookEntry, int, error)
	UpdateEntryItem(ctx context.Context, it LogbookEntryItem) error
	UpdateEntry(ctx context.Context, e LogbookEntry) error
	SetEntryScore(ctx context.Context, id string, completion, kpiScore float64) error
	DeleteEntry(ctx context.Context, id string) error
	LogbookSummaryRows(ctx context.Context, departmentID string, from, to string) ([]LogbookSummaryEntry, error)
	DailyTemplateDepartments(ctx context.Context) ([]string, error)
}

// RecapStore persists daily and monthly recaps.
type RecapStore interface {
	GetDailyRecap(ctx context.Context, employeeID, date string) (*DailyRecap, error)
	// UpsertDailyRecap writes items, summary and model; notes are kept.
	UpsertDailyRecap(ctx context.Context, r DailyRecap) error
	SetDailyNotes(ctx context.Context, r DailyRecap) error
	ListDailyRecaps(ctx context.Context, f DailyRecapFilter) ([]DailyRecap, error)
	UpsertMonthlyRecap(ctx context.Context, r MonthlyRecap) (string, error)
	GetMonthlyRecap(ctx context.Context, id string) (*MonthlyRecap, error)
	ListMonthlyRecaps(ctx context.Context, f MonthlyRecapFilter) ([]MonthlyRecap, error)
}

// ImprovementStore persists improvement items and retros.
type ImprovementStore interface {
	// InsertImprovement returns false when source+source_ref already exists.
	InsertImprovement(ctx context.Context, imp Improvement) (bool, error)
	GetImprovement(ctx context.Context, id string) (*Improvement, error)
	UpdateImprovement(ctx context.Context, imp Improvement) error
	SoftDeleteImprovement(ctx context.Context, id string, by string) error
	ListImprovements(ctx context.Context, f ImprovementFilter) ([]Improvement, error)
	OverdueImprovements(ctx context.Context, today string) ([]Improvement, error)
	ListRetros(ctx context.Context, departmentIDs []string) ([]RetroSession, error)
	GetRetro(ctx context.Context, id string) (*RetroSession, error)
	GetRetroFor(ctx context.Context, departmentID string, p Period) (*RetroSession, error)
	InsertRetro(ctx context.Context, r RetroSession) error
	UpdateRetro(ctx context.Context, r RetroSession) error
}

// Store is every port plus transactions and the reminder sent log.
type Store interface {
	KPIStore
	ReviewStore
	FeedbackStore
	TaskStore
	LogbookStore
	RecapStore
	ImprovementStore
	DashboardStore
	// ClaimSend records (kind, subject, day); false when already sent.
	ClaimSend(ctx context.Context, kind, subjectID, day string) (bool, error)
	WithTx(ctx context.Context, fn func(tx Store) error) error
}
