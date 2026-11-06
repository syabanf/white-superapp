package app

import (
	"context"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// ShareClient is a marketing client whose share page the visitor unlocked.
type ShareClient struct{ ID, Name string }

// ShareGate checks a share slug and its unlock cookie through identity.
// It returns nil when the page is missing, disabled, still locked, or has
// no PIN: billing never shows on a page anyone can open.
type ShareGate interface {
	UnlockedClient(ctx context.Context, slug, unlockToken string) (*ShareClient, error)
}

// PermissionHolders lists active users holding a permission, admins included.
type PermissionHolders interface {
	Holders(ctx context.Context, permission string) ([]User, error)
}

// Portal is the client side of the share page: invoices with what is left
// to pay, contracts, and transfer proofs Finance turns into payments.
type Portal struct {
	*Deps
	billing *Billing
}

func NewPortal(d *Deps, billing *Billing) *Portal { return &Portal{Deps: d, billing: billing} }

type PortalView struct {
	Client    ShareClient
	Invoices  []domain.Invoice
	Contracts []domain.Contract
	Proofs    []domain.PaymentProof
	BankLine  string
}

const (
	ProofPending  = "PENDING"
	ProofApproved = "APPROVED"
	ProofRejected = "REJECTED"
	NotifProof    = "SALES_PAYMENT_PROOF"
)

var proofTypes = map[string]bool{"image/png": true, "image/jpeg": true, "image/webp": true, "application/pdf": true}

func (p *Portal) client(ctx context.Context, slug, token string) (*ShareClient, error) {
	c, err := p.Share.UnlockedClient(ctx, slug, token)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, apperr.New(apperr.NotFound, msgLinkInvalid)
	}
	return c, nil
}

// View lists issued invoices, sent or signed contracts and the proofs sent
// for those invoices.
func (p *Portal) View(ctx context.Context, slug, token string) (*PortalView, error) {
	c, err := p.client(ctx, slug, token)
	if err != nil {
		return nil, err
	}
	invoices, err := p.Store.ListInvoices(ctx, domain.InvoiceFilter{MarketingClientID: c.ID, Status: InvoiceSent})
	if err != nil {
		return nil, err
	}
	all, err := p.Store.ListContracts(ctx, domain.ContractFilter{MarketingClientID: c.ID})
	if err != nil {
		return nil, err
	}
	contracts := make([]domain.Contract, 0, len(all))
	for _, k := range all {
		if k.Status != domain.ContractDraft && k.Status != domain.ContractCancelled {
			contracts = append(contracts, k)
		}
	}
	view := &PortalView{Client: *c, Invoices: invoices, Contracts: contracts, BankLine: p.company(ctx).BankLine}
	if len(invoices) == 0 {
		return view, nil
	}
	invoiceIDs := make([]string, len(invoices))
	for i, inv := range invoices {
		invoiceIDs[i] = inv.ID
	}
	if view.Proofs, err = p.Store.ListPaymentProofs(ctx, domain.PaymentProofFilter{InvoiceIDs: invoiceIDs, Limit: 100}); err != nil {
		return nil, err
	}
	return view, nil
}

// portalInvoice loads an issued invoice that belongs to the unlocked client.
func (p *Portal) portalInvoice(ctx context.Context, slug, token, invoiceID string) (*domain.Invoice, error) {
	c, err := p.client(ctx, slug, token)
	if err != nil {
		return nil, err
	}
	list, err := p.Store.ListInvoices(ctx, domain.InvoiceFilter{MarketingClientID: c.ID, Status: InvoiceSent})
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].ID == invoiceID {
			return &list[i], nil
		}
	}
	return nil, notFound(msgInvoiceNotFound)
}

// InvoicePDF renders an invoice for the client who owns it.
func (p *Portal) InvoicePDF(ctx context.Context, slug, token, invoiceID string) (string, []byte, error) {
	if _, err := p.portalInvoice(ctx, slug, token, invoiceID); err != nil {
		return "", nil, err
	}
	return p.billing.RenderInvoicePDF(ctx, invoiceID)
}

type ProofInput struct {
	Slug, Token, InvoiceID, Amount, PaidOn, PayerName, Note, FileName string
	Data                                                              []byte
}

