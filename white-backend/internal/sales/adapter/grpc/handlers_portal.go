package grpc

import (
	"context"
	"mime"
	"path/filepath"

	salesv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/sales/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/app"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
)

type portalServer struct {
	salesv1.UnimplementedPortalServiceServer
	svc Services
}

func proofPB(p domain.PaymentProof) *salesv1.PaymentProof {
	return &salesv1.PaymentProof{Id: p.ID, InvoiceId: p.InvoiceID, InvoiceNumber: p.InvoiceNumber, DealId: p.DealID, ClientName: p.ClientName, Amount: p.Amount.String(),
		PaidOn: domain.FormatDay(&p.PaidOn), PayerName: p.PayerName, Note: p.Note, FileName: p.FileName, Status: p.Status, RejectReason: p.RejectReason,
		PaymentId: str(p.PaymentID), DecidedAt: tsPtr(p.DecidedAt), CreatedAt: ts(p.CreatedAt)}
}

func proofsPB(ps []domain.PaymentProof) []*salesv1.PaymentProof {
	out := make([]*salesv1.PaymentProof, 0, len(ps))
	for _, p := range ps {
		out = append(out, proofPB(p))
	}
	return out
}

// portalInvoicePB leaves out staff-only fields: who issued it and the
// internal note.
func portalInvoicePB(i domain.Invoice) *salesv1.Invoice {
	out := invoicePB(i)
	out.IssuedBy, out.Note, out.QuotationId, out.TermId = "", "", "", ""
	return out
}

func (s *portalServer) GetPortal(ctx context.Context, r *salesv1.PortalRequest) (*salesv1.Portal, error) {
	v, err := s.svc.Portal.View(ctx, r.Slug, r.UnlockToken)
	if err != nil {
		return nil, err
	}
	out := &salesv1.Portal{ClientId: v.Client.ID, ClientName: v.Client.Name, BankLine: v.BankLine, Proofs: proofsPB(v.Proofs)}
	for _, i := range v.Invoices {
		out.Invoices = append(out.Invoices, portalInvoicePB(i))
	}
	for _, c := range v.Contracts {
		out.Contracts = append(out.Contracts, &salesv1.PortalContract{Id: c.ID, ContractNumber: c.ContractNumber, Title: c.Title, Status: c.Status,
			StartDate: domain.FormatDay(c.StartDate), EndDate: domain.FormatDay(c.EndDate), SignedAt: tsPtr(c.SignedAt), SignToken: c.SignToken})
	}
	return out, nil
}

func (s *portalServer) SubmitPaymentProof(ctx context.Context, r *salesv1.SubmitPaymentProofRequest) (*salesv1.PaymentProof, error) {
	p, err := s.svc.Portal.SubmitProof(ctx, app.ProofInput{Slug: r.Slug, Token: r.UnlockToken, InvoiceID: r.InvoiceId, Amount: r.Amount, PaidOn: r.PaidOn,
		PayerName: r.PayerName, Note: r.Note, FileName: r.FileName, Data: r.Data})
	if err != nil {
		return nil, err
	}
	return proofPB(*p), nil
}

func (s *portalServer) RenderPortalInvoicePdf(ctx context.Context, r *salesv1.PortalInvoiceRequest) (*salesv1.File, error) {
	name, data, err := s.svc.Portal.InvoicePDF(ctx, r.Slug, r.UnlockToken, r.InvoiceId)
	if err != nil {
		return nil, err
	}
	return file(name, "application/pdf", data), nil
}

func (s *portalServer) ListPaymentProofs(ctx context.Context, r *salesv1.ListPaymentProofsRequest) (*salesv1.ListPaymentProofsResponse, error) {
	if _, err := salesOrFinanceWriter(ctx); err != nil {
		return nil, err
	}
	ps, err := s.svc.Portal.ListProofs(ctx, r.Status)
	if err != nil {
		return nil, err
	}
	return &salesv1.ListPaymentProofsResponse{Proofs: proofsPB(ps)}, nil
}

func (s *portalServer) ApprovePaymentProof(ctx context.Context, r *salesv1.ApprovePaymentProofRequest) (*salesv1.PaymentProof, error) {
	a, err := salesOrFinanceWriter(ctx)
	if err != nil {
		return nil, err
	}
	p, err := s.svc.Portal.ApproveProof(ctx, a, app.ApproveInput{ID: r.Id, Amount: r.Amount, PaidOn: r.PaidOn, BankFee: r.BankFee, Withheld: r.Withheld, WithheldTaxCode: r.WithheldTaxCode})
	if err != nil {
		return nil, err
	}
	return proofPB(*p), nil
}

func (s *portalServer) RejectPaymentProof(ctx context.Context, r *salesv1.RejectPaymentProofRequest) (*salesv1.PaymentProof, error) {
	a, err := salesOrFinanceWriter(ctx)
	if err != nil {
		return nil, err
	}
	p, err := s.svc.Portal.RejectProof(ctx, a, r.Id, r.Reason)
	if err != nil {
		return nil, err
	}
	return proofPB(*p), nil
}

func (s *portalServer) GetPaymentProofFile(ctx context.Context, r *salesv1.IdRequest) (*salesv1.File, error) {
	if _, err := salesOrFinanceWriter(ctx); err != nil {
		return nil, err
	}
	name, data, err := s.svc.Portal.ProofFile(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	return file(name, mime.TypeByExtension(filepath.Ext(name)), data), nil
}
