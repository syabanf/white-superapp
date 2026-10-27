package seed

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"slices"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/app"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/payroll"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Steps that make every HR and self-service screen show data: employment
// history, documents, requests in every status, contracts in every status,
// expired and archived announcements, payroll additions, loans in every
// status and payroll runs in every state. Each item is keyed by something
// stable (a reason, a title, a purpose) and skipped when it exists.

// history records the signed contract, the passed probation and, for
// former employees, the exit on each employee's timeline.
func (s *Seeder) history(ctx context.Context, emps []domain.Employee, _ map[string]string) error {
	for i, e := range emps {
		rows, err := s.D.Store.ListHistory(ctx, e.ID)
		if err != nil {
			return err
		}
		if len(rows) > 0 {
			continue
		}
		p := people[i]
		probation, status := "probation", p.Status
		entries := []domain.HistoryEntry{{ChangeType: "contract", Notes: "Kontrak kerja pertama ditandatangani",
			NewStatus: &probation, NewDepartmentID: e.DepartmentID, NewPositionID: e.PositionID, CreatedAt: wibMorning(e.JoinDate)}}
		if p.Status == "permanent" && p.JoinMonthsAgo >= 4 {
			entries = append(entries, domain.HistoryEntry{ChangeType: "update", Notes: "Lulus masa percobaan, diangkat menjadi karyawan tetap",
				PrevStatus: &probation, NewStatus: &status, CreatedAt: wibMorning(hris.AddDaysISO(e.JoinDate, 90))})
		}
		if p.Level == "lead" || p.Level == "manager" {
			entries = append(entries, domain.HistoryEntry{ChangeType: "update", Notes: "Promosi jabatan menjadi " + p.Title,
				NewPositionID: e.PositionID, CreatedAt: wibMorning(hris.AddDaysISO(e.JoinDate, 400))})
		}
		if dep, ok := departures[p.Email]; ok && e.EndDate != nil {
			entries = append(entries, domain.HistoryEntry{ChangeType: "offboarding", Notes: "Keluar: " + dep.Reason,
				PrevStatus: &status, NewStatus: &dep.Status, CreatedAt: wibMorning(*e.EndDate)})
		}
		for _, h := range entries {
			if h.CreatedAt.After(s.now) {
				continue
			}
			h.ID, h.EmployeeID, h.CreatedBy = ids.New(), e.ID, s.actor
			if err := s.D.Store.InsertHistory(ctx, h); err != nil {
				return err
			}
		}
	}
	return nil
}

// documents gives the first twelve employees an ID card scan, a CV and
// the signed contract.
func (s *Seeder) documents(ctx context.Context, emps []domain.Employee, _ map[string]string) error {
	scan, err := idCardPNG()
	if err != nil {
		return err
	}
	for _, e := range emps[:12] {
		docs, err := s.D.Store.ListDocuments(ctx, e.ID)
		if err != nil {
			return err
		}
		if len(docs) > 0 {
			continue
		}
		files := []struct {
			typ, name string
			data      []byte
		}{
			{"ktp", "KTP " + e.FullName + ".png", scan},
			{"cv", "CV " + e.FullName + ".pdf", tinyPDF("Curriculum Vitae " + e.FullName)},
			{"contract", "Kontrak kerja " + e.EmployeeNo + ".pdf", tinyPDF("Perjanjian Kerja " + e.FullName)},
		}
		for _, f := range files {
			key, ct, err := s.D.Files.Put(ctx, "documents/"+e.ID, f.data, []string{"application/pdf", "image/png"}, 10<<20)
			if err != nil {
				return err
			}
			doc := domain.Document{ID: ids.New(), EmployeeID: e.ID, DocType: f.typ, Name: f.name, StorageKey: key,
				ContentType: ct, SizeBytes: int64(len(f.data)), CreatedBy: s.actor}
			if err := s.D.Store.InsertDocument(ctx, doc); err != nil {
				return err
			}
		}
	}
	return nil
}

