package app

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// Fiscal covers fiscal years, periods, coverage, period closing and the
// beginning balance (NüHabit service_fiscal.go, service_opening.go).
type Fiscal struct{ Deps }

func NewFiscal(d Deps) *Fiscal { return &Fiscal{d} }

func (f *Fiscal) ListYears(ctx context.Context, search string, active *bool) ([]domain.FiscalYear, error) {
	return f.Store.ListFiscalYears(ctx, trim(search), active)
}

func (f *Fiscal) GetYear(ctx context.Context, id string) (*domain.FiscalYear, error) {
	y, err := f.Store.GetFiscalYear(ctx, id)
	if err == nil && y == nil {
		err = domain.NotFound("Fiscal year tidak ditemukan")
	}
	return y, err
}

// YearInput is the fiscal year form; no periods means twelve monthly ones.
type YearInput struct {
	Code, Name, StartDate, EndDate string
	IsActive                       bool
	Periods                        []domain.PeriodInput
}

func (in *YearInput) normalize() error {
	in.Code, in.Name = trim(in.Code), trim(in.Name)
	v := validate.New().Required("code", in.Code, "Kode wajib diisi").Required("name", in.Name, "Nama wajib diisi")
	if err := v.Err(); err != nil {
		return err
	}
	start, err := parseDay("start_date", in.StartDate)
	if err != nil {
		return err
	}
	end, err := parseDay("end_date", in.EndDate)
	if err != nil {
		return err
	}
	in.StartDate, in.EndDate = start, end
	if len(in.Periods) == 0 {
		s, _ := time.Parse(time.DateOnly, start)
		e, _ := time.Parse(time.DateOnly, end)
		in.Periods = domain.MonthlyPeriods(s, e)
	}
	if msg := domain.ValidateFiscalYear(in.StartDate, in.EndDate, in.Periods); msg != "" {
		return domain.Bad(msg)
	}
	return nil
}

// assertPreviousYearsClosed: a year with an OPEN period needs every earlier
// year fully CLOSED.
func assertPreviousYearsClosed(ctx context.Context, s domain.Store, start, excludeID string, periods []domain.PeriodInput) error {
	if !domain.HasOpenPeriod(periods) {
		return nil
	}
	prior, err := s.PriorYearWithOpenPeriods(ctx, start, excludeID)
	if err != nil || prior == nil {
		return err
	}
	return domain.Bad(fmt.Sprintf("Tidak bisa OPEN fiscal: fiscal year %s (%s) masih punya %d period OPEN. Closing semua period fiscal sebelumnya terlebih dahulu.", prior.Code, prior.Name, prior.OpenCount))
}

const duplicateYearCode = "Kode fiscal year sudah dipakai"

func (f *Fiscal) CreateYear(ctx context.Context, userID string, in YearInput) (*domain.FiscalYear, error) {
	if err := in.normalize(); err != nil {
		return nil, err
	}
	id := ids.New()
	err := f.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := assertPreviousYearsClosed(ctx, tx, in.StartDate, "", in.Periods); err != nil {
			return err
		}
		if err := tx.InsertFiscalYear(ctx, domain.FiscalYear{ID: id, Code: in.Code, Name: in.Name, StartDate: in.StartDate, EndDate: in.EndDate, IsActive: in.IsActive}, userID); err != nil {
			return uniqueAs(err, duplicateYearCode)
		}
		return tx.UpsertPeriods(ctx, id, in.Periods)
	})
	if err != nil {
		return nil, err
	}
	return f.Store.GetFiscalYear(ctx, id)
}

