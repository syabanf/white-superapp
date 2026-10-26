package app

import (
	"context"
	"fmt"
	"sort"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// Reviews ports NüHabit payroll/performance.go: quarterly cycles, a draft
// review per employee with behavioural items, sign-off and HRD finalize.
// Finalizing files the reviewer and manager notes as improvement items.
type Reviews struct {
	Deps
	improvements *Improvements
}

func NewReviews(d Deps, imp *Improvements) *Reviews { return &Reviews{d, imp} }

const msgReviewGone = "Review tidak ditemukan"

// ListCycles also reports whether the caller may open and close cycles,
// the HR write rule OpenCycle and CloseCycle enforce.
func (r *Reviews) ListCycles(ctx context.Context) ([]domain.ReviewCycle, bool, error) {
	a, err := r.actor(ctx)
	if err != nil {
		return nil, false, err
	}
	list, err := r.Store.ListReviewCycles(ctx)
	return list, a.HRWrite, err
}

// OpenCycle creates the cycle, a draft review per active employee with a
// copy of the active behavioural standards, then prefills each review's
// work result from the quarter's KPI. dueDate defaults to 14 days after
// the quarter ends.
func (r *Reviews) OpenCycle(ctx context.Context, year, quarter int, dueDate string) (*domain.ReviewCycle, string, error) {
	a, err := r.actor(ctx)
	if err != nil {
		return nil, "", err
	}
	if !a.HRWrite {
		return nil, "", apperr.New(apperr.Unauthorized, "Hanya HRD yang boleh membuka siklus review")
	}
	if year < 2020 || year > 2100 {
		return nil, "", apperr.Field("periodYear", "Tahun tidak valid")
	}
	if quarter < 1 || quarter > 4 {
		return nil, "", apperr.Field("periodQuarter", "Kuartal harus 1–4")
	}
	label := domain.QuarterLabel(year, quarter)
	exists, err := r.Store.ReviewCycleExists(ctx, year, quarter)
	if err != nil {
		return nil, "", err
	}
	if exists {
		return nil, "", apperr.New(apperr.Conflict, "Siklus "+label+" sudah ada")
	}
	start, end := domain.QuarterRange(year, quarter)
	if dueDate == "" {
		e, _ := domain.ParseDay(end)
		dueDate = e.AddDate(0, 0, 14).Format(domain.IsoDate)
	} else if !validDay(dueDate) {
		return nil, "", apperr.Field("dueDate", "Tanggal tenggat tidak valid")
	}
	ppl, err := r.people(ctx)
	if err != nil {
		return nil, "", err
	}
	standards, err := r.Store.ListBehavioralStandards(ctx)
	if err != nil {
		return nil, "", err
	}
	creator := a.Name
	if creator == "" {
		creator = "HRD"
	}
	cycle := domain.ReviewCycle{ID: newID(), Name: label, Year: year, Quarter: quarter, StartDate: start, EndDate: end, DueDate: dueDate,
		Status: domain.CycleOpen, CreatedBy: a.by(), CreatedByName: creator}
	var reviewIDs []string
	err = r.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.InsertReviewCycle(ctx, cycle); err != nil {
			return err
		}
		for _, e := range ppl.active() {
			rv := domain.Review{ID: newID(), CycleID: cycle.ID, EmployeeID: e.ID, DepartmentID: e.DepartmentID, PeriodLabel: label, StartDate: start, EndDate: end}
			if err := tx.InsertReview(ctx, rv); err != nil {
				return err
			}
			for i, s := range standards {
				if err := tx.InsertReviewItem(ctx, domain.ReviewItem{ID: newID(), ReviewID: rv.ID, ValueName: s.ValueName, Competency: s.CompetencyName,
					BehavioralStandard: s.StandardDescription, ScoreDescriptions: s.ScoreDescriptions, Weight: s.Weight, ItemOrder: i + 1}); err != nil {
					return err
				}
			}
			reviewIDs = append(reviewIDs, rv.ID)
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	for _, id := range reviewIDs {
		if err := r.recalc(ctx, id); err != nil {
			return nil, "", err
		}
	}
	return &cycle, fmt.Sprintf("Siklus %s dibuka, %d draft review dibuat", label, len(reviewIDs)), nil
}

func (r *Reviews) CloseCycle(ctx context.Context, id string) error {
	a, err := r.actor(ctx)
	if err != nil {
		return err
	}
	if err := requireHRWrite(a); err != nil {
		return err
	}
	c, err := r.Store.GetReviewCycle(ctx, id)
	if err != nil {
		return err
	}
	if c == nil {
		return apperr.New(apperr.NotFound, "Siklus review tidak ditemukan")
	}
	return r.Store.SetReviewCycleStatus(ctx, id, domain.CycleClosed)
}

// recalc recomputes a draft review's totals from the quarter's KPI, the
// behavioural ratings and the project score.
func (r *Reviews) recalc(ctx context.Context, reviewID string) error {
	rv, err := r.Store.GetReview(ctx, reviewID)
	if err != nil || rv == nil || rv.Status == domain.ReviewFinal {
		return err
	}
	return r.recalcReview(ctx, rv)
}

func (r *Reviews) recalcReview(ctx context.Context, rv *domain.Review) error {
	cards, err := r.Store.QuarterScorecards(ctx, []string{rv.EmployeeID}, rv.CycleYear, domain.QuarterMonths(rv.CycleQuarter))
	if err != nil {
		return err
	}
	sum, n := 0.0, 0
	for _, c := range cards {
		if c.Score != nil {
			sum += *c.Score
			n++
		}
	}
	var work *float64
	if n > 0 {
		v := domain.Round2(sum / float64(n))
		work = &v
	}
	items, err := r.Store.ListReviewItems(ctx, rv.ID)
	if err != nil {
		return err
	}
	bi := make([]domain.BehaviorItem, len(items))
	for i, it := range items {
		bi[i] = domain.BehaviorItem{Score: it.Score, Weight: it.Weight}
	}
	project := rv.Project
	if project != nil && *project == 0 {
		project = nil
	}
	rv.Work, rv.Behavior = work, domain.BehaviorScore(bi)
	rv.Grand = domain.CombineReviewScores(rv.Work, rv.Behavior, project)
	cats, err := r.Store.ListPerformanceCategories(ctx)
	if err != nil {
		return err
	}
	rv.Category = nil
	if c := domain.CategoryFor(cats, rv.Grand); c != "" {
		rv.Category = &c
	}
	return r.Store.UpdateReview(ctx, *rv)
}

// ReviewView is a review with names for the list and detail pages.
type ReviewView struct {
	domain.Review
	EmployeeName   string
	DepartmentName string
}

// ListReviews: HR sees every review of the cycle, leads their team and
// themselves, everyone else their own.
func (r *Reviews) ListReviews(ctx context.Context, cycleID string) ([]ReviewView, bool, error) {
	a, err := r.actor(ctx)
	if err != nil {
		return nil, false, err
	}
	rows, err := r.Store.ListReviewsByCycle(ctx, cycleID)
	if err != nil {
		return nil, false, err
	}
	ppl, err := r.people(ctx)
	if err != nil {
		return nil, false, err
	}
	out := []ReviewView{}
	for _, rv := range rows {
		e, ok := ppl.employee(rv.EmployeeID)
		if !ok {
			continue
		}
		if !a.SeesEmployee(e) {
			continue
		}
		out = append(out, ReviewView{Review: rv, EmployeeName: e.Name, DepartmentName: ppl.deptName(rv.DepartmentID)})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].DepartmentName != out[j].DepartmentName {
			return out[i].DepartmentName != "" && (out[j].DepartmentName == "" || out[i].DepartmentName < out[j].DepartmentName)
		}
		return out[i].EmployeeName < out[j].EmployeeName
	})
	return out, a.HRWrite || a.IsLead(), nil
}

