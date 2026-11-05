package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// Documents covers vendors, purchase requests, purchase orders and letterhead
// quotations and invoices, down to a draft payable or receivable.
type Documents struct{ Deps }

func NewDocuments(d Deps) *Documents { return &Documents{d} }

// ─── Vendors ─────────────────────────────────

func (s *Documents) ListVendors(ctx context.Context, query string, activeOnly bool) ([]domain.Vendor, error) {
	return s.Store.ListVendors(ctx, trim(query), activeOnly)
}

// DeleteVendor removes a vendor nothing refers to yet.
func (s *Documents) DeleteVendor(ctx context.Context, id string) error {
	used, err := s.Store.VendorInUse(ctx, id)
	if err != nil {
		return err
	}
	if used {
		return domain.Conflict("Vendor sudah dipakai dokumen. Nonaktifkan saja.")
	}
	return s.Store.DeleteVendor(ctx, id)
}

func (s *Documents) SaveVendor(ctx context.Context, userID string, v domain.Vendor) (*domain.Vendor, error) {
	v.Name = trim(v.Name)
	check := validate.New().Required("name", v.Name, "Nama vendor wajib diisi")
	if v.PaymentTermsDays < 0 || v.PaymentTermsDays > 365 {
		check.Add("payment_terms_days", "Termin pembayaran 0 sampai 365 hari")
	}
	if err := check.Err(); err != nil {
		return nil, err
	}
	if v.ID == "" {
		v.ID = ids.New()
	} else if existing, err := s.Store.GetVendor(ctx, v.ID); err != nil {
		return nil, err
	} else if existing == nil {
		return nil, domain.NotFound("Vendor tidak ditemukan")
	}
	v.By = userID
	if err := s.Store.SaveVendor(ctx, v); err != nil {
		var unique *domain.ErrUnique
		if errors.As(err, &unique) {
			return nil, apperr.Field("name", "Nama vendor sudah dipakai")
		}
		return nil, err
	}
	return s.Store.GetVendor(ctx, v.ID)
}

// ─── Documents ───────────────────────────────

var docPrefix = map[string]string{domain.DocPR: "PR", domain.DocPO: "PO", domain.DocQuotation: "QT", domain.DocInvoice: "INV"}

func (s *Documents) List(ctx context.Context, f domain.DocumentFilter, page, size int) ([]domain.Document, int, error) {
	if docPrefix[f.Kind] == "" {
		return nil, 0, apperr.Field("kind", "Jenis dokumen tidak valid")
	}
	f.Limit, f.Offset = Page(page, size)
	f.Query = trim(f.Query)
	return s.Store.ListDocuments(ctx, f)
}

func (s *Documents) load(ctx context.Context, id string) (*domain.Document, error) {
	d, err := s.Store.GetDocument(ctx, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, domain.NotFound("Dokumen tidak ditemukan")
	}
	return d, nil
}

func (s *Documents) Get(ctx context.Context, id string) (*domain.Document, error) {
	return s.load(ctx, id)
}

