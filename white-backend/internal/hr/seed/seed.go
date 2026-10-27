// Package seed inserts demo data for Mixist, an Indonesian digital agency:
// departments, positions, shifts, 34 employees (nine joined and five left
// within the last year) with managers, weekly schedules, public holidays,
// twelve months of attendance, leave, overtime and paid monthly payroll
// runs, contracts, salary structures, a loan, announcements and an
// onboarding checklist. Every step skips what already exists, so running it
// again changes nothing. The seed runs with a no-op publisher, so the paid
// runs never reach finance.
package seed

import (
	"context"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/app"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/payroll"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

type department struct{ Code, Name string }

var departments = []department{
	{"MGT", "Management"}, {"CRV", "Creative"}, {"SOC", "Social Media"}, {"SEO", "SEO"},
	{"ADS", "Ads"}, {"SLS", "Sales"}, {"FIN", "Finance"}, {"HR", "HR"},
}

type person struct {
	Name, Email, Dept, Title, Level string
	Manager                         int // 1-based index into people, 0 = none
	Status                          string
	JoinMonthsAgo                   int
	Gender                          string
	Base                            float64
	Shift                           string
	PTKP                            string
	Birth                           string // MM-DD
}

// people are Mixist's 25 employees; index+1 is the reference used by Manager.
var people = []person{
	{"Andika Wijaya", "admin@mixist.id", "MGT", "Chief Executive Officer", "director", 0, "permanent", 60, "male", 35_000_000, "reg", "K/2", "03-14"},
	{"Sari Kusuma", "sari.kusuma@mixist.id", "MGT", "Chief Operating Officer", "director", 1, "permanent", 54, "female", 30_000_000, "reg", "K/1", "07-22"},
	{"Bima Prasetyo", "bima.prasetyo@mixist.id", "CRV", "Creative Director", "manager", 1, "permanent", 48, "male", 22_000_000, "reg", "K/1", "11-02"},
	{"Citra Ayu Lestari", "citra.ayu@mixist.id", "CRV", "Senior Graphic Designer", "senior", 3, "permanent", 36, "female", 12_500_000, "late", "TK/0", "01-30"},
	{"Dimas Saputra", "dimas.saputra@mixist.id", "CRV", "Graphic Designer", "staff", 3, "contract", 14, "male", 8_000_000, "late", "TK/0", "09-09"},
	{"Eka Putri Handayani", "eka.putri@mixist.id", "CRV", "Video Editor", "staff", 3, "permanent", 26, "female", 9_000_000, "late", "TK/1", "05-17"},
	{"Fajar Nugroho", "fajar.nugroho@mixist.id", "SOC", "Social Media Lead", "lead", 2, "permanent", 40, "male", 16_000_000, "reg", "K/0", "12-05"},
	{"Gita Permata", "tim@mixist.id", "SOC", "Content Strategist", "senior", 7, "permanent", 30, "female", 11_000_000, "reg", "TK/0", "10-26"},
	{"Hana Lestari", "hana.lestari@mixist.id", "SOC", "Community Specialist", "staff", 7, "permanent", 20, "female", 7_500_000, "late", "TK/0", "04-11"},
	{"Indra Kurniawan", "indra.kurniawan@mixist.id", "SOC", "Copywriter", "staff", 7, "contract", 11, "male", 7_000_000, "reg", "TK/0", "08-28"},
	{"Joko Santoso", "joko.santoso@mixist.id", "SEO", "SEO Lead", "lead", 2, "permanent", 44, "male", 15_000_000, "reg", "K/2", "02-19"},
	{"Kartika Sari Dewi", "kartika.sari@mixist.id", "SEO", "SEO Specialist", "staff", 11, "permanent", 22, "female", 8_500_000, "reg", "TK/0", "06-06"},
	{"Lukman Hakim", "lukman.hakim@mixist.id", "SEO", "Content Writer", "staff", 11, "contract", 9, "male", 6_500_000, "reg", "TK/0", "10-29"},
	{"Maya Anggraini", "maya.anggraini@mixist.id", "ADS", "Performance Marketing Lead", "lead", 2, "permanent", 38, "female", 17_000_000, "reg", "K/1", "03-03"},
	{"Nanda Firmansyah", "nanda.firmansyah@mixist.id", "ADS", "Ads Specialist", "staff", 14, "permanent", 18, "male", 9_500_000, "reg", "TK/0", "07-07"},
	{"Oki Ramadhan", "oki.ramadhan@mixist.id", "ADS", "Data Analyst", "staff", 14, "permanent", 16, "male", 10_000_000, "reg", "TK/0", "01-12"},
	{"Putri Rahayu", "putri.rahayu@mixist.id", "SLS", "Sales Manager", "manager", 1, "permanent", 42, "female", 18_000_000, "reg", "K/1", "09-21"},
	{"Rizky Maulana", "rizky.maulana@mixist.id", "SLS", "Account Executive", "staff", 17, "permanent", 24, "male", 9_000_000, "reg", "TK/0", "05-25"},
	{"Salsabila Nur Aini", "salsabila.nur@mixist.id", "SLS", "Account Manager", "senior", 17, "permanent", 28, "female", 11_500_000, "reg", "TK/0", "11-15"},
	{"Taufik Hidayat", "taufik.hidayat@mixist.id", "SLS", "Business Development", "staff", 17, "contract", 13, "male", 8_500_000, "reg", "K/0", "12-31"},
	{"Utami Dewi", "utami.dewi@mixist.id", "FIN", "Finance Manager", "manager", 2, "permanent", 46, "female", 19_000_000, "reg", "K/2", "04-04"},
	{"Vina Oktaviani", "vina.oktaviani@mixist.id", "FIN", "Accounting Staff", "staff", 21, "permanent", 19, "female", 8_000_000, "reg", "TK/0", "10-08"},
	{"Wulan Sari", "wulan.sari@mixist.id", "HR", "HR Manager", "manager", 2, "permanent", 45, "female", 18_500_000, "reg", "K/1", "08-17"},
	{"Yoga Pratama", "yoga.pratama@mixist.id", "HR", "HR Generalist", "staff", 23, "permanent", 21, "male", 8_500_000, "reg", "TK/0", "02-02"},
	{"Zahra Amelia", "zahra.amelia@mixist.id", "SOC", "Social Media Officer", "staff", 7, "probation", 2, "female", 6_000_000, "reg", "TK/0", "06-30"},
	// Joined during the last year.
	{"Rangga Saputra", "rangga.saputra@mixist.id", "CRV", "Motion Designer", "staff", 3, "contract", 6, "male", 8_500_000, "late", "TK/0", "02-14"},
	{"Nadia Paramitha", "nadia.paramitha@mixist.id", "ADS", "Ads Specialist", "staff", 14, "probation", 1, "female", 8_000_000, "reg", "TK/0", "09-03"},
	{"Bayu Aditya", "bayu.aditya@mixist.id", "SEO", "SEO Specialist", "staff", 11, "contract", 4, "male", 7_500_000, "reg", "TK/0", "12-12"},
	{"Laras Wening", "laras.wening@mixist.id", "SLS", "Account Executive", "staff", 17, "permanent", 8, "female", 8_500_000, "reg", "TK/0", "07-19"},
	// Left during the last year (see departures).
	{"Reza Pahlevi", "reza.pahlevi@mixist.id", "CRV", "Graphic Designer", "staff", 3, "permanent", 30, "male", 8_000_000, "late", "TK/0", "04-23"},
	{"Anisa Rahma", "anisa.rahma@mixist.id", "SOC", "Community Specialist", "staff", 7, "permanent", 18, "female", 7_500_000, "late", "TK/0", "11-09"},
	{"Galih Prakoso", "galih.prakoso@mixist.id", "SLS", "Account Executive", "staff", 17, "contract", 14, "male", 8_000_000, "reg", "TK/0", "03-27"},
	{"Dewi Maharani", "dewi.maharani@mixist.id", "FIN", "Accounting Staff", "staff", 21, "permanent", 26, "female", 8_000_000, "reg", "K/0", "08-08"},
	{"Hendra Gunawan", "hendra.gunawan@mixist.id", "ADS", "Ads Specialist", "staff", 14, "permanent", 20, "male", 9_000_000, "reg", "TK/0", "01-05"},
}

// departure is how a seeded former employee left: days before today and
// the offboarding type.
type departure struct {
	DaysAgo int
	Type    string
	Status  string
	Reason  string
}

var departures = map[string]departure{
	"reza.pahlevi@mixist.id":   {300, "voluntary", "resigned", "Pindah ke agensi lain"},
	"anisa.rahma@mixist.id":    {210, "voluntary", "resigned", "Melanjutkan studi S2"},
	"galih.prakoso@mixist.id":  {120, "end_of_contract", "resigned", "Kontrak PKWT selesai, tidak diperpanjang"},
	"dewi.maharani@mixist.id":  {60, "voluntary", "resigned", "Ikut pasangan pindah ke Surabaya"},
	"hendra.gunawan@mixist.id": {150, "termination", "terminated", "Restrukturisasi tim Ads"},
}

// historyMonths is how many full months of attendance, leave, overtime and
// payroll the seed builds before the current month.
const historyMonths = 12

// Seeder holds the store and the use cases it drives.
type Seeder struct {
	D       app.Deps
	Payroll *app.Payroll
	now     time.Time
	today   string
	actor   *string
	admin   app.Actor
	docs    app.Documents
}

func New(d app.Deps, docs app.Documents) *Seeder {
	now := d.Clock.Now()
	actor := "hr-seed"
	return &Seeder{D: d, Payroll: app.NewPayroll(d, docs), now: now, today: hris.TodayWIB(now), actor: &actor, docs: docs,
		admin: app.Actor{Principal: authn.Principal{UserID: actor, Role: authn.RoleAdmin, Name: "Seeder"}}}
}

// Run seeds everything; users are identity accounts to link by email (may
// be empty when identity is unreachable).
func (s *Seeder) Run(ctx context.Context, users []app.UserInfo) error {
	depts, err := s.departments(ctx)
	if err != nil {
		return fmt.Errorf("departments: %w", err)
	}
	positions, err := s.positions(ctx, depts)
	if err != nil {
		return fmt.Errorf("positions: %w", err)
	}
	shifts, err := s.shifts(ctx)
	if err != nil {
		return fmt.Errorf("shifts: %w", err)
	}
	emps, err := s.employees(ctx, depts, positions, users)
	if err != nil {
		return fmt.Errorf("employees: %w", err)
	}
	steps := []struct {
		name string
		fn   func(context.Context, []domain.Employee, map[string]string) error
	}{
		{"schedules", s.schedules}, {"holidays", s.holidays}, {"contracts", s.contracts}, {"salaries", s.salaries},
		{"leaves", s.leaves}, {"leave history", s.leaveHistory}, {"attendance", s.attendance}, {"overtime", s.overtime},
		{"overtime history", s.overtimeHistory}, {"offboarding", s.offboarding}, {"loans", s.loans},
		{"announcements", s.announcements}, {"onboarding", s.onboarding}, {"payroll", s.payrollRuns},
		{"history", s.history}, {"documents", s.documents}, {"offboarding in progress", s.offboardingInProgress},
		{"request statuses", s.requestStatuses}, {"contract statuses", s.contractStatuses},
		{"more announcements", s.moreAnnouncements}, {"loan statuses", s.loanStatuses}, {"additions", s.additions},
		{"open runs", s.openRuns}, {"today", s.todayAttendance},
	}
	for _, st := range steps {
		if err := st.fn(ctx, emps, shifts); err != nil {
			return fmt.Errorf("%s: %w", st.name, err)
		}
	}
	return nil
}

func (s *Seeder) departments(ctx context.Context) (map[string]domain.Department, error) {
	existing, err := s.D.Store.ListDepartments(ctx, true)
	if err != nil {
		return nil, err
	}
	out := map[string]domain.Department{}
	for _, d := range existing {
		out[d.Code] = d
	}
	for _, d := range departments {
		if _, ok := out[d.Code]; ok {
			continue
		}
		dep := domain.Department{ID: ids.New(), Code: d.Code, Name: d.Name, IsActive: true}
		if err := s.D.Store.InsertDepartment(ctx, dep, s.actor); err != nil {
			return nil, err
		}
		out[d.Code] = dep
	}
	return out, nil
}

func (s *Seeder) positions(ctx context.Context, depts map[string]domain.Department) (map[string]string, error) {
	existing, err := s.D.Store.ListPositions(ctx, "", true)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, p := range existing {
		out[p.Title] = p.ID
	}
	for _, p := range people {
		if _, ok := out[p.Title]; ok {
			continue
		}
		deptID := depts[p.Dept].ID
		pos := domain.Position{ID: ids.New(), Title: p.Title, DepartmentID: &deptID, Level: p.Level,
			SalaryMin: p.Base * 0.8, SalaryMax: p.Base * 1.3, IsActive: true}
		if err := s.D.Store.InsertPosition(ctx, pos, s.actor); err != nil {
			return nil, err
		}
		out[p.Title] = pos.ID
	}
	return out, nil
}

func (s *Seeder) shifts(ctx context.Context) (map[string]string, error) {
	existing, err := s.D.Store.ListShifts(ctx, true)
	if err != nil {
		return nil, err
	}
	byName := map[string]string{}
	for _, sh := range existing {
		byName[sh.Name] = sh.ID
	}
	defs := []struct{ key, name, start, end string }{
		{"reg", "Reguler", "09:00", "18:00"},
		{"late", "Siang (Kreatif)", "11:00", "20:00"},
	}
	out := map[string]string{}
	for i, d := range defs {
		if id, ok := byName[d.name]; ok {
			out[d.key] = id
			continue
		}
		sh := domain.Shift{ID: ids.New(), Name: d.name, StartTime: d.start, EndTime: d.end, BreakMinutes: 60,
			LateToleranceMinutes: 15, IsActive: true, SortOrder: i + 1}
		if err := s.D.Store.InsertShift(ctx, sh, s.actor); err != nil {
			return nil, err
		}
		out[d.key] = sh.ID
	}
	return out, nil
}

// employees creates the people missing by email, wires managers and
// department heads, and links identity users with the same email.
func (s *Seeder) employees(ctx context.Context, depts map[string]domain.Department, positions map[string]string, users []app.UserInfo) ([]domain.Employee, error) {
	all, _, err := s.D.Store.ListEmployees(ctx, domain.EmployeeFilter{IncludeInactive: true})
	if err != nil {
		return nil, err
	}
	byEmail := map[string]domain.Employee{}
	for _, e := range all {
		byEmail[strings.ToLower(e.Email)] = e
	}
	userByEmail := map[string]string{}
	for _, u := range users {
		userByEmail[strings.ToLower(u.Email)] = u.ID
	}
	ids_ := make([]string, len(people))
	for i, p := range people {
		if e, ok := byEmail[p.Email]; ok {
			ids_[i] = e.ID
			continue
		}
		join := hris.AddDaysISO(s.today, -p.JoinMonthsAgo*30-7)
		year := join[:4]
		birthYear := 1985 + i%14
		birth := fmt.Sprintf("%d-%s", birthYear, p.Birth)
		deptID, posID := depts[p.Dept].ID, positions[p.Title]
		e := domain.Employee{ID: ids.New(), EmployeeNo: fmt.Sprintf("EMP-%s-%05d", year, i+1), FullName: p.Name, Email: p.Email,
			Phone: fmt.Sprintf("08%010d", 1234500000+i*7919), NIK: fmt.Sprintf("3174%012d", 10000000+i*104729),
			NPWP: fmt.Sprintf("%02d.%03d.%03d.%d-%03d.000", 70+i%20, 100+i, 200+i*3, i%10, 300+i), BpjsTkNo: fmt.Sprintf("2001%07d", 1000+i),
			BpjsKesNo: fmt.Sprintf("0001%09d", 5000+i), BirthPlace: []string{"Jakarta", "Bandung", "Surabaya", "Yogyakarta", "Medan"}[i%5],
			BirthDate: &birth, Gender: p.Gender, MaritalStatus: map[bool]string{true: "married", false: "single"}[strings.HasPrefix(p.PTKP, "K/")],
			Religion: "Islam", Address: fmt.Sprintf("Jl. Kemang Raya No. %d, Jakarta Selatan", 10+i), DepartmentID: &deptID, PositionID: &posID,
			EmploymentStatus: p.Status, IsActive: true, JoinDate: join, BankName: []string{"BCA", "Mandiri", "BNI", "BRI"}[i%4],
			BankAccountNo: fmt.Sprintf("%010d", 5270000000+i*3571), BankAccountName: p.Name,
			EmergencyName: "Keluarga " + strings.Fields(p.Name)[0], EmergencyPhone: fmt.Sprintf("08%010d", 1987600000+i*31), EmergencyRelation: "Orang tua",
			CreatedBy: s.actor, UpdatedBy: s.actor}
		if p.Status == "probation" {
			end := hris.AddDaysISO(join, 90)
			e.ProbationEndDate = &end
		}
		if dep, ok := departures[p.Email]; ok {
			end := hris.AddDaysISO(s.today, -dep.DaysAgo)
			e.EndDate, e.IsActive, e.EmploymentStatus = &end, false, dep.Status
		}
		if err := s.D.Store.InsertEmployee(ctx, e); err != nil {
			return nil, err
		}
		ids_[i] = e.ID
	}
	for i, p := range people {
		e, err := s.D.Store.GetEmployee(ctx, ids_[i])
		if err != nil || e == nil {
			return nil, err
		}
		changed := false
		if p.Manager > 0 && e.ManagerID == nil {
			m := ids_[p.Manager-1]
			e.ManagerID, changed = &m, true
		}
		if changed {
			e.UpdatedBy = s.actor
			if _, err := s.D.Store.UpdateEmployee(ctx, *e); err != nil {
				return nil, err
			}
		}
		if uid, ok := userByEmail[p.Email]; ok && e.UserID == nil {
			if _, err := s.D.Store.SetEmployeeUser(ctx, e.ID, &uid, s.actor); err != nil {
				return nil, err
			}
		}
	}
	heads := map[string]int{"MGT": 1, "CRV": 3, "SOC": 7, "SEO": 11, "ADS": 14, "SLS": 17, "FIN": 21, "HR": 23}
	for code, idx := range heads {
		d := depts[code]
		if d.HeadEmployeeID != nil {
			continue
		}
		head := ids_[idx-1]
		d.HeadEmployeeID = &head
		if _, err := s.D.Store.UpdateDepartment(ctx, d, s.actor); err != nil {
			return nil, err
		}
	}
	out := make([]domain.Employee, len(ids_))
	for i, id := range ids_ {
		e, err := s.D.Store.GetEmployee(ctx, id)
		if err != nil || e == nil {
			return nil, err
		}
		out[i] = *e
	}
	return out, nil
}

func (s *Seeder) schedules(ctx context.Context, emps []domain.Employee, shifts map[string]string) error {
	rows, err := s.D.Store.ListScheduleRows(ctx, idsOf(emps))
	if err != nil {
		return err
	}
	has := map[string]bool{}
	for _, r := range rows {
		has[r.EmployeeID] = true
	}
	for i, e := range emps {
		if has[e.ID] {
			continue
		}
		shift := shifts[people[i].Shift]
		week := make([]domain.ScheduleRow, 7)
		for d := 1; d <= 7; d++ {
			week[d-1] = domain.ScheduleRow{ID: ids.New(), DayOfWeek: d}
			if d <= 5 {
				week[d-1].ShiftID = &shift
			}
		}
		from := e.JoinDate
		if err := s.D.Store.ReplaceSchedule(ctx, e.ID, from, hris.AddDaysISO(from, -1), week, s.actor); err != nil {
			return err
		}
	}
	return nil
}

func (s *Seeder) holidays(ctx context.Context, _ []domain.Employee, _ map[string]string) error {
	first, _ := strconv.Atoi(s.historyStart()[:4])
	last, _ := strconv.Atoi(s.today[:4])
	for y := first; y <= last; y++ {
		if err := s.holidaysOf(ctx, y); err != nil {
			return err
		}
	}
	return nil
}

func (s *Seeder) holidaysOf(ctx context.Context, y int) error {
	year := strconv.Itoa(y)
	defs := []domain.Holiday{
		{Date: year + "-01-01", Name: "Tahun Baru Masehi", Type: "nasional"},
		{Date: year + "-05-01", Name: "Hari Buruh Internasional", Type: "nasional"},
		{Date: year + "-06-01", Name: "Hari Lahir Pancasila", Type: "nasional"},
		{Date: year + "-08-17", Name: "Hari Kemerdekaan Republik Indonesia", Type: "nasional"},
		{Date: year + "-12-24", Name: "Cuti Bersama Natal", Type: "cuti_bersama", DeductsLeave: true},
		{Date: year + "-12-25", Name: "Hari Raya Natal", Type: "nasional"},
		{Date: year + "-10-30", Name: "Ulang Tahun Mixist", Type: "perusahaan"},
	}
	switch year {
	case "2025":
		defs = append(defs,
			domain.Holiday{Date: "2025-03-31", Name: "Hari Raya Idul Fitri 1446 H", Type: "nasional"},
			domain.Holiday{Date: "2025-04-01", Name: "Hari Raya Idul Fitri 1446 H (hari kedua)", Type: "nasional"})
	case "2026":
		defs = append(defs,
			domain.Holiday{Date: "2026-03-20", Name: "Cuti Bersama Idul Fitri", Type: "cuti_bersama", DeductsLeave: true},
			domain.Holiday{Date: "2026-03-21", Name: "Hari Raya Idul Fitri 1447 H", Type: "nasional"},
			domain.Holiday{Date: "2026-03-22", Name: "Hari Raya Idul Fitri 1447 H (hari kedua)", Type: "nasional"},
			domain.Holiday{Date: "2026-03-23", Name: "Cuti Bersama Idul Fitri (2)", Type: "cuti_bersama", DeductsLeave: true})
	}
	existing, err := s.D.Store.ListHolidays(ctx, year+"-01-01", year+"-12-31", false)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, h := range existing {
		seen[h.Date+h.Name] = true
	}
	for _, h := range defs {
		if seen[h.Date+h.Name] {
			continue
		}
		h.ID, h.Status = ids.New(), "aktif"
		if err := s.D.Store.InsertHoliday(ctx, h, s.actor); err != nil {
			return err
		}
	}
	return nil
}

// contracts: an active PKWTT for permanent and probation staff, a PKWT for
// contract staff (one of them ending within the reminder window).
func (s *Seeder) contracts(ctx context.Context, emps []domain.Employee, _ map[string]string) error {
	existing, err := s.D.Store.ListContracts(ctx, domain.ContractFilter{})
	if err != nil {
		return err
	}
	has := map[string]bool{}
	for _, c := range existing {
		has[c.EmployeeID] = true
	}
	seq := len(existing)
	contractIndex := 0
	for i, e := range emps {
		if has[e.ID] {
			continue
		}
		p := people[i]
		base := p.Base
		c := domain.Contract{ID: ids.New(), EmployeeID: e.ID, ContractType: "pkwtt", StartDate: e.JoinDate, Sequence: 1,
			PositionTitle: p.Title, DepartmentName: e.DepartmentName, WorkLocation: "Kantor Mixist, Jakarta Selatan", BaseSalary: &base, CreatedBy: s.actor}
		if p.Status == "contract" {
			// The first contract ends inside the 30-day reminder window.
			end := hris.AddDaysISO(s.today, 20+contractIndex*75)
			contractIndex++
			c.ContractType, c.EndDate = "pkwt", &end
		}
		if p.Status == "probation" {
			c.ProbationEndDate = e.ProbationEndDate
		}
		if e.EndDate != nil {
			c.EndDate = e.EndDate
		}
		// One running sequence across seeded contracts keeps every number
		// unique; contracts made later in the app count these first.
		t, _ := hris.ParseDate(c.StartDate)
		seq++
		c.ContractNumber = hris.BuildContractNumber(c.ContractType, seq, t)
		if err := s.D.Store.InsertContract(ctx, c); err != nil {
			return err
		}
		signed := c.StartDate
		if _, err := s.D.Store.SetContractStatus(ctx, domain.ContractStatusUpdate{ID: c.ID, From: "draft", To: "active", SignedAt: &signed, Actor: s.actor}); err != nil {
			return err
		}
		if e.EndDate != nil {
			if _, err := s.D.Store.SetContractStatus(ctx, domain.ContractStatusUpdate{ID: c.ID, From: "active", To: "ended", Actor: s.actor}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Seeder) salaries(ctx context.Context, emps []domain.Employee, _ map[string]string) error {
	for i, e := range emps {
		cur, err := s.D.Store.ActiveSalary(ctx, e.ID)
		if err != nil {
			return err
		}
		if cur != nil {
			continue
		}
		p := people[i]
		sal := domain.Salary{ID: ids.New(), EmployeeID: e.ID, BaseSalary: p.Base, FixedAllowance: roundK(p.Base * 0.1),
			TransportAllowance: 750_000, MealAllowance: 660_000, PTKPStatus: p.PTKP, IsTaxable: true,
			BpjsTkEnrolled: true, BpjsKesEnrolled: true, TaperaEnrolled: false, EffectiveDate: e.JoinDate, IsActive: true}
		if p.Level == "director" || p.Level == "manager" {
			sal.HousingAllowance = roundK(p.Base * 0.1)
		}
		if err := s.D.Store.InsertSalary(ctx, sal, hris.AddDaysISO(e.JoinDate, -1), s.actor); err != nil {
			return err
		}
	}
	return nil
}

func roundK(v float64) float64 { return float64(int64(v/1000)) * 1000 }

// leaves: a balance for this year, an approved annual leave last month for
// a few people, a sick day, and requests waiting for the manager and HR.
func (s *Seeder) leaves(ctx context.Context, emps []domain.Employee, _ map[string]string) error {
	year := s.now.Year()
	existing, _, err := s.D.Store.ListLeaves(ctx, domain.LeaveFilter{Limit: 1})
	if err != nil {
		return err
	}
	first := yearOf(s.historyStart())
	for _, e := range emps {
		join := e.JoinDate
		for y := first; y <= year; y++ {
			if !employedBetween(e, fmt.Sprintf("%d-01-01", y), fmt.Sprintf("%d-12-31", y)) {
				continue
			}
			if _, err := s.D.Store.EnsureLeaveBalance(ctx, domain.LeaveBalance{ID: ids.New(), EmployeeID: e.ID, Year: y,
				AnnualTotal: hris.ProratedAnnualQuota(&join, y)}, s.actor); err != nil {
				return err
			}
		}
	}
	if len(existing) > 0 {
		return nil
	}
	lastMonth := s.lastMonthStart()
	type req struct {
		idx        int
		typ        string
		start, end string
		status     string
		reason     string
	}
	reqs := []req{
		{4, "annual", weekday(lastMonth, 8), weekday(lastMonth, 9), hris.LeaveApproved, "Liburan keluarga ke Bali"},
		{12, "sick", weekday(lastMonth, 15), weekday(lastMonth, 15), hris.LeaveApproved, "Demam, ada surat dokter"},
		{18, "annual", weekday(lastMonth, 3), weekday(lastMonth, 3), hris.LeaveApproved, "Urusan keluarga"},
		{9, "annual", weekday(s.today, 10), weekday(s.today, 11), hris.LeavePending, "Menghadiri pernikahan saudara di Bandung"},
		{16, "annual", weekday(s.today, 6), weekday(s.today, 6), hris.LeaveManagerApproved, "Mengurus dokumen di kantor imigrasi"},
		{22, "unpaid", weekday(lastMonth, 20), weekday(lastMonth, 20), hris.LeaveApproved, "Izin pribadi"},
	}
	for _, r := range reqs {
		e := emps[r.idx-1]
		days, _, err := s.workingDays(ctx, r.start, r.end)
		if err != nil {
			return err
		}
		l := domain.Leave{ID: ids.New(), EmployeeID: e.ID, LeaveType: r.typ, StartDate: r.start, EndDate: r.end, TotalDays: days,
			Reason: r.reason, Status: r.status, CreatedBy: s.actor}
		decided := s.now.Add(-24 * time.Hour)
		if r.status == hris.LeaveManagerApproved || r.status == hris.LeaveApproved {
			l.ManagerApprovedBy, l.ManagerApprovedAt = e.ManagerID, &decided
		}
		if r.status == hris.LeaveApproved {
			hr := emps[22].ID
			l.ApprovedBy, l.ApprovedAt = &hr, &decided
		}
		if err := s.D.Store.InsertLeave(ctx, l); err != nil {
			return err
		}
		if r.status == hris.LeaveApproved {
			u := domain.LeaveUsage{}
			switch r.typ {
			case "annual":
				u.Annual = days
			case "sick":
				u.Sick = days
			case "unpaid":
				u.Unpaid = days
			}
			if err := s.D.Store.AddLeaveUsage(ctx, e.ID, yearOf(r.start), u); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Seeder) workingDays(ctx context.Context, start, end string) (int, []hris.ExcludedHoliday, error) {
	hs, err := s.D.Store.ListHolidays(ctx, start, end, true)
	if err != nil {
		return 0, nil, err
	}
	rows := make([]hris.Holiday, len(hs))
	for i, h := range hs {
		rows[i] = hris.Holiday{Date: h.Date, Name: h.Name, Type: h.Type, DeductsLeave: h.DeductsLeave}
	}
	n, ex := hris.DescribeLeaveDays(start, end, hris.IndexHolidays(rows))
	return max(n, 1), ex, nil
}

// attendance from the start of the history to yesterday (or the last
// working day of a former employee), skipping weekends, holidays and
// approved leave; clock times jitter deterministically per employee and day,
// with Mondays and Fridays running later. Existing days are kept.
func (s *Seeder) attendance(ctx context.Context, emps []domain.Employee, _ map[string]string) error {
	from := s.historyStart()
	to := hris.AddDaysISO(s.today, -1)
	rows, err := s.D.Store.ListScheduleRows(ctx, idsOf(emps))
	if err != nil {
		return err
	}
	holidays, err := s.D.Store.ListHolidays(ctx, from, to, true)
	if err != nil {
		return err
	}
	off := map[string]bool{}
	for _, h := range holidays {
		if !h.DeductsLeave {
			off[h.Date] = true
		}
	}
	leaves, _, err := s.D.Store.ListLeaves(ctx, domain.LeaveFilter{Status: hris.LeaveApproved, From: from, To: to})
	if err != nil {
		return err
	}
	validator := emps[23].ID
	for _, e := range emps {
		var mine []domain.ScheduleRow
		for _, r := range rows {
			if r.EmployeeID == e.ID {
				mine = append(mine, r)
			}
		}
		last := to
		if e.EndDate != nil {
			last = min(last, *e.EndDate)
		}
		for _, d := range hris.EachDateISO(max(from, e.JoinDate), last) {
			if off[d] || onLeave(leaves, e.ID, d) {
				continue
			}
			sh := app.ShiftOn(mine, d)
			if sh == nil {
				continue
			}
			h := jitter(e.EmployeeNo + d)
			times := hris.ShiftTimes{StartTime: sh.StartTime, EndTime: sh.EndTime, LateToleranceMinutes: sh.LateToleranceMinutes}
			start, end := hris.ScheduledWindow(d, times)
			a := domain.Attendance{ID: ids.New(), EmployeeID: e.ID, Date: d, ShiftID: &sh.ID, ScheduledStart: &start, ScheduledEnd: &end, Status: "present"}
			if h%37 == 0 {
				a.Status = "absent"
				if err := s.D.Store.InsertAttendanceRecord(ctx, a, s.actor); err != nil {
					return err
				}
				continue
			}
			in := start.Add(time.Duration(int(h%41)-15+lateBias(d)) * time.Minute)
			out := end.Add(time.Duration(h%95) * time.Minute)
			late, mins := hris.Lateness(in, d, times)
			brk := 60
			a.ClockIn, a.ClockOut, a.IsLate, a.LateMinutes = &in, &out, late, mins
			a.WorkHours = hris.WorkHours(in, out, &brk)
			if out.After(end) {
				a.OvertimeHours = float64(int(out.Sub(end).Minutes()*100/60)) / 100
			}
			if late {
				a.Status = "late"
			} else if h%29 == 0 {
				a.Status = "remote"
			}
			if d < hris.AddDaysISO(s.today, -7) {
				at := out.Add(24 * time.Hour)
				a.ValidatedBy, a.ValidatedAt = &validator, &at
			}
			if err := s.D.Store.InsertAttendanceRecord(ctx, a, s.actor); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Seeder) overtime(ctx context.Context, emps []domain.Employee, _ map[string]string) error {
	existing, err := s.D.Store.ListOvertime(ctx, domain.OvertimeFilter{})
	if err != nil || len(existing) > 0 {
		return err
	}
	lastMonth := s.lastMonthStart()
	defs := []struct {
		idx                      int
		date, start, end, status string
		reason                   string
	}{
		{5, weekday(lastMonth, 10), "18:00", "20:30", "approved", "Revisi key visual kampanye Ramadan klien"},
		{6, weekday(lastMonth, 12), "20:00", "22:00", "approved", "Rendering video launching produk"},
		{15, weekday(lastMonth, 17), "18:00", "20:00", "approved", "Optimasi iklan akhir bulan"},
		{9, weekday(s.today, -2), "18:00", "19:30", "pending", "Live report event klien"},
	}
	for _, d := range defs {
		e := emps[d.idx-1]
		o := domain.Overtime{ID: ids.New(), EmployeeID: e.ID, Date: d.date, StartTime: d.start, EndTime: d.end,
			Hours: hris.OvertimeHoursFromTimes(d.start, d.end), Source: "employee", Status: d.status, Reason: d.reason, RequestedBy: &e.ID}
		if d.status == "approved" {
			at := s.now.Add(-48 * time.Hour)
			o.DecidedBy, o.DecidedAt = e.ManagerID, &at
		}
		if err := s.D.Store.InsertOvertime(ctx, o, s.actor); err != nil {
			return err
		}
	}
	return nil
}

// loans: a kasbon approved two months ago, repaid through payroll from
// last month.
func (s *Seeder) loans(ctx context.Context, emps []domain.Employee, _ map[string]string) error {
	existing, err := s.D.Store.ListLoans(ctx, "", "")
	if err != nil || len(existing) > 0 {
		return err
	}
	e := emps[17]
	_, monthly, total := payroll.LoanTerms(3_000_000, 0, 3)
	loan := domain.Loan{ID: ids.New(), EmployeeID: e.ID, LoanType: "kasbon", PrincipalAmount: 3_000_000, TenorMonths: 3,
		MonthlyInstallment: monthly, RemainingBalance: total, Purpose: "Biaya perbaikan motor untuk transport kerja", CreatedBy: s.actor}
	if err := s.D.Store.InsertLoan(ctx, loan); err != nil {
		return err
	}
	last, _ := hris.ParseDate(s.lastMonthStart())
	approved := last.AddDate(0, -1, 5)
	hr := emps[22].ID
	_, err = s.D.Store.ApproveLoan(ctx, loan.ID, &hr, approved, int(last.Month()), last.Year())
	if err != nil {
		return err
	}
	pending := domain.Loan{ID: ids.New(), EmployeeID: emps[12].ID, LoanType: "loan", PrincipalAmount: 2_000_000, TenorMonths: 4,
		MonthlyInstallment: 500_000, RemainingBalance: 2_000_000, Purpose: "Biaya sekolah anak", CreatedBy: s.actor}
	return s.D.Store.InsertLoan(ctx, pending)
}

func (s *Seeder) announcements(ctx context.Context, emps []domain.Employee, _ map[string]string) error {
	existing, err := s.D.Store.ListAnnouncements(ctx, "")
	if err != nil || len(existing) > 0 {
		return err
	}
	published := s.now.Add(-72 * time.Hour)
	items := []domain.Announcement{
		{Title: "Jadwal libur Natal dan cuti bersama", Body: "Kantor libur 24 dan 25 Desember. Cuti bersama 24 Desember memotong cuti tahunan. Tim on-call akan dijadwalkan per klien.",
			Tags: []string{"libur", "cuti"}, Status: "published", IsPinned: true, TargetScope: "global", PublishAt: &published},
		{Title: "Town hall bulanan Mixist", Body: "Town hall Jumat ini pukul 16.00 di ruang utama dan Google Meet. Agenda: hasil kampanye kuartal ini dan rencana klien baru.",
			Tags: []string{"acara"}, Status: "published", TargetScope: "global", PublishAt: &published},
		{Title: "Pengumpulan bukti reimburse", Body: "Bukti reimburse bulan ini paling lambat tanggal 25 melalui menu Keuangan.", Tags: []string{"keuangan"}, Status: "draft", TargetScope: "global"},
	}
	for _, a := range items {
		a.ID = ids.New()
		if _, err := s.D.Store.SaveAnnouncement(ctx, a, true, s.actor); err != nil {
			return err
		}
	}
	return nil
}

// onboarding gives everyone the checklist: done for people past their
// first 90 days, partly done for the two still in probation.
func (s *Seeder) onboarding(ctx context.Context, emps []domain.Employee, _ map[string]string) error {
	for i, e := range emps {
		done := len(app.OnboardingTemplate)
		if i == 24 || i == 26 {
			done = 6
		}
		if err := s.onboardingOf(ctx, e, emps[23].ID, done); err != nil {
			return err
		}
	}
	return nil
}

func (s *Seeder) onboardingOf(ctx context.Context, newest domain.Employee, hrID string, done int) error {
	tasks, err := s.D.Store.ListOnboardingTasks(ctx, newest.ID)
	if err != nil || len(tasks) > 0 {
		return err
	}
	for i, t := range app.OnboardingTemplate {
		due := hris.AddDaysISO(newest.JoinDate, t.Days)
		task := domain.OnboardingTask{ID: ids.New(), EmployeeID: newest.ID, TaskName: t.Name, Category: t.Category, Priority: t.Priority, DueDate: &due}
		if err := s.D.Store.InsertOnboardingTask(ctx, task, s.actor); err != nil {
			return err
		}
		if i < done {
			task.IsCompleted, task.CompletedBy = true, &hrID
			if _, err := s.D.Store.UpdateOnboardingTask(ctx, task); err != nil {
				return err
			}
		}
	}
	return nil
}

// payrollRuns calculates, approves and pays every month of the history up
// to last month, oldest first, each stamped on its payday (the 25th).
func (s *Seeder) payrollRuns(ctx context.Context, _ []domain.Employee, _ map[string]string) error {
	for _, m := range monthStarts(s.historyStart(), hris.AddDaysISO(s.lastMonthStart(), 1)) {
		t, _ := hris.ParseDate(m)
		if err := s.payrollRun(ctx, int(t.Month()), t.Year()); err != nil {
			return fmt.Errorf("%s: %w", m[:7], err)
		}
	}
	return nil
}

func (s *Seeder) payrollRun(ctx context.Context, month, year int) error {
	d := s.D
	d.Clock = clock.Fixed{T: time.Date(year, time.Month(month), 25, 3, 0, 0, 0, time.UTC)}
	pay := app.NewPayroll(d, s.docs)
	run, err := d.Store.GetPayrollRunByPeriod(ctx, month, year)
	if err != nil {
		return err
	}
	if run == nil {
		if run, err = pay.CreateRun(ctx, s.admin, month, year, ""); err != nil {
			return err
		}
	}
	if run.Status == payroll.RunDraft || run.Status == payroll.RunCalculated {
		if _, _, err := pay.Calculate(ctx, s.admin, run.ID, false); err != nil {
			return err
		}
		if _, err := pay.Approve(ctx, s.admin, run.ID); err != nil {
			return err
		}
	}
	if r, err := d.Store.GetPayrollRun(ctx, run.ID); err != nil {
		return err
	} else if r.Status == payroll.RunApproved {
		_, _, err = pay.MarkPaid(ctx, s.admin, run.ID, "Ditransfer via BCA payroll")
		return err
	}
	return nil
}

// leaveHistory spreads approved leave from the start of the history to the
// end of next month (more of it in December) and queues pending requests of
// different ages for the approval widgets. A request is skipped when it
// overlaps another one or lands on a day that already has attendance, so
// reruns add nothing.
func (s *Seeder) leaveHistory(ctx context.Context, emps []domain.Employee, _ map[string]string) error {
	from, until := s.historyStart(), monthEnd(addMonth(s.lastMonthStart(), 2))
	hr := emps[22].ID
	reasons := map[string]string{"annual": "Liburan keluarga", "sick": "Sakit, ada surat dokter", "emergency": "Keluarga sakit", "unpaid": "Izin pribadi"}
	for _, m := range monthStarts(from, until) {
		mEnd := monthEnd(m)
		for _, e := range emps {
			h := jitter(e.EmployeeNo + m + "leave")
			typ, days := pickLeave(h, m)
			start := weekday(m, int(h%20)+1)
			if typ == "" || start > mEnd || !employedBetween(e, start, start) {
				continue
			}
			end := start
			for k := 1; k < days; k++ {
				if next := weekday(end, 1); next <= mEnd {
					end = next
				}
			}
			if e.EndDate != nil && end > *e.EndDate {
				end = start
			}
			if free, err := s.leaveFree(ctx, e.ID, start, end); err != nil || !free {
				if err != nil {
					return err
				}
				continue
			}
			n, _, err := s.workingDays(ctx, start, end)
			if err != nil {
				return err
			}
			requested := earliest(wibMorning(hris.AddDaysISO(start, -3-int(h%9))), s.now.Add(-48*time.Hour))
			decided := earliest(requested.Add(26*time.Hour), s.now.Add(-time.Hour))
			l := domain.Leave{ID: ids.New(), EmployeeID: e.ID, LeaveType: typ, StartDate: start, EndDate: end, TotalDays: n,
				Reason: reasons[typ], Status: hris.LeaveApproved, ManagerApprovedBy: e.ManagerID, ManagerApprovedAt: &decided,
				ApprovedBy: &hr, ApprovedAt: &decided, CreatedBy: s.actor, CreatedAt: requested}
			if err := s.D.Store.InsertLeave(ctx, l); err != nil {
				return err
			}
			u := domain.LeaveUsage{}
			switch typ {
			case "annual":
				u.Annual = n
			case "sick":
				u.Sick = n
			case "unpaid":
				u.Unpaid = n
			default:
				u.Other = n
			}
			if err := s.D.Store.AddLeaveUsage(ctx, e.ID, yearOf(start), u); err != nil {
				return err
			}
		}
	}
	// Pending requests waiting 1 to 25 days, for the approval aging chart.
	pending := []struct {
		idx, age, ahead int
		typ, status     string
		reason          string
	}{
		{4, 1, 9, "annual", hris.LeavePending, "Mudik ke Yogyakarta"},
		{12, 4, 14, "annual", hris.LeavePending, "Acara wisuda adik"},
		{19, 9, 18, "annual", hris.LeaveManagerApproved, "Liburan akhir tahun"},
		{22, 16, 24, "annual", hris.LeavePending, "Renovasi rumah"},
		{26, 25, 30, "annual", hris.LeaveManagerApproved, "Menikah"},
	}
	for _, p := range pending {
		e := emps[p.idx-1]
		waiting, _, err := s.D.Store.ListLeaves(ctx, domain.LeaveFilter{EmployeeIDs: []string{e.ID},
			Statuses: []string{hris.LeavePending, hris.LeaveManagerApproved}, Limit: 1})
		if err != nil {
			return err
		}
		start := weekday(s.today, p.ahead)
		if busy, err := s.D.Store.LeaveOverlaps(ctx, e.ID, start, start); err != nil || busy || len(waiting) > 0 {
			if err != nil {
				return err
			}
			continue
		}
		requested := wibMorning(hris.AddDaysISO(s.today, -p.age))
		l := domain.Leave{ID: ids.New(), EmployeeID: e.ID, LeaveType: p.typ, StartDate: start, EndDate: start, TotalDays: 1,
			Reason: p.reason, Status: p.status, CreatedBy: s.actor, CreatedAt: requested}
		if p.status == hris.LeaveManagerApproved {
			at := requested.Add(20 * time.Hour)
			l.ManagerApprovedBy, l.ManagerApprovedAt = e.ManagerID, &at
		}
		if err := s.D.Store.InsertLeave(ctx, l); err != nil {
			return err
		}
	}
	return nil
}

// leaveFree is whether [start, end] has no other request and, for past
// days, no attendance yet.
func (s *Seeder) leaveFree(ctx context.Context, employeeID, start, end string) (bool, error) {
	if busy, err := s.D.Store.LeaveOverlaps(ctx, employeeID, start, end); err != nil || busy {
		return false, err
	}
	for _, d := range hris.EachDateISO(start, min(end, hris.AddDaysISO(s.today, -1))) {
		if id, err := s.D.Store.AttendanceIDOn(ctx, employeeID, d); err != nil || id != "" {
			return false, err
		}
	}
	return true, nil
}

// pickLeave decides from a hash whether an employee takes leave in a month,
// which type and for how many working days.
func pickLeave(h uint32, month string) (string, int) {
	r := int(h % 100)
	annual := 14
	if month[5:7] == "12" {
		annual = 34
	}
	switch {
	case r < annual:
		return "annual", 1 + int(h/7%3)
	case r < annual+7:
		return "sick", 1 + int(h/11%2)
	case r < annual+9:
		return "emergency", 1
	case r < annual+10:
		return "unpaid", 1
	}
	return "", 0
}

// overtimeHistory adds approved overtime (a few rejected) from the start of
// the history to yesterday: creative and ads work late most, more around
// month end. Days that already have a request are kept as they are.
func (s *Seeder) overtimeHistory(ctx context.Context, emps []domain.Employee, _ map[string]string) error {
	from, until := s.historyStart(), hris.AddDaysISO(s.today, -1)
	existing, err := s.D.Store.ListOvertime(ctx, domain.OvertimeFilter{From: from, To: until})
	if err != nil {
		return err
	}
	taken := map[string]bool{}
	for _, o := range existing {
		taken[o.EmployeeID+o.Date] = true
	}
	chance := map[string]uint32{"CRV": 9, "ADS": 7, "SOC": 5, "SEO": 3, "SLS": 3, "FIN": 2, "HR": 1, "MGT": 1}
	reasons := []string{"Revisi materi kampanye klien", "Rendering video konten", "Optimasi iklan akhir bulan",
		"Laporan bulanan klien", "Persiapan pitching klien baru", "Live report event klien"}
	for i, e := range emps {
		p := people[i]
		for _, d := range hris.EachDateISO(max(from, e.JoinDate), until) {
			if hris.IsWeekend(d) || !employedBetween(e, d, d) || taken[e.ID+d] {
				continue
			}
			h := jitter(e.EmployeeNo + d + "ot")
			odds := chance[p.Dept]
			if day := d[8:10]; day >= "24" {
				odds += 3
			}
			if h%100 >= odds {
				continue
			}
			shiftEnd := 18
			if p.Shift == "late" {
				shiftEnd = 20
			}
			minutes := 90 + int(h/100%5)*30
			start := fmt.Sprintf("%02d:00", shiftEnd)
			end := fmt.Sprintf("%02d:%02d", shiftEnd+minutes/60, minutes%60)
			status := "approved"
			if h%17 == 0 {
				status = "rejected"
			}
			requested := wibMorning(d)
			decided := requested.Add(30 * time.Hour)
			o := domain.Overtime{ID: ids.New(), EmployeeID: e.ID, Date: d, StartTime: start, EndTime: end,
				Hours: hris.OvertimeHoursFromTimes(start, end), Source: "employee", Status: status,
				Reason: reasons[int(h/7)%len(reasons)], RequestedBy: &e.ID, DecidedBy: e.ManagerID, DecidedAt: &decided, CreatedAt: requested}
			if err := s.D.Store.InsertOvertime(ctx, o, s.actor); err != nil {
				return err
			}
		}
	}
	return nil
}

// offboarding records a completed exit for each former employee.
func (s *Seeder) offboarding(ctx context.Context, emps []domain.Employee, _ map[string]string) error {
	for i, e := range emps {
		dep, ok := departures[people[i].Email]
		if !ok || e.EndDate == nil {
			continue
		}
		cur, err := s.D.Store.GetOffboarding(ctx, e.ID)
		if err != nil || cur != nil {
			if err != nil {
				return err
			}
			continue
		}
		o := domain.Offboarding{ID: ids.New(), EmployeeID: e.ID, ResignationType: dep.Type,
			ResignationDate: hris.AddDaysISO(*e.EndDate, -30), LastWorkingDay: *e.EndDate, Reason: dep.Reason}
		if err := s.D.Store.InsertOffboarding(ctx, o, s.actor); err != nil {
			return err
		}
		done := wibMorning(*e.EndDate)
		o.Status, o.ClearanceHR, o.ClearanceIT, o.ClearanceFinance, o.ClearanceManager = "completed", true, true, true, true
		o.ExitInterviewNotes, o.CompletedAt = "Wawancara keluar selesai, serah terima pekerjaan lengkap.", &done
		if err := s.D.Store.UpdateOffboarding(ctx, o, s.actor); err != nil {
			return err
		}
	}
	return nil
}

// ─── helpers ─────────────────────────────────────────────────

// historyStart is the first day of the month historyMonths before this one.
func (s *Seeder) historyStart() string {
	t, _ := hris.ParseDate(s.today)
	return time.Date(t.Year(), t.Month()-historyMonths, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}

// monthStarts lists the first day of every month from the month of from up
// to before until.
func monthStarts(from, until string) []string {
	t, _ := hris.ParseDate(from)
	t = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	var out []string
	for d := t.Format("2006-01-02"); d < until; d = t.Format("2006-01-02") {
		out = append(out, d)
		t = t.AddDate(0, 1, 0)
	}
	return out
}

func addMonth(start string, n int) string {
	t, _ := hris.ParseDate(start)
	return t.AddDate(0, n, 0).Format("2006-01-02")
}

func monthEnd(start string) string {
	t, _ := hris.ParseDate(start)
	return t.AddDate(0, 1, -1).Format("2006-01-02")
}

// employedBetween is whether the employee was on the books on some day of
// [from, to].
func employedBetween(e domain.Employee, from, to string) bool {
	return e.JoinDate <= to && (e.EndDate == nil || *e.EndDate >= from)
}

func earliest(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

// wibMorning is 09:00 WIB on date.
func wibMorning(date string) time.Time {
	t, _ := hris.ParseDate(date)
	return t.Add(2 * time.Hour)
}

func (s *Seeder) lastMonthStart() string {
	t, _ := hris.ParseDate(s.today)
	return time.Date(t.Year(), t.Month()-1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}

// weekday is the n-th day from base moved forward to a weekday.
func weekday(base string, n int) string {
	d := hris.AddDaysISO(base, n)
	for hris.IsWeekend(d) {
		d = hris.AddDaysISO(d, 1)
	}
	return d
}

// lateBias shifts clock-ins later at the edges of the week.
func lateBias(date string) int {
	switch hris.IsoDayOfWeek(date) {
	case 1:
		return 6
	case 5:
		return 4
	}
	return 0
}

func jitter(key string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return h.Sum32()
}

func onLeave(leaves []domain.Leave, employeeID, date string) bool {
	for _, l := range leaves {
		if l.EmployeeID == employeeID && l.StartDate <= date && l.EndDate >= date {
			return true
		}
	}
	return false
}

func idsOf(emps []domain.Employee) []string {
	out := make([]string, len(emps))
	for i, e := range emps {
		out[i] = e.ID
	}
	return out
}

func yearOf(date string) int {
	t, _ := hris.ParseDate(date)
	return t.Year()
}
