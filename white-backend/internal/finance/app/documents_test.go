package app_test

import (
	"strings"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
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
	_, err = l.CreateCreditNote(h.ctx, "u1", inv.ID, "", rp(1), "x")
	h.wantErr(err, "Credit note butuh tagihan yang sudah diposting dan punya jurnal")
	_, _, err = l.PostInvoice(h.ctx, domain.KindAR, "u1", inv.ID)
	h.ok(err)
	_, err = l.CreateCreditNote(h.ctx, "u1", inv.ID, "", rp(1), " ")
	h.wantErr(err, "Alasan wajib diisi")
	_, err = l.CreateCreditNote(h.ctx, "u1", inv.ID, "", rp(99_000_000), "x")
	h.wantErr(err, "Nilai credit note melebihi sisa tagihan")

	// A quarter comes back on every line.
	after, err := l.CreateCreditNote(h.ctx, "u1", inv.ID, today, rp(2_775_000), "Revisi nilai")
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
	closed, err := l.CreateCreditNote(h.ctx, "u1", inv.ID, today, 0, "Batal")
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
