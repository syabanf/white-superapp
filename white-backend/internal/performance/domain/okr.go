package domain

import (
	"context"
	"time"
)

// Objective is a quarterly goal at company, department or employee level.
type Objective struct {
	ID, Period, Level, Title, Description, Status string
	DepartmentID, EmployeeID, ParentID, CreatedBy *string
}

// KeyResult measures an objective from Start to Target.
type KeyResult struct {
	ID, ObjectiveID, Title, Metric, Unit string
	Start, Target, Current               float64
	SortOrder                            int
}

// Progress is how far Current moved from Start toward Target, 0 to 1. A
// target below the start (cost down, churn down) works the same way.
func (k KeyResult) Progress() float64 {
	p := (k.Current - k.Start) / (k.Target - k.Start)
	return min(max(p, 0), 1)
}

type Checkin struct {
	ID, KeyResultID, Note string
	Value                 float64
	Confidence            int
	CreatedBy             *string
	CreatedAt             time.Time
}

const (
	OKRCompany    = "company"
	OKRDepartment = "department"
	OKREmployee   = "employee"
)

type OKRStore interface {
	ListObjectives(ctx context.Context, period string) ([]Objective, error)
	GetObjective(ctx context.Context, id string) (*Objective, error)
	SaveObjective(ctx context.Context, o Objective) error
	DeleteObjective(ctx context.Context, id string) (bool, error)
	ListKeyResults(ctx context.Context, period string) ([]KeyResult, error)
	GetKeyResult(ctx context.Context, id string) (*KeyResult, error)
	SaveKeyResult(ctx context.Context, k KeyResult) error
	DeleteKeyResult(ctx context.Context, id string) (bool, error)
	SetKeyResultValue(ctx context.Context, id string, v float64) error
	InsertCheckin(ctx context.Context, c Checkin) error
	ListCheckins(ctx context.Context, keyResultID string) ([]Checkin, error)
	// LatestConfidence maps each key result of the period to its last check-in confidence.
	LatestConfidence(ctx context.Context, period string) (map[string]int, error)
}
