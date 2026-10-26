package postgres

import (
	"context"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
)

// dashArgs are the shared optional filters of the dashboard queries.
type dashArgs struct{ pipeline, owner, line, source *string }

func dashFilter(f domain.DashFilter) dashArgs {
	return dashArgs{nz(f.PipelineID), nz(f.OwnerUserID), nz(f.ServiceLine), nz(f.Source)}
}

func (s *Store) DashClosedMonthly(ctx context.Context, from, to time.Time, f domain.DashFilter) ([]domain.DashClosedMonth, error) {
	a := dashFilter(f)
	rows, err := s.q.DashClosedMonthly(ctx, sqlc.DashClosedMonthlyParams{FromAt: from, ToAt: to, Pipeline: a.pipeline, Owner: a.owner, Line: a.line, Source: a.source})
	return many(rows, err, func(r sqlc.DashClosedMonthlyRow) domain.DashClosedMonth { return domain.DashClosedMonth(r) })
}

func (s *Store) DashWonByOwnerMonthly(ctx context.Context, from, to time.Time, f domain.DashFilter) ([]domain.DashOwnerMonth, error) {
	a := dashFilter(f)
	rows, err := s.q.DashWonByOwnerMonthly(ctx, sqlc.DashWonByOwnerMonthlyParams{FromAt: from, ToAt: to, Pipeline: a.pipeline, Owner: a.owner, Line: a.line, Source: a.source})
	return many(rows, err, func(r sqlc.DashWonByOwnerMonthlyRow) domain.DashOwnerMonth { return domain.DashOwnerMonth(r) })
}

func (s *Store) DashOpenAsOf(ctx context.Context, at time.Time, f domain.DashFilter) (domain.DashOpenSnapshot, error) {
	a := dashFilter(f)
	r, err := s.q.DashOpenAsOf(ctx, sqlc.DashOpenAsOfParams{At: at, Pipeline: a.pipeline, Owner: a.owner, Line: a.line, Source: a.source})
	return domain.DashOpenSnapshot(r), err
}

func (s *Store) DashOpenDeals(ctx context.Context, f domain.DashFilter) ([]domain.DashOpenDeal, error) {
	a := dashFilter(f)
	rows, err := s.q.DashOpenDeals(ctx, sqlc.DashOpenDealsParams{Pipeline: a.pipeline, Owner: a.owner, Line: a.line, Source: a.source})
	return many(rows, err, func(r sqlc.DashOpenDealsRow) domain.DashOpenDeal { return domain.DashOpenDeal(r) })
}

func (s *Store) DashFunnel(ctx context.Context, pipelineID string, from, to time.Time, f domain.DashFilter) ([]domain.DashStageCount, error) {
	a := dashFilter(f)
	rows, err := s.q.DashFunnel(ctx, sqlc.DashFunnelParams{PipelineID: pipelineID, FromAt: from, ToAt: to, Owner: a.owner, Line: a.line, Source: a.source})
	return many(rows, err, func(r sqlc.DashFunnelRow) domain.DashStageCount { return domain.DashStageCount(r) })
}

func (s *Store) DashStageDays(ctx context.Context, from, to time.Time, f domain.DashFilter) ([]domain.DashStageDay, error) {
	a := dashFilter(f)
	rows, err := s.q.DashStageDays(ctx, sqlc.DashStageDaysParams{FromAt: from, ToAt: to, Pipeline: a.pipeline, Owner: a.owner, Line: a.line, Source: a.source})
	return many(rows, err, func(r sqlc.DashStageDaysRow) domain.DashStageDay { return domain.DashStageDay(r) })
}

func (s *Store) DashClosedBreakdown(ctx context.Context, from, to time.Time, f domain.DashFilter) ([]domain.DashBreakdown, error) {
	a := dashFilter(f)
	rows, err := s.q.DashClosedBreakdown(ctx, sqlc.DashClosedBreakdownParams{FromAt: from, ToAt: to, Pipeline: a.pipeline, Owner: a.owner, Line: a.line, Source: a.source})
	return many(rows, err, func(r sqlc.DashClosedBreakdownRow) domain.DashBreakdown { return domain.DashBreakdown(r) })
}

func (s *Store) DashQuotes(ctx context.Context, from, to time.Time, f domain.DashFilter) ([]domain.DashQuoteStatus, error) {
	a := dashFilter(f)
	rows, err := s.q.DashQuotes(ctx, sqlc.DashQuotesParams{FromAt: from, ToAt: to, Pipeline: a.pipeline, Owner: a.owner, Line: a.line, Source: a.source})
	return many(rows, err, func(r sqlc.DashQuotesRow) domain.DashQuoteStatus { return domain.DashQuoteStatus(r) })
}

