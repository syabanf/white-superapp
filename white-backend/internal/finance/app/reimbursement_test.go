package app_test

import (
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
)

var png = []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0}

// staff: Sekar (u-sekar) reports to Andika (u-andika); Taufik has no manager.
func (h *harness) staff() {
	h.hr.byID["e-andika"] = &app.Employee{ID: "e-andika", UserID: "u-andika", Name: "Andika Pratama", DepartmentName: "Creative"}
	h.hr.byID["e-sekar"] = &app.Employee{ID: "e-sekar", UserID: "u-sekar", Name: "Sekar Ayu", DepartmentName: "Creative", ManagerID: "e-andika"}
	h.hr.byID["e-taufik"] = &app.Employee{ID: "e-taufik", UserID: "u-taufik", Name: "Taufik", DepartmentName: "Finance"}
	for _, e := range h.hr.byID {
		h.hr.byUser[e.UserID] = e
	}
}

func (h *harness) draft(r *app.Reimbursements, p authn.Principal, category string, amount int64, date string) *domain.Claim {
	h.t.Helper()
	c, err := r.Create(h.ctx, p, app.ClaimInput{Category: category, TransactionDate: date, Amount: rp(amount), Description: "Grab ke klien"})
	h.ok(err)
	c, err = r.UploadReceipt(h.ctx, p, c.ID, "struk.png", png)
	h.ok(err)
	return c
}

