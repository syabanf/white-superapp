package seed

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"os"
	"strings"
	"testing"

	hrv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/hr/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/performance"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeHR serves a small WHITE directory over real gRPC so the seed runs
// end to end through the hr client.
type fakeHR struct {
	hrv1.UnimplementedEmployeeServiceServer
}

func sp(s string) *string { return &s }

var hrDepts = []*hrv1.Department{
	{Id: "d_soc", Code: "SOC", Name: "Social Media", HeadEmployeeId: sp("e1"), IsActive: true},
	{Id: "d_seo", Code: "SEO", Name: "SEO", HeadEmployeeId: sp("e4"), IsActive: true},
	{Id: "d_fin", Code: "FIN", Name: "Finance", IsActive: true},
}

func hrEmployees() []*hrv1.Employee {
	mk := func(id, name, dept, manager string) *hrv1.Employee {
		e := &hrv1.Employee{Id: id, FullName: name, DepartmentId: sp(dept), Status: hrv1.EmploymentStatus_ACTIVE, UserId: sp("u_" + id)}
		if manager != "" {
			e.ManagerId = sp(manager)
		}
		return e
	}
	return []*hrv1.Employee{mk("e1", "Hana", "d_soc", ""), mk("e2", "Rani", "d_soc", "e1"), mk("e3", "Dimas", "d_soc", "e1"),
		mk("e4", "Sari", "d_seo", ""), mk("e5", "Bayu", "d_seo", "e4"), mk("e6", "Wulan", "d_fin", "")}
}

func (fakeHR) ListEmployees(context.Context, *hrv1.ListEmployeesRequest) (*hrv1.ListEmployeesResponse, error) {
	return &hrv1.ListEmployeesResponse{Employees: hrEmployees()}, nil
}

func (fakeHR) ListDepartments(context.Context, *hrv1.ListDepartmentsRequest) (*hrv1.ListDepartmentsResponse, error) {
	return &hrv1.ListDepartmentsResponse{Departments: hrDepts}, nil
}

func (fakeHR) GetEmployeeByUser(_ context.Context, req *hrv1.GetEmployeeByUserRequest) (*hrv1.Employee, error) {
	for _, e := range hrEmployees() {
		if e.GetUserId() == req.UserId {
			return e, nil
		}
	}
	return nil, status.Error(codes.NotFound, "karyawan tidak ditemukan")
}

func (fakeHR) GetAttendanceSummary(_ context.Context, req *hrv1.GetAttendanceSummaryRequest) (*hrv1.GetAttendanceSummaryResponse, error) {
	out := &hrv1.GetAttendanceSummaryResponse{}
	for _, id := range req.EmployeeIds {
		out.Summaries = append(out.Summaries, &hrv1.AttendanceSummary{EmployeeId: id, WorkingDays: 21, PresentDays: 20, LateDays: 2, LateMinutes: 25})
	}
	return out, nil
}

func TestSeedEndToEnd(t *testing.T) {
	url := os.Getenv("WHITE_TEST_DATABASE_URL")
	pool := pg.TestPool(t, performance.Migrations(), "migrations", performance.Schema)
	pg.TruncateSchema(t, pool, performance.Schema)
	raw, err := fs.ReadFile(performance.Migrations(), "00002_reference_data.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), strings.SplitN(string(raw), "-- +goose Down", 2)[0]); err != nil {
		t.Fatal(err)
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer()
	hrv1.RegisterEmployeeServiceServer(srv, fakeHR{})
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	t.Setenv("DATABASE_URL", url)
	t.Setenv("HR_ADDR", lis.Addr().String())
	t.Setenv("IDENTITY_ADDR", "127.0.0.1:1")
	t.Setenv("INTEGRATIONS_ADDR", "127.0.0.1:1")
	t.Setenv("PERFORMANCE_UPLOAD_DIR", t.TempDir())

	counts := func() string {
		var out string
		for _, tbl := range []string{"kpi_indicators", "kpi_department_indicators", "department_tasks", "department_task_occurrences",
			"logbook_entries", "daily_recaps", "review_cycles", "performance_reviews", "kpi_scorecards", "feedback_summaries", "improvements", "monthly_recaps"} {
			var n int
			if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM performance."+tbl).Scan(&n); err != nil {
				t.Fatal(err)
			}
			if n == 0 && tbl != "improvements" {
				t.Errorf("%s is empty", tbl)
			}
			out += fmt.Sprintf("%s=%d ", tbl, n)
		}
		return out
	}
	if err := Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	first := counts()
	if err := Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if second := counts(); second != first {
		t.Fatalf("seed not idempotent:\n%s\n%s", first, second)
	}
	t.Log(first)
}