// normalize validates what a person typed and fills the vendor's details on a
// supplier document. A PR has no party yet: the vendor comes with the PO.
func (s *Documents) normalize(ctx context.Context, d *domain.Document) error {
	d.Title, d.PartyName = trim(d.Title), trim(d.PartyName)
	check := validate.New().Required("title", d.Title, "Judul wajib diisi")
	if d.Kind == domain.DocPO || d.Kind == domain.DocPR {
		d.Side = domain.SideSupplier
	}
	if d.Side != domain.SideB2B && d.Side != domain.SideSupplier {
		check.Add("side", "Pihak dokumen tidak valid")
	}
	if len(d.Items) == 0 {
		check.Add("items", "Tambahkan minimal satu baris")
	}
	for i := range d.Items {
		d.Items[i].Description = trim(d.Items[i].Description)
		if d.Items[i].Description == "" || d.Items[i].Qty <= 0 || d.Items[i].UnitPrice < 0 {
			check.Add("items", "Setiap baris perlu keterangan, jumlah di atas 0, dan harga")
			break
		}
	}
	if d.DiscountPercent < 0 || d.DiscountPercent > domain.Rupiah(100) {
		check.Add("discount_percent", "Diskon 0 sampai 100 persen")
	}
	if d.VendorID != "" {
		v, err := s.Store.GetVendor(ctx, d.VendorID)
		if err != nil {
			return err
		}
		if v == nil || !v.IsActive {
			check.Add("vendor_id", "Vendor tidak ditemukan atau nonaktif")
		} else {
			d.PartyName, d.PicName, d.Phone, d.Email, d.Address = v.Name, v.PicName, v.Phone, v.Email, v.Address
		}
	}
	if d.PartyName == "" && d.Kind != domain.DocPR {
		check.Add("party_name", "Pihak wajib diisi")
	}
	if err := check.Err(); err != nil {
		return err
	}
	day, err := parseDay("issue_date", d.IssueDate)
	if err != nil {
		return err
	}
	due, err := optionalDay("due_date", d.DueDate)
	if err != nil {
		return err
	}
	d.IssueDate, d.DueDate = day, due
	if err := s.applyMaster(ctx, d); err != nil {
		return err
	}
	d.Recalculate()
	return nil
}

// applyMaster copies the chosen taxes and charges onto the document, checking
// each against the document's side and date.
func (s *Documents) applyMaster(ctx context.Context, d *domain.Document) error {
	taxes, err := resolveTaxes(ctx, s.Store, d.TaxIDs, d.Side, d.IssueDate, d.PartyName)
	if err != nil {
		return err
	}
	d.Taxes = taxes

	charges := make([]domain.DocCharge, 0, len(d.Charges))
	for _, in := range d.Charges {
		c, err := s.Store.GetChargeType(ctx, in.ChargeID)
		if err != nil {
			return err
		}
		if c == nil || !c.IsActive {
			return apperr.Field("charges", "Biaya tambahan tidak ditemukan atau nonaktif")
		}
		if !domain.ScopeFits(c.Scope, d.Side) {
			return apperr.Field("charges", c.Code+" tidak berlaku untuk pihak ini")
		}
		value := c.Value
		if in.Value > 0 {
			value = in.Value
		}
		if value <= 0 {
			return apperr.Field("charges", "Isi nilai "+c.Name)
		}
		if c.Computation == domain.ChargePercent && value > domain.Rupiah(100) {
			return apperr.Field("charges", c.Name+" tidak boleh lebih dari 100%")
		}
		charges = append(charges, domain.DocCharge{ChargeID: c.ID, Code: c.Code, Name: c.Name, Computation: c.Computation, Value: value, Taxable: c.Taxable, IsDeduction: c.IsDeduction, AccountID: c.AccountFor(d.Side)})
	}
	d.Charges = charges
	return nil
}

// issuePostings is the document's tax lines plus charges with their own account.
func issuePostings(d domain.Document, kind string) ([]domain.InvoicePosting, error) {
	out, err := taxPostings(d.Taxes, kind)
	if err != nil {
		return nil, err
	}
	natural, other := naturalSides(kind)
	for _, c := range d.Charges {
		// Without its own account a charge stays in the main line.
		if c.AccountID == "" || c.Amount == 0 {
			continue
		}
		side, amount := natural, c.Amount
		if amount < 0 {
			side, amount = other, -amount
		}
		out = append(out, domain.InvoicePosting{AccountID: c.AccountID, Side: side, Amount: amount, Memo: c.Name, SortOrder: len(out)})
	}
	return out, nil
}

func (s *Documents) number(ctx context.Context, tx domain.Store, kind string) (string, error) {
	prefix := docPrefix[kind] + "-" + s.now().In(Jakarta).Format("200601") + "-"
	n, err := tx.NextSequence(ctx, prefix)
	if err != nil {
		return "", err
	}
	return domain.SequenceNo(prefix, n), nil
}

func (s *Documents) insert(ctx context.Context, tx domain.Store, d domain.Document) (string, error) {
	no, err := s.number(ctx, tx, d.Kind)
	if err != nil {
		return "", err
	}
	d.ID, d.Number, d.Status = ids.New(), no, domain.DocDraft
	return d.ID, tx.InsertDocument(ctx, d)
}

