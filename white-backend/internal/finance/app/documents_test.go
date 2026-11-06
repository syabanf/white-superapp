package app_test

import (
	"context"
	"strings"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
)

func item(desc string, qty, price int64) domain.DocItem {
	return domain.DocItem{Description: desc, Unit: "paket", Qty: rp(qty), UnitPrice: rp(price)}
}

// vat adds an 11% VAT code to the harness and returns its id.
func (h *harness) vat(saleAccount, purchaseAccount string) string {
	h.t.Helper()
	t, err := app.NewTaxSetup(h.deps).SaveTaxCode(h.ctx, "u1", domain.TaxCode{Code: "PPN-T", Name: "PPN 11%", Kind: domain.TaxVAT, Scope: domain.ScopeBoth, Rate: rp(11), BaseNumerator: 1, BaseDenominator: 1,
		AccountSaleID: h.id(saleAccount), AccountPurchaseID: h.id(purchaseAccount), IsActive: true})
	h.ok(err)
	return t.ID
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
	vat := h.vat("2103003", "1104001")
	pr, err := s.Create(h.ctx, maker.UserID, domain.Document{Kind: domain.DocPR, Title: "Cetak brosur", IssueDate: today, Items: []domain.DocItem{item("Brosur", 2, 1_000_000)}, TaxIDs: []string{vat}})
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
		Items: []domain.DocItem{item("Bulanan", 1, 10_000_000)}, DiscountPercent: rp(10), TaxIDs: []string{h.vat("2103003", "1104001")}})
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

// A sale with PPN, PPh 23 withheld by the client, and a stamp duty charge posts
// a balanced journal and shows in the tax register.
func TestDocumentPostsTaxesAndCharges(t *testing.T) {
	h := newHarness(t)
	setup := app.NewTaxSetup(h.deps)
	s := app.NewDocuments(h.deps)
	l := app.NewLedgers(h.deps)
	revenue := h.id("4101001")

	ppn := h.vat("2103003", "1104001")
	pph, err := setup.SaveTaxCode(h.ctx, "u1", domain.TaxCode{Code: "PPH23-T", Name: "PPh 23 2%", Kind: domain.TaxWithholding, Scope: domain.ScopeBoth, Rate: rp(2), BaseNumerator: 1, BaseDenominator: 1,
		AccountSaleID: h.id("1104002"), AccountPurchaseID: h.id("2103002"), IsActive: true})
	h.ok(err)
	materai, err := setup.SaveChargeType(h.ctx, "u1", domain.ChargeType{Code: "MAT", Name: "Bea materai", Scope: domain.ScopeBoth, Computation: domain.ChargeFixed, Value: rp(10_000), IsActive: true})
	h.ok(err)
	ongkir, err := setup.SaveChargeType(h.ctx, "u1", domain.ChargeType{Code: "ONG", Name: "Ongkir", Scope: domain.ScopeBoth, Computation: domain.ChargeFixed, Taxable: true, IsActive: true, AccountSaleID: h.id("4101002")})
	h.ok(err)

	base := domain.Document{Kind: domain.DocInvoice, Side: domain.SideB2B, Title: "Retainer", PartyName: "PT Tepat", IssueDate: today, Items: []domain.DocItem{item("Bulanan", 1, 10_000_000)}}
	bad := base
	bad.Charges = []domain.DocCharge{{ChargeID: ongkir.ID}}
	_, err = s.Create(h.ctx, "u1", bad)
	h.wantErr(err, "Isi nilai Ongkir")

	doc := base
	doc.TaxIDs = []string{ppn, pph.ID}
	doc.Charges = []domain.DocCharge{{ChargeID: materai.ID}, {ChargeID: ongkir.ID, Value: rp(100_000)}}
	d, err := s.Create(h.ctx, "u1", doc)
	h.ok(err)
	// DPP 10.100.000, PPN 1.111.000, materai 10.000, PPh 202.000.
	if d.Tax != rp(1_111_000) || d.Withholding != rp(202_000) || d.Total != rp(11_221_000) || d.NetDue() != rp(11_019_000) {
		t.Fatalf("totals tax %d wh %d total %d net %d", d.Tax, d.Withholding, d.Total, d.NetDue())
	}

	// Changing the master later leaves this document alone.
	changed := *mustTax(t, h, ppn)
	changed.Rate = rp(50)
	_, err = setup.SaveTaxCode(h.ctx, "u1", changed)
	h.ok(err)
	again, err := s.Get(h.ctx, d.ID)
	h.ok(err)
	if again.Taxes[0].Rate != rp(11) || again.Tax != rp(1_111_000) {
		t.Fatalf("snapshot moved: %+v", again.Taxes[0])
	}

	issued, err := s.Issue(h.ctx, "u1", d.ID, revenue, "")
	h.ok(err)
	inv, err := l.GetInvoice(h.ctx, domain.KindAR, issued.LedgerInvoiceID)
	h.ok(err)
	if inv.TotalAmount != rp(11_019_000) || inv.TaxAmount != rp(1_111_000) {
		t.Fatalf("invoice %+v", inv)
	}
	_, err = l.UpdateInvoice(h.ctx, domain.KindAR, "u1", inv.ID, app.InvoiceInput{InvoiceDate: today, PartyName: "X", Subtotal: rp(1)})
	h.wantErr(err, "Tagihan ini dibuat dari dokumen, jadi tidak bisa diubah atau dihapus di sini")

	posted, note, err := l.PostInvoice(h.ctx, domain.KindAR, "u1", inv.ID)
	h.ok(err)
	if posted.JournalEntryID == "" {
		t.Fatalf("journal not posted: %s", note)
	}
	e, err := app.NewJournals(h.deps).Get(h.ctx, posted.JournalEntryID)
	h.ok(err)
	want := map[string]struct {
		side   string
		amount int64
	}{
		h.id("1103001"): {domain.Debit, 11_019_000},
		h.id("1104002"): {domain.Debit, 202_000},
		h.id("2103003"): {domain.Credit, 1_111_000},
		h.id("4101002"): {domain.Credit, 100_000},
		revenue:         {domain.Credit, 10_010_000}, // Service plus the materai with no account of its own.
	}
	if len(e.Lines) != len(want) {
		t.Fatalf("lines %+v", e.Lines)
	}
	for _, line := range e.Lines {
		w, ok := want[line.AccountID]
		if !ok || line.Side != w.side || line.Amount != rp(w.amount) {
			t.Fatalf("line %s %s %d, want %+v", line.AccountCode, line.Side, line.Amount, w)
		}
	}

	// The client pays the net and the invoice closes.
	_, err = l.CreatePayment(h.ctx, domain.KindAR, "u1", app.PaymentInput{PaymentDate: today, CashAccountID: h.id("1101001"), Allocations: []domain.Allocation{{InvoiceID: inv.ID, Amount: rp(11_019_000)}}})
	h.ok(err)
	paid, err := l.GetInvoice(h.ctx, domain.KindAR, inv.ID)
	h.ok(err)
	if paid.Outstanding() != 0 {
		t.Fatalf("outstanding %d", paid.Outstanding())
	}

	reg, err := setup.TaxRegister(h.ctx, "2026-10-01", "2026-10-31")
	h.ok(err)
	if reg.OutputVAT != rp(1_111_000) || reg.VATPosition != rp(1_111_000) || reg.WithheldByClients != rp(202_000) || len(reg.Rows) != 2 {
		t.Fatalf("register %+v", reg)
	}
}

func TestPurchaseWithholdingPostsPayable(t *testing.T) {
	h := newHarness(t)
	setup := app.NewTaxSetup(h.deps)
	s := app.NewDocuments(h.deps)
	l := app.NewLedgers(h.deps)
	ppn := h.vat("2103003", "1104001")
	pph, err := setup.SaveTaxCode(h.ctx, "u1", domain.TaxCode{Code: "PPH23-T", Name: "PPh 23 2%", Kind: domain.TaxWithholding, Scope: domain.ScopeBoth, Rate: rp(2), BaseNumerator: 1, BaseDenominator: 1,
		AccountSaleID: h.id("1104002"), AccountPurchaseID: h.id("2103002"), IsActive: true})
	h.ok(err)
	d, err := s.Create(h.ctx, "u1", domain.Document{Kind: domain.DocInvoice, Side: domain.SideSupplier, Title: "Jasa cetak", PartyName: "CV Cetak", IssueDate: today, Items: []domain.DocItem{item("Cetak", 1, 1_000_000)}, TaxIDs: []string{ppn, pph.ID}})
	h.ok(err)
	issued, err := s.Issue(h.ctx, "u1", d.ID, h.id("6301004"), "")
	h.ok(err)
	posted, note, err := l.PostInvoice(h.ctx, domain.KindAP, "u1", issued.LedgerInvoiceID)
	h.ok(err)
	if posted.TotalAmount != rp(1_090_000) || posted.JournalEntryID == "" {
		t.Fatalf("payable %+v %s", posted, note)
	}
	e, err := app.NewJournals(h.deps).Get(h.ctx, posted.JournalEntryID)
	h.ok(err)
	if l1 := lineOn(e, h.id("2103002")); l1.Side != domain.Credit || l1.Amount != rp(20_000) {
		t.Fatalf("withheld %+v", l1)
	}
	if l2 := lineOn(e, h.id("1104001")); l2.Side != domain.Debit || l2.Amount != rp(110_000) {
		t.Fatalf("input VAT %+v", l2)
	}
	if l3 := lineOn(e, h.id("2101001")); l3.Side != domain.Credit || l3.Amount != rp(1_090_000) {
		t.Fatalf("payable line %+v", l3)
	}
	reg, err := setup.TaxRegister(h.ctx, "2026-10-01", "2026-10-31")
	h.ok(err)
	if reg.InputVAT != rp(110_000) || reg.VATPosition != rp(-110_000) || reg.WithheldFromVendors != rp(20_000) {
		t.Fatalf("register %+v", reg)
	}
}

func TestTaxMasterRules(t *testing.T) {
	h := newHarness(t)
	setup := app.NewTaxSetup(h.deps)
	s := app.NewDocuments(h.deps)
	valid := domain.TaxCode{Code: "A", Name: "A", Kind: domain.TaxVAT, Scope: domain.ScopeBoth, Rate: rp(11), BaseNumerator: 1, BaseDenominator: 1, AccountSaleID: h.id("2103003"), AccountPurchaseID: h.id("1104001"), IsActive: true}
	bad := valid
	bad.BaseNumerator = 13
	bad.BaseDenominator = 12
	_, err := setup.SaveTaxCode(h.ctx, "u1", bad)
	code(t, err, apperr.Validation)
	bad = valid
	bad.Kind, bad.PriceIncluded = domain.TaxWithholding, true
	_, err = setup.SaveTaxCode(h.ctx, "u1", bad)
	code(t, err, apperr.Validation)
	bad = valid
	bad.AccountSaleID = ""
	_, err = setup.SaveTaxCode(h.ctx, "u1", bad)
	h.wantErr(err, "Kode pajak aktif perlu akun posting")
	bad = valid
	bad.AccountSaleID = h.id("1101000")
	_, err = setup.SaveTaxCode(h.ctx, "u1", bad)
	code(t, err, apperr.Validation)

	// One default per side.
	first := valid
	first.IsDefaultSale = true
	a, err := setup.SaveTaxCode(h.ctx, "u1", first)
	h.ok(err)
	second := valid
	second.Code, second.IsDefaultSale = "B", true
	b, err := setup.SaveTaxCode(h.ctx, "u1", second)
	h.ok(err)
	if got := mustTax(t, h, a.ID); got.IsDefaultSale {
		t.Fatal("two defaults for sales")
	}
	_, err = setup.SaveTaxCode(h.ctx, "u1", valid)
	h.wantErr(err, "Kode pajak sudah dipakai")

	item1 := []domain.DocItem{item("X", 1, 100)}
	_, err = s.Create(h.ctx, "u1", domain.Document{Kind: domain.DocInvoice, Side: domain.SideB2B, Title: "T", PartyName: "P", IssueDate: today, Items: item1, TaxIDs: []string{a.ID, b.ID}})
	h.wantErr(err, "Satu dokumen hanya boleh memakai satu kode PPN")

	// Validity and side.
	dated := valid
	dated.Code, dated.ValidFrom = "C", "2030-01-01"
	c, err := setup.SaveTaxCode(h.ctx, "u1", dated)
	h.ok(err)
	_, err = s.Create(h.ctx, "u1", domain.Document{Kind: domain.DocInvoice, Side: domain.SideB2B, Title: "T", PartyName: "P", IssueDate: today, Items: item1, TaxIDs: []string{c.ID}})
	h.wantErr(err, "C tidak berlaku pada tanggal dokumen")
	sale := valid
	sale.Code, sale.Scope, sale.AccountPurchaseID = "D", domain.ScopeSale, ""
	d, err := setup.SaveTaxCode(h.ctx, "u1", sale)
	h.ok(err)
	_, err = s.Create(h.ctx, "u1", domain.Document{Kind: domain.DocPR, Title: "T", IssueDate: today, Items: item1, TaxIDs: []string{d.ID}})
	h.wantErr(err, "D tidak berlaku untuk pihak ini")
	// Inactive codes are refused.
	off := *mustTax(t, h, d.ID)
	off.IsActive = false
	_, err = setup.SaveTaxCode(h.ctx, "u1", off)
	h.ok(err)
	_, err = s.Create(h.ctx, "u1", domain.Document{Kind: domain.DocInvoice, Side: domain.SideB2B, Title: "T", PartyName: "P", IssueDate: today, Items: item1, TaxIDs: []string{d.ID}})
	h.wantErr(err, "Kode pajak tidak ditemukan atau nonaktif")
}

