package hris

import (
	"fmt"
	"sort"
	"time"
)

/* ── Monthly HRIS report (lib/hris/reports-summary) ──────────────────── */

// MonthRange is [first of month, first of next month).
func MonthRange(month, year int) (string, string) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	return start.Format(DateLayout), start.AddDate(0, 1, 0).Format(DateLayout)
}

// ReportEmployee, ReportAttendance and ReportLeave are the report inputs;
// dates are "YYYY-MM-DD".
type ReportEmployee struct {
	EmploymentStatus string
	IsActive         bool
	DepartmentID     *string
	JoinDate         string
	EndDate          *string
}

type ReportDepartment struct{ ID, Name string }

type ReportAttendance struct {
	Status    string
	IsLate    bool
	WorkHours float64
	Date      string
}

type ReportLeave struct {
	LeaveType string
	Status    string
	TotalDays float64
}

var reportLeaveLabels = map[string]string{
	"annual": "Tahunan", "sick": "Sakit", "maternity": "Melahirkan", "paternity": "Ayah",
	"unpaid": "Tidak Dibayar", "emergency": "Darurat", "pilgrimage": "Haji/Umrah",
	"menstrual": "Haid", "marriage": "Pernikahan", "bereavement": "Duka Cita",
}

type StatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type NameCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type MonthCount struct {
	Month string `json:"month"`
	Count int    `json:"count"`
}

type DailyAttendance struct {
	Date    string `json:"date"`
	Present int    `json:"present"`
	Absent  int    `json:"absent"`
	Late    int    `json:"late"`
}

type LeaveDays struct {
	Type string  `json:"type"`
	Days float64 `json:"days"`
}

// HrisReport is the GET /api/hris/reports body.
type HrisReport struct {
	Period struct {
		Month int `json:"month"`
		Year  int `json:"year"`
	} `json:"period"`
	Headcount struct {
		TotalActive   int           `json:"total_active"`
		NewHires      int           `json:"new_hires"`
		TurnoverCount int           `json:"turnover_count"`
		TurnoverRate  float64       `json:"turnover_rate"`
		ByStatus      []StatusCount `json:"by_status"`
		ByDepartment  []NameCount   `json:"by_department"`
		MonthlyTrend  []MonthCount  `json:"monthly_trend"`
	} `json:"headcount"`
	Attendance struct {
		TotalRecords int               `json:"total_records"`
		PresentCount int               `json:"present_count"`
		AbsentCount  int               `json:"absent_count"`
		LateCount    int               `json:"late_count"`
		PresentRate  float64           `json:"present_rate"`
		LateRate     float64           `json:"late_rate"`
		AvgWorkHours float64           `json:"avg_work_hours"`
		DailyTrend   []DailyAttendance `json:"daily_trend"`
	} `json:"attendance"`
	Leaves struct {
		ApprovedCount int         `json:"approved_count"`
		PendingCount  int         `json:"pending_count"`
		TotalDays     float64     `json:"total_days"`
		ByType        []LeaveDays `json:"by_type"`
	} `json:"leaves"`
}

func pct1(part, whole int) float64 {
	if whole <= 0 {
		return 0
	}
	return ToFixed(float64(part)/float64(whole)*100, 1)
}

