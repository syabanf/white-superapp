package domain

import (
	"fmt"
	"time"
)

// Follow-up statuses. Nothing reaches a client without a person approving it.
const (
	FollowupPending  = "PENDING"
	FollowupApproved = "APPROVED"
	FollowupSent     = "SENT"
	FollowupSkipped  = "SKIPPED"
)

// FollowupStages are days from the due date.
var FollowupStages = []int{-3, 0, 3, 7}

// followupGraceDays lets a missed daily tick catch up without nagging about
// invoices long past a stage.
const followupGraceDays = 2

// InvoiceFollowup is one queued WhatsApp message about an unpaid invoice.
type InvoiceFollowup struct {
	ID, InvoiceID, Phone, Message, Status, LastError string
	Stage                                            int
	ApprovedBy, MessageID                            *string
	ApprovedAt, SentAt                               *time.Time
	CreatedAt                                        time.Time
	// Joined.
	InvoiceNumber, DealID, ClientName string
}

// FollowupStageDue returns the stage to queue for a due date, if one is due
// today or within the grace days.
func FollowupStageDue(due, now time.Time) (int, bool) {
	dueDay := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, Jakarta)
	local := now.In(Jakarta)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, Jakarta)
	days := int(today.Sub(dueDay).Hours() / 24)
	for i := len(FollowupStages) - 1; i >= 0; i-- {
		if s := FollowupStages[i]; days >= s && days <= s+followupGraceDays {
			return s, true
		}
	}
	return 0, false
}

// FollowupSendable is the sending window: Monday to Friday, 08.00 to 17.59 WIB.
func FollowupSendable(now time.Time) bool {
	t := now.In(Jakarta)
	return t.Weekday() != time.Saturday && t.Weekday() != time.Sunday && t.Hour() >= 8 && t.Hour() < 18
}

// FollowupMessage drafts the text a person reviews before it is sent.
func FollowupMessage(pic, invoiceNumber string, remaining Money, due time.Time, stage int) string {
	greeting := "Halo"
	if pic != "" {
		greeting = "Halo " + pic
	}
	date := due.Format("02-01-2006")
	var when string
	switch {
	case stage < 0:
		when = "akan jatuh tempo pada " + date
	case stage == 0:
		when = "jatuh tempo hari ini"
	default:
		when = fmt.Sprintf("sudah lewat %d hari dari jatuh tempo %s", stage, date)
	}
	return fmt.Sprintf("%s, kami mengingatkan invoice %s dengan sisa tagihan %s %s. Mohon kirim bukti transfer di chat ini bila sudah dibayar. Terima kasih.",
		greeting, invoiceNumber, remaining.Rupiah(), when)
}
