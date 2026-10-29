package app

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
)

// Automations are the scheduled jobs, driven by the scheduler's ERP ticks
// so the service stays stateless. Every reminder claims a row in
// reminder_sends first, so a redelivered tick never sends twice a day.
type Automations struct {
	Deps
	kpi    *KPI
	tasks  *Tasks
	recaps *Recaps
}

func NewAutomations(d Deps, kpi *KPI, tasks *Tasks, recaps *Recaps) *Automations {
	return &Automations{Deps: d, kpi: kpi, tasks: tasks, recaps: recaps}
}

// Reminder times (WIB) and limits.
const (
	logbookReminderHour   = 17
	reviewLeadDays        = 3
	feedbackRemindEvery   = 48 * time.Hour
	feedbackMaxReminders  = 5
	reminderKindDigest    = "dept_tasks_digest"
	reminderKindOverdue   = "dept_tasks_overdue"
	reminderKindLogbook   = "logbook_eod"
	reminderKindReview    = "review_deadline"
	reminderKindRate      = "review_rate"
	reminderKindImprove   = "improvement_overdue"
	reminderKindKPISnap   = "kpi_snapshot"
	reminderKindFeedback  = "feedback_pending"
	reminderKindKPIRubric = "kpi_rubric"
)

// Tick15m: task digests and overdue notices, the end-of-day logbook
// reminder and the 17:30 daily recap.
func (au *Automations) Tick15m(ctx context.Context, now time.Time) error {
	_, recapErr := au.recaps.RunDaily(ctx, now)
	return errors.Join(au.taskReminders(ctx, now), au.logbookReminders(ctx, now), recapErr)
}

// TickHourly materializes task occurrences.
func (au *Automations) TickHourly(ctx context.Context, now time.Time) error {
	return au.tasks.Materialize(ctx, now)
}

// TickDaily (07:05 WIB): review, 360 and improvement reminders; on the
// 1st the KPI snapshot of last month and the monthly recaps.
func (au *Automations) TickDaily(ctx context.Context, now time.Time) error {
	_, monthlyErr := au.recaps.RunMonthly(ctx, now)
	return errors.Join(au.reviewDeadlines(ctx, now), au.feedbackReminders(ctx, now), au.improvementReminders(ctx, now),
		au.kpiSnapshotDue(ctx, now), monthlyErr)
}

// claim records the send; false when it already happened today.
func (au *Automations) claim(ctx context.Context, kind, subject, day string) bool {
	ok, err := au.Store.ClaimSend(ctx, kind, subject, day)
	if err != nil {
		au.log().Warn("reminder claim failed", "kind", kind, "err", err)
		return false
	}
	return ok
}

// taskReminders: from 07:00 WIB a digest of today's and overdue open
// occurrences per assignee; from 16:00 the overdue notice.
func (au *Automations) taskReminders(ctx context.Context, now time.Time) error {
	digest := domain.ReminderDue(now, domain.DigestHour, 0, domain.Jakarta)
	overdue := domain.ReminderDue(now, domain.OverdueHour, 0, domain.Jakarta)
	if !digest && !overdue {
		return nil
	}
	today := domain.DayKey(now, domain.Jakarta)
	lines, err := au.Store.ReminderLines(ctx, today)
	if err != nil {
		return err
	}
	byEmp := map[string][]domain.ReminderLine{}
	var order []string
	for _, l := range lines {
		if _, ok := byEmp[l.EmployeeID]; !ok {
			order = append(order, l.EmployeeID)
		}
		byEmp[l.EmployeeID] = append(byEmp[l.EmployeeID], l)
	}
	if len(order) == 0 {
		return nil
	}
	ppl, err := au.people(ctx)
	if err != nil {
		return err
	}
	for _, emp := range order {
		var titles, late []string
		for _, l := range byEmp[emp] {
			titles = append(titles, l.Title)
			if l.Date < today {
				late = append(late, l.Title)
			}
		}
		if digest && au.claim(ctx, reminderKindDigest, emp, today) {
			title, body := domain.ComposeTaskNotice(titles, len(late), false)
			au.notify(ctx, Notice{UserIDs: ppl.userIDs(emp), Type: NotifTaskDigest, Title: title, Body: body, Href: hrefTasks})
		}
		if overdue && len(late) > 0 && au.claim(ctx, reminderKindOverdue, emp, today) {
			title, body := domain.ComposeTaskNotice(late, len(late), true)
			au.notify(ctx, Notice{UserIDs: ppl.userIDs(emp), Type: NotifTaskOverdue, Title: title, Body: body, Href: hrefTasks})
		}
	}
	return nil
}

