package grpcapi

import (
	"context"

	hrv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/hr/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/app"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
)

func (h *reportServer) GetHrDashboard(ctx context.Context, req *hrv1.HrDashboardRequest) (*hrv1.HrDashboard, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	d, err := h.s.Dashboard.HR(ctx, a, app.DashboardQuery{From: req.From, To: req.To, Compare: req.Compare,
		Filter: domain.DashFilter{DepartmentID: req.DepartmentId, EmploymentStatus: req.EmploymentStatus, PositionID: req.PositionId}})
	if err != nil {
		return nil, err
	}
	return hrDashboardPB(d), nil
}

func (h *selfServer) GetMyDashboard(ctx context.Context, req *hrv1.MyDashboardRequest) (*hrv1.MyDashboard, error) {
	a, _, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	d, err := h.s.Dashboard.Me(ctx, a, req.Month)
	if err != nil {
		return nil, err
	}
	e := d.Employee
	out := &hrv1.MyDashboard{Employee: refPB(e.ID, e.FullName, e.EmployeeNo, e.DepartmentName), Month: d.Month, Today: d.Today,
		Stats: myStatsPB(d.Stats), Previous: myStatsPB(d.Prev), LeaveBalance: balancePB(d.Balance), IsManager: d.IsManager}
	for _, x := range d.Days {
		out.Days = append(out.Days, &hrv1.MyDashboardDay{Date: x.Date, Status: x.Status, ClockIn: x.ClockIn, ClockOut: x.ClockOut,
			WorkHours: x.WorkHours, LateMinutes: int32(x.LateMinutes), Note: x.Note, OvertimeHours: x.OvertimeHours})
	}
	for _, m := range d.Trend {
		out.Trend = append(out.Trend, &hrv1.MyDashboardMonth{Month: m.Month, OnTimeRate: m.OnTimeRate, TeamOnTimeRate: m.TeamOnTimeRate,
			WorkHours: m.WorkHours, OvertimeHours: m.OvertimeHours, NetPay: money(m.NetPay), GrossPay: money(m.GrossPay),
			PresentDays: int32(m.PresentDays), LateDays: int32(m.LateDays), LeaveDays: m.LeaveDays})
	}
	for _, w := range d.Weeks {
		out.Weeks = append(out.Weeks, &hrv1.MyDashboardWeek{WeekStart: w.WeekStart, WorkHours: w.WorkHours, Days: int32(w.Days), TargetHours: w.Target})
	}
	for _, l := range d.LeaveHistory {
		out.LeaveHistory = append(out.LeaveHistory, h.s.leavePB(ctx, a, l, false))
	}
	if p := d.LatestPayslip; p != nil {
		allowances := p.FixedAllowance + p.VariableAllowance + p.TransportAllowance + p.MealAllowance + p.HousingAllowance + p.Thr + p.Bonus
		bpjs := p.BpjsTkJhtDeduction + p.BpjsTkJpDeduction + p.BpjsKesDeduction + p.TaperaDeduction
		other := p.UnpaidLeaveDeduction + p.LateDeduction
		out.LatestPayslip = &hrv1.MyPayslipSummary{PayslipId: p.ID, PeriodMonth: int32(p.PeriodMonth), PeriodYear: int32(p.PeriodYear),
			BaseSalary: money(p.BaseSalary), Allowances: money(allowances), OvertimePay: money(p.OvertimePay), GrossSalary: money(p.GrossSalary),
			Bpjs: money(bpjs), Pph21: money(p.Pph21Deduction), LoanDeduction: money(p.LoanDeduction), OtherDeductions: money(other),
			TotalDeductions: money(p.TotalDeductions), NetSalary: money(p.NetSalary), PresentDays: p.PresentDays, WorkingDays: p.WorkingDays}
	}
	for _, p := range d.Pending {
		out.Pending = append(out.Pending, &hrv1.MyPendingRequest{Kind: p.Kind, Id: p.ID, Type: p.Type, Date: p.Date, EndDate: p.EndDate,
			Amount: p.Amount, Status: p.Status, CreatedAt: ts(p.CreatedAt)})
	}
	for _, m := range d.Team {
		out.TeamToday = append(out.TeamToday, &hrv1.MyTeamMember{EmployeeId: m.EmployeeID, FullName: m.FullName, PositionName: m.PositionName,
			Status: m.Status, ClockIn: m.ClockIn, Note: m.Note})
	}
	return out, nil
}