func (s *Documents) Create(ctx context.Context, userID string, d domain.Document) (*domain.Document, error) {
	if docPrefix[d.Kind] == "" {
		return nil, apperr.Field("kind", "Jenis dokumen tidak valid")
	}
	if err := s.normalize(ctx, &d); err != nil {
		return nil, err
	}
	d.CreatedBy = userID
	var id string
	err := s.Store.WithTx(ctx, func(tx domain.Store) (err error) {
		id, err = s.insert(ctx, tx, d)
		return err
	})
	if err != nil {
		return nil, err
	}
	return s.load(ctx, id)
}

func (s *Documents) Update(ctx context.Context, userID string, in domain.Document) (*domain.Document, error) {
	d, err := s.load(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if d.Status != domain.DocDraft {
		return nil, domain.Conflict("Hanya dokumen draf yang bisa diubah")
	}
	in.Kind, in.Number, in.Status, in.SourceID, in.CreatedBy, in.UpdatedBy = d.Kind, d.Number, d.Status, d.SourceID, d.CreatedBy, userID
	if err := s.normalize(ctx, &in); err != nil {
		return nil, err
	}
	if err := s.Store.UpdateDocument(ctx, in); err != nil {
		return nil, err
	}
	return s.load(ctx, in.ID)
}

// Delete removes a draft and frees the source it came from.
func (s *Documents) Delete(ctx context.Context, id string) error {
	d, err := s.load(ctx, id)
	if err != nil {
		return err
	}
	if d.Status != domain.DocDraft {
		return domain.Conflict("Hanya dokumen draf yang bisa dihapus")
	}
	return s.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.DeleteDocument(ctx, id); err != nil {
			return err
		}
		if d.SourceID == "" {
			return nil
		}
		src, err := tx.GetDocument(ctx, d.SourceID)
		if err != nil || src == nil {
			return err
		}
		_, src.Status, _ = domain.ConvertTarget(src.Kind)
		return tx.UpdateDocument(ctx, *src)
	})
}

// stamp copies the letterhead when a document leaves draft.
func (s *Documents) stamp(d *domain.Document) {
	if len(d.Letterhead) == 0 {
		d.Letterhead, _ = json.Marshal(s.Company)
	}
}

// Act moves a document along its flow. An approver cannot decide a document
// they made; Admin can, because a small team may have one finance person.
func (s *Documents) Act(ctx context.Context, p authn.Principal, id, action, reason string) (*domain.Document, error) {
	d, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	next, ok := domain.DocTransition(d.Kind, d.Status, action)
	if !ok {
		return nil, domain.Conflict("Aksi ini tidak berlaku untuk status dokumen sekarang")
	}
	if domain.DocNeedsApprover(action) && d.CreatedBy == p.UserID && !p.IsAdmin() {
		return nil, apperr.New(apperr.Unauthorized, "Pembuat dokumen tidak bisa memutuskan dokumennya sendiri")
	}
	reason = trim(reason)
	if action == domain.ActReject && reason == "" {
		return nil, apperr.Field("reason", "Alasan penolakan wajib diisi")
	}
	if d.Kind == domain.DocPO && action == domain.ActSubmit && d.VendorID == "" {
		return nil, apperr.Field("vendor_id", "Pilih vendor sebelum mengajukan PO")
	}
	if d.Status == domain.DocDraft {
		s.stamp(d)
	}
	if action == domain.ActReopen {
		d.Letterhead = nil
	}
	d.Status, d.UpdatedBy = next, p.UserID
	if domain.DocNeedsApprover(action) {
		now := s.now()
		d.DecidedBy, d.DecidedAt = p.UserID, &now
	}
	if action == domain.ActReject || action == domain.ActDecline {
		d.RejectReason = reason
	} else {
		d.RejectReason = ""
	}
	if err := s.Store.UpdateDocument(ctx, *d); err != nil {
		return nil, err
	}
	return s.load(ctx, id)
}