func mustTax(t *testing.T, h *harness, id string) *domain.TaxCode {
	t.Helper()
	got, err := h.deps.Store.GetTaxCode(h.ctx, id)
	h.ok(err)
	return got
}

func TestManualInvoiceTaxCodes(t *testing.T) {
	h := newHarness(t)
	l := app.NewLedgers(h.deps)
	ppn := h.vat("2103003", "1104001")
	inv, err := l.CreateInvoice(h.ctx, domain.KindAP, "u1", app.InvoiceInput{InvoiceDate: today, PartyName: "CV A", Subtotal: rp(1_000_000), TaxAmount: rp(999), TaxIDs: []string{ppn}})
	h.ok(err)
	if inv.TaxAmount != rp(110_000) || inv.TotalAmount != rp(1_110_000) {
		t.Fatalf("typed tax must lose to the code: %+v", inv)
	}
	ids, err := l.InvoiceTaxIDs(h.ctx, inv.ID)
	h.ok(err)
	if len(ids) != 1 || ids[0] != ppn {
		t.Fatalf("ids %v", ids)
	}
	updated, err := l.UpdateInvoice(h.ctx, domain.KindAP, "u1", inv.ID, app.InvoiceInput{InvoiceDate: today, PartyName: "CV A", Subtotal: rp(2_000_000), TaxIDs: []string{ppn}})
	h.ok(err)
	if updated.TaxAmount != rp(220_000) {
		t.Fatalf("update %+v", updated)
	}
	cleared, err := l.UpdateInvoice(h.ctx, domain.KindAP, "u1", inv.ID, app.InvoiceInput{InvoiceDate: today, PartyName: "CV A", Subtotal: rp(2_000_000), TaxAmount: rp(5)})
	h.ok(err)
	ids, err = l.InvoiceTaxIDs(h.ctx, inv.ID)
	h.ok(err)
	if cleared.TaxAmount != rp(5) || len(ids) != 0 {
		t.Fatalf("clear %+v %v", cleared, ids)
	}
}

func TestPartyProfilesAndFiscalPositions(t *testing.T) {
	h := newHarness(t)
	setup := app.NewTaxSetup(h.deps)
	s := app.NewDocuments(h.deps)
	ppn := h.vat("2103003", "1104001")
	doc := domain.Document{Kind: domain.DocInvoice, Side: domain.SideSupplier, Title: "T", PartyName: "CV Kecil", IssueDate: today, Items: []domain.DocItem{item("X", 1, 1_000_000)}, TaxIDs: []string{ppn}}

	// A vendor outside PKP cannot charge VAT.
	_, err := setup.SavePartyProfile(h.ctx, "u1", domain.PartyProfile{Side: domain.SideSupplier, Name: "CV Kecil", IsPKP: false})
	h.ok(err)
	_, err = s.Create(h.ctx, "u1", doc)
	h.wantErr(err, "CV Kecil bukan PKP, jadi tidak bisa memungut PPN")
	_, err = setup.SavePartyProfile(h.ctx, "u1", domain.PartyProfile{Side: domain.SideSupplier, Name: "cv kecil", IsPKP: true})
	h.wantErr(err, "Profil pihak ini sudah ada")

	// A fiscal position drops the code instead, and matches the name in any case.
	fp, err := setup.SaveFiscalPosition(h.ctx, "u1", domain.FiscalPosition{Name: "Non-PKP", IsActive: true, Rules: []domain.FiscalRule{{FromTaxID: ppn}}})
	h.ok(err)
	_, err = setup.SavePartyProfile(h.ctx, "u1", domain.PartyProfile{Side: domain.SideSupplier, Name: "CV Dua", IsPKP: false, FiscalPositionID: fp.ID})
	h.ok(err)
	two := doc
	two.PartyName = "cv dua"
	d, err := s.Create(h.ctx, "u1", two)
	h.ok(err)
	if len(d.Taxes) != 0 || d.Tax != 0 {
		t.Fatalf("VAT stayed: %+v", d.Taxes)
	}

	// A swap replaces one code with another.
	wh, err := setup.SaveTaxCode(h.ctx, "u1", domain.TaxCode{Code: "PPH-4", Name: "PPh 23 4%", Kind: domain.TaxWithholding, Scope: domain.ScopeBoth, Rate: rp(4), BaseNumerator: 1, BaseDenominator: 1, AccountSaleID: h.id("1104002"), AccountPurchaseID: h.id("2103002"), IsActive: true})
	h.ok(err)
	base, err := setup.SaveTaxCode(h.ctx, "u1", domain.TaxCode{Code: "PPH-2", Name: "PPh 23 2%", Kind: domain.TaxWithholding, Scope: domain.ScopeBoth, Rate: rp(2), BaseNumerator: 1, BaseDenominator: 1, AccountSaleID: h.id("1104002"), AccountPurchaseID: h.id("2103002"), IsActive: true})
	h.ok(err)
	swap, err := setup.SaveFiscalPosition(h.ctx, "u1", domain.FiscalPosition{Name: "Tanpa NPWP", IsActive: true, Rules: []domain.FiscalRule{{FromTaxID: base.ID, ToTaxID: wh.ID}}})
	h.ok(err)
	_, err = setup.SavePartyProfile(h.ctx, "u1", domain.PartyProfile{Side: domain.SideSupplier, Name: "CV Tiga", IsPKP: true, FiscalPositionID: swap.ID})
	h.ok(err)
	three := doc
	three.PartyName, three.TaxIDs = "CV Tiga", []string{base.ID}
	d, err = s.Create(h.ctx, "u1", three)
	h.ok(err)
	if d.Withholding != rp(40_000) || d.Taxes[0].Code != "PPH-4" {
		t.Fatalf("swap %+v", d.Taxes)
	}
}

func TestCreditNote(t *testing.T) {
	h := newHarness(t)
	l := app.NewLedgers(h.deps)
	setup := app.NewTaxSetup(h.deps)
	ppn := h.vat("2103003", "1104001")
	inv, err := l.CreateInvoice(h.ctx, domain.KindAR, "u1", app.InvoiceInput{InvoiceDate: today, PartyName: "PT A", AccountID: h.id("4101001"), Subtotal: rp(10_000_000), TaxIDs: []string{ppn}})
	h.ok(err)
	_, err = l.CreateCreditNote(h.ctx, "u1", inv.ID, app.CreditNoteInput{Date: "", Amount: rp(1), Reason: "x"})
	h.wantErr(err, "Credit note butuh tagihan yang sudah diposting dan punya jurnal")
	_, _, err = l.PostInvoice(h.ctx, domain.KindAR, "u1", inv.ID)
	h.ok(err)
	_, err = l.CreateCreditNote(h.ctx, "u1", inv.ID, app.CreditNoteInput{Date: "", Amount: rp(1), Reason: " "})
	h.wantErr(err, "Alasan wajib diisi")
	_, err = l.CreateCreditNote(h.ctx, "u1", inv.ID, app.CreditNoteInput{Date: "", Amount: rp(99_000_000), Reason: "x"})
	h.wantErr(err, "Nilai credit note melebihi tagihan yang belum dikreditkan")

	// A quarter comes back on every line.
	after, err := l.CreateCreditNote(h.ctx, "u1", inv.ID, app.CreditNoteInput{Date: today, Amount: rp(2_775_000), Reason: "Revisi nilai"})
	h.ok(err)
	if after.Outstanding() != rp(8_325_000) {
		t.Fatalf("outstanding %d", after.Outstanding())
	}
	reg, err := setup.TaxRegister(h.ctx, "2026-10-01", "2026-10-31")
	h.ok(err)
	if reg.OutputVAT != rp(825_000) {
		t.Fatalf("register %+v", reg)
	}
	bs, err := app.NewReports(h.deps).BalanceSheet(h.ctx, today)
	h.ok(err)
	if bs.TotalAssets != bs.TotalLiabilitiesAndEquity {
		t.Fatalf("balance sheet off: %d vs %d", bs.TotalAssets, bs.TotalLiabilitiesAndEquity)
	}
	pays, _, err := l.ListPayments(h.ctx, domain.PaymentFilter{Kind: domain.KindAR, Limit: 10})
	h.ok(err)
	if len(pays) != 1 || pays[0].Method != "credit_note" {
		t.Fatalf("payments %+v", pays)
	}
	_, err = l.VoidPayment(h.ctx, domain.KindAR, "u1", pays[0].ID, "salah")
	h.wantErr(err, "Credit note tidak bisa dibatalkan. Buat tagihan baru bila perlu.")
	// The rest closes the invoice.
	closed, err := l.CreateCreditNote(h.ctx, "u1", inv.ID, app.CreditNoteInput{Date: today, Amount: 0, Reason: "Batal"})
	h.ok(err)
	if closed.Outstanding() != 0 {
		t.Fatalf("closed %d", closed.Outstanding())
	}
}

func TestCashBasisVAT(t *testing.T) {
	h := newHarness(t)
	l := app.NewLedgers(h.deps)
	setup := app.NewTaxSetup(h.deps)
	transition := h.id("2103002")
	code, err := setup.SaveTaxCode(h.ctx, "u1", domain.TaxCode{Code: "PPN-KAS", Name: "PPN kas", Kind: domain.TaxVAT, Scope: domain.ScopeBoth, Rate: rp(11), BaseNumerator: 1, BaseDenominator: 1,
		AccountSaleID: h.id("2103003"), AccountPurchaseID: h.id("1104001"), CashBasis: true, TransitionAccountID: transition, IsActive: true})
	h.ok(err)
	_, err = setup.SaveTaxCode(h.ctx, "u1", domain.TaxCode{Code: "X", Name: "X", Kind: domain.TaxVAT, Scope: domain.ScopeBoth, Rate: rp(11), BaseNumerator: 1, BaseDenominator: 1, AccountSaleID: h.id("2103003"), AccountPurchaseID: h.id("1104001"), CashBasis: true, IsActive: true})
	h.wantErr(err, "PPN berbasis kas perlu akun transisi dan hanya untuk PPN")

	inv, err := l.CreateInvoice(h.ctx, domain.KindAR, "u1", app.InvoiceInput{InvoiceDate: today, PartyName: "PT K", AccountID: h.id("4101001"), Subtotal: rp(1_000_000), TaxIDs: []string{code.ID}})
	h.ok(err)
	posted, _, err := l.PostInvoice(h.ctx, domain.KindAR, "u1", inv.ID)
	h.ok(err)
	e, err := app.NewJournals(h.deps).Get(h.ctx, posted.JournalEntryID)
	h.ok(err)
	if lineOn(e, transition).Amount != rp(110_000) || lineOn(e, h.id("2103003")).Amount != 0 {
		t.Fatalf("VAT must wait on the transition account: %+v", e.Lines)
	}
	reg, err := setup.TaxRegister(h.ctx, "2026-10-01", "2026-10-31")
	h.ok(err)
	if reg.OutputVAT != 0 {
		t.Fatalf("recognized before payment: %+v", reg)
	}

	// Half paid recognizes half.
	res, err := l.CreatePayment(h.ctx, domain.KindAR, "u1", app.PaymentInput{PaymentDate: today, CashAccountID: h.id("1101001"), Allocations: []domain.Allocation{{InvoiceID: inv.ID, Amount: rp(555_000)}}})
	h.ok(err)
	reg, err = setup.TaxRegister(h.ctx, "2026-10-01", "2026-10-31")
	h.ok(err)
	if reg.OutputVAT != rp(55_000) {
		t.Fatalf("half recognized: %+v (%s)", reg, res.Note)
	}
	// A void takes it back.
	_, err = l.VoidPayment(h.ctx, domain.KindAR, "u1", res.Payment.ID, "Salah catat")
	h.ok(err)
	reg, err = setup.TaxRegister(h.ctx, "2026-10-01", "2026-10-31")
	h.ok(err)
	if reg.OutputVAT != 0 {
		t.Fatalf("void kept the VAT: %+v", reg)
	}
}

