package postgres

import (
	"context"
	"encoding/json"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

func toProjectTag(r sqlc.FinanceProjectTag) domain.ProjectTag {
	return domain.ProjectTag{ID: r.ID, Name: r.Name, ClientName: r.ClientName, DealID: str(r.DealID), IsActive: r.IsActive, Notes: r.Notes, UpdatedBy: str(r.UpdatedBy)}
}

func (s *Store) ListProjectTags(ctx context.Context, activeOnly bool) ([]domain.ProjectTag, error) {
	rows, err := s.q.ListProjectTags(ctx, activeOnly)
	return mapAll(rows, toProjectTag), err
}

func (s *Store) GetProjectTag(ctx context.Context, id string) (*domain.ProjectTag, error) {
	r, err := s.q.GetProjectTag(ctx, id)
	return one(r, err, toProjectTag)
}

func (s *Store) SaveProjectTag(ctx context.Context, t domain.ProjectTag) error {
	return s.q.UpsertProjectTag(ctx, sqlc.UpsertProjectTagParams{ID: t.ID, Name: t.Name, ClientName: t.ClientName, IsActive: t.IsActive, Notes: t.Notes, By: sp(t.UpdatedBy)})
}

func (s *Store) EnsureDealTag(ctx context.Context, t domain.ProjectTag) (string, error) {
	return s.q.EnsureDealTag(ctx, sqlc.EnsureDealTagParams{ID: t.ID, Name: t.Name, ClientName: t.ClientName, DealID: sp(t.DealID)})
}

func (s *Store) ProjectTagInUse(ctx context.Context, id string) (bool, error) {
	used, err := s.q.ProjectTagInUse(ctx, id)
	return used != nil && *used, err
}

func (s *Store) DeleteProjectTag(ctx context.Context, id string) error {
	return s.q.DeleteProjectTag(ctx, id)
}

func (s *Store) ProjectProfit(ctx context.Context, from, to, clientName string) ([]domain.ProjectProfitRow, error) {
	rows, err := s.q.ProjectProfit(ctx, sqlc.ProjectProfitParams{DateFrom: day(from), DateTo: day(to), ClientName: sp(clientName)})
	return mapAll(rows, func(r sqlc.ProjectProfitRow) domain.ProjectProfitRow {
		return domain.ProjectProfitRow{TagID: r.TagID, TagName: r.TagName, ClientName: r.ClientName, Revenue: r.Revenue, Cost: r.Cost}
	}), err
}

func toRecurringInvoice(r sqlc.FinanceRecurringInvoice) domain.RecurringInvoice {
	out := domain.RecurringInvoice{ID: r.ID, Kind: r.Kind, Name: r.Name, PartyName: r.PartyName, ClientID: r.ClientID, AccountID: str(r.AccountID), TagID: str(r.TagID), Description: r.Description,
		Subtotal: r.Subtotal, TaxAmount: r.TaxAmount, DueDays: int(r.DueDays), AnchorDay: int(r.AnchorDay), Frequency: r.Frequency, NextRun: ds(r.NextRun), EndDate: dsp(r.EndDate),
		LastInvoiceID: str(r.LastInvoiceID), LastError: r.LastError, AutoPost: r.AutoPost, IsActive: r.IsActive, CreatedBy: str(r.CreatedBy)}
	_ = json.Unmarshal(r.TaxIds, &out.TaxIDs) // Written by SaveRecurringInvoice only.
	return out
}

func (s *Store) ListRecurringInvoices(ctx context.Context) ([]domain.RecurringInvoice, error) {
	rows, err := s.q.ListRecurringInvoices(ctx)
	return mapAll(rows, toRecurringInvoice), err
}

func (s *Store) DueRecurringInvoices(ctx context.Context, today string) ([]domain.RecurringInvoice, error) {
	rows, err := s.q.DueRecurringInvoices(ctx, day(today))
	return mapAll(rows, toRecurringInvoice), err
}

func (s *Store) GetRecurringInvoice(ctx context.Context, id string) (*domain.RecurringInvoice, error) {
	r, err := s.q.GetRecurringInvoice(ctx, id)
	return one(r, err, toRecurringInvoice)
}

func (s *Store) SaveRecurringInvoice(ctx context.Context, r domain.RecurringInvoice, by string) error {
	taxIDs, err := json.Marshal(append([]string{}, r.TaxIDs...))
	if err != nil {
		return err
	}
	return s.q.UpsertRecurringInvoice(ctx, sqlc.UpsertRecurringInvoiceParams{ID: r.ID, Kind: r.Kind, Name: r.Name, PartyName: r.PartyName, ClientID: r.ClientID, AccountID: sp(r.AccountID), TagID: sp(r.TagID),
		Description: r.Description, Subtotal: r.Subtotal, TaxAmount: r.TaxAmount, TaxIds: taxIDs, DueDays: int32(r.DueDays), Frequency: r.Frequency, AnchorDay: int32(r.AnchorDay), NextRun: day(r.NextRun),
		EndDate: dayp(r.EndDate), AutoPost: r.AutoPost, IsActive: r.IsActive, LastError: r.LastError, By: sp(by)})
}

func (s *Store) UpdateRecurringInvoiceRun(ctx context.Context, r domain.RecurringInvoice) error {
	return s.q.UpdateRecurringInvoiceRun(ctx, sqlc.UpdateRecurringInvoiceRunParams{ID: r.ID, NextRun: day(r.NextRun), IsActive: r.IsActive, LastInvoiceID: sp(r.LastInvoiceID), LastError: r.LastError})
}

func (s *Store) SetRecurringInvoiceActive(ctx context.Context, id string, active bool, by string) error {
	return s.q.SetRecurringInvoiceActive(ctx, sqlc.SetRecurringInvoiceActiveParams{ID: id, IsActive: active, By: sp(by)})
}

func (s *Store) DeleteRecurringInvoice(ctx context.Context, id string) error {
	return s.q.DeleteRecurringInvoice(ctx, id)
}

func toAdvance(r sqlc.ListAdvancesRow) domain.Advance {
	return domain.Advance{ID: r.ID, Kind: r.Kind, AdvanceNo: r.AdvanceNo, Date: ds(r.AdvanceDate), PartyName: r.PartyName, Amount: r.Amount, Applied: r.Applied, Refunded: r.Refunded,
		CashAccountID: str(r.CashAccountID), CashAccountName: r.CashAccountName, AdvanceAccountID: r.AdvanceAccountID, AdvanceAccountName: r.AdvanceAccountName,
		Reference: r.Reference, Notes: r.Notes, Status: r.Status, JournalEntryID: str(r.JournalEntryID), VoidReason: r.VoidReason, CreatedBy: str(r.CreatedBy), SourcePaymentID: str(r.SourcePaymentID)}
}

func (s *Store) ListAdvances(ctx context.Context, kind string) ([]domain.Advance, error) {
	rows, err := s.q.ListAdvances(ctx, sqlc.ListAdvancesParams{Kind: sp(kind)})
	if err != nil {
		return nil, err
	}
	return s.withRefunds(ctx, mapAll(rows, toAdvance))
}

func (s *Store) GetAdvance(ctx context.Context, id string) (*domain.Advance, error) {
	return s.oneAdvance(ctx, sqlc.ListAdvancesParams{ID: sp(id)})
}

func (s *Store) AdvanceBySourcePayment(ctx context.Context, paymentID string) (*domain.Advance, error) {
	return s.oneAdvance(ctx, sqlc.ListAdvancesParams{SourcePaymentID: sp(paymentID)})
}

func (s *Store) AdvanceBySalesPayment(ctx context.Context, salesPaymentID string) (*domain.Advance, error) {
	return s.oneAdvance(ctx, sqlc.ListAdvancesParams{SalesPaymentID: sp(salesPaymentID)})
}

func (s *Store) InvoiceCredited(ctx context.Context, invoiceID string) (domain.Money, error) {
	return s.q.InvoiceCredited(ctx, invoiceID)
}

func (s *Store) oneAdvance(ctx context.Context, params sqlc.ListAdvancesParams) (*domain.Advance, error) {
	rows, err := s.q.ListAdvances(ctx, params)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	out, err := s.withRefunds(ctx, []domain.Advance{toAdvance(rows[0])})
	if err != nil {
		return nil, err
	}
	return &out[0], nil
}

func (s *Store) LockAdvance(ctx context.Context, id string) error {
	return s.q.LockAdvance(ctx, id)
}

func (s *Store) InsertAdvance(ctx context.Context, a domain.Advance) error {
	return s.q.InsertAdvance(ctx, sqlc.InsertAdvanceParams{ID: a.ID, Kind: a.Kind, AdvanceNo: a.AdvanceNo, AdvanceDate: day(a.Date), PartyName: a.PartyName, Amount: a.Amount, CashAccountID: sp(a.CashAccountID),
		AdvanceAccountID: a.AdvanceAccountID, Reference: a.Reference, Notes: a.Notes, Status: a.Status, JournalEntryID: sp(a.JournalEntryID), By: sp(a.CreatedBy), SourcePaymentID: sp(a.SourcePaymentID), SalesPaymentID: sp(a.SalesPaymentID)})
}

func (s *Store) VoidAdvance(ctx context.Context, id, reason string) error {
	return s.q.VoidAdvance(ctx, sqlc.VoidAdvanceParams{ID: id, VoidReason: reason})
}

func (s *Store) InsertAdvanceApplication(ctx context.Context, paymentID, advanceID string) error {
	return s.q.InsertAdvanceApplication(ctx, sqlc.InsertAdvanceApplicationParams{PaymentID: paymentID, AdvanceID: advanceID})
}

func (s *Store) DeletePostingLog(ctx context.Context, eventCode, documentID string) error {
	return s.q.DeletePostingLog(ctx, sqlc.DeletePostingLogParams{EventCode: eventCode, DocumentID: documentID})
}

func (s *Store) PendingPostings(ctx context.Context, from, to string) (int, error) {
	n, err := s.q.PendingPostings(ctx, sqlc.PendingPostingsParams{DateFrom: day(from), DateTo: day(to)})
	return int(n), err
}

func (s *Store) SubledgerTotals(ctx context.Context) ([]domain.SubledgerTotal, error) {
	rows, err := s.q.SubledgerTotals(ctx)
	return mapAll(rows, func(r sqlc.SubledgerTotalsRow) domain.SubledgerTotal {
		return domain.SubledgerTotal{Kind: r.Kind, Outstanding: r.Outstanding, UnjournaledCount: int(r.UnjournaledCount), UnjournaledAmount: r.UnjournaledAmount}
	}), err
}

func (s *Store) ControlAdjustments(ctx context.Context, accountIDs []string) (domain.Money, domain.Money, error) {
	r, err := s.q.ControlAdjustments(ctx, accountIDs)
	return r.Opening, r.Recon, err
}

func (s *Store) InsertPaymentAdjustments(ctx context.Context, paymentID string, adjustments []domain.PaymentAdjustment) error {
	for i, a := range adjustments {
		if err := s.q.InsertPaymentAdjustment(ctx, sqlc.InsertPaymentAdjustmentParams{ID: ids.New(), PaymentID: paymentID, Kind: a.Kind, AccountID: a.AccountID, TaxCode: a.TaxCode, TaxName: a.TaxName,
			TaxRate: a.TaxRate, ReportCode: a.ReportCode, TaxBase: a.TaxBase, Amount: a.Amount, Memo: a.Memo, SortOrder: int32(i)}); err != nil {
			return wrap(err)
		}
	}
	return nil
}

func toAdvanceRefund(r sqlc.ListAdvanceRefundsRow) domain.AdvanceRefund {
	return domain.AdvanceRefund{ID: r.ID, AdvanceID: r.AdvanceID, RefundNo: r.RefundNo, Date: ds(r.RefundDate), Amount: r.Amount, CashAccountID: r.CashAccountID, CashAccountName: r.CashAccountName,
		Reason: r.Reason, Status: r.Status, JournalEntryID: r.JournalEntryID, VoidReason: r.VoidReason, CreatedBy: str(r.CreatedBy)}
}

func (s *Store) InsertAdvanceRefund(ctx context.Context, r domain.AdvanceRefund) error {
	return s.q.InsertAdvanceRefund(ctx, sqlc.InsertAdvanceRefundParams{ID: r.ID, AdvanceID: r.AdvanceID, RefundNo: r.RefundNo, RefundDate: day(r.Date), Amount: r.Amount, CashAccountID: r.CashAccountID,
		Reason: r.Reason, Status: r.Status, JournalEntryID: r.JournalEntryID, By: sp(r.CreatedBy)})
}

func (s *Store) GetAdvanceRefund(ctx context.Context, id string) (*domain.AdvanceRefund, error) {
	rows, err := s.q.ListAdvanceRefunds(ctx, sqlc.ListAdvanceRefundsParams{ID: sp(id)})
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	r := toAdvanceRefund(rows[0])
	return &r, nil
}

func (s *Store) VoidAdvanceRefund(ctx context.Context, id, reason string) error {
	return s.q.VoidAdvanceRefund(ctx, sqlc.VoidAdvanceRefundParams{ID: id, VoidReason: reason})
}

func (s *Store) withRefunds(ctx context.Context, out []domain.Advance) ([]domain.Advance, error) {
	if len(out) == 0 {
		return out, nil
	}
	idx := map[string]int{}
	advanceIDs := make([]string, len(out))
	for i, a := range out {
		idx[a.ID], advanceIDs[i] = i, a.ID
	}
	rows, err := s.q.ListAdvanceRefunds(ctx, sqlc.ListAdvanceRefundsParams{AdvanceIds: advanceIDs})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		a := &out[idx[r.AdvanceID]]
		a.Refunds = append(a.Refunds, toAdvanceRefund(r))
	}
	return out, nil
}
