package app_test

import (
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
)

func item(desc string, qty, price int64) domain.DocItem {
	return domain.DocItem{Description: desc, Unit: "paket", Qty: rp(qty), UnitPrice: rp(price)}
}

func code(t *testing.T, err error, want apperr.Code) {
	t.Helper()
	if err == nil || apperr.As(err).Code != want {
		t.Fatalf("want %v, got %v", want, err)
	}
}

func TestPurchaseFlowToPayable(t *testing.T) {
	h := newHarness(t)
	s := app.NewDocuments(h.deps)
	maker := principal("u-maker", authn.PermFinanceWrite)
	checker := principal("u-checker", authn.PermFinanceApprove)
	admin := authn.Principal{UserID: "u-maker", Role: authn.RoleAdmin}
	expense := h.id("6301004")

	_, err := s.SaveVendor(h.ctx, "u1", domain.Vendor{Name: " "})
	code(t, err, apperr.Validation)
	vendor, err := s.SaveVendor(h.ctx, "u1", domain.Vendor{Name: "CV Cetak", PicName: "Budi", Phone: "62811", IsActive: true, PaymentTermsDays: 30, BankName: "BCA"})
	h.ok(err)
	_, err = s.SaveVendor(h.ctx, "u1", domain.Vendor{Name: "cv cetak", IsActive: true})
	h.wantErr(err, "Nama vendor sudah dipakai")

	// A request has no vendor yet and needs a second person to approve it.
	_, err = s.Create(h.ctx, maker.UserID, domain.Document{Kind: domain.DocPR, Title: "Cetak", IssueDate: today})
	code(t, err, apperr.Validation)
	pr, err := s.Create(h.ctx, maker.UserID, domain.Document{Kind: domain.DocPR, Title: "Cetak brosur", IssueDate: today, Items: []domain.DocItem{item("Brosur", 2, 1_000_000)}, PPNPercent: rp(11)})
	h.ok(err)
	if pr.Status != domain.DocDraft || pr.Total != rp(2_220_000) || pr.Side != domain.SideSupplier {
		t.Fatalf("pr %+v", pr)
	}
	_, err = s.Act(h.ctx, checker, pr.ID, domain.ActApprove, "")
	code(t, err, apperr.Conflict)
	_, err = s.Act(h.ctx, maker, pr.ID, domain.ActSubmit, "")
	h.ok(err)
	_, err = s.Act(h.ctx, principal("u-maker", authn.PermFinanceApprove), pr.ID, domain.ActApprove, "")
	h.wantErr(err, "Pembuat dokumen tidak bisa memutuskan dokumennya sendiri")
	_, err = s.Act(h.ctx, checker, pr.ID, domain.ActReject, " ")
	h.wantErr(err, "Alasan penolakan wajib diisi")
	rejected, err := s.Act(h.ctx, checker, pr.ID, domain.ActReject, "Terlalu mahal")
	h.ok(err)
	if rejected.RejectReason != "Terlalu mahal" || rejected.DecidedBy != "u-checker" {
		t.Fatalf("rejected %+v", rejected)
	}
	_, err = s.Act(h.ctx, maker, pr.ID, domain.ActReopen, "")
	h.ok(err)
	_, err = s.Act(h.ctx, maker, pr.ID, domain.ActSubmit, "")
	h.ok(err)
	approved, err := s.Act(h.ctx, admin, pr.ID, domain.ActApprove, "") // Admin may decide their own.
	h.ok(err)
	if approved.Status != domain.DocApproved || approved.RejectReason != "" {
		t.Fatalf("approved %+v", approved)
	}

	// The request becomes a purchase order once.
	_, err = s.Convert(h.ctx, maker.UserID, pr.ID, "")
	h.wantErr(err, "Pilih vendor untuk PO")
	po, err := s.Convert(h.ctx, maker.UserID, pr.ID, vendor.ID)
	h.ok(err)
	if po.Kind != domain.DocPO || po.Status != domain.DocDraft || po.PartyName != "CV Cetak" || po.PicName != "Budi" || po.SourceID != pr.ID || po.Total != pr.Total {
		t.Fatalf("po %+v", po)
	}
	_, err = s.Convert(h.ctx, maker.UserID, pr.ID, vendor.ID)
	code(t, err, apperr.Conflict)

	// Deleting the draft PO frees the request.
	h.ok(s.Delete(h.ctx, po.ID))
	again, err := s.Get(h.ctx, pr.ID)
	h.ok(err)
	if again.Status != domain.DocApproved {
		t.Fatalf("source %+v", again)
	}
	po, err = s.Convert(h.ctx, maker.UserID, pr.ID, vendor.ID)
	h.ok(err)

	// A draft PO cannot bill; an approved one bills once, as a draft payable.
	_, err = s.Issue(h.ctx, maker.UserID, po.ID, expense, "")
	code(t, err, apperr.Conflict)
	_, err = s.Act(h.ctx, maker, po.ID, domain.ActSubmit, "")
	h.ok(err)
	_, err = s.Act(h.ctx, checker, po.ID, domain.ActApprove, "")
	h.ok(err)
	_, err = s.Issue(h.ctx, maker.UserID, po.ID, "", "")
	h.wantErr(err, "Pilih akun beban atau pendapatan")
	billed, err := s.Issue(h.ctx, maker.UserID, po.ID, expense, "2026-11-30")
	h.ok(err)
	if billed.Status != domain.DocConverted || billed.LedgerInvoiceID == "" {
		t.Fatalf("billed %+v", billed)
	}
	inv, err := app.NewLedgers(h.deps).GetInvoice(h.ctx, domain.KindAP, billed.LedgerInvoiceID)
	h.ok(err)
	if inv.Status != domain.StatusDraft || inv.PartyName != "CV Cetak" || inv.Subtotal != rp(2_000_000) || inv.TaxAmount != rp(220_000) || inv.TotalAmount != rp(2_220_000) || inv.Reference != po.Number {
		t.Fatalf("payable %+v", inv)
	}
	_, err = s.Issue(h.ctx, maker.UserID, po.ID, expense, "")
	code(t, err, apperr.Conflict)
	_, err = s.Update(h.ctx, maker.UserID, *billed)
	h.wantErr(err, "Hanya dokumen draf yang bisa diubah")
}