// UpdateYear keeps periods used by journals: they are updated in place and
// never dropped.
func (f *Fiscal) UpdateYear(ctx context.Context, userID, id string, in YearInput) (*domain.FiscalYear, error) {
	if _, err := f.GetYear(ctx, id); err != nil {
		return nil, err
	}
	if err := in.normalize(); err != nil {
		return nil, err
	}
	err := f.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := assertPreviousYearsClosed(ctx, tx, in.StartDate, id, in.Periods); err != nil {
			return err
		}
		used, err := tx.UsedPeriodNos(ctx, id)
		if err != nil {
			return err
		}
		for _, no := range used {
			if !slices.ContainsFunc(in.Periods, func(p domain.PeriodInput) bool { return p.PeriodNo == no }) {
				return domain.Bad(fmt.Sprintf("Period %d masih dipakai journal entry dan tidak boleh dihapus", no))
			}
		}
		if err := tx.UpsertPeriods(ctx, id, in.Periods); err != nil {
			return err
		}
		if err := tx.UpdateFiscalYear(ctx, domain.FiscalYear{ID: id, Code: in.Code, Name: in.Name, StartDate: in.StartDate, EndDate: in.EndDate, IsActive: in.IsActive}, userID); err != nil {
			return uniqueAs(err, duplicateYearCode)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return f.Store.GetFiscalYear(ctx, id)
}

func (f *Fiscal) DeleteYear(ctx context.Context, userID, id string) error {
	if _, err := f.GetYear(ctx, id); err != nil {
		return err
	}
	n, err := f.Store.CountYearEntries(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return domain.Bad("Fiscal year masih dipakai journal entry dan tidak bisa dihapus")
	}
	return f.Store.WithTx(ctx, func(tx domain.Store) error { return tx.SoftDeleteFiscalYear(ctx, id, userID) })
}

// ─── Periods ────────────────────────────────────────────────

func (f *Fiscal) ListPeriods(ctx context.Context, filter domain.PeriodFilter) ([]domain.FiscalPeriod, error) {
	filter.Search = trim(filter.Search)
	return f.Store.ListPeriods(ctx, filter)
}

func (f *Fiscal) period(ctx context.Context, s domain.Store, id string) (*domain.FiscalPeriod, error) {
	p, err := s.GetPeriod(ctx, id)
	if err == nil && p == nil {
		err = domain.NotFound("Fiscal period tidak ditemukan")
	}
	return p, err
}

func periodNames(ps []domain.FiscalPeriod) string {
	names := make([]string, len(ps))
	for i, p := range ps {
		names[i] = p.Name
	}
	return strings.Join(names, ", ")
}

// OpenPeriod opens a period; earlier OPEN periods of the year must be
// closed first, or closePrevious closes them.
func (f *Fiscal) OpenPeriod(ctx context.Context, id string, closePrevious bool) (*domain.FiscalPeriod, error) {
	err := f.Store.WithTx(ctx, func(tx domain.Store) error {
		target, err := f.period(ctx, tx, id)
		if err != nil {
			return err
		}
		if !target.YearActive {
			return domain.Bad("Fiscal year tidak aktif")
		}
		if target.Status == domain.PeriodOpen {
			return nil
		}
		year, err := tx.GetFiscalYear(ctx, target.FiscalYearID)
		if err != nil {
			return err
		}
		prior, err := tx.PriorYearWithOpenPeriods(ctx, year.StartDate, year.ID)
		if err != nil {
			return err
		}
		if prior != nil {
			return domain.Bad("Tidak bisa OPEN: fiscal year " + prior.Code + " belum fully CLOSED")
		}
		previous, err := tx.PreviousOpenPeriods(ctx, target.FiscalYearID, target.PeriodNo)
		if err != nil {
			return err
		}
		if len(previous) > 0 {
			if !closePrevious {
				return domain.Bad("Tutup period " + periodNames(previous) + " terlebih dahulu sebelum OPEN " + target.Name)
			}
			pids := make([]string, len(previous))
			for i, p := range previous {
				pids[i] = p.ID
			}
			if err := tx.SetPeriodStatus(ctx, pids, domain.PeriodClosed); err != nil {
				return err
			}
		}
		return tx.SetPeriodStatus(ctx, []string{id}, domain.PeriodOpen)
	})
	if err != nil {
		return nil, err
	}
	return f.period(ctx, f.Store, id)
}

// ClosePreview lists the blockers of closing a period.
type ClosePreview struct {
	Period       domain.FiscalPeriod
	DraftEntries []domain.JournalEntry
	CanClose     bool
	Blockers     []string
}

func (f *Fiscal) ClosePreview(ctx context.Context, id string) (*ClosePreview, error) {
	p, err := f.period(ctx, f.Store, id)
	if err != nil {
		return nil, err
	}
	drafts, err := f.Store.DraftEntries(ctx, id, 20)
	if err != nil {
		return nil, err
	}
	blockers := domain.CloseBlockers(p.Name, p.Status, p.YearActive, p.DraftCount)
	return &ClosePreview{Period: *p, DraftEntries: drafts, CanClose: len(blockers) == 0, Blockers: blockers}, nil
}

// ClosePeriod is a soft close, refused while DRAFT journals remain. No
// closing journals are created.
func (f *Fiscal) ClosePeriod(ctx context.Context, id string) (*domain.FiscalPeriod, error) {
	preview, err := f.ClosePreview(ctx, id)
	if err != nil {
		return nil, err
	}
	if !preview.CanClose {
		return nil, domain.Bad(preview.Blockers[0])
	}
	if err := f.Store.SetPeriodStatus(ctx, []string{id}, domain.PeriodClosed); err != nil {
		return nil, err
	}
	return f.period(ctx, f.Store, id)
}

// Coverage says whether a journal can be dated on date, and how to open
// its period otherwise.
type Coverage struct {
	Date       string
	Ready      bool
	Period     *domain.FiscalPeriod
	Suggestion *OpenSuggestion
}

type OpenSuggestion struct {
	Period        domain.FiscalPeriod
	CanOpen       bool
	ClosePrevious []domain.FiscalPeriod
	Message       string
}

func (f *Fiscal) Coverage(ctx context.Context, date string) (*Coverage, error) {
	if date == "" {
		date = f.today()
	}
	date, err := parseDay("date", date)
	if err != nil {
		return nil, err
	}
	open, err := f.Store.OpenPeriodFor(ctx, date)
	if err != nil {
		return nil, err
	}
	if open != nil {
		return &Coverage{Date: date, Ready: true, Period: open}, nil
	}
	covering, err := f.Store.CoveringPeriod(ctx, date)
	if err != nil {
		return nil, err
	}
	if covering == nil {
		return &Coverage{Date: date}, nil
	}
	year, err := f.Store.GetFiscalYear(ctx, covering.FiscalYearID)
	if err != nil {
		return nil, err
	}
	prior, err := f.Store.PriorYearWithOpenPeriods(ctx, year.StartDate, year.ID)
	if err != nil {
		return nil, err
	}
	if prior != nil {
		return &Coverage{Date: date, Suggestion: &OpenSuggestion{Period: *covering, Message: fmt.Sprintf(
			"Period %s masih CLOSED, tapi fiscal year %s masih punya %d period OPEN. Closing fiscal sebelumnya dulu.", covering.Name, prior.Code, prior.OpenCount)}}, nil
	}
	previous, err := f.Store.PreviousOpenPeriods(ctx, covering.FiscalYearID, covering.PeriodNo)
	if err != nil {
		return nil, err
	}
	msg := "Period " + covering.Name + " belum OPEN. Buka sekarang agar bisa input jurnal di tanggal ini."
	if len(previous) > 0 {
		msg = "Period " + covering.Name + " belum OPEN. Sistem bisa menutup " + periodNames(previous) + " lalu membuka " + covering.Name + "."
	}
	return &Coverage{Date: date, Suggestion: &OpenSuggestion{Period: *covering, CanOpen: true, ClosePrevious: previous, Message: msg}}, nil
}

// ─── Beginning balance ──────────────────────────────────────

type OpeningLine struct {
	Account         domain.Account
	SuggestedAmount domain.Money
	Amount          domain.Money
	Side            string
	Source          string
}

type OpeningSuggestion struct {
	Year             domain.FiscalYear
	Period           *domain.FiscalPeriod
	Prior            *domain.YearOpenCount
	RetainedEarnings *domain.Account
	Lines            []OpeningLine
	TotalDebit       domain.Money
	TotalCredit      domain.Money
	Existing         *domain.JournalEntry
	CanEdit          bool
	Message          string
}

// retainedEarnings is the first postable equity account named like
// retained earnings / laba ditahan, or coded 32x / 3102x.
func retainedEarnings(accounts []domain.Account) *domain.Account {
	for _, a := range accounts {
		name := strings.ToLower(a.Name)
		if a.TypeCode != domain.TypeEquity || !a.IsPostable || !a.IsActive {
			continue
		}
		if strings.Contains(name, "retained") || strings.Contains(name, "laba ditahan") ||
			strings.HasPrefix(a.Code, "32") || strings.HasPrefix(a.Code, "3102") {
			return &a
		}
	}
	return nil
}

func openingTotals(lines []OpeningLine) (d, c domain.Money) {
	for _, l := range lines {
		if l.Side == domain.Debit {
			d += l.Amount
		} else {
			c += l.Amount
		}
	}
	return d, c
}

// BeginningBalance is the OPENING journal of a year, or a suggestion from
// the prior year's closing balance sheet (P&L folded into retained earnings).
func (f *Fiscal) BeginningBalance(ctx context.Context, yearID string) (*OpeningSuggestion, error) {
	year, err := f.GetYear(ctx, yearID)
	if err != nil {
		return nil, err
	}
	accounts, err := f.Store.ListAccounts(ctx, domain.AccountFilter{})
	if err != nil {
		return nil, err
	}
	byID := map[string]domain.Account{}
	for _, a := range accounts {
		byID[a.ID] = a
	}
	out := &OpeningSuggestion{Year: *year, CanEdit: true, RetainedEarnings: retainedEarnings(accounts)}
	if len(year.Periods) > 0 {
		out.Period = &year.Periods[0]
	}
	if out.Prior, err = f.Store.PriorFiscalYear(ctx, year.StartDate); err != nil {
		return nil, err
	}
	existing, err := f.Store.OpeningEntry(ctx, yearID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		for _, l := range existing.Lines {
			source := "MANUAL"
			if out.RetainedEarnings != nil && l.AccountID == out.RetainedEarnings.ID {
				source = "RETAINED_EARNINGS"
			}
			out.Lines = append(out.Lines, OpeningLine{Account: byID[l.AccountID], SuggestedAmount: l.Amount, Amount: l.Amount, Side: l.Side, Source: source})
		}
		out.TotalDebit, out.TotalCredit = openingTotals(out.Lines)
		out.Existing = existing
		out.CanEdit = existing.Status == domain.StatusDraft
		out.Message = "Beginning balance draft, bisa diedit."
		if existing.Status == domain.StatusPosted {
			out.Message = "Beginning balance sudah POSTED dan terkunci."
		}
		return out, nil
	}
	prior := out.Prior
	var msgs []string
	switch {
	case prior == nil:
		msgs = append(msgs, "Fiscal "+year.Code+" adalah fiscal year pertama. Tidak ada FY sebelumnya untuk di-suggest, isi saldo awal Kas & Bank secara manual, lalu simpan/post.")
	case prior.OpenCount > 0:
		msgs = append(msgs, fmt.Sprintf("Fiscal %s belum fully CLOSED (%d period OPEN). Suggest saldo akhir belum final, sebaiknya closing dulu.", prior.Code, prior.OpenCount))
	}
	if prior != nil {
		totals, err := f.Store.YearAccountTotals(ctx, prior.ID)
		if err != nil {
			return nil, err
		}
		var plNetCredit domain.Money
		for _, t := range totals {
			if domain.IsProfitLossType(t.TypeCode) {
				plNetCredit += t.Credit - t.Debit
				continue
			}
			side, amount, ok := domain.OpeningLine(t.Debit, t.Credit)
			if !ok {
				continue
			}
			out.Lines = append(out.Lines, OpeningLine{Account: byID[t.ID], SuggestedAmount: amount, Amount: amount, Side: side, Source: "PRIOR_BS"})
		}
		switch {
		case plNetCredit != 0 && out.RetainedEarnings != nil:
			side := domain.Debit
			if plNetCredit > 0 {
				side = domain.Credit
			}
			out.Lines = append(out.Lines, OpeningLine{Account: *out.RetainedEarnings, SuggestedAmount: plNetCredit.Abs(), Amount: plNetCredit.Abs(), Side: side, Source: "RETAINED_EARNINGS"})
		case plNetCredit != 0:
			msgs = append(msgs, "Laba/rugi bersih "+plNetCredit.Number()+" perlu akun Retained Earnings (Equity), buat/pilih akun Laba Ditahan di COA.")
		}
		if len(msgs) == 0 {
			msgs = append(msgs, "Suggest dari saldo akhir "+prior.Code+" (POSTED). Angka bisa diedit sebelum simpan.")
		}
	}
	sort.SliceStable(out.Lines, func(i, j int) bool {
		oi, oj := domain.BalanceTypeOrder(out.Lines[i].Account.TypeCode), domain.BalanceTypeOrder(out.Lines[j].Account.TypeCode)
		if oi != oj {
			return oi < oj
		}
		return out.Lines[i].Account.Code < out.Lines[j].Account.Code
	})
	out.TotalDebit, out.TotalCredit = openingTotals(out.Lines)
	out.Message = strings.Join(msgs, " ")
	return out, nil
}

// SaveBeginningBalance keeps one OPENING entry per year, editable while
// DRAFT. Period 1 is forced OPEN for the opening date.
func (f *Fiscal) SaveBeginningBalance(ctx context.Context, userID, yearID string, input []domain.Line, post bool) (*domain.JournalEntry, error) {
	sug, err := f.BeginningBalance(ctx, yearID)
	if err != nil {
		return nil, err
	}
	if !sug.CanEdit {
		return nil, domain.Bad("Beginning balance sudah POSTED dan tidak bisa diubah")
	}
	if sug.Period == nil {
		return nil, domain.Bad("Fiscal year belum punya period")
	}
	var lines []domain.Line
	var debit, credit domain.Money
	for _, l := range input {
		if l.Amount <= 0 {
			continue
		}
		lines = append(lines, l)
		if l.Side == domain.Debit {
			debit += l.Amount
		} else {
			credit += l.Amount
		}
	}
	if len(lines) < 2 {
		return nil, domain.Bad("Minimal 2 baris saldo awal (debit & credit)")
	}
	if debit != credit {
		return nil, domain.Bad("Saldo awal tidak balance: Debit " + debit.Number() + " ≠ Credit " + credit.Number())
	}
	status := domain.StatusDraft
	if post {
		status = domain.StatusPosted
	}
	year := sug.Year
	entryID := ""
	err = f.Store.WithTx(ctx, func(tx domain.Store) error {
		accountIDs := make([]string, len(lines))
		for i, l := range lines {
			accountIDs[i] = l.AccountID
		}
		if err := checkAccounts(ctx, tx, accountIDs, "Akun COA wajib diisi", "Akun jurnal harus postable (bukan header)"); err != nil {
			return err
		}
		if sug.Period.Status != domain.PeriodOpen {
			if err := tx.SetPeriodStatus(ctx, []string{sug.Period.ID}, domain.PeriodOpen); err != nil {
				return err
			}
		}
		e := domain.JournalEntry{EntryDate: year.StartDate, Description: "Beginning balance " + year.Code, FiscalPeriodID: sug.Period.ID, Status: status}
		if sug.Existing != nil {
			e.ID = sug.Existing.ID
			if err := tx.UpdateJournalHeader(ctx, e, userID); err != nil {
				return err
			}
		} else {
			prefix := domain.OpeningNoPrefix(year.StartDate)
			n, err := tx.NextSequence(ctx, prefix)
			if err != nil {
				return err
			}
			e.ID, e.EntryNo, e.EntryType, e.SourceFiscalYearID, e.CreatedBy = ids.New(), domain.SequenceNo(prefix, n), domain.EntryOpening, year.ID, userID
			if err := tx.InsertJournalEntry(ctx, e); err != nil {
				return err
			}
		}
		entryID = e.ID
		return tx.ReplaceJournalLines(ctx, e.ID, lines)
	})
	if err != nil {
		return nil, err
	}
	return f.Store.GetJournalEntry(ctx, entryID)
}
