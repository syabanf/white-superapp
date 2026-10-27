package domain

import (
	"bytes"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"
)

// Reimbursement claims (new for Mixist; NüHabit had none).
//
//	DRAFT -> SUBMITTED -> APPROVED_MANAGER -> APPROVED_FINANCE -> PAID
//	SUBMITTED | APPROVED_MANAGER -> REJECTED (with a reason)
//	DRAFT | SUBMITTED -> CANCELLED (by the owner, before any approval)

type ClaimStatus string

const (
	ClaimDraft           ClaimStatus = "DRAFT"
	ClaimSubmitted       ClaimStatus = "SUBMITTED"
	ClaimApprovedManager ClaimStatus = "APPROVED_MANAGER"
	ClaimApprovedFinance ClaimStatus = "APPROVED_FINANCE"
	ClaimPaid            ClaimStatus = "PAID"
	ClaimRejected        ClaimStatus = "REJECTED"
	ClaimCancelled       ClaimStatus = "CANCELLED"
)

// Claim categories.
const (
	CategoryTransport   = "TRANSPORT"
	CategoryMakan       = "MAKAN"
	CategoryAkomodasi   = "AKOMODASI"
	CategoryOperasional = "OPERASIONAL_PROYEK"
	CategoryLainnya     = "LAINNYA"
)

// ClaimCategories in display order with their labels.
var ClaimCategories = []struct{ Code, Label string }{
	{CategoryTransport, "Transport"},
	{CategoryMakan, "Makan"},
	{CategoryAkomodasi, "Akomodasi"},
	{CategoryOperasional, "Operasional proyek"},
	{CategoryLainnya, "Lainnya"},
}

// CategoryLabel is the display label, the code when unknown.
func CategoryLabel(code string) string {
	for _, c := range ClaimCategories {
		if c.Code == code {
			return c.Label
		}
	}
	return code
}

func IsClaimCategory(code string) bool {
	for _, c := range ClaimCategories {
		if c.Code == code {
			return true
		}
	}
	return false
}

// Rules and messages.
const (
	ClaimMaxAgeDays     = 30
	MaxReceiptBytes     = 10 * 1024 * 1024
	ReminderAfter       = 48 * time.Hour
	MsgClaimTooOld      = "Klaim melebihi 30 hari sejak tanggal transaksi."
	MsgClaimNotFound    = "Klaim reimbursement tidak ditemukan."
	MsgClaimNotEditable = "Klaim hanya bisa diubah saat masih draft."
	MsgClaimTransition  = "Aksi tidak diizinkan untuk status klaim saat ini."
	MsgNoReceipt        = "Lampirkan minimal satu bukti transaksi."
	MsgReceiptTooLarge  = "Ukuran file melebihi 10 MB."
	MsgReceiptType      = "Bukti harus berupa gambar (JPG, PNG, WEBP) atau PDF."
	MsgRejectReason     = "Alasan penolakan wajib diisi."
	MsgNotApprover      = "Anda bukan penyetuju klaim ini."
	MsgOverLimitApprove = "Klaim melebihi batas kebijakan dan hanya bisa disetujui pemegang izin finance.approve."
	MsgSelfApprove      = "Anda tidak bisa menyetujui klaim sendiri."
	MsgNoEmployee       = "Akun Anda belum terhubung ke data karyawan. Hubungi HR."
	MsgFutureDate       = "Tanggal transaksi tidak boleh di masa depan."
	MsgPayNothing       = "Pilih minimal satu klaim yang sudah disetujui finance."
	MsgPayNotApproved   = "Hanya klaim berstatus APPROVED_FINANCE yang bisa dibayar."
)

// CanCancel: the owner may cancel before any approval.
func CanCancel(s ClaimStatus) bool { return s == ClaimDraft || s == ClaimSubmitted }

// NextApproval is the status an approval moves to, "" when none applies.
func NextApproval(s ClaimStatus) ClaimStatus {
	switch s {
	case ClaimSubmitted:
		return ClaimApprovedManager
	case ClaimApprovedManager:
		return ClaimApprovedFinance
	}
	return ""
}

