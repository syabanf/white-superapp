package app_test

import (
	"math"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/app"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
)

func TestTimesheetSpreadsPayrollCostOverProjects(t *testing.T) {
	e := setup(t)
	o := e.org()
	ts := app.NewTimesheet(e.d)
	hrApprover := e.actor("user-wulan", authn.PermHRRead, authn.PermHRApprove, authn.PermPayroll, authn.PermSelfService)
	ayu, budi := e.actor("user-ayu", authn.PermSelfService), e.actor("user-budi", authn.PermSelfService)

	for _, in := range []struct {
		who                   app.Actor
		day, project, name, h string
	}{
		{ayu, "2026-09-10", "p-kopi", "Kopi Nusantara", "6"},
		{ayu, "2026-09-10", "", "", "2"},
		{ayu, "2026-09-11", "p-villa", "Bali Villa", "4"},
		{budi, "2026-09-10", "p-kopi", "Kopi Nusantara", "8"},
	} {
		_, err := ts.Save(e.ctx, in.who, app.TimesheetInput{WorkDate: in.day, ProjectID: in.project, ProjectName: in.name, Hours: in.h})
		e.must(err)
	}
	_, err := ts.Save(e.ctx, ayu, app.TimesheetInput{WorkDate: "2026-09-10", Hours: "17"})
	wantCode(t, err, apperr.Validation) // 25 hours that day
	_, err = ts.Save(e.ctx, ayu, app.TimesheetInput{WorkDate: "2026-09-12", Hours: "1.1"})
	wantCode(t, err, apperr.Validation) // quarter hours only
	_, err = ts.Save(e.ctx, ayu, app.TimesheetInput{WorkDate: "2026-12-01", Hours: "1"})
	wantCode(t, err, apperr.Validation) // future

	mine, err := ts.List(e.ctx, ayu, domain.TimesheetFilter{EmployeeID: o.budi, From: "2026-09-01", To: "2026-09-30"})
	e.must(err)
	if len(mine) != 3 {
		t.Fatalf("an employee sees only their own entries, got %d", len(mine))
	}
	for _, a := range []app.Actor{ayu, budi} {
		n, err := ts.Submit(e.ctx, a, "2026-09-01", "2026-09-30")
		e.must(err)
		if n == 0 {
			t.Fatal("nothing submitted")
		}
	}
	_, err = ts.Save(e.ctx, ayu, app.TimesheetInput{ID: mine[0].ID, WorkDate: "2026-09-10", Hours: "1"})
	wantCode(t, err, apperr.Conflict)

	wulanEntry, err := ts.Save(e.ctx, hrApprover, app.TimesheetInput{WorkDate: "2026-09-10", Hours: "8"})
	e.must(err)
	_, err = ts.Submit(e.ctx, hrApprover, "2026-09-01", "2026-09-30")
	e.must(err)
	_, err = ts.Decide(e.ctx, hrApprover, []string{wulanEntry.ID}, true, "")
	wantCode(t, err, apperr.Unauthorized)

	queue, err := ts.List(e.ctx, hrApprover, domain.TimesheetFilter{Status: "submitted", From: "2026-09-01", To: "2026-09-30"})
	e.must(err)
	var ids []string
	for _, q := range queue {
		if q.EmployeeID != o.wulan {
			ids = append(ids, q.ID)
		}
	}
	n, err := ts.Decide(e.ctx, hrApprover, ids, true, "")
	e.must(err)
	if n != 4 {
		t.Fatalf("approved %d", n)
	}

	_, err = ts.ProjectCosts(e.ctx, "2026-09")
	wantCode(t, err, apperr.Conflict) // payroll not approved yet

	for _, s := range []app.SalaryInput{
		{EmployeeID: o.budi, BaseSalary: "15000000", PTKPStatus: "TK/0", EffectiveDate: "2024-01-15"},
		{EmployeeID: o.ayu, BaseSalary: "10000000", PTKPStatus: "TK/0", EffectiveDate: "2024-01-15"},
	} {
		_, err := e.payroll.CreateSalary(e.ctx, hrApprover, s)
		e.must(err)
	}
	run, err := e.payroll.CreateRun(e.ctx, hrApprover, 9, 2026, "")
	e.must(err)
	_, _, err = e.payroll.Calculate(e.ctx, hrApprover, run.ID, false)
	e.must(err)
	_, err = e.payroll.Approve(e.ctx, hrApprover, run.ID)
	e.must(err)

	slips, err := e.d.Store.ListPayslips(e.ctx, domain.PayslipFilter{RunID: run.ID})
	e.must(err)
	cost := map[string]float64{}
	for _, p := range slips {
		cost[p.EmployeeID] = p.GrossSalary + p.BpjsTkJhtEmployer + p.BpjsTkJpEmployer + p.BpjsTkJkkEmployer + p.BpjsTkJkmEmployer + p.BpjsKesEmployer
	}
	costs, err := ts.ProjectCosts(e.ctx, "2026-09")
	e.must(err)
	want := map[string]float64{
		"p-kopi":  math.Round(cost[o.ayu]*6/12) + math.Round(cost[o.budi]),
		"p-villa": math.Round(cost[o.ayu] * 4 / 12),
	}
	if len(costs) != 2 {
		t.Fatalf("%+v", costs)
	}
	for _, c := range costs {
		if c.Cost != want[c.ProjectID] || c.Cost <= 0 {
			t.Fatalf("%s: got %.0f want %.0f", c.ProjectID, c.Cost, want[c.ProjectID])
		}
	}
	if costs[0].ProjectID != "p-kopi" || costs[0].Hours != 14 {
		t.Fatalf("%+v", costs[0])
	}
}

