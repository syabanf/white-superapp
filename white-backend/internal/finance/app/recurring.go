package app

import (
	"context"
	"fmt"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// Schedules posts recurring journals and fixed asset depreciation. The daily
// tick drives it. A closed period stops one schedule with its error on the row
// and leaves the date due, so it catches up once the period opens.
type Schedules struct{ Deps }

func NewSchedules(d Deps) *Schedules { return &Schedules{d} }

// A schedule left off for years catches up in steps, not all in one tick.
const maxCatchUp = 24

// RecurringInput is a new recurring journal.
type RecurringInput struct {
	TagID                                            string
	Name, Description, Frequency, StartDate, EndDate string
	Lines                                            []domain.Line
}

func (s *Schedules) ListRecurring(ctx context.Context) ([]domain.RecurringJournal, error) {
	return s.Store.ListRecurringJournals(ctx)
}

// checkRecurring validates the input and returns its first run and end date.
func (s *Schedules) checkRecurring(ctx context.Context, in *RecurringInput) (start, end string, err error) {
	in.Name = trim(in.Name)
	v := validate.New().Required("name", in.Name, "Nama wajib diisi")
	if in.Frequency != domain.FreqMonthly && in.Frequency != domain.FreqQuarterly && in.Frequency != domain.FreqYearly {
		v.Add("frequency", "Frekuensi tidak valid")
	}
	if err := v.Err(); err != nil {
		return "", "", err
	}
	if start, err = parseDay("start_date", in.StartDate); err != nil {
		return "", "", err
	}
	if end, err = optionalDay("end_date", in.EndDate); err != nil {
		return "", "", err
	}
	if end != "" && end < start {
		return "", "", apperr.Field("end_date", "Tanggal selesai harus setelah tanggal mulai")
	}
	if err := domain.ValidateBalanced(in.Lines); err != nil {
		return "", "", err
	}
	accountIDs := make([]string, len(in.Lines))
	for i, l := range in.Lines {
		accountIDs[i] = l.AccountID
	}
	if err := checkTag(ctx, s.Store, in.TagID); err != nil {
		return "", "", err
	}
	return start, end, checkAccounts(ctx, s.Store, accountIDs, "Akun COA wajib diisi", "Akun jurnal harus postable (bukan header)")
}

func (s *Schedules) CreateRecurring(ctx context.Context, userID string, in RecurringInput) (*domain.RecurringJournal, error) {
	start, end, err := s.checkRecurring(ctx, &in)
	if err != nil {
		return nil, err
	}
	first, _ := time.Parse(time.DateOnly, start)
	r := domain.RecurringJournal{ID: ids.New(), Name: in.Name, Description: trim(in.Description), Frequency: in.Frequency, AnchorDay: first.Day(),
		NextRun: start, EndDate: end, Lines: in.Lines, IsActive: true, CreatedBy: userID, TagID: in.TagID}
	if err := s.Store.InsertRecurringJournal(ctx, r); err != nil {
		return nil, err
	}
	return s.Store.GetRecurringJournal(ctx, r.ID)
}

// UpdateRecurring changes a schedule from its next run onward. Journals already
// posted stay as they are, and a schedule that has run cannot move backward,
// because that would post a period twice.
func (s *Schedules) UpdateRecurring(ctx context.Context, userID, id string, in RecurringInput) (*domain.RecurringJournal, error) {
	r, err := s.Store.GetRecurringJournal(ctx, id)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, domain.NotFound("Jurnal berulang tidak ditemukan")
	}
	start, end, err := s.checkRecurring(ctx, &in)
	if err != nil {
		return nil, err
	}
	if r.LastEntryID != "" && start < r.NextRun {
		return nil, apperr.Field("start_date", "Jurnal ini sudah berjalan, jadwal berikutnya tidak bisa mundur")
	}
	first, _ := time.Parse(time.DateOnly, start)
	r.Name, r.Description, r.Frequency, r.AnchorDay, r.NextRun, r.EndDate, r.Lines, r.TagID = in.Name, trim(in.Description), in.Frequency, first.Day(), start, end, in.Lines, in.TagID
	if err := s.Store.UpdateRecurringJournal(ctx, *r, userID); err != nil {
		return nil, err
	}
	return s.Store.GetRecurringJournal(ctx, id)
}

func (s *Schedules) SetRecurringActive(ctx context.Context, userID, id string, active bool) (*domain.RecurringJournal, error) {
	r, err := s.Store.GetRecurringJournal(ctx, id)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, domain.NotFound("Jurnal berulang tidak ditemukan")
	}
	if err := s.Store.SetRecurringActive(ctx, id, active, userID); err != nil {
		return nil, err
	}
	return s.Store.GetRecurringJournal(ctx, id)
}

