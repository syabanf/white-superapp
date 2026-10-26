// Package postgres implements the reports ports on top of sqlc queries.
package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/syabanf/white-superapp/white-backend/internal/reports/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/reports/domain"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

type Store struct {
	q *sqlc.Queries
}

func NewStore(pool *pgxpool.Pool) *Store { return &Store{q: sqlc.New(pool)} }

func toReport(r sqlc.ReportsReport) domain.Report {
	return domain.Report{
		ID: r.ID, ClientID: r.ClientID, Title: r.Title, DateFrom: r.DateFrom, DateTo: r.DateTo, Compare: r.Compare,
		Modules: r.Modules, Language: r.Language, AiSummary: r.AiSummary, CreatedByID: r.CreatedByID, CreatedAt: r.CreatedAt,
	}
}

func toInsight(r sqlc.ReportsAiInsight) *domain.Insight {
	return &domain.Insight{
		ID: r.ID, ClientID: r.ClientID, Module: r.Module, DateFrom: r.DateFrom, DateTo: r.DateTo,
		Language: r.Language, Model: r.Model, Content: r.Content, CreatedAt: r.CreatedAt,
	}
}

func (s *Store) ListReports(ctx context.Context, clientID string) ([]domain.Report, error) {
	rows, err := s.q.ListReports(ctx, clientID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Report, 0, len(rows))
	for _, r := range rows {
		out = append(out, toReport(r))
	}
	return out, nil
}

func (s *Store) GetReport(ctx context.Context, id string) (*domain.Report, error) {
	r, err := s.q.GetReport(ctx, id)
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := toReport(r)
	return &out, nil
}

func (s *Store) InsertReport(ctx context.Context, r domain.Report) (*domain.Report, error) {
	row, err := s.q.InsertReport(ctx, sqlc.InsertReportParams{
		ID: r.ID, ClientID: r.ClientID, Title: r.Title, DateFrom: r.DateFrom, DateTo: r.DateTo, Compare: r.Compare,
		Modules: r.Modules, Language: r.Language, AiSummary: r.AiSummary, CreatedByID: r.CreatedByID, CreatedAt: r.CreatedAt,
	})
	if err != nil {
		return nil, err
	}
	out := toReport(row)
	return &out, nil
}

func (s *Store) DeleteReport(ctx context.Context, clientID, id string) (bool, error) {
	n, err := s.q.DeleteReport(ctx, sqlc.DeleteReportParams{ID: id, ClientID: clientID})
	return n > 0, err
}

func (s *Store) InsertInsight(ctx context.Context, in domain.Insight) (*domain.Insight, error) {
	row, err := s.q.InsertInsight(ctx, sqlc.InsertInsightParams{
		ID: in.ID, ClientID: in.ClientID, Module: in.Module, DateFrom: in.DateFrom, DateTo: in.DateTo,
		Language: in.Language, Model: in.Model, Content: in.Content, CreatedAt: in.CreatedAt,
	})
	if err != nil {
		return nil, err
	}
	return toInsight(row), nil
}

func (s *Store) LatestInsight(ctx context.Context, clientID, module string, from, to time.Time) (*domain.Insight, error) {
	row, err := s.q.LatestInsight(ctx, sqlc.LatestInsightParams{ClientID: clientID, Module: module, DateFrom: from, DateTo: to})
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toInsight(row), nil
}

func (s *Store) PurgeClient(ctx context.Context, clientID string) (int64, int64, error) {
	reports, err := s.q.DeleteClientReports(ctx, clientID)
	if err != nil {
		return 0, 0, err
	}
	insights, err := s.q.DeleteClientInsights(ctx, clientID)
	return reports, insights, err
}

func (s *Store) ClearCreator(ctx context.Context, userID string) (int64, error) {
	return s.q.ClearReportCreator(ctx, &userID)
}