// ReviewDetail is one review with items, the quarter's KPI and flags.
type ReviewDetail struct {
	Review      ReviewView
	Items       []domain.ReviewItem
	KPIMonths   []domain.Scorecard
	CanRate     bool
	CanFinalize bool
	IsOwner     bool
}

type reviewAccess struct {
	review  *domain.Review
	emp     domain.Employee
	ppl     *people
	isOwner bool
	canRate bool
}

func (r *Reviews) access(ctx context.Context, a *Actor, id string) (*reviewAccess, error) {
	rv, err := r.Store.GetReview(ctx, id)
	if err != nil {
		return nil, err
	}
	if rv == nil {
		return nil, apperr.New(apperr.NotFound, msgReviewGone)
	}
	ppl, err := r.people(ctx)
	if err != nil {
		return nil, err
	}
	e, _ := ppl.employee(rv.EmployeeID)
	e.ID = rv.EmployeeID
	if e.DepartmentID == nil {
		e.DepartmentID = rv.DepartmentID
	}
	return &reviewAccess{review: rv, emp: e, ppl: ppl, isOwner: a.IsEmployee(rv.EmployeeID), canRate: a.HRWrite || a.Leads(e)}, nil
}

func (r *Reviews) GetReview(ctx context.Context, id string) (*ReviewDetail, error) {
	a, err := r.actor(ctx)
	if err != nil {
		return nil, err
	}
	acc, err := r.access(ctx, a, id)
	if err != nil {
		return nil, err
	}
	if !a.HRRead && !acc.canRate && !acc.isOwner {
		return nil, apperr.New(apperr.Unauthorized, "Tidak berhak melihat review ini")
	}
	items, err := r.Store.ListReviewItems(ctx, id)
	if err != nil {
		return nil, err
	}
	kpi, err := r.Store.QuarterScorecards(ctx, []string{acc.review.EmployeeID}, acc.review.CycleYear, domain.QuarterMonths(acc.review.CycleQuarter))
	if err != nil {
		return nil, err
	}
	return &ReviewDetail{Review: ReviewView{Review: *acc.review, EmployeeName: acc.emp.Name, DepartmentName: acc.ppl.deptName(acc.review.DepartmentID)},
		Items: items, KPIMonths: kpi, CanRate: acc.canRate, CanFinalize: a.HRApprove, IsOwner: acc.isOwner}, nil
}

