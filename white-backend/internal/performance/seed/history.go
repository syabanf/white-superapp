package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
)

// ─── daily recaps ────────────────────────────────────────────

// dailyRecaps generates the recap of most workdays of every employee from
// that day's tasks and logbook, today included: coverage climbs over the
// year and differs a little per department. Days that already have a
// recap with content are skipped, so a re-run only fills new days.
func (s *seeder) dailyRecaps(ctx context.Context) error {
	// Rows with nothing in them (placeholders of an older seed, or a day
	// generated before any work was recorded) and no note of their own.
	if err := s.exec(ctx, `DELETE FROM performance.daily_recaps WHERE items = '{"tasks": [], "logbook": []}'::jsonb AND notes = ''`); err != nil {
		return err
	}
	rows, err := s.pool.Query(ctx, `SELECT employee_id, recap_date FROM performance.daily_recaps`)
	if err != nil {
		return err
	}
	done := map[string]bool{}
	for rows.Next() {
		var emp string
		var day time.Time
		if err := rows.Scan(&emp, &day); err != nil {
			return err
		}
		done[emp+day.Format(domain.IsoDate)] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	ctx = admin(ctx)
	rc := s.w.Services.Recaps
	today := s.today()
	for _, e := range s.staff {
		dept := ""
		if e.DepartmentID != nil {
			dept = *e.DepartmentID
		}
		for _, day := range domain.EachDate(s.start(), today) {
			t, _ := domain.ParseDay(day)
			if !domain.IsWorkday(t) || done[e.ID+day] {
				continue
			}
			if float64(pick("seed_dr_"+e.ID+"_"+day, 100)) < 4+(1-s.progress(day))*18-bias(dept)*4 {
				continue
			}
			if err := rc.GenerateDaily(ctx, e.ID, day); err != nil {
				return err
			}
			at := time.Date(t.Year(), t.Month(), t.Day(), domain.DailyRecapHour, domain.DailyRecapMinute, 0, 0, domain.Jakarta).UTC()
			if day == today {
				at = s.now
			}
			if err := s.exec(ctx, `UPDATE performance.daily_recaps SET generated_at = $3, created_at = $3, updated_at = $3
				WHERE employee_id = $1 AND recap_date = $2`, e.ID, day, at); err != nil {
				return err
			}
		}
	}
	return nil
}

// ─── 360 feedback ────────────────────────────────────────────

// feedback runs two 360 cycles: the quarter before last (closed, every
// form in) and last quarter (active, most forms in), each created once.
func (s *seeder) feedback(ctx context.Context) error {
	for n := 2; n >= 1; n-- {
		year, q := s.quarterBack(n)
		if err := s.feedbackCycle(ctx, year, q, n == 2); err != nil {
			return err
		}
	}
	return nil
}

func (s *seeder) feedbackCycle(ctx context.Context, year, q int, closed bool) error {
	label := domain.QuarterLabel(year, q)
	name := "Feedback 360 " + label
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM performance.feedback_cycles WHERE name = $1)`, name).Scan(&exists); err != nil || exists {
		return err
	}
	ctx = admin(ctx)
	fb := s.w.Services.Feedback
	start, end := domain.QuarterRange(year, q)
	kpiW, fbW := 60.0, 40.0
	cycle, err := fb.SaveCycle(ctx, domain.FeedbackCycle{Name: name, PeriodLabel: label, StartDate: &start, EndDate: &end,
		Status: domain.FeedbackActive, KPIWeight: &kpiW, FeedbackWeight: &fbW, IsAnonymous: true, AllowSelfAssessment: true})
	if err != nil {
		return err
	}
	if _, err := fb.AutoAssign(ctx, cycle.ID, 2); err != nil {
		return err
	}
	cats, err := s.w.Services.Feedback.ListCategories(ctx)
	if err != nil {
		return err
	}
	list, err := fb.ListAssignments(ctx, cycle.ID, "", "", "")
	if err != nil {
		return err
	}
	submitted := at(end, 15, 0).AddDate(0, 0, 7)
	for _, a := range list {
		r := pick(a.ID, 100)
		if !closed && r < 22 {
			continue // still pending
		}
		for ci, c := range cats {
			for _, cr := range c.Criteria {
				// Each category leans its own way so the radar has a shape.
				rating := 3 + pick(a.ID+cr.ID, 3)
				if ci%2 == 1 && pick(a.ID+cr.ID+"dip", 3) == 0 {
					rating--
				}
				if !closed && ci == 0 && rating < 5 && pick(a.ID+"lift", 2) == 0 {
					rating++
				}
				if err := s.exec(ctx, `INSERT INTO performance.feedback_responses (id, assignment_id, criteria_id, rating)
					VALUES ($1, $2, $3, $4) ON CONFLICT (assignment_id, criteria_id) DO NOTHING`,
					fmt.Sprintf("seed_fr_%s_%s", a.ID, cr.ID), a.ID, cr.ID, rating); err != nil {
					return err
				}
			}
		}
		if err := s.exec(ctx, `UPDATE performance.feedback_assignments SET status = 'approved', started_at = $2, submitted_at = $2 WHERE id = $1`,
			a.ID, submitted.Add(-time.Duration(r)*time.Hour)); err != nil {
			return err
		}
	}
	if _, _, err := fb.GenerateSummaries(ctx, cycle.ID); err != nil {
		return err
	}
	if closed {
		return s.exec(ctx, `UPDATE performance.feedback_cycles SET status = 'closed' WHERE id = $1`, cycle.ID)
	}
	return nil
}

// ─── improvement history ─────────────────────────────────────

// ageAutoItems dates the items filed by review and 360 automation to a
// couple of weeks after their cycle ended (the seed runs them all today),
// and closes most of those from closed cycles. Rows already moved are left
// alone, so a re-run changes nothing.
func (s *seeder) ageAutoItems(ctx context.Context) error {
	if err := s.exec(ctx, `UPDATE performance.improvements m SET
			created_at = (rc.end_date + 12)::timestamp AT TIME ZONE 'Asia/Jakarta' + interval '10 hours',
			updated_at = (rc.end_date + 12)::timestamp AT TIME ZONE 'Asia/Jakarta' + interval '10 hours',
			due_date = rc.end_date + 45,
			status = CASE WHEN rc.status = 'closed' AND abs(hashtext(m.id)) % 10 < 7 THEN 'done' ELSE m.status END,
			completed_at = CASE WHEN rc.status = 'closed' AND abs(hashtext(m.id)) % 10 < 7
				THEN (rc.end_date + 30 + abs(hashtext(m.id)) % 25)::timestamp AT TIME ZONE 'Asia/Jakarta' + interval '14 hours' END
		FROM performance.performance_reviews pr
		JOIN performance.review_cycles rc ON rc.id = pr.cycle_id
		WHERE m.source = 'review' AND split_part(m.source_ref, ':', 1) = pr.id AND m.created_by = $1
			AND m.created_at > (rc.end_date + 30)::timestamp`, seedUser); err != nil {
		return err
	}
	// Recap suggestions: HR has already accepted or declined a part of them.
	if err := s.exec(ctx, `UPDATE performance.improvements SET
			status = CASE WHEN abs(hashtext(id)) % 10 < 3 THEN 'declined' ELSE 'open' END, updated_at = now()
		WHERE source = 'ai' AND source_ref IS NOT NULL AND status = 'suggested' AND abs(hashtext(id)) % 10 < 6`); err != nil {
		return err
	}
	return s.exec(ctx, `UPDATE performance.improvements m SET
			created_at = (fc.end_date + 10)::timestamp AT TIME ZONE 'Asia/Jakarta' + interval '10 hours',
			updated_at = (fc.end_date + 10)::timestamp AT TIME ZONE 'Asia/Jakarta' + interval '10 hours',
			due_date = fc.end_date + 40,
			status = CASE WHEN fc.status = 'closed' AND abs(hashtext(m.id)) % 10 < 6 THEN 'done' ELSE m.status END,
			completed_at = CASE WHEN fc.status = 'closed' AND abs(hashtext(m.id)) % 10 < 6
				THEN (fc.end_date + 25 + abs(hashtext(m.id)) % 25)::timestamp AT TIME ZONE 'Asia/Jakarta' + interval '14 hours' END
		FROM performance.feedback_summaries fs
		JOIN performance.feedback_cycles fc ON fc.id = fs.cycle_id
		WHERE m.source = 'feedback' AND split_part(m.source_ref, ':', 1) = fs.id AND m.created_by = $1
			AND fc.end_date IS NOT NULL AND m.created_at > (fc.end_date + 30)::timestamp`, seedUser)
}

var historyItems = map[string][]string{
	domain.SourceManualItem: {"Rapikan SOP serah terima klien", "Buat checklist QC sebelum kirim", "Jadwalkan sesi berbagi ilmu bulanan",
		"Standarkan penamaan file aset"},
	domain.SourceRetro: {"Kurangi rapat yang tidak perlu", "Perjelas pembagian PIC per klien", "Tetapkan batas revisi per brief"},
	domain.SourceAI: {"Kirim update progres sebelum tengah hari", "Pecah tugas besar menjadi sub-tugas", "Blok waktu fokus tanpa rapat",
		"Catat kendala di logbook harian", "Review ulang brief sebelum mulai desain"},
	domain.SourceTaskReject: {"Perbaiki ketelitian laporan mingguan", "Cek ulang caption sebelum dijadwalkan"},
	domain.SourceReview:     {"Tingkatkan komunikasi kendala ke AM"},
}

var historySources = []string{domain.SourceManualItem, domain.SourceAI, domain.SourceAI, domain.SourceRetro, domain.SourceTaskReject,
	domain.SourceManualItem, domain.SourceReview, domain.SourceAI}

// improvementHistory files two to five items per department per month over
// the last year, with statuses that fit their age.
func (s *seeder) improvementHistory(ctx context.Context) error {
	months := domain.MonthsUpTo(s.today(), 12)
	for _, d := range s.depts {
		members := s.members(d.ID)
		if len(members) == 0 {
			continue
		}
		for _, p := range months {
			n := 2 + pick(d.ID+p.Key(), 4)
			for i := 0; i < n; i++ {
				id := fmt.Sprintf("seed_imp_h_%s_%s_%d", d.ID, p.Key(), i)
				if err := s.historyItem(ctx, id, d, members, p, i); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *seeder) historyItem(ctx context.Context, id string, d domain.Department, members []domain.Employee, p domain.Period, i int) error {
	source := historySources[pick(id+"src", len(historySources))]
	titles := historyItems[source]
	title := titles[pick(id+"t", len(titles))]
	day := min(28, 1+pick(id+"d", 28))
	created := time.Date(p.Year, time.Month(p.Month), day, 10, pick(id+"m", 60), 0, 0, domain.Jakarta)
	if created.After(s.now) {
		return nil
	}
	age := int(s.now.Sub(created).Hours() / 24)
	due := created.AddDate(0, 0, 14+pick(id+"due", 22))
	r := pick(id+"st", 100)
	status := domain.ImpOpen
	switch {
	case source == domain.SourceAI && age < 21 && r < 55:
		status = domain.ImpSuggested
	case source == domain.SourceAI && r < 30-int(bias(d.ID)*10):
		status = domain.ImpDeclined
	case age > 45 && r < 82:
		status = domain.ImpDone
	case age > 45 && r < 90:
		status = domain.ImpDropped
	case r < 45:
		status = domain.ImpInProgress
	case age > 20 && r < 75:
		status = domain.ImpDone
	}
	updated := created.Add(time.Duration(1+pick(id+"u", 72)) * time.Hour)
	var completed *time.Time
	if status == domain.ImpDone {
		c := created.AddDate(0, 0, 5+pick(id+"c", 30))
		if c.After(s.now) {
			c = s.now.Add(-time.Hour)
		}
		completed, updated = &c, c
	}
	scope, empID := domain.ScopeDepartment, (*string)(nil)
	owner := d.HeadEmployeeID
	if i%2 == 1 || source == domain.SourceAI || source == domain.SourceTaskReject {
		m := members[pick(id+"who", len(members))]
		scope, empID, owner = domain.ScopeEmployee, &m.ID, &m.ID
	}
	return s.exec(ctx, `INSERT INTO performance.improvements (id, scope, employee_id, department_id, source, title, description, owner_employee_id,
			due_date, status, completed_at, created_by, updated_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14) ON CONFLICT (id) DO NOTHING`,
		id, scope, empID, d.ID, source, title, "Riwayat perbaikan tim "+d.Name+".", owner, due.Format(domain.IsoDate), status, completed,
		seedUser, created.UTC(), updated.UTC())
}