func TestTaxExport(t *testing.T) {
	h := newHarness(t)
	l := app.NewLedgers(h.deps)
	setup := app.NewTaxSetup(h.deps)
	ppn := h.vat("2103003", "1104001")
	_, err := setup.SavePartyProfile(h.ctx, "u1", domain.PartyProfile{Side: domain.SideB2B, Name: "PT Ekspor", NPWP: "0123456789012345", Address: "Jakarta", IsPKP: true})
	h.ok(err)
	inv, err := l.CreateInvoice(h.ctx, domain.KindAR, "u1", app.InvoiceInput{InvoiceDate: today, PartyName: "PT Ekspor", AccountID: h.id("4101001"), Subtotal: rp(1_000_000), TaxIDs: []string{ppn}})
	h.ok(err)
	_, _, err = l.PostInvoice(h.ctx, domain.KindAR, "u1", inv.ID)
	h.ok(err)
	name, data, err := setup.Export(h.ctx, app.ExportEFaktur, "2026-10-01", "2026-10-31")
	h.ok(err)
	out := string(data)
	if name != "efaktur-2026-10-01-2026-10-31.csv" || !strings.Contains(out, "0123456789012345") || !strings.Contains(out, "PT Ekspor") || !strings.Contains(out, "110000.00") {
		t.Fatalf("%s: %s", name, out)
	}
	_, _, err = setup.Export(h.ctx, "lain", "", "")
	h.wantErr(err, "Jenis ekspor tidak valid")
}

func TestMasterDeleteGuards(t *testing.T) {
	h := newHarness(t)
	setup := app.NewTaxSetup(h.deps)
	docs := app.NewDocuments(h.deps)
	ppn := h.vat("2103003", "1104001")

	free, err := setup.SaveTaxCode(h.ctx, "u1", domain.TaxCode{Code: "BEBAS", Name: "Bebas", Kind: domain.TaxVAT, Scope: domain.ScopeBoth, Rate: rp(1), BaseNumerator: 1, BaseDenominator: 1, AccountSaleID: h.id("2103003"), AccountPurchaseID: h.id("1104001"), IsActive: true})
	h.ok(err)
	h.ok(setup.DeleteTaxCode(h.ctx, free.ID))

	vendor, err := docs.SaveVendor(h.ctx, "u1", domain.Vendor{Name: "CV Pakai", IsActive: true})
	h.ok(err)
	_, err = docs.Create(h.ctx, "u1", domain.Document{Kind: domain.DocInvoice, Side: domain.SideSupplier, Title: "T", VendorID: vendor.ID, PartyName: "CV Pakai", IssueDate: today, Items: []domain.DocItem{item("X", 1, 1_000)}, TaxIDs: []string{ppn}})
	h.ok(err)
	h.wantErr(setup.DeleteTaxCode(h.ctx, ppn), "Kode pajak sudah dipakai. Nonaktifkan saja.")
	h.wantErr(docs.DeleteVendor(h.ctx, vendor.ID), "Vendor sudah dipakai dokumen. Nonaktifkan saja.")

	fp, err := setup.SaveFiscalPosition(h.ctx, "u1", domain.FiscalPosition{Name: "FP", IsActive: true})
	h.ok(err)
	prof, err := setup.SavePartyProfile(h.ctx, "u1", domain.PartyProfile{Side: domain.SideB2B, Name: "PT P", IsPKP: true, FiscalPositionID: fp.ID})
	h.ok(err)
	h.wantErr(setup.DeleteFiscalPosition(h.ctx, fp.ID), "Fiscal position masih dipakai profil pihak. Nonaktifkan saja.")
	h.ok(setup.DeletePartyProfile(h.ctx, prof.ID))
	h.ok(setup.DeleteFiscalPosition(h.ctx, fp.ID))
}

func TestUpdateSchedules(t *testing.T) {
	h := newHarness(t)
	s := app.NewSchedules(h.deps)
	lines := []domain.Line{line(h.id("6301004"), domain.Debit, 100), line(h.id("1101001"), domain.Credit, 100)}
	r, err := s.CreateRecurring(h.ctx, "u1", app.RecurringInput{Name: "Sewa", Frequency: domain.FreqMonthly, StartDate: "2026-10-05", Lines: lines})
	h.ok(err)
	changed, err := s.UpdateRecurring(h.ctx, "u1", r.ID, app.RecurringInput{Name: "Sewa kantor", Frequency: domain.FreqQuarterly, StartDate: "2026-10-20", Lines: lines})
	h.ok(err)
	if changed.Name != "Sewa kantor" || changed.NextRun != "2026-10-20" || changed.AnchorDay != 20 {
		t.Fatalf("%+v", changed)
	}
	_, err = s.RunDue(h.ctx)
	h.ok(err)
	ran, err := h.deps.Store.GetRecurringJournal(h.ctx, r.ID)
	h.ok(err)
	_, err = s.UpdateRecurring(h.ctx, "u1", r.ID, app.RecurringInput{Name: "Sewa kantor", Frequency: domain.FreqQuarterly, StartDate: "2026-10-01", Lines: lines})
	if ran.LastEntryID == "" || err == nil {
		t.Fatalf("a schedule that ran moved backward: %+v, %v", ran, err)
	}

	asset := app.AssetInput{Name: "Laptop", AcquiredOn: "2026-10-01", Cost: rp(12_000_000), LifeMonths: 12, ExpenseAccountID: h.id("6301004"), AccumAccountID: h.id("1101001")}
	a, err := s.CreateAsset(h.ctx, "u1", asset)
	h.ok(err)
	asset.LifeMonths = 24
	a, err = s.UpdateAsset(h.ctx, "u1", a.ID, asset)
	h.ok(err)
	if a.LifeMonths != 24 {
		t.Fatalf("%+v", a)
	}
	// October has not ended on the pinned date, so record the first month directly.
	a.DepreciatedMonths = 1
	h.ok(h.deps.Store.UpdateFixedAssetRun(h.ctx, *a))
	asset.Cost = rp(1)
	_, err = s.UpdateAsset(h.ctx, "u1", a.ID, asset)
	h.wantErr(err, "Aset yang sudah disusutkan hanya bisa diubah nama dan akunnya")
	asset.Cost, asset.Name = rp(12_000_000), "Laptop kantor"
	a, err = s.UpdateAsset(h.ctx, "u1", a.ID, asset)
	h.ok(err)
	if a.Name != "Laptop kantor" {
		t.Fatalf("%+v", a)
	}
}

func TestProjectProfitMatchesIncomeStatement(t *testing.T) {
	h := newHarness(t)
	tags := app.NewProjectTags(h.deps)
	l := app.NewLedgers(h.deps)
	j := app.NewJournals(h.deps)
	a, err := tags.Save(h.ctx, "u1", domain.ProjectTag{Name: "Kampanye Oktober", ClientName: "PT A", IsActive: true})
	h.ok(err)
	_, err = tags.Save(h.ctx, "u1", domain.ProjectTag{Name: "kampanye oktober", ClientName: "pt a", IsActive: true})
	h.wantErr(err, "Proyek ini sudah ada untuk klien tersebut")

	inv, err := l.CreateInvoice(h.ctx, domain.KindAR, "u1", app.InvoiceInput{InvoiceDate: today, PartyName: "PT A", AccountID: h.id("4101001"), Subtotal: rp(10_000_000), TagID: a.ID})
	h.ok(err)
	_, _, err = l.PostInvoice(h.ctx, domain.KindAR, "u1", inv.ID)
	h.ok(err)
	_, err = j.Create(h.ctx, "u1", app.EntryInput{EntryDate: today, Description: "Biaya produksi", TagID: a.ID, Post: true,
		Lines: []domain.Line{line(h.id("6301004"), domain.Debit, 3_000_000), line(h.id("1101001"), domain.Credit, 3_000_000)}})
	h.ok(err)
	_, err = j.Create(h.ctx, "u1", app.EntryInput{EntryDate: today, Description: "Sewa kantor", Post: true,
		Lines: []domain.Line{line(h.id("6301004"), domain.Debit, 500_000), line(h.id("1101001"), domain.Credit, 500_000)}})
	h.ok(err)
	// A credit note takes the revenue back from the same project.
	_, err = l.CreateCreditNote(h.ctx, "u1", inv.ID, app.CreditNoteInput{Date: today, Amount: rp(1_000_000), Reason: "Revisi nilai"})
	h.ok(err)

	rows, from, to, err := tags.ProjectProfit(h.ctx, "2026-10-01", "2026-10-31", "")
	h.ok(err)
	var tagged *domain.ProjectProfitRow
	var profit domain.Money
	for i := range rows {
		profit += rows[i].Revenue - rows[i].Cost
		if rows[i].TagID == a.ID {
			tagged = &rows[i]
		}
	}
	if tagged == nil || tagged.Revenue != rp(9_000_000) || tagged.Cost != rp(3_000_000) || tagged.ClientName != "PT A" {
		t.Fatalf("tagged %+v", tagged)
	}
	is, err := app.NewReports(h.deps).IncomeStatement(h.ctx, from, to)
	h.ok(err)
	if profit != is.NetIncome {
		t.Fatalf("rows add to %d, income statement says %d", profit, is.NetIncome)
	}
	only, _, _, err := tags.ProjectProfit(h.ctx, from, to, "pt a")
	h.ok(err)
	if len(only) != 1 || only[0].TagID != a.ID {
		t.Fatalf("client filter %+v", only)
	}
	h.wantErr(tags.Delete(h.ctx, a.ID), "Tag sudah dipakai transaksi. Nonaktifkan saja.")
	a.IsActive = false
	_, err = tags.Save(h.ctx, "u1", *a)
	h.ok(err)
	_, err = l.CreateInvoice(h.ctx, domain.KindAR, "u1", app.InvoiceInput{InvoiceDate: today, PartyName: "PT A", Subtotal: rp(1), TagID: a.ID})
	h.wantErr(err, "Tag proyek tidak ditemukan atau nonaktif")
}

func TestSalesInvoiceTagsItsDeal(t *testing.T) {
	h := newHarness(t)
	auto := app.NewAutoPost(h.deps)
	issue := func(id string) *domain.Invoice {
		inv, err := auto.SalesInvoiceIssued(h.ctx, domain.SalesInvoiceIssued{InvoiceID: id, InvoiceNumber: id, DealID: "deal-1", CustomerName: "PT D", IssueDate: today, Subtotal: rp(100), TotalAmount: rp(100), Description: "Termin 1 · Retainer SEO"})
		h.ok(err)
		return inv
	}
	first, second := issue("s1"), issue("s2")
	if first.TagID == "" || first.TagID != second.TagID || first.TagName != "Retainer SEO" {
		t.Fatalf("%+v %+v", first, second)
	}
}

