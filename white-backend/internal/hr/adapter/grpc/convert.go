// Package grpcapi exposes the hr use cases over gRPC. Handlers resolve the
// caller, translate proto to Go and back; rules live in app.
package grpcapi

import (
	"encoding/json"
	"time"

	hrv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/hr/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/app"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/payroll"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ts(t time.Time) *timestamppb.Timestamp { return timestamppb.New(t) }

func tsPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func timePtr(t *timestamppb.Timestamp) *time.Time {
	if t == nil {
		return nil
	}
	v := t.AsTime()
	return &v
}

func money(v float64) string { return domain.Money(v) }

func moneyPtr(v *float64) string {
	if v == nil {
		return ""
	}
	return domain.Money(*v)
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// statusOf maps the master employment status to the directory enum.
func statusOf(e domain.Employee) hrv1.EmploymentStatus {
	switch e.EmploymentStatus {
	case "resigned":
		return hrv1.EmploymentStatus_RESIGNED
	case "terminated":
		return hrv1.EmploymentStatus_TERMINATED
	case "probation":
		return hrv1.EmploymentStatus_PROBATION
	case "on_leave":
		return hrv1.EmploymentStatus_ON_LEAVE
	}
	if !e.IsActive {
		return hrv1.EmploymentStatus_RESIGNED
	}
	return hrv1.EmploymentStatus_ACTIVE
}

func employeePB(e domain.Employee) *hrv1.Employee {
	return &hrv1.Employee{Id: e.ID, UserId: e.UserID, EmployeeNo: e.EmployeeNo, FullName: e.FullName, Email: e.Email, Phone: e.Phone,
		DepartmentId: e.DepartmentID, DepartmentName: e.DepartmentName, PositionId: e.PositionID, PositionName: e.PositionName,
		ManagerId: e.ManagerID, ManagerUserId: e.ManagerUserID, Status: statusOf(e), JoinDate: e.JoinDate, EndDate: e.EndDate, CreatedAt: ts(e.CreatedAt)}
}

func profilePB(e domain.Employee) *hrv1.EmployeeProfile {
	return &hrv1.EmployeeProfile{Employee: employeePB(e), EmploymentStatus: e.EmploymentStatus, EmploymentStatusName: e.EmploymentStatusName,
		ManagerName: e.ManagerName, Nik: e.NIK, Npwp: e.NPWP, BpjsTkNo: e.BpjsTkNo, BpjsKesNo: e.BpjsKesNo, BirthPlace: e.BirthPlace,
		BirthDate: e.BirthDate, Gender: e.Gender, MaritalStatus: e.MaritalStatus, Religion: e.Religion, Address: e.Address,
		ProbationEndDate: e.ProbationEndDate, BankName: e.BankName, BankAccountNo: e.BankAccountNo, BankAccountName: e.BankAccountName,
		EmergencyContactName: e.EmergencyName, EmergencyContactPhone: e.EmergencyPhone, EmergencyContactRelation: e.EmergencyRelation,
		IsActive: e.IsActive, UpdatedAt: ts(e.UpdatedAt)}
}

func refPB(id, name, no, dept string) *hrv1.EmployeeRef {
	return &hrv1.EmployeeRef{Id: id, FullName: name, EmployeeNo: no, DepartmentName: dept}
}

func employeeInput(in *hrv1.EmployeeInput) app.EmployeeInput {
	if in == nil {
		return app.EmployeeInput{}
	}
	return app.EmployeeInput{EmployeeNo: in.EmployeeNo, FullName: in.FullName, Email: in.Email, Phone: in.Phone,
		DepartmentID: in.DepartmentId, PositionID: in.PositionId, ManagerID: in.ManagerId, EmploymentStatus: in.EmploymentStatus,
		JoinDate: in.JoinDate, EndDate: in.EndDate, ProbationEndDate: in.ProbationEndDate, NIK: in.Nik, NPWP: in.Npwp,
		BpjsTkNo: in.BpjsTkNo, BpjsKesNo: in.BpjsKesNo, BirthPlace: in.BirthPlace, BirthDate: in.BirthDate, Gender: in.Gender,
		MaritalStatus: in.MaritalStatus, Religion: in.Religion, Address: in.Address, BankName: in.BankName,
		BankAccountNo: in.BankAccountNo, BankAccountName: in.BankAccountName, EmergencyName: in.EmergencyContactName,
		EmergencyPhone: in.EmergencyContactPhone, EmergencyRelation: in.EmergencyContactRelation, UserID: in.UserId}
}

func departmentPB(d domain.Department) *hrv1.Department {
	return &hrv1.Department{Id: d.ID, Code: d.Code, Name: d.Name, ParentId: d.ParentID, HeadEmployeeId: d.HeadEmployeeID, IsActive: d.IsActive}
}

func positionPB(p domain.Position) *hrv1.Position {
	return &hrv1.Position{Id: p.ID, Title: p.Title, DepartmentId: p.DepartmentID, DepartmentName: p.DepartmentName, Level: p.Level,
		SalaryMin: money(p.SalaryMin), SalaryMax: money(p.SalaryMax), IsActive: p.IsActive}
}

func documentPB(d domain.Document) *hrv1.EmployeeDocument {
	return &hrv1.EmployeeDocument{Id: d.ID, EmployeeId: d.EmployeeID, DocType: d.DocType, Name: d.Name, ContentType: d.ContentType,
		SizeBytes: d.SizeBytes, CreatedAt: ts(d.CreatedAt)}
}

func orgNodePB(n *app.OrgNode) *hrv1.OrgNode {
	out := &hrv1.OrgNode{EmployeeId: n.Employee.ID, FullName: n.Employee.FullName, PositionName: n.Employee.PositionName,
		DepartmentName: n.Employee.DepartmentName}
	for _, r := range n.Reports {
		out.Reports = append(out.Reports, orgNodePB(r))
	}
	return out
}

// ─── Workforce ───────────────────────────────────────────────

func locationPB(l *domain.Location) *hrv1.GeoLocation {
	if l == nil {
		return nil
	}
	return &hrv1.GeoLocation{Latitude: l.Latitude, Longitude: l.Longitude, Accuracy: l.Accuracy, Address: l.Address}
}

func locationFrom(l *hrv1.GeoLocation) *domain.Location {
	if l == nil {
		return nil
	}
	return &domain.Location{Latitude: l.Latitude, Longitude: l.Longitude, Accuracy: l.Accuracy, Address: l.Address}
}

func attendancePB(a domain.Attendance) *hrv1.AttendanceRecord {
	return &hrv1.AttendanceRecord{Id: a.ID, Employee: refPB(a.EmployeeID, a.EmployeeName, a.EmployeeNo, a.DepartmentName),
		Date: a.Date, ClockIn: tsPtr(a.ClockIn), ClockOut: tsPtr(a.ClockOut), ClockInLocation: locationPB(a.ClockInLocation),
		ClockOutLocation: locationPB(a.ClockOutLocation), HasClockInPhoto: a.ClockInPhotoKey != nil, HasClockOutPhoto: a.ClockOutPhotoKey != nil,
		WorkHours: a.WorkHours, Status: a.Status, IsLate: a.IsLate, LateMinutes: int32(a.LateMinutes), OvertimeHours: a.OvertimeHours,
		ShiftId: str(a.ShiftID), ShiftName: a.ShiftName, ScheduledStart: tsPtr(a.ScheduledStart), ScheduledEnd: tsPtr(a.ScheduledEnd),
		IsValidated: a.ValidatedAt != nil, ValidatedByName: a.ValidatedByName, ValidatedAt: tsPtr(a.ValidatedAt),
		ValidationNotes: a.ValidationNotes, Notes: a.Notes}
}

func attendanceList(rows []domain.Attendance, total int) *hrv1.ListAttendanceResponse {
	out := &hrv1.ListAttendanceResponse{Total: int32(total)}
	for _, r := range rows {
		out.Records = append(out.Records, attendancePB(r))
	}
	return out
}

func shiftPB(s domain.Shift) *hrv1.Shift {
	return &hrv1.Shift{Id: s.ID, Name: s.Name, StartTime: domain.HHMM(s.StartTime), EndTime: domain.HHMM(s.EndTime), BreakMinutes: int32(s.BreakMinutes),
		LateToleranceMinutes: int32(s.LateToleranceMinutes), IsOvernight: s.IsOvernight, IsActive: s.IsActive, SortOrder: int32(s.SortOrder)}
}

func scheduleDayPB(r domain.ScheduleRow) *hrv1.ScheduleDay {
	d := &hrv1.ScheduleDay{DayOfWeek: int32(r.DayOfWeek), ShiftId: str(r.ShiftID), EffectiveFrom: r.EffectiveFrom, EffectiveTo: r.EffectiveTo}
	if r.Shift != nil {
		d.ShiftName = r.Shift.Name
	}
	return d
}

func teamDaysPB(days []app.TeamDay) []*hrv1.TeamScheduleDay {
	out := make([]*hrv1.TeamScheduleDay, len(days))
	for i, d := range days {
		out[i] = &hrv1.TeamScheduleDay{Date: d.Date, Status: d.Status, ShiftName: d.ShiftName, StartTime: d.StartTime, EndTime: d.EndTime}
	}
	return out
}

func teamSchedulePB(monday string, rows []app.TeamRow) *hrv1.TeamSchedule {
	out := &hrv1.TeamSchedule{WeekStart: monday}
	for _, r := range rows {
		e := r.Employee
		out.Rows = append(out.Rows, &hrv1.TeamScheduleRow{Employee: refPB(e.ID, e.FullName, e.EmployeeNo, e.DepartmentName), Days: teamDaysPB(r.Days)})
	}
	return out
}

func holidayPB(h domain.Holiday) *hrv1.Holiday {
	return &hrv1.Holiday{Id: h.ID, HolidayDate: h.Date, Name: h.Name, Type: h.Type, DeductsLeave: h.DeductsLeave, Status: h.Status, Note: h.Note}
}

func balancePB(b *app.Balance) *hrv1.LeaveBalance {
	if b == nil {
		return nil
	}
	return &hrv1.LeaveBalance{EmployeeId: b.EmployeeID, Year: int32(b.Year), AnnualLeaveTotal: int32(b.AnnualTotal),
		AnnualLeaveUsed: int32(b.AnnualUsed), AnnualLeaveRemaining: int32(b.AnnualRemaining()), SickLeaveUsed: int32(b.SickUsed),
		UnpaidLeaveUsed: int32(b.UnpaidUsed), OtherLeaveUsed: int32(b.OtherUsed), PendingDays: int32(b.PendingDays)}
}

func overtimePB(o domain.Overtime) *hrv1.Overtime {
	return &hrv1.Overtime{Id: o.ID, Employee: refPB(o.EmployeeID, o.EmployeeName, o.EmployeeNo, o.DepartmentName), Date: o.Date,
		StartTime: o.StartTime, EndTime: o.EndTime, Hours: o.Hours, Source: o.Source, Status: o.Status, Reason: o.Reason,
		RequestedByName: o.RequestedByName, DecidedByName: o.DecidedByName, DecidedAt: tsPtr(o.DecidedAt),
		RejectionReason: o.RejectionReason, CreatedAt: ts(o.CreatedAt)}
}

func overtimeList(rows []domain.Overtime) *hrv1.ListOvertimeResponse {
	out := &hrv1.ListOvertimeResponse{}
	for _, r := range rows {
		out.Overtime = append(out.Overtime, overtimePB(r))
	}
	return out
}

func contractPB(c domain.Contract, today string) *hrv1.Contract {
	out := &hrv1.Contract{Id: c.ID, ContractNumber: c.ContractNumber, Employee: refPB(c.EmployeeID, c.EmployeeName, c.EmployeeNo, c.DepartmentNameNow),
		ContractType: c.ContractType, Status: c.Status, StartDate: c.StartDate, EndDate: c.EndDate, ProbationEndDate: c.ProbationEndDate,
		ParentContractId: c.ParentContractID, Sequence: int32(c.Sequence), PositionTitle: c.PositionTitle, DepartmentName: c.DepartmentName,
		WorkLocation: c.WorkLocation, BaseSalary: moneyPtr(c.BaseSalary), SignedAt: c.SignedAt, CompensationAmount: moneyPtr(c.CompensationAmount),
		TerminatedReason: c.TerminatedReason, Notes: c.Notes, CreatedAt: ts(c.CreatedAt)}
	if c.EndDate != nil {
		if n, ok := hris.DaysUntil(today, *c.EndDate); ok {
			v := int32(n)
			out.DaysLeft = &v
		}
	}
	return out
}

func announcementPB(a domain.Announcement) *hrv1.Announcement {
	return &hrv1.Announcement{Id: a.ID, Title: a.Title, Body: a.Body, Tags: a.Tags, Status: a.Status, IsPinned: a.IsPinned,
		TargetScope: a.TargetScope, DepartmentIds: a.DepartmentIDs, PublishAt: tsPtr(a.PublishAt), ExpiresAt: tsPtr(a.ExpiresAt),
		CreatedByName: a.CreatedByName, CreatedAt: ts(a.CreatedAt), IsRead: a.IsRead, ReadCount: int32(a.ReadCount)}
}

func announcementInput(id string, in *hrv1.AnnouncementInput) domain.Announcement {
	if in == nil {
		return domain.Announcement{ID: id}
	}
	return domain.Announcement{ID: id, Title: in.Title, Body: in.Body, Tags: in.Tags, Status: in.Status, IsPinned: in.IsPinned,
		TargetScope: in.TargetScope, DepartmentIDs: in.DepartmentIds, PublishAt: timePtr(in.PublishAt), ExpiresAt: timePtr(in.ExpiresAt)}
}

func taskPB(t domain.OnboardingTask) *hrv1.OnboardingTask {
	return &hrv1.OnboardingTask{Id: t.ID, EmployeeId: t.EmployeeID, TaskName: t.TaskName, Category: t.Category, Priority: int32(t.Priority),
		DueDate: t.DueDate, IsCompleted: t.IsCompleted, CompletedAt: tsPtr(t.CompletedAt), CompletedByName: t.CompletedByName, Notes: t.Notes}
}

func checklistPB(employeeID string, tasks []domain.OnboardingTask) *hrv1.OnboardingChecklist {
	out := &hrv1.OnboardingChecklist{EmployeeId: employeeID, Total: int32(len(tasks))}
	for _, t := range tasks {
		if t.IsCompleted {
			out.Completed++
		}
		out.Tasks = append(out.Tasks, taskPB(t))
	}
	return out
}

func offboardingPB(o domain.Offboarding) *hrv1.Offboarding {
	return &hrv1.Offboarding{Id: o.ID, EmployeeId: o.EmployeeID, ResignationType: o.ResignationType, Status: o.Status,
		ResignationDate: o.ResignationDate, LastWorkingDay: o.LastWorkingDay, Reason: o.Reason, ExitInterviewNotes: o.ExitInterviewNotes,
		ClearanceHr: o.ClearanceHR, ClearanceIt: o.ClearanceIT, ClearanceFinance: o.ClearanceFinance, ClearanceManager: o.ClearanceManager,
		Notes: o.Notes, CreatedAt: ts(o.CreatedAt), CompletedAt: tsPtr(o.CompletedAt)}
}

// ─── Compensation ────────────────────────────────────────────

func settingsPB(s domain.PayrollSettings) *hrv1.PayrollSettings {
	out := &hrv1.PayrollSettings{BpjsTkJhtEmployee: money(s.BpjsTkJhtEmployee), BpjsTkJhtEmployer: money(s.BpjsTkJhtEmployer),
		BpjsTkJpEmployee: money(s.BpjsTkJpEmployee), BpjsTkJpEmployer: money(s.BpjsTkJpEmployer), BpjsTkJkk: money(s.BpjsTkJkk),
		BpjsTkJkm: money(s.BpjsTkJkm), BpjsKesEmployee: money(s.BpjsKesEmployee), BpjsKesEmployer: money(s.BpjsKesEmployer),
		BpjsKesMaxUpah: money(s.BpjsKesMaxUpah), TaperaEmployee: money(s.TaperaEmployee), TaperaEmployer: money(s.TaperaEmployer),
		Ptkp: map[string]string{}, OvertimeMultiplier: money(s.OvertimeMultiplier), OvertimeMultiplierHoliday: money(s.OvertimeMultiplierHoliday),
		OvertimeHourlyDivisor: money(s.OvertimeHourlyDivisor), LateDeductionMode: s.LateDeductionMode,
		LateDeductionAmount: money(s.LateDeductionAmount), LoanMaxInstallmentPercent: money(s.LoanMaxInstallmentPercent),
		LoanMaxActivePerEmployee: int32(s.LoanMaxActivePerEmployee), PayrollDay: int32(s.PayrollDay), Pph21Method: "progressive_annual",
		UpdatedAt: ts(s.UpdatedAt)}
	for k, v := range s.PTKP {
		out.Ptkp[k] = money(v)
	}
	for _, b := range s.BracketLimits {
		out.Pph21BracketLimits = append(out.Pph21BracketLimits, money(b))
	}
	return out
}

func salaryPB(s domain.Salary) *hrv1.Salary {
	return &hrv1.Salary{Id: s.ID, Employee: refPB(s.EmployeeID, s.EmployeeName, s.EmployeeNo, s.DepartmentName),
		BaseSalary: money(s.BaseSalary), FixedAllowance: money(s.FixedAllowance), VariableAllowance: money(s.VariableAllowance),
		TransportAllowance: money(s.TransportAllowance), MealAllowance: money(s.MealAllowance), HousingAllowance: money(s.HousingAllowance),
		PtkpStatus: s.PTKPStatus, IsTaxable: s.IsTaxable, BpjsTkEnrolled: s.BpjsTkEnrolled, BpjsKesEnrolled: s.BpjsKesEnrolled,
		TaperaEnrolled: s.TaperaEnrolled, EffectiveDate: s.EffectiveDate, EndDate: s.EndDate, IsActive: s.IsActive, Notes: s.Notes}
}

func salaryInput(in *hrv1.SalaryInput) app.SalaryInput {
	if in == nil {
		return app.SalaryInput{}
	}
	return app.SalaryInput{EmployeeID: in.EmployeeId, BaseSalary: in.BaseSalary, FixedAllowance: in.FixedAllowance,
		VariableAllowance: in.VariableAllowance, TransportAllowance: in.TransportAllowance, MealAllowance: in.MealAllowance,
		HousingAllowance: in.HousingAllowance, PTKPStatus: in.PtkpStatus, EffectiveDate: in.EffectiveDate, Notes: in.Notes,
		IsTaxable: in.IsTaxable, BpjsTkEnrolled: in.BpjsTkEnrolled, BpjsKesEnrolled: in.BpjsKesEnrolled, TaperaEnrolled: in.TaperaEnrolled}
}

func runPB(r domain.PayrollRun) *hrv1.PayrollRun {
	return &hrv1.PayrollRun{Id: r.ID, RunName: r.RunName, PeriodMonth: int32(r.PeriodMonth), PeriodYear: int32(r.PeriodYear), Status: r.Status,
		TotalEmployees: int32(r.TotalEmployees), TotalGross: money(r.TotalGross), TotalDeductions: money(r.TotalDeductions),
		TotalNet: money(r.TotalNet), TotalBpjsEmployee: money(r.TotalBpjsEmployee), TotalBpjsEmployer: money(r.TotalBpjsEmployer),
		TotalPph21: money(r.TotalPph21), NeedsRecalculation: r.NeedsRecalculation, CalculatedAt: tsPtr(r.CalculatedAt),
		ApprovedAt: tsPtr(r.ApprovedAt), ApprovedByName: r.ApprovedByName, PaidAt: tsPtr(r.PaidAt), Notes: r.Notes, CreatedAt: ts(r.CreatedAt)}
}

func payslipPB(s domain.Payslip) *hrv1.Payslip {
	out := &hrv1.Payslip{Id: s.ID, RunId: s.RunID, PeriodMonth: int32(s.PeriodMonth), PeriodYear: int32(s.PeriodYear), RunStatus: s.RunStatus,
		Employee: refPB(s.EmployeeID, s.EmployeeName, s.EmployeeNo, s.DepartmentName), BaseSalary: money(s.BaseSalary),
		FixedAllowance: money(s.FixedAllowance), VariableAllowance: money(s.VariableAllowance), TransportAllowance: money(s.TransportAllowance),
		MealAllowance: money(s.MealAllowance), HousingAllowance: money(s.HousingAllowance), OvertimePay: money(s.OvertimePay),
		Thr: money(s.Thr), Bonus: money(s.Bonus), GrossSalary: money(s.GrossSalary), BpjsTkJhtDeduction: money(s.BpjsTkJhtDeduction),
		BpjsTkJpDeduction: money(s.BpjsTkJpDeduction), BpjsKesDeduction: money(s.BpjsKesDeduction), TaperaDeduction: money(s.TaperaDeduction),
		Pph21Deduction: money(s.Pph21Deduction), UnpaidLeaveDeduction: money(s.UnpaidLeaveDeduction), LateDeduction: money(s.LateDeduction),
		LoanDeduction: money(s.LoanDeduction), TotalDeductions: money(s.TotalDeductions), NetSalary: money(s.NetSalary),
		BpjsTkJhtEmployer: money(s.BpjsTkJhtEmployer), BpjsTkJpEmployer: money(s.BpjsTkJpEmployer), BpjsTkJkkEmployer: money(s.BpjsTkJkkEmployer),
		BpjsTkJkmEmployer: money(s.BpjsTkJkmEmployer), BpjsKesEmployer: money(s.BpjsKesEmployer), TaperaEmployer: money(s.TaperaEmployer),
		TaxableIncome: money(s.TaxableIncome), PtkpAmount: money(s.PTKPAmount), Pph21Annual: money(s.Pph21Annual), WorkingDays: s.WorkingDays,
		PresentDays: s.PresentDays, LateDays: s.LateDays, UnpaidLeaveDays: s.UnpaidLeaveDays, OvertimeHours: s.OvertimeHours,
		ProrateFactor: s.ProrateFactor}
	var details []payroll.LoanInstallmentDetail
	_ = json.Unmarshal(s.LoanDetails, &details)
	for _, d := range details {
		li := &hrv1.LoanInstallment{LoanId: d.LoanID, LoanType: str(d.LoanType), InstallmentNo: int32(d.InstallmentNo), Amount: money(d.Amount),
			RemainingBefore: money(d.RemainingBefore), RemainingAfter: money(d.RemainingAfter), Label: payroll.LoanInstallmentLabel(d)}
		if d.TenorMonths != nil {
			li.TenorMonths = int32(*d.TenorMonths)
		}
		out.LoanDetails = append(out.LoanDetails, li)
	}
	return out
}

func payslipList(rows []domain.Payslip) *hrv1.ListPayslipsResponse {
	out := &hrv1.ListPayslipsResponse{}
	for _, r := range rows {
		out.Payslips = append(out.Payslips, payslipPB(r))
	}
	return out
}

func additionPB(a domain.PayrollAddition) *hrv1.PayrollAddition {
	return &hrv1.PayrollAddition{Id: a.ID, Employee: refPB(a.EmployeeID, a.EmployeeName, a.EmployeeNo, a.DepartmentName),
		PeriodMonth: int32(a.PeriodMonth), PeriodYear: int32(a.PeriodYear), Amount: money(a.Amount), Label: a.Label, CreatedAt: ts(a.CreatedAt)}
}

func loanPB(l domain.Loan, withSchedule bool) *hrv1.Loan {
	out := &hrv1.Loan{Id: l.ID, Employee: refPB(l.EmployeeID, l.EmployeeName, l.EmployeeNo, l.DepartmentName), LoanType: l.LoanType,
		PrincipalAmount: money(l.PrincipalAmount), InterestRate: money(l.InterestRate), TenorMonths: int32(l.TenorMonths),
		MonthlyInstallment: money(l.MonthlyInstallment), RemainingBalance: money(l.RemainingBalance), PaidAmount: money(l.PaidAmount),
		Status: l.Status, IsActive: l.IsActive, Purpose: l.Purpose, ApprovedByName: l.ApprovedByName, ApprovedAt: tsPtr(l.ApprovedAt),
		RejectionReason: l.RejectionReason, CreatedAt: ts(l.CreatedAt)}
	if l.FirstInstallmentMonth != nil && l.FirstInstallmentYear != nil {
		out.FirstInstallmentMonth, out.FirstInstallmentYear = int32(*l.FirstInstallmentMonth), int32(*l.FirstInstallmentYear)
	}
	if withSchedule {
		for _, r := range app.Schedule(l) {
			out.Schedule = append(out.Schedule, &hrv1.LoanScheduleRow{InstallmentNo: int32(r.No), PeriodMonth: int32(r.Month),
				PeriodYear: int32(r.Year), Amount: money(r.Amount), RemainingAfter: money(r.RemainingAfter), IsPaid: r.IsPaid})
		}
	}
	return out
}

func loanList(rows []domain.Loan) *hrv1.ListLoansResponse {
	out := &hrv1.ListLoansResponse{}
	for _, r := range rows {
		out.Loans = append(out.Loans, loanPB(r, false))
	}
	return out
}

func reportPB(r *hris.HrisReport) *hrv1.HrisReport {
	h, a, l := r.Headcount, r.Attendance, r.Leaves
	out := &hrv1.HrisReport{Month: int32(r.Period.Month), Year: int32(r.Period.Year), TotalActive: int32(h.TotalActive),
		NewHires: int32(h.NewHires), TurnoverCount: int32(h.TurnoverCount), TurnoverRate: h.TurnoverRate,
		AttendanceRecords: int32(a.TotalRecords), PresentCount: int32(a.PresentCount), AbsentCount: int32(a.AbsentCount),
		LateCount: int32(a.LateCount), PresentRate: a.PresentRate, LateRate: a.LateRate, AvgWorkHours: a.AvgWorkHours,
		LeaveApprovedCount: int32(l.ApprovedCount), LeavePendingCount: int32(l.PendingCount), LeaveTotalDays: l.TotalDays}
	for _, s := range h.ByStatus {
		out.ByStatus = append(out.ByStatus, &hrv1.LabelCount{Label: s.Status, Count: int32(s.Count)})
	}
	for _, d := range h.ByDepartment {
		out.ByDepartment = append(out.ByDepartment, &hrv1.LabelCount{Label: d.Name, Count: int32(d.Count)})
	}
	for _, m := range h.MonthlyTrend {
		out.MonthlyTrend = append(out.MonthlyTrend, &hrv1.LabelCount{Label: m.Month, Count: int32(m.Count)})
	}
	for _, d := range a.DailyTrend {
		out.DailyTrend = append(out.DailyTrend, &hrv1.DailyAttendancePoint{Date: d.Date, Present: int32(d.Present), Absent: int32(d.Absent), Late: int32(d.Late)})
	}
	for _, t := range l.ByType {
		out.LeaveByType = append(out.LeaveByType, &hrv1.LeaveDaysByType{Type: t.Type, Days: t.Days})
	}
	return out
}