func idCardPNG() ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, 86, 54))
	for y := 0; y < 54; y++ {
		for x := 0; x < 86; x++ {
			c := color.RGBA{R: 206, G: 226, B: 245, A: 255}
			if x > 60 && x < 80 && y > 12 && y < 40 {
				c = color.RGBA{R: 120, G: 140, B: 170, A: 255}
			}
			img.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	err := png.Encode(&b, img)
	return b.Bytes(), err
}

// tinyPDF is a one-page PDF with a line of text.
func tinyPDF(text string) []byte {
	stream := fmt.Sprintf("BT /F1 18 Tf 72 720 Td (%s) Tj ET", text)
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs))
	for i, o := range objs {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	return b.Bytes()
}

// offboardingInProgress puts Taufik (business development) in his notice
// period.
func (s *Seeder) offboardingInProgress(ctx context.Context, emps []domain.Employee, _ map[string]string) error {
	e := emps[19]
	cur, err := s.D.Store.GetOffboarding(ctx, e.ID)
	if err != nil || cur != nil {
		return err
	}
	o := domain.Offboarding{ID: ids.New(), EmployeeID: e.ID, ResignationType: "voluntary", ResignationDate: hris.AddDaysISO(s.today, -9),
		LastWorkingDay: weekday(s.today, 21), Reason: "Melanjutkan usaha keluarga di Makassar"}
	if err := s.D.Store.InsertOffboarding(ctx, o, s.actor); err != nil {
		return err
	}
	o.Status, o.ClearanceHR, o.Notes = "notice_period", true, "Serah terima klien ke Rizky dan Salsabila"
	return s.D.Store.UpdateOffboarding(ctx, o, s.actor)
}