func myStatsPB(s app.MyStats) *hrv1.MyDashboardStats {
	return &hrv1.MyDashboardStats{ScheduledDays: int32(s.ScheduledDays), PresentDays: int32(s.PresentDays), LateDays: int32(s.LateDays),
		AbsentDays: int32(s.AbsentDays), LeaveDays: int32(s.LeaveDays), OnTimeRate: s.OnTimeRate, AttendanceRate: s.AttendanceRate,
		WorkHours: s.WorkHours, OvertimeHours: s.OvertimeHours, LateMinutes: int32(s.LateMinutes), AvgClockIn: s.AvgClockIn}
}

func hrDashboardPB(d *app.HrDashboard) *hrv1.HrDashboard {
	out := &hrv1.HrDashboard{From: d.From, To: d.To, CompareFrom: d.CompareFrom, CompareTo: d.CompareTo, Bucket: d.Bucket,
		PayrollVisible: d.PayrollVisible, Current: kpisPB(d.Current), Previous: kpisPB(d.Previous),
		Buckets: bucketsPB(d.Buckets), PreviousBuckets: bucketsPB(d.PreviousBuckets), Months: bucketsPB(d.Months),
		LateByWeekday: cellsPB(d.LateByWeekday), LateByDay: cellsPB(d.LateByDay), AbsenceReasons: slicesPB(d.AbsenceReasons),
		ClockInOffsets: slicesPB(d.ClockInOffsets), TopLate: empStatsPB(d.TopLate), TopOvertime: empStatsPB(d.TopOvertime),
		LeaveByType: slicesPB(d.LeaveByType), PendingAging: slicesPB(d.PendingAging), ByStatus: slicesPB(d.ByStatus),
		ByPosition: slicesPB(d.ByPosition), ByLevel: slicesPB(d.ByLevel), ByContractType: slicesPB(d.ByContractType),
		ByGender: slicesPB(d.ByGender), AgeBands: slicesPB(d.AgeBands), TenureBands: slicesPB(d.TenureBands),
		LoansOutstanding: money(d.LoansOutstanding), PendingCount: int32(len(d.Pending))}
	for _, m := range d.DeptMonths {
		out.DeptMonths = append(out.DeptMonths, &hrv1.HrDashDeptMonth{Month: m.Month, DepartmentId: m.DepartmentID, Headcount: int32(m.Headcount),
			AttendanceRecords: int32(m.Records), AttendanceRate: m.AttendanceRate, Late: int32(m.Late), LeaveDays: m.LeaveDays,
			OvertimeHours: m.OvertimeHours, OvertimeCost: money(m.OvertimeCost), PayrollGross: money(m.PayrollGross)})
	}
	for _, x := range d.Departments {
		out.Departments = append(out.Departments, &hrv1.HrDashDept{DepartmentId: x.DepartmentID, Name: x.Name, Headcount: int32(x.Headcount),
			Joiners: int32(x.Joiners), Leavers: int32(x.Leavers), AttendanceRecords: int32(x.Records), AttendanceRate: x.AttendanceRate,
			Late: int32(x.Late), LateRate: x.LateRate, LateMinutes: int32(x.LateMinutes), Absent: int32(x.Absent), AvgWorkHours: x.AvgWorkHours,
			LeaveDays: x.LeaveDays, OvertimeHours: x.OvertimeHours, OvertimeCost: money(x.OvertimeCost), PayrollGross: money(x.PayrollGross),
			PayrollEmployerCost: money(x.PayrollEmployerCost), LeaveQuota: int32(x.LeaveQuota), LeaveUsed: int32(x.LeaveUsed)})
	}
	for _, t := range d.LeaveTypeMonths {
		out.LeaveTypeMonths = append(out.LeaveTypeMonths, &hrv1.HrDashTypeMonth{Month: t.Month, Type: t.Type, Days: t.Days})
	}
	for _, p := range d.Pending {
		out.Pending = append(out.Pending, &hrv1.HrDashPending{Kind: p.Kind, Id: p.ID, EmployeeId: p.EmployeeID, FullName: p.FullName,
			DepartmentName: p.DepartmentName, Type: p.Type, Date: p.Date, Amount: p.Amount, Status: p.Status, AgeDays: int32(p.AgeDays)})
	}
	for _, e := range d.Expiring {
		out.Expiring = append(out.Expiring, &hrv1.HrDashExpiry{Kind: e.Kind, EmployeeId: e.EmployeeID, FullName: e.FullName,
			DepartmentName: e.DepartmentName, ContractId: e.ContractID, ContractNumber: e.ContractNumber, ContractType: e.ContractType,
			EndDate: e.EndDate, DaysLeft: int32(e.DaysLeft)})
	}
	for _, l := range d.SalaryByLevel {
		out.SalaryByLevel = append(out.SalaryByLevel, &hrv1.HrDashLevel{Level: l.Level, Employees: int32(l.Employees), AvgBase: money(l.AvgBase),
			AvgGross: money(l.AvgGross), MinGross: money(l.MinGross), MaxGross: money(l.MaxGross)})
	}
	for _, l := range d.Loans {
		out.Loans = append(out.Loans, &hrv1.HrDashLoan{LoanId: l.ID, EmployeeId: l.EmployeeID, FullName: l.FullName, DepartmentName: l.DepartmentName,
			LoanType: l.LoanType, PrincipalAmount: money(l.Principal), RemainingBalance: money(l.Remaining), MonthlyInstallment: money(l.Monthly)})
	}
	return out
}