// logbookReminders: from 17:00 WIB on workdays, members of departments
// with a daily template whose logbook of today is not submitted yet.
func (au *Automations) logbookReminders(ctx context.Context, now time.Time) error {
	if !domain.IsWorkday(now.In(domain.Jakarta)) || !domain.ReminderDue(now, logbookReminderHour, 0, domain.Jakarta) {
		return nil
	}
	today := domain.DayKey(now, domain.Jakarta)
	depts, err := au.Store.DailyTemplateDepartments(ctx)
	if err != nil || len(depts) == 0 {
		return err
	}
	entries, _, err := au.Store.ListEntries(ctx, domain.EntryFilter{Date: today, Limit: 1000})
	if err != nil {
		return err
	}
	sent := map[string]bool{}
	drafts := map[string]int{}
	for _, e := range entries {
		if e.Status == domain.LogbookDraft {
			drafts[e.EmployeeID]++
		} else {
			sent[e.EmployeeID] = true
		}
	}
	ppl, err := au.people(ctx)
	if err != nil {
		return err
	}
	for _, d := range depts {
		for _, e := range ppl.members(d) {
			if sent[e.ID] || !au.claim(ctx, reminderKindLogbook, e.ID, today) {
				continue
			}
			body := "Logbook hari ini belum diisi. Isi checklist lalu kirim sebelum pulang."
			if drafts[e.ID] > 0 {
				body = fmt.Sprintf("Ada %d logbook hari ini yang masih draft. Jangan lupa kirim sebelum pulang.", drafts[e.ID])
			}
			au.notify(ctx, Notice{UserIDs: ppl.userIDs(e.ID), Type: NotifLogbookPending, Title: "Logbook hari ini belum dikirim", Body: body, Href: hrefLogbook})
		}
	}
	return nil
}

// reviewDeadlines: from three days before an open cycle's due date, a
// daily reminder to employees without a self assessment and to leads with
// unrated or unsigned reviews.
func (au *Automations) reviewDeadlines(ctx context.Context, now time.Time) error {
	today := domain.DayKey(now, domain.Jakarta)
	reviews, err := au.Store.ListOpenCycleReviews(ctx)
	if err != nil || len(reviews) == 0 {
		return err
	}
	ppl, err := au.people(ctx)
	if err != nil {
		return err
	}
	pendingByLead := map[string][]string{}
	dueByLead := map[string]string{}
	for _, rv := range reviews {
		due, _ := domain.ParseDay(rv.CycleDueDate)
		if today < due.AddDate(0, 0, -reviewLeadDays).Format(domain.IsoDate) {
			continue
		}
		when := "tenggat " + rv.CycleDueDate
		if today > rv.CycleDueDate {
			when = "sudah lewat tenggat " + rv.CycleDueDate
		}
		e, ok := ppl.employee(rv.EmployeeID)
		if !ok || !e.Active {
			continue
		}
		if rv.SelfAssessment == nil && au.claim(ctx, reminderKindReview, rv.ID, today) {
			au.notify(ctx, Notice{UserIDs: ppl.userIDs(e.ID), Type: NotifReviewDeadline, Href: hrefReviews,
				Title: "Self assessment " + rv.CycleName + " belum diisi", Body: "Isi self assessment review " + rv.CycleName + " (" + when + ")."})
		}
		if rv.RatedItems < rv.TotalItems || rv.ReviewerSignDate == nil {
			for _, lead := range ppl.leadsOf(e) {
				pendingByLead[lead] = append(pendingByLead[lead], e.Name)
				dueByLead[lead] = rv.CycleDueDate
			}
		}
	}
	leads := make([]string, 0, len(pendingByLead))
	for l := range pendingByLead {
		leads = append(leads, l)
	}
	sort.Strings(leads)
	for _, lead := range leads {
		if !au.claim(ctx, reminderKindRate, lead, today) {
			continue
		}
		names := pendingByLead[lead]
		au.notify(ctx, Notice{UserIDs: ppl.userIDs(lead), Type: NotifReviewDeadline, Href: hrefReviews,
			Title: fmt.Sprintf("%d review tim belum selesai", len(names)),
			Body:  fmt.Sprintf("Lengkapi penilaian dan tanda tangan untuk %s (tenggat %s).", joinNames(names, 5), dueByLead[lead])})
	}
	return nil
}

func joinNames(names []string, limit int) string {
	sort.Strings(names)
	if len(names) <= limit {
		return listID(names)
	}
	return fmt.Sprintf("%s dan %d lainnya", listID(names[:limit]), len(names)-limit)
}

// listID joins "a, b dan c".
func listID(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	out := items[0]
	for _, s := range items[1 : len(items)-1] {
		out += ", " + s
	}
	return out + " dan " + items[len(items)-1]
}