func (s *Schedules) DeleteRecurring(ctx context.Context, id string) error {
	return s.Store.DeleteRecurringJournal(ctx, id)
}

// AssetInput is a new fixed asset. Recording the purchase stays a normal
// journal or payable, so one purchase never gets booked twice.
type AssetInput struct {
	Name, AcquiredOn, ExpenseAccountID, AccumAccountID string
	Cost, Salvage                                      domain.Money
	LifeMonths                                         int
}

func (s *Schedules) ListAssets(ctx context.Context) ([]domain.FixedAsset, error) {
	return s.Store.ListFixedAssets(ctx, false)
}

// checkAsset validates the input and returns its acquisition date.
func (s *Schedules) checkAsset(ctx context.Context, in *AssetInput) (string, error) {
	in.Name = trim(in.Name)
	v := validate.New().Required("name", in.Name, "Nama aset wajib diisi")
	if in.Cost <= 0 {
		v.Add("cost", "Harga perolehan harus lebih dari 0")
	}
	if in.Salvage < 0 || in.Salvage >= in.Cost {
		v.Add("salvage", "Nilai sisa harus di bawah harga perolehan")
	}
	if in.LifeMonths <= 0 || in.LifeMonths > 600 {
		v.Add("life_months", "Umur manfaat 1 sampai 600 bulan")
	}
	if in.ExpenseAccountID == "" {
		v.Add("expense_account_id", "Pilih akun beban penyusutan")
	}
	if in.AccumAccountID == "" {
		v.Add("accum_account_id", "Pilih akun akumulasi penyusutan")
	}
	if err := v.Err(); err != nil {
		return "", err
	}
	day, err := parseDay("acquired_on", in.AcquiredOn)
	if err != nil {
		return "", err
	}
	return day, checkAccounts(ctx, s.Store, []string{in.ExpenseAccountID, in.AccumAccountID}, "Akun COA wajib diisi", "Akun harus postable (bukan header)")
}

func (s *Schedules) CreateAsset(ctx context.Context, userID string, in AssetInput) (*domain.FixedAsset, error) {
	day, err := s.checkAsset(ctx, &in)
	if err != nil {
		return nil, err
	}
	a := domain.FixedAsset{ID: ids.New(), Name: in.Name, AcquiredOn: day, Cost: in.Cost, Salvage: in.Salvage, LifeMonths: in.LifeMonths,
		ExpenseAccountID: in.ExpenseAccountID, AccumAccountID: in.AccumAccountID, IsActive: true, CreatedBy: userID}
	if err := s.Store.InsertFixedAsset(ctx, a); err != nil {
		return nil, err
	}
	return s.Store.GetFixedAsset(ctx, a.ID)
}

// UpdateAsset corrects an asset. Once depreciation has posted, changing the cost,
// salvage, life or date would make posted journals disagree with the schedule.
func (s *Schedules) UpdateAsset(ctx context.Context, userID, id string, in AssetInput) (*domain.FixedAsset, error) {
	a, err := s.Store.GetFixedAsset(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, domain.NotFound("Aset tidak ditemukan")
	}
	day, err := s.checkAsset(ctx, &in)
	if err != nil {
		return nil, err
	}
	if a.DepreciatedMonths > 0 && (day != a.AcquiredOn || in.Cost != a.Cost || in.Salvage != a.Salvage || in.LifeMonths != a.LifeMonths) {
		return nil, domain.Conflict("Aset yang sudah disusutkan hanya bisa diubah nama dan akunnya")
	}
	a.Name, a.AcquiredOn, a.Cost, a.Salvage, a.LifeMonths, a.ExpenseAccountID, a.AccumAccountID = in.Name, day, in.Cost, in.Salvage, in.LifeMonths, in.ExpenseAccountID, in.AccumAccountID
	if err := s.Store.UpdateFixedAsset(ctx, *a, userID); err != nil {
		return nil, err
	}
	return s.Store.GetFixedAsset(ctx, id)
}

func (s *Schedules) SetAssetActive(ctx context.Context, userID, id string, active bool) (*domain.FixedAsset, error) {
	a, err := s.Store.GetFixedAsset(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, domain.NotFound("Aset tidak ditemukan")
	}
	if err := s.Store.SetFixedAssetActive(ctx, id, active, userID); err != nil {
		return nil, err
	}
	return s.Store.GetFixedAsset(ctx, id)
}

