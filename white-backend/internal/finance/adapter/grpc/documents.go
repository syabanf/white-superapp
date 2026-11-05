package grpcapi

import (
	"context"

	financev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/finance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"google.golang.org/protobuf/types/known/emptypb"
)

type documentServer struct {
	financev1.UnimplementedDocumentServiceServer
	svc Services
}

func vendorPB(v *domain.Vendor) *financev1.Vendor {
	return &financev1.Vendor{Id: v.ID, Name: v.Name, PicName: v.PicName, Phone: v.Phone, Email: v.Email, Address: v.Address, Npwp: v.NPWP, BankName: v.BankName,
		BankAccountNo: v.BankAccountNo, BankAccountName: v.BankAccountName, PaymentTermsDays: int32(v.PaymentTermsDays), Notes: v.Notes, IsActive: v.IsActive}
}

func documentPB(d *domain.Document) *financev1.FinanceDocument {
	out := &financev1.FinanceDocument{Id: d.ID, Kind: d.Kind, Side: d.Side, Number: d.Number, Status: d.Status, VendorId: d.VendorID, PartyName: d.PartyName, PicName: d.PicName, Phone: d.Phone,
		Email: d.Email, Address: d.Address, Title: d.Title, IssueDate: d.IssueDate, DueDate: d.DueDate, ExternalNumber: d.ExternalNumber, Notes: d.Notes, DiscountPercent: d.DiscountPercent.String(),
		PpnPercent: d.PPNPercent.String(), Subtotal: d.Subtotal.String(), DiscountAmount: d.Discount.String(), TaxAmount: d.Tax.String(), TotalAmount: d.Total.String(), AccountId: d.AccountID,
		SourceId: d.SourceID, LedgerInvoiceId: d.LedgerInvoiceID, RejectReason: d.RejectReason, CreatedBy: d.CreatedBy, DecidedBy: d.DecidedBy}
	for _, it := range d.Items {
		out.Items = append(out.Items, &financev1.DocumentItem{Description: it.Description, Unit: it.Unit, Qty: it.Qty.String(), UnitPrice: it.UnitPrice.String(), LineTotal: it.LineTotal.String()})
	}
	return out
}

func documentFrom(r *financev1.DocumentInput) (domain.Document, error) {
	d := domain.Document{ID: r.Id, Kind: r.Kind, Side: r.Side, VendorID: r.VendorId, PartyName: r.PartyName, PicName: r.PicName, Phone: r.Phone, Email: r.Email, Address: r.Address, Title: r.Title,
		IssueDate: r.IssueDate, DueDate: r.DueDate, ExternalNumber: r.ExternalNumber, Notes: r.Notes, AccountID: r.AccountId}
	var err error
	if d.DiscountPercent, err = money("discount_percent", orZero(r.DiscountPercent)); err != nil {
		return d, err
	}
	if d.PPNPercent, err = money("ppn_percent", orZero(r.PpnPercent)); err != nil {
		return d, err
	}
	for _, it := range r.Items {
		q, err := money("items", orZero(it.Qty))
		if err != nil {
			return d, err
		}
		price, err := money("items", orZero(it.UnitPrice))
		if err != nil {
			return d, err
		}
		d.Items = append(d.Items, domain.DocItem{Description: it.Description, Unit: it.Unit, Qty: q, UnitPrice: price})
	}
	return d, nil
}

func (s *documentServer) ListVendors(ctx context.Context, r *financev1.ListVendorsRequest) (*financev1.ListVendorsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Documents.ListVendors(ctx, r.Q, r.ActiveOnly)
	if err != nil {
		return nil, err
	}
	out := &financev1.ListVendorsResponse{}
	for i := range rows {
		out.Vendors = append(out.Vendors, vendorPB(&rows[i]))
	}
	return out, nil
}

