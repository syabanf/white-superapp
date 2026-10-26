package app

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// Feedback is 360 feedback (NüHabit payroll/feedback*.go): cycles,
// assignments, responses, manager approval and summaries whose
// weaknesses become improvement items.
type Feedback struct {
	Deps
	improvements *Improvements
}

func NewFeedback(d Deps, imp *Improvements) *Feedback { return &Feedback{d, imp} }

const (
	msgCycleGone      = "Siklus feedback tidak ditemukan"
	msgAssignmentGone = "Penugasan feedback tidak ditemukan"
)

func (f *Feedback) ListCategories(ctx context.Context) ([]domain.FeedbackCategory, error) {
	if _, err := f.actor(ctx); err != nil {
		return nil, err
	}
	return f.Store.ListFeedbackCategories(ctx)
}

func (f *Feedback) hrReader(ctx context.Context) (*Actor, error) {
	a, err := f.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !a.HRRead {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	return a, nil
}

func (f *Feedback) hrWriter(ctx context.Context) (*Actor, error) {
	a, err := f.actor(ctx)
	if err != nil {
		return nil, err
	}
	return a, requireHRWrite(a)
}

func (f *Feedback) ListCycles(ctx context.Context, status string) ([]domain.FeedbackCycle, error) {
	if _, err := f.hrReader(ctx); err != nil {
		return nil, err
	}
	return f.Store.ListFeedbackCycles(ctx, status)
}

func (f *Feedback) GetCycle(ctx context.Context, id string) (*domain.FeedbackCycle, error) {
	if _, err := f.hrReader(ctx); err != nil {
		return nil, err
	}
	c, err := f.Store.GetFeedbackCycle(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, apperr.New(apperr.NotFound, msgCycleGone)
	}
	return c, nil
}

// SaveCycle creates (ID empty) or updates a cycle.
func (f *Feedback) SaveCycle(ctx context.Context, c domain.FeedbackCycle) (*domain.FeedbackCycle, error) {
	a, err := f.hrWriter(ctx)
	if err != nil {
		return nil, err
	}
	c.Name, c.PeriodLabel = strings.TrimSpace(c.Name), strings.TrimSpace(c.PeriodLabel)
	if c.Status == "" {
		c.Status = domain.FeedbackDraft
	}
	v := map[string][]string{}
	if c.Name == "" || len([]rune(c.Name)) > 255 {
		v["name"] = []string{"Nama siklus wajib diisi (maks. 255 karakter)"}
	}
	if !slices.Contains([]string{domain.FeedbackDraft, domain.FeedbackActive, domain.FeedbackClosed}, c.Status) {
		v["status"] = []string{"Status siklus tidak dikenal"}
	}
	for key, w := range map[string]*float64{"kpiWeight": c.KPIWeight, "feedbackWeight": c.FeedbackWeight} {
		if w != nil && (*w < 0 || *w > 100) {
			v[key] = []string{"Bobot harus 0–100"}
		}
	}
	for key, d := range map[string]*string{"startDate": c.StartDate, "endDate": c.EndDate} {
		if d != nil && !validDay(*d) {
			v[key] = []string{"Tanggal tidak valid"}
		}
	}
	if len(v) > 0 {
		return nil, apperr.New(apperr.Validation, "Data tidak valid").Fields(v)
	}
	c.UpdatedBy = a.by()
	if c.ID == "" {
		c.ID, c.CreatedBy = newID(), a.by()
		err = f.Store.InsertFeedbackCycle(ctx, c)
	} else {
		cur, gerr := f.Store.GetFeedbackCycle(ctx, c.ID)
		if gerr != nil {
			return nil, gerr
		}
		if cur == nil {
			return nil, apperr.New(apperr.NotFound, msgCycleGone)
		}
		err = f.Store.UpdateFeedbackCycle(ctx, c)
	}
	if err != nil {
		return nil, err
	}
	return f.Store.GetFeedbackCycle(ctx, c.ID)
}

func (f *Feedback) DeleteCycle(ctx context.Context, id string) error {
	if _, err := f.hrWriter(ctx); err != nil {
		return err
	}
	return f.Store.DeleteFeedbackCycle(ctx, id)
}

// AssignmentInput is one reviewer of one employee.
type AssignmentInput struct {
	EmployeeID   string
	ReviewerID   string
	Relationship string
	DueDate      *string
}

func (f *Feedback) CreateAssignments(ctx context.Context, cycleID string, items []AssignmentInput) (int, error) {
	a, err := f.hrWriter(ctx)
	if err != nil {
		return 0, err
	}
	c, err := f.Store.GetFeedbackCycle(ctx, cycleID)
	if err != nil {
		return 0, err
	}
	if c == nil {
		return 0, apperr.New(apperr.NotFound, msgCycleGone)
	}
	if len(items) == 0 || len(items) > 500 {
		return 0, apperr.New(apperr.Validation, "Data tidak valid")
	}
	ppl, err := f.people(ctx)
	if err != nil {
		return 0, err
	}
	for _, it := range items {
		if _, ok := ppl.employee(it.EmployeeID); !ok {
			return 0, apperr.Field("employeeId", msgEmployeeGone)
		}
		if _, ok := ppl.employee(it.ReviewerID); !ok {
			return 0, apperr.Field("reviewerId", "Penilai tidak ditemukan")
		}
		if !slices.Contains(domain.FeedbackRelationships, it.Relationship) {
			return 0, apperr.Field("relationshipType", "Hubungan penilai tidak dikenal")
		}
		if (it.Relationship == domain.RelSelf) != (it.EmployeeID == it.ReviewerID) {
			return 0, apperr.Field("relationshipType", "Penilaian diri hanya untuk karyawan itu sendiri")
		}
		if it.Relationship == domain.RelSelf && !c.AllowSelfAssessment {
			return 0, apperr.Field("relationshipType", "Siklus ini tidak memakai penilaian diri")
		}
		if err := validDue(it.DueDate); err != nil {
			return 0, err
		}
	}
	created := 0
	err = f.Store.WithTx(ctx, func(tx domain.Store) error {
		for _, it := range items {
			due := it.DueDate
			if due == nil {
				due = c.EndDate
			}
			ok, err := tx.InsertAssignment(ctx, domain.FeedbackAssignment{ID: newID(), CycleID: cycleID, EmployeeID: it.EmployeeID,
				ReviewerID: it.ReviewerID, Relationship: it.Relationship, DueDate: due, CreatedBy: a.by()})
			if err != nil {
				return err
			}
			if ok {
				created++
			}
		}
		return nil
	})
	return created, err
}

// AutoAssign plans a full 360 per active employee: self (when the cycle
// allows it), the direct manager, each direct report and up to `peers`
// colleagues of the same department.
func (f *Feedback) AutoAssign(ctx context.Context, cycleID string, peers int) (int, error) {
	if _, err := f.hrWriter(ctx); err != nil {
		return 0, err
	}
	c, err := f.Store.GetFeedbackCycle(ctx, cycleID)
	if err != nil {
		return 0, err
	}
	if c == nil {
		return 0, apperr.New(apperr.NotFound, msgCycleGone)
	}
	ppl, err := f.people(ctx)
	if err != nil {
		return 0, err
	}
	peers = max(0, min(peers, 5))
	active := ppl.active()
	var items []AssignmentInput
	for _, e := range active {
		if c.AllowSelfAssessment {
			items = append(items, AssignmentInput{EmployeeID: e.ID, ReviewerID: e.ID, Relationship: domain.RelSelf})
		}
		if e.ManagerID != nil {
			if _, ok := ppl.employee(*e.ManagerID); ok {
				items = append(items, AssignmentInput{EmployeeID: e.ID, ReviewerID: *e.ManagerID, Relationship: domain.RelManager})
			}
		}
		for _, r := range active {
			if r.ManagerID != nil && *r.ManagerID == e.ID {
				items = append(items, AssignmentInput{EmployeeID: e.ID, ReviewerID: r.ID, Relationship: domain.RelSubordinate})
			}
		}
		if e.DepartmentID == nil || peers == 0 {
			continue
		}
		mates := ppl.members(*e.DepartmentID)
		start := slices.IndexFunc(mates, func(m domain.Employee) bool { return m.ID == e.ID })
		picked := 0
		for i := 1; i < len(mates) && picked < peers; i++ {
			m := mates[(start+i)%len(mates)]
			if (e.ManagerID != nil && *e.ManagerID == m.ID) || (m.ManagerID != nil && *m.ManagerID == e.ID) {
				continue
			}
			items = append(items, AssignmentInput{EmployeeID: e.ID, ReviewerID: m.ID, Relationship: domain.RelPeer})
			picked++
		}
	}
	if len(items) == 0 {
		return 0, nil
	}
	return f.CreateAssignments(ctx, cycleID, items)
}

// AssignmentView is an assignment with names (the reviewer's name is
// hidden from the reviewee in anonymous cycles by the handler's caller).
type AssignmentView struct {
	domain.FeedbackAssignment
	EmployeeName string
	ReviewerName string
}

func (f *Feedback) views(ppl *people, rows []domain.FeedbackAssignment) []AssignmentView {
	out := make([]AssignmentView, len(rows))
	for i, r := range rows {
		out[i] = AssignmentView{FeedbackAssignment: r, EmployeeName: ppl.name(&r.EmployeeID), ReviewerName: ppl.name(&r.ReviewerID)}
	}
	return out
}

// ListAssignments is the HR overview; leads may list their team's.
func (f *Feedback) ListAssignments(ctx context.Context, cycleID, employeeID, reviewerID, status string) ([]AssignmentView, error) {
	a, err := f.actor(ctx)
	if err != nil {
		return nil, err
	}
	ppl, err := f.people(ctx)
	if err != nil {
		return nil, err
	}
	if !a.HRRead {
		e, ok := ppl.employee(employeeID)
		if !ok || !a.Leads(e) {
			return nil, apperr.New(apperr.Unauthorized, "")
		}
	}
	rows, err := f.Store.ListAssignments(ctx, cycleID, employeeID, reviewerID, status)
	if err != nil {
		return nil, err
	}
	return f.views(ppl, rows), nil
}

// MyAssignments lists what the caller has to review.
func (f *Feedback) MyAssignments(ctx context.Context) ([]AssignmentView, error) {
	a, err := f.actor(ctx)
	if err != nil {
		return nil, err
	}
	if a.EmployeeID() == "" {
		return []AssignmentView{}, nil
	}
	rows, err := f.Store.ListAssignments(ctx, "", "", a.EmployeeID(), "")
	if err != nil {
		return nil, err
	}
	ppl, err := f.people(ctx)
	if err != nil {
		return nil, err
	}
	return f.views(ppl, rows), nil
}

// AssignmentDetail is one assignment with its answers and the form.
type AssignmentDetail struct {
	Assignment AssignmentView
	Responses  []domain.FeedbackResponse
	Categories []domain.FeedbackCategory
	CanAnswer  bool
	CanDecide  bool
}

func (f *Feedback) loadAssignment(ctx context.Context, id string) (*domain.FeedbackAssignment, *domain.FeedbackCycle, *people, error) {
	asg, err := f.Store.GetAssignment(ctx, id)
	if err != nil {
		return nil, nil, nil, err
	}
	if asg == nil {
		return nil, nil, nil, apperr.New(apperr.NotFound, msgAssignmentGone)
	}
	c, err := f.Store.GetFeedbackCycle(ctx, asg.CycleID)
	if err != nil {
		return nil, nil, nil, err
	}
	ppl, err := f.people(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	return asg, c, ppl, nil
}

func (f *Feedback) canDecide(a *Actor, ppl *people, asg domain.FeedbackAssignment) bool {
	if a.HRApprove {
		return true
	}
	e, ok := ppl.employee(asg.EmployeeID)
	return ok && a.Leads(e) && !a.IsEmployee(asg.ReviewerID)
}

func (f *Feedback) GetAssignment(ctx context.Context, id string) (*AssignmentDetail, error) {
	a, err := f.actor(ctx)
	if err != nil {
		return nil, err
	}
	asg, _, ppl, err := f.loadAssignment(ctx, id)
	if err != nil {
		return nil, err
	}
	isReviewer := a.IsEmployee(asg.ReviewerID)
	decide := f.canDecide(a, ppl, *asg)
	if !isReviewer && !a.HRRead && !decide {
		return nil, apperr.New(apperr.NotFound, msgAssignmentGone)
	}
	responses, err := f.Store.ListResponses(ctx, id)
	if err != nil {
		return nil, err
	}
	cats, err := f.Store.ListFeedbackCategories(ctx)
	if err != nil {
		return nil, err
	}
	return &AssignmentDetail{Assignment: f.views(ppl, []domain.FeedbackAssignment{*asg})[0], Responses: responses, Categories: cats,
		CanAnswer: isReviewer && domain.AssignmentOpen(asg.Status) && asg.CycleStatus == domain.FeedbackActive,
		CanDecide: decide && asg.Status == domain.AssignmentSubmitted}, nil
}

// ResponseInput is one answer.
type ResponseInput struct {
	CriteriaID string
	Rating     int
	Comments   *string
}

// SubmitResponses saves answers; with submit every criterion must be
// answered and the assignment moves to submitted.
func (f *Feedback) SubmitResponses(ctx context.Context, id string, answers []ResponseInput, submit bool) (*domain.FeedbackAssignment, error) {
	a, err := f.actor(ctx)
	if err != nil {
		return nil, err
	}
	asg, c, ppl, err := f.loadAssignment(ctx, id)
	if err != nil {
		return nil, err
	}
	if !a.IsEmployee(asg.ReviewerID) {
		return nil, apperr.New(apperr.Unauthorized, "Hanya penilai yang ditugaskan yang boleh mengisi feedback ini")
	}
	if c == nil || c.Status != domain.FeedbackActive {
		return nil, apperr.New(apperr.Conflict, "Siklus feedback tidak aktif")
	}
	if !domain.AssignmentOpen(asg.Status) {
		return nil, apperr.New(apperr.Conflict, "Feedback sudah dikirim")
	}
	cats, err := f.Store.ListFeedbackCategories(ctx)
	if err != nil {
		return nil, err
	}
	criteria := map[string]bool{}
	for _, cat := range cats {
		for _, cr := range cat.Criteria {
			criteria[cr.ID] = true
		}
	}
	for _, ans := range answers {
		if !criteria[ans.CriteriaID] {
			return nil, apperr.Field("criteriaId", "Kriteria tidak dikenal")
		}
		if ans.Rating < 1 || ans.Rating > 5 {
			return nil, apperr.Field("rating", "Rating harus 1-5")
		}
	}
	now := f.now()
	err = f.Store.WithTx(ctx, func(tx domain.Store) error {
		for _, ans := range answers {
			if err := tx.UpsertResponse(ctx, domain.FeedbackResponse{ID: newID(), AssignmentID: id, CriteriaID: ans.CriteriaID,
				Rating: ans.Rating, Comments: trimToNil(ans.Comments)}); err != nil {
				return err
			}
		}
		saved, err := tx.ListResponses(ctx, id)
		if err != nil {
			return err
		}
		if asg.StartedAt == nil {
			asg.StartedAt = &now
		}
		asg.Status = domain.AssignmentInProgress
		if submit {
			if len(saved) < len(criteria) {
				return apperr.New(apperr.Validation, "Lengkapi semua kriteria sebelum mengirim")
			}
			asg.Status, asg.SubmittedAt, asg.RejectionReason = domain.AssignmentSubmitted, &now, nil
			if !c.RequireManagerReview {
				asg.Status = domain.AssignmentApproved
			}
		}
		return tx.UpdateAssignment(ctx, *asg)
	})
	if err != nil {
		return nil, err
	}
	if submit && c.RequireManagerReview {
		if e, ok := ppl.employee(asg.EmployeeID); ok {
			f.notify(ctx, Notice{UserIDs: ppl.userIDs(ppl.leadsOf(e)...), Type: NotifFeedbackPending, Href: hrefFeedback,
				Title: "Feedback 360 menunggu persetujuan", Body: "Feedback untuk " + e.Name + " sudah dikirim dan menunggu persetujuan."})
		}
	}
	return asg, nil
}

// Decide approves or rejects a submitted assignment.
func (f *Feedback) Decide(ctx context.Context, id string, approve bool, comments, reason *string) (*domain.FeedbackAssignment, string, error) {
	a, err := f.actor(ctx)
	if err != nil {
		return nil, "", err
	}
	asg, _, ppl, err := f.loadAssignment(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if !f.canDecide(a, ppl, *asg) {
		return nil, "", apperr.New(apperr.Unauthorized, "")
	}
	if asg.Status != domain.AssignmentSubmitted {
		return nil, "", apperr.New(apperr.Conflict, "Hanya feedback berstatus submitted yang bisa diputuskan")
	}
	now := f.now()
	msg := "Feedback disetujui"
	if approve {
		asg.Status, asg.ApprovedBy, asg.ApprovedAt, asg.ManagerComments = domain.AssignmentApproved, a.by(), &now, trimToNil(comments)
	} else {
		r := trimToNil(reason)
		if r == nil {
			return nil, "", apperr.Field("rejectionReason", "Alasan penolakan wajib diisi")
		}
		asg.Status, asg.RejectionReason = domain.AssignmentRejected, r
		msg = "Feedback ditolak, penilai diminta merevisi"
		f.notify(ctx, Notice{UserIDs: ppl.userIDs(asg.ReviewerID), Type: NotifFeedbackPending, Href: hrefFeedback,
			Title: "Feedback 360 perlu direvisi", Body: *r})
	}
	if err := f.Store.UpdateAssignment(ctx, *asg); err != nil {
		return nil, "", err
	}
	return asg, msg, nil
}

func (f *Feedback) DeleteAssignment(ctx context.Context, id string) error {
	if _, err := f.hrWriter(ctx); err != nil {
		return err
	}
	return f.Store.DeleteAssignment(ctx, id)
}

// GenerateSummaries derives every employee's summary from the counted
// answers (approved; submitted too when the cycle needs no approval),
// adds the KPI average of the cycle's months and files each weakness as an
// improvement item. Locked summaries are kept.
func (f *Feedback) GenerateSummaries(ctx context.Context, cycleID string) (int, int, error) {
	a, err := f.hrWriter(ctx)
	if err != nil {
		return 0, 0, err
	}
	c, err := f.Store.GetFeedbackCycle(ctx, cycleID)
	if err != nil {
		return 0, 0, err
	}
	if c == nil {
		return 0, 0, apperr.New(apperr.NotFound, msgCycleGone)
	}
	cats, err := f.Store.ListFeedbackCategories(ctx)
	if err != nil {
		return 0, 0, err
	}
	weights := make([]domain.FeedbackCategoryWeight, len(cats))
	for i, cat := range cats {
		weights[i] = domain.FeedbackCategoryWeight{Code: cat.Code, Name: cat.Name, Weight: cat.Weight}
	}
	ratings, err := f.Store.CycleRatings(ctx, cycleID)
	if err != nil {
		return 0, 0, err
	}
	counted := func(status string) bool {
		return status == domain.AssignmentApproved || (!c.RequireManagerReview && status == domain.AssignmentSubmitted)
	}
	byEmp := map[string][]domain.FeedbackRating{}
	var order []string
	for _, r := range ratings {
		if !counted(r.AssignmentStatus) {
			continue
		}
		if _, ok := byEmp[r.EmployeeID]; !ok {
			order = append(order, r.EmployeeID)
		}
		byEmp[r.EmployeeID] = append(byEmp[r.EmployeeID], r.FeedbackRating)
	}
	sort.Strings(order)
	kpi, err := f.cycleKPI(ctx, *c, order)
	if err != nil {
		return 0, 0, err
	}
	ppl, err := f.people(ctx)
	if err != nil {
		return 0, 0, err
	}
	existing, err := f.Store.ListFeedbackSummaries(ctx, cycleID, "")
	if err != nil {
		return 0, 0, err
	}
	locked := map[string]bool{}
	for _, s := range existing {
		locked[s.EmployeeID] = s.IsLocked
	}
	generated, filed := 0, 0
	for _, empID := range order {
		if locked[empID] {
			continue
		}
		sc := domain.SummarizeFeedback(weights, byEmp[empID])
		s := domain.FeedbackSummary{ID: newID(), CycleID: cycleID, EmployeeID: empID, Scores: map[string]*float64{}, Overall360: sc.Overall,
			KPIScore: kpi[empID], Strengths: sc.Strengths, Weaknesses: sc.Weaknesses, ResponsesCount: sc.Responses}
		for code, v := range sc.Categories {
			s.Scores[code] = &v
		}
		if s.KPIScore != nil && s.Overall360 != nil {
			score, grade := domain.FinalScore(*s.KPIScore, *s.Overall360, c.KPIWeight, c.FeedbackWeight)
			score = domain.Round2(score)
			s.FinalScore, s.FinalGrade = &score, &grade
		}
		if err := f.Store.UpsertFeedbackSummary(ctx, s); err != nil {
			return generated, filed, err
		}
		generated++
		stored, err := f.Store.ListFeedbackSummaries(ctx, cycleID, empID)
		if err != nil || len(stored) == 0 {
			continue
		}
		e, _ := ppl.employee(empID)
		e.ID = empID
		filed += f.improvements.fromFeedback(ctx, stored[0], e, c.Name, a.UserID)
	}
	return generated, filed, nil
}

// cycleKPI averages each employee's scorecards over the cycle's months
// (the three months before the end date without dates).
func (f *Feedback) cycleKPI(ctx context.Context, c domain.FeedbackCycle, employeeIDs []string) (map[string]*float64, error) {
	out := map[string]*float64{}
	if len(employeeIDs) == 0 {
		return out, nil
	}
	end := domain.PeriodOf(f.today())
	if c.EndDate != nil {
		end = domain.PeriodOf(*c.EndDate)
	}
	start := end.Previous().Previous()
	if c.StartDate != nil {
		start = domain.PeriodOf(*c.StartDate)
	}
	sums, counts := map[string]float64{}, map[string]int{}
	for p := start; p.Key() <= end.Key(); p = p.Next() {
		cards, err := f.Store.ListScorecards(ctx, p, employeeIDs)
		if err != nil {
			return nil, err
		}
		for _, sc := range cards {
			if sc.Score != nil {
				sums[sc.EmployeeID] += *sc.Score
				counts[sc.EmployeeID]++
			}
		}
	}
	for id, n := range counts {
		v := domain.Round2(sums[id] / float64(n))
		out[id] = &v
	}
	return out, nil
}

// SummaryView is a summary with the employee's name.
type SummaryView struct {
	domain.FeedbackSummary
	EmployeeName   string
	DepartmentName string
}

// ListSummaries: HR all, leads their team, everyone their own.
func (f *Feedback) ListSummaries(ctx context.Context, cycleID, employeeID string) ([]SummaryView, error) {
	a, err := f.actor(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := f.Store.ListFeedbackSummaries(ctx, cycleID, employeeID)
	if err != nil {
		return nil, err
	}
	ppl, err := f.people(ctx)
	if err != nil {
		return nil, err
	}
	out := []SummaryView{}
	for _, s := range rows {
		e, ok := ppl.employee(s.EmployeeID)
		if !ok {
			e = domain.Employee{ID: s.EmployeeID}
		}
		if !a.SeesEmployee(e) {
			continue
		}
		out = append(out, SummaryView{FeedbackSummary: s, EmployeeName: e.Name, DepartmentName: ppl.deptName(e.DepartmentID)})
	}
	return out, nil
}

// UpdateSummary edits the narrative fields and the lock (hr.write).
func (f *Feedback) UpdateSummary(ctx context.Context, id string, strengths, weaknesses []string, comments *string, locked bool) (*domain.FeedbackSummary, error) {
	if _, err := f.hrWriter(ctx); err != nil {
		return nil, err
	}
	s, err := f.Store.GetFeedbackSummary(ctx, id)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, apperr.New(apperr.NotFound, "Ringkasan feedback tidak ditemukan")
	}
	s.Strengths, s.Weaknesses = domain.CleanList(strengths, 20), domain.CleanList(weaknesses, 20)
	s.ManagerComments, s.IsLocked = trimToNil(comments), locked
	if err := f.Store.UpdateFeedbackSummary(ctx, *s); err != nil {
		return nil, err
	}
	return f.Store.GetFeedbackSummary(ctx, id)
}

// pendingReminderText is the reminder body for open assignments.
func pendingReminderText(n int, nearest string) string {
	if nearest == "" {
		return fmt.Sprintf("Anda punya %d feedback 360 yang belum diisi.", n)
	}
	return fmt.Sprintf("Anda punya %d feedback 360 yang belum diisi, tenggat terdekat %s.", n, nearest)
}