// feedbackReminders: reviewers with open 360 assignments due within three
// days (or without a due date), at most every 48 hours and five times per
// assignment (reminder_count, last_reminder_at).
func (au *Automations) feedbackReminders(ctx context.Context, now time.Time) error {
	today := domain.DayKey(now, domain.Jakarta)
	rows, err := au.Store.ListRemindableAssignments(ctx)
	if err != nil || len(rows) == 0 {
		return err
	}
	byReviewer := map[string][]domain.FeedbackAssignment{}
	for _, a := range rows {
		if a.ReminderCount >= feedbackMaxReminders || (a.LastReminderAt != nil && now.Sub(*a.LastReminderAt) < feedbackRemindEvery) {
			continue
		}
		if a.DueDate != nil {
			due, _ := domain.ParseDay(*a.DueDate)
			if today < due.AddDate(0, 0, -reviewLeadDays).Format(domain.IsoDate) {
				continue
			}
		}
		byReviewer[a.ReviewerID] = append(byReviewer[a.ReviewerID], a)
	}
	if len(byReviewer) == 0 {
		return nil
	}
	ppl, err := au.people(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for reviewer, list := range byReviewer {
		if !au.claim(ctx, reminderKindFeedback, reviewer, today) {
			continue
		}
		nearest := ""
		for i := range list {
			a := &list[i]
			if a.DueDate != nil && (nearest == "" || *a.DueDate < nearest) {
				nearest = *a.DueDate
			}
			a.ReminderCount++
			a.LastReminderAt = &now
			if err := au.Store.UpdateAssignment(ctx, *a); err != nil {
				errs = append(errs, err)
			}
		}
		au.notify(ctx, Notice{UserIDs: ppl.userIDs(reviewer), Type: NotifFeedbackPending, Href: hrefFeedback,
			Title: "Feedback 360 menunggu Anda", Body: pendingReminderText(len(list), nearest)})
	}
	return errors.Join(errs...)
}

// improvementReminders: open items past their due date, per owner (the
// department head for department items without an owner).
func (au *Automations) improvementReminders(ctx context.Context, now time.Time) error {
	today := domain.DayKey(now, domain.Jakarta)
	items, err := au.Store.OverdueImprovements(ctx, today)
	if err != nil || len(items) == 0 {
		return err
	}
	ppl, err := au.people(ctx)
	if err != nil {
		return err
	}
	byOwner := map[string][]string{}
	for _, imp := range items {
		owner := deref(imp.OwnerEmployeeID)
		if owner == "" {
			owner = deref(imp.EmployeeID)
		}
		if owner == "" && imp.DepartmentID != nil {
			owner = deref(ppl.depts[*imp.DepartmentID].HeadEmployeeID)
		}
		if owner != "" {
			byOwner[owner] = append(byOwner[owner], imp.Title)
		}
	}
	for owner, titles := range byOwner {
		if !au.claim(ctx, reminderKindImprove, owner, today) {
			continue
		}
		_, body := domain.ComposeTaskNotice(titles, 0, true)
		au.notify(ctx, Notice{UserIDs: ppl.userIDs(owner), Type: NotifImprovement, Href: hrefImprovements,
			Title: fmt.Sprintf("%d item perbaikan lewat tenggat", len(titles)), Body: body})
	}
	return nil
}

// kpiSnapshotDue: on the 1st the previous month's snapshot runs (drafts)
// and every department head is asked to fill the supervisor rubric.
func (au *Automations) kpiSnapshotDue(ctx context.Context, now time.Time) error {
	local := now.In(domain.Jakarta)
	if local.Day() != 1 {
		return nil
	}
	today := domain.DayKey(now, domain.Jakarta)
	prev := domain.Period{Year: local.Year(), Month: int(local.Month())}.Previous()
	if !au.claim(ctx, reminderKindKPISnap, prev.Key(), today) {
		return nil
	}
	sum, err := au.kpi.Snapshot(ctx, prev)
	if err != nil {
		return err
	}
	ppl, err := au.people(ctx)
	if err != nil {
		return err
	}
	for _, d := range ppl.depts {
		if d.HeadEmployeeID == nil || !au.claim(ctx, reminderKindKPIRubric, *d.HeadEmployeeID, today) {
			continue
		}
		au.notify(ctx, Notice{UserIDs: ppl.userIDs(*d.HeadEmployeeID), Type: NotifKPIDue, Href: hrefKPI,
			Title: "Snapshot KPI " + prev.Label() + " siap",
			Body: fmt.Sprintf("%d scorecard draft dibuat. Isi penilaian atasan untuk tim %s sebelum HRD memfinalkan.",
				sum.ScorecardsUpserted, d.Name)})
	}
	return nil
}
