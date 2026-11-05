package app

import (
	"context"
	"fmt"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// RecurringInvoiceInput is the schedule form.
type RecurringInvoiceInput struct {
	Kind, Name, PartyName, ClientID, AccountID, TagID, Description string
	Frequency, StartDate, EndDate                                  string
	Subtotal, TaxAmount                                            domain.Money
	TaxIDs                                                         []string
	DueDays                                                        int
	AutoPost                                                       bool
}

func (s *Schedules) ListRecurringInvoices(ctx context.Context) ([]domain.RecurringInvoice, error) {
	return s.Store.ListRecurringInvoices(ctx)
}

// fill validates the input with the same rules a real invoice meets, so a
// schedule that saves can issue, and copies it onto r.
func (s *Schedules) fillRecurringInvoice(ctx context.Context, r *domain.RecurringInvoice, in RecurringInvoiceInput) error {
	if in.Kind != domain.KindAP && in.Kind != domain.KindAR {
		return apperr.Field("kind", "Jenis tagihan tidak valid")
	}
	if in.Name = trim(in.Name); in.Name == "" {
		return apperr.Field("name", "Nama jadwal wajib diisi")
	}
	if in.Frequency != domain.FreqMonthly && in.Frequency != domain.FreqQuarterly && in.Frequency != domain.FreqYearly {
		return apperr.Field("frequency", "Frekuensi tidak valid")
	}
	if in.DueDays < 0 || in.DueDays > 365 {
		return apperr.Field("due_days", "Jatuh tempo 0 sampai 365 hari")
	}
	start, err := parseDay("start_date", in.StartDate)
	if err != nil {
		return err
	}
	end, err := optionalDay("end_date", in.EndDate)
	if err != nil {
		return err
	}
	if end != "" && end < start {
		return apperr.Field("end_date", "Tanggal selesai harus setelah tanggal mulai")
	}
	l := NewLedgers(s.Deps)
	probe := InvoiceInput{InvoiceDate: start, PartyName: in.PartyName, AccountID: in.AccountID, TagID: in.TagID, Subtotal: in.Subtotal, TaxAmount: in.TaxAmount, TaxIDs: in.TaxIDs}
	if err := l.normalizeInvoice(ctx, &probe); err != nil {
		return err
	}
	if _, err := l.applyInvoiceTaxes(ctx, in.Kind, &probe); err != nil {
		return err
	}
	first, _ := time.Parse(time.DateOnly, start)
	r.Kind, r.Name, r.PartyName, r.ClientID, r.AccountID, r.TagID, r.Description = in.Kind, in.Name, trim(in.PartyName), in.ClientID, in.AccountID, in.TagID, trim(in.Description)
	r.Subtotal, r.TaxAmount, r.TaxIDs, r.DueDays, r.AutoPost = in.Subtotal, in.TaxAmount, in.TaxIDs, in.DueDays, in.AutoPost
	r.Frequency, r.AnchorDay, r.NextRun, r.EndDate = in.Frequency, first.Day(), start, end
	return nil
}

func (s *Schedules) CreateRecurringInvoice(ctx context.Context, userID string, in RecurringInvoiceInput) (*domain.RecurringInvoice, error) {
	r := domain.RecurringInvoice{ID: ids.New(), IsActive: true, CreatedBy: userID}
	if err := s.fillRecurringInvoice(ctx, &r, in); err != nil {
		return nil, err
	}
	if err := s.Store.SaveRecurringInvoice(ctx, r, userID); err != nil {
		return nil, err
	}
	return s.Store.GetRecurringInvoice(ctx, r.ID)
}

// UpdateRecurringInvoice changes future invoices only. A schedule that has run
// cannot move backward, because that would bill a period twice.
func (s *Schedules) UpdateRecurringInvoice(ctx context.Context, userID, id string, in RecurringInvoiceInput) (*domain.RecurringInvoice, error) {
	r, err := s.Store.GetRecurringInvoice(ctx, id)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, domain.NotFound("Tagihan berulang tidak ditemukan")
	}
	previous, ran := r.NextRun, r.LastInvoiceID != ""
	if err := s.fillRecurringInvoice(ctx, r, in); err != nil {
		return nil, err
	}
	if ran && r.NextRun < previous {
		return nil, apperr.Field("start_date", "Jadwal ini sudah berjalan, jadwal berikutnya tidak bisa mundur")
	}
	r.LastError = ""
	if err := s.Store.SaveRecurringInvoice(ctx, *r, userID); err != nil {
		return nil, err
	}
	return s.Store.GetRecurringInvoice(ctx, id)
}

func (s *Schedules) SetRecurringInvoiceActive(ctx context.Context, userID, id string, active bool) (*domain.RecurringInvoice, error) {
	if err := s.Store.SetRecurringInvoiceActive(ctx, id, active, userID); err != nil {
		return nil, err
	}
	r, err := s.Store.GetRecurringInvoice(ctx, id)
	if err == nil && r == nil {
		err = domain.NotFound("Tagihan berulang tidak ditemukan")
	}
	return r, err
}

// DeleteRecurringInvoice stops the schedule; invoices already issued stay.
func (s *Schedules) DeleteRecurringInvoice(ctx context.Context, id string) error {
	return s.Store.DeleteRecurringInvoice(ctx, id)
}

// runRecurringInvoice issues everything due. Each invoice, its journal and the
// schedule update share a transaction, so a crash cannot bill a period twice. A
// journal that cannot post rolls the invoice back and leaves the date due, so it
// catches up once the period opens.
func (s *Schedules) runRecurringInvoice(ctx context.Context, r domain.RecurringInvoice, today string, res *RunResult) {
	for range maxCatchUp {
		if r.NextRun > today || !r.IsActive {
			return
		}
		runDay := r.NextRun
		err := s.Store.WithTx(ctx, func(tx domain.Store) error {
			d := s.Deps
			d.Store = tx
			l := NewLedgers(d)
			day, err := time.Parse(time.DateOnly, runDay)
			if err != nil {
				return err
			}
			inv, err := l.CreateInvoice(ctx, r.Kind, r.CreatedBy, InvoiceInput{InvoiceDate: runDay, DueDate: day.AddDate(0, 0, r.DueDays).Format(time.DateOnly),
				PartyName: r.PartyName, Reference: fmt.Sprintf("%s %s", r.Name, runDay[:7]), AccountID: r.AccountID, Description: r.Description, ClientID: r.ClientID,
				TagID: r.TagID, Subtotal: r.Subtotal, TaxAmount: r.TaxAmount, TaxIDs: r.TaxIDs})
			if err != nil {
				return err
			}
			if r.AutoPost {
				if _, _, err := l.PostInvoice(ctx, r.Kind, r.CreatedBy, inv.ID); err != nil {
					return err
				}
			}
			next, err := domain.NextRunAfter(runDay, r.Frequency, r.AnchorDay)
			if err != nil {
				return err
			}
			r.NextRun, r.LastInvoiceID, r.LastError = next, inv.ID, ""
			r.IsActive = r.EndDate == "" || next <= r.EndDate
			return tx.UpdateRecurringInvoiceRun(ctx, r)
		})
		if err != nil {
			res.Failed++
			r.NextRun, r.LastError = runDay, apperr.As(err).Message
			s.recordRunError(ctx, s.Store.UpdateRecurringInvoiceRun(ctx, r), "recurring invoice", r.ID)
			return
		}
		res.Posted++
	}
}
