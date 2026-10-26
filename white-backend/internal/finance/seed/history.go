package seed

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Settled reimbursement history: eight to twelve claims a month, paid in
// one batch on the 27th, with one rejection every other month. The batch
// note marks a month done.

var historyCategories = []struct {
	code string
	base int64
}{
	{domain.CategoryTransport, 85_000}, {domain.CategoryMakan, 120_000}, {domain.CategoryAkomodasi, 650_000},
	{domain.CategoryOperasional, 450_000}, {domain.CategoryLainnya, 90_000},
}

var historyNotes = map[string]string{
	domain.CategoryTransport: "Transport meeting klien", domain.CategoryMakan: "Konsumsi rapat tim",
	domain.CategoryAkomodasi: "Perjalanan dinas klien", domain.CategoryOperasional: "Kebutuhan produksi konten",
	domain.CategoryLainnya: "Keperluan administrasi",
}

func historyNote(ym string) string { return "Reimbursement bulanan " + ym }

func (s *demo) claimMonthDone(ctx context.Context, ym string) (bool, error) {
	batches, err := s.d.Store.ListBatches(ctx, 200)
	if err != nil {
		return false, err
	}
	for _, b := range batches {
		if strings.HasPrefix(b.Notes, historyNote(ym)) {
			return true, nil
		}
	}
	return false, nil
}

// claimHistory seeds every past month of the current year, opening a closed
// period for the payment journal and closing it again.
func (s *demo) claimHistory(ctx context.Context, year *domain.FiscalYear) error {
	year, err := s.fiscal.GetYear(ctx, year.ID)
	if err != nil {
		return err
	}
	for _, p := range year.Periods {
		start, _ := time.Parse(time.DateOnly, p.StartDate)
		if start.Year() == s.today.Year() && start.Month() >= s.today.Month() {
			break
		}
		done, err := s.claimMonthDone(ctx, p.StartDate[:7])
		if err != nil || done {
			if err != nil {
				return err
			}
			continue
		}
		reopened := p.Status != domain.PeriodOpen
		if reopened {
			if _, err := s.fiscal.OpenPeriod(ctx, p.ID, false); err != nil {
				return err
			}
		}
		if err := s.claimMonth(ctx, start); err != nil {
			return fmt.Errorf("%s: %w", p.Name, err)
		}
		if reopened {
			if _, err := s.fiscal.ClosePeriod(ctx, p.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// claimMonth seeds one month of settled claims; its period must be OPEN.
func (s *demo) claimMonth(ctx context.Context, start time.Time) error {
	ym := start.Format("2006-01")
	if done, err := s.claimMonthDone(ctx, ym); err != nil || done {
		return err
	}
	seed := start.Year()*12 + int(start.Month())
	at := func(d, hour int) time.Time {
		return time.Date(start.Year(), start.Month(), d, hour, 0, 0, 0, app.Jakarta)
	}
	n := 8 + seed%5
	var toPay []string
	for k := 0; k < n; k++ {
		emp := employees[(seed*7+k*5)%len(employees)]
		cat := historyCategories[(seed+k*3)%len(historyCategories)]
		amount := (cat.base * int64(2+(seed+k)%5) / 2) / 5_000 * 5_000
		trxDay := 1 + (k*3+seed)%16
		submitted := at(trxDay+1, 9)
		managerAt := submitted.Add(time.Duration(1+(k+seed)%3) * 24 * time.Hour)
		financeAt := managerAt.Add(time.Duration(1+k%2) * 24 * time.Hour)
		c := domain.Claim{
			ID: ids.New(), EmployeeID: emp.id, EmployeeName: emp.name, DepartmentName: emp.dept, ManagerEmployeeID: emp.manager,
			Category: cat.code, TransactionDate: at(trxDay, 0).Format(time.DateOnly), Amount: domain.Rupiah(amount),
			Description: historyNotes[cat.code] + " " + monthNames[start.Month()-1], Status: domain.ClaimDraft,
		}
		prefix := "RB-" + start.Format("200601") + "-"
		no, err := s.d.Store.NextSequence(ctx, prefix)
		if err != nil {
			return err
		}
		c.ClaimNo = domain.SequenceNo(prefix, no)
		if err := s.d.Store.InsertClaim(ctx, c); err != nil {
			return err
		}
		policy, err := s.d.Store.GetPolicy(ctx, c.Category)
		if err != nil {
			return err
		}
		c.OverLimit, c.LimitNote = domain.EvaluateLimit(policy, c.Amount, 0)
		c.SubmittedAt, c.ManagerApprovedAt = &submitted, &managerAt
		history := []string{"CREATED", "SUBMITTED", "APPROVED_MANAGER"}
		if k == n-1 && seed%2 == 0 {
			c.Status, c.RejectedAt, c.RejectedStage, c.RejectReason = domain.ClaimRejected, &financeAt, string(domain.ClaimApprovedManager), "Nota tidak sesuai kebijakan reimbursement."
			history = append(history, "REJECTED")
		} else {
			c.Status, c.FinanceApprovedAt = domain.ClaimApprovedFinance, &financeAt
			history = append(history, "APPROVED_FINANCE")
			toPay = append(toPay, c.ID)
		}
		if err := s.d.Store.UpdateClaim(ctx, c); err != nil {
			return err
		}
		for _, action := range history {
			if err := s.d.Store.InsertClaimEvent(ctx, domain.ClaimEvent{ClaimID: c.ID, Action: action, ActorName: "Seeder", Note: c.RejectReason}); err != nil {
				return err
			}
		}
	}
	paidOn := at(27, 15)
	if _, err := app.NewReimbursements(s.d).Pay(ctx, admin, app.PayInput{ClaimIDs: toPay, CashAccountID: s.id("1102001"),
		PaidAt: paidOn.Format(time.DateOnly), Notes: historyNote(ym)}); err != nil {
		return err
	}
	for _, id := range toPay {
		c, err := s.d.Store.GetClaim(ctx, id)
		if err != nil {
			return err
		}
		c.PaidAt = &paidOn
		if err := s.d.Store.UpdateClaim(ctx, *c); err != nil {
			return err
		}
	}
	return nil
}