// Convert makes the next document in the chain as a draft. The unique index on
// source_id stops a second conversion even in a race.
func (s *Documents) Convert(ctx context.Context, userID, id, vendorID string) (*domain.Document, error) {
	src, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	target, from, ok := domain.ConvertTarget(src.Kind)
	if !ok || src.Status != from {
		return nil, domain.Conflict("Dokumen ini belum bisa dikonversi")
	}
	d := *src
	d.Kind, d.SourceID, d.CreatedBy, d.VendorID = target, src.ID, userID, trim(vendorID)
	d.IssueDate, d.Letterhead, d.DecidedBy, d.DecidedAt, d.RejectReason = s.today(), nil, "", nil, ""
	d.TaxIDs = nil
	for _, t := range src.Taxes {
		d.TaxIDs = append(d.TaxIDs, t.TaxID)
	}
	if target == domain.DocPO && d.VendorID == "" {
		return nil, apperr.Field("vendor_id", "Pilih vendor untuk PO")
	}
	if err := s.normalize(ctx, &d); err != nil {
		return nil, err
	}
	var newID string
	err = s.Store.WithTx(ctx, func(tx domain.Store) error {
		if newID, err = s.insert(ctx, tx, d); err != nil {
			return err
		}
		src.Status, src.UpdatedBy = domain.DocConverted, userID
		return tx.UpdateDocument(ctx, *src)
	})
	if err != nil {
		return nil, err
	}
	return s.load(ctx, newID)
}

// Issue creates the draft payable or receivable. The ledger invoice and the
// status change share a transaction, so one document cannot be billed twice.
// Posting it stays in Hutang or Piutang usaha under finance.approve.
func (s *Documents) Issue(ctx context.Context, userID, id, accountID, dueDate string) (*domain.Document, error) {
	d, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if !domain.IssueReady(*d) {
		return nil, domain.Conflict("Dokumen ini belum bisa diterbitkan menjadi tagihan")
	}
	if accountID = trim(accountID); accountID == "" {
		accountID = d.AccountID
	}
	if accountID == "" {
		return nil, apperr.Field("account_id", "Pilih akun beban atau pendapatan")
	}
	if due := trim(dueDate); due != "" {
		d.DueDate = due
	}
	kind := domain.KindAP
	if d.Side == domain.SideB2B {
		kind = domain.KindAR
	}
	postings, err := issuePostings(*d, kind)
	if err != nil {
		return nil, err
	}
	reference := d.Number
	if d.ExternalNumber != "" {
		reference = d.ExternalNumber
	}
	err = s.Store.WithTx(ctx, func(tx domain.Store) error {
		deps := s.Deps
		deps.Store = tx
		inv, err := NewLedgers(deps).CreateInvoice(ctx, kind, userID, InvoiceInput{InvoiceDate: s.today(), DueDate: d.DueDate, PartyName: d.PartyName, Reference: reference,
			AccountID: accountID, Description: strings.TrimSpace(d.Number + " " + d.Title), Subtotal: d.Total - d.Tax, TaxAmount: d.Tax, Withholding: d.Withholding})
		if err != nil {
			return err
		}
		for _, p := range postings {
			p.InvoiceID = inv.ID
			if err := tx.InsertInvoicePosting(ctx, p); err != nil {
				return err
			}
		}
		s.stamp(d)
		d.AccountID, d.LedgerInvoiceID, d.UpdatedBy = accountID, inv.ID, userID
		if d.Kind == domain.DocPO {
			d.Status = domain.DocConverted
		} else {
			d.Status = domain.DocIssued
		}
		return tx.UpdateDocument(ctx, *d)
	})
	if err != nil {
		return nil, err
	}
	return s.load(ctx, id)
}

// PDF prints on the letterhead stored with the document, or today's for a draft.
func (s *Documents) PDF(ctx context.Context, id string) (string, []byte, error) {
	d, err := s.load(ctx, id)
	if err != nil {
		return "", nil, err
	}
	company := s.Company
	if len(d.Letterhead) > 0 {
		if err := json.Unmarshal(d.Letterhead, &company); err != nil {
			return "", nil, err
		}
	}
	data, err := s.RenderDocument(*d, company)
	if err != nil {
		return "", nil, err
	}
	return d.Number + ".pdf", data, nil
}
