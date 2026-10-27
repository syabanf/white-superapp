package docs

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/app"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/payroll"
)

func TestAttendanceXLSX(t *testing.T) {
	in := time.Date(2026, 10, 5, 1, 58, 0, 0, time.UTC)
	hours, status, nip := 8.5, "late", "EMP-2026-00001"
	data, err := Renderer{CompanyName: "Mixist"}.AttendanceXLSX([]hris.ReportRow{{Date: "2026-10-05", EmployeeName: "Ayu & Co <QA>",
		NIP: &nip, ClockIn: &in, WorkHours: &hours, Status: &status, IsLate: true, LateMinutes: 48}},
		app.AttendanceReportMeta{CompanyName: "Mixist", PeriodLabel: "1–31 Oktober 2026", GeneratedAt: in})
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var sheet string
	for _, f := range zr.File {
		if f.Name == "xl/worksheets/sheet1.xml" {
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			sheet = string(b)
		}
	}
	for _, want := range []string{"Mixist: Rekap Absensi", "Periode: 1–31 Oktober 2026", "Ayu &amp; Co &lt;QA&gt;", "Sen, 05 Okt 2026",
		"08.58", "<v>8.5</v>", "Terlambat", "<v>48</v>", `<mergeCell ref="A1:K1"/>`} {
		if !strings.Contains(sheet, want) {
			t.Errorf("sheet lacks %q", want)
		}
	}
}

func TestPayslipPDF(t *testing.T) {
	r := payroll.Calculate(payroll.Input{BaseSalary: 10_000_000, FixedAllowance: 2_000_000, WorkingDays: 22, PresentDays: 22,
		PTKPStatus: "TK/0", IsTaxable: true, BpjsTkEnrolled: true, BpjsKesEnrolled: true, TaperaEnrolled: true, LoanDeduction: 500_000}, payroll.DefaultConfig())
	tenor := 3
	kasbon := "kasbon"
	slip := domain.Payslip{EmployeeName: "Ayu Lestari", EmployeeNo: "EMP-2026-00001", PeriodMonth: 10, PeriodYear: 2026,
		BaseSalary: r.BaseSalary, FixedAllowance: r.FixedAllowance, GrossSalary: r.GrossSalary, Pph21Deduction: r.Pph21Deduction,
		LoanDeduction: r.LoanDeduction, TotalDeductions: r.TotalDeductions, NetSalary: r.NetSalary, ProrateFactor: 1}
	data, err := Renderer{CompanyName: "Mixist"}.PayslipPDF(slip, []payroll.LoanInstallmentDetail{{LoanType: &kasbon, InstallmentNo: 1, TenorMonths: &tenor, Amount: 500_000}})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) || len(data) < 1000 {
		t.Fatalf("not a pdf (%d bytes)", len(data))
	}
}