func (s *Store) DashOwners(ctx context.Context, from, to, prevFrom, prevTo time.Time, f domain.DashFilter) ([]domain.DashOwnerRow, error) {
	a := dashFilter(f)
	rows, err := s.q.DashOwners(ctx, sqlc.DashOwnersParams{FromAt: from, ToAt: to, PrevFromAt: prevFrom, PrevToAt: prevTo, Pipeline: a.pipeline, Owner: a.owner, Line: a.line, Source: a.source})
	return many(rows, err, func(r sqlc.DashOwnersRow) domain.DashOwnerRow { return domain.DashOwnerRow(r) })
}

func (s *Store) DashTargets(ctx context.Context, fromMonth, toMonth string) ([]domain.DashTarget, error) {
	rows, err := s.q.DashTargets(ctx, sqlc.DashTargetsParams{FromMonth: fromMonth, ToMonth: toMonth})
	return many(rows, err, func(r sqlc.DashTargetsRow) domain.DashTarget { return domain.DashTarget(r) })
}

func (s *Store) DashLeadsMonthly(ctx context.Context, from, to time.Time, f domain.DashFilter) ([]domain.DashLeadMonth, error) {
	a := dashFilter(f)
	rows, err := s.q.DashLeadsMonthly(ctx, sqlc.DashLeadsMonthlyParams{FromAt: from, ToAt: to, Owner: a.owner, Source: a.source})
	return many(rows, err, func(r sqlc.DashLeadsMonthlyRow) domain.DashLeadMonth { return domain.DashLeadMonth(r) })
}

func (s *Store) DashLeadSources(ctx context.Context, from, to time.Time, f domain.DashFilter) ([]domain.DashLeadSource, error) {
	a := dashFilter(f)
	rows, err := s.q.DashLeadSources(ctx, sqlc.DashLeadSourcesParams{FromAt: from, ToAt: to, Owner: a.owner, Source: a.source})
	return many(rows, err, func(r sqlc.DashLeadSourcesRow) domain.DashLeadSource { return domain.DashLeadSource(r) })
}

func (s *Store) DashLeadScores(ctx context.Context, from, to time.Time, f domain.DashFilter) ([]domain.DashLeadScore, error) {
	a := dashFilter(f)
	rows, err := s.q.DashLeadScores(ctx, sqlc.DashLeadScoresParams{FromAt: from, ToAt: to, Owner: a.owner, Source: a.source})
	return many(rows, err, func(r sqlc.DashLeadScoresRow) domain.DashLeadScore { return domain.DashLeadScore(r) })
}

func (s *Store) DashConversations(ctx context.Context, from, to time.Time, f domain.DashFilter) ([]domain.DashConversationMonth, error) {
	rows, err := s.q.DashConversations(ctx, sqlc.DashConversationsParams{FromAt: from, ToAt: to, Owner: nz(f.OwnerUserID)})
	return many(rows, err, func(r sqlc.DashConversationsRow) domain.DashConversationMonth { return domain.DashConversationMonth(r) })
}

func (s *Store) DashFormSubmissions(ctx context.Context, from, to time.Time) ([]domain.DashFormCount, error) {
	rows, err := s.q.DashFormSubmissions(ctx, sqlc.DashFormSubmissionsParams{FromAt: from, ToAt: to})
	return many(rows, err, func(r sqlc.DashFormSubmissionsRow) domain.DashFormCount { return domain.DashFormCount(r) })
}

func (s *Store) DashBillingMonthly(ctx context.Context, from, to time.Time, f domain.DashFilter) ([]domain.DashBillingMonth, error) {
	a := dashFilter(f)
	rows, err := s.q.DashBillingMonthly(ctx, sqlc.DashBillingMonthlyParams{FromAt: from, ToAt: to, FromDay: from, ToDay: to, Pipeline: a.pipeline, Owner: a.owner, Line: a.line, Source: a.source})
	return many(rows, err, func(r sqlc.DashBillingMonthlyRow) domain.DashBillingMonth { return domain.DashBillingMonth(r) })
}

func (s *Store) DashReceivables(ctx context.Context, asOf time.Time, f domain.DashFilter) ([]domain.DashReceivable, error) {
	a := dashFilter(f)
	rows, err := s.q.DashReceivables(ctx, sqlc.DashReceivablesParams{AsOf: asOf, Pipeline: a.pipeline, Owner: a.owner, Line: a.line, Source: a.source})
	return many(rows, err, func(r sqlc.DashReceivablesRow) domain.DashReceivable {
		return domain.DashReceivable{ID: r.ID, InvoiceNumber: r.InvoiceNumber, DealID: r.DealID, DealTitle: r.DealTitle, AccountName: r.AccountName, Amount: r.Amount, DueDate: r.DueDate, Paid: r.Paid}
	})
}

func (s *Store) DashTermProgress(ctx context.Context, f domain.DashFilter) ([]domain.DashTermRow, error) {
	a := dashFilter(f)
	rows, err := s.q.DashTermProgress(ctx, sqlc.DashTermProgressParams{Pipeline: a.pipeline, Owner: a.owner, Line: a.line, Source: a.source})
	return many(rows, err, func(r sqlc.DashTermProgressRow) domain.DashTermRow { return domain.DashTermRow(r) })
}