// CanReject: any approval step may reject.
func CanReject(s ClaimStatus) bool { return s == ClaimSubmitted || s == ClaimApprovedManager }

// CheckClaimAge rejects transactions older than 30 days (Jakarta calendar)
// and dates in the future.
func CheckClaimAge(transactionDate, today time.Time) error {
	if transactionDate.After(today) {
		return Bad(MsgFutureDate)
	}
	if today.Sub(transactionDate) > ClaimMaxAgeDays*24*time.Hour {
		return Bad(MsgClaimTooOld)
	}
	return nil
}

// Policy is the configurable limit of one category. Zero limits mean none.
type Policy struct {
	Category         string
	Label            string
	MonthlyLimit     Money
	PerClaimLimit    Money
	ExpenseAccountID string
	// AutoApprove lets claims within the limits skip the finance step.
	AutoApprove bool
	IsActive    bool
}

// EvaluateLimit reports whether a claim exceeds the policy, given what the
// employee already claimed in the same category and month (submitted and
// later, excluding this claim). The note explains which limit.
func EvaluateLimit(p *Policy, amount, monthToDate Money) (overLimit bool, note string) {
	if p == nil || !p.IsActive {
		return false, ""
	}
	var notes []string
	if p.PerClaimLimit > 0 && amount > p.PerClaimLimit {
		notes = append(notes, "melebihi batas per klaim Rp"+FormatRupiah(p.PerClaimLimit))
	}
	if p.MonthlyLimit > 0 && monthToDate+amount > p.MonthlyLimit {
		notes = append(notes, "total bulan ini Rp"+FormatRupiah(monthToDate+amount)+" melebihi batas bulanan Rp"+FormatRupiah(p.MonthlyLimit))
	}
	if len(notes) == 0 {
		return false, ""
	}
	return true, "Klaim " + strings.Join(notes, "; ") + "."
}

// FormatRupiah is "1.250.000" (dot thousands, decimals only when present).
func FormatRupiah(m Money) string {
	v := int64(m)
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	whole := fmt.Sprintf("%d", v/100)
	var b strings.Builder
	for i, r := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	if cents := v % 100; cents != 0 {
		return fmt.Sprintf("%s%s,%02d", sign, b.String(), cents)
	}
	return sign + b.String()
}

// ─── Receipt files ──────────────────────────────────────────

var receiptTypes = []struct{ mime, ext string }{
	{"image/jpeg", "jpg"},
	{"image/png", "png"},
	{"image/webp", "webp"},
	{"application/pdf", "pdf"},
}

// SniffReceipt returns the content type and extension from the magic
// bytes, "" for anything that is not an image or PDF.
func SniffReceipt(b []byte) (string, string) {
	var mime string
	switch {
	case len(b) >= 3 && b[0] == 0xff && b[1] == 0xd8 && b[2] == 0xff:
		mime = "image/jpeg"
	case len(b) >= 8 && bytes.Equal(b[:8], []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}):
		mime = "image/png"
	case len(b) >= 12 && string(b[:4]) == "RIFF" && string(b[8:12]) == "WEBP":
		mime = "image/webp"
	case len(b) >= 5 && string(b[:5]) == "%PDF-":
		mime = "application/pdf"
	}
	for _, t := range receiptTypes {
		if t.mime == mime {
			return t.mime, t.ext
		}
	}
	return "", ""
}

var safeKeyRe = regexp.MustCompile(`^[A-Za-z0-9_-]+/[A-Za-z0-9_.-]+$`)

// IsSafeKey allows only "<segment>/<segment>" in a whitelisted charset.
func IsSafeKey(key string) bool { return safeKeyRe.MatchString(key) && !strings.Contains(key, "..") }

// CleanFilename keeps the base name, at most 200 characters.
func CleanFilename(name string) string {
	name = path.Base(strings.ReplaceAll(strings.TrimSpace(name), "\\", "/"))
	if name == "." || name == "/" || name == "" {
		name = "bukti"
	}
	if r := []rune(name); len(r) > 200 {
		name = string(r[:200])
	}
	return name
}