func TestQuotationToReceivable(t *testing.T) {
	h := newHarness(t)
	s := app.NewDocuments(h.deps)
	maker := principal("u1", authn.PermFinanceWrite)
	revenue := h.id("4101001")

	qt, err := s.Create(h.ctx, "u1", domain.Document{Kind: domain.DocQuotation, Side: domain.SideB2B, Title: "Retainer", PartyName: "PT Tepat", IssueDate: today,
		Items: []domain.DocItem{item("Bulanan", 1, 10_000_000)}, DiscountPercent: rp(10), PPNPercent: rp(11)})
	h.ok(err)
	if qt.Total != rp(9_990_000) {
		t.Fatalf("total %d", qt.Total)
	}
	_, err = s.Convert(h.ctx, "u1", qt.ID, "")
	code(t, err, apperr.Conflict)
	sent, err := s.Act(h.ctx, maker, qt.ID, domain.ActSend, "")
	h.ok(err)
	if len(sent.Letterhead) == 0 {
		t.Fatal("letterhead is stored when a document leaves draft")
	}
	_, err = s.Act(h.ctx, maker, qt.ID, domain.ActAccept, "")
	h.ok(err)
	inv, err := s.Convert(h.ctx, "u1", qt.ID, "")
	h.ok(err)
	if inv.Kind != domain.DocInvoice || inv.Side != domain.SideB2B || inv.Status != domain.DocDraft {
		t.Fatalf("invoice %+v", inv)
	}
	issued, err := s.Issue(h.ctx, "u1", inv.ID, revenue, "2026-11-30")
	h.ok(err)
	if issued.Status != domain.DocIssued {
		t.Fatalf("issued %+v", issued)
	}
	ar, err := app.NewLedgers(h.deps).GetInvoice(h.ctx, domain.KindAR, issued.LedgerInvoiceID)
	h.ok(err)
	if ar.TotalAmount != rp(9_990_000) || ar.Subtotal != rp(9_000_000) {
		t.Fatalf("receivable %+v", ar)
	}
	rows, total, err := s.List(h.ctx, domain.DocumentFilter{Kind: domain.DocInvoice, Query: "tepat"}, 1, 10)
	h.ok(err)
	if total != 1 || rows[0].ID != inv.ID {
		t.Fatalf("list %d %+v", total, rows)
	}
}
