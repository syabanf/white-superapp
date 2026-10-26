package domain

import (
	"fmt"
	"unicode/utf16"
)

// OfferStatusForAction maps a portal action to the offer status it sets.
var OfferStatusForAction = map[string]string{"accept": "accepted", "negotiate": "negotiating", "decline": "declined"}

var offerActionLabels = map[string]string{"accept": "menerima", "negotiate": "mengajukan negosiasi", "decline": "menolak"}

var manualStatusLabels = map[string]string{"accepted": "menerima", "negotiating": "mengajukan negosiasi", "declined": "menolak"}

// ManualOfferStatuses are the responses HR may record by hand.
var ManualOfferStatuses = []string{"negotiating", "accepted", "declined"}

var offerStatusLabels = map[string]string{
	"sent": "Terkirim", "negotiating": "Negosiasi", "accepted": "Diterima", "declined": "Ditolak", "expired": "Kedaluwarsa",
}

// OfferStatusLabel is the label of an offer status.
func OfferStatusLabel(s string) string {
	if l, ok := offerStatusLabels[s]; ok {
		return l
	}
	return s
}

// OfferResponseVerb is the verb of a response status ("menerima").
func OfferResponseVerb(status string) string { return manualStatusLabels[status] }

// IsOfferOpen reports whether an offer can still be answered.
func IsOfferOpen(status string) bool { return status == "sent" || status == "negotiating" }

// IsOfferFinal reports a response the candidate cannot take back.
func IsOfferFinal(status string) bool { return status == "accepted" || status == "declined" }

// Offer limits of offerSchema.
const (
	MaxOfferBenefits    = 15
	MaxBenefitChars     = 120
	MaxOfferSalary      = 1_000_000_000_000
	DefaultOfferExpires = 7
)

// Messages of the offer flows.
const (
	MsgOfferClosed       = "Penawaran ini sudah tidak bisa direspons"
	MsgOfferFinal        = "Offer ini sudah direspons final oleh kandidat"
	MsgNegotiateNoteNeed = "Tuliskan catatan negosiasi Anda (mis. angka yang diharapkan)"
)

// OfferResponseDescription is the activity text of a portal response; the
// note is cut at 300 UTF-16 units like String.slice.
func OfferResponseDescription(action string, version int, note *string) string {
	s := fmt.Sprintf("Kandidat %s offer v%d via portal", offerActionLabels[action], version)
	if note != nil && *note != "" {
		s += ` — "` + JSSlice(*note, 300) + `"`
	}
	return s
}

// ManualOfferResponseDescription is the activity text when HR records a
// response by hand.
func ManualOfferResponseDescription(status string, version int, note string) string {
	s := fmt.Sprintf("Kandidat %s offer v%d (dicatat manual)", manualStatusLabels[status], version)
	if note != "" {
		s += ` — "` + JSSlice(note, 300) + `"`
	}
	return s
}

// OfferCreatedDescription is the activity of a new offer version.
func OfferCreatedDescription(version, expiresDays int) string {
	return fmt.Sprintf("Offer v%d dibuat (berlaku %d hari)", version, expiresDays)
}

// JSSlice is String.prototype.slice(0, n): the first n UTF-16 code units.
func JSSlice(s string, n int) string {
	u := utf16.Encode([]rune(s))
	if len(u) <= n {
		return s
	}
	return string(utf16.Decode(u[:n]))
}

// UTF16Len is String.prototype.length.
func UTF16Len(s string) int { return len(utf16.Encode([]rune(s))) }
