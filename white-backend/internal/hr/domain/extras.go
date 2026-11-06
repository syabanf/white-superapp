package domain

import "context"

// Asset is company equipment, optionally lent to an employee.
type Asset struct {
	ID, AssetCode, Name, Category, SerialNo, Status, Notes string
	PurchaseDate                                           *string
	PurchaseValue                                          float64
	EmployeeID                                             *string
	CreatedBy                                              *string
	// Joined.
	EmployeeName, EmployeeNo string
}

type AssetAssignment struct {
	ID, AssetID, EmployeeID, AssignedOn string
	ReturnedOn                          *string
	ConditionOut, ConditionIn           string
	CreatedBy                           *string
	// Joined.
	EmployeeName, EmployeeNo string
}

type AssetFilter struct{ EmployeeID, Status, Search string }

// Training is a course or workshop with its participants.
type Training struct {
	ID, Title, Provider, StartDate, EndDate, Location, Status, Notes string
	Cost                                                             float64
	CreatedBy                                                        *string
	ParticipantCount                                                 int
}

type TrainingParticipant struct {
	TrainingID, EmployeeID, Status, Certificate string
	Score                                       *float64
	// Joined.
	EmployeeName, EmployeeNo, DepartmentName string
}

type TrainingFilter struct {
	Status, EmployeeID string
	Year               int
}

// TimesheetEntry is hours an employee spent on a Finance project on a day.
// An empty ProjectID is internal work: it still counts toward the day's
// total when payroll cost is spread over projects.
type TimesheetEntry struct {
	ID, EmployeeID, WorkDate, ProjectID, ProjectName, Description, Status, RejectReason string
	Hours                                                                               float64
	DecidedBy                                                                           *string
	// Joined.
	EmployeeName, EmployeeNo string
}

type TimesheetFilter struct{ EmployeeID, Status, From, To string }

// ProjectHours is approved time per employee and project.
type ProjectHours struct {
	EmployeeID, ProjectID, ProjectName string
	Hours                              float64
}

type ExtrasRepo interface {
	ListAssets(ctx context.Context, f AssetFilter) ([]Asset, error)
	GetAsset(ctx context.Context, id string) (*Asset, error)
	SaveAsset(ctx context.Context, a Asset) error
	InsertAssetAssignment(ctx context.Context, x AssetAssignment) error
	// CloseAssetAssignment reports false when no assignment was open.
	CloseAssetAssignment(ctx context.Context, assetID, returnedOn, condition string) (bool, error)
	ListAssetAssignments(ctx context.Context, assetID string) ([]AssetAssignment, error)

	ListTrainings(ctx context.Context, f TrainingFilter) ([]Training, error)
	GetTraining(ctx context.Context, id string) (*Training, error)
	SaveTraining(ctx context.Context, t Training) error
	DeleteTraining(ctx context.Context, id string) (bool, error)
	ListTrainingParticipants(ctx context.Context, trainingID string) ([]TrainingParticipant, error)
	// SetTrainingParticipants keeps exactly employeeIDs, leaving results of those already in.
	SetTrainingParticipants(ctx context.Context, trainingID string, employeeIDs []string) error
	UpdateTrainingParticipant(ctx context.Context, p TrainingParticipant) (bool, error)

	ListTimesheet(ctx context.Context, f TimesheetFilter) ([]TimesheetEntry, error)
	GetTimesheetEntry(ctx context.Context, id string) (*TimesheetEntry, error)
	// SumTimesheetDay is the employee's hours that day, leaving out exceptID and rejected entries.
	SumTimesheetDay(ctx context.Context, employeeID, day, exceptID string) (float64, error)
	SaveTimesheetEntry(ctx context.Context, e TimesheetEntry) error
	DeleteTimesheetEntry(ctx context.Context, id string) (bool, error)
	SubmitTimesheet(ctx context.Context, employeeID, from, to string) (int, error)
	DecideTimesheetEntry(ctx context.Context, id, status, reason string, by *string) (bool, error)
	ApprovedProjectHours(ctx context.Context, from, to string) ([]ProjectHours, error)
}