func (s *documentServer) SaveVendor(ctx context.Context, r *financev1.VendorInput) (*financev1.Vendor, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	v, err := s.svc.Documents.SaveVendor(ctx, p.UserID, domain.Vendor{ID: r.Id, Name: r.Name, PicName: r.PicName, Phone: r.Phone, Email: r.Email, Address: r.Address, NPWP: r.Npwp, BankName: r.BankName,
		BankAccountNo: r.BankAccountNo, BankAccountName: r.BankAccountName, PaymentTermsDays: int(r.PaymentTermsDays), Notes: r.Notes, IsActive: r.IsActive})
	if err != nil {
		return nil, err
	}
	return vendorPB(v), nil
}

func (s *documentServer) ListDocuments(ctx context.Context, r *financev1.ListDocumentsRequest) (*financev1.ListDocumentsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, total, err := s.svc.Documents.List(ctx, domain.DocumentFilter{Kind: r.Kind, Status: r.Status, Query: r.Q}, int(r.Page), int(r.PageSize))
	if err != nil {
		return nil, err
	}
	out := &financev1.ListDocumentsResponse{Total: int32(total)}
	for i := range rows {
		out.Documents = append(out.Documents, documentPB(&rows[i]))
	}
	return out, nil
}

func (s *documentServer) GetDocument(ctx context.Context, r *financev1.IdRequest) (*financev1.FinanceDocument, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	d, err := s.svc.Documents.Get(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	return documentPB(d), nil
}

func (s *documentServer) CreateDocument(ctx context.Context, r *financev1.DocumentInput) (*financev1.FinanceDocument, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	in, err := documentFrom(r)
	if err != nil {
		return nil, err
	}
	d, err := s.svc.Documents.Create(ctx, p.UserID, in)
	if err != nil {
		return nil, err
	}
	return documentPB(d), nil
}

func (s *documentServer) UpdateDocument(ctx context.Context, r *financev1.DocumentInput) (*financev1.FinanceDocument, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	in, err := documentFrom(r)
	if err != nil {
		return nil, err
	}
	d, err := s.svc.Documents.Update(ctx, p.UserID, in)
	if err != nil {
		return nil, err
	}
	return documentPB(d), nil
}

func (s *documentServer) DeleteDocument(ctx context.Context, r *financev1.IdRequest) (*emptypb.Empty, error) {
	if _, err := write(ctx); err != nil {
		return nil, err
	}
	return empty, s.svc.Documents.Delete(ctx, r.Id)
}

func (s *documentServer) ActOnDocument(ctx context.Context, r *financev1.ActOnDocumentRequest) (*financev1.FinanceDocument, error) {
	var (
		p   authn.Principal
		err error
	)
	if domain.DocNeedsApprover(r.Action) {
		p, err = approve(ctx)
	} else {
		p, err = write(ctx)
	}
	if err != nil {
		return nil, err
	}
	d, err := s.svc.Documents.Act(ctx, p, r.Id, r.Action, r.Reason)
	if err != nil {
		return nil, err
	}
	return documentPB(d), nil
}

func (s *documentServer) ConvertDocument(ctx context.Context, r *financev1.ConvertDocumentRequest) (*financev1.FinanceDocument, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.svc.Documents.Convert(ctx, p.UserID, r.Id, r.VendorId)
	if err != nil {
		return nil, err
	}
	return documentPB(d), nil
}

func (s *documentServer) IssueDocument(ctx context.Context, r *financev1.IssueDocumentRequest) (*financev1.FinanceDocument, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.svc.Documents.Issue(ctx, p.UserID, r.Id, r.AccountId, r.DueDate)
	if err != nil {
		return nil, err
	}
	return documentPB(d), nil
}

func (s *documentServer) RenderDocumentPdf(ctx context.Context, r *financev1.IdRequest) (*financev1.DocumentPdf, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	name, data, err := s.svc.Documents.PDF(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	return &financev1.DocumentPdf{FileName: name, ContentType: "application/pdf", Data: data}, nil
}