func TestRecurringInvoices(t *testing.T) {
	h := newHarness(t)
	s := app.NewSchedules(h.deps)
	l := app.NewLedgers(h.deps)
	ppn := h.vat("2103003", "1104001")
	in := app.RecurringInvoiceInput{Kind: domain.KindAR, Name: "Retainer", PartyName: "PT R", AccountID: h.id("4101001"), Subtotal: rp(1_000_000), TaxIDs: []string{ppn}, DueDays: 14, Frequency: domain.FreqMonthly, StartDate: "2026-10-05", AutoPost: true}
	bad := in
	bad.Subtotal = 0
	_, err := s.CreateRecurringInvoice(h.ctx, "u1", bad)
	h.wantErr(err, "Subtotal harus lebih dari 0")
	r, err := s.CreateRecurringInvoice(h.ctx, "u1", in)
	h.ok(err)

	res, err := s.RunDue(h.ctx)
	h.ok(err)
	if res.Posted != 1 || res.Failed != 0 {
		t.Fatalf("run %+v", res)
	}
	r, err = h.deps.Store.GetRecurringInvoice(h.ctx, r.ID)
	h.ok(err)
	if r.NextRun != "2026-11-05" || r.LastInvoiceID == "" {
		t.Fatalf("%+v", r)
	}
	inv, err := l.GetInvoice(h.ctx, domain.KindAR, r.LastInvoiceID)
	h.ok(err)
	if inv.Status != domain.StatusPosted || inv.JournalEntryID == "" || inv.TaxAmount != rp(110_000) || inv.DueDate != "2026-10-19" {
		t.Fatalf("%+v", inv)
	}
	// Running again bills nothing new.
	res, err = s.RunDue(h.ctx)
	h.ok(err)
	if res.Posted != 0 {
		t.Fatalf("billed twice %+v", res)
	}
	back := in
	back.StartDate = "2026-10-01"
	_, err = s.UpdateRecurringInvoice(h.ctx, "u1", r.ID, back)
	h.wantErr(err, "Jadwal ini sudah berjalan, jadwal berikutnya tidak bisa mundur")

	// A closed period leaves no invoice, keeps the date due and reports why.
	closed := in
	closed.PartyName, closed.StartDate = "PT Tutup", "2026-09-05"
	c, err := s.CreateRecurringInvoice(h.ctx, "u1", closed)
	h.ok(err)
	res, err = s.RunDue(h.ctx)
	h.ok(err)
	if res.Failed != 1 {
		t.Fatalf("closed period %+v", res)
	}
	c, err = h.deps.Store.GetRecurringInvoice(h.ctx, c.ID)
	h.ok(err)
	rows, _, err := l.ListInvoices(h.ctx, domain.InvoiceFilter{Kind: domain.KindAR, PartyName: "PT Tutup", Limit: 5}, "")
	h.ok(err)
	if c.NextRun != "2026-09-05" || c.LastError == "" || len(rows) != 0 {
		t.Fatalf("%+v %d", c, len(rows))
	}
	h.ok(s.DeleteRecurringInvoice(h.ctx, c.ID))
}

func TestAdvances(t *testing.T) {
	h := newHarness(t)
	l := app.NewLedgers(h.deps)
	cash, liability := h.id("1101001"), h.id("2105001")
	in := app.AdvanceInput{Kind: domain.KindAR, Date: today, PartyName: "PT M", CashAccountID: cash, AdvanceAccountID: liability, Amount: rp(5_000_000)}
	wrong := in
	wrong.AdvanceAccountID = h.id("1103001")
	_, err := l.CreateAdvance(h.ctx, "u1", wrong)
	h.wantErr(err, "Akun uang muka harus akun liabilitas yang postable dan bukan kas")
	a, err := l.CreateAdvance(h.ctx, "u1", in)
	h.ok(err)
	if a.Remaining() != rp(5_000_000) || a.JournalEntryID == "" {
		t.Fatalf("%+v", a)
	}

	inv, err := l.CreateInvoice(h.ctx, domain.KindAR, "u1", app.InvoiceInput{InvoiceDate: today, PartyName: "PT M", AccountID: h.id("4101001"), Subtotal: rp(8_000_000)})
	h.ok(err)
	_, _, err = l.PostInvoice(h.ctx, domain.KindAR, "u1", inv.ID)
	h.ok(err)
	other, err := l.CreateInvoice(h.ctx, domain.KindAR, "u1", app.InvoiceInput{InvoiceDate: today, PartyName: "PT Lain", AccountID: h.id("4101001"), Subtotal: rp(1_000)})
	h.ok(err)
	_, _, err = l.PostInvoice(h.ctx, domain.KindAR, "u1", other.ID)
	h.ok(err)
	_, err = l.ApplyAdvance(h.ctx, "u1", a.ID, other.ID, today, rp(1_000))
	h.wantErr(err, "Uang muka hanya bisa dipakai untuk tagihan pihak yang sama")
	_, err = l.ApplyAdvance(h.ctx, "u1", a.ID, inv.ID, today, rp(6_000_000))
	h.wantErr(err, "Nilai melebihi sisa uang muka")

	res, err := l.ApplyAdvance(h.ctx, "u1", a.ID, inv.ID, today, rp(2_000_000))
	h.ok(err)
	if res.Invoices[0].Outstanding() != rp(6_000_000) || res.Payment.Method != "advance" {
		t.Fatalf("%+v", res)
	}
	e, err := app.NewJournals(h.deps).Get(h.ctx, res.Payment.JournalEntryID)
	h.ok(err)
	if lineOn(e, liability).Amount != rp(2_000_000) || lineOn(e, liability).Side != domain.Debit {
		t.Fatalf("advance account not debited: %+v", e.Lines)
	}
	a, err = h.deps.Store.GetAdvance(h.ctx, a.ID)
	h.ok(err)
	if a.Remaining() != rp(3_000_000) {
		t.Fatalf("%+v", a)
	}
	_, err = l.VoidAdvance(h.ctx, "u1", a.ID, "Salah catat")
	h.wantErr(err, "Uang muka sudah dipakai tagihan. Batalkan pemakaiannya dulu.")

	// Voiding the application frees the advance, which can then be voided.
	_, err = l.VoidPayment(h.ctx, domain.KindAR, "u1", res.Payment.ID, "Salah tagihan")
	h.ok(err)
	a, err = l.VoidAdvance(h.ctx, "u1", a.ID, "Salah catat")
	h.ok(err)
	if a.Status != domain.StatusVoid || a.Remaining() != 0 {
		t.Fatalf("%+v", a)
	}
	bs, err := app.NewReports(h.deps).BalanceSheet(h.ctx, today)
	h.ok(err)
	if bs.TotalAssets != bs.TotalLiabilitiesAndEquity {
		t.Fatalf("balance sheet off: %d vs %d", bs.TotalAssets, bs.TotalLiabilitiesAndEquity)
	}

	vendor, err := l.CreateAdvance(h.ctx, "u1", app.AdvanceInput{Kind: domain.KindAP, Date: today, PartyName: "CV V", CashAccountID: cash, AdvanceAccountID: h.id("1106001"), Amount: rp(1_000_000)})
	h.ok(err)
	bill, err := l.CreateInvoice(h.ctx, domain.KindAP, "u1", app.InvoiceInput{InvoiceDate: today, PartyName: "CV V", Subtotal: rp(1_000_000)})
	h.ok(err)
	_, _, err = l.PostInvoice(h.ctx, domain.KindAP, "u1", bill.ID)
	h.ok(err)
	paid, err := l.ApplyAdvance(h.ctx, "u1", vendor.ID, bill.ID, today, rp(1_000_000))
	h.ok(err)
	if paid.Invoices[0].Outstanding() != 0 {
		t.Fatalf("%+v", paid.Invoices[0])
	}
}

func TestManualTransactionsNeedTheirJournal(t *testing.T) {
	h := newHarness(t)
	l := app.NewLedgers(h.deps)
	// September is closed on the harness, so its journal cannot post.
	closed := "2026-09-15"
	inv, err := l.CreateInvoice(h.ctx, domain.KindAR, "u1", app.InvoiceInput{InvoiceDate: closed, PartyName: "PT T", AccountID: h.id("4101001"), Subtotal: rp(1_000)})
	h.ok(err)
	if _, _, err = l.PostInvoice(h.ctx, domain.KindAR, "u1", inv.ID); err == nil {
		t.Fatal("invoice posted without a journal")
	}
	inv, err = l.GetInvoice(h.ctx, domain.KindAR, inv.ID)
	h.ok(err)
	if inv.Status != domain.StatusDraft {
		t.Fatalf("status %s", inv.Status)
	}

	open, err := l.CreateInvoice(h.ctx, domain.KindAR, "u1", app.InvoiceInput{InvoiceDate: today, PartyName: "PT T", AccountID: h.id("4101001"), Subtotal: rp(1_000)})
	h.ok(err)
	_, _, err = l.PostInvoice(h.ctx, domain.KindAR, "u1", open.ID)
	h.ok(err)
	pay := func(date string) (*app.PaymentResult, error) {
		return l.CreatePayment(h.ctx, domain.KindAR, "u1", app.PaymentInput{PaymentDate: date, CashAccountID: h.id("1101001"), Allocations: []domain.Allocation{{InvoiceID: open.ID, Amount: rp(100)}}})
	}
	_, err = pay("2026-09-15")
	h.wantErr(err, "Tanggal pembayaran tidak boleh sebelum tanggal tagihan")
	rows, _, err := l.ListPayments(h.ctx, domain.PaymentFilter{Kind: domain.KindAR, Limit: 5})
	h.ok(err)
	if len(rows) != 0 {
		t.Fatalf("payment left behind: %+v", rows)
	}

	// An advance applied into a closed period leaves nothing behind.
	a, err := l.CreateAdvance(h.ctx, "u1", app.AdvanceInput{Kind: domain.KindAR, Date: today, PartyName: "PT T", CashAccountID: h.id("1101001"), AdvanceAccountID: h.id("2105001"), Amount: rp(500)})
	h.ok(err)
	_, err = l.ApplyAdvance(h.ctx, "u1", a.ID, open.ID, "2026-09-15", rp(100))
	h.wantErr(err, "Tanggal pemakaian tidak boleh sebelum tanggal uang muka")
	a, err = h.deps.Store.GetAdvance(h.ctx, a.ID)
	h.ok(err)
	if a.Applied != 0 {
		t.Fatalf("applied %d", a.Applied)
	}
}

func TestVoidRefusedWhenReversalCannotPost(t *testing.T) {
	h := newHarness(t)
	l := app.NewLedgers(h.deps)
	inv, err := l.CreateInvoice(h.ctx, domain.KindAR, "u1", app.InvoiceInput{InvoiceDate: today, PartyName: "PT V", AccountID: h.id("4101001"), Subtotal: rp(1_000)})
	h.ok(err)
	_, _, err = l.PostInvoice(h.ctx, domain.KindAR, "u1", inv.ID)
	h.ok(err)
	res, err := l.CreatePayment(h.ctx, domain.KindAR, "u1", app.PaymentInput{PaymentDate: today, CashAccountID: h.id("1101001"), Allocations: []domain.Allocation{{InvoiceID: inv.ID, Amount: rp(100)}}})
	h.ok(err)
	october, err := h.periodFor("2026-10")
	h.ok(err)
	h.ok(h.deps.Store.SetPeriodStatus(h.ctx, []string{october}, domain.PeriodClosed))
	if _, err = l.VoidPayment(h.ctx, domain.KindAR, "u1", res.Payment.ID, "Salah catat"); err == nil {
		t.Fatal("void passed without its reversal")
	}
	p, err := h.deps.Store.GetPayment(h.ctx, res.Payment.ID)
	h.ok(err)
	if p.Status != domain.StatusPosted {
		t.Fatalf("payment %s", p.Status)
	}
}