// requestStatuses adds leave and overtime in every status for the two demo
// logins (admin@mixist.id is Andika, tim@mixist.id is Gita) and requests from
// Andika's direct reports for his approval queue.
func (s *Seeder) requestStatuses(ctx context.Context, emps []domain.Employee, _ map[string]string) error {
	leaves := []struct {
		idx, from, days int
		typ, status     string
		reason          string
	}{
		{8, 12, 2, "annual", hris.LeavePending, "Menghadiri pernikahan sahabat di Malang"},
		{8, 30, 1, "annual", "rejected", "Libur tambahan setelah kampanye akhir tahun"},
		{8, 4, 1, "sick", "cancelled", "Kontrol gigi (dibatalkan, jadwal dokter pindah)"},
		{8, 18, 3, "annual", hris.LeaveApproved, "Liburan keluarga ke Lombok"},
		{1, 25, 2, "annual", hris.LeaveApproved, "Perjalanan dinas sekaligus cuti di Singapura"},
		{1, 7, 1, "annual", "cancelled", "Acara keluarga (dibatalkan)"},
		{1, 38, 2, "annual", hris.LeavePending, "Libur akhir tahun bersama keluarga"},
		{3, 9, 2, "annual", hris.LeavePending, "Menghadiri festival desain di Bali"},
		{17, 15, 1, "emergency", hris.LeavePending, "Mengantar orang tua operasi"},
		{11, 20, 2, "annual", "rejected", "Liburan saat tenggat audit SEO klien"},
	}
	hr := emps[22].ID
	for _, l := range leaves {
		e := emps[l.idx-1]
		if done, err := s.hasLeaveReason(ctx, e.ID, l.reason); err != nil || done {
			if err != nil {
				return err
			}
			continue
		}
		start := weekday(s.today, l.from)
		end := start
		for k := 1; k < l.days; k++ {
			end = weekday(end, 1)
		}
		if busy, err := s.D.Store.LeaveOverlaps(ctx, e.ID, start, end); err != nil || busy {
			if err != nil {
				return err
			}
			continue
		}
		n, _, err := s.workingDays(ctx, start, end)
		if err != nil {
			return err
		}
		requested := s.now.Add(-time.Duration(20+l.from*3) * time.Hour)
		row := domain.Leave{ID: ids.New(), EmployeeID: e.ID, LeaveType: l.typ, StartDate: start, EndDate: end, TotalDays: n,
			Reason: l.reason, Status: hris.LeavePending, CreatedBy: s.actor, CreatedAt: requested}
		if l.status == hris.LeaveApproved {
			at := requested.Add(6 * time.Hour)
			row.Status, row.ManagerApprovedBy, row.ManagerApprovedAt, row.ApprovedBy, row.ApprovedAt = l.status, e.ManagerID, &at, &hr, &at
		}
		if err := s.D.Store.InsertLeave(ctx, row); err != nil {
			return err
		}
		switch l.status {
		case "rejected":
			by := e.ManagerID
			if by == nil {
				by = &hr
			}
			if _, err := s.D.Store.SetLeaveStatus(ctx, domain.LeaveStatusUpdate{ID: row.ID, From: hris.LeavePending, To: "rejected",
				RejectedBy: by, RejectionReason: "Bentrok dengan tenggat klien, silakan ajukan tanggal lain", Actor: s.actor}); err != nil {
				return err
			}
		case "cancelled":
			if _, err := s.D.Store.SetLeaveStatus(ctx, domain.LeaveStatusUpdate{ID: row.ID, From: hris.LeavePending, To: "cancelled", Actor: s.actor}); err != nil {
				return err
			}
		case hris.LeaveApproved:
			u := domain.LeaveUsage{}
			if l.typ == "annual" {
				u.Annual = n
			} else {
				u.Other = n
			}
			if err := s.D.Store.AddLeaveUsage(ctx, e.ID, yearOf(start), u); err != nil {
				return err
			}
		}
	}

	overtime := []struct {
		idx, daysAgo int
		start, end   string
		status       string
		reason       string
	}{
		{8, 1, "18:00", "20:00", "pending", "Menyiapkan kalender konten November"},
		{8, 6, "18:00", "21:00", "rejected", "Riset tren konten akhir pekan"},
		{8, 9, "18:00", "19:30", "cancelled", "Rapat klien diundur"},
		{8, 13, "18:00", "20:30", "approved", "Laporan performa kampanye Oktober"},
		{1, 3, "18:00", "21:00", "approved", "Pitching klien enterprise"},
		{1, 8, "18:00", "20:00", "cancelled", "Makan malam klien dibatalkan"},
		{17, 2, "18:00", "20:30", "pending", "Finalisasi proposal tender"},
		{2, 1, "18:00", "20:00", "pending", "Review anggaran kuartal empat"},
	}
	for _, o := range overtime {
		e := emps[o.idx-1]
		existing, err := s.D.Store.ListOvertime(ctx, domain.OvertimeFilter{EmployeeIDs: []string{e.ID}})
		if err != nil {
			return err
		}
		if slices.ContainsFunc(existing, func(x domain.Overtime) bool { return x.Reason == o.reason }) {
			continue
		}
		date := weekday(hris.AddDaysISO(s.today, -o.daysAgo), 0)
		for slices.ContainsFunc(existing, func(x domain.Overtime) bool { return x.Date == date }) || date >= s.today {
			date = hris.AddDaysISO(date, -1)
			for hris.IsWeekend(date) {
				date = hris.AddDaysISO(date, -1)
			}
		}
		requested := wibMorning(date)
		row := domain.Overtime{ID: ids.New(), EmployeeID: e.ID, Date: date, StartTime: o.start, EndTime: o.end,
			Hours: hris.OvertimeHoursFromTimes(o.start, o.end), Source: "employee", Status: "pending", Reason: o.reason,
			RequestedBy: &e.ID, CreatedAt: requested}
		if err := s.D.Store.InsertOvertime(ctx, row, s.actor); err != nil {
			return err
		}
		if o.status != "pending" {
			by, rejection := e.ManagerID, ""
			if o.status == "cancelled" {
				by = &e.ID
			}
			if o.status == "rejected" {
				rejection = "Belum mendesak, bisa dikerjakan di jam kerja"
			}
			if _, err := s.D.Store.DecideOvertime(ctx, row.ID, o.status, by, rejection); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Seeder) hasLeaveReason(ctx context.Context, employeeID, reason string) (bool, error) {
	rows, _, err := s.D.Store.ListLeaves(ctx, domain.LeaveFilter{EmployeeIDs: []string{employeeID}})
	if err != nil {
		return false, err
	}
	return slices.ContainsFunc(rows, func(l domain.Leave) bool { return l.Reason == reason }), nil
}

// contractStatuses adds a draft renewal, a converted PKWT and marks the
// laid-off employee's contract terminated.
func (s *Seeder) contractStatuses(ctx context.Context, emps []domain.Employee, _ map[string]string) error {
	all, err := s.D.Store.ListContracts(ctx, domain.ContractFilter{})
	if err != nil {
		return err
	}
	has := func(employeeID, status string) bool {
		return slices.ContainsFunc(all, func(c domain.Contract) bool { return c.EmployeeID == employeeID && c.Status == status })
	}
	seq := len(all)
	insert := func(c domain.Contract) error {
		t, _ := hris.ParseDate(c.StartDate)
		for {
			seq++
			c.ContractNumber = hris.BuildContractNumber(c.ContractType, seq, t)
			err := s.D.Store.InsertContract(ctx, c)
			var u *domain.ErrUnique
			if err == nil || !errors.As(err, &u) {
				return err
			}
		}
	}
	// Dimas: the renewal of his PKWT, drafted and waiting for signature.
	dimas := emps[4]
	if !has(dimas.ID, "draft") {
		start := hris.AddDaysISO(s.today, 21)
		end := hris.AddDaysISO(start, 364)
		base := people[4].Base * 1.08
		c := domain.Contract{ID: ids.New(), EmployeeID: dimas.ID, ContractType: "pkwt", StartDate: start, EndDate: &end, Sequence: 2,
			PositionTitle: people[4].Title, DepartmentName: dimas.DepartmentName, WorkLocation: "Kantor Mixist, Jakarta Selatan",
			BaseSalary: &base, Notes: "Perpanjangan PKWT kedua, kenaikan gaji 8%", CreatedBy: s.actor}
		if err := insert(c); err != nil {
			return err
		}
	}
	// Oki: started on a PKWT that was converted to the current PKWTT.
	oki := emps[15]
	if !has(oki.ID, "converted") {
		end := hris.AddDaysISO(oki.JoinDate, 180)
		base := people[15].Base * 0.9
		c := domain.Contract{ID: ids.New(), EmployeeID: oki.ID, ContractType: "pkwt", StartDate: oki.JoinDate, EndDate: &end, Sequence: 1,
			PositionTitle: people[15].Title, DepartmentName: oki.DepartmentName, WorkLocation: "Kantor Mixist, Jakarta Selatan",
			BaseSalary: &base, Notes: "Dikonversi menjadi PKWTT setelah evaluasi enam bulan", CreatedBy: s.actor}
		if err := insert(c); err != nil {
			return err
		}
		if _, err := s.D.Store.SetContractStatus(ctx, domain.ContractStatusUpdate{ID: c.ID, From: "draft", To: "converted", SignedAt: &c.StartDate, Actor: s.actor}); err != nil {
			return err
		}
	}
	// Hendra was let go in the restructuring: terminated, with compensation.
	hendra := emps[33]
	for _, c := range all {
		if c.EmployeeID == hendra.ID && c.Status == "ended" {
			reason, pay := "Restrukturisasi tim Ads", people[33].Base*2
			if _, err := s.D.Store.SetContractStatus(ctx, domain.ContractStatusUpdate{ID: c.ID, From: "ended", To: "terminated",
				TerminatedReason: &reason, Compensation: &pay, Actor: s.actor}); err != nil {
				return err
			}
		}
	}
	return nil
}

// moreAnnouncements adds an expired and an archived announcement.
func (s *Seeder) moreAnnouncements(ctx context.Context, _ []domain.Employee, _ map[string]string) error {
	existing, err := s.D.Store.ListAnnouncements(ctx, "")
	if err != nil {
		return err
	}
	published, expired := s.now.AddDate(0, -2, 0), s.now.AddDate(0, -1, -10)
	items := []domain.Announcement{
		{Title: "Pendaftaran vaksin influenza kantor", Body: "Pendaftaran vaksin influenza gratis untuk karyawan dibuka sampai pertengahan September.",
			Tags: []string{"kesehatan"}, Status: "published", TargetScope: "global", PublishAt: &published, ExpiresAt: &expired},
		{Title: "Pindah sementara ke lantai 3", Body: "Selama renovasi pantry, tim Creative bekerja di lantai 3.",
			Tags: []string{"kantor"}, Status: "archived", TargetScope: "global", PublishAt: &published},
		{Title: "Jadwal family gathering", Body: "Family gathering Mixist diadakan Sabtu, 14 November di Puncak. Konfirmasi kehadiran ke HR.",
			Tags: []string{"acara"}, Status: "published", IsPinned: true, TargetScope: "global", PublishAt: &s.now},
	}
	for _, a := range items {
		if slices.ContainsFunc(existing, func(x domain.Announcement) bool { return x.Title == a.Title }) {
			continue
		}
		a.ID = ids.New()
		if _, err := s.D.Store.SaveAnnouncement(ctx, a, true, s.actor); err != nil {
			return err
		}
	}
	return nil
}

// loanStatuses adds a rejected loan, a loan already paid off, an active
// loan for Gita (tim@mixist.id) and a repaid kasbon for Andika.
func (s *Seeder) loanStatuses(ctx context.Context, emps []domain.Employee, _ map[string]string) error {
	hr := emps[22].ID
	type loanDef struct {
		idx       int
		typ       string
		principal float64
		tenor     int
		purpose   string
		status    string
		approved  time.Time
	}
	this, _ := hris.ParseDate(s.today[:8] + "01")
	defs := []loanDef{
		{22, "loan", 15_000_000, 12, "Uang muka kendaraan", "rejected", time.Time{}},
		{24, "kasbon", 2_400_000, 3, "Biaya pindah kos", "paid_off", this.AddDate(0, -8, 4)},
		{8, "loan", 6_000_000, 6, "Biaya kursus sertifikasi digital marketing", "approved", s.now.Add(-72 * time.Hour)},
		{1, "kasbon", 5_000_000, 2, "Kebutuhan mendesak keluarga", "paid_off", this.AddDate(0, -6, 2)},
	}
	for _, d := range defs {
		e := emps[d.idx-1]
		existing, err := s.D.Store.ListLoans(ctx, e.ID, "")
		if err != nil {
			return err
		}
		if slices.ContainsFunc(existing, func(l domain.Loan) bool { return l.Purpose == d.purpose }) {
			continue
		}
		_, monthly, total := payroll.LoanTerms(d.principal, 0, d.tenor)
		l := domain.Loan{ID: ids.New(), EmployeeID: e.ID, LoanType: d.typ, PrincipalAmount: d.principal, TenorMonths: d.tenor,
			MonthlyInstallment: monthly, RemainingBalance: total, Purpose: d.purpose, CreatedBy: s.actor}
		if err := s.D.Store.InsertLoan(ctx, l); err != nil {
			return err
		}
		switch d.status {
		case "rejected":
			if _, err := s.D.Store.RejectLoan(ctx, l.ID, &hr, "Melebihi batas 30% dari gaji bulanan"); err != nil {
				return err
			}
		case "approved", "paid_off":
			m, y := payroll.FirstInstallmentPeriod(d.approved)
			if _, err := s.D.Store.ApproveLoan(ctx, l.ID, &hr, d.approved, m, y); err != nil {
				return err
			}
			if d.status == "paid_off" {
				if err := s.D.Store.ApplyLoanPayment(ctx, l.ID, total, 0); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// additions are this month's bonuses and incentives.
func (s *Seeder) additions(ctx context.Context, emps []domain.Employee, _ map[string]string) error {
	t, _ := hris.ParseDate(s.today)
	month, year := int(t.Month()), t.Year()
	defs := []struct {
		idx    int
		amount float64
		label  string
	}{
		{18, 2_500_000, "Komisi penjualan"}, {19, 3_000_000, "Komisi penjualan"}, {15, 1_000_000, "Insentif kampanye terbaik"},
		{8, 750_000, "Insentif konten viral"},
	}
	for _, d := range defs {
		e := emps[d.idx-1]
		rows, err := s.D.Store.ListPayrollAdditions(ctx, month, year, e.ID)
		if err != nil {
			return err
		}
		if len(rows) > 0 {
			continue
		}
		if err := s.D.Store.InsertPayrollAddition(ctx, domain.PayrollAddition{ID: ids.New(), EmployeeID: e.ID, PeriodMonth: month,
			PeriodYear: year, Amount: d.amount, Label: d.label, CreatedBy: s.actor}); err != nil {
			return err
		}
	}
	return nil
}

// openRuns leaves this month's run approved and waiting for payment, next
// month's calculated and the month after as a draft.
func (s *Seeder) openRuns(ctx context.Context, _ []domain.Employee, _ map[string]string) error {
	this := s.today[:8] + "01"
	steps := []struct {
		month string
		upTo  string
	}{{this, payroll.RunApproved}, {addMonth(this, 1), payroll.RunCalculated}, {addMonth(this, 2), payroll.RunDraft}}
	d := s.D
	d.Clock = clock.Fixed{T: s.now}
	pay := app.NewPayroll(d, s.docs)
	for _, st := range steps {
		t, _ := hris.ParseDate(st.month)
		run, err := d.Store.GetPayrollRunByPeriod(ctx, int(t.Month()), t.Year())
		if err != nil {
			return err
		}
		if run != nil {
			continue
		}
		if run, err = pay.CreateRun(ctx, s.admin, int(t.Month()), t.Year(), ""); err != nil {
			return err
		}
		if st.upTo == payroll.RunDraft {
			continue
		}
		if _, _, err := pay.Calculate(ctx, s.admin, run.ID, false); err != nil {
			return err
		}
		if st.upTo == payroll.RunApproved {
			if _, err := pay.Approve(ctx, s.admin, run.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// todayAttendance clocks in most of the team this morning (no clock-out
// yet) so "who is in" views have data. The two demo logins are left out so
// they can try clocking in themselves.
func (s *Seeder) todayAttendance(ctx context.Context, emps []domain.Employee, _ map[string]string) error {
	rows, err := s.D.Store.ListScheduleRows(ctx, idsOf(emps))
	if err != nil {
		return err
	}
	holidays, err := s.D.Store.ListHolidays(ctx, s.today, s.today, true)
	if err != nil || len(holidays) > 0 {
		return err
	}
	onLeave, err := s.D.Store.ApprovedLeavesOn(ctx, s.today)
	if err != nil {
		return err
	}
	for i, e := range emps {
		email := people[i].Email
		if !e.IsActive || email == "admin@mixist.id" || email == "tim@mixist.id" || onLeave[e.ID] != "" {
			continue
		}
		var mine []domain.ScheduleRow
		for _, r := range rows {
			if r.EmployeeID == e.ID {
				mine = append(mine, r)
			}
		}
		sh := app.ShiftOn(mine, s.today)
		h := jitter(e.EmployeeNo + s.today)
		if sh == nil || h%10 >= 8 {
			continue
		}
		times := hris.ShiftTimes{StartTime: sh.StartTime, EndTime: sh.EndTime, LateToleranceMinutes: sh.LateToleranceMinutes}
		start, end := hris.ScheduledWindow(s.today, times)
		in := start.Add(time.Duration(int(h%41)-15+lateBias(s.today)) * time.Minute)
		if in.After(s.now) {
			continue
		}
		late, mins := hris.Lateness(in, s.today, times)
		a := domain.Attendance{ID: ids.New(), EmployeeID: e.ID, Date: s.today, ShiftID: &sh.ID, ScheduledStart: &start, ScheduledEnd: &end,
			Status: "present", ClockIn: &in, IsLate: late, LateMinutes: mins}
		if late {
			a.Status = "late"
		}
		if err := s.D.Store.InsertAttendanceRecord(ctx, a, s.actor); err != nil {
			return err
		}
	}
	return nil
}
