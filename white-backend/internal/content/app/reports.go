package app

import (
	"context"

	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Reports builds the monthly content report per client.
type Reports struct{ Deps }

func NewReports(d Deps) *Reports { return &Reports{d} }

// engagement reads metrics for the month's published items (analytics
// first, the stored snapshot otherwise).
func (r *Reports) engagement(ctx context.Context, clientID string, items []domain.Item) (map[string]domain.PostMetrics, error) {
	var published []publishedRef
	for _, it := range items {
		if it.Status == domain.ItemPublished {
			published = append(published, publishedRef{ID: it.ID, ExternalIDs: it.ExternalIDs})
		}
	}
	return r.Deps.metrics(ctx, clientID, published)
}

// Generate computes the month and writes (or overwrites) its report.
func (r *Reports) Generate(ctx context.Context, userID, clientID, month string) (*domain.Report, error) {
	m, ok := domain.ParseMonth(month)
	if !ok {
		return nil, apperr.Field("month", msgMonthInvalid)
	}
	a, err := r.manage(ctx, userID, clientID)
	if err != nil {
		return nil, err
	}
	items, err := r.Store.ListItems(ctx, clientID, domain.ItemFilter{Month: m})
	if err != nil {
		return nil, err
	}
	name := a.Name
	if name == "" {
		name = a.Slug
	}
	metrics, err := r.engagement(ctx, clientID, items)
	if err != nil {
		return nil, err
	}
	data := domain.ComputeReport(m, name, items, metrics, r.now())
	summary, model, err := r.AI.SummarizeReport(ctx, data)
	if err != nil {
		return nil, err
	}
	return r.Store.UpsertReport(ctx, domain.Report{ID: ids.New(), ClientID: clientID, Month: m, Data: data, Summary: summary, AIModel: model, CreatedBy: &userID})
}

func (r *Reports) List(ctx context.Context, userID, clientID string) ([]domain.Report, error) {
	if _, err := r.access(ctx, userID, clientID); err != nil {
		return nil, err
	}
	return r.Store.ListReports(ctx, clientID)
}

func (r *Reports) Get(ctx context.Context, userID, clientID, reportID string) (*domain.Report, error) {
	if _, err := r.access(ctx, userID, clientID); err != nil {
		return nil, err
	}
	return ensureFound(r.Store.GetReport(ctx, clientID, reportID))(msgReportNotFound)
}

func (r *Reports) PDF(ctx context.Context, userID, clientID, reportID string) (*File, error) {
	rep, err := r.Get(ctx, userID, clientID, reportID)
	if err != nil {
		return nil, err
	}
	data, err := r.Renderer.Render(*rep)
	if err != nil {
		return nil, err
	}
	return &File{Name: "laporan-konten-" + rep.Month.String() + ".pdf", ContentType: "application/pdf", Data: data}, nil
}