func TestReimbursementFlow(t *testing.T) {
	h := newHarness(t)
	h.staff()
	r := app.NewReimbursements(h.deps)
	sekar := principal("u-sekar", authn.PermSelfService)
	andika := principal("u-andika", authn.PermSelfService)
	finance := principal("u-fin", authn.PermFinanceRead, authn.PermFinanceApprove)
	stranger := principal("u-x", authn.PermSelfService)

	_, err := r.Create(h.ctx, stranger, app.ClaimInput{Category: domain.CategoryTransport, TransactionDate: "2026-10-20", Amount: rp(1), Description: "x"})
	h.wantErr(err, domain.MsgNoEmployee)
	_, err = r.Create(h.ctx, sekar, app.ClaimInput{Category: domain.CategoryTransport, TransactionDate: "2026-10-27", Amount: rp(1), Description: "x"})
	h.wantErr(err, domain.MsgFutureDate)

	c, err := r.Create(h.ctx, sekar, app.ClaimInput{Category: domain.CategoryTransport, TransactionDate: "2026-10-20", Amount: rp(180_000), Description: "Grab ke klien"})
	h.ok(err)
	if c.ClaimNo != "RB-202610-0001" || c.ManagerUserID != "u-andika" || c.Status != domain.ClaimDraft {
		t.Fatalf("draft %+v", c)
	}
	_, err = r.Submit(h.ctx, sekar, c.ID)
	h.wantErr(err, domain.MsgNoReceipt)
	_, err = r.UploadReceipt(h.ctx, sekar, c.ID, "x.gif", []byte("GIF89a......"))
	h.wantErr(err, domain.MsgReceiptType)
	_, err = r.UploadReceipt(h.ctx, sekar, c.ID, "big.pdf", append([]byte("%PDF-"), make([]byte, domain.MaxReceiptBytes)...))
	h.wantErr(err, domain.MsgReceiptTooLarge)
	c, err = r.UploadReceipt(h.ctx, sekar, c.ID, "../struk.png", png)
	h.ok(err)
	if len(c.Receipts) != 1 || c.Receipts[0].Filename != "struk.png" || h.storage.files[c.Receipts[0].StorageKey] == nil {
		t.Fatalf("receipt %+v", c.Receipts)
	}
	_, _, _, err = r.ReceiptFile(h.ctx, stranger, c.Receipts[0].ID)
	h.wantErr(err, domain.MsgClaimNotFound)

	c, err = r.Submit(h.ctx, sekar, c.ID)
	h.ok(err)
	if c.Status != domain.ClaimSubmitted || c.OverLimit {
		t.Fatalf("submitted %+v", c)
	}
	if n := h.identity.last(); n.Type != app.NotifClaimSubmitted || n.UserIDs[0] != "u-andika" {
		t.Fatalf("notify manager %+v", n)
	}
	_, err = r.Cancel(h.ctx, andika, c.ID)
	h.wantErr(err, domain.MsgClaimNotFound)
	_, err = r.Approve(h.ctx, finance, c.ID, "")
	if !apperr.Is(err, apperr.Unauthorized) {
		t.Fatalf("finance cannot do the manager step: %v", err)
	}
	inbox, err := r.Inbox(h.ctx, andika)
	h.ok(err)
	if len(inbox) != 1 || inbox[0].ID != c.ID {
		t.Fatalf("manager inbox %+v", inbox)
	}
	// Transport within the limits: the policy auto-approves the finance step.
	c, err = r.Approve(h.ctx, andika, c.ID, "ok")
	h.ok(err)
	if c.Status != domain.ClaimApprovedFinance || c.ManagerApprovedBy != "u-andika" {
		t.Fatalf("approved %+v", c)
	}

	// Akomodasi above the per-claim limit waits for finance.approve.
	big := h.draft(r, sekar, domain.CategoryAkomodasi, 3_000_000, "2026-10-15")
	big, err = r.Submit(h.ctx, sekar, big.ID)
	h.ok(err)
	if !big.OverLimit || big.LimitNote != "Klaim melebihi batas per klaim Rp2.500.000." {
		t.Fatalf("limit %+v", big)
	}
	big, err = r.Approve(h.ctx, andika, big.ID, "")
	h.ok(err)
	if big.Status != domain.ClaimApprovedManager {
		t.Fatalf("over limit stays at finance step %+v", big.Status)
	}
	_, err = r.Approve(h.ctx, andika, big.ID, "")
	h.wantErr(err, domain.MsgOverLimitApprove)
	finInbox, err := r.Inbox(h.ctx, finance)
	h.ok(err)
	if len(finInbox) != 1 || finInbox[0].ID != big.ID {
		t.Fatalf("finance inbox %+v", finInbox)
	}
	big, err = r.Approve(h.ctx, finance, big.ID, "")
	h.ok(err)
	if big.Status != domain.ClaimApprovedFinance || big.FinanceApprovedBy != "u-fin" {
		t.Fatalf("finance approved %+v", big)
	}

	// Rejection needs a reason; cancel only before approval; 30-day rule.
	rej := h.draft(r, sekar, domain.CategoryMakan, 100_000, "2026-10-25")
	rej, err = r.Submit(h.ctx, sekar, rej.ID)
	h.ok(err)
	_, err = r.Reject(h.ctx, andika, rej.ID, " ")
	h.wantErr(err, domain.MsgRejectReason)
	rej, err = r.Reject(h.ctx, andika, rej.ID, "Tidak ada bukti yang jelas")
	h.ok(err)
	if rej.Status != domain.ClaimRejected || rej.RejectedStage != "SUBMITTED" || h.identity.last().Type != app.NotifClaimRejected {
		t.Fatalf("rejected %+v", rej)
	}
	old := h.draft(r, sekar, domain.CategoryTransport, 50_000, "2026-09-25")
	_, err = r.Submit(h.ctx, sekar, old.ID)
	h.wantErr(err, domain.MsgClaimTooOld)
	old, err = r.Cancel(h.ctx, sekar, old.ID)
	h.ok(err)
	if old.Status != domain.ClaimCancelled {
		t.Fatal(old.Status)
	}

	// No manager: straight to the finance step.
	taufik := principal("u-taufik", authn.PermSelfService)
	own := h.draft(r, taufik, domain.CategoryOperasional, 400_000, "2026-10-24")
	own, err = r.Submit(h.ctx, taufik, own.ID)
	h.ok(err)
	if own.Status != domain.ClaimApprovedManager || h.identity.last().UserIDs[0] != "finance-user" {
		t.Fatalf("no manager %+v", own)
	}

	// Pay both approved claims in one batch: one journal, one event.
	_, err = r.Pay(h.ctx, finance, app.PayInput{ClaimIDs: []string{c.ID, own.ID}})
	h.wantErr(err, domain.MsgPayNotApproved+" ("+own.ClaimNo+")")
	batch, err := r.Pay(h.ctx, finance, app.PayInput{ClaimIDs: []string{c.ID, big.ID}, CashAccountID: h.id("1101001"), PaidAt: "2026-10-26"})
	h.ok(err)
	if batch.JournalStatus != domain.PostPosted || batch.TotalAmount != rp(3_180_000) || batch.ClaimCount != 2 || batch.BatchNo != "RBP-20261026-0001" {
		t.Fatalf("batch %+v", batch)
	}
	entry, err := h.deps.Store.GetJournalEntry(h.ctx, batch.JournalEntryID)
	h.ok(err)
	byAccount := map[string]domain.Money{}
	for _, l := range entry.Lines {
		byAccount[l.AccountID+l.Side] = l.Amount
	}
	if byAccount[h.id("6302001")+domain.Debit] != rp(180_000) || byAccount[h.id("6302003")+domain.Debit] != rp(3_000_000) || byAccount[h.id("1101001")+domain.Credit] != rp(3_180_000) {
		t.Fatalf("journal lines %+v", entry.Lines)
	}
	if len(h.events.sent) != 1 || h.events.sent[0].subject != events.FinanceReimbursementPaid {
		t.Fatalf("events %+v", h.events.sent)
	}
	payload := h.events.sent[0].payload.(domain.ReimbursementPaid)
	if len(payload.ClaimIDs) != 2 || payload.EmployeeIDs[0] != "e-sekar" || payload.PaidAt != "2026-10-26" {
		t.Fatalf("payload %+v", payload)
	}
	paid, err := r.Get(h.ctx, sekar, c.ID)
	h.ok(err)
	if paid.Status != domain.ClaimPaid || paid.PaymentBatchID != batch.ID || len(paid.History) != 4 {
		t.Fatalf("paid %+v", paid)
	}

	month, rows, err := r.MonthlySummary(h.ctx, sekar, "2026-10", false)
	h.ok(err)
	var paidTotal domain.Money
	for _, row := range rows {
		if row.EmployeeID != "e-sekar" {
			t.Fatalf("self-service sees only own rows: %+v", row)
		}
		paidTotal += row.PaidAmount
	}
	if month != "2026-10" || paidTotal != rp(3_180_000) {
		t.Fatalf("summary %s %v", month, paidTotal)
	}
	mine, total, _, err := r.MyClaims(h.ctx, sekar, "")
	h.ok(err)
	if total != 4 || len(mine) != 4 {
		t.Fatalf("my claims %d", total)
	}
}

