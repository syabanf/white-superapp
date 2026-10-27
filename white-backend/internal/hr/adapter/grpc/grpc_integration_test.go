package grpcapi_test

import (
	"context"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	hrv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/hr/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/hr"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/adapter/docs"
	grpcapi "github.com/syabanf/white-superapp/white-backend/internal/hr/adapter/grpc"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/app"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/seed"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

type noIdentity struct{}

func (noIdentity) GetUser(context.Context, string) (*app.UserInfo, error) { return nil, nil }
func (noIdentity) Notify(context.Context, app.NotifyInput) error          { return nil }
func (noIdentity) PermissionHolders(context.Context, string) ([]string, error) {
	return nil, nil
}

// TestHandlersOverGRPC seeds the demo data and calls the service through a
// real gRPC server: directory access rules, /me pinning and permissions.
func TestHandlersOverGRPC(t *testing.T) {
	pool := pg.TestPool(t, hr.Migrations(), "migrations", hr.Schema)
	pg.TruncateSchema(t, pool, hr.Schema)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO hr.payroll_settings (id) VALUES (1) ON CONFLICT DO NOTHING;
		INSERT INTO hr.employment_statuses (code, name) VALUES ('permanent','Karyawan Tetap'),('contract','Kontrak'),('probation','Masa Percobaan') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	d := app.Deps{Store: postgres.NewStore(pool), Identity: noIdentity{}, Files: storage.NewLocal(t.TempDir()), Events: events.Noop{},
		Clock: clock.At("2026-10-21T01:20:00Z"), Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))}
	r := docs.Renderer{CompanyName: "Mixist"}
	if err := seed.New(d, r).Run(ctx, []app.UserInfo{{ID: "user-tim", Email: "tim@mixist.id"}}); err != nil {
		t.Fatal(err)
	}
	leaves, ann := app.NewLeaves(d), app.NewAnnouncements(d)
	svc := &grpcapi.Services{Deps: d, People: app.NewPeople(d), Attendance: app.NewAttendance(d, r), Schedules: app.NewSchedules(d),
		Leaves: leaves, Overtime: app.NewOvertime(d), Contracts: app.NewContracts(d), Announcements: ann, Lifecycle: app.NewLifecycle(d),
		Payroll: app.NewPayroll(d, r), Loans: app.NewLoans(d), Self: app.NewSelfService(d, leaves, ann), Reports: app.NewReports(d),
		Dashboard: app.NewDashboard(d, leaves)}

	priv, err := authn.KeyPair("", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	pub, err := authn.PublicKey("", "", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpcx.NewServer(grpcx.ServerOptions{Logger: d.Logger, Verifier: authn.NewVerifier(pub)})
	grpcapi.Register(srv, svc)
	lis := bufconn.Listen(1 << 20)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	issuer := authn.NewIssuer(priv, time.Hour)
	as := func(userID string, perms ...string) context.Context {
		tok, _, err := issuer.Issue(authn.Principal{UserID: userID, Role: authn.RoleMember, Perms: perms}, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		return grpcx.WithBearer(ctx, tok)
	}
	code := func(err error) apperr.Code { return grpcx.FromStatus(err).Code }

	directory := hrv1.NewEmployeeServiceClient(conn)
	list, err := directory.ListEmployees(ctx, &hrv1.ListEmployeesRequest{})
	if err != nil || list.Total != 29 {
		t.Fatalf("service-to-service directory: %v %v", list.GetTotal(), err)
	}
	if _, err := directory.ListEmployees(metadata.AppendToOutgoingContext(ctx, grpcx.EdgeHeader, "1"), &hrv1.ListEmployeesRequest{}); code(err) != apperr.Unauthenticated {
		t.Fatalf("anonymous edge call: %v", err)
	}
	edgeSelf := metadata.AppendToOutgoingContext(as("user-tim", authn.PermSelfService), grpcx.EdgeHeader, "1")
	if _, err := directory.ListEmployees(edgeSelf, &hrv1.ListEmployeesRequest{}); code(err) != apperr.Unauthorized {
		t.Fatalf("gateway calls need hr.read: %v", err)
	}
	page, err := directory.ListEmployees(as("user-tim", authn.PermSelfService), &hrv1.ListEmployeesRequest{PageSize: 10, PageToken: "3"})
	if err != nil || len(page.Employees) != 9 || page.NextPageToken != "" {
		t.Fatalf("paging: %v %v", page, err)
	}
	me, err := directory.GetEmployeeByUser(ctx, &hrv1.GetEmployeeByUserRequest{UserId: "user-tim"})
	if err != nil || me.FullName != "Gita Permata" || me.DepartmentName != "Social Media" || me.ManagerUserId != nil || me.ManagerId == nil {
		t.Fatalf("by user: %v %v", me, err)
	}
	if _, err := directory.GetEmployeeByUser(ctx, &hrv1.GetEmployeeByUserRequest{UserId: "user-nobody"}); code(err) != apperr.NotFound {
		t.Fatalf("unlinked user: %v", err)
	}
	sum, err := directory.GetAttendanceSummary(ctx, &hrv1.GetAttendanceSummaryRequest{EmployeeIds: []string{me.Id}, Month: "2026-09"})
	if err != nil || sum.Summaries[0].WorkingDays != 22 || sum.Summaries[0].PresentDays == 0 {
		t.Fatalf("attendance summary: %v %v", sum, err)
	}

	self := hrv1.NewSelfServiceServiceClient(conn)
	home, err := self.GetMyHome(as("user-tim", authn.PermSelfService), &emptypb.Empty{})
	if err != nil || home.Employee.FullName != "Gita Permata" || home.TodayShift == nil || len(home.Week) != 7 || home.LeaveBalance == nil {
		t.Fatalf("home: %v %v", home, err)
	}
	slips, err := self.ListMyPayslips(as("user-tim", authn.PermSelfService), &hrv1.ListMyRequest{})
	if err != nil || len(slips.Payslips) < 9 || slips.Payslips[0].Employee.Id != me.Id {
		t.Fatalf("my payslips: %v %v", len(slips.GetPayslips()), err)
	}
	dash, err := self.GetMyDashboard(as("user-tim", authn.PermSelfService), &hrv1.MyDashboardRequest{Month: "2026-09"})
	if err != nil || len(dash.Days) != 30 || len(dash.Trend) != 12 || dash.LatestPayslip == nil {
		t.Fatalf("my dashboard: %v", err)
	}
	if _, err := hrv1.NewHrReportServiceClient(conn).GetHrDashboard(as("user-tim", authn.PermSelfService), &hrv1.HrDashboardRequest{}); code(err) != apperr.Unauthorized {
		t.Fatalf("hr dashboard needs hr.read: %v", err)
	}
	pdf, err := self.GetMyPayslipPdf(as("user-tim", authn.PermSelfService), &hrv1.PayslipRequest{PayslipId: slips.Payslips[0].Id})
	if err != nil || pdf.ContentType != "application/pdf" || len(pdf.Data) < 1000 {
		t.Fatalf("my payslip pdf: %v", err)
	}
	if _, err := self.GetMyHome(as("user-nobody", authn.PermSelfService), &emptypb.Empty{}); code(err) != apperr.NotFound {
		t.Fatalf("unlinked account: %v", err)
	}
	leave, err := self.CreateMyLeave(as("user-tim", authn.PermSelfService), &hrv1.CreateLeaveRequest{EmployeeId: list.Employees[0].Id,
		LeaveType: "annual", StartDate: "2027-02-08", EndDate: "2027-02-09", Reason: "Keluarga"})
	if err != nil || leave.Employee.Id != me.Id || leave.TotalDays != 2 || !leave.CanCancel {
		t.Fatalf("my leave is pinned to the caller: %v %v", leave, err)
	}

	people := hrv1.NewPeopleServiceClient(conn)
	if _, err := people.CreateEmployee(as("user-tim", authn.PermSelfService), &hrv1.EmployeeInput{FullName: "X", JoinDate: "2026-01-01"}); code(err) != apperr.Unauthorized {
		t.Fatalf("self.service cannot create employees: %v", err)
	}
	chart, err := people.GetOrgChart(as("user-tim", authn.PermSelfService), &hrv1.GetOrgChartRequest{})
	if err != nil || chart.Total != 29 || len(chart.Roots) != 1 {
		t.Fatalf("org chart: %v %v", chart.GetTotal(), err)
	}

	payroll := hrv1.NewPayrollServiceClient(conn)
	if _, err := payroll.ListPayrollRuns(as("user-tim", authn.PermSelfService), &hrv1.ListPayrollRunsRequest{}); code(err) != apperr.Unauthorized {
		t.Fatalf("payroll needs hr.payroll: %v", err)
	}
	runs, err := payroll.ListPayrollRuns(as("user-hr", authn.PermPayroll), &hrv1.ListPayrollRunsRequest{})
	if err != nil || len(runs.Runs) != 15 {
		t.Fatalf("runs: %v %v", runs, err)
	}
	report, err := hrv1.NewHrReportServiceClient(conn).GetHrisReport(as("user-hr", authn.PermHRRead), &hrv1.HrisReportRequest{Month: 9, Year: 2026})
	if err != nil || report.TotalActive != 29 || report.AttendanceRecords == 0 || len(report.ByDepartment) != 8 {
		t.Fatalf("report: %v %v", report, err)
	}
	export, err := hrv1.NewAttendanceServiceClient(conn).ExportAttendance(as("user-hr", authn.PermHRRead), &hrv1.ExportAttendanceRequest{StartDate: "2026-09-01", EndDate: "2026-09-30"})
	if err != nil || len(export.Data) < 1000 {
		t.Fatalf("export: %v", err)
	}
	listFilters(t, conn, as("user-hr", authn.PermHRRead), *me.DepartmentId, me.DepartmentName)
}

// listFilters checks the dashboard drill filters on the HR list endpoints.
func listFilters(t *testing.T, conn *grpc.ClientConn, ctx context.Context, deptID, deptName string) {
	t.Helper()
	directory := hrv1.NewEmployeeServiceClient(conn)
	all, err := directory.ListEmployees(ctx, &hrv1.ListEmployeesRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		req  *hrv1.ListEmployeesRequest
		want int
	}{
		{"gender", &hrv1.ListEmployeesRequest{Gender: "female"}, 0},
		{"level", &hrv1.ListEmployeesRequest{Level: "director"}, 0},
		{"status", &hrv1.ListEmployeesRequest{EmploymentStatus: "contract"}, 0},
		{"position", &hrv1.ListEmployeesRequest{PositionId: all.Employees[0].GetPositionId()}, 0},
		{"no match", &hrv1.ListEmployeesRequest{Gender: "female", Level: "nobody"}, -1},
	} {
		got, err := directory.ListEmployees(ctx, tc.req)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if tc.want < 0 && got.Total != 0 || tc.want == 0 && (got.Total == 0 || got.Total >= all.Total) {
			t.Fatalf("%s filter: %d of %d", tc.name, got.Total, all.Total)
		}
	}
	directors, _ := directory.ListEmployees(ctx, &hrv1.ListEmployeesRequest{Level: "director"})
	if directors.Total != 2 {
		t.Fatalf("seeded directors: %d", directors.Total)
	}

	inDept := func(name string, refs []*hrv1.EmployeeRef) {
		t.Helper()
		if len(refs) == 0 {
			t.Fatalf("%s: department filter returned nothing", name)
		}
		for _, r := range refs {
			if r.DepartmentName != deptName {
				t.Fatalf("%s: %s is in %q, want %q", name, r.FullName, r.DepartmentName, deptName)
			}
		}
	}
	att := hrv1.NewAttendanceServiceClient(conn)
	every, err := att.ListAttendance(ctx, &hrv1.ListAttendanceRequest{StartDate: "2026-09-01", EndDate: "2026-09-30", PageSize: 500})
	if err != nil {
		t.Fatal(err)
	}
	dept, err := att.ListAttendance(ctx, &hrv1.ListAttendanceRequest{StartDate: "2026-09-01", EndDate: "2026-09-30", PageSize: 500, DepartmentId: deptID})
	if err != nil || dept.Total >= every.Total {
		t.Fatalf("attendance by department: %d of %d, %v", dept.GetTotal(), every.Total, err)
	}
	refs := make([]*hrv1.EmployeeRef, 0, len(dept.Records))
	for _, r := range dept.Records {
		refs = append(refs, r.Employee)
	}
	inDept("attendance", refs)

	leaves, err := hrv1.NewLeaveServiceClient(conn).ListLeaves(ctx, &hrv1.ListLeavesRequest{DepartmentId: deptID, PageSize: 200})
	if err != nil {
		t.Fatal(err)
	}
	refs = refs[:0]
	for _, l := range leaves.Leaves {
		refs = append(refs, l.Employee)
	}
	if len(refs) > 0 {
		inDept("leaves", refs)
	}
	ot, err := hrv1.NewOvertimeServiceClient(conn).ListOvertime(ctx, &hrv1.ListOvertimeRequest{DepartmentId: deptID})
	if err != nil {
		t.Fatal(err)
	}
	refs = refs[:0]
	for _, o := range ot.Overtime {
		refs = append(refs, o.Employee)
	}
	if len(refs) > 0 {
		inDept("overtime", refs)
	}
	none, err := att.ListAttendance(ctx, &hrv1.ListAttendanceRequest{DepartmentId: "no-such-department"})
	if err != nil || none.Total != 0 {
		t.Fatalf("unknown department: %v %v", none.GetTotal(), err)
	}
}
