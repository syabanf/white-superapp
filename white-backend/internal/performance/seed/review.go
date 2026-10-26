package seed

import (
	"context"
	"fmt"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
)

var selfAssessments = []string{
	"Kuartal ini saya fokus menjaga kualitas deliverable klien. Rencana perbaikan: lebih disiplin update progres harian.",
	"Target utama tercapai, tetapi beberapa revisi terlambat. Rencana perbaikan: memecah pekerjaan besar menjadi sub-task.",
	"Saya banyak belajar tools baru dan membantu rekan tim. Rencana perbaikan: dokumentasi proses agar mudah diteruskan.",
}

var reviewerNotes = []string{
	"Kualitas kerja konsisten. Perlu lebih proaktif mengomunikasikan kendala ke AM sebelum tenggat.",
	"Kolaborasi dengan tim sangat baik. Tingkatkan ketelitian laporan agar revisi dari klien berkurang.",
	"Inisiatif tinggi. Kelola prioritas lebih rapi supaya tugas mingguan tidak menumpuk di akhir pekan.",
}

// review opens the previous quarter's cycle and fills part of it (self
// assessments, behavioural ratings, reviewer notes and a few final reviews
// whose notes become improvement items), and the three quarters before it
// fully finalized and closed. Each cycle is created once.
func (s *seeder) review(ctx context.Context) error {
	for n := 4; n >= 1; n-- {
		year, q := s.quarterBack(n)
		if err := s.reviewCycle(ctx, year, q, n > 1); err != nil {
			return err
		}
	}
	return nil
}

func (s *seeder) reviewCycle(ctx context.Context, year, q int, closed bool) error {
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM performance.review_cycles WHERE period_year = $1 AND period_quarter = $2)`,
		year, q).Scan(&exists); err != nil || exists {
		return err
	}
	ctx = admin(ctx)
	rv := s.w.Services.Reviews
	cycle, _, err := rv.OpenCycle(ctx, year, q, "")
	if err != nil {
		return err
	}
	reviews, _, err := rv.ListReviews(ctx, cycle.ID)
	if err != nil {
		return err
	}
	for i, r := range reviews {
		n := pick(r.EmployeeID+cycle.ID, 100)
		if closed {
			n = 0
		}
		if n < 60 {
			if err := s.exec(ctx, `UPDATE performance.performance_reviews SET self_assessment = $2, employee_sign_date = $3 WHERE id = $1`,
				r.ID, selfAssessments[i%len(selfAssessments)], cycle.DueDate); err != nil {
				return err
			}
		}
		if n >= 75 {
			continue // still waiting for the reviewer
		}
		detail, err := rv.GetReview(ctx, r.ID)
		if err != nil {
			return err
		}
		for j, it := range detail.Items {
			score := float64(2 + pick(fmt.Sprintf("%s%d", r.ID, j), 4))
			if score < 3 && pick(r.ID+"low", 3) > 0 {
				score = 3
			}
			if _, err := rv.Act(ctx, r.ID, app.ReviewAction{Action: domain.ReviewRateItem, ItemID: it.ID, Score: &score}); err != nil {
				return err
			}
		}
		project := float64(60 + pick(r.ID+"project", 38))
		note := reviewerNotes[i%len(reviewerNotes)]
		if _, err := rv.Act(ctx, r.ID, app.ReviewAction{Action: domain.ReviewReviewerNotes, Text: &note, ProjectScore: &project}); err != nil {
			return err
		}
		if _, err := rv.Act(ctx, r.ID, app.ReviewAction{Action: domain.ReviewSign}); err != nil {
			return err
		}
		if n < 30 {
			if _, err := rv.Act(ctx, r.ID, app.ReviewAction{Action: domain.ReviewFinalize}); err != nil {
				return err
			}
		}
	}
	if closed {
		return rv.CloseCycle(ctx, cycle.ID)
	}
	return nil
}

// improvements files a few manual items per department and per person.
func (s *seeder) improvements(ctx context.Context) error {
	deptItems := []string{"Standarkan template brief klien", "Kurangi revisi bolak-balik dengan klien", "Rapikan folder aset bersama"}
	personal := []struct{ title, desc string }{
		{"Kirim update progres sebelum jam 12", "Atasan meminta update tengah hari agar kendala cepat terlihat."},
		{"Ikut kelas Google Ads lanjutan", "Menambah kemampuan optimasi kampanye untuk klien e-commerce."},
		{"Dokumentasikan SOP pekerjaan rutin", "Supaya pekerjaan bisa diteruskan saat cuti."},
	}
	statuses := []string{domain.ImpOpen, domain.ImpInProgress, domain.ImpDone}
	today := s.now.In(domain.Jakarta)
	for i, d := range s.depts {
		members := s.members(d.ID)
		owner := d.HeadEmployeeID
		due := today.AddDate(0, 0, 14+i).Format(domain.IsoDate)
		if err := s.exec(ctx, `INSERT INTO performance.improvements (id, scope, department_id, source, title, description, owner_employee_id, due_date, status, created_by, updated_by)
			VALUES ($1, 'department', $2, 'manual', $3, $4, $5, $6, 'open', $7, $7) ON CONFLICT (id) DO NOTHING`,
			"seed_imp_dept_"+d.ID, d.ID, deptItems[i%len(deptItems)], "Hasil diskusi tim "+d.Name+".", owner, due, seedUser); err != nil {
			return err
		}
		for j, m := range members {
			if j >= 2 {
				break
			}
			it := personal[(i+j)%len(personal)]
			status := statuses[pick(m.ID, len(statuses))]
			due := today.AddDate(0, 0, -3+7*j).Format(domain.IsoDate) // the first one is overdue
			if err := s.exec(ctx, `INSERT INTO performance.improvements (id, scope, employee_id, department_id, source, title, description, owner_employee_id,
				due_date, status, completed_at, created_by, updated_by)
				VALUES ($1, 'employee', $2, $3, 'manual', $4, $5, $2, $6, $7, CASE WHEN $7 = 'done' THEN now() END, $8, $8) ON CONFLICT (id) DO NOTHING`,
				"seed_imp_emp_"+m.ID, m.ID, d.ID, it.title, it.desc, due, status, seedUser); err != nil {
				return err
			}
		}
	}
	return nil
}

// recaps generates last month's recaps per department and employee (with
// AI suggestions stored as suggested improvements).
func (s *seeder) recaps(ctx context.Context) error {
	ctx = admin(ctx)
	prev := s.kpiPeriods()
	p := prev[len(prev)-2]
	rc := s.w.Services.Recaps
	for _, d := range s.depts {
		if len(s.members(d.ID)) == 0 {
			continue
		}
		if _, _, err := rc.RegenerateMonthly(ctx, domain.ScopeDepartment, d.ID, p); err != nil {
			return err
		}
	}
	for _, e := range s.staff {
		if _, _, err := rc.RegenerateMonthly(ctx, domain.ScopeEmployee, e.ID, p); err != nil {
			return err
		}
	}
	return nil
}
