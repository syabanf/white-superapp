// Package app holds the reports use cases: saved reports, AI insights,
// report sections, PDF rendering and the public share dashboard.
package app

import (
	"context"

	"github.com/syabanf/white-superapp/white-backend/internal/reports/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
)

type ClientInfo struct {
	ID           string
	Name         string
	Slug         string
	Currency     string
	ShareModules []string
}

// Access is identity CheckClientAccess. Role is ADMIN, MANAGER or VIEWER.
type Access struct {
	Role      string
	CanManage bool
	Client    ClientInfo
}

type ShareStatus string

const (
	ShareOK       ShareStatus = "OK"
	ShareNotFound ShareStatus = "NOT_FOUND"
	ShareDisabled ShareStatus = "DISABLED"
	ShareLocked   ShareStatus = "LOCKED"
)

type ShareContext struct {
	Status ShareStatus
	// Zero unless the status is OK, DISABLED or LOCKED.
	Client ClientInfo
}

type Identity interface {
	Check(ctx context.Context, userID, clientID string) (*Access, error)
	// UserName reports whether the user exists, with its name.
	UserName(ctx context.Context, userID string) (string, bool, error)
	ShareContext(ctx context.Context, slug, unlockToken string) (*ShareContext, error)
}

type Analytics interface {
	ReportData(ctx context.Context, clientID string, current, previous dates.Range, modules []string, topN int) (*domain.ReportData, error)
	// SuitePropertyID is the client's first SEO property by created_at, "" when none.
	SuitePropertyID(ctx context.Context, clientID string) (string, error)
}

type Publishing interface {
	Summary(ctx context.Context, clientID string, rg dates.Range) (*domain.PublishingSummary, error)
}

type SeoSuite interface {
	Rank(ctx context.Context, propertyID string, current, previous dates.Range) (*domain.RankData, error)
	Backlinks(ctx context.Context, propertyID string, rg dates.Range) (*domain.BacklinkData, error)
}

// InsightGenerator returns the markdown and the model id to store.
type InsightGenerator interface {
	Generate(ctx context.Context, in domain.InsightInput) (string, string, error)
}

// Renderer turns PDF data into bytes.
type Renderer func(domain.PdfData) ([]byte, error)

type Deps struct {
	Store      domain.Store
	Identity   Identity
	Analytics  Analytics
	Publishing Publishing
	SeoSuite   SeoSuite
	Insights   InsightGenerator
	Render     Renderer
	Clock      clock.Clock
}