func TestReimbursementRemindersAndForget(t *testing.T) {
	h := newHarness(t)
	h.staff()
	r := app.NewReimbursements(h.deps)
	sekar := principal("u-sekar", authn.PermSelfService)
	c := h.draft(r, sekar, domain.CategoryMakan, 100_000, "2026-10-20")
	_, err := r.Submit(h.ctx, sekar, c.ID)
	h.ok(err)
	draft := h.draft(r, sekar, domain.CategoryMakan, 50_000, "2026-10-21")

	n, err := r.RemindApprovers(h.ctx, ptime("2026-10-27T03:00:00Z"))
	h.ok(err)
	if n != 0 {
		t.Fatal("waiting one day is not late")
	}
	n, err = r.RemindApprovers(h.ctx, ptime("2026-10-29T03:00:00Z"))
	h.ok(err)
	if n != 1 || h.identity.last().Type != app.NotifClaimReminder || h.identity.last().UserIDs[0] != "u-andika" {
		t.Fatalf("reminder %d %+v", n, h.identity.last())
	}
	n, err = r.RemindApprovers(h.ctx, ptime("2026-10-29T09:00:00Z"))
	h.ok(err)
	if n != 0 {
		t.Fatal("reminded at most once a day")
	}

	h.ok(r.ForgetUser(h.ctx, "u-andika"))
	got, err := h.deps.Store.GetClaim(h.ctx, c.ID)
	h.ok(err)
	if got.ManagerUserID != "" {
		t.Fatal("manager reference kept")
	}
	h.ok(r.ForgetUser(h.ctx, "u-sekar"))
	gone, err := h.deps.Store.GetClaim(h.ctx, draft.ID)
	h.ok(err)
	if gone != nil || h.storage.files[draft.Receipts[0].StorageKey] != nil {
		t.Fatal("draft claim and receipt must be deleted")
	}
}