// ReviewAction is PATCH /reviews/{id}.
type ReviewAction struct {
	Action       string
	Text         *string
	ItemID       string
	Score        *float64
	Notes        *string
	ProjectScore *float64
}

func (r *Reviews) Act(ctx context.Context, id string, in ReviewAction) (string, error) {
	a, err := r.actor(ctx)
	if err != nil {
		return "", err
	}
	acc, err := r.access(ctx, a, id)
	if err != nil {
		return "", err
	}
	rv := acc.review
	if rv.Status == domain.ReviewFinal && in.Action != domain.ReviewSign {
		return "", apperr.New(apperr.Validation, "Review sudah final, tidak bisa diubah")
	}
	myName := a.Name
	if myName == "" {
		myName = "HRD"
	}
	setReviewer := func() {
		if rv.ReviewerID == nil && a.EmployeeID() != "" {
			rv.ReviewerID = strPtr(a.EmployeeID())
		}
		rv.ReviewerName = &myName
	}
	today := r.today()
	switch in.Action {
	case domain.ReviewSelfAssessment:
		if !acc.isOwner {
			return "", apperr.New(apperr.Unauthorized, "Hanya karyawan yang bersangkutan yang boleh mengisi self assessment")
		}
		rv.SelfAssessment = trimToNil(in.Text)
		return "Self assessment tersimpan", r.Store.UpdateReview(ctx, *rv)

	case domain.ReviewRateItem:
		if !acc.canRate {
			return "", apperr.New(apperr.Unauthorized, "Hanya Head Division departemen ini (atau HRD) yang boleh menilai perilaku")
		}
		if in.Score == nil {
			return "", apperr.New(apperr.Validation, "Skor harus 1–5")
		}
		score, msg := domain.BehaviorRating(*in.Score)
		if msg != "" {
			return "", apperr.New(apperr.Validation, msg)
		}
		items, err := r.Store.ListReviewItems(ctx, id)
		if err != nil {
			return "", err
		}
		var item *domain.ReviewItem
		for i := range items {
			if items[i].ID == in.ItemID {
				item = &items[i]
			}
		}
		if item == nil {
			return "", apperr.New(apperr.Validation, "Item bukan milik review ini")
		}
		weighted := domain.Round2(item.Weight * float64(score) / 5)
		item.Score, item.WeightedScore, item.Notes = &score, &weighted, trimToNil(in.Notes)
		if err := r.Store.UpdateReviewItem(ctx, *item); err != nil {
			return "", err
		}
		setReviewer()
		if err := r.Store.UpdateReview(ctx, *rv); err != nil {
			return "", err
		}
		return "Penilaian tersimpan", r.recalc(ctx, id)

	case domain.ReviewReviewerNotes:
		if !acc.canRate {
			return "", apperr.New(apperr.Unauthorized, "Hanya Head Division (atau HRD) yang boleh mengisi catatan reviewer")
		}
		project, msg := domain.ProjectScore(in.ProjectScore)
		if msg != "" {
			return "", apperr.New(apperr.Validation, msg)
		}
		rv.ReviewerNotes, rv.Project = trimToNil(in.Text), project
		setReviewer()
		if err := r.Store.UpdateReview(ctx, *rv); err != nil {
			return "", err
		}
		return "Catatan reviewer tersimpan", r.recalc(ctx, id)

	case domain.ReviewManagerNotes:
		if !a.HRApprove && !a.Reports[rv.EmployeeID] {
			return "", apperr.New(apperr.Unauthorized, "Hanya atasan langsung (atau HRD) yang boleh mengisi catatan manajer")
		}
		rv.ManagerNotes = trimToNil(in.Text)
		if a.EmployeeID() != "" {
			rv.ManagerID = strPtr(a.EmployeeID())
		}
		return "Catatan manajer tersimpan", r.Store.UpdateReview(ctx, *rv)

	case domain.ReviewSign:
		if acc.isOwner {
			rv.EmployeeSignDate = &today
			return "Ditandatangani sebagai karyawan", r.Store.UpdateReview(ctx, *rv)
		}
		if !acc.canRate {
			return "", apperr.New(apperr.Unauthorized, "Tidak berhak menandatangani review ini")
		}
		rv.ReviewerSignDate = &today
		if rv.ReviewerID == nil && a.EmployeeID() != "" {
			rv.ReviewerID = strPtr(a.EmployeeID())
		}
		if rv.ReviewerName == nil {
			rv.ReviewerName = &myName
		}
		return "Ditandatangani sebagai reviewer", r.Store.UpdateReview(ctx, *rv)

	case domain.ReviewFinalize:
		if !a.HRApprove {
			return "", apperr.New(apperr.Unauthorized, "Hanya HRD yang boleh memfinalkan review")
		}
		if err := r.recalcReview(ctx, rv); err != nil {
			return "", err
		}
		if rv.ManagerNotes == nil {
			rv.ManagerNotes = trimToNil(in.Text)
		}
		if rv.ManagerID == nil && a.EmployeeID() != "" {
			rv.ManagerID = strPtr(a.EmployeeID())
		}
		rv.Status, rv.FinalizedBy, rv.FinalizedAt = domain.ReviewFinal, a.by(), timePtr(r.now())
		if err := r.Store.UpdateReview(ctx, *rv); err != nil {
			return "", err
		}
		r.improvements.fromReview(ctx, *rv, a.UserID)
		return "Review difinalkan, nilai terkunci", nil
	}
	return "", apperr.New(apperr.Validation, "Aksi tidak dikenal")
}