func kpisPB(k app.DashKpis) *hrv1.HrDashKpis {
	return &hrv1.HrDashKpis{Headcount: int32(k.Headcount), Joiners: int32(k.Joiners), Leavers: int32(k.Leavers), TurnoverRate: k.TurnoverRate,
		AttendanceRecords: int32(k.AttendanceRecords), AttendanceRate: k.AttendanceRate, LateCount: int32(k.LateCount), LateRate: k.LateRate,
		AvgLateMinutes: k.AvgLateMinutes, AbsentCount: int32(k.AbsentCount), AvgWorkHours: k.AvgWorkHours, LeaveDays: k.LeaveDays,
		OvertimeHours: k.OvertimeHours, OvertimeCost: money(k.OvertimeCost), PayrollGross: money(k.PayrollGross), PayrollNet: money(k.PayrollNet),
		PayrollEmployerCost: money(k.PayrollEmployerCost), PayrollEmployees: int32(k.PayrollEmployees)}
}

func bucketsPB(rows []app.DashBucket) []*hrv1.HrDashBucket {
	out := make([]*hrv1.HrDashBucket, len(rows))
	for i, b := range rows {
		out[i] = &hrv1.HrDashBucket{Date: b.Date, Headcount: int32(b.Headcount), Joiners: int32(b.Joiners), Leavers: int32(b.Leavers),
			AttendanceRecords: int32(b.Records), Present: int32(b.Present), Late: int32(b.Late), Absent: int32(b.Absent),
			AttendanceRate: b.AttendanceRate, LateRate: b.LateRate, AvgWorkHours: b.AvgWork, LeaveDays: b.LeaveDays,
			OvertimeHours: b.OvertimeHours, OvertimeCost: money(b.OvertimeCost), TurnoverRate: b.TurnoverRate,
			PayrollGross: money(b.PayrollGross), PayrollNet: money(b.PayrollNet), PayrollPph21: money(b.PayrollPph21),
			PayrollBpjsEmployee: money(b.BpjsEmployee), PayrollBpjsEmployer: money(b.BpjsEmployer), PayrollEmployees: int32(b.PayrollEmployees)}
	}
	return out
}

func cellsPB(rows []app.DashCell) []*hrv1.HrDashCell {
	out := make([]*hrv1.HrDashCell, len(rows))
	for i, c := range rows {
		out[i] = &hrv1.HrDashCell{Row: c.Row, Col: int32(c.Col), Value: int32(c.Value), Total: int32(c.Total)}
	}
	return out
}

func slicesPB(rows []app.DashSlice) []*hrv1.HrDashSlice {
	out := make([]*hrv1.HrDashSlice, len(rows))
	for i, s := range rows {
		out[i] = &hrv1.HrDashSlice{Key: s.Key, Label: s.Label, Value: s.Value, Count: int32(s.Count)}
	}
	return out
}

func empStatsPB(rows []app.DashEmployeeStat) []*hrv1.HrDashEmployee {
	out := make([]*hrv1.HrDashEmployee, len(rows))
	for i, e := range rows {
		out[i] = &hrv1.HrDashEmployee{EmployeeId: e.EmployeeID, FullName: e.FullName, DepartmentName: e.DepartmentName, PositionName: e.PositionName,
			AttendanceRecords: int32(e.Records), Late: int32(e.Late), LateMinutes: int32(e.LateMinutes), Absent: int32(e.Absent),
			AttendanceRate: e.AttendanceRate, OvertimeHours: e.OvertimeHours, OvertimeCost: money(e.OvertimeCost)}
	}
	return out
}
