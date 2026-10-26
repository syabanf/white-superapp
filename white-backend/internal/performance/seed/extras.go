package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
)

// Extras fill the screens the history does not reach: KPI targets, task
// comments and attachments, every 360 assignment state, retros, this
// month's monthly recaps.

func (s *seeder) targets(ctx context.Context) error {
	year := s.now.In(domain.Jakarta).Year()
	for _, d := range s.depts {
		p := profileFor(d)
		members := s.members(d.ID)
		for i, ind := range p.indicators {
			id, ok := s.ids[ind.code]
			if !ok {
				continue
			}
			target := ind.target
			if ind.direction == domain.LowerBetter {
				target *= 0.9
			} else {
				target *= 1.1
			}
			if err := s.exec(ctx, `INSERT INTO performance.kpi_targets (id, indicator_id, period_year, department_id, target, created_by)
				VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (id) DO NOTHING`,
				fmt.Sprintf("seed_tgt_%s_%d", d.ID, i), id, year, d.ID, domain.Round2(target), seedUser); err != nil {
				return err
			}
			if len(members) > 0 {
				m := members[pick(d.ID+ind.code, len(members))]
				if err := s.exec(ctx, `INSERT INTO performance.kpi_targets (id, indicator_id, period_year, employee_id, target, created_by)
					VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (id) DO NOTHING`,
					fmt.Sprintf("seed_tgt_%s_%d_emp", d.ID, i), id, year, m.ID, domain.Round2(target*1.05), seedUser); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// A 1x1 PNG: the attachment list and preview need a real image.
var tinyPNG = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01,
	0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41, 0x54, 0x78,
	0x9c, 0x63, 0xf8, 0xcf, 0xc0, 0xf0, 0x1f, 0x00, 0x05, 0x00, 0x01, 0xff, 0x89, 0x99, 0x3d, 0x1d, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45,
	0x4e, 0x44, 0xae, 0x42, 0x60, 0x82}

var taskComments = []string{"Sudah saya kerjakan, mohon dicek ya.", "Klien minta revisi kecil di bagian akhir.",
	"Siap, saya lanjutkan besok pagi.", "Terima kasih, hasilnya rapi."}

// taskExtras adds comments to about a third of the last two weeks'
// occurrences and a screenshot to some of them.
func (s *seeder) taskExtras(ctx context.Context) error {
	files := storage.NewLocal(config.Env("PERFORMANCE_UPLOAD_DIR", ".uploads/performance"))
	rows, err := s.pool.Query(ctx, `SELECT o.id, o.occurrence_date, t.assignee_employee_id, t.department_id
		FROM performance.department_task_occurrences o JOIN performance.department_tasks t ON t.id = o.task_id
		WHERE o.id LIKE 'seed_occ_%' AND o.occurrence_date >= $1 AND o.occurrence_date <= $2 AND o.status <> 'pending'`,
		domain.AddDays(s.today(), -14), s.today())
	if err != nil {
		return err
	}
	type occ struct{ id, day, assignee, dept string }
	var list []occ
	for rows.Next() {
		var o occ
		var d time.Time
		if err := rows.Scan(&o.id, &d, &o.assignee, &o.dept); err != nil {
			return err
		}
		o.day = d.Format(domain.IsoDate)
		list = append(list, o)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	heads := map[string]*string{}
	for _, d := range s.depts {
		heads[d.ID] = d.HeadEmployeeID
	}
	for _, o := range list {
		r := pick(o.id+"extra", 9)
		if r > 2 {
			continue
		}
		author := domain.Employee{ID: o.assignee, Name: s.name(&o.assignee)}
		if r == 1 && heads[o.dept] != nil {
			author = domain.Employee{ID: *heads[o.dept], Name: s.name(heads[o.dept])}
		}
		note := taskComments[pick(o.id+"note", len(taskComments))]
		if err := s.exec(ctx, `INSERT INTO performance.department_task_updates (id, occurrence_id, kind, note, author_employee_id, author_name, created_at)
			VALUES ($1, $2, 'comment', $3, $4, $5, $6) ON CONFLICT (id) DO NOTHING`,
			o.id+"_c", o.id, note, author.ID, firstNonEmpty(author.Name, "Tim"), at(o.day, 13, pick(o.id, 60))); err != nil {
			return err
		}
		if r != 0 {
			continue
		}
		key := "tasks/seed/" + o.id + ".png"
		if err := files.Put(ctx, key, tinyPNG); err != nil {
			return err
		}
		if err := s.exec(ctx, `INSERT INTO performance.department_task_attachments (id, occurrence_id, name, mime, size_bytes, storage_path,
				uploaded_by_name, created_at)
			VALUES ($1, $2, $3, 'image/png', $4, $5, $6, $7) ON CONFLICT (id) DO NOTHING`,
			"seed_att_"+o.id, o.id, "bukti-"+o.day+".png", len(tinyPNG), key, firstNonEmpty(author.Name, "Tim"), at(o.day, 14, 0)); err != nil {
			return err
		}
	}
	return nil
}

// feedbackStates spreads the active 360 cycle over every assignment state:
// it needs manager approval, a share waits for it, some forms are half
// filled and one is sent back. Each update moves a fixed subset once.
func (s *seeder) feedbackStates(ctx context.Context) error {
	year, q := s.quarterBack(1)
	name := "Feedback 360 " + domain.QuarterLabel(year, q)
	for _, sql := range []string{
		`UPDATE performance.feedback_cycles SET require_manager_review = true WHERE name = $1`,
		`UPDATE performance.feedback_assignments a SET status = 'submitted' FROM performance.feedback_cycles c
			WHERE c.id = a.cycle_id AND c.name = $1 AND a.status = 'approved' AND abs(hashtext(a.id)) % 6 = 0`,
		`UPDATE performance.feedback_assignments a SET status = 'in_progress', started_at = c.start_date + 60 FROM performance.feedback_cycles c
			WHERE c.id = a.cycle_id AND c.name = $1 AND a.status = 'pending' AND abs(hashtext(a.id)) % 2 = 0`,
		`UPDATE performance.feedback_assignments a SET status = 'rejected', rejection_reason = 'Komentar belum spesifik, mohon dilengkapi.'
			FROM performance.feedback_cycles c
			WHERE c.id = a.cycle_id AND c.name = $1 AND a.status = 'approved' AND a.relationship_type = 'peer' AND abs(hashtext(a.id)) % 15 = 1`,
	} {
		if err := s.exec(ctx, sql, name); err != nil {
			return err
		}
	}
	return nil
}

var retroWell = []string{"Deadline klien terpenuhi", "Kolaborasi lintas tim lancar", "Laporan mingguan tepat waktu", "Klien puas dengan hasil kampanye"}
var retroImprove = []string{"Brief sering berubah di tengah jalan", "Revisi menumpuk di akhir pekan", "Dokumentasi proses belum rapi", "Rapat terlalu banyak"}

// retros files a retro per department for the last three months: older
// ones closed, last month's still a draft.
func (s *seeder) retros(ctx context.Context) error {
	months := domain.MonthsUpTo(s.today(), 4)[:3]
	for _, d := range s.depts {
		if len(s.members(d.ID)) == 0 {
			continue
		}
		for i, p := range months {
			key := d.ID + p.Key()
			status, closedAt := "closed", any(p.Next().Span().Start+" 10:00:00+07")
			if i == len(months)-1 {
				status, closedAt = "draft", nil
			}
			due := p.Next().Span().End
			actions, _ := json.Marshal([]domain.RetroAction{{Description: retroImprove[pick(key+"a", len(retroImprove))], OwnerEmployeeID: d.HeadEmployeeID, DueDate: &due}})
			well := []string{retroWell[pick(key+"w1", len(retroWell))], retroWell[pick(key+"w2", len(retroWell))]}
			if well[0] == well[1] {
				well = well[:1]
			}
			if err := s.exec(ctx, `INSERT INTO performance.retro_sessions (id, department_id, period_year, period_month, went_well, to_improve, actions,
					status, created_by, updated_by, closed_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9, $10) ON CONFLICT (department_id, period_year, period_month) DO NOTHING`,
				"seed_retro_"+key, d.ID, p.Year, p.Month, well, []string{retroImprove[pick(key+"i", len(retroImprove))]}, actions, status,
				seedUser, closedAt); err != nil {
				return err
			}
		}
	}
	return nil
}

// recentRecaps writes this month's monthly recaps.
func (s *seeder) recentRecaps(ctx context.Context) error {
	ctx = admin(ctx)
	rc := s.w.Services.Recaps
	// This month's recaps once: regenerating them refiles AI suggestions.
	cur := s.kpiPeriods()[len(s.kpiPeriods())-1]
	monthly := func(scope, subject string) error {
		var exists bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM performance.monthly_recaps WHERE scope = $1 AND subject_id = $2
			AND period_year = $3 AND period_month = $4)`, scope, subject, cur.Year, cur.Month).Scan(&exists); err != nil || exists {
			return err
		}
		_, _, err := rc.RegenerateMonthly(ctx, scope, subject, cur)
		return err
	}
	for _, d := range s.depts {
		if len(s.members(d.ID)) == 0 {
			continue
		}
		if err := monthly(domain.ScopeDepartment, d.ID); err != nil {
			return err
		}
	}
	for _, e := range s.staff {
		if err := monthly(domain.ScopeEmployee, e.ID); err != nil {
			return err
		}
	}
	return nil
}