// RealtimeRow is one employee's quarter KPI so far.
type RealtimeRow struct {
	EmployeeID     string
	EmployeeName   string
	DepartmentName string
	AvgScore       *float64
	Months         []domain.Scorecard
}

// Realtime is realtimeKPI: the quarter's monthly scorecards per employee
// in reach, best average first.
func (r *Reviews) Realtime(ctx context.Context, year, quarter int) ([]RealtimeRow, error) {
	a, err := r.actor(ctx)
	if err != nil {
		return nil, err
	}
	if year < 2020 || year > 2100 || quarter < 1 || quarter > 4 {
		return nil, apperr.New(apperr.Validation, msgPeriodInvalid)
	}
	ppl, err := r.people(ctx)
	if err != nil {
		return nil, err
	}
	var emps []domain.Employee
	for _, e := range ppl.active() {
		if a.SeesEmployee(e) {
			emps = append(emps, e)
		}
	}
	if len(emps) == 0 {
		return []RealtimeRow{}, nil
	}
	ids := make([]string, len(emps))
	for i, e := range emps {
		ids[i] = e.ID
	}
	cards, err := r.Store.QuarterScorecards(ctx, ids, year, domain.QuarterMonths(quarter))
	if err != nil {
		return nil, err
	}
	byEmp := map[string][]domain.Scorecard{}
	for _, c := range cards {
		byEmp[c.EmployeeID] = append(byEmp[c.EmployeeID], c)
	}
	out := make([]RealtimeRow, len(emps))
	for i, e := range emps {
		row := RealtimeRow{EmployeeID: e.ID, EmployeeName: e.Name, DepartmentName: ppl.deptName(e.DepartmentID), Months: byEmp[e.ID]}
		sum, n := 0.0, 0
		for _, c := range row.Months {
			if c.Score != nil {
				sum += *c.Score
				n++
			}
		}
		if n > 0 {
			v := domain.Round2(sum / float64(n))
			row.AvgScore = &v
		}
		out[i] = row
	}
	sort.SliceStable(out, func(i, j int) bool {
		ai, aj := out[i].AvgScore, out[j].AvgScore
		if (ai == nil) != (aj == nil) {
			return ai != nil
		}
		if ai != nil && *ai != *aj {
			return *ai > *aj
		}
		return out[i].EmployeeName < out[j].EmployeeName
	})
	return out, nil
}
