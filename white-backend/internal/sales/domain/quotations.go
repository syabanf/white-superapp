package domain

import (
	"math"
	"time"
)

// Quotation statuses (QUOTATION_STATUSES plus the superseded state that a
// revision leaves behind).
const (
	QuoteDraft      = "draft"
	QuoteSent       = "terkirim"
	QuoteAccepted   = "diterima"
	QuoteRejected   = "ditolak"
	QuoteSuperseded = "superseded"
)

// Approval states of a quotation.
const (
	ApprovalNone     = "none"
	ApprovalPending  = "pending"
	ApprovalApproved = "approved"
	ApprovalRejected = "rejected"
)

// MaxLineTotal bounds a quotation line (numeric(18,2) leaves room, NüHabit
// capped at numeric(14,2)).
const MaxLineTotal Money = 999_999_999_999_00

// QuotationLine is one item of a quotation payload.
type QuotationLine struct {
	ItemType    string // katalog | bebas
	CatalogID   *string
	Description string
	Scope       string
	Unit        string
	Qty         float64
	UnitPrice   Money
	LineTotal   Money
}

// QuotationTermInput is one payment term (termin) of a payload.
type QuotationTermInput struct {
	Label   string
	Percent float64
	DueDate *time.Time
}

// QuotationPayload is quotationPayloadSchema's output.
type QuotationPayload struct {
	UsePpn          bool
	PpnPercent      float64
	DiscountPercent float64
	Notes           *string
	ValidUntil      *time.Time
	Items           []QuotationLine
	Terms           []QuotationTermInput
}

// Totals is computeTotals' result.
type Totals struct {
	Subtotal, DiscountNominal, PpnNominal, Total Money
}

// ComputeTotals is computeTotals: line totals to the sen, the discount
// from the subtotal, PPN from the base after discount. Line totals are
// written back into Items.
func ComputeTotals(p *QuotationPayload) Totals {
	var subtotal Money
	for i := range p.Items {
		it := &p.Items[i]
		it.LineTotal = it.UnitPrice.Times(it.Qty)
		subtotal += it.LineTotal
	}
	pct := math.Min(100, math.Max(0, p.DiscountPercent))
	discount := subtotal.Percent(pct)
	dpp := subtotal - discount
	var ppn Money
	if p.UsePpn {
		ppn = dpp.Percent(p.PpnPercent)
	}
	return Totals{Subtotal: subtotal, DiscountNominal: discount, PpnNominal: ppn, Total: dpp + ppn}
}

// TermsSumTo100 is the terms refine: no terms, or the percents add up to
// 100 within 0.01.
func TermsSumTo100(terms []QuotationTermInput) bool {
	if len(terms) == 0 {
		return true
	}
	sum := 0.0
	for _, t := range terms {
		sum += t.Percent
	}
	return math.Abs(sum-100) <= 0.01
}

// ValidateQuotation applies quotationPayloadSchema's rules and returns the
// first violation per field (keys are the old zod field names).
func ValidateQuotation(p QuotationPayload) map[string]string {
	errs := map[string]string{}
	if len(p.Items) == 0 {
		errs["items"] = "Minimal 1 item"
	}
	for _, it := range p.Items {
		switch {
		case JSTrim(it.Description) == "":
			errs["items"] = "Deskripsi item wajib diisi"
		case it.Qty <= 0:
			errs["items"] = "Qty harus lebih dari 0"
		case it.UnitPrice < 0:
			errs["items"] = "Harga tidak boleh negatif"
		case it.UnitPrice.Times(it.Qty) > MaxLineTotal:
			errs["items"] = "Total baris terlalu besar"
		case it.ItemType == "katalog" && (it.CatalogID == nil || *it.CatalogID == ""):
			errs["items"] = "Item katalog wajib memilih layanan"
		}
	}
	if p.DiscountPercent < 0 || p.DiscountPercent > 100 {
		errs["discountPercent"] = "Diskon harus 0-100%"
	}
	if p.PpnPercent < 0 || p.PpnPercent > 100 {
		errs["ppnPercent"] = "PPN harus 0-100%"
	}
	for _, t := range p.Terms {
		if JSTrim(t.Label) == "" {
			errs["terms"] = "Label termin wajib diisi"
		} else if t.Percent <= 0 || t.Percent > 100 {
			errs["terms"] = "Persentase termin harus 0-100"
		}
	}
	if _, bad := errs["terms"]; !bad && !TermsSumTo100(p.Terms) {
		errs["terms"] = "Total persentase termin harus 100%"
	}
	return errs
}

