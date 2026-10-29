// Package app holds the performance use cases: KPI scorecards, quarterly
// reviews, 360 feedback, department tasks, logbook, daily and monthly work
// recaps, improvement items and the scheduled automations.
package app

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Directory is the slice of the hr EmployeeService this service reads.
type Directory interface {
	// EmployeeByUser is nil when the user has no employee record.
	EmployeeByUser(ctx context.Context, userID string) (*domain.Employee, error)
	// Employees lists the directory (active only unless includeInactive).
	Employees(ctx context.Context, includeInactive bool) ([]domain.Employee, error)
	Departments(ctx context.Context) ([]domain.Department, error)
	Attendance(ctx context.Context, employeeIDs []string, p domain.Period) (map[string]domain.Attendance, error)
}

// Notice is one in-app notification (identity NotificationService.Notify).
type Notice struct {
	UserIDs []string
	Type    string
	Title   string
	Body    string
	Href    string
}

// Notifier delivers notices. WhatsApp and email can be added behind it.
type Notifier interface {
	Notify(ctx context.Context, n Notice) error
}

// AIWriter writes the recap texts: OpenRouter when a key is configured,
// the deterministic mock otherwise. The model id is stored with the text.
type AIWriter interface {
	DailySummary(ctx context.Context, in domain.DailyRecapInput) (string, string, error)
	MonthlyNarrative(ctx context.Context, in domain.MonthlyRecapInput) (string, string, error)
	SuggestImprovements(ctx context.Context, in domain.MonthlyRecapInput) ([]domain.ImprovementSuggestion, string, error)
}