// SubmitProof stores the receipt and tells Finance. Nothing posts until a
// Finance user approves it.
func (p *Portal) SubmitProof(ctx context.Context, in ProofInput) (*domain.PaymentProof, error) {
	inv, err := p.portalInvoice(ctx, in.Slug, in.Token, in.InvoiceID)
	if err != nil {
		return nil, err
	}
	errs := map[string]string{}
	amount, err := domain.ParseMoney(in.Amount)
	if err != nil || amount <= 0 {
		errs["amount"] = "Nominal transfer harus lebih dari 0"
	}
	paidOn, err := domain.ParseDate(in.PaidOn)
	if err != nil {
		errs["paidOn"] = "Tanggal transfer tidak valid"
	} else if paidOn.After(p.now()) {
		errs["paidOn"] = "Tanggal transfer tidak boleh di masa depan"
	}
	payer := strings.TrimSpace(in.PayerName)
	if payer == "" {
		errs["payerName"] = "Nama pengirim wajib diisi"
	}
	ct, ext, err := domain.SniffAsset(in.Data, in.FileName)
	if err != nil || !proofTypes[ct] {
		errs["file"] = "Unggah bukti transfer berupa PDF, PNG, JPG, atau WEBP maksimal 15 MB"
	}
	if len(errs) > 0 {
		return nil, fieldErrs(errs)
	}
	pending, err := p.Store.ListPaymentProofs(ctx, domain.PaymentProofFilter{Status: ProofPending, InvoiceIDs: []string{inv.ID}, Limit: 20})
	if err != nil {
		return nil, err
	}
	if len(pending) >= 5 {
		return nil, conflict("Masih ada 5 bukti transfer yang menunggu pemeriksaan untuk invoice ini")
	}
	proof := domain.PaymentProof{ID: ids.New(), InvoiceID: inv.ID, Amount: amount, PaidOn: *paidOn, PayerName: payer, Note: strings.TrimSpace(in.Note), FileName: domain.SafeFileName(in.FileName)}
	proof.FileKey = "payment-proofs/" + inv.ID + "/" + proof.ID + "." + ext
	if err := p.Files.Put(ctx, proof.FileKey, in.Data); err != nil {
		return nil, err
	}
	if err := p.Store.InsertPaymentProof(ctx, proof); err != nil {
		return nil, err
	}
	p.notify(ctx, Notification{UserIDs: p.holderIDs(ctx), Type: NotifProof, Title: "Bukti transfer " + inv.InvoiceNumber,
		Body: payer + " · " + amount.Rupiah() + ". Periksa mutasi bank lalu setujui.", Href: "/erp/finance/payment-proofs"})
	return p.Store.GetPaymentProof(ctx, proof.ID)
}

func (p *Portal) holderIDs(ctx context.Context) []string {
	if p.Holders == nil {
		return nil
	}
	users, err := p.Holders.Holders(ctx, authn.PermFinanceApprove)
	if err != nil {
		p.warn("proof approver lookup failed", "err", err)
		return nil
	}
	out := make([]string, len(users))
	for i, u := range users {
		out[i] = u.ID
	}
	return out
}

func (p *Portal) ListProofs(ctx context.Context, status string) ([]domain.PaymentProof, error) {
	return p.Store.ListPaymentProofs(ctx, domain.PaymentProofFilter{Status: status, Limit: 200})
}

func (p *Portal) ProofFile(ctx context.Context, id string) (string, []byte, error) {
	proof, err := p.loadProof(ctx, id, false)
	if err != nil {
		return "", nil, err
	}
	data, err := p.Files.Get(ctx, proof.FileKey)
	return proof.FileName, data, err
}

// loadProof finds a proof; pendingOnly refuses one already decided.
func (p *Portal) loadProof(ctx context.Context, id string, pendingOnly bool) (*domain.PaymentProof, error) {
	proof, err := p.Store.GetPaymentProof(ctx, id)
	if err != nil {
		return nil, err
	}
	if proof == nil {
		return nil, notFound("Bukti transfer tidak ditemukan")
	}
	if pendingOnly && proof.Status != ProofPending {
		return nil, conflict("Bukti transfer ini sudah diputuskan")
	}
	return proof, nil
}

// ApproveInput lets Finance correct the amount or date to what the bank
// statement shows and split off a bank fee or withheld tax.
type ApproveInput struct {
	ID, Amount, PaidOn, BankFee, Withheld, WithheldTaxCode string
}

// ApproveProof records the payment, which publishes it to Finance as usual.
func (p *Portal) ApproveProof(ctx context.Context, actor Actor, in ApproveInput) (*domain.PaymentProof, error) {
	proof, err := p.loadProof(ctx, in.ID, true)
	if err != nil {
		return nil, err
	}
	amount, paidOn := firstNonEmpty(in.Amount, proof.Amount.String()), firstNonEmpty(in.PaidOn, domain.FormatDay(&proof.PaidOn))
	pay, err := p.billing.RecordPayment(ctx, actor, PaymentInput{DealID: proof.DealID, InvoiceID: proof.InvoiceID, Amount: amount, Method: "transfer", PaidOn: paidOn,
		Note: "Bukti transfer dari " + proof.PayerName, BankFee: in.BankFee, Withheld: in.Withheld, WithheldTaxCode: in.WithheldTaxCode})
	if err != nil {
		return nil, err
	}
	now := p.now()
	proof.Status, proof.PaymentID, proof.DecidedBy, proof.DecidedAt = ProofApproved, &pay.ID, actor.ref(), &now
	ok, err := p.Store.DecidePaymentProof(ctx, *proof)
	if err != nil {
		return nil, err
	}
	if !ok {
		// Another approver won the race: undo this payment so it is not counted twice.
		if derr := p.billing.DeletePayment(ctx, actor, pay.ID); derr != nil {
			p.warn("proof race: payment left behind", "payment", pay.ID, "err", derr)
		}
		return nil, conflict("Bukti transfer ini sudah diputuskan")
	}
	return p.Store.GetPaymentProof(ctx, proof.ID)
}

func (p *Portal) RejectProof(ctx context.Context, actor Actor, id, reason string) (*domain.PaymentProof, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, fieldErrs(map[string]string{"reason": "Alasan penolakan wajib diisi agar klien tahu yang harus diperbaiki"})
	}
	proof, err := p.loadProof(ctx, id, true)
	if err != nil {
		return nil, err
	}
	now := p.now()
	proof.Status, proof.RejectReason, proof.DecidedBy, proof.DecidedAt = ProofRejected, reason, actor.ref(), &now
	ok, err := p.Store.DecidePaymentProof(ctx, *proof)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, conflict("Bukti transfer ini sudah diputuskan")
	}
	return p.Store.GetPaymentProof(ctx, proof.ID)
}
