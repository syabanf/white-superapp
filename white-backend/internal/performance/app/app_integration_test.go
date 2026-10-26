package app_test

import (
	"bytes"
	"context"
	"io/fs"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/performance"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/adapter/pdf"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/adapter/providers/openrouter"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

// ─── fakes ───────────────────────────────────────────────────

func sp(s string) *string { return &s }

const (
	deptSocial = "dept_social"
	deptSEO    = "dept_seo"
)

// The WHITE fixture: Social Media (head Hana, members Rani and Dimas) and
// SEO (head Sari, member Bayu). hr users have no employee record.
var staff = []domain.Employee{
	{ID: "emp_hana", UserID: sp("u_hana"), Name: "Hana", DepartmentID: sp(deptSocial), DepartmentName: "Social Media", Active: true},
	{ID: "emp_rani", UserID: sp("u_rani"), Name: "Rani", DepartmentID: sp(deptSocial), DepartmentName: "Social Media", ManagerID: sp("emp_hana"), Active: true},
	{ID: "emp_dimas", UserID: sp("u_dimas"), Name: "Dimas", DepartmentID: sp(deptSocial), DepartmentName: "Social Media", ManagerID: sp("emp_hana"), Active: true},
	{ID: "emp_sari", UserID: sp("u_sari"), Name: "Sari", DepartmentID: sp(deptSEO), DepartmentName: "SEO", Active: true},
	{ID: "emp_bayu", UserID: sp("u_bayu"), Name: "Bayu", DepartmentID: sp(deptSEO), DepartmentName: "SEO", ManagerID: sp("emp_sari"), Active: true},
}

var departments = []domain.Department{
	{ID: deptSocial, Code: "SOC", Name: "Social Media", HeadEmployeeID: sp("emp_hana"), Active: true},
	{ID: deptSEO, Code: "SEO", Name: "SEO", HeadEmployeeID: sp("emp_sari"), Active: true},
}

type fakeDirectory struct{ attendanceErr error }

func (fakeDirectory) EmployeeByUser(_ context.Context, userID string) (*domain.Employee, error) {
	for _, e := range staff {
		if e.UserID != nil && *e.UserID == userID {
			out := e
			return &out, nil
		}
	}
	return nil, nil
}

func (fakeDirectory) Employees(context.Context, bool) ([]domain.Employee, error) { return staff, nil }

func (fakeDirectory) Departments(context.Context) ([]domain.Department, error) {
	return departments, nil
}

func (f fakeDirectory) Attendance(_ context.Context, ids []string, _ domain.Period) (map[string]domain.Attendance, error) {
	if f.attendanceErr != nil {
		return nil, f.attendanceErr
	}
	out := map[string]domain.Attendance{}
	for _, id := range ids {
		out[id] = domain.Attendance{EmployeeID: id, WorkingDays: 20, PresentDays: 20, LateDays: 2, LateMinutes: 30}
	}
	return out, nil
}

type fakeNotifier struct {
	mu   sync.Mutex
	sent []app.Notice
}

func (f *fakeNotifier) Notify(_ context.Context, n app.Notice) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, n)
	return nil
}

func (f *fakeNotifier) count(userID, typ string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, s := range f.sent {
		for _, u := range s.UserIDs {
			if u == userID && (typ == "" || s.Type == typ) {
				n++
			}
		}
	}
	return n
}

func (f *fakeNotifier) titles(userID string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, s := range f.sent {
		for _, u := range s.UserIDs {
			if u == userID {
				out = append(out, s.Title+" | "+s.Body)
			}
		}
	}
	return strings.Join(out, "\n")
}

// ─── fixture ─────────────────────────────────────────────────

type fixture struct {
	store    *postgres.Store
	clk      *clock.Fixed
	notifier *fakeNotifier
	kpi      *app.KPI
	reviews  *app.Reviews
	feedback *app.Feedback
	tasks    *app.Tasks
	logbook  *app.Logbook
	recaps   *app.Recaps
	imp      *app.Improvements
	auto     *app.Automations
}

// referenceData re-applies 00002 after the truncate.
func referenceData(t *testing.T) string {
	t.Helper()
	raw, err := fs.ReadFile(performance.Migrations(), "00002_reference_data.sql")
	if err != nil {
		t.Fatal(err)
	}
	up := strings.SplitN(string(raw), "-- +goose Down", 2)[0]
	return strings.Replace(up, "-- +goose Up", "", 1)
}

