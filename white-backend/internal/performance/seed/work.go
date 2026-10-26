package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// seedDays of task, logbook and recap history: a year, so the dashboard
// trends have something to show.
const seedDays = 365

func (s *seeder) today() string { return domain.DayKey(s.now, domain.Jakarta) }

func (s *seeder) start() string {
	return domain.DayKey(s.now.AddDate(0, 0, -seedDays), domain.Jakarta)
}

// progress is 0 for the oldest seeded day and 1 for today: the demo team
// gets a little better over the year.
func (s *seeder) progress(day string) float64 {
	age := domain.DaysBetween(day, s.today())
	return max(0, min(1, 1-float64(age)/seedDays))
}

// bias is a stable -1..1 lean per department (some teams run tighter).
func bias(deptID string) float64 { return float64(pick("bias"+deptID, 201)-100) / 100 }

// at is a WIB wall time on day.
func at(day string, hour, minute int) time.Time {
	d, _ := domain.ParseDay(day)
	return time.Date(d.Year(), d.Month(), d.Day(), hour, minute, 0, 0, domain.Jakarta).UTC()
}

func (s *seeder) name(id *string) string {
	if id == nil {
		return ""
	}
	for _, e := range s.staff {
		if e.ID == *id {
			return e.Name
		}
	}
	return ""
}

