package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
)

// Maintenance runs the event-driven jobs: forgetting deleted users and the
// daily reminders to HR (contract expiry, probation end, birthdays).
type Maintenance struct{ d Deps }

func NewMaintenance(d Deps) *Maintenance { return &Maintenance{d: d} }

// ForgetUser unlinks a deleted identity user from its employee record.
func (m *Maintenance) ForgetUser(ctx context.Context, userID string) error {
	_, err := m.d.Store.UnlinkUser(ctx, userID)
	return err
}

// DailyReport is what one daily tick sent.
type DailyReport struct {
	ContractReminders, ProbationReminders, Birthdays int
}

// Daily sends the HR reminders of the WIB day of runAt. Each reminder is
// claimed in hr.reminder_sends first, so a redelivered tick sends nothing
// twice.
func (m *Maintenance) Daily(ctx context.Context, runAt time.Time) (DailyReport, error) {
	var rep DailyReport
	today := hris.TodayWIB(runAt)
	contracts, err := m.d.Store.ListContracts(ctx, domain.ContractFilter{Status: "active"})
	if err != nil {
		return rep, err
	}
	for _, c := range contracts {
		if c.EndDate == nil {
			continue
		}
		n, due := hris.ReminderDue(today, *c.EndDate, hris.ContractReminderDays)
		if !due {
			continue
		}
		sent, err := m.remind(ctx, "contract_expiry", fmt.Sprintf("%s:%d", c.ID, n), today, NotifyInput{
			Type: "HR_CONTRACT_EXPIRING", Title: fmt.Sprintf("Kontrak %s berakhir %d hari lagi", c.EmployeeName, n),
			Body: fmt.Sprintf("%s (%s) berakhir %s. Perpanjang, konversi, atau akhiri kontrak.", c.ContractNumber, strings.ToUpper(c.ContractType), *c.EndDate),
			Href: "/erp/hr/contracts"})
		if err != nil {
			return rep, err
		}
		if sent {
			rep.ContractReminders++
		}
	}
	probations, err := m.d.Store.ProbationsEnding(ctx, today, hris.AddDaysISO(today, 14))
	if err != nil {
		return rep, err
	}
	for _, e := range probations {
		n, due := hris.ReminderDue(today, deref(e.ProbationEndDate), hris.ProbationReminderDays)
		if !due {
			continue
		}
		title := fmt.Sprintf("Masa percobaan %s berakhir %d hari lagi", e.FullName, n)
		if n == 0 {
			title = "Masa percobaan " + e.FullName + " berakhir hari ini"
		}
		sent, err := m.remind(ctx, "probation_end", fmt.Sprintf("%s:%d", e.ID, n), today, NotifyInput{
			Type: "HR_PROBATION_ENDING", Title: title, Body: "Siapkan evaluasi dan keputusan status kepegawaian.",
			Href: "/erp/hr/employees/" + e.ID})
		if err != nil {
			return rep, err
		}
		if sent {
			rep.ProbationReminders++
		}
	}
	people, err := m.d.Store.ActiveBirthdays(ctx)
	if err != nil {
		return rep, err
	}
	var names []string
	for _, e := range people {
		if e.BirthDate != nil && hris.IsBirthday(*e.BirthDate, today) {
			names = append(names, e.FullName)
		}
	}
	if len(names) > 0 {
		sent, err := m.remind(ctx, "birthdays", "all", today, NotifyInput{Type: "HR_BIRTHDAYS",
			Title: fmt.Sprintf("Ulang tahun hari ini (%d)", len(names)), Body: strings.Join(names, ", "), Href: "/erp/hr/employees"})
		if err != nil {
			return rep, err
		}
		if sent {
			rep.Birthdays = len(names)
		}
	}
	return rep, nil
}

func (m *Maintenance) remind(ctx context.Context, kind, ref, today string, in NotifyInput) (bool, error) {
	claimed, err := m.d.Store.ClaimReminder(ctx, kind, ref, today)
	if err != nil || !claimed {
		return false, err
	}
	notifyHR(ctx, m.d, in)
	return true, nil
}