func TestSalesInvoiceWaitsForItsJournal(t *testing.T) {
	h := newHarness(t)
	auto := app.NewAutoPost(h.deps)
	l := app.NewLedgers(h.deps)
	reports := app.NewReports(h.deps)
	inv, err := auto.SalesInvoiceIssued(h.ctx, domain.SalesInvoiceIssued{InvoiceID: "late", InvoiceNumber: "INV-LATE", CustomerName: "PT W", IssueDate: "2026-09-15", Subtotal: rp(1_000), TaxAmount: rp(110), TotalAmount: rp(1_110)})
	h.ok(err)
	if inv.Status != domain.StatusPosted || inv.JournalEntryID != "" {
		t.Fatalf("%+v", inv)
	}
	// Sales already took this money, so an early date cannot refuse the event.
	early, err := auto.SalesPaymentRecorded(h.ctx, domain.SalesPaymentRecorded{PaymentID: "pay-early", InvoiceID: "late", Amount: rp(10), PaidOn: "2026-09-01", Method: "transfer"})
	h.ok(err)
	if early.Payment.JournalEntryID != "" {
		t.Fatalf("%+v", early.Payment)
	}
	h.ok(h.deps.Store.DeletePostingLog(h.ctx, domain.EventSaleARReceipt, early.Payment.ID))
	_, err = l.VoidPayment(h.ctx, domain.KindAR, "u1", early.Payment.ID, "Salah catat")
	h.ok(err)
	open, err := l.OpenInvoices(h.ctx, domain.KindAR)
	h.ok(err)
	if len(open) != 0 {
		t.Fatalf("unjournaled invoice in the balance: %+v", open)
	}
	_, err = l.CreatePayment(h.ctx, domain.KindAR, "u1", app.PaymentInput{PaymentDate: today, CashAccountID: h.id("1101001"), Allocations: []domain.Allocation{{InvoiceID: inv.ID, Amount: rp(10)}}})
	h.wantErr(err, "Tagihan "+inv.InvoiceNo+" belum berjurnal. Selesaikan dulu di log posting.")
	rows, pending, err := reports.SubledgerRecon(h.ctx)
	h.ok(err)
	if rows[0].Kind != domain.KindAR || rows[0].UnjournaledCount != 1 || rows[0].UnjournaledAmount != rp(1_110) || rows[0].Subledger != 0 || pending != 1 {
		t.Fatalf("%+v pending %d", rows[0], pending)
	}
	september, err := h.periodFor("2026-09")
	h.ok(err)
	preview, err := app.NewFiscal(h.deps).ClosePreview(h.ctx, september)
	h.ok(err)
	blocked := false
	for _, b := range preview.Blockers {
		blocked = blocked || strings.Contains(b, "belum berjurnal")
	}
	if !blocked {
		t.Fatalf("%+v", preview.Blockers)
	}

	// Once the period opens the daily run books it and the ledgers tie out.
	h.ok(h.deps.Store.SetPeriodStatus(h.ctx, []string{september}, domain.PeriodOpen))
	_, err = app.NewSchedules(h.deps).RunDue(h.ctx)
	h.ok(err)
	inv, err = l.GetInvoice(h.ctx, domain.KindAR, inv.ID)
	h.ok(err)
	if inv.JournalEntryID == "" {
		t.Fatalf("%+v", inv)
	}
	rows, pending, err = reports.SubledgerRecon(h.ctx)
	h.ok(err)
	if rows[0].UnjournaledCount != 0 || rows[0].Subledger != rp(1_110) || rows[0].Difference() != 0 || pending != 0 {
		t.Fatalf("%+v pending %d", rows[0], pending)
	}
}

func TestControlAccountsStayOutOfManualEntries(t *testing.T) {
	h := newHarness(t)
	j := app.NewJournals(h.deps)
	lines := []domain.Line{line(h.id("1103001"), domain.Debit, 100), line(h.id("1101001"), domain.Credit, 100)}
	_, err := j.Create(h.ctx, "u1", app.EntryInput{EntryDate: today, Description: "Langsung ke piutang", Lines: lines})
	h.wantErr(err, "Akun piutang atau hutang usaha diisi lewat tagihan dan pembayaran. Untuk koreksi, tandai jurnal ini sebagai koreksi (recon).")
	_, err = j.Create(h.ctx, "u1", app.EntryInput{EntryDate: today, Description: "Koreksi saldo", IsRecon: true, Post: true, Lines: lines})
	h.ok(err)
	rows, _, err := app.NewReports(h.deps).SubledgerRecon(h.ctx)
	h.ok(err)
	if rows[0].Recon != rp(100) || rows[0].Difference() != rp(100) {
		t.Fatalf("%+v", rows[0])
	}
}

func settlementHarness(t *testing.T, kind, party string) (*harness, *app.Ledgers, *domain.Invoice) {
	t.Helper()
	h := newHarness(t)
	l := app.NewLedgers(h.deps)
	in := app.InvoiceInput{InvoiceDate: today, PartyName: party, Subtotal: rp(10_000_000)}
	if kind == domain.KindAR {
		in.AccountID = h.id("4101001")
	}
	inv, err := l.CreateInvoice(h.ctx, kind, "u1", in)
	h.ok(err)
	_, _, err = l.PostInvoice(h.ctx, kind, "u1", inv.ID)
	h.ok(err)
	inv, err = l.GetInvoice(h.ctx, kind, inv.ID)
	h.ok(err)
	return h, l, inv
}

func TestReceiptWithFeeAndWithheldTax(t *testing.T) {
	h, l, inv := settlementHarness(t, domain.KindAR, "PT Potong")
	setup := app.NewTaxSetup(h.deps)
	pph, err := setup.SaveTaxCode(h.ctx, "u1", domain.TaxCode{Code: "PPH-2B", Name: "PPh 23 2%", Kind: domain.TaxWithholding, Scope: domain.ScopeBoth, Rate: rp(2), BaseNumerator: 1, BaseDenominator: 1, AccountSaleID: h.id("1104002"), AccountPurchaseID: h.id("2103002"), IsActive: true})
	h.ok(err)
	bank, fee := h.id("1102001"), h.id("8101001")
	in := app.PaymentInput{PaymentDate: today, CashAccountID: bank, Allocations: []domain.Allocation{{InvoiceID: inv.ID, Amount: inv.TotalAmount}},
		Adjustments: []app.AdjustmentInput{{Kind: domain.AdjustFee, AccountID: fee, Amount: rp(6_500)}, {Kind: domain.AdjustWithholding, TaxID: pph.ID, Amount: rp(200_000)}}}
	bad := in
	bad.Adjustments = []app.AdjustmentInput{{Kind: domain.AdjustFee, AccountID: h.id("1103001"), Amount: rp(1)}}
	_, err = l.CreatePayment(h.ctx, domain.KindAR, "u1", bad)
	h.wantErr(err, "Akun selisih harus postable, bukan kas, dan bukan akun piutang atau hutang")
	all := in
	all.Adjustments = []app.AdjustmentInput{{Kind: domain.AdjustFee, AccountID: fee, Amount: inv.TotalAmount}}
	_, err = l.CreatePayment(h.ctx, domain.KindAR, "u1", all)
	h.wantErr(err, "Selisih tidak boleh menghabiskan seluruh nilai pembayaran")

	res, err := l.CreatePayment(h.ctx, domain.KindAR, "u1", in)
	h.ok(err)
	if res.Invoices[0].Outstanding() != 0 || len(res.Payment.Adjustments) != 2 {
		t.Fatalf("%+v", res)
	}
	e, err := app.NewJournals(h.deps).Get(h.ctx, res.Payment.JournalEntryID)
	h.ok(err)
	if lineOn(e, bank).Amount != rp(9_793_500) || lineOn(e, fee).Amount != rp(6_500) || lineOn(e, h.id("1104002")).Amount != rp(200_000) || lineOn(e, h.id("1103001")).Amount != rp(10_000_000) {
		t.Fatalf("%+v", e.Lines)
	}
	reg, err := setup.TaxRegister(h.ctx, "2026-10-01", "2026-10-31")
	h.ok(err)
	if reg.WithheldByClients != rp(200_000) {
		t.Fatalf("%+v", reg)
	}
	_, data, err := setup.Export(h.ctx, app.ExportBupotMasuk, "2026-10-01", "2026-10-31")
	h.ok(err)
	if !strings.Contains(string(data), "10000000.00") {
		t.Fatalf("base missing: %s", data)
	}

	// A void restores the invoice, the tax ledger and the books.
	_, err = l.VoidPayment(h.ctx, domain.KindAR, "u1", res.Payment.ID, "Salah catat")
	h.ok(err)
	reg, err = setup.TaxRegister(h.ctx, "2026-10-01", "2026-10-31")
	h.ok(err)
	_, data, err = setup.Export(h.ctx, app.ExportBupotMasuk, "2026-10-01", "2026-10-31")
	h.ok(err)
	if reg.WithheldByClients != 0 || strings.Contains(string(data), "10000000.00") {
		t.Fatalf("%+v %s", reg, data)
	}
	bs, err := app.NewReports(h.deps).BalanceSheet(h.ctx, today)
	h.ok(err)
	if bs.TotalAssets != bs.TotalLiabilitiesAndEquity {
		t.Fatalf("balance sheet off: %d vs %d", bs.TotalAssets, bs.TotalLiabilitiesAndEquity)
	}
	rows, _, err := app.NewReports(h.deps).SubledgerRecon(h.ctx)
	h.ok(err)
	if rows[0].Difference() != 0 {
		t.Fatalf("%+v", rows[0])
	}
}

func TestVendorPaymentWithFeeAndOverpayment(t *testing.T) {
	h, l, inv := settlementHarness(t, domain.KindAP, "CV Lebih")
	bank, fee, advance := h.id("1102001"), h.id("8101001"), h.id("1106001")
	res, err := l.CreatePayment(h.ctx, domain.KindAP, "u1", app.PaymentInput{PaymentDate: today, CashAccountID: bank, Allocations: []domain.Allocation{{InvoiceID: inv.ID, Amount: inv.TotalAmount}},
		Adjustments: []app.AdjustmentInput{{Kind: domain.AdjustFee, AccountID: fee, Amount: rp(6_500)}}, Excess: rp(500_000), ExcessAccountID: advance})
	h.ok(err)
	e, err := app.NewJournals(h.deps).Get(h.ctx, res.Payment.JournalEntryID)
	h.ok(err)
	if lineOn(e, bank).Amount != rp(10_506_500) || lineOn(e, advance).Side != domain.Debit || lineOn(e, bank).Side != domain.Credit || lineOn(e, fee).Amount != rp(6_500) || lineOn(e, fee).Side != domain.Debit {
		t.Fatalf("%+v", e.Lines)
	}
	rows, err := l.ListAdvances(h.ctx, domain.KindAP)
	h.ok(err)
	if len(rows) != 1 || rows[0].Remaining() != rp(500_000) || rows[0].PartyName != "CV Lebih" || rows[0].Reference != res.Payment.PaymentNo {
		t.Fatalf("%+v", rows)
	}
}

