// Package pdf renders the candidate pipeline report (pipeline-report-pdf.ts)
// and the offer letter as A4 PDFs with the built-in Helvetica faces.
package pdf

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
)

const margin = 17.0

// Renderer implements app.PDFRenderer.
type Renderer struct{}

// safe maps the symbols Helvetica cannot print and drops the rest outside
// Latin-1 (sanitizeText).
func safe(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '→':
			b.WriteString("->")
		case '←':
			b.WriteString("<-")
		case '’', '‘', '‚':
			b.WriteByte('\'')
		case '“', '”', '„':
			b.WriteByte('"')
		case '–', '—', '−':
			b.WriteByte('-')
		case '•':
			b.WriteByte('*')
		case '…':
			b.WriteString("...")
		default:
			if r <= 0xFF {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// doc wraps fpdf with the report's text helpers.
type doc struct {
	*fpdf.Fpdf
	tr func(string) string
}

func newDoc(footer func() string) *doc {
	f := fpdf.New("P", "mm", "A4", "")
	d := &doc{Fpdf: f, tr: f.UnicodeTranslatorFromDescriptor("")}
	f.SetMargins(margin, margin, margin)
	f.SetAutoPageBreak(true, margin+4)
	f.AliasNbPages("{nb}")
	f.SetFooterFunc(func() {
		f.SetY(-13)
		f.SetFont("Helvetica", "", 8)
		f.SetTextColor(107, 114, 128)
		f.CellFormat(0, 5, d.text(footer()+" · Halaman "+strconv.Itoa(f.PageNo())+" dari {nb}"), "", 0, "C", false, 0, "")
	})
	f.AddPage()
	return d
}

func (d *doc) text(s string) string { return d.tr(safe(s)) }

func (d *doc) title(s, sub string) {
	d.SetFont("Helvetica", "B", 16)
	d.SetTextColor(31, 41, 55)
	d.MultiCell(0, 8, d.text(s), "", "L", false)
	if sub != "" {
		d.SetFont("Helvetica", "", 10)
		d.SetTextColor(107, 114, 128)
		d.MultiCell(0, 5, d.text(sub), "", "L", false)
	}
	d.Ln(2)
}

func (d *doc) section(s string) {
	d.Ln(3)
	if d.GetY() > 250 {
		d.AddPage()
	}
	d.SetFont("Helvetica", "B", 12)
	d.SetTextColor(3, 105, 161)
	d.CellFormat(0, 7, d.text(s), "", 1, "L", false, 0, "")
	d.SetDrawColor(229, 231, 235)
	d.Line(margin, d.GetY(), 210-margin, d.GetY())
	d.Ln(2)
}

func (d *doc) heading(s string) {
	d.SetFont("Helvetica", "B", 10)
	d.SetTextColor(31, 41, 55)
	d.MultiCell(0, 5.5, d.text(s), "", "L", false)
}

func (d *doc) keyValue(label, value string) {
	if strings.TrimSpace(value) == "" {
		value = "-"
	}
	d.SetFont("Helvetica", "", 9.5)
	d.SetTextColor(107, 114, 128)
	d.CellFormat(50, 5.5, d.text(label), "", 0, "L", false, 0, "")
	d.SetTextColor(31, 41, 55)
	d.MultiCell(0, 5.5, d.text(value), "", "L", false)
}

func (d *doc) paragraph(s string) {
	d.SetFont("Helvetica", "", 9.5)
	d.SetTextColor(31, 41, 55)
	d.MultiCell(0, 5, d.text(s), "", "L", false)
}

func (d *doc) bullets(items []string) {
	for _, it := range items {
		d.paragraph("* " + it)
	}
}

func (d *doc) empty(s string) {
	d.SetFont("Helvetica", "I", 9.5)
	d.SetTextColor(107, 114, 128)
	d.MultiCell(0, 5.5, d.text(s), "", "L", false)
}

func (d *doc) bytes() ([]byte, error) {
	if d.Err() {
		return nil, d.Error()
	}
	var buf bytes.Buffer
	if err := d.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ─── formatting ──────────────────────────────────────────────

func dateOf(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return domain.FormatDate(*t)
}

func dateTimeOf(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return domain.FormatDateTime(*t)
}

func day(s *string) string {
	if s == nil {
		return "-"
	}
	t, ok := domain.ParseDay(*s)
	if !ok {
		return *s
	}
	return domain.FormatDate(t.Add(12 * time.Hour))
}

func rupiah(n *int64) string {
	if n == nil {
		return "-"
	}
	return domain.Rupiah(*n).FormatIDR()
}

func boolLabel(b *bool) string {
	switch {
	case b == nil:
		return "-"
	case *b:
		return "Ya"
	}
	return "Tidak"
}

func score(v *float64) string {
	if v == nil {
		return "-"
	}
	return strconv.FormatFloat(float64(int(*v+0.5)), 'f', -1, 64) + "/100"
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// ─── pipeline report ─────────────────────────────────────────

// PipelineReport is buildPipelineReportPdf: profile, AI analysis, HR
// screening, psikotes, AI interview, scheduled interviews, offers, the
// hire event and the activity history.
func (Renderer) PipelineReport(r domain.PipelineReport, now time.Time) ([]byte, error) {
	c := r.Candidate
	d := newDoc(func() string { return r.CompanyName + " Rekrutmen · " + c.FullName })
	position := c.JobOpeningTitle
	if position == "" {
		position = "Posisi tidak tercatat"
	}
	status, _ := domain.StatusLabel(c.Status)
	d.title("Laporan Pipeline Rekrutmen", c.FullName+" - "+position+"\nStatus saat ini: "+status+" · Dicetak "+domain.FormatDateTime(now))

	d.section("1. Profil Kandidat")
	d.keyValue("Nama lengkap", c.FullName)
	d.keyValue("Email", c.Email)
	d.keyValue("Telepon", c.Phone)
	d.keyValue("Domisili", c.Domicile)
	d.keyValue("Sumber lamaran", domain.SourceLabel(c.Source))
	d.keyValue("Pendidikan terakhir", c.LastEducation)
	d.keyValue("Pengalaman terakhir", c.LastExperience)
	d.keyValue("Ekspektasi gaji", rupiah(c.ExpectedSalary))
	d.keyValue("Tanggal melamar", domain.FormatDate(c.CreatedAt))

	d.section("2. Analisis Profil (AI)")
	if a := r.Analysis; a != nil {
		d.keyValue("Skor kecocokan", strconv.Itoa(a.MatchScore)+"/100")
		d.keyValue("Alasan", a.MatchReason)
		d.keyValue("Ringkasan profil", a.Summary)
		d.keyValue("Model / waktu analisis", a.Model+" · "+domain.FormatDateTime(a.UpdatedAt))
	} else {
		d.empty("Belum ada analisis AI untuk kandidat ini.")
	}

	d.section("3. Screening HR")
	if s := r.Screening; s != nil {
		d.keyValue("Sudah dihubungi", boolLabel(&s.Contacted))
		d.keyValue("Berminat", boolLabel(s.Interested))
		d.keyValue("Ketersediaan", s.AvailabilityNote)
		d.keyValue("Gaji terkonfirmasi", rupiah(s.ConfirmedSalary))
		d.keyValue("Bersedia shift", boolLabel(s.WillingShift))
		d.keyValue("Bersedia penempatan", boolLabel(s.WillingPlacement))
		if s.Notes != "" {
			d.keyValue("Catatan", s.Notes)
		}
		d.keyValue("Rekomendasi", domain.RecommendationLabel(s.Recommendation))
		d.keyValue("Diperbarui oleh", orDash(s.UpdatedByName)+" · "+domain.FormatDateTime(s.UpdatedAt))
	} else {
		d.empty("Tahap screening belum diisi.")
	}

	d.section("4. Psikotes")
	if len(r.PsikotesTests) == 0 {
		d.empty("Belum ada sesi psikotes.")
	}
	for _, t := range r.PsikotesTests {
		d.heading(t.InstrumentName)
		d.keyValue("Status", domain.TestStatusLabel(t.Status))
		d.keyValue("Skor", score(t.Score))
		if t.ScoreDetail != nil && len(t.ScoreDetail.Dominant) > 0 {
			var dom []string
			for _, x := range t.ScoreDetail.Dominant {
				dom = append(dom, fmt.Sprintf("%s (%s, %d)", x.Label, x.Code, x.Count))
			}
			d.keyValue("Skala dominan", strings.Join(dom, ", "))
		}
		if t.ReviewNotes != "" {
			d.keyValue("Catatan review", t.ReviewNotes)
		}
		if t.ReviewedByName != "" {
			d.keyValue("Direview oleh", t.ReviewedByName)
		}
		d.Ln(1)
	}
	if s := r.PsikotesSummary; s != nil {
		d.heading("Kesimpulan Psikotes")
		d.keyValue("Rekomendasi", domain.RecommendationLabel(s.Recommendation))
		if s.Notes != "" {
			d.keyValue("Catatan", s.Notes)
		}
		d.keyValue("Diputuskan oleh", orDash(s.UpdatedByName)+" · "+domain.FormatDateTime(s.UpdatedAt))
	}

	d.section("5. Interview AI")
	if len(r.AiInterviews) == 0 {
		d.empty("Belum ada sesi interview AI.")
	}
	for i, s := range r.AiInterviews {
		if len(r.AiInterviews) > 1 {
			d.heading("Sesi " + strconv.Itoa(i+1))
		}
		answered := 0
		for _, t := range s.Turns {
			if t.AnsweredAt != nil {
				answered++
			}
		}
		d.keyValue("Status sesi", s.Status)
		d.keyValue("Selesai pada", dateTimeOf(s.CompletedAt))
		d.keyValue("Pertanyaan terjawab", strconv.Itoa(answered))
		sum := s.Summary
		if sum == nil {
			d.empty("Sesi belum memiliki kesimpulan AI.")
			continue
		}
		d.keyValue("Ringkasan", sum.Summary)
		d.keyValue("Relevansi", strconv.Itoa(sum.Relevance.Score)+"/100 - "+domain.RelevanceLabel(sum.Relevance.Conclusion))
		d.keyValue("Alasan", sum.Relevance.Reason)
		if len(sum.Skills) > 0 {
			d.keyValue("Keahlian", strings.Join(sum.Skills, ", "))
		}
		if sum.Salary.Mentioned {
			v := sum.Salary.Note
			if sum.Salary.Value != nil {
				v = *sum.Salary.Value
			}
			d.keyValue("Ekspektasi gaji (interview)", v)
		}
		if len(sum.RedFlags) > 0 {
			d.heading("Red flags:")
			d.bullets(sum.RedFlags)
		}
		d.Ln(1)
	}

	d.section("6. Interview dengan Tim")
	if len(r.Interviews) == 0 {
		d.empty("Belum ada interview terjadwal.")
	}
	for _, iv := range r.Interviews {
		d.heading("Interview " + domain.InterviewTypeLabel(iv.Type) + " · " + domain.FormatDateTime(iv.InterviewDate))
		d.keyValue("Interviewer", iv.InterviewerName)
		d.keyValue("Status", iv.Status)
		if iv.OverallScore != nil {
			d.keyValue("Nilai rata-rata", strconv.FormatFloat(*iv.OverallScore, 'f', 2, 64)+" / 5")
		}
		for _, it := range iv.Scorecard {
			line := strconv.Itoa(it.Score) + "/5"
			if it.Note != "" {
				line += " - " + it.Note
			}
			d.keyValue("  "+it.Criterion, line)
		}
		if iv.Recommendation != "" {
			d.keyValue("Rekomendasi", domain.InterviewRecommendationLabel(iv.Recommendation))
		}
		d.Ln(1)
	}

	d.section("7. Offer")
	if len(r.Offers) == 0 {
		d.empty("Belum ada offer yang dibuat.")
	}
	for _, o := range r.Offers {
		d.heading("Offer v" + strconv.Itoa(o.Version) + " - " + domain.OfferStatusLabel(o.Status))
		d.keyValue("Posisi", o.PositionTitle)
		d.keyValue("Gaji pokok", o.BaseSalary.FormatIDR())
		d.keyValue("Mulai kerja", day(o.StartDate))
		d.keyValue("Dikirim", dateTimeOf(o.SentAt))
		d.keyValue("Berlaku sampai", dateTimeOf(o.ExpiresAt))
		if o.RespondedAt != nil {
			d.keyValue("Direspons", dateTimeOf(o.RespondedAt))
		}
		if o.ResponseNote != "" {
			d.keyValue("Catatan respons", o.ResponseNote)
		}
		d.keyValue("Dibuat oleh", o.CreatedByName)
		d.Ln(1)
	}

	d.section("8. Hired")
	switch {
	case c.Status == "hired" && c.HiredEventAt != nil:
		d.keyValue("Tanggal diterima", dateOf(c.HiredAt))
		d.keyValue("Data karyawan", "Diteruskan ke HR pada "+domain.FormatDateTime(*c.HiredEventAt))
	case c.Status == "hired":
		d.keyValue("Tanggal diterima", dateOf(c.HiredAt))
		d.empty("Kandidat sudah Hired, data karyawan belum diteruskan ke HR.")
	default:
		d.empty("Kandidat belum berstatus Hired.")
	}

	d.section("9. Riwayat Aktivitas Pipeline")
	if len(r.Activities) == 0 {
		d.empty("Belum ada aktivitas tercatat.")
	}
	for _, a := range r.Activities {
		line := domain.FormatDateTime(a.CreatedAt)
		if a.CreatedByName != "" {
			line += " · " + a.CreatedByName
		}
		d.SetFont("Helvetica", "", 8.5)
		d.SetTextColor(107, 114, 128)
		d.MultiCell(0, 4.5, d.text(line), "", "L", false)
		d.paragraph(a.Description)
		d.Ln(0.8)
	}
	return d.bytes()
}

// ─── offer letter ────────────────────────────────────────────

var romanMonths = []string{"I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII"}

// LetterNumber is "OL-0003/MIXIST/X/2026": the offer version within the
// candidate, the company and the Roman month of issue.
func LetterNumber(company string, version int, issued time.Time) string {
	t := issued.In(domain.Jakarta)
	return fmt.Sprintf("OL-%04d/%s/%s/%d", version, strings.ToUpper(domain.Slugify(company)), romanMonths[t.Month()-1], t.Year())
}

// OfferLetter renders the offer the candidate can download and sign.
func (Renderer) OfferLetter(l domain.OfferLetter) ([]byte, error) {
	o := l.Offer
	d := newDoc(func() string { return l.CompanyName + " · Surat Penawaran Kerja v" + strconv.Itoa(o.Version) })
	d.SetFont("Helvetica", "B", 20)
	d.SetTextColor(17, 17, 17)
	d.CellFormat(0, 10, d.text(l.CompanyName), "", 1, "L", false, 0, "")
	d.SetFont("Helvetica", "", 9)
	d.SetTextColor(107, 114, 128)
	d.CellFormat(0, 5, d.text("Digital Agency · Jakarta"), "", 1, "L", false, 0, "")
	d.Ln(6)

	d.SetFont("Helvetica", "B", 14)
	d.SetTextColor(31, 41, 55)
	d.CellFormat(0, 8, d.text("Surat Penawaran Kerja"), "", 1, "L", false, 0, "")
	d.keyValue("Nomor", LetterNumber(l.CompanyName, o.Version, l.IssuedAt))
	d.keyValue("Tanggal", domain.FormatDate(l.IssuedAt))
	d.Ln(4)

	d.paragraph("Kepada Yth.\n" + o.CandidateName)
	d.Ln(3)
	position := orDash(o.PositionTitle)
	d.paragraph(fmt.Sprintf("Dengan hormat, terima kasih atas waktu dan minat Anda selama proses rekrutmen. Kami dengan senang hati "+
		"menawarkan posisi %s di %s dengan ketentuan sebagai berikut:", position, l.CompanyName))
	d.Ln(3)
	d.keyValue("Posisi", position)
	if l.Department != "" {
		d.keyValue("Departemen", l.Department)
	}
	if l.Location != "" {
		d.keyValue("Lokasi kerja", l.Location)
	}
	d.keyValue("Gaji pokok", o.BaseSalary.FormatIDR()+" per bulan")
	d.keyValue("Tanggal mulai", day(o.StartDate))
	if len(o.Benefits) > 0 {
		d.keyValue("Benefit", "")
		for _, b := range o.Benefits {
			d.CellFormat(50, 5, "", "", 0, "L", false, 0, "")
			d.MultiCell(0, 5, d.text("* "+b), "", "L", false)
		}
	}
	d.Ln(3)
	d.paragraph("Penawaran ini berlaku sampai " + dateTimeOf(o.ExpiresAt) + ". Silakan konfirmasi melalui tautan penawaran yang " +
		"kami kirimkan atau hubungi tim HR. Ketentuan lain mengikuti perjanjian kerja dan peraturan perusahaan yang berlaku.")
	d.Ln(3)
	d.paragraph("Kami menantikan kehadiran Anda di tim " + l.CompanyName + ".")
	d.Ln(12)
	d.paragraph("Hormat kami,\n\n\n\n" + orDash(o.CreatedByName) + "\nHuman Resources, " + l.CompanyName)
	if o.Status == "accepted" && o.RespondedAt != nil {
		d.Ln(6)
		d.SetFont("Helvetica", "B", 10)
		d.SetTextColor(21, 128, 61)
		d.MultiCell(0, 5.5, d.text("Diterima oleh kandidat pada "+domain.FormatDateTime(*o.RespondedAt)), "", "L", false)
	}
	return d.bytes()
}