// inserted runs an INSERT ... ON CONFLICT DO NOTHING RETURNING id and
// reports whether a row was written.
func (s *seeder) inserted(ctx context.Context, sql string, args ...any) (bool, error) {
	var id string
	err := s.pool.QueryRow(ctx, sql, args...).Scan(&id)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// ─── tasks ───────────────────────────────────────────────────

func (s *seeder) tasks(ctx context.Context) error {
	start := s.start()
	for _, d := range s.depts {
		members := s.members(d.ID)
		if len(members) == 0 {
			continue
		}
		for n, t := range profileFor(d).tasks {
			if err := s.seedTask(ctx, d, members, fmt.Sprintf("seed_task_%s_%d", d.ID, n), t, start); err != nil {
				return err
			}
		}
	}
	return s.personalTasks(ctx)
}

// personalTasks gives everyone without a department task two of their own
// (90 days of history), so every "Tugas saya" page has work on it.
func (s *seeder) personalTasks(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT assignee_employee_id FROM performance.department_tasks
		WHERE assignee_employee_id IS NOT NULL AND id NOT LIKE 'seed_task_p_%'`)
	if err != nil {
		return err
	}
	busy := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		busy[id] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	start := domain.DayKey(s.now.AddDate(0, 0, -90), domain.Jakarta)
	own := []taskSeed{
		{"Update progres harian ke atasan", "daily", "normal", "", 0, []string{"Tulis capaian hari ini", "Catat kendala"}},
		{"Rangkum pekerjaan minggu ini", "weekly", "normal", "", 5, nil},
	}
	for _, d := range s.depts {
		for _, m := range s.members(d.ID) {
			if busy[m.ID] {
				continue
			}
			for n, t := range own {
				if err := s.seedTask(ctx, d, []domain.Employee{m}, fmt.Sprintf("seed_task_p_%s_%d", m.ID, n), t, start); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// seedTask inserts a task (once) for one of members and its occurrences
// from start to today.
func (s *seeder) seedTask(ctx context.Context, d domain.Department, members []domain.Employee, taskID string, t taskSeed, start string) error {
	today := s.today()
	created := at(start, 8, 0)
	head := d.HeadEmployeeID
	assignee := members[pick(taskID, len(members))]
	sched := domain.TaskSchedule{Recurrence: t.recurrence, CreatedDate: &start}
	var weekly, monthly *int
	switch t.recurrence {
	case "weekly":
		day := t.day
		weekly, sched.WeeklyDay = &day, &day
	case "monthly":
		day := t.day
		monthly, sched.MonthlyDay = &day, &day
	}
	var kpi *string
	var contribution *float64
	if t.kpi != "" {
		id, one := s.ids[t.kpi], 1.0
		kpi, contribution = &id, &one
	}
	ok, err := s.inserted(ctx, `INSERT INTO performance.department_tasks (id, department_id, assignee_employee_id, title, description,
		recurrence, weekly_day, monthly_day, priority, kpi_indicator_id, kpi_contribution, review_required, created_by_employee_id,
		created_by_user_id, created_by_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, true, $12, $13, $14, $15, $15) ON CONFLICT (id) DO NOTHING RETURNING id`,
		taskID, d.ID, assignee.ID, t.title, "Tugas rutin "+d.Name+".", t.recurrence, weekly, monthly, t.priority, kpi, contribution,
		head, seedUser, firstNonEmpty(s.name(head), "Seeder"), created)
	if err != nil {
		return err
	}
	subtasks := domain.SplitSubtaskWeights(t.subtasks)
	if ok {
		for i, st := range subtasks {
			if err := s.exec(ctx, `INSERT INTO performance.department_task_subtasks (id, task_id, title, weight, sort_order) VALUES ($1, $2, $3, $4, $5)`,
				fmt.Sprintf("%s_st%d", taskID, i), taskID, st.Title, st.Weight, i); err != nil {
				return err
			}
		}
	} else if err := s.exec(ctx, `UPDATE performance.department_tasks SET created_at = LEAST(created_at, $2) WHERE id = $1`, taskID, created); err != nil {
		return err
	}
	// Assignee comes from the stored task on a re-run.
	if err := s.pool.QueryRow(ctx, `SELECT assignee_employee_id FROM performance.department_tasks WHERE id = $1`, taskID).Scan(&assignee.ID); err != nil {
		return err
	}
	for _, m := range members {
		if m.ID == assignee.ID {
			assignee = m
		}
	}
	for _, day := range domain.OccurrenceDates(sched, start, today) {
		if err := s.occurrence(ctx, d.ID, taskID, day, today, assignee, head, len(subtasks), kpi != nil); err != nil {
			return err
		}
	}
	return nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// occurrence writes one dated occurrence with a plausible history.
func (s *seeder) occurrence(ctx context.Context, deptID, taskID, day, today string, assignee domain.Employee, head *string, subtasks int, kpi bool) error {
	id := "seed_occ_" + taskID + "_" + day
	r := pick(id, 100)
	status := domain.StatusPending
	recent := day >= domain.DayKey(s.now.AddDate(0, 0, -3), domain.Jakarta)
	old := domain.DaysBetween(day, today) > 30
	prog, lean := s.progress(day), bias(deptID)
	switch {
	case day == today && r < 50:
	case day == today:
		status = domain.StatusInProgress
	case old:
		// Older history is mostly closed; a few items were never finished.
		rejected := int(3 + (1-prog)*4 - lean*2)
		switch {
		case r < rejected:
			status = domain.StatusRejected
		case r < rejected+2+int((1-prog)*3):
		case r < rejected+3+int((1-prog)*3):
			status = domain.StatusInProgress
		default:
			status = domain.StatusApproved
		}
	case r < 70:
		status = domain.StatusApproved
	case r < 78:
		status = domain.StatusRejected
	case r < 86 && recent:
		status = domain.StatusDone
	case r < 86:
		status = domain.StatusApproved
	case r < 93:
		status = domain.StatusInProgress
	}
	var doneAt, reviewedAt, startedAt *time.Time
	var doneBy, doneByName, reviewedBy, reviewedByName, notes *string
	var progress *int
	st := at(day, 9, r%50)
	startedAt = &st
	if status == domain.StatusPending {
		startedAt = nil
	}
	if status == domain.StatusApproved || status == domain.StatusDone || status == domain.StatusRejected {
		hour := 10 + r%8
		finishDay := day
		late := r%5 == 0 // finished a day late now and then
		if old {
			late = float64(pick(id+"late", 100)) < 20-prog*10-lean*6
		}
		if late {
			finishDay = domain.DayKey(at(day, 12, 0).AddDate(0, 0, 1+pick(id+"lag", 3)), domain.Jakarta)
		}
		t := at(finishDay, hour, r%60)
		doneAt, doneBy, doneByName = &t, &assignee.ID, &assignee.Name
	}
	if status == domain.StatusApproved || status == domain.StatusRejected {
		wait := 2 * time.Hour
		if old {
			// Review turnaround: hours mostly, a slow week now and then.
			hours := 1 + pick(id+"turn", 20) + int((1-prog)*10)
			if pick(id+"slow", 12) == 0 {
				hours += 48 + pick(id+"slowh", 48)
			}
			wait = time.Duration(hours) * time.Hour
		}
		t := doneAt.Add(wait)
		reviewedAt, reviewedBy = &t, head
		name := firstNonEmpty(s.name(head), "HRD")
		reviewedByName = &name
	}
	if status == domain.StatusRejected {
		n := []string{"Caption belum sesuai brief klien, mohon revisi.", "Data laporan belum lengkap, tambahkan sumbernya.",
			"Hasil belum sesuai standar QC, cek ulang sebelum kirim."}[r%3]
		notes = &n
	}
	if status == domain.StatusInProgress && subtasks == 0 {
		p := 20 + r%60
		progress = &p
	}
	ok, err := s.inserted(ctx, `INSERT INTO performance.department_task_occurrences (id, task_id, occurrence_date, status, progress_percent,
		started_at, done_at, done_by, done_by_name, review_notes, reviewed_by, reviewed_by_name, reviewed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) ON CONFLICT (task_id, occurrence_date) DO NOTHING RETURNING id`,
		id, taskID, day, status, progress, startedAt, doneAt, doneBy, doneByName, notes, reviewedBy, reviewedByName, reviewedAt)
	if err != nil || !ok {
		return err
	}
	for i := 0; i < subtasks; i++ {
		checked := status == domain.StatusApproved || status == domain.StatusDone || status == domain.StatusRejected ||
			(status == domain.StatusInProgress && i == 0)
		if err := s.exec(ctx, `INSERT INTO performance.department_task_occurrence_items (occurrence_id, subtask_id, is_checked, checked_at, checked_by_name)
			VALUES ($1, $2, $3, $4, $5)`, id, fmt.Sprintf("%s_st%d", taskID, i), checked, startedAt, assignee.Name); err != nil {
			return err
		}
	}
	update := func(n int, kind string, note *string, when *time.Time, author domain.Employee) error {
		if when == nil {
			return nil
		}
		return s.exec(ctx, `INSERT INTO performance.department_task_updates (id, occurrence_id, kind, note, author_employee_id, author_name, detail, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, fmt.Sprintf("%s_u%d", id, n), id, kind, note, author.ID, author.Name,
			fmt.Sprintf(`{"status": %q}`, status), *when)
	}
	if status == domain.StatusInProgress {
		note := "Sedang dikerjakan, target selesai sore ini."
		if err := update(0, domain.UpdateProgress, &note, startedAt, assignee); err != nil {
			return err
		}
	}
	if err := update(1, domain.UpdateStatus, nil, doneAt, assignee); err != nil {
		return err
	}
	if reviewedAt != nil {
		reviewer := domain.Employee{Name: deref(reviewedByName)}
		if head != nil {
			reviewer.ID = *head
		}
		if err := update(2, domain.UpdateReview, notes, reviewedAt, reviewer); err != nil {
			return err
		}
	}
	if status == domain.StatusApproved && kpi {
		p := domain.PeriodOf(day)
		return s.exec(ctx, `INSERT INTO performance.department_task_kpi_contributions (occurrence_id, indicator_id, department_id, employee_id, period_year, period_month, amount)
			SELECT $1, t.kpi_indicator_id, t.department_id, t.assignee_employee_id, $2, $3, t.kpi_contribution
			FROM performance.department_tasks t WHERE t.id = $4 ON CONFLICT DO NOTHING`, id, p.Year, p.Month, taskID)
	}
	return nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ─── logbook ─────────────────────────────────────────────────

var logbookRejectNotes = []string{"Bukti checklist belum dilampirkan, mohon lengkapi.", "Ada item wajib yang terlewat, cek ulang sebelum kirim."}

func (s *seeder) logbook(ctx context.Context) error {
	start, today := s.start(), s.today()
	for _, d := range s.depts {
		members := s.members(d.ID)
		if len(members) == 0 {
			continue
		}
		p := profileFor(d)
		tplID := "seed_lbt_" + d.ID
		if err := s.exec(ctx, `INSERT INTO performance.logbook_templates (id, department_id, name, description, frequency, created_by, updated_by)
			VALUES ($1, $2, $3, $4, 'daily', $5, $5) ON CONFLICT (id) DO NOTHING`,
			tplID, d.ID, "Checklist harian "+d.Name, "Rutinitas harian tim "+d.Name+".", seedUser); err != nil {
			return err
		}
		weights := make([]float64, len(p.checklist))
		for i, item := range p.checklist {
			weights[i] = 1
			if i == 1 {
				weights[i] = 2
			}
			if err := s.exec(ctx, `INSERT INTO performance.logbook_template_items (id, template_id, title, weight, is_required, sort_order)
				VALUES ($1, $2, $3, $4, true, $5) ON CONFLICT (id) DO NOTHING`, fmt.Sprintf("%s_%d", tplID, i), tplID, item, weights[i], i); err != nil {
				return err
			}
		}
		for _, m := range members {
			for _, day := range domain.EachDate(start, today) {
				dt, _ := domain.ParseDay(day)
				if !domain.IsWorkday(dt) {
					continue
				}
				if err := s.entry(ctx, tplID, d, m, day, p.checklist, weights); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *seeder) entry(ctx context.Context, tplID string, d domain.Department, m domain.Employee, day string, items []string, weights []float64) error {
	id := "seed_lbe_" + m.ID + "_" + day
	r := pick(id, 100)
	prog, lean := s.progress(day), bias(d.ID)
	if float64(r) < 6+(1-prog)*10-lean*4 {
		return nil // skipped that day
	}
	missRate := 20 - int(prog*6) - int(lean*5) // share of unticked items, in percent
	status := domain.LogbookDraft
	today := day == s.today()
	switch {
	case today && r < 55:
		// today's checklist is still being filled
	case today:
		status = domain.LogbookSubmitted
	case r < 70:
		status = domain.LogbookReviewed
	case r < 78:
		status = domain.LogbookRejected
	case r < 92:
		status = domain.LogbookSubmitted
	}
	submitted := at(day, 17, r%45)
	if today {
		submitted = at(day, 11, r%45)
	}
	var submittedAt, reviewedAt *time.Time
	var reviewer, reviewNotes *string
	if status != domain.LogbookDraft {
		submittedAt = &submitted
	}
	if status == domain.LogbookReviewed || status == domain.LogbookRejected {
		t := submitted.Add(16 * time.Hour)
		reviewedAt, reviewer = &t, strPtr(seedUser)
	}
	if status == domain.LogbookRejected {
		reviewNotes = &logbookRejectNotes[r%2]
	}
	title := domain.LogbookTitle("Checklist harian "+d.Name, day, "")
	ok, err := s.inserted(ctx, `INSERT INTO performance.logbook_entries (id, template_id, department_id, employee_id, entry_date, title, status,
		submitted_by, submitted_at, reviewed_by, reviewed_at, review_notes, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $8, $13, $13) ON CONFLICT DO NOTHING RETURNING id`,
		id, tplID, d.ID, m.ID, day, title, status, seedUser, submittedAt, reviewer, reviewedAt, reviewNotes, at(day, 8, 30))
	if err != nil || !ok {
		return err
	}
	states := make([]domain.LogbookItemState, len(items))
	for i, item := range items {
		checked := pick(fmt.Sprintf("%s_%d", id, i), 100) >= missRate
		states[i] = domain.LogbookItemState{Checked: checked, Weight: weights[i]}
		var checkedAt *time.Time
		if checked {
			t := at(day, 10+i, 0)
			checkedAt = &t
		}
		if err := s.exec(ctx, `INSERT INTO performance.logbook_entry_items (id, entry_id, template_item_id, title, weight, is_required, sort_order, is_checked, checked_at)
			VALUES ($1, $2, $3, $4, $5, true, $6, $7, $8)`, fmt.Sprintf("%s_%d", id, i), id, fmt.Sprintf("%s_%d", tplID, i), item, weights[i], i,
			checked, checkedAt); err != nil {
			return err
		}
	}
	completion, kpi := domain.LogbookScore(states)
	return s.exec(ctx, `UPDATE performance.logbook_entries SET completion_percentage = $2, kpi_score = $3 WHERE id = $1`, id, completion, kpi)
}

func strPtr(s string) *string { return &s }

// ─── KPI ─────────────────────────────────────────────────────

// kpiPeriods are the months that get scorecards: the last twelve months
// and this month, oldest first.
func (s *seeder) kpiPeriods() []domain.Period {
	return domain.MonthsUpTo(s.today(), 13)
}

// quarterBack is the quarter n quarters before the current one.
func (s *seeder) quarterBack(n int) (year, quarter int) {
	local := s.now.In(domain.Jakarta)
	idx := local.Year()*4 + domain.QuarterOf(int(local.Month())) - 1 - n
	return idx / 4, idx%4 + 1
}

func (s *seeder) kpi(ctx context.Context) error {
	ctx = admin(ctx)
	k := s.w.Services.KPI
	periods := s.kpiPeriods()
	for pi, p := range periods {
		prog := float64(pi) / float64(len(periods)-1)
		if _, err := k.Snapshot(ctx, p); err != nil {
			return err
		}
		for _, d := range s.depts {
			prof := profileFor(d)
			for _, m := range s.members(d.ID) {
				key := m.ID + p.Key()
				rubric := float64(3 + pick(key+"rubric", 3))
				if _, _, err := k.SaveManual(ctx, app.ManualInput{EmployeeID: m.ID, Period: p, Value: rubric,
					Notes: strPtr("Penilaian atasan bulanan")}); err != nil && !isFinal(err) {
					return err
				}
				for _, ind := range prof.indicators {
					trend := prog
					if ind.direction == domain.LowerBetter {
						trend = 1 - prog
					}
					v := ind.base + ind.spread*(0.65*float64(pick(key+ind.code, 100))/100+0.35*trend)
					if ind.direction == domain.Boolean {
						v = float64(int(v))
					} else {
						v = domain.Round2(v)
					}
					if _, _, err := k.SaveManual(ctx, app.ManualInput{EmployeeID: m.ID, IndicatorID: s.ids[ind.code], Period: p, Value: v}); err != nil && !isFinal(err) {
						return err
					}
				}
			}
		}
	}
	return s.finalizeScorecards(ctx)
}

// finalizeScorecards closes most months before last month and half of
// last month, the way HR signs scorecards off.
func (s *seeder) finalizeScorecards(ctx context.Context) error {
	periods := s.kpiPeriods()
	last := periods[len(periods)-2]
	return s.exec(ctx, `UPDATE performance.kpi_scorecards SET status = 'final', reviewed_by = $1,
			reviewed_at = make_timestamptz(period_year, period_month, 1, 9, 0, 0, 'Asia/Jakarta') + interval '1 month 5 days'
		WHERE status = 'draft' AND (
			(make_date(period_year, period_month, 1) < make_date($2, $3, 1) AND abs(hashtext(employee_id || period_month::text)) % 10 <> 0)
			OR (period_year = $2 AND period_month = $3 AND abs(hashtext(employee_id)) % 2 = 0))`,
		seedUser, last.Year, last.Month)
}

// isFinal: a re-run meets finalized scorecards; those stay as they are.
func isFinal(err error) bool { return apperr.Is(err, apperr.Conflict) }