func TestAdvanceRefund(t *testing.T) {
	h := newHarness(t)
	l := app.NewLedgers(h.deps)
	cash, liability := h.id("1102001"), h.id("2105001")
	a, err := l.CreateAdvance(h.ctx, "u1", app.AdvanceInput{Kind: domain.KindAR, Date: today, PartyName: "PT Balik", CashAccountID: cash, AdvanceAccountID: liability, Amount: rp(1_000_000)})
	h.ok(err)
	refund := app.RefundInput{Date: today, CashAccountID: cash, Reason: "Proyek batal", Amount: rp(2_000_000)}
	_, err = l.RefundAdvance(h.ctx, "u1", a.ID, refund)
	h.wantErr(err, "Nilai melebihi sisa uang muka")
	refund.Amount = rp(400_000)
	a, err = l.RefundAdvance(h.ctx, "u1", a.ID, refund)
	h.ok(err)
	if a.Remaining() != rp(600_000) || a.Refunded != rp(400_000) {
		t.Fatalf("%+v", a)
	}
	_, err = l.VoidAdvance(h.ctx, "u1", a.ID, "Salah catat")
	h.wantErr(err, "Uang muka sudah dikembalikan sebagian. Batalkan pengembaliannya dulu.")
	r := a.Refunds[0]
	e, err := app.NewJournals(h.deps).Get(h.ctx, r.JournalEntryID)
	h.ok(err)
	if lineOn(e, liability).Side != domain.Debit || lineOn(e, cash).Side != domain.Credit || lineOn(e, cash).Amount != rp(400_000) {
		t.Fatalf("%+v", e.Lines)
	}
	a, err = l.VoidAdvanceRefund(h.ctx, "u1", r.ID, "Salah nilai")
	h.ok(err)
	if a.Remaining() != rp(1_000_000) {
		t.Fatalf("%+v", a)
	}
	_, err = l.VoidAdvanceRefund(h.ctx, "u1", r.ID, "Salah nilai")
	h.wantErr(err, "Pengembalian ini sudah dibatalkan")
	a, err = l.VoidAdvance(h.ctx, "u1", a.ID, "Salah catat")
	h.ok(err)
	bs, err := app.NewReports(h.deps).BalanceSheet(h.ctx, today)
	h.ok(err)
	if a.Status != domain.StatusVoid || bs.TotalAssets != bs.TotalLiabilitiesAndEquity {
		t.Fatalf("%+v %d %d", a, bs.TotalAssets, bs.TotalLiabilitiesAndEquity)
	}
}

func TestExcessSharesOneJournalAndVoidsWithPayment(t *testing.T) {
	h, l, inv := settlementHarness(t, domain.KindAR, "PT Lebih")
	bank, liability := h.id("1102001"), h.id("2105001")
	res, err := l.CreatePayment(h.ctx, domain.KindAR, "u1", app.PaymentInput{PaymentDate: today, CashAccountID: bank, Excess: rp(300_000), ExcessAccountID: liability, Allocations: []domain.Allocation{{InvoiceID: inv.ID, Amount: inv.TotalAmount}}})
	h.ok(err)
	e, err := app.NewJournals(h.deps).Get(h.ctx, res.Payment.JournalEntryID)
	h.ok(err)
	bankLines := 0
	for _, ln := range e.Lines {
		if ln.AccountID == bank {
			bankLines++
			if ln.Amount != inv.TotalAmount+rp(300_000) {
				t.Fatalf("bank line %s", ln.Amount)
			}
		}
	}
	if bankLines != 1 {
		t.Fatalf("bank lines %d: %+v", bankLines, e.Lines)
	}
	a, err := h.deps.Store.AdvanceBySourcePayment(h.ctx, res.Payment.ID)
	h.ok(err)
	if a == nil || a.Remaining() != rp(300_000) || a.JournalEntryID != e.ID {
		t.Fatalf("advance %+v", a)
	}
	_, err = l.VoidAdvance(h.ctx, "u1", a.ID, "Salah catat")
	h.wantErr(err, "Uang muka ini lahir dari "+res.Payment.PaymentNo+". Batalkan dokumen itu.")
	a, err = l.RefundAdvance(h.ctx, "u1", a.ID, app.RefundInput{Date: today, CashAccountID: bank, Reason: "Dikembalikan", Amount: rp(100_000)})
	h.ok(err)
	_, err = l.VoidPayment(h.ctx, domain.KindAR, "u1", res.Payment.ID, "Salah catat")
	h.wantErr(err, "Uang muka "+a.AdvanceNo+" dari kelebihan bayar sudah dipakai atau dikembalikan. Batalkan pemakaiannya dulu.")
	_, err = l.VoidAdvanceRefund(h.ctx, "u1", a.Refunds[0].ID, "Salah nilai")
	h.ok(err)
	_, err = l.VoidPayment(h.ctx, domain.KindAR, "u1", res.Payment.ID, "Salah catat")
	h.ok(err)
	a, err = h.deps.Store.GetAdvance(h.ctx, a.ID)
	h.ok(err)
	if a.Status != domain.StatusVoid || a.Remaining() != 0 {
		t.Fatalf("advance after void %+v", a)
	}
}

func TestCreditNoteOnPaidInvoiceBecomesAdvance(t *testing.T) {
	h, l, inv := settlementHarness(t, domain.KindAR, "PT Retur")
	_, err := l.CreatePayment(h.ctx, domain.KindAR, "u1", app.PaymentInput{PaymentDate: today, CashAccountID: h.id("1102001"), Allocations: []domain.Allocation{{InvoiceID: inv.ID, Amount: inv.TotalAmount}}})
	h.ok(err)
	in := app.CreditNoteInput{Date: today, Reason: "Retur", Amount: inv.TotalAmount / 4}
	_, err = l.CreateCreditNote(h.ctx, "u1", inv.ID, in)
	h.wantErr(err, "Akun uang muka harus akun liabilitas yang postable dan bukan kas")
	in.AdvanceAccountID = h.id("2105001")
	after, err := l.CreateCreditNote(h.ctx, "u1", inv.ID, in)
	h.ok(err)
	if after.Outstanding() != 0 {
		t.Fatalf("outstanding %s", after.Outstanding())
	}
	rows, err := l.ListAdvances(h.ctx, domain.KindAR)
	h.ok(err)
	var credit *domain.Advance
	for i := range rows {
		if rows[i].PartyName == "PT Retur" {
			credit = &rows[i]
		}
	}
	if credit == nil || credit.Remaining() != in.Amount || credit.JournalEntryID == "" {
		t.Fatalf("credit %+v", credit)
	}
	e, err := app.NewJournals(h.deps).Get(h.ctx, credit.JournalEntryID)
	h.ok(err)
	for _, ln := range e.Lines {
		if ln.AccountID == h.id("1103001") {
			t.Fatalf("paid share must leave the receivable alone: %+v", e.Lines)
		}
	}
	in.Amount = inv.TotalAmount
	_, err = l.CreateCreditNote(h.ctx, "u1", inv.ID, in)
	h.wantErr(err, "Nilai credit note melebihi tagihan yang belum dikreditkan")
}

func TestSalesPaymentKeepsFeeWithheldAndExcess(t *testing.T) {
	h := newHarness(t)
	auto := app.NewAutoPost(h.deps)
	inv, err := auto.SalesInvoiceIssued(h.ctx, domain.SalesInvoiceIssued{InvoiceID: "si-x", InvoiceNumber: "INV-X", CustomerName: "PT Sales", IssueDate: today, Subtotal: rp(1_000_000), TotalAmount: rp(1_000_000)})
	h.ok(err)
	setup := app.NewTaxSetup(h.deps)
	_, err = setup.SaveTaxCode(h.ctx, "u1", domain.TaxCode{Code: "PPH-S", Name: "PPh 23 2%", Kind: domain.TaxWithholding, Scope: domain.ScopeBoth, Rate: rp(2), BaseNumerator: 1, BaseDenominator: 1, AccountSaleID: h.id("1104002"), AccountPurchaseID: h.id("2103002"), IsActive: true})
	h.ok(err)
	res, err := auto.SalesPaymentRecorded(h.ctx, domain.SalesPaymentRecorded{PaymentID: "sp-x", InvoiceID: "si-x", Amount: rp(1_050_000), PaidOn: today, Method: "transfer", BankFee: rp(6_500), Withheld: rp(20_000), WithheldTaxCode: "PPH-S"})
	h.ok(err)
	if len(res.Payment.Adjustments) != 3 || res.Payment.Amount != inv.TotalAmount {
		t.Fatalf("%+v", res.Payment)
	}
	if got := domain.CashMoved(domain.KindAR, res.Payment.Amount, res.Payment.Adjustments); got != rp(1_023_500) {
		t.Fatalf("cash %s", got)
	}

	// An unknown tax code leaves that share open on the invoice.
	open, err := auto.SalesInvoiceIssued(h.ctx, domain.SalesInvoiceIssued{InvoiceID: "si-y", InvoiceNumber: "INV-Y", CustomerName: "PT Sales", IssueDate: today, Subtotal: rp(500_000), TotalAmount: rp(500_000)})
	h.ok(err)
	res, err = auto.SalesPaymentRecorded(h.ctx, domain.SalesPaymentRecorded{PaymentID: "sp-y", InvoiceID: "si-y", Amount: open.TotalAmount, PaidOn: today, Method: "transfer", Withheld: rp(10_000), WithheldTaxCode: "TIDAK-ADA"})
	h.ok(err)
	if len(res.Payment.Adjustments) != 0 || res.Payment.Amount != rp(490_000) {
		t.Fatalf("%+v", res.Payment)
	}

	// A payment on a settled invoice waits as an advance, once.
	for range 2 {
		_, err = auto.SalesPaymentRecorded(h.ctx, domain.SalesPaymentRecorded{PaymentID: "sp-z", InvoiceID: "si-x", Amount: rp(25_000), PaidOn: today, Method: "transfer"})
		h.ok(err)
	}
	a, err := h.deps.Store.AdvanceBySalesPayment(h.ctx, "sp-z")
	h.ok(err)
	if a == nil || a.Amount != rp(25_000) || a.PartyName != "PT Sales" {
		t.Fatalf("advance %+v", a)
	}
}

func TestDeferralRecognizesMonthByMonth(t *testing.T) {
	h := newHarness(t)
	l := app.NewLedgers(h.deps)
	in := domain.Deferral{Kind: domain.DeferRevenue, Name: "Retainer Q4", DeferredAccountID: h.id("4101001"), TargetAccountID: h.id("4101001"), StartDate: "2026-10-01", Months: 3, Amount: rp(1_000_000)}
	_, err := l.CreateDeferral(h.ctx, "u1", in)
	h.wantErr(err, "Akun di muka tidak sesuai dengan jenis jadwal")
	in.DeferredAccountID = h.id("2105001")
	d, err := l.CreateDeferral(h.ctx, "u1", in)
	h.ok(err)
	if d.AmountFor(1)+d.AmountFor(2)+d.AmountFor(3) != rp(1_000_000) {
		t.Fatalf("months must add up: %s %s %s", d.AmountFor(1), d.AmountFor(2), d.AmountFor(3))
	}
	_, err = app.NewSchedules(h.deps).RunDue(h.ctx)
	h.ok(err)
	after, err := h.deps.Store.GetDeferral(h.ctx, d.ID)
	h.ok(err)
	if after.PostedMonths != 0 {
		// October has not ended on the harness clock.
		t.Fatalf("%+v", after)
	}
	_, err = l.CreateDeferral(h.ctx, "u1", domain.Deferral{Kind: "X", Name: "n", Months: 1, Amount: rp(1), StartDate: "2026-10-01"})
	h.wantErr(err, "Jenis jadwal tidak valid")

	// November: October has ended and its period is open.
	later := h.deps
	later.Clock = clock.At("2026-11-05T03:00:00Z")
	res, err := app.NewSchedules(later).RunDue(h.ctx)
	h.ok(err)
	after, err = h.deps.Store.GetDeferral(h.ctx, d.ID)
	h.ok(err)
	if res.Posted != 1 || after.PostedMonths != 1 || after.Recognized() != d.AmountFor(1) {
		t.Fatalf("%+v %+v", res, after)
	}
	e, err := app.NewJournals(h.deps).Get(h.ctx, after.LastEntryID)
	h.ok(err)
	if lineOn(e, h.id("2105001")).Side != domain.Debit || lineOn(e, h.id("4101001")).Side != domain.Credit {
		t.Fatalf("%+v", e.Lines)
	}
	err = l.DeleteDeferral(h.ctx, d.ID)
	h.wantErr(err, "Jadwal yang sudah berjalan tidak bisa dihapus. Hentikan saja.")
}

func TestDeferralPostsPastMonths(t *testing.T) {
	h := newHarness(t)
	l := app.NewLedgers(h.deps)
	d, err := l.CreateDeferral(h.ctx, "u1", domain.Deferral{Kind: domain.DeferExpense, Name: "Sewa", DeferredAccountID: h.id("1105001"), TargetAccountID: h.id("8101001"), StartDate: "2026-01-01", Months: 1, Amount: rp(100)})
	h.ok(err)
	res, err := app.NewSchedules(h.deps).RunDue(h.ctx)
	h.ok(err)
	after, err := h.deps.Store.GetDeferral(h.ctx, d.ID)
	h.ok(err)
	// January is not an open period here, so the run records why it stopped.
	if res.Failed == 0 || after.PostedMonths != 0 || after.LastError == "" {
		t.Fatalf("%+v %+v", res, after)
	}
}

