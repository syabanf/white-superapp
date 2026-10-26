package domain

import (
	"encoding/json"
	"time"
)

// ─── People (from the hr service; never stored beyond ids) ───

// Employee is the hr directory entry the rules need.
type Employee struct {
	ID             string
	UserID         *string
	EmployeeNo     string
	Name           string
	Email          string
	DepartmentID   *string
	DepartmentName string
	PositionName   string
	ManagerID      *string
	Active         bool
}

// InDepartment reports membership of department id.
func (e Employee) InDepartment(id string) bool { return e.DepartmentID != nil && *e.DepartmentID == id }

// Department is an hr department.
type Department struct {
	ID             string
	Code           string
	Name           string
	HeadEmployeeID *string
	Active         bool
}

// ─── KPI ─────────────────────────────────────────────────────

type Indicator struct {
	ID            string
	Code          string
	Name          string
	Description   string
	Unit          string
	Direction     string
	SourceKind    string
	DefaultTarget *float64
	IsActive      bool
	CreatedBy     *string
	UpdatedBy     *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// DepartmentWeight is one kpi_department_indicators row.
type DepartmentWeight struct {
	DepartmentID  string
	IndicatorID   string
	IndicatorCode string
	Weight        float64
	UpdatedBy     *string
	UpdatedAt     time.Time
}

type Snapshot struct {
	ID            string
	EmployeeID    string
	IndicatorID   string
	IndicatorCode string
	IndicatorName string
	SourceKind    string
	Period        Period
	Actual        *float64
	Target        *float64
	Attainment    *float64
	SampleSize    float64
	SourceDetail  json.RawMessage
	UpdatedAt     time.Time
}

type Scorecard struct {
	ID           string
	EmployeeID   string
	DepartmentID *string
	Period       Period
	Score        *float64
	RawScore     *float64
	UsedWeight   float64
	Breakdown    []BreakdownRow
	Status       string
	ReviewedBy   *string
	ReviewedAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ScoreAverage is an employee's average over the latest scorecards.
type ScoreAverage struct {
	EmployeeID string
	AvgScore   float64
	Periods    int
	HasFinal   bool
}

// ─── Reviews ─────────────────────────────────────────────────

type ReviewCycle struct {
	ID            string
	Name          string
	Year          int
	Quarter       int
	StartDate     string
	EndDate       string
	DueDate       string
	Status        string
	CreatedBy     *string
	CreatedByName string
	CreatedAt     time.Time
	TotalReviews  int
	FinalReviews  int
	RatedReviews  int
}

type BehavioralStandard struct {
	ID                  string
	ValueName           string
	CompetencyName      string
	StandardDescription string
	ScoreDescriptions   [5]string
	Weight              float64
}

type Review struct {
	ID               string
	CycleID          string
	CycleName        string
	CycleStatus      string
	CycleDueDate     string
	CycleYear        int
	CycleQuarter     int
	EmployeeID       string
	DepartmentID     *string
	PeriodLabel      string
	StartDate        string
	EndDate          string
	Work             *float64
	Behavior         *float64
	Project          *float64
	Grand            *float64
	Category         *string
	Status           string
	SelfAssessment   *string
	ReviewerID       *string
	ReviewerName     *string
	ReviewerNotes    *string
	ManagerID        *string
	ManagerNotes     *string
	EmployeeSignDate *string
	ReviewerSignDate *string
	FinalizedBy      *string
	FinalizedAt      *time.Time
	RatedItems       int
	TotalItems       int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ReviewItem struct {
	ID                 string
	ReviewID           string
	ValueName          string
	Competency         string
	BehavioralStandard string
	ScoreDescriptions  [5]string
	Weight             float64
	Score              *int
	WeightedScore      *float64
	Notes              *string
	ItemOrder          int
}

// ─── 360 feedback ────────────────────────────────────────────

type FeedbackCycle struct {
	ID                   string
	Name                 string
	PeriodLabel          string
	StartDate            *string
	EndDate              *string
	Status               string
	KPIWeight            *float64
	FeedbackWeight       *float64
	IsAnonymous          bool
	AllowSelfAssessment  bool
	RequireManagerReview bool
	CreatedBy            *string
	UpdatedBy            *string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	AssignmentsCount     int
	SubmittedCount       int
	SummariesCount       int
}

type FeedbackCriterion struct {
	ID           string
	CategoryID   string
	Name         string
	Description  string
	DisplayOrder int
}

type FeedbackCategory struct {
	ID           string
	Code         string
	Name         string
	Description  string
	Weight       float64
	DisplayOrder int
	Criteria     []FeedbackCriterion
}

type FeedbackAssignment struct {
	ID              string
	CycleID         string
	CycleName       string
	CycleStatus     string
	EmployeeID      string
	ReviewerID      string
	Relationship    string
	Status          string
	DueDate         *string
	StartedAt       *time.Time
	SubmittedAt     *time.Time
	ReminderCount   int
	LastReminderAt  *time.Time
	ManagerComments *string
	RejectionReason *string
	ApprovedBy      *string
	ApprovedAt      *time.Time
	CreatedBy       *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type FeedbackResponse struct {
	ID           string
	AssignmentID string
	CriteriaID   string
	Rating       int
	Comments     *string
}

// CycleRating is one answer of a cycle with its context.
type CycleRating struct {
	EmployeeID       string
	AssignmentStatus string
	FeedbackRating
	Comments *string
}

type FeedbackSummary struct {
	ID              string
	CycleID         string
	EmployeeID      string
	Scores          map[string]*float64
	Overall360      *float64
	KPIScore        *float64
	FinalScore      *float64
	FinalGrade      *string
	Strengths       []string
	Weaknesses      []string
	ManagerComments *string
	ResponsesCount  int
	IsLocked        bool
	GeneratedAt     time.Time
	UpdatedAt       time.Time
}

// ─── Department tasks ────────────────────────────────────────

type Task struct {
	ID                  string
	DepartmentID        string
	AssigneeEmployeeID  *string
	Title               string
	Description         *string
	Recurrence          string
	WeeklyDay           *int
	MonthlyDay          *int
	DueDate             *string
	IsActive            bool
	Priority            string
	KPIIndicatorID      *string
	KPIContribution     *float64
	Timezone            string
	ReviewRequired      bool
	CreatedByEmployeeID *string
	CreatedByUserID     *string
	CreatedByName       string
	UpdatedBy           *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// TaskScheduleRow is an active task's recurrence for occurrence generation.
type TaskScheduleRow struct {
	TaskID string
	TaskSchedule
}

// Occurrence is one department_task_occurrences row joined with the
// task fields the state machine needs.
type Occurrence struct {
	ID              string
	TaskID          string
	Date            string
	Status          string
	ProgressPercent *int
	StartedAt       *time.Time
	DoneAt          *time.Time
	DoneBy          *string
	DoneByName      *string
	ReviewNotes     *string
	ReviewedBy      *string
	ReviewedByName  *string
	ReviewedAt      *time.Time
	// From the task.
	Title          string
	DepartmentID   string
	AssigneeID     *string
	ReviewRequired bool
	CreatorUserID  *string
	KPIIndicatorID *string
}

// TaskItem is one occurrence as the views return it.
type TaskItem struct {
	OccurrenceID     string
	TaskID           string
	Title            string
	Description      string
	DepartmentID     string
	AssigneeID       *string
	CreatorUserID    *string
	CreatorName      string
	Date             string
	Status           string
	ProgressPercent  *int
	StartedAt        *time.Time
	DoneAt           *time.Time
	DoneByName       *string
	ReviewNotes      *string
	ReviewedByName   *string
	Recurrence       string
	Priority         string
	ReviewRequired   bool
	Timezone         string
	KPIIndicatorID   *string
	KPIIndicatorCode *string
	KPIIndicatorName *string
	KPIContribution  *float64
	SubtaskTotal     *float64
	SubtaskDone      *float64
	SubtaskCount     int
	CommentCount     int
	AttachmentCount  int
	LastNote         *string
	LastNoteBy       *string
	LastNoteAt       *time.Time
	WeeklyDay        *int
	MonthlyDay       *int
	DueDate          *string
}

// SubtaskPercent is the weighted share of ticked sub-tasks, nil without any.
func (i TaskItem) SubtaskPercent() *float64 {
	if i.SubtaskTotal == nil || *i.SubtaskTotal <= 0 {
		return nil
	}
	done := 0.0
	if i.SubtaskDone != nil {
		done = *i.SubtaskDone
	}
	v := done / *i.SubtaskTotal * 100
	return &v
}

// ItemQuery selects occurrences. Nil and empty fields do not filter.
type ItemQuery struct {
	OccurrenceID  string
	DepartmentIDs []string
	// Mine: assigned to the employee, unassigned in their department, or
	// filed by the user.
	MineEmployeeID   string
	MineDepartmentID string
	MineUserID       string
	MineOnly         bool
	SpanStart        string
	SpanEnd          string
	// OverdueBefore also returns open rows dated before this day.
	OverdueBefore string
	AssigneeID    string
	Status        string
	IndicatorID   string
	KPIOnly       bool
	Q             string
}

type SubtaskState struct {
	ID            string
	Title         string
	Weight        float64
	SortOrder     int
	IsChecked     bool
	CheckedByName *string
	CheckedAt     *time.Time
}

type TaskUpdate struct {
	ID               string
	OccurrenceID     string
	Kind             string
	Note             *string
	ProgressPercent  *int
	AuthorEmployeeID *string
	AuthorUserID     *string
	AuthorName       string
	Detail           map[string]any
	CreatedAt        time.Time
}

type TaskAttachment struct {
	ID               string
	OccurrenceID     string
	Name             string
	Mime             string
	SizeBytes        int64
	StoragePath      string
	UploadedByUserID *string
	UploadedByName   string
	CreatedAt        time.Time
}

type TaskContribution struct {
	OccurrenceID string
	IndicatorID  string
	DepartmentID string
	EmployeeID   *string
	Period       Period
	Amount       *float64
	RecordedAt   time.Time
}

// ContributionSummary is an employee's linked occurrences of one
// indicator in a month and what they added.
type ContributionSummary struct {
	IndicatorID       string
	Code              string
	TasksTotal        int
	TasksDone         int
	ContributionTotal float64
}

// KPIIndicatorWeight is an indicator of a department's KPI view.
type KPIIndicatorWeight struct {
	Indicator
	Weight *float64
}

// ReminderLine is one open occurrence of an assignee.
type ReminderLine struct {
	EmployeeID string
	Title      string
	Date       string
	Priority   string
}

// RecapOccurrence is an occurrence an employee touched in a window.
type RecapOccurrence struct {
	OccurrenceID    string
	TaskID          string
	Title           string
	Status          string
	Priority        string
	ProgressPercent *int
	SubtaskTotal    *float64
	SubtaskDone     *float64
	CompletedInDay  bool
}

// RecapNote is a history note an employee wrote in a window.
type RecapNote struct {
	OccurrenceID string
	Note         string
}

// ─── Logbook ─────────────────────────────────────────────────

type LogbookTemplate struct {
	ID           string
	DepartmentID string
	Name         string
	Description  *string
	Frequency    string
	IsActive     bool
	CreatedBy    *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Items        []LogbookTemplateItem
	// DepartmentName comes from the hr directory, not the store.
	DepartmentName string
}

type LogbookTemplateItem struct {
	ID          string
	TemplateID  string
	Title       string
	Description *string
	Weight      float64
	IsRequired  bool
	SortOrder   int
}

type LogbookEntry struct {
	ID           string
	TemplateID   string
	TemplateName string
	Frequency    string
	DepartmentID string
	EmployeeID   string
	EntryDate    string
	Title        string
	Status       string
	Completion   float64
	KPIScore     float64
	Notes        *string
	SubmittedBy  *string
	SubmittedAt  *time.Time
	ReviewedBy   *string
	ReviewedAt   *time.Time
	ReviewNotes  *string
	CreatedBy    *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Items        []LogbookEntryItem
	// EmployeeName and DepartmentName come from the hr directory, not the store.
	EmployeeName, DepartmentName string
}

type LogbookEntryItem struct {
	ID             string
	EntryID        string
	TemplateItemID *string
	Title          string
	Description    *string
	Weight         float64
	IsRequired     bool
	SortOrder      int
	IsChecked      bool
	CheckedBy      *string
	CheckedAt      *time.Time
	Notes          *string
}

// EntryFilter selects logbook entries; empty fields do not filter.
type EntryFilter struct {
	DepartmentID string
	EmployeeIDs  []string
	Status       string
	Date         string
	From         string
	To           string
	Limit        int
	Offset       int
}

// LogbookStats are an employee's logbook numbers over a span.
type LogbookStats struct {
	Entries int
	LogbookCount
}

// ─── Recaps ──────────────────────────────────────────────────

type DailyRecap struct {
	ID           string
	EmployeeID   string
	DepartmentID *string
	Date         string
	Items        DailyRecapItems
	Notes        string
	Summary      string
	SummaryModel string
	GeneratedAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// DailyRecapFilter selects daily recaps.
type DailyRecapFilter struct {
	Date        string
	EmployeeIDs []string
	From        string
	To          string
	Limit       int
}

type MonthlyRecap struct {
	ID              string
	Scope           string
	SubjectID       string
	DepartmentID    *string
	Period          Period
	Metrics         MonthlyMetrics
	Accomplishments []string
	Narrative       string
	NarrativeModel  string
	GeneratedAt     time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// MonthlyRecapFilter selects monthly recaps.
type MonthlyRecapFilter struct {
	Period       *Period
	Scope        string
	SubjectIDs   []string
	DepartmentID string
	Limit        int
}

// ─── Improvements ────────────────────────────────────────────

type Improvement struct {
	ID              string
	Scope           string
	EmployeeID      *string
	DepartmentID    *string
	Source          string
	SourceRef       *string
	Title           string
	Description     string
	OwnerEmployeeID *string
	DueDate         *string
	Status          string
	LinkedTaskID    *string
	CompletedAt     *time.Time
	CreatedBy       *string
	UpdatedBy       *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ImprovementFilter selects improvements; empty fields do not filter.
// PersonID matches the employee or the owner.
type ImprovementFilter struct {
	Scope         string
	EmployeeIDs   []string
	DepartmentIDs []string
	PersonID      string
	Statuses      []string
	Source        string
	Limit         int
}

type RetroSession struct {
	ID           string
	DepartmentID string
	Period       Period
	WentWell     []string
	ToImprove    []string
	Actions      []RetroAction
	Status       string
	CreatedBy    *string
	UpdatedBy    *string
	ClosedAt     *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