// DeleteAsset removes an asset entered by mistake, before any depreciation.
func (s *Schedules) DeleteAsset(ctx context.Context, id string) error {
	a, err := s.Store.GetFixedAsset(ctx, id)
	if err != nil {
		return err
	}
	if a == nil {
		return nil
	}
	if a.DepreciatedMonths > 0 {
		return domain.Conflict("Aset yang sudah disusutkan tidak bisa dihapus. Nonaktifkan saja.")
	}
	return s.Store.DeleteFixedAsset(ctx, id)
}

// RunResult counts one run.
type RunResult struct{ Posted, Failed int }

// RunDue posts everything due by today. Each posting and its schedule update
// share a transaction, so a crash cannot post the same period twice.
func (s *Schedules) RunDue(ctx context.Context) (RunResult, error) {
	var res RunResult
	today := s.today()
	due, err := s.Store.DueRecurringJournals(ctx, today)
	if err != nil {
		return res, err
	}
	for _, r := range due {
		s.runRecurring(ctx, r, today, &res)
	}
	invoices, err := s.Store.DueRecurringInvoices(ctx, today)
	if err != nil {
		return res, err
	}
	for _, r := range invoices {
		s.runRecurringInvoice(ctx, r, today, &res)
	}
	assets, err := s.Store.ListFixedAssets(ctx, true)
	if err != nil {
		return res, err
	}
	for _, a := range assets {
		s.runAsset(ctx, a, today, &res)
	}
	return res, nil
}

func (s *Schedules) runRecurring(ctx context.Context, r domain.RecurringJournal, today string, res *RunResult) {
	for range maxCatchUp {
		if r.NextRun > today || !r.IsActive {
			return
		}
		runDay := r.NextRun
		err := s.Store.WithTx(ctx, func(tx domain.Store) error {
			e, err := createEntry(ctx, tx, NewEntry{TagID: r.TagID, EntryDate: runDay, Description: fmt.Sprintf("%s %s", r.Name, runDay[:7]), Post: true, EntryType: domain.EntryAuto,
				SourceModule: "finance", SourceEventCode: "RECURRING_JOURNAL", SourceDocumentType: "recurring_journal", SourceDocumentID: r.ID + ":" + runDay, Lines: r.Lines})
			if err != nil {
				return err
			}
			next, err := domain.NextRunAfter(runDay, r.Frequency, r.AnchorDay)
			if err != nil {
				return err
			}
			r.NextRun, r.LastEntryID, r.LastError = next, e.ID, ""
			r.IsActive = r.EndDate == "" || next <= r.EndDate
			return tx.UpdateRecurringRun(ctx, r)
		})
		if err != nil {
			res.Failed++
			r.NextRun, r.LastError = runDay, apperr.As(err).Message
			s.recordRunError(ctx, s.Store.UpdateRecurringRun(ctx, r), "recurring", r.ID)
			return
		}
		res.Posted++
	}
}

func (s *Schedules) runAsset(ctx context.Context, a domain.FixedAsset, today string, res *RunResult) {
	for range maxCatchUp {
		n := a.DepreciatedMonths + 1
		if n > a.LifeMonths {
			return
		}
		end, err := a.MonthEnd(n)
		if err != nil || end > today {
			return
		}
		amount := a.DepreciationFor(n)
		before := a
		err = s.Store.WithTx(ctx, func(tx domain.Store) error {
			var entryID string
			if amount > 0 {
				e, err := createEntry(ctx, tx, NewEntry{EntryDate: end, Description: fmt.Sprintf("Penyusutan %s %s", a.Name, end[:7]), Post: true, EntryType: domain.EntryAuto,
					SourceModule: "finance", SourceEventCode: "ASSET_DEPRECIATION", SourceDocumentType: "fixed_asset", SourceDocumentID: fmt.Sprintf("%s:%d", a.ID, n),
					Lines: []domain.Line{
						{AccountID: a.ExpenseAccountID, Side: domain.Debit, Amount: amount, Memo: a.Name},
						{AccountID: a.AccumAccountID, Side: domain.Credit, Amount: amount, Memo: a.Name, SortOrder: 1},
					}})
				if err != nil {
					return err
				}
				entryID = e.ID
			}
			a.DepreciatedMonths, a.LastEntryID, a.LastError, a.IsActive = n, entryID, "", n < a.LifeMonths
			return tx.UpdateFixedAssetRun(ctx, a)
		})
		if err != nil {
			res.Failed++
			before.LastError = apperr.As(err).Message
			s.recordRunError(ctx, s.Store.UpdateFixedAssetRun(ctx, before), "asset", a.ID)
			return
		}
		res.Posted++
		if !a.IsActive {
			return
		}
	}
}

func (s *Schedules) recordRunError(_ context.Context, err error, kind, id string) {
	if err != nil && s.Logger != nil {
		s.Logger.Error("schedule error not saved", "kind", kind, "id", id, "err", err)
	}
}
