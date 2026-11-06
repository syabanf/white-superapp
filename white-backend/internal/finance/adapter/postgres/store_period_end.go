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

func (s *Store) SetPaymentSalesReversal(ctx context.Context, id, reason, failure string) error {
	return s.q.SetPaymentSalesReversal(ctx, sqlc.SetPaymentSalesReversalParams{ID: id, SalesDeletedReason: reason, SalesReversalError: failure})
}

func (s *Store) SetInvoiceSalesReversal(ctx context.Context, id, reason, failure string, settled bool) error {
	return s.q.SetInvoiceSalesReversal(ctx, sqlc.SetInvoiceSalesReversalParams{ID: id, SalesCancelReason: reason, SalesReversalError: failure, SalesCancelSettled: settled})
}

func (s *Store) PendingSalesReversals(ctx context.Context) ([]domain.SalesReversal, error) {
	rows, err := s.q.PendingSalesReversals(ctx)
	return mapAll(rows, func(r sqlc.PendingSalesReversalsRow) domain.SalesReversal {
		return domain.SalesReversal{Kind: r.Kind, ID: r.ID, DocNo: r.DocNo, PartyName: r.PartyName, Amount: r.Amount, Reason: r.Reason, Error: r.Error}
	}), err
}

func (s *Store) DisposeFixedAsset(ctx context.Context, id, date, entryID, by string, proceeds domain.Money) error {
	return s.q.DisposeFixedAsset(ctx, sqlc.DisposeFixedAssetParams{ID: id, DisposedOn: dayp(date), DisposalProceeds: proceeds, DisposalEntryID: sp(entryID), By: sp(by)})
}

func toGoodsReceipt(r sqlc.ListGoodsReceiptsRow) domain.GoodsReceipt {
	return domain.GoodsReceipt{ID: r.ID, ReceiptNo: r.ReceiptNo, DocumentID: r.DocumentID, Date: ds(r.ReceiptDate), Amount: r.Amount, Notes: r.Notes, Status: r.Status, VoidReason: r.VoidReason, CreatedBy: str(r.CreatedBy)}
}

func (s *Store) ListGoodsReceipts(ctx context.Context, documentID string) ([]domain.GoodsReceipt, error) {
	rows, err := s.q.ListGoodsReceipts(ctx, sqlc.ListGoodsReceiptsParams{DocumentID: sp(documentID)})
	return mapAll(rows, toGoodsReceipt), err
}

func (s *Store) GetGoodsReceipt(ctx context.Context, id string) (*domain.GoodsReceipt, error) {
	rows, err := s.q.ListGoodsReceipts(ctx, sqlc.ListGoodsReceiptsParams{ID: sp(id)})
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	r := toGoodsReceipt(rows[0])
	return &r, nil
}

func (s *Store) InsertGoodsReceipt(ctx context.Context, r domain.GoodsReceipt) error {
	return wrap(s.q.InsertGoodsReceipt(ctx, sqlc.InsertGoodsReceiptParams{ID: r.ID, ReceiptNo: r.ReceiptNo, DocumentID: r.DocumentID, ReceiptDate: day(r.Date), Amount: r.Amount, Notes: r.Notes, Status: r.Status, By: sp(r.CreatedBy)}))
}

func (s *Store) VoidGoodsReceipt(ctx context.Context, id, reason string) error {
	return s.q.VoidGoodsReceipt(ctx, sqlc.VoidGoodsReceiptParams{ID: id, VoidReason: reason})
}

// SetInvoicePOLink matches an invoice to a purchase order; an empty document unlinks it.
func (s *Store) SetInvoicePOLink(ctx context.Context, invoiceID, documentID string) error {
	if documentID == "" {
		return s.q.DeleteInvoicePOLink(ctx, invoiceID)
	}
	return s.q.SetInvoicePOLink(ctx, sqlc.SetInvoicePOLinkParams{InvoiceID: invoiceID, DocumentID: documentID})
}

func (s *Store) InvoicePOLink(ctx context.Context, invoiceID string) (string, error) {
	rows, err := s.q.InvoicePOLink(ctx, invoiceID)
	if err != nil || len(rows) == 0 {
		return "", err
	}
	return rows[0], nil
}

func (s *Store) POInvoices(ctx context.Context, documentID string) ([]domain.POInvoice, error) {
	rows, err := s.q.POInvoices(ctx, sp(documentID))
	return mapAll(rows, func(r sqlc.POInvoicesRow) domain.POInvoice {
		return domain.POInvoice{DocumentID: r.DocumentID, InvoiceID: r.ID, InvoiceNo: r.InvoiceNo, PartyName: r.PartyName, Status: r.Status, Subtotal: r.Subtotal}
	}), err
}

func (s *Store) ListProjectAllocations(ctx context.Context, fromMonth, toMonth string) ([]domain.ProjectAllocation, error) {
	rows, err := s.q.ListProjectAllocations(ctx, sqlc.ListProjectAllocationsParams{MonthFrom: fromMonth, MonthTo: toMonth})
	return mapAll(rows, func(r sqlc.ListProjectAllocationsRow) domain.ProjectAllocation {
		return domain.ProjectAllocation{ID: r.ID, TagID: r.TagID, TagName: r.TagName, ClientName: r.ClientName, Month: r.Month, Kind: r.Kind, Amount: r.Amount, Notes: r.Notes, CreatedBy: str(r.CreatedBy)}
	}), err
}

func (s *Store) InsertProjectAllocation(ctx context.Context, a domain.ProjectAllocation) error {
	return wrap(s.q.InsertProjectAllocation(ctx, sqlc.InsertProjectAllocationParams{ID: a.ID, TagID: a.TagID, Month: a.Month, Kind: a.Kind, Amount: a.Amount, Notes: a.Notes, By: sp(a.CreatedBy)}))
}

func (s *Store) DeleteProjectAllocation(ctx context.Context, id string) error {
	return s.q.DeleteProjectAllocation(ctx, id)
}

func (s *Store) InsertInvoiceReminder(ctx context.Context, invoiceID string, stage int) (bool, error) {
	n, err := s.q.InsertInvoiceReminder(ctx, sqlc.InsertInvoiceReminderParams{InvoiceID: invoiceID, Stage: int32(stage)})
	return n > 0, err
}