// FileStore keeps task attachments.
type FileStore interface {
	Put(ctx context.Context, key string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

// RecapPDF renders a monthly recap as PDF.
type RecapPDF func(r domain.MonthlyRecap, subjectName, departmentName string) ([]byte, error)

// Deps is shared by every use-case struct.
type Deps struct {
	Store     domain.Store
	Directory Directory
	Notifier  Notifier
	AI        AIWriter
	Files     FileStore
	RenderPDF RecapPDF
	Clock     clock.Clock
	Logger    *slog.Logger
}

func (d Deps) now() time.Time {
	if d.Clock == nil {
		return time.Now().UTC()
	}
	return d.Clock.Now()
}

func (d Deps) log() *slog.Logger {
	if d.Logger == nil {
		return slog.Default()
	}
	return d.Logger
}

func (d Deps) today() string { return domain.DayKey(d.now(), domain.Jakarta) }

// Today is today's date in WIB (YYYY-MM-DD).
func (d Deps) Today() string { return d.today() }

// CurrentPeriod is this year and month in WIB.
func (d Deps) CurrentPeriod() (int, int) {
	y, m, _ := d.now().In(domain.Jakarta).Date()
	return y, int(m)
}

func newID() string { return ids.New() }

// Notification types.
const (
	NotifTaskDigest      = "TASK_DIGEST"
	NotifTaskOverdue     = "TASK_OVERDUE"
	NotifTaskReview      = "TASK_REVIEW"
	NotifReviewDeadline  = "REVIEW_DEADLINE"
	NotifFeedbackPending = "FEEDBACK_PENDING"
	NotifLogbookPending  = "LOGBOOK_PENDING"
	NotifKPIDue          = "KPI_SNAPSHOT_DUE"
	NotifKPIFinal        = "KPI_FINAL"
	NotifDailyRecap      = "DAILY_RECAP"
	NotifMonthlyRecap    = "MONTHLY_RECAP"
	NotifImprovement     = "IMPROVEMENT"
)

// Links the notices open (Next.js routes of the ERP workspace).
const (
	hrefTasks        = "/erp/me/tasks"
	hrefReviews      = "/erp/performance/reviews"
	hrefFeedback     = "/erp/performance/feedback"
	hrefLogbook      = "/erp/performance/logbook"
	hrefKPI          = "/erp/performance/kpi"
	hrefRecaps       = "/erp/performance/recaps"
	hrefImprovements = "/erp/performance/improvements"
)

// ─── Actor ───────────────────────────────────────────────────

// Actor is the caller with their employee record and reach.
type Actor struct {
	UserID    string
	Name      string
	Employee  *domain.Employee
	Self      bool
	HRRead    bool
	HRWrite   bool
	HRApprove bool
	// HeadOf are the departments the employee heads (hr head_employee_id).
	HeadOf map[string]bool
	// Reports are the employee's direct reports (hr manager_id).
	Reports map[string]bool
}

func (a *Actor) EmployeeID() string {
	if a.Employee == nil {
		return ""
	}
	return a.Employee.ID
}

func (a *Actor) DepartmentID() string {
	if a.Employee == nil || a.Employee.DepartmentID == nil {
		return ""
	}
	return *a.Employee.DepartmentID
}

// IsEmployee reports the caller's own employee id.
func (a *Actor) IsEmployee(id string) bool { return id != "" && a.EmployeeID() == id }

// Leads reports a department head or the direct manager of e.
func (a *Actor) Leads(e domain.Employee) bool {
	return a.Reports[e.ID] || (e.DepartmentID != nil && a.HeadOf[*e.DepartmentID])
}

// SeesEmployee: HR readers, the employee themselves and their leads.
func (a *Actor) SeesEmployee(e domain.Employee) bool {
	return a.HRRead || a.IsEmployee(e.ID) || a.Leads(e)
}

// SeesDepartment: HR readers and the department head.
func (a *Actor) SeesDepartment(id string) bool { return a.HRRead || a.HeadOf[id] }

// IsLead reports whether the caller heads a department or manages anyone.
func (a *Actor) IsLead() bool { return len(a.HeadOf) > 0 || len(a.Reports) > 0 }

func (a *Actor) by() *string { return &a.UserID }

// actor resolves the caller. Every RPC needs self.service or an hr key.
func (d Deps) actor(ctx context.Context) (*Actor, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	a := &Actor{UserID: p.UserID, Name: p.Name, Self: p.Can(authn.PermSelfService), HRRead: p.Can(authn.PermHRRead),
		HRWrite: p.Can(authn.PermHRWrite), HRApprove: p.Can(authn.PermHRApprove), HeadOf: map[string]bool{}, Reports: map[string]bool{}}
	// Writers and approvers act on what they can see.
	a.HRRead = a.HRRead || a.HRWrite || a.HRApprove
	if !a.Self && !a.HRRead {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	if a.Employee, err = d.Directory.EmployeeByUser(ctx, p.UserID); err != nil {
		return nil, err
	}
	if a.Employee == nil {
		return a, nil
	}
	a.Name = a.Employee.Name
	depts, err := d.Directory.Departments(ctx)
	if err != nil {
		return nil, err
	}
	for _, dept := range depts {
		if dept.HeadEmployeeID != nil && *dept.HeadEmployeeID == a.Employee.ID {
			a.HeadOf[dept.ID] = true
		}
	}
	people, err := d.Directory.Employees(ctx, false)
	if err != nil {
		return nil, err
	}
	for _, e := range people {
		if e.ManagerID != nil && *e.ManagerID == a.Employee.ID {
			a.Reports[e.ID] = true
		}
	}
	return a, nil
}

func requireHRWrite(a *Actor) error {
	if !a.HRWrite {
		return apperr.New(apperr.Unauthorized, "")
	}
	return nil
}

func requireHRApprove(a *Actor) error {
	if !a.HRApprove {
		return apperr.New(apperr.Unauthorized, "")
	}
	return nil
}

var errNoEmployee = apperr.New(apperr.Unauthorized, "Akun ini tidak terhubung ke data karyawan")

// ─── Directory helpers ───────────────────────────────────────

// people indexes the directory, inactive employees included so old rows
// keep their names.
type people struct {
	byID  map[string]domain.Employee
	depts map[string]domain.Department
}

func (d Deps) people(ctx context.Context) (*people, error) {
	all, err := d.Directory.Employees(ctx, true)
	if err != nil {
		return nil, err
	}
	depts, err := d.Directory.Departments(ctx)
	if err != nil {
		return nil, err
	}
	p := &people{byID: map[string]domain.Employee{}, depts: map[string]domain.Department{}}
	for _, e := range all {
		p.byID[e.ID] = e
	}
	for _, dep := range depts {
		p.depts[dep.ID] = dep
	}
	return p, nil
}

func (p *people) name(id *string) string {
	if id == nil {
		return ""
	}
	return p.byID[*id].Name
}

func (p *people) deptName(id *string) string {
	if id == nil {
		return ""
	}
	return p.depts[*id].Name
}

func (p *people) employee(id string) (domain.Employee, bool) {
	e, ok := p.byID[id]
	return e, ok
}

// active lists active employees sorted by name.
func (p *people) active() []domain.Employee {
	var out []domain.Employee
	for _, e := range p.byID {
		if e.Active {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// members are the active employees of a department.
func (p *people) members(departmentID string) []domain.Employee {
	var out []domain.Employee
	for _, e := range p.active() {
		if e.InDepartment(departmentID) {
			out = append(out, e)
		}
	}
	return out
}

// userIDs maps employee ids to their identity users, skipping those
// without a login.
func (p *people) userIDs(employeeIDs ...string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range employeeIDs {
		e, ok := p.byID[id]
		if !ok || e.UserID == nil || seen[*e.UserID] {
			continue
		}
		seen[*e.UserID] = true
		out = append(out, *e.UserID)
	}
	return out
}

// leadsOf are the department head and the direct manager of e.
func (p *people) leadsOf(e domain.Employee) []string {
	var out []string
	if e.ManagerID != nil {
		out = append(out, *e.ManagerID)
	}
	if e.DepartmentID != nil {
		if h := p.depts[*e.DepartmentID].HeadEmployeeID; h != nil && *h != e.ID && (e.ManagerID == nil || *e.ManagerID != *h) {
			out = append(out, *h)
		}
	}
	return out
}

// notify sends best effort: delivery failures are logged, never returned.
func (d Deps) notify(ctx context.Context, n Notice) {
	if len(n.UserIDs) == 0 || d.Notifier == nil {
		return
	}
	if err := d.Notifier.Notify(ctx, n); err != nil {
		d.log().Warn("notify failed", "type", n.Type, "err", err)
	}
}

// ─── small helpers ───────────────────────────────────────────

func trimToNil(s *string) *string {
	if s == nil {
		return nil
	}
	if t := strings.TrimSpace(*s); t != "" {
		return &t
	}
	return nil
}

func strPtr(s string) *string { return &s }

func timePtr(t time.Time) *time.Time { return &t }

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func validDay(s string) bool {
	_, ok := domain.ParseDay(s)
	return ok && len(s) == 10
}
