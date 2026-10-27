package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
)

// Demo data for Mixist, a 25-person digital agency: the prior fiscal year
// (prior.go), then fiscal year 2026 with an opening balance and month by month (January to the current month)
// client invoices and receipts through the sales events, vendor bills and
// payments, payroll through the hr event, cash movements and depreciation,
// each period opened in turn. Every past month gets settled reimbursement
// claims; the current month gets claims in every state and one AI memo. A month is skipped once its
// depreciation journal (written last) exists.

var admin = authn.Principal{Role: authn.RoleAdmin, Name: "Seeder"}

type client struct {
	name     string
	retainer int64
}

var clients = []client{
	{"PT Kopi Nusantara Jaya", 45_000_000}, {"PT Batik Prameswari", 38_000_000}, {"PT Sehat Sentosa Farma", 65_000_000},
	{"CV Rumah Kayu Jepara", 22_000_000}, {"PT Arunika Properti", 75_000_000}, {"PT Gizi Bunda Indonesia", 52_000_000},
	{"PT Lintas Logistik Cepat", 41_000_000}, {"Klinik Estetika Ayu", 28_000_000}, {"PT EduPintar Digital", 60_000_000},
}

type employee struct{ id, name, dept, manager string }

var employees = []employee{
	{"demo-emp-01", "Andika Pratama", "Creative", ""}, {"demo-emp-02", "Sekar Ayuningtyas", "Creative", "demo-emp-01"},
	{"demo-emp-03", "Rizky Maulana", "Creative", "demo-emp-01"}, {"demo-emp-04", "Nadia Putri", "Creative", "demo-emp-01"},
	{"demo-emp-05", "Bayu Saputra", "Social Media", ""}, {"demo-emp-06", "Citra Lestari", "Social Media", "demo-emp-05"},
	{"demo-emp-07", "Dimas Aditya", "Social Media", "demo-emp-05"}, {"demo-emp-08", "Intan Permatasari", "Social Media", "demo-emp-05"},
	{"demo-emp-09", "Fajar Nugroho", "SEO", ""}, {"demo-emp-10", "Gita Savitri", "SEO", "demo-emp-09"},
	{"demo-emp-11", "Hendra Wijaya", "SEO", "demo-emp-09"}, {"demo-emp-12", "Kevin Halim", "Ads", ""},
	{"demo-emp-13", "Larasati Dewi", "Ads", "demo-emp-12"}, {"demo-emp-14", "Made Surya", "Ads", "demo-emp-12"},
	{"demo-emp-15", "Novia Rahma", "Ads", "demo-emp-12"}, {"demo-emp-16", "Oki Ramadhan", "Sales", ""},
	{"demo-emp-17", "Putri Anggraini", "Sales", "demo-emp-16"}, {"demo-emp-18", "Reza Firmansyah", "Sales", "demo-emp-16"},
	{"demo-emp-19", "Salsabila Zahra", "Sales", "demo-emp-16"}, {"demo-emp-20", "Taufik Hidayat", "Finance", ""},
	{"demo-emp-21", "Utami Kusuma", "Finance", "demo-emp-20"}, {"demo-emp-22", "Vina Oktaviani", "Finance", "demo-emp-20"},
	{"demo-emp-23", "Wulan Sari", "HR", ""}, {"demo-emp-24", "Yoga Prasetyo", "HR", "demo-emp-23"}, {"demo-emp-25", "Zaki Alfarizi", "HR", "demo-emp-23"},
}

var monthNames = []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

type demo struct {
	d        app.Deps
	acc      map[string]domain.Account
	today    time.Time
	fiscal   *app.Fiscal
	ledgers  *app.Ledgers
	cash     *app.CashBank
	journals *app.Journals
	auto     *app.AutoPost
}

func (s *demo) id(code string) string { return s.acc[code].ID }

// Demo loads the demo transactions. Reference must have run first.
func Demo(ctx context.Context, d app.Deps) error {
	acc, err := d.Store.AccountsByCode(ctx)
	if err != nil {
		return err
	}
	now := d.Clock.Now().In(app.Jakarta)
	s := &demo{d: d, acc: acc, today: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
		fiscal: app.NewFiscal(d), ledgers: app.NewLedgers(d), cash: app.NewCashBank(d), journals: app.NewJournals(d), auto: app.NewAutoPost(d)}
	if err := s.priorYear(ctx); err != nil {
		return fmt.Errorf("tahun sebelumnya: %w", err)
	}
	year, err := s.fiscalYear(ctx)
	if err != nil {
		return err
	}
	if err := s.opening(ctx, year); err != nil {
		return err
	}
	for _, p := range year.Periods {
		start, _ := time.Parse(time.DateOnly, p.StartDate)
		if start.After(s.today) {
			break
		}
		if err := s.month(ctx, p); err != nil {
			return fmt.Errorf("%s: %w", p.Name, err)
		}
	}
	if err := s.claimHistory(ctx, year); err != nil {
		return fmt.Errorf("riwayat reimbursement: %w", err)
	}
	if err := s.claims(ctx); err != nil {
		return fmt.Errorf("reimbursement: %w", err)
	}
	if err := s.extras(ctx); err != nil {
		return fmt.Errorf("data pelengkap: %w", err)
	}
	return s.insight(ctx)
}

func (s *demo) fiscalYear(ctx context.Context) (*domain.FiscalYear, error) {
	code := fmt.Sprintf("FY%d", s.today.Year())
	years, err := s.fiscal.ListYears(ctx, code, nil)
	if err != nil {
		return nil, err
	}
	for i := range years {
		if years[i].Code == code {
			return &years[i], nil
		}
	}
	return s.fiscal.CreateYear(ctx, "", app.YearInput{Code: code, Name: fmt.Sprintf("Tahun Buku %d", s.today.Year()),
		StartDate: fmt.Sprintf("%d-01-01", s.today.Year()), EndDate: fmt.Sprintf("%d-12-31", s.today.Year()), IsActive: true})
}

func (s *demo) opening(ctx context.Context, year *domain.FiscalYear) error {
	existing, err := s.d.Store.OpeningEntry(ctx, year.ID)
	if err != nil || existing != nil {
		return err
	}
	_, err = s.fiscal.SaveBeginningBalance(ctx, "", year.ID, s.openingLines(), true)
	return err
}