// AllocateTermAmounts is allocateTermAmounts: cumulative rounding so the
// amounts always add up to the total.
func AllocateTermAmounts(total Money, percents []float64) []Money {
	amounts := make([]Money, 0, len(percents))
	var assigned Money
	cum := 0.0
	for i, p := range percents {
		cum += p
		target := total
		if i < len(percents)-1 {
			target = total.Percent(cum)
		}
		amounts = append(amounts, target-assigned)
		assigned = target
	}
	return amounts
}

// TermProgress is one term's payment progress.
type TermProgress struct {
	TermID  string
	Label   string
	DueDate *time.Time
	Percent float64
	Amount  Money
	Paid    Money
	Status  string // belum | sebagian | lunas
}

// TermInput is a term as loaded for progress.
type TermInput struct {
	ID      string
	Label   string
	DueDate *time.Time
	Percent float64
}

// TermProgressOf is termProgress: payments fill the terms in order.
func TermProgressOf(terms []TermInput, total, paidTotal Money) []TermProgress {
	percents := make([]float64, len(terms))
	for i, t := range terms {
		percents[i] = t.Percent
	}
	amounts := AllocateTermAmounts(total, percents)
	remaining := paidTotal
	out := make([]TermProgress, 0, len(terms))
	for i, t := range terms {
		amount := amounts[i]
		paid := min(amount, max(0, remaining))
		remaining -= paid
		out = append(out, TermProgress{TermID: t.ID, Label: t.Label, DueDate: t.DueDate, Percent: t.Percent, Amount: amount, Paid: paid, Status: PaymentStatus(paid, amount)})
	}
	return out
}

// PaymentStatus is derivePaymentStatus.
func PaymentStatus(paid, amount Money) string {
	switch {
	case amount > 0 && paid >= amount:
		return "lunas"
	case paid > 0:
		return "sebagian"
	}
	return "belum"
}

// ReferenceTotal is resolveReferenceTotal: an accepted quotation, then the
// deal's final value, then the latest quotation, then the estimate.
func ReferenceTotal(latest *Quotation, valueFinal, valueEstimate *Money) Money {
	switch {
	case latest != nil && latest.Status == QuoteAccepted:
		return latest.Total
	case valueFinal != nil:
		return *valueFinal
	case latest != nil:
		return latest.Total
	case valueEstimate != nil:
		return *valueEstimate
	}
	return 0
}

// CanRelease is the release gate: a quotation may be sent only when no
// approval is needed or it was approved.
func CanRelease(approvalStatus string) bool {
	return approvalStatus == ApprovalNone || approvalStatus == ApprovalApproved
}

// QuoteTransitionAllowed is the PATCH status rule: draft -> terkirim,
// terkirim -> diterima | ditolak; superseded and decided quotes are frozen.
func QuoteTransitionAllowed(from, to string) bool {
	switch to {
	case QuoteSent:
		return from == QuoteDraft
	case QuoteAccepted, QuoteRejected:
		return from == QuoteSent || from == QuoteDraft
	}
	return false
}

// ExpiresSoon reports a sent quotation whose validity ends within three
// days (quotation-expiry-watcher) and that was not reminded yet.
func ExpiresSoon(q Quotation, now time.Time) bool {
	if q.Status != QuoteSent || q.ValidUntil == nil || q.ExpiryRemindedAt != nil {
		return false
	}
	today := time.Date(now.In(Jakarta).Year(), now.In(Jakarta).Month(), now.In(Jakarta).Day(), 0, 0, 0, 0, time.UTC)
	days := q.ValidUntil.Sub(today).Hours() / 24
	return days >= 0 && days <= 3
}