// BuildHrisReport is buildHrisReport.
func BuildHrisReport(employees []ReportEmployee, departments []ReportDepartment,
	attendance []ReportAttendance, leaves []ReportLeave, month, year int) HrisReport {
	var r HrisReport
	r.Period.Month, r.Period.Year = month, year
	start, end := MonthRange(month, year)

	var active []ReportEmployee
	for _, e := range employees {
		if e.IsActive {
			active = append(active, e)
		}
	}
	h := &r.Headcount
	h.TotalActive = len(active)
	yearStart, yearEnd := fmt.Sprintf("%d-01-01", year), fmt.Sprintf("%d-01-01", year+1)
	for _, e := range employees {
		if e.JoinDate >= start && e.JoinDate < end {
			h.NewHires++
		}
		if (e.EmploymentStatus == "resigned" || e.EmploymentStatus == "terminated") &&
			e.EndDate != nil && *e.EndDate != "" && *e.EndDate >= yearStart && *e.EndDate < yearEnd {
			h.TurnoverCount++
		}
	}
	if h.TotalActive > 0 {
		h.TurnoverRate = pct1(h.TurnoverCount, h.TotalActive+h.TurnoverCount)
	}
	h.ByStatus = []StatusCount{}
	statusIndex := map[string]int{}
	for _, e := range active {
		i, ok := statusIndex[e.EmploymentStatus]
		if !ok {
			i = len(h.ByStatus)
			statusIndex[e.EmploymentStatus] = i
			h.ByStatus = append(h.ByStatus, StatusCount{Status: e.EmploymentStatus})
		}
		h.ByStatus[i].Count++
	}
	h.ByDepartment = []NameCount{}
	for _, d := range departments {
		n := 0
		for _, e := range active {
			if e.DepartmentID != nil && *e.DepartmentID == d.ID {
				n++
			}
		}
		if n > 0 {
			h.ByDepartment = append(h.ByDepartment, NameCount{Name: d.Name, Count: n})
		}
	}
	h.MonthlyTrend = make([]MonthCount, 6)
	for i := range 6 {
		d := time.Date(year, time.Month(month-(5-i)), 1, 0, 0, 0, 0, time.UTC)
		mStart, mEnd := d.Format(DateLayout), d.AddDate(0, 1, 0).Format(DateLayout)
		n := 0
		for _, e := range employees {
			if e.JoinDate < mEnd && (e.EndDate == nil || *e.EndDate == "" || *e.EndDate >= mStart) {
				n++
			}
		}
		h.MonthlyTrend[i] = MonthCount{Month: fmt.Sprintf("%s %02d", idMonths[d.Month()-1], d.Year()%100), Count: n}
	}

	a := &r.Attendance
	a.TotalRecords = len(attendance)
	daily := map[string]*DailyAttendance{}
	var hours float64
	for _, at := range attendance {
		day, ok := daily[at.Date]
		if !ok {
			day = &DailyAttendance{}
			daily[at.Date] = day
		}
		switch at.Status {
		case "present":
			a.PresentCount++
			day.Present++
		case "absent":
			a.AbsentCount++
			day.Absent++
		}
		if at.IsLate {
			a.LateCount++
			day.Late++
		}
		hours += at.WorkHours
	}
	if a.TotalRecords > 0 {
		a.AvgWorkHours = ToFixed(hours/float64(a.TotalRecords), 1)
	}
	a.PresentRate = pct1(a.PresentCount, a.TotalRecords)
	a.LateRate = pct1(a.LateCount, a.PresentCount)
	dates := make([]string, 0, len(daily))
	for d := range daily {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	a.DailyTrend = make([]DailyAttendance, len(dates))
	for i, d := range dates {
		row := *daily[d]
		t, _ := time.Parse(DateLayout, d)
		row.Date = fmt.Sprintf("%02d %s", t.Day(), idMonths[t.Month()-1])
		a.DailyTrend[i] = row
	}

	l := &r.Leaves
	var typeOrder []string
	byType := map[string]float64{}
	for _, lv := range leaves {
		switch lv.Status {
		case "approved":
			l.ApprovedCount++
			l.TotalDays += lv.TotalDays
			if _, ok := byType[lv.LeaveType]; !ok {
				typeOrder = append(typeOrder, lv.LeaveType)
			}
			byType[lv.LeaveType] += lv.TotalDays
		case "pending":
			l.PendingCount++
		}
	}
	l.ByType = make([]LeaveDays, len(typeOrder))
	for i, t := range typeOrder {
		label := t
		if v, ok := reportLeaveLabels[t]; ok {
			label = v
		}
		l.ByType[i] = LeaveDays{Type: label, Days: byType[t]}
	}
	sort.SliceStable(l.ByType, func(i, j int) bool { return l.ByType[i].Days > l.ByType[j].Days })
	return r
}

/* ── Employee self service (lib/hris/me-profile) ─────────────────────── */

// ESSScheduleRow is a pattern row joined with its shift.
type ESSScheduleRow struct {
	ScheduleRow
	ShiftName            *string
	StartTime            *string
	EndTime              *string
	LateToleranceMinutes *int64
}

func rowsOf(rows []ESSScheduleRow) []ScheduleRow {
	out := make([]ScheduleRow, len(rows))
	for i, r := range rows {
		out[i] = r.ScheduleRow
	}
	return out
}

// shiftDetail is the first row naming the winner's shift (else the winner).
func shiftDetail(rows []ESSScheduleRow, winner int) *ESSScheduleRow {
	if winner < 0 || rows[winner].ShiftID == nil {
		return nil
	}
	for i := range rows {
		if rows[i].ShiftID != nil && *rows[i].ShiftID == *rows[winner].ShiftID && rows[i].ShiftName != nil && *rows[i].ShiftName != "" {
			return &rows[i]
		}
	}
	return &rows[winner]
}

// TodayShift is the shift that applies today.
type TodayShift struct {
	Name                 string  `json:"name"`
	StartTime            *string `json:"start_time"`
	EndTime              *string `json:"end_time"`
	LateToleranceMinutes int64   `json:"late_tolerance_minutes"`
}

// TodayShiftOf: the shift on today, nil on a day off or without a pattern.
func TodayShiftOf(rows []ESSScheduleRow, today string) *TodayShift {
	winner := ResolveScheduleRow(rowsOf(rows), today)
	d := shiftDetail(rows, winner)
	if d == nil || d.ShiftName == nil || *d.ShiftName == "" {
		return nil
	}
	t := &TodayShift{Name: *d.ShiftName, StartTime: d.StartTime, EndTime: d.EndTime}
	if d.LateToleranceMinutes != nil {
		t.LateToleranceMinutes = *d.LateToleranceMinutes
	}
	return t
}
