package postgres

import (
	"context"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

func toDeferral(r sqlc.ListDeferralsRow) domain.Deferral {
	return domain.Deferral{ID: r.ID, Kind: r.Kind, Name: r.Name, PartyName: r.PartyName, Reference: r.Reference, DeferredAccountID: r.DeferredAccountID, DeferredAccountName: r.DeferredAccountName,
		TargetAccountID: r.TargetAccountID, TargetAccountName: r.TargetAccountName, TagID: str(r.TagID), StartDate: ds(r.StartDate), Months: int(r.Months), PostedMonths: int(r.PostedMonths), Amount: r.Amount,
		IsActive: r.IsActive, LastEntryID: str(r.LastEntryID), LastError: r.LastError, CreatedBy: str(r.CreatedBy)}
}

func (s *Store) ListDeferrals(ctx context.Context, activeOnly bool) ([]domain.Deferral, error) {
	rows, err := s.q.ListDeferrals(ctx, sqlc.ListDeferralsParams{ActiveOnly: activeOnly})
	return mapAll(rows, toDeferral), err
}

func (s *Store) GetDeferral(ctx context.Context, id string) (*domain.Deferral, error) {
	rows, err := s.q.ListDeferrals(ctx, sqlc.ListDeferralsParams{ID: sp(id)})
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	d := toDeferral(rows[0])
	return &d, nil
}

func (s *Store) InsertDeferral(ctx context.Context, d domain.Deferral) error {
	return wrap(s.q.InsertDeferral(ctx, sqlc.InsertDeferralParams{ID: d.ID, Kind: d.Kind, Name: d.Name, PartyName: d.PartyName, Reference: d.Reference, DeferredAccountID: d.DeferredAccountID, TargetAccountID: d.TargetAccountID,
		TagID: sp(d.TagID), StartDate: day(d.StartDate), Months: int32(d.Months), Amount: d.Amount, By: sp(d.CreatedBy)}))
}

func (s *Store) UpdateDeferralRun(ctx context.Context, d domain.Deferral) error {
	return s.q.UpdateDeferralRun(ctx, sqlc.UpdateDeferralRunParams{ID: d.ID, PostedMonths: int32(d.PostedMonths), LastEntryID: sp(d.LastEntryID), LastError: d.LastError, IsActive: d.IsActive})
}

func (s *Store) SetDeferralActive(ctx context.Context, id string, active bool) error {
	return s.q.SetDeferralActive(ctx, sqlc.SetDeferralActiveParams{ID: id, IsActive: active})
}

func (s *Store) DeleteDeferral(ctx context.Context, id string) error {
	return s.q.DeleteDeferral(ctx, id)
}

func toTaxRemittance(r sqlc.ListTaxRemittancesRow) domain.TaxRemittance {
	return domain.TaxRemittance{ID: r.ID, RemitNo: r.RemitNo, Period: r.Period, Date: ds(r.RemitDate), AccountID: r.AccountID, AccountName: r.AccountName, CashAccountID: r.CashAccountID, CashAccountName: r.CashAccountName,
		Amount: r.Amount, NTPN: r.Ntpn, Notes: r.Notes, Status: r.Status, JournalEntryID: r.JournalEntryID, VoidReason: r.VoidReason, CreatedBy: str(r.CreatedBy)}
}

func (s *Store) ListTaxRemittances(ctx context.Context) ([]domain.TaxRemittance, error) {
	rows, err := s.q.ListTaxRemittances(ctx, nil)
	return mapAll(rows, toTaxRemittance), err
}

func (s *Store) GetTaxRemittance(ctx context.Context, id string) (*domain.TaxRemittance, error) {
	rows, err := s.q.ListTaxRemittances(ctx, sp(id))
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	r := toTaxRemittance(rows[0])
	return &r, nil
}

func (s *Store) InsertTaxRemittance(ctx context.Context, r domain.TaxRemittance) error {
	return wrap(s.q.InsertTaxRemittance(ctx, sqlc.InsertTaxRemittanceParams{ID: r.ID, RemitNo: r.RemitNo, Period: r.Period, RemitDate: day(r.Date), AccountID: r.AccountID, CashAccountID: r.CashAccountID, Amount: r.Amount,
		Ntpn: r.NTPN, Notes: r.Notes, Status: r.Status, JournalEntryID: r.JournalEntryID, By: sp(r.CreatedBy)}))
}

func (s *Store) VoidTaxRemittance(ctx context.Context, id, reason string) error {
	return s.q.VoidTaxRemittance(ctx, sqlc.VoidTaxRemittanceParams{ID: id, VoidReason: reason})
}

func (s *Store) SeparateApprover(ctx context.Context) (bool, error) {
	return s.q.SeparateApprover(ctx)
}

func (s *Store) SetSeparateApprover(ctx context.Context, on bool, by string) error {
	return s.q.SetSeparateApprover(ctx, sqlc.SetSeparateApproverParams{SeparateApprover: on, By: sp(by)})
}

func (s *Store) InsertAuditEntry(ctx context.Context, e domain.AuditEntry) error {
	return s.q.InsertAuditEntry(ctx, sqlc.InsertAuditEntryParams{ID: ids.New(), ActorID: e.ActorID, ActorName: e.ActorName, Action: e.Action, DocID: e.DocID, DocNo: e.DocNo, Result: e.Result, Message: e.Message, Request: e.Request, Response: e.Response})
}

func (s *Store) ListAuditEntries(ctx context.Context, f domain.AuditFilter) ([]domain.AuditEntry, int, error) {
	total, err := s.q.CountAuditEntries(ctx, sqlc.CountAuditEntriesParams{Search: sp(f.Search), Result: sp(f.Result), DateFrom: dayp(f.From), DateTo: dayp(f.To)})
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.q.ListAuditEntries(ctx, sqlc.ListAuditEntriesParams{Search: sp(f.Search), Result: sp(f.Result), DateFrom: dayp(f.From), DateTo: dayp(f.To), Lim: int32(f.Limit), Off: int32(f.Offset)})
	return mapAll(rows, func(r sqlc.FinanceAuditTrail) domain.AuditEntry {
		return domain.AuditEntry{ID: r.ID, At: r.At.Format(time.RFC3339), ActorID: r.ActorID, ActorName: r.ActorName, Action: r.Action, DocID: r.DocID, DocNo: r.DocNo, Result: r.Result, Message: r.Message, Request: r.Request, Response: r.Response}
	}), int(total), err
}