func TestAssetsLendAndReturn(t *testing.T) {
	e := setup(t)
	o := e.org()
	assets := app.NewAssets(e.d)
	hrWriter := e.actor("user-wulan", authn.PermHRRead, authn.PermHRWrite, authn.PermSelfService)
	ayu := e.actor("user-ayu", authn.PermSelfService)

	laptop, err := assets.Save(e.ctx, hrWriter, app.AssetInput{AssetCode: "LTP-001", Name: "MacBook Air", Category: "laptop", SerialNo: "C02X", PurchaseValue: "18000000"})
	e.must(err)
	_, err = assets.Save(e.ctx, hrWriter, app.AssetInput{AssetCode: "LTP-001", Name: "Lain", Category: "laptop"})
	wantCode(t, err, apperr.Conflict)
	_, err = assets.Save(e.ctx, ayu, app.AssetInput{AssetCode: "X", Name: "X", Category: "laptop"})
	wantCode(t, err, apperr.Unauthorized)

	lent, err := assets.Assign(e.ctx, hrWriter, laptop.ID, o.ayu, "2026-10-01", "Mulus")
	e.must(err)
	if lent.Status != "assigned" || deref(lent.EmployeeID) != o.ayu || e.identity.count("HR_ASSET_ASSIGNED") != 1 {
		t.Fatalf("%+v", lent)
	}
	_, err = assets.Assign(e.ctx, hrWriter, laptop.ID, o.budi, "2026-10-02", "")
	wantCode(t, err, apperr.Conflict)
	_, err = assets.Save(e.ctx, hrWriter, app.AssetInput{ID: laptop.ID, AssetCode: "LTP-001", Name: "MacBook Air", Category: "laptop", Status: "retired"})
	wantCode(t, err, apperr.Conflict)

	held, err := assets.List(e.ctx, ayu, domain.AssetFilter{})
	e.must(err)
	if len(held) != 1 {
		t.Fatalf("ayu holds %d", len(held))
	}
	back, err := assets.Return(e.ctx, hrWriter, laptop.ID, "2026-10-20", "Lecet di tutup")
	e.must(err)
	if back.Status != "available" || back.EmployeeID != nil {
		t.Fatalf("%+v", back)
	}
	_, history, err := assets.Get(e.ctx, hrWriter, laptop.ID)
	e.must(err)
	if len(history) != 1 || deref(history[0].ReturnedOn) != "2026-10-20" || history[0].ConditionIn != "Lecet di tutup" {
		t.Fatalf("%+v", history)
	}
}

func TestTrainingParticipantsAndResults(t *testing.T) {
	e := setup(t)
	o := e.org()
	tr := app.NewTrainings(e.d)
	hrWriter := e.actor("user-wulan", authn.PermHRRead, authn.PermHRWrite, authn.PermSelfService)
	ayu := e.actor("user-ayu", authn.PermSelfService)

	_, err := tr.Save(e.ctx, hrWriter, app.TrainingInput{Title: "Meta Ads Lanjutan", StartDate: "2026-11-10", EndDate: "2026-11-09"})
	wantCode(t, err, apperr.Validation)
	course, err := tr.Save(e.ctx, hrWriter, app.TrainingInput{Title: "Meta Ads Lanjutan", Provider: "Meta Blueprint", StartDate: "2026-11-10", EndDate: "2026-11-11", Cost: "2500000"})
	e.must(err)
	_, err = tr.SetParticipants(e.ctx, hrWriter, course.ID, []string{o.ayu, o.budi, o.ayu})
	e.must(err)
	if e.identity.count("HR_TRAINING") != 2 {
		t.Fatalf("notified %d", e.identity.count("HR_TRAINING"))
	}
	e.must(tr.RecordResult(e.ctx, hrWriter, course.ID, app.ParticipantResult{EmployeeID: o.ayu, Status: "attended", Score: "88", Certificate: "MB-123"}))
	wantCode(t, tr.RecordResult(e.ctx, hrWriter, course.ID, app.ParticipantResult{EmployeeID: o.ayu, Status: "attended", Score: "101"}), apperr.Validation)

	people, err := tr.SetParticipants(e.ctx, hrWriter, course.ID, []string{o.ayu})
	e.must(err)
	if len(people) != 1 || people[0].Score == nil || *people[0].Score != 88 {
		t.Fatalf("result lost: %+v", people)
	}
	_, own, err := tr.Get(e.ctx, ayu, course.ID)
	e.must(err)
	if len(own) != 1 || own[0].EmployeeID != o.ayu {
		t.Fatalf("%+v", own)
	}
	list, err := tr.List(e.ctx, e.actor("user-budi", authn.PermSelfService), domain.TrainingFilter{})
	e.must(err)
	if len(list) != 0 {
		t.Fatal("budi was removed from the course")
	}
	_, _, err = tr.Get(e.ctx, e.actor("user-budi", authn.PermSelfService), course.ID)
	wantCode(t, err, apperr.NotFound)
}