func TestWriteOffAndRecovery(t *testing.T) {
	h, l, inv := settlementHarness(t, domain.KindAR, "PT Macet")
	expense := h.id("8102001")
	in := app.WriteOffInput{Date: today, ExpenseAccountID: h.id("1102001"), Reason: "Klien pailit"}
	_, err := l.WriteOffInvoice(h.ctx, "u1", inv.ID, in)
	h.wantErr(err, "Akun harus akun beban yang postable")
	in.ExpenseAccountID = expense
	after, err := l.WriteOffInvoice(h.ctx, "u1", inv.ID, in)
	h.ok(err)
	if after.Outstanding() != 0 {
		t.Fatalf("outstanding %s", after.Outstanding())
	}
	rows, _, err := l.ListPayments(h.ctx, domain.PaymentFilter{Kind: domain.KindAR, Limit: 10})
	h.ok(err)
	if len(rows) != 1 || rows[0].Method != "write_off" {
		t.Fatalf("%+v", rows)
	}
	e, err := app.NewJournals(h.deps).Get(h.ctx, rows[0].JournalEntryID)
	h.ok(err)
	if lineOn(e, expense).Amount != inv.TotalAmount || lineOn(e, expense).Side != domain.Debit {
		t.Fatalf("%+v", e.Lines)
	}
	res, err := l.VoidPayment(h.ctx, domain.KindAR, "u1", rows[0].ID, "Klien membayar")
	h.ok(err)
	if res.Invoices[0].Outstanding() != inv.TotalAmount {
		t.Fatalf("receivable must return: %s", res.Invoices[0].Outstanding())
	}
}

func TestVATClosingAndRemittance(t *testing.T) {
	h := newHarness(t)
	l := app.NewLedgers(h.deps)
	setup := app.NewTaxSetup(h.deps)
	_, err := setup.SaveTaxCode(h.ctx, "u1", domain.TaxCode{Code: "PPN-T", Name: "PPN 11%", Kind: domain.TaxVAT, Scope: domain.ScopeBoth, Rate: rp(11), BaseNumerator: 1, BaseDenominator: 1, AccountSaleID: h.id("2103003"), AccountPurchaseID: h.id("1104001"), IsActive: true})
	h.ok(err)
	_, err = l.CloseVAT(h.ctx, "u1", "2026-13")
	h.wantErr(err, "Masa pajak harus berformat YYYY-MM")
	_, err = l.CloseVAT(h.ctx, "u1", "2026-10")
	h.wantErr(err, "Tidak ada PPN untuk dikompensasi pada masa ini")

	journals := app.NewJournals(h.deps)
	bank := h.id("1102001")
	for _, lines := range [][]domain.Line{
		{line(bank, domain.Debit, 1_100_000), line(h.id("2103003"), domain.Credit, 1_100_000)},
		{line(h.id("1104001"), domain.Debit, 400_000), line(bank, domain.Credit, 400_000)},
	} {
		_, err = journals.Create(h.ctx, "u1", app.EntryInput{EntryDate: today, Description: "PPN uji", Lines: lines, Post: true})
		h.ok(err)
	}
	c, err := l.CloseVAT(h.ctx, "u1", "2026-10")
	h.ok(err)
	if c.Offset != rp(400_000) || c.Payable != rp(700_000) || c.Compensation != 0 {
		t.Fatalf("%+v", c)
	}
	_, err = l.CloseVAT(h.ctx, "u1", "2026-10")
	h.wantErr(err, "Tidak ada PPN untuk dikompensasi pada masa ini")

	in := domain.TaxRemittance{Period: "2026-10", Date: "2026-10-31", AccountID: h.id("2103003"), CashAccountID: bank, Amount: rp(800_000), NTPN: "ABC123"}
	_, err = l.RemitTax(h.ctx, "u1", in)
	h.wantErr(err, "Nilai melebihi saldo hutang pajak")
	in.Amount = rp(700_000)
	r, err := l.RemitTax(h.ctx, "u1", in)
	h.ok(err)
	positions, err := l.TaxPositions(h.ctx, "2026-10-31")
	h.ok(err)
	for _, p := range positions {
		if p.AccountID == h.id("2103003") && p.Balance != 0 {
			t.Fatalf("payable left %s", p.Balance)
		}
	}
	r, err = l.VoidTaxRemittance(h.ctx, "u1", r.ID, "Salah nominal")
	h.ok(err)
	if r.Status != domain.StatusVoid {
		t.Fatalf("%+v", r)
	}
	_, err = l.VoidTaxRemittance(h.ctx, "u1", r.ID, "Salah nominal")
	h.wantErr(err, "Setoran ini sudah dibatalkan")
}

func TestSeparateApproverBlocksOwnDraft(t *testing.T) {
	h := newHarness(t)
	l := app.NewLedgers(h.deps)
	h.ok(l.SetSeparateApprover(h.ctx, "boss", true))
	inv, err := l.CreateInvoice(h.ctx, domain.KindAR, "maker", app.InvoiceInput{InvoiceDate: today, PartyName: "PT Dua", Subtotal: rp(100), AccountID: h.id("4101001")})
	h.ok(err)
	_, _, err = l.PostInvoice(h.ctx, domain.KindAR, "maker", inv.ID)
	h.wantErr(err, "Pembuat dokumen tidak boleh memproses dokumennya sendiri. Minta rekan lain memeriksanya.")
	_, _, err = l.PostInvoice(h.ctx, domain.KindAR, "checker", inv.ID)
	h.ok(err)

	journals := app.NewJournals(h.deps)
	e, err := journals.Create(h.ctx, "maker", app.EntryInput{EntryDate: today, Description: "Draft", Lines: []domain.Line{line(h.id("1102001"), domain.Debit, 10), line(h.id("8101001"), domain.Credit, 10)}})
	h.ok(err)
	_, err = journals.Post(h.ctx, "maker", e.ID)
	h.wantErr(err, "Pembuat dokumen tidak boleh memproses dokumennya sendiri. Minta rekan lain memeriksanya.")
	_, err = journals.Post(h.ctx, "checker", e.ID)
	h.ok(err)
	_, err = journals.Create(h.ctx, "maker", app.EntryInput{EntryDate: today, Description: "Langsung", Post: true, Lines: []domain.Line{line(h.id("1102001"), domain.Debit, 10), line(h.id("8101001"), domain.Credit, 10)}})
	h.wantErr(err, "Pembuat dokumen tidak boleh memproses dokumennya sendiri. Minta rekan lain memeriksanya.")
}

func TestAuditTrailIsAppendOnly(t *testing.T) {
	h := newHarness(t)
	l := app.NewLedgers(h.deps)
	h.deps.AuditSystem(h.ctx, "RunSchedules", "doc-1", map[string]int{"posted": 1}, "selesai")
	rows, total, err := l.ListAuditTrail(h.ctx, domain.AuditFilter{Search: "RunSched", Limit: 10})
	h.ok(err)
	if total != 1 || rows[0].ActorName != "Sistem" || rows[0].DocID != "doc-1" {
		t.Fatalf("%d %+v", total, rows)
	}
	csv, err := l.ExportAuditTrail(h.ctx, domain.AuditFilter{})
	h.ok(err)
	if !strings.Contains(string(csv), "RunSchedules") {
		t.Fatalf("%s", csv)
	}
	if err := h.pool.QueryRow(h.ctx, "DELETE FROM finance.audit_trail RETURNING id").Scan(new(string)); err == nil || !strings.Contains(err.Error(), "audit trail tidak bisa diubah") {
		t.Fatalf("delete must fail: %v", err)
	}
}

func TestSalesReversalsWaitForAnOpenPeriod(t *testing.T) {
	h := newHarness(t)
	auto := app.NewAutoPost(h.deps)
	inv, err := auto.SalesInvoiceIssued(h.ctx, domain.SalesInvoiceIssued{InvoiceID: "si-r", InvoiceNumber: "INV-R", CustomerName: "PT Balik", IssueDate: today, Subtotal: rp(1_000_000), TotalAmount: rp(1_000_000)})
	h.ok(err)
	_, err = auto.SalesPaymentRecorded(h.ctx, domain.SalesPaymentRecorded{PaymentID: "sp-r", InvoiceID: "si-r", Amount: rp(400_000), PaidOn: today, Method: "transfer"})
	h.ok(err)
	if err := auto.SalesPaymentDeleted(h.ctx, domain.SalesPaymentDeleted{PaymentID: "belum-ada"}); err == nil {
		t.Fatal("an early delete must be redelivered")
	}

	// November has no open period, so the reversal waits and says why.
	later := h.deps
	later.Clock = clock.At("2026-11-05T03:00:00Z")
	h.ok(app.NewAutoPost(later).SalesPaymentDeleted(h.ctx, domain.SalesPaymentDeleted{PaymentID: "sp-r"}))
	pending, err := h.deps.Store.PendingSalesReversals(h.ctx)
	h.ok(err)
	if len(pending) != 1 || pending[0].Kind != "PAYMENT" || pending[0].Error == "" {
		t.Fatalf("%+v", pending)
	}

	// The daily run finishes it once the period is open.
	_, err = app.NewSchedules(h.deps).RunDue(h.ctx)
	h.ok(err)
	pending, err = h.deps.Store.PendingSalesReversals(h.ctx)
	h.ok(err)
	after, err := h.deps.Store.GetInvoice(h.ctx, inv.ID)
	h.ok(err)
	if len(pending) != 0 || after.Outstanding() != inv.TotalAmount {
		t.Fatalf("%+v outstanding %s", pending, after.Outstanding())
	}
	// A repeat is harmless.
	h.ok(auto.SalesPaymentDeleted(h.ctx, domain.SalesPaymentDeleted{PaymentID: "sp-r"}))

	h.ok(auto.SalesInvoiceCancelled(h.ctx, domain.SalesInvoiceCancelled{InvoiceID: "si-r", Reason: "Salah terbit"}))
	after, err = h.deps.Store.GetInvoice(h.ctx, inv.ID)
	h.ok(err)
	if after.Outstanding() != 0 {
		t.Fatalf("cancelled invoice left %s", after.Outstanding())
	}
	pending, err = h.deps.Store.PendingSalesReversals(h.ctx)
	h.ok(err)
	if len(pending) != 0 {
		t.Fatalf("%+v", pending)
	}
	h.ok(auto.SalesInvoiceCancelled(h.ctx, domain.SalesInvoiceCancelled{InvoiceID: "si-r", Reason: "Salah terbit"}))
}

func TestSalesExcessAdvanceVoidsWithItsPayment(t *testing.T) {
	h := newHarness(t)
	auto := app.NewAutoPost(h.deps)
	_, err := auto.SalesInvoiceIssued(h.ctx, domain.SalesInvoiceIssued{InvoiceID: "si-e", InvoiceNumber: "INV-E", CustomerName: "PT Lebih", IssueDate: today, Subtotal: rp(100_000), TotalAmount: rp(100_000)})
	h.ok(err)
	_, err = auto.SalesPaymentRecorded(h.ctx, domain.SalesPaymentRecorded{PaymentID: "sp-e", InvoiceID: "si-e", Amount: rp(150_000), PaidOn: today, Method: "transfer"})
	h.ok(err)
	h.ok(auto.SalesPaymentDeleted(h.ctx, domain.SalesPaymentDeleted{PaymentID: "sp-e"}))
	rows, err := app.NewLedgers(h.deps).ListAdvances(h.ctx, domain.KindAR)
	h.ok(err)
	if len(rows) != 1 || rows[0].Status != domain.StatusVoid {
		t.Fatalf("%+v", rows)
	}
}

