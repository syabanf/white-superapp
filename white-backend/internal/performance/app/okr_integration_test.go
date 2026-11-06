package app_test

import (
	"math"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

func TestOKRBoardPermissionsAndCheckIns(t *testing.T) {
	f := setup(t)
	const q = "2026-Q4"

	_, err := f.okr.SaveObjective(hana, app.ObjectiveInput{Period: q, Level: domain.OKRCompany, Title: "Tumbuh 30%"})
	code(t, err, apperr.Unauthorized) // company OKRs are HR's
	company, err := f.okr.SaveObjective(hrAdmin, app.ObjectiveInput{Period: q, Level: domain.OKRCompany, Title: "Tumbuh 30%"})
	must(t, err)
	social, err := f.okr.SaveObjective(hana, app.ObjectiveInput{Period: q, Level: domain.OKRDepartment, DepartmentID: deptSocial, ParentID: company.ID, Title: "Engagement naik"})
	must(t, err)
	_, err = f.okr.SaveObjective(hana, app.ObjectiveInput{Period: q, Level: domain.OKRDepartment, DepartmentID: deptSEO, Title: "Bukan departemen Hana"})
	code(t, err, apperr.Unauthorized)
	_, err = f.okr.SaveObjective(hana, app.ObjectiveInput{Period: "2026-Q1", Level: domain.OKRDepartment, DepartmentID: deptSocial, ParentID: company.ID, Title: "Induk beda periode"})
	code(t, err, apperr.Validation)

	// Rani writes her own; her head Hana may edit it, Bayu may not.
	ranis, err := f.okr.SaveObjective(rani, app.ObjectiveInput{Period: q, Level: domain.OKREmployee, Title: "Konten video"})
	must(t, err)
	if deref(ranis.EmployeeID) != "emp_rani" {
		t.Fatalf("own objective %+v", ranis)
	}
	reach, err := f.okr.SaveKeyResult(hana, app.KeyResultInput{ObjectiveID: ranis.ID, Title: "Video per bulan", Metric: "number", Start: 2, Target: 10})
	must(t, err)
	_, err = f.okr.SaveKeyResult(bayu, app.KeyResultInput{ObjectiveID: ranis.ID, Title: "x", Metric: "number", Start: 0, Target: 1})
	code(t, err, apperr.Unauthorized)
	_, err = f.okr.SaveKeyResult(rani, app.KeyResultInput{ObjectiveID: ranis.ID, Title: "Sama", Metric: "number", Start: 5, Target: 5})
	code(t, err, apperr.Validation)
	churn, err := f.okr.SaveKeyResult(hana, app.KeyResultInput{ObjectiveID: social.ID, Title: "Unfollow per minggu", Metric: "number", Start: 100, Target: 40})
	must(t, err)

	_, err = f.okr.CheckIn(rani, reach.ID, 6, 7, "Setengah jalan")
	must(t, err)
	_, err = f.okr.CheckIn(rani, reach.ID, 6, 11, "")
	code(t, err, apperr.Validation)
	_, err = f.okr.CheckIn(hana, churn.ID, 70, 5, "")
	must(t, err)

	view, err := f.okr.Board(bayu, q) // everyone sees the board
	must(t, err)
	board := view.Objectives
	if len(board) != 3 || board[0].Level != domain.OKRCompany || board[0].CanEdit {
		t.Fatalf("board %+v", board)
	}
	if view.CanCompany || len(view.Departments) != 0 || len(view.Employees) != 1 || view.Employees[0].Name != "Bayu" {
		t.Fatalf("bayu writes only his own: %+v", view)
	}
	head, err := f.okr.Board(hana, q)
	must(t, err)
	if len(head.Departments) != 1 || len(head.Employees) != 3 {
		t.Fatalf("hana writes for Social Media and her team: %+v %+v", head.Departments, head.Employees)
	}
	for _, o := range board {
		switch o.ID {
		case ranis.ID:
			if o.OwnerName != "Rani" || math.Abs(o.Progress-0.5) > 1e-9 || o.KeyResults[0].Confidence != 7 {
				t.Fatalf("rani %+v", o)
			}
		case social.ID:
			if o.OwnerName != "Social Media" || math.Abs(o.Progress-0.5) > 1e-9 {
				t.Fatalf("a falling target counts the same way: %+v", o)
			}
		}
	}
	history, err := f.okr.Checkins(dimas, reach.ID)
	must(t, err)
	if len(history) != 1 || history[0].Note != "Setengah jalan" {
		t.Fatalf("%+v", history)
	}

	_, err = f.okr.SaveObjective(rani, app.ObjectiveInput{ID: ranis.ID, Title: "Konten video", Status: "done"})
	must(t, err)
	_, err = f.okr.CheckIn(rani, reach.ID, 8, 8, "")
	code(t, err, apperr.Conflict)
	must(t, f.okr.DeleteObjective(hrAdmin, company.ID))
	view, err = f.okr.Board(hana, q)
	must(t, err)
	board = view.Objectives
	if len(board) != 2 || board[0].ParentID != nil {
		t.Fatalf("children survive their parent: %+v", board)
	}
}