func setup(t *testing.T) *fixture {
	t.Helper()
	pool := pg.TestPool(t, performance.Migrations(), "migrations", performance.Schema)
	pg.TruncateSchema(t, pool, performance.Schema)
	if _, err := pool.Exec(context.Background(), referenceData(t)); err != nil {
		t.Fatal(err)
	}
	clk := &clock.Fixed{T: time.Date(2026, 10, 21, 3, 0, 0, 0, time.UTC)} // Wednesday 10:00 WIB
	f := &fixture{store: postgres.NewStore(pool), clk: clk, notifier: &fakeNotifier{}}
	deps := app.Deps{Store: f.store, Directory: fakeDirectory{}, Notifier: f.notifier,
		AI:    openrouter.NewSource(func(context.Context) (openrouter.Credentials, error) { return openrouter.Credentials{}, nil }, openrouter.Options{}),
		Files: storage.NewLocal(t.TempDir()), RenderPDF: pdf.RenderMonthlyRecap, Clock: clk}
	f.imp = app.NewImprovements(deps)
	f.kpi = app.NewKPI(deps)
	f.reviews = app.NewReviews(deps, f.imp)
	f.feedback = app.NewFeedback(deps, f.imp)
	f.tasks = app.NewTasks(deps, f.imp)
	f.logbook = app.NewLogbook(deps, f.imp)
	f.recaps = app.NewRecaps(deps, f.imp)
	f.auto = app.NewAutomations(deps, f.kpi, f.tasks, f.recaps)
	return f
}

func (f *fixture) at(t time.Time) { f.clk.T = t.UTC() }

// as signs in a user with ERP permission keys.
func as(userID string, perms ...string) context.Context {
	return authn.WithPrincipal(context.Background(), authn.Principal{UserID: userID, Role: authn.RoleMember, Perms: perms})
}

var (
	hrAdmin = as("u_hr", authn.PermHRRead, authn.PermHRWrite, authn.PermHRApprove)
	hana    = as("u_hana", authn.PermSelfService)
	rani    = as("u_rani", authn.PermSelfService)
	dimas   = as("u_dimas", authn.PermSelfService)
	bayu    = as("u_bayu", authn.PermSelfService)
)