func TestReverseJournal(t *testing.T) {
	h := newHarness(t)
	journals := app.NewJournals(h.deps)
	bank, fee := h.id("1102001"), h.id("8101001")
	e, err := journals.Create(h.ctx, "maker", app.EntryInput{EntryDate: today, Description: "Salah catat", Post: true, Lines: []domain.Line{line(fee, domain.Debit, 100), line(bank, domain.Credit, 100)}})
	h.ok(err)
	_, err = journals.Reverse(h.ctx, "boss", e.ID, today, "x")
	h.wantErr(err, "Alasan minimal 5 karakter")
	back, err := journals.Reverse(h.ctx, "boss", e.ID, today, "Salah akun")
	h.ok(err)
	if lineOn(back, fee).Side != domain.Credit || lineOn(back, bank).Side != domain.Debit {
		t.Fatalf("%+v", back.Lines)
	}
	_, err = journals.Reverse(h.ctx, "boss", e.ID, today, "Salah akun")
	h.wantErr(err, "Jurnal ini sudah dibalik oleh "+back.EntryNo)
	_, err = journals.Reverse(h.ctx, "boss", back.ID, today, "Balik lagi")
	h.wantErr(err, "Jurnal otomatis dibalik lewat dokumen sumbernya")
}

func TestDisposeAsset(t *testing.T) {
	h := newHarness(t)
	schedules := app.NewSchedules(h.deps)
	a, err := schedules.CreateAsset(h.ctx, "u1", app.AssetInput{Name: "Kamera", AcquiredOn: "2026-10-01", Cost: rp(12_000_000), LifeMonths: 12, ExpenseAccountID: h.id("6301001"), AccumAccountID: h.id("1106001")})
	h.ok(err)
	in := app.DisposalInput{Date: today, AssetAccountID: h.id("1106001"), CashAccountID: h.id("1102001"), GainLossAccountID: h.id("8101001"), Reason: "Dijual", Proceeds: rp(10_000_000)}
	out, err := schedules.DisposeAsset(h.ctx, "u1", a.ID, in)
	h.ok(err)
	if out.DisposedOn != today || out.IsActive {
		t.Fatalf("%+v", out)
	}
	_, err = schedules.DisposeAsset(h.ctx, "u1", a.ID, in)
	h.wantErr(err, "Aset ini sudah dilepas")
	_, err = schedules.SetAssetActive(h.ctx, "u1", a.ID, true)
	h.wantErr(err, "Aset yang sudah dilepas tidak bisa diaktifkan lagi")
	lines := app.DisposalLines(domain.FixedAsset{Name: "x", AccumAccountID: "acc", Cost: rp(100), LifeMonths: 10, DepreciatedMonths: 5}, app.DisposalInput{AssetAccountID: "asset", CashAccountID: "cash", GainLossAccountID: "gl", Proceeds: rp(70)})
	h.ok(domain.ValidateBalanced(lines))
}

func TestProjectAllocationKeepsTheTotal(t *testing.T) {
	h := newHarness(t)
	tags := app.NewProjectTags(h.deps)
	tag, err := tags.Save(h.ctx, "u1", domain.ProjectTag{Name: "Retainer A", ClientName: "PT A", IsActive: true})
	h.ok(err)
	_, err = app.NewJournals(h.deps).Create(h.ctx, "u1", app.EntryInput{EntryDate: today, Description: "Gaji", Post: true, Lines: []domain.Line{line(h.id("6101001"), domain.Debit, 1_000), line(h.id("1102001"), domain.Credit, 1_000)}})
	h.ok(err)
	_, err = tags.CreateAllocation(h.ctx, "u1", domain.ProjectAllocation{TagID: tag.ID, Month: "2026-10", Kind: domain.AllocPayroll, Amount: rp(300)})
	h.ok(err)
	_, err = tags.CreateAllocation(h.ctx, "u1", domain.ProjectAllocation{TagID: tag.ID, Month: "bad", Kind: domain.AllocPayroll, Amount: rp(1)})
	h.wantErr(err, "Bulan harus berformat YYYY-MM")
	rows, _, _, err := tags.ProjectProfit(h.ctx, "2026-10-01", "2026-10-31", "")
	h.ok(err)
	var total, project domain.Money
	for _, r := range rows {
		total += r.Cost
		if r.TagID == tag.ID {
			project = r.Cost
		}
	}
	if project != rp(300) || total != rp(1_000) {
		t.Fatalf("project %s total %s %+v", project, total, rows)
	}
}

func TestCompanyProfileFallsBackThenStamps(t *testing.T) {
	h := newHarness(t)
	p := app.NewCompanyProfile(h.deps)
	if got := p.Get(h.ctx); got.LegalName != h.deps.Company.LegalName || got.BankAccountNo != "" {
		t.Fatalf("defaults before any save: %+v", got)
	}
	_, err := p.Save(h.ctx, "u1", domain.Company{LegalName: "PT Mixist", BankAccountNo: "12a"})
	h.wantErr(err, "Nomor rekening hanya berisi angka")
	saved, err := p.Save(h.ctx, "u1", domain.Company{LegalName: " PT Mixist Digital ", BankName: "BCA", BankAccountNo: "123 456 7890"})
	h.ok(err)
	if saved.LegalName != "PT Mixist Digital" || saved.BankLine() != "Transfer ke BCA 123 456 7890 a.n. PT Mixist Digital. Cantumkan nomor invoice pada berita transfer." {
		t.Fatalf("%+v %q", saved, saved.BankLine())
	}
	if got := p.Get(h.ctx); got.BankName != "BCA" {
		t.Fatalf("saved profile wins: %+v", got)
	}
}

type fakeTimesheet struct{ costs []app.ProjectCost }

func (f fakeTimesheet) ProjectCosts(context.Context, string) ([]app.ProjectCost, error) {
	return f.costs, nil
}

func TestAllocateFromTimesheetReplacesOnlyItsOwnRows(t *testing.T) {
	h := newHarness(t)
	tag, err := app.NewProjectTags(h.deps).Save(h.ctx, "u1", domain.ProjectTag{Name: "Retainer A", ClientName: "PT A", IsActive: true})
	h.ok(err)
	h.deps.Timesheet = fakeTimesheet{costs: []app.ProjectCost{
		{ProjectID: tag.ID, ProjectName: "Retainer A", Hours: 12.5, Cost: "4500000.00"},
		{ProjectID: "gone", ProjectName: "Proyek lama", Hours: 2, Cost: "100000.00"},
	}}
	tags := app.NewProjectTags(h.deps)
	_, err = tags.CreateAllocation(h.ctx, "u1", domain.ProjectAllocation{TagID: tag.ID, Month: "2026-10", Kind: domain.AllocReimbursement, Amount: rp(50)})
	h.ok(err)
	for range 2 { // a rerun replaces, never doubles
		res, err := tags.AllocateFromTimesheet(h.ctx, "u1", "2026-10")
		h.ok(err)
		if len(res.Skipped) != 1 || res.Skipped[0] != "Proyek lama" || len(res.Allocations) != 2 {
			t.Fatalf("%+v", res)
		}
	}
	rows, err := tags.ListAllocations(h.ctx, "2026-10-01", "2026-10-31")
	h.ok(err)
	var fromTimesheet, manual int
	for _, a := range rows {
		switch a.Source {
		case domain.AllocTimesheet:
			fromTimesheet++
			if a.Amount != rp(4_500_000) || a.Kind != domain.AllocPayroll || a.Notes != "Dari timesheet: 12.5 jam" {
				t.Fatalf("%+v", a)
			}
		case domain.AllocManual:
			manual++
		}
	}
	if fromTimesheet != 1 || manual != 1 {
		t.Fatalf("timesheet %d manual %d", fromTimesheet, manual)
	}
}

func TestOverdueReminderOncePerStage(t *testing.T) {
	h := newHarness(t)
	l := app.NewLedgers(h.deps)
	inv, err := l.CreateInvoice(h.ctx, domain.KindAR, "u1", app.InvoiceInput{InvoiceDate: "2026-10-01", DueDate: "2026-10-10", PartyName: "PT Telat", Subtotal: rp(100), AccountID: h.id("4101001")})
	h.ok(err)
	_, _, err = l.PostInvoice(h.ctx, domain.KindAR, "u1", inv.ID)
	h.ok(err)
	s := app.NewSchedules(h.deps)
	_, err = s.RunDue(h.ctx)
	h.ok(err)
	if h.identity.last().Title != "Tagihan "+inv.InvoiceNo+" lewat 14 hari" {
		t.Fatalf("%+v", h.identity.last())
	}
	sent := len(h.identity.sent)
	_, err = s.RunDue(h.ctx)
	h.ok(err)
	if len(h.identity.sent) != sent {
		t.Fatal("a stage reminds once")
	}
}

func TestPOMatchStopsBillingBeyondDelivery(t *testing.T) {
	h := newHarness(t)
	l := app.NewLedgers(h.deps)
	docs := app.NewDocuments(h.deps)
	vendor, err := docs.SaveVendor(h.ctx, "u1", domain.Vendor{Name: "CV Kirim", IsActive: true})
	h.ok(err)
	po, err := docs.Create(h.ctx, "u1", domain.Document{Kind: domain.DocPO, Side: domain.SideSupplier, VendorID: vendor.ID, PartyName: "CV Kirim", Title: "Sewa alat", IssueDate: today, Items: []domain.DocItem{{Description: "Alat", Qty: rp(1), UnitPrice: rp(1_000)}}})
	h.ok(err)
	for _, act := range []string{domain.ActSubmit, domain.ActApprove} {
		po, err = docs.Act(h.ctx, principal("boss", authn.PermFinanceApprove), po.ID, act, "")
		h.ok(err)
	}
	inv, err := l.CreateInvoice(h.ctx, domain.KindAP, "u1", app.InvoiceInput{InvoiceDate: today, PartyName: "CV Kirim", Subtotal: rp(800)})
	h.ok(err)
	h.ok(l.MatchInvoiceToPO(h.ctx, inv.ID, po.ID))
	_, _, err = l.PostInvoice(h.ctx, domain.KindAP, "u1", inv.ID)
	h.wantErr(err, "Nilai tagihan melebihi barang atau jasa yang sudah diterima pada purchase order ini")
	_, err = l.ReceiveGoods(h.ctx, "u1", domain.GoodsReceipt{DocumentID: po.ID, Date: today, Amount: rp(2_000)})
	h.wantErr(err, "Nilai melebihi sisa purchase order")
	r, err := l.ReceiveGoods(h.ctx, "u1", domain.GoodsReceipt{DocumentID: po.ID, Date: today, Amount: rp(900)})
	h.ok(err)
	_, _, err = l.PostInvoice(h.ctx, domain.KindAP, "u1", inv.ID)
	h.ok(err)
	err = l.VoidGoodsReceipt(h.ctx, r.ID, "Salah catat")
	h.wantErr(err, "Tagihan yang sudah diposting memakai penerimaan ini")
	rows, err := l.ListPOMatches(h.ctx)
	h.ok(err)
	if len(rows) != 1 || rows[0].Received != rp(900) || rows[0].Billed != rp(800) {
		t.Fatalf("%+v", rows)
	}
}

func TestSeparateApproverCoversPaymentsToo(t *testing.T) {
	h, l, inv := settlementHarness(t, domain.KindAR, "PT Dua")
	h.ok(l.SetSeparateApprover(h.ctx, "boss", true))
	const refused = "Pembuat dokumen tidak boleh memproses dokumennya sendiri. Minta rekan lain memeriksanya."
	in := app.PaymentInput{PaymentDate: today, CashAccountID: h.id("1102001"), Allocations: []domain.Allocation{{InvoiceID: inv.ID, Amount: rp(100)}}}
	_, err := l.CreatePayment(h.ctx, domain.KindAR, "u1", in)
	h.wantErr(err, refused)
	_, err = l.CreateCreditNote(h.ctx, "u1", inv.ID, app.CreditNoteInput{Date: today, Amount: rp(1), Reason: "Retur"})
	h.wantErr(err, refused)
	_, err = l.WriteOffInvoice(h.ctx, "u1", inv.ID, app.WriteOffInput{Date: today, ExpenseAccountID: h.id("8102001"), Reason: "Klien pailit"})
	h.wantErr(err, refused)
	_, err = l.CreatePayment(h.ctx, domain.KindAR, "kasir", in)
	h.ok(err)
}