func code(t *testing.T, err error, want apperr.Code) {
	t.Helper()
	if !apperr.Is(err, want) {
		t.Fatalf("want %s, got %v", want, err)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) improvements(t *testing.T, source string) []app.ImprovementView {
	t.Helper()
	list, err := f.imp.List(hrAdmin, app.ImprovementQuery{Source: source})
	must(t, err)
	return list
}

func (f *fixture) todayItem(t *testing.T, ctx context.Context, title string) app.TaskView {
	t.Helper()
	items, err := f.tasks.ListItems(ctx, app.ItemFilter{Scope: "mine", View: "today"})
	must(t, err)
	for _, it := range items {
		if it.Title == title {
			return it
		}
	}
	t.Fatalf("task %q not in today's list (%d items)", title, len(items))
	return app.TaskView{}
}

// ─── tests ───────────────────────────────────────────────────

func TestPermissions(t *testing.T) {
	f := setup(t)
	if _, err := f.tasks.Meta(context.Background()); !apperr.Is(err, apperr.Unauthenticated) {
		t.Fatal(err)
	}
	if _, err := f.tasks.Meta(as("u_rani")); !apperr.Is(err, apperr.Unauthorized) {
		t.Fatal("no self.service and no hr key", err)
	}
	_, err := f.kpi.SaveIndicator(rani, app.IndicatorInput{Code: "x_y", Name: "Xyz", Direction: domain.HigherBetter, SourceKind: domain.SourceManual})
	code(t, err, apperr.Unauthorized)
	meta, err := f.tasks.Meta(hana)
	must(t, err)
	if meta.TeamScope != "heads" || len(meta.Departments) != 1 || len(meta.Members) != 3 {
		t.Fatalf("%+v", meta)
	}
}

func TestTaskLifecycle(t *testing.T) {
	f := setup(t)
	ind, err := f.kpi.SaveIndicator(hrAdmin, app.IndicatorInput{Code: "posting_ontime", Name: "Posting tepat jadwal", Direction: domain.HigherBetter,
		SourceKind: domain.SourceManual, IsActive: true})
	must(t, err)

	// Rani is no head: she may only file a personal task.
	_, err = f.tasks.Create(rani, app.TaskInput{AssigneeID: "emp_dimas", Body: domain.TaskBody{Title: "Bantu Dimas", Recurrence: "daily"}})
	code(t, err, apperr.Unauthorized)
	_, err = f.tasks.Create(hana, app.TaskInput{AssigneeID: "emp_bayu", Body: domain.TaskBody{Title: "Lintas tim", Recurrence: "daily"}})
	code(t, err, apperr.Validation)

	_, err = f.tasks.Create(hana, app.TaskInput{AssigneeID: "emp_rani", Priority: "high", KPIIndicatorID: &ind.ID, KPIContribution: fp(1),
		Body: domain.TaskBody{Title: "Posting konten harian", Recurrence: "daily", Subtasks: []string{"Caption", "Jadwalkan"}}})
	must(t, err)
	it := f.todayItem(t, rani, "Posting konten harian")
	if !it.CanWork || it.CanReview || it.SubtaskCount != 2 || it.Progress != 0 {
		t.Fatalf("%+v", it)
	}
	if _, err := f.tasks.GetOccurrence(bayu, it.OccurrenceID); !apperr.Is(err, apperr.NotFound) {
		t.Fatal("other department sees it", err)
	}
	_, err = f.tasks.Act(rani, it.OccurrenceID, app.OccurrenceAction{Action: domain.ActProgress, ProgressPercent: ip(50)})
	code(t, err, apperr.Validation) // sub-tasks drive the progress

	detail, err := f.tasks.GetOccurrence(rani, it.OccurrenceID)
	must(t, err)
	res, err := f.tasks.Act(rani, it.OccurrenceID, app.OccurrenceAction{Action: domain.ActCheckSubtask, SubtaskID: detail.Subtasks[0].ID, Checked: true})
	must(t, err)
	if res.Complete || *res.Progress != 50 {
		t.Fatalf("%+v", res)
	}
	res, err = f.tasks.Act(rani, it.OccurrenceID, app.OccurrenceAction{Action: domain.ActCheckSubtask, SubtaskID: detail.Subtasks[1].ID, Checked: true})
	must(t, err)
	if !res.Complete || f.notifier.count("u_hana", app.NotifTaskReview) != 1 {
		t.Fatalf("%+v %s", res, f.notifier.titles("u_hana"))
	}
	_, err = f.tasks.Act(dimas, it.OccurrenceID, app.OccurrenceAction{Action: domain.ActApprove})
	code(t, err, apperr.Unauthorized)

	// Rejected: the review note becomes an improvement item for Rani.
	_, err = f.tasks.Act(hana, it.OccurrenceID, app.OccurrenceAction{Action: domain.ActReject, Notes: sp("Caption belum sesuai brief klien.")})
	must(t, err)
	imps := f.improvements(t, domain.SourceTaskReject)
	if len(imps) != 1 || deref(imps[0].EmployeeID) != "emp_rani" || imps[0].Title != "Perbaiki: Posting konten harian" ||
		imps[0].Description != "Caption belum sesuai brief klien." {
		t.Fatalf("%+v", imps)
	}
	if f.notifier.count("u_rani", app.NotifTaskReview) != 1 {
		t.Fatal("assignee not told about the rejection")
	}
	_, err = f.tasks.Act(rani, it.OccurrenceID, app.OccurrenceAction{Action: domain.ActDone})
	must(t, err)
	_, err = f.tasks.Act(hana, it.OccurrenceID, app.OccurrenceAction{Action: domain.ActApprove, Notes: sp("Mantap")})
	must(t, err)
	detail, err = f.tasks.GetOccurrence(rani, it.OccurrenceID)
	must(t, err)
	if detail.Status != domain.StatusApproved || detail.Contribution == nil || *detail.Contribution.Amount != 1 || len(detail.Updates) < 4 {
		t.Fatalf("%+v %+v", detail.TaskView, detail.Contribution)
	}
	_, err = f.tasks.Act(rani, it.OccurrenceID, app.OccurrenceAction{Action: domain.ActReopen})
	code(t, err, apperr.Unauthorized) // only the head reopens an approved row

	// A personal task needs no review: done is approved.
	_, err = f.tasks.Create(dimas, app.TaskInput{AssigneeID: "emp_dimas", Body: domain.TaskBody{Title: "Belajar Canva", Recurrence: "once", DueDate: "2026-10-21"}})
	must(t, err)
	own := f.todayItem(t, dimas, "Belajar Canva")
	if own.ReviewRequired || !own.CanManage {
		t.Fatalf("%+v", own)
	}
	_, err = f.tasks.Act(dimas, own.OccurrenceID, app.OccurrenceAction{Action: domain.ActDone})
	must(t, err)

	// Attachments.
	att, err := f.tasks.AddAttachment(rani, it.OccurrenceID, "bukti.png", "image/png", []byte("\x89PNG\r\n\x1a\nfake"))
	must(t, err)
	got, data, err := f.tasks.GetAttachment(hana, it.OccurrenceID, att.ID)
	must(t, err)
	if got.Mime != "image/png" || !bytes.HasPrefix(data, []byte("\x89PNG")) {
		t.Fatalf("%+v", got)
	}
	must(t, f.tasks.DeleteAttachment(rani, it.OccurrenceID, att.ID))

	// Views.
	kpi, err := f.tasks.KPIView(hana, deptSocial, "2026-10")
	must(t, err)
	if len(kpi) != 1 || kpi[0].Approved != 1 || kpi[0].ContributionTotal != 1 {
		t.Fatalf("%+v", kpi)
	}
	mon, err := f.tasks.Monitor(hrAdmin, "", "2026-10")
	must(t, err)
	if mon.Scope != "all" || mon.Summary.Approved != 2 {
		t.Fatalf("%+v", mon.Summary)
	}
	contrib, err := f.tasks.Contributions(rani, "me", "2026-10")
	must(t, err)
	if len(contrib) != 1 || contrib[0].TasksDone != 1 {
		t.Fatalf("%+v", contrib)
	}
	_, err = f.tasks.ListItems(rani, app.ItemFilter{Scope: "team"})
	code(t, err, apperr.Unauthorized)
}

func fp(v float64) *float64 { return &v }
func ip(v int) *int         { return &v }

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func TestLogbookFlow(t *testing.T) {
	f := setup(t)
	_, err := f.logbook.CreateTemplate(rani, app.TemplateInput{DepartmentID: deptSocial, Name: "Harian", Items: []app.TemplateItemInput{{Title: "Cek DM"}}})
	code(t, err, apperr.Unauthorized)
	tpl, err := f.logbook.CreateTemplate(hana, app.TemplateInput{Name: "Checklist harian", Items: []app.TemplateItemInput{
		{Title: "Cek DM klien", Weight: fp(2)}, {Title: "Posting sesuai jadwal"}, {Title: " "}}})
	must(t, err)
	if tpl.DepartmentID != deptSocial || tpl.DepartmentName != "Social Media" || len(tpl.Items) != 2 {
		t.Fatalf("%+v", tpl)
	}
	tpls, err := f.logbook.ListTemplates(hrAdmin, deptSocial, false)
	must(t, err)
	if len(tpls) != 1 || tpls[0].DepartmentName != "Social Media" {
		t.Fatalf("templates %+v", tpls)
	}
	if _, err := f.logbook.ListTemplates(bayu, deptSocial, false); !apperr.Is(err, apperr.Unauthorized) {
		t.Fatal(err)
	}
	e, err := f.logbook.CreateEntry(rani, tpl.ID, "2026-10-21", nil, nil)
	must(t, err)
	if e.Title != "Checklist harian - 2026-10-21" || len(e.Items) != 2 {
		t.Fatalf("%+v", e)
	}
	_, err = f.logbook.CreateEntry(rani, tpl.ID, "2026-10-21", nil, nil)
	code(t, err, apperr.Conflict)
	e, err = f.logbook.UpdateItem(rani, e.Items[0].ID, bp(true), nil)
	must(t, err)
	if e.Completion != 50 || e.KPIScore != 66.67 {
		t.Fatalf("%v %v", e.Completion, e.KPIScore)
	}
	_, err = f.logbook.Review(hana, e.ID, true, sp("x"))
	code(t, err, apperr.Conflict) // still draft
	_, err = f.logbook.Submit(rani, e.ID, sp("Semua aman"))
	must(t, err)
	if f.notifier.count("u_hana", app.NotifLogbookPending) != 1 {
		t.Fatal("head not told")
	}
	_, err = f.logbook.Review(bayu, e.ID, false, nil)
	code(t, err, apperr.NotFound)
	_, err = f.logbook.Review(hana, e.ID, true, sp("Bukti posting belum dilampirkan."))
	must(t, err)
	imps := f.improvements(t, domain.SourceLogbookReject)
	if len(imps) != 1 || imps[0].Title != "Logbook: Bukti posting belum dilampirkan" {
		t.Fatalf("%+v", imps)
	}
	sum, _, err := f.logbook.Summary(hrAdmin, "", "2026-10-01", "2026-10-31")
	must(t, err)
	if len(sum) != 1 || sum[0].RejectedEntries != 1 {
		t.Fatalf("%+v", sum)
	}
	list, total, err := f.logbook.ListEntries(hana, app.EntryQuery{DepartmentID: deptSocial})
	must(t, err)
	if total != 1 || len(list) != 1 || list[0].EmployeeName != "Rani" || list[0].DepartmentName != "Social Media" {
		t.Fatalf("%d %+v", total, list)
	}
	got, _, err := f.logbook.GetEntry(hana, e.ID)
	must(t, err)
	if got.EmployeeName != "Rani" || got.DepartmentName != "Social Media" {
		t.Fatalf("entry names %q %q", got.EmployeeName, got.DepartmentName)
	}
	msg, archived, err := f.logbook.DeleteTemplate(hrAdmin, tpl.ID)
	must(t, err)
	if !archived || msg == "" {
		t.Fatal("used template must be archived")
	}
}

func bp(b bool) *bool { return &b }

func socialWeights(t *testing.T, f *fixture) {
	t.Helper()
	_, err := f.kpi.SaveDepartmentConfig(hrAdmin, deptSocial, []app.WeightInput{
		{IndicatorID: "ind_task_completion", Weight: 40}, {IndicatorID: "ind_att_ontime", Weight: 20}, {IndicatorID: "ind_supervisor_rubric", Weight: 40}})
	must(t, err)
}

func TestKPISnapshotAndScorecard(t *testing.T) {
	f := setup(t)
	_, err := f.kpi.SaveDepartmentConfig(hrAdmin, deptSocial, nil)
	code(t, err, apperr.Validation)
	socialWeights(t, f)

	_, err = f.tasks.Create(hana, app.TaskInput{AssigneeID: "emp_rani", Body: domain.TaskBody{Title: "Posting harian", Recurrence: "daily"}})
	must(t, err)
	it := f.todayItem(t, rani, "Posting harian")
	_, err = f.tasks.Act(rani, it.OccurrenceID, app.OccurrenceAction{Action: domain.ActDone})
	must(t, err)
	_, err = f.tasks.Act(hana, it.OccurrenceID, app.OccurrenceAction{Action: domain.ActApprove})
	must(t, err)

	oct := domain.Period{Year: 2026, Month: 10}
	sum, msg, err := f.kpi.RunSnapshot(hrAdmin, oct)
	must(t, err)
	if sum.ScorecardsUpserted != 3 || !strings.HasPrefix(msg, "Snapshot KPI 10/2026") || sum.Collectors[domain.CodeTaskCompletion] != 1 {
		t.Fatalf("%+v %s", sum, msg)
	}
	_, _, err = f.kpi.SaveManual(bayu, app.ManualInput{EmployeeID: "emp_rani", Period: oct, Value: 4})
	code(t, err, apperr.Unauthorized)
	_, _, err = f.kpi.SaveManual(hana, app.ManualInput{EmployeeID: "emp_rani", Period: oct, Value: 6})
	code(t, err, apperr.Validation)
	score, status, err := f.kpi.SaveManual(hana, app.ManualInput{EmployeeID: "emp_rani", Period: oct, Value: 4})
	must(t, err)
	// task 1/1 vs 0.9 -> 1.111, on time 18/20 vs 0.95 -> 0.947, rubric 4/5 -> 0.8:
	// 40*1.111 + 20*0.947 + 40*0.8 = 95.39
	if status != "updated" || score == nil || *score != 95.39 {
		t.Fatalf("%v %v", status, score)
	}
	team, err := f.kpi.ListScorecards(hana, app.ScorecardQuery{Period: oct, Team: true})
	must(t, err)
	if len(team) != 2 || team[0].EmployeeName == "" {
		t.Fatalf("%+v", team)
	}
	mine, err := f.kpi.ListScorecards(dimas, app.ScorecardQuery{Period: oct, EmployeeID: "emp_rani"})
	if !apperr.Is(err, apperr.Unauthorized) || mine != nil {
		t.Fatal("peer reads a scorecard", err)
	}
	cards, err := f.kpi.ListScorecards(rani, app.ScorecardQuery{Period: oct})
	must(t, err)
	if len(cards) != 1 {
		t.Fatalf("%+v", cards)
	}
	_, _, err = f.kpi.SetScorecardStatus(hana, cards[0].ID, "finalize")
	code(t, err, apperr.Unauthorized)
	_, _, err = f.kpi.SetScorecardStatus(hrAdmin, cards[0].ID, "finalize")
	must(t, err)
	if f.notifier.count("u_rani", app.NotifKPIFinal) != 1 {
		t.Fatal("employee not told about the final score")
	}
	_, _, err = f.kpi.SaveManual(hana, app.ManualInput{EmployeeID: "emp_rani", Period: oct, Value: 5})
	code(t, err, apperr.Conflict)
	sum, _, err = f.kpi.RunSnapshot(hrAdmin, oct)
	must(t, err)
	if sum.SkippedFinal != 1 || sum.ScorecardsUpserted != 2 {
		t.Fatalf("%+v", sum)
	}
	detail, err := f.kpi.GetScorecard(rani, cards[0].ID)
	must(t, err)
	if detail.Scorecard.Status != domain.ScorecardFinal || len(detail.Snapshots) != 3 {
		t.Fatalf("%+v", detail)
	}
}

func TestReviewCycle(t *testing.T) {
	f := setup(t)
	_, _, err := f.reviews.OpenCycle(hana, 2026, 3, "")
	code(t, err, apperr.Unauthorized)
	cycle, msg, err := f.reviews.OpenCycle(hrAdmin, 2026, 3, "")
	must(t, err)
	if cycle.DueDate != "2026-10-14" || msg != "Siklus Q3 2026 dibuka, 5 draft review dibuat" {
		t.Fatalf("%+v %s", cycle, msg)
	}
	_, _, err = f.reviews.OpenCycle(hrAdmin, 2026, 3, "")
	code(t, err, apperr.Conflict)
	admin := authn.WithPrincipal(context.Background(), authn.Principal{UserID: "u_admin", Role: authn.RoleAdmin})
	for ctx, want := range map[context.Context]bool{hrAdmin: true, admin: true, hana: false} {
		cycles, canManage, err := f.reviews.ListCycles(ctx)
		must(t, err)
		if len(cycles) != 1 || canManage != want {
			t.Fatalf("cycles %d canManage %v want %v", len(cycles), canManage, want)
		}
	}
	list, canReview, err := f.reviews.ListReviews(hana, cycle.ID)
	must(t, err)
	if len(list) != 3 || !canReview {
		t.Fatalf("%d %v", len(list), canReview)
	}
	var raniReview string
	for _, r := range list {
		if r.EmployeeID == "emp_rani" {
			raniReview = r.ID
		}
	}
	if _, err := f.reviews.GetReview(bayu, raniReview); !apperr.Is(err, apperr.Unauthorized) {
		t.Fatal(err)
	}
	_, err = f.reviews.Act(hana, raniReview, app.ReviewAction{Action: domain.ReviewSelfAssessment, Text: sp("x")})
	code(t, err, apperr.Unauthorized)
	_, err = f.reviews.Act(rani, raniReview, app.ReviewAction{Action: domain.ReviewSelfAssessment, Text: sp("Target tercapai.")})
	must(t, err)
	d, err := f.reviews.GetReview(hana, raniReview)
	must(t, err)
	for _, it := range d.Items {
		_, err := f.reviews.Act(hana, raniReview, app.ReviewAction{Action: domain.ReviewRateItem, ItemID: it.ID, Score: fp(4)})
		must(t, err)
	}
	_, err = f.reviews.Act(hana, raniReview, app.ReviewAction{Action: domain.ReviewRateItem, ItemID: d.Items[0].ID, Score: fp(6)})
	code(t, err, apperr.Validation)
	_, err = f.reviews.Act(hana, raniReview, app.ReviewAction{Action: domain.ReviewReviewerNotes, Text: sp("Lebih proaktif melapor kendala."),
		ProjectScore: fp(90)})
	must(t, err)
	_, err = f.reviews.Act(hana, raniReview, app.ReviewAction{Action: domain.ReviewManagerNotes, Text: sp("Ikut pelatihan copywriting.")})
	must(t, err)
	_, err = f.reviews.Act(hana, raniReview, app.ReviewAction{Action: domain.ReviewFinalize})
	code(t, err, apperr.Unauthorized)
	_, err = f.reviews.Act(hrAdmin, raniReview, app.ReviewAction{Action: domain.ReviewFinalize})
	must(t, err)
	d, err = f.reviews.GetReview(rani, raniReview)
	must(t, err)
	// behaviour 4/5 = 80, project 90, no KPI: 80*30/40 + 90*10/40 = 82.5
	if d.Review.Status != domain.ReviewFinal || *d.Review.Grand != 82.5 || deref(d.Review.Category) != "Baik" {
		t.Fatalf("%+v", d.Review.Review)
	}
	_, err = f.reviews.Act(rani, raniReview, app.ReviewAction{Action: domain.ReviewSelfAssessment, Text: sp("ubah")})
	code(t, err, apperr.Validation)
	imps := f.improvements(t, domain.SourceReview)
	if len(imps) != 2 {
		t.Fatalf("%+v", imps)
	}
}

func TestFeedback360(t *testing.T) {
	f := setup(t)
	cycle, err := f.feedback.SaveCycle(hrAdmin, domain.FeedbackCycle{Name: "360 Q4", Status: domain.FeedbackActive, AllowSelfAssessment: true,
		StartDate: sp("2026-10-01"), EndDate: sp("2026-10-31")})
	must(t, err)
	n, err := f.feedback.AutoAssign(hrAdmin, cycle.ID, 1)
	must(t, err)
	// self 5 + manager 3 + subordinate 3 + peers that are neither manager
	// nor report (Rani and Dimas review each other) 2 = 13
	if n != 13 {
		t.Fatal(n)
	}
	mine, err := f.feedback.MyAssignments(dimas)
	must(t, err)
	cats, err := f.feedback.ListCategories(dimas)
	must(t, err)
	answers := func(rating func(code string) int) []app.ResponseInput {
		var out []app.ResponseInput
		for _, c := range cats {
			for _, cr := range c.Criteria {
				out = append(out, app.ResponseInput{CriteriaID: cr.ID, Rating: rating(c.Code)})
			}
		}
		return out
	}
	var peerOfRani string
	for _, a := range mine {
		if a.EmployeeID == "emp_rani" {
			peerOfRani = a.ID
		}
	}
	if peerOfRani == "" {
		t.Fatalf("%+v", mine)
	}
	_, err = f.feedback.SubmitResponses(rani, peerOfRani, nil, false)
	code(t, err, apperr.Unauthorized)
	_, err = f.feedback.SubmitResponses(dimas, peerOfRani, answers(func(string) int { return 4 })[:2], true)
	code(t, err, apperr.Validation) // incomplete
	low := func(code string) int {
		if code == "communication" {
			return 2
		}
		return 5
	}
	asg, err := f.feedback.SubmitResponses(dimas, peerOfRani, answers(low), true)
	must(t, err)
	if asg.Status != domain.AssignmentApproved {
		t.Fatal(asg.Status)
	}
	gen, filed, err := f.feedback.GenerateSummaries(hrAdmin, cycle.ID)
	must(t, err)
	if gen != 1 || filed != 1 {
		t.Fatal(gen, filed)
	}
	sums, err := f.feedback.ListSummaries(rani, cycle.ID, "")
	must(t, err)
	if len(sums) != 1 || *sums[0].Overall360 != 88 || !strings.HasPrefix(sums[0].Weaknesses[0], "Komunikasi:") {
		t.Fatalf("%+v", sums)
	}
	if others, _ := f.feedback.ListSummaries(bayu, cycle.ID, ""); len(others) != 0 {
		t.Fatal("bayu sees rani's summary")
	}
	imps := f.improvements(t, domain.SourceFeedback)
	if len(imps) != 1 || !strings.HasPrefix(imps[0].Title, "Tingkatkan Komunikasi") {
		t.Fatalf("%+v", imps)
	}
}

func TestAutomations(t *testing.T) {
	f := setup(t)
	socialWeights(t, f)
	tpl, err := f.logbook.CreateTemplate(hana, app.TemplateInput{Name: "Harian", Items: []app.TemplateItemInput{{Title: "Cek DM"}}})
	must(t, err)
	_, err = f.tasks.Create(hana, app.TaskInput{AssigneeID: "emp_rani", Body: domain.TaskBody{Title: "Posting harian", Recurrence: "daily"}})
	must(t, err)
	_, err = f.tasks.Create(hana, app.TaskInput{AssigneeID: "emp_rani", Body: domain.TaskBody{Title: "Laporan klien", Recurrence: "once", DueDate: "2026-10-21"}})
	must(t, err)

	// 07:10 WIB: digest; reruns send nothing new.
	f.at(time.Date(2026, 10, 21, 0, 10, 0, 0, time.UTC))
	must(t, f.auto.Tick15m(context.Background(), f.clk.T))
	must(t, f.auto.Tick15m(context.Background(), f.clk.T))
	if f.notifier.count("u_rani", app.NotifTaskDigest) != 1 || !strings.Contains(f.notifier.titles("u_rani"), "Tugas hari ini: 2") {
		t.Fatal(f.notifier.titles("u_rani"))
	}

	// During the day Rani finishes one task and writes a note.
	f.at(time.Date(2026, 10, 21, 6, 0, 0, 0, time.UTC))
	it := f.todayItem(t, rani, "Laporan klien")
	_, err = f.tasks.Act(rani, it.OccurrenceID, app.OccurrenceAction{Action: domain.ActDone})
	must(t, err)
	_, err = f.recaps.UpdateMyNotes(rani, "", "Meeting brief dengan klien kopi.")
	must(t, err)
	e, err := f.logbook.CreateEntry(rani, tpl.ID, "2026-10-21", nil, nil)
	must(t, err)

	// 17:45 WIB: logbook reminder (draft) and the daily recap with digest.
	f.at(time.Date(2026, 10, 21, 10, 45, 0, 0, time.UTC))
	must(t, f.auto.Tick15m(context.Background(), f.clk.T))
	must(t, f.auto.Tick15m(context.Background(), f.clk.T))
	if f.notifier.count("u_rani", app.NotifLogbookPending) != 1 || f.notifier.count("u_dimas", app.NotifLogbookPending) != 1 {
		t.Fatal("logbook reminders")
	}
	if f.notifier.count("u_rani", app.NotifTaskOverdue) != 0 {
		t.Fatal("nothing overdue yet")
	}
	recap, err := f.recaps.MyDaily(rani, "2026-10-21")
	must(t, err)
	if !recap.Saved || recap.SummaryModel != openrouter.MockModel || len(recap.Items.Tasks) != 1 || !recap.Items.Tasks[0].Completed ||
		len(recap.Items.Logbook) != 1 || !strings.Contains(recap.Summary, "Catatan: Meeting brief dengan klien kopi.") {
		t.Fatalf("%+v", recap)
	}
	if f.notifier.count("u_hana", app.NotifDailyRecap) != 1 || !strings.Contains(f.notifier.titles("u_hana"), "1 dari 2 anggota tim") {
		t.Fatal(f.notifier.titles("u_hana"))
	}
	team, err := f.recaps.TeamDaily(hana, "2026-10-21", "")
	must(t, err)
	if len(team) != 2 {
		t.Fatalf("%+v", team)
	}
	_, err = f.recaps.TeamDaily(rani, "2026-10-21", "")
	code(t, err, apperr.Unauthorized)
	_ = e

	// Next day 16:30 WIB: yesterday's daily occurrence is overdue.
	f.at(time.Date(2026, 10, 22, 9, 30, 0, 0, time.UTC))
	must(t, f.auto.TickHourly(context.Background(), f.clk.T))
	must(t, f.auto.Tick15m(context.Background(), f.clk.T))
	if f.notifier.count("u_rani", app.NotifTaskOverdue) != 1 || !strings.Contains(f.notifier.titles("u_rani"), "1 tugas terlambat") {
		t.Fatal(f.notifier.titles("u_rani"))
	}

	// 1 November 07:05 WIB: KPI snapshot of October, monthly recaps and
	// AI suggestions (suggested until accepted).
	f.at(time.Date(2026, 11, 1, 0, 5, 0, 0, time.UTC))
	must(t, f.auto.TickDaily(context.Background(), f.clk.T))
	must(t, f.auto.TickDaily(context.Background(), f.clk.T))
	if f.notifier.count("u_hana", app.NotifKPIDue) != 1 || f.notifier.count("u_rani", app.NotifMonthlyRecap) != 1 {
		t.Fatal(f.notifier.titles("u_hana"))
	}
	monthly, err := f.recaps.ListMonthly(rani, app.MonthlyQuery{})
	must(t, err)
	var mine *app.MonthlyView
	for i := range monthly {
		if monthly[i].Scope == domain.ScopeEmployee && monthly[i].SubjectID == "emp_rani" {
			mine = &monthly[i]
		}
	}
	if mine == nil || mine.Metrics.TasksDone != 0 || mine.Metrics.TasksPlanned < 2 || mine.Narrative == "" {
		t.Fatalf("%+v", monthly)
	}
	name, data, err := f.recaps.ExportMonthly(rani, mine.ID)
	must(t, err)
	if !bytes.HasPrefix(data, []byte("%PDF")) || name != "rekap-rani-2026-10.pdf" {
		t.Fatal(name)
	}
	sugg, err := f.imp.List(rani, app.ImprovementQuery{Status: domain.ImpSuggested, Mine: true})
	must(t, err)
	if len(sugg) == 0 {
		t.Fatal("no suggestions")
	}
	accepted, err := f.imp.Decide(rani, sugg[0].ID, true, nil, sp("2026-11-15"))
	must(t, err)
	if accepted.Status != domain.ImpOpen || deref(accepted.DueDate) != "2026-11-15" {
		t.Fatalf("%+v", accepted)
	}
	_, err = f.imp.Decide(rani, sugg[0].ID, false, nil, nil)
	code(t, err, apperr.Conflict)

	// 20 November: the accepted item is overdue.
	f.at(time.Date(2026, 11, 20, 0, 5, 0, 0, time.UTC))
	must(t, f.auto.TickDaily(context.Background(), f.clk.T))
	if f.notifier.count("u_rani", app.NotifImprovement) != 1 {
		t.Fatal(f.notifier.titles("u_rani"))
	}
}

func TestImprovementsAndRetro(t *testing.T) {
	f := setup(t)
	_, err := f.imp.Create(dimas, app.ImprovementInput{Scope: domain.ScopeEmployee, EmployeeID: sp("emp_rani"), Title: sp("Bantu Rani")})
	code(t, err, apperr.Unauthorized)
	v, err := f.imp.Create(hana, app.ImprovementInput{Scope: domain.ScopeEmployee, EmployeeID: sp("emp_rani"), Title: sp("Kirim update sebelum jam 12")})
	must(t, err)
	if deref(v.OwnerEmployeeID) != "emp_rani" || v.OwnerName != "Rani" {
		t.Fatalf("%+v", v)
	}
	_, err = f.imp.Update(rani, v.ID, app.ImprovementInput{Status: sp(domain.ImpDeclined)})
	code(t, err, apperr.Validation)
	v, err = f.imp.Update(rani, v.ID, app.ImprovementInput{Status: sp(domain.ImpDone)})
	must(t, err)
	if v.CompletedAt == nil {
		t.Fatal("completed_at")
	}
	if _, err := f.imp.Get(bayu, v.ID); !apperr.Is(err, apperr.NotFound) {
		t.Fatal(err)
	}

	_, err = f.imp.SaveRetro(rani, app.RetroInput{DepartmentID: deptSocial, Period: domain.Period{Year: 2026, Month: 10}})
	code(t, err, apperr.Unauthorized)
	r, err := f.imp.SaveRetro(hana, app.RetroInput{DepartmentID: deptSocial, Period: domain.Period{Year: 2026, Month: 10},
		WentWell: []string{"Engagement naik", " "}, ToImprove: []string{"Brief sering berubah"},
		Actions: []domain.RetroAction{{Description: "Standarkan template brief", OwnerEmployeeID: sp("emp_dimas"), DueDate: sp("2026-11-10")}, {Description: ""}}})
	must(t, err)
	if len(r.WentWell) != 1 || len(r.Actions) != 1 || r.DepartmentName != "Social Media" {
		t.Fatalf("%+v", r)
	}
	closed, filed, err := f.imp.CloseRetro(hana, r.ID)
	must(t, err)
	if filed != 1 || closed.Status != domain.RetroClosed || closed.Actions[0].ImprovementID == nil {
		t.Fatalf("%+v", closed)
	}
	_, _, err = f.imp.CloseRetro(hana, r.ID)
	code(t, err, apperr.Conflict)
	dept, err := f.imp.List(dimas, app.ImprovementQuery{Scope: domain.ScopeDepartment})
	must(t, err)
	if len(dept) != 1 || dept[0].Source != domain.SourceRetro || dept[0].OwnerName != "Dimas" {
		t.Fatalf("%+v", dept)
	}
	retros, err := f.imp.ListRetros(bayu, "")
	must(t, err)
	if len(retros) != 0 {
		t.Fatal("SEO sees Social retros")
	}
}
