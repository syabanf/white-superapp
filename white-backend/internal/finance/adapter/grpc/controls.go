package grpcapi

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	financev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/finance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
)

func (s *periodEndServer) ListSalesReversals(ctx context.Context, _ *financev1.ListSalesReversalsRequest) (*financev1.ListSalesReversalsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Ledgers.PendingSalesReversals(ctx)
	if err != nil {
		return nil, err
	}
	out := &financev1.ListSalesReversalsResponse{}
	for _, r := range rows {
		out.Reversals = append(out.Reversals, &financev1.SalesReversal{Kind: r.Kind, Id: r.ID, DocNo: r.DocNo, PartyName: r.PartyName, Amount: r.Amount.String(), Reason: r.Reason, Error: r.Error})
	}
	return out, nil
}

func (s *periodEndServer) ReverseJournalEntry(ctx context.Context, r *financev1.ReverseJournalRequest) (*financev1.JournalEntry, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	e, err := s.svc.Journals.Reverse(ctx, p.UserID, r.Id, r.ReversalDate, r.Reason)
	if err != nil {
		return nil, err
	}
	return entryPB(e), nil
}

func (s *periodEndServer) DisposeFixedAsset(ctx context.Context, r *financev1.DisposeAssetRequest) (*financev1.FixedAsset, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	proceeds, err := money("proceeds", orZero(r.Proceeds))
	if err != nil {
		return nil, err
	}
	a, err := s.svc.Schedules.DisposeAsset(ctx, p.UserID, r.Id, app.DisposalInput{Date: r.DisposalDate, AssetAccountID: r.AssetAccountId, CashAccountID: r.CashAccountId, GainLossAccountID: r.GainLossAccountId, Reason: r.Reason, Proceeds: proceeds})
	if err != nil {
		return nil, err
	}
	return assetPB(a), nil
}

func goodsReceiptPB(r *domain.GoodsReceipt) *financev1.GoodsReceipt {
	return &financev1.GoodsReceipt{Id: r.ID, ReceiptNo: r.ReceiptNo, DocumentId: r.DocumentID, ReceiptDate: r.Date, Amount: r.Amount.String(), Notes: r.Notes, Status: r.Status, VoidReason: r.VoidReason}
}

func (s *periodEndServer) ListPoMatches(ctx context.Context, _ *financev1.ListPoMatchesRequest) (*financev1.ListPoMatchesResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Ledgers.ListPOMatches(ctx)
	if err != nil {
		return nil, err
	}
	out := &financev1.ListPoMatchesResponse{}
	for _, m := range rows {
		row := &financev1.PoMatch{DocumentId: m.Document.ID, Number: m.Document.Number, PartyName: m.Document.PartyName, Title: m.Document.Title, Ordered: m.Ordered.String(), Received: m.Received.String(), Billed: m.Billed.String()}
		for i := range m.Receipts {
			row.Receipts = append(row.Receipts, goodsReceiptPB(&m.Receipts[i]))
		}
		for _, inv := range m.Invoices {
			row.Invoices = append(row.Invoices, &financev1.PoInvoice{InvoiceId: inv.InvoiceID, InvoiceNo: inv.InvoiceNo, Status: inv.Status, Subtotal: inv.Subtotal.String()})
		}
		out.Matches = append(out.Matches, row)
	}
	return out, nil
}

func (s *periodEndServer) ReceiveGoods(ctx context.Context, r *financev1.GoodsReceipt) (*financev1.GoodsReceipt, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	amount, err := money("amount", orZero(r.Amount))
	if err != nil {
		return nil, err
	}
	out, err := s.svc.Ledgers.ReceiveGoods(ctx, p.UserID, domain.GoodsReceipt{DocumentID: r.DocumentId, Date: r.ReceiptDate, Amount: amount, Notes: r.Notes})
	if err != nil {
		return nil, err
	}
	return goodsReceiptPB(out), nil
}

func (s *periodEndServer) VoidGoodsReceipt(ctx context.Context, r *financev1.VoidAdvanceRequest) (*emptypb.Empty, error) {
	if _, err := approve(ctx); err != nil {
		return nil, err
	}
	return empty, s.svc.Ledgers.VoidGoodsReceipt(ctx, r.Id, r.Reason)
}

func (s *periodEndServer) MatchInvoiceToPo(ctx context.Context, r *financev1.MatchInvoiceRequest) (*emptypb.Empty, error) {
	if _, err := write(ctx); err != nil {
		return nil, err
	}
	return empty, s.svc.Ledgers.MatchInvoiceToPO(ctx, r.InvoiceId, r.DocumentId)
}

func (s *periodEndServer) ListProjectAllocations(ctx context.Context, r *financev1.ProjectProfitRequest) (*financev1.ListProjectAllocationsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.ProjectTags.ListAllocations(ctx, r.DateFrom, r.DateTo)
	if err != nil {
		return nil, err
	}
	return allocationsPB(rows), nil
}

func allocationsPB(rows []domain.ProjectAllocation) *financev1.ListProjectAllocationsResponse {
	out := &financev1.ListProjectAllocationsResponse{}
	for _, a := range rows {
		out.Allocations = append(out.Allocations, &financev1.ProjectAllocation{Id: a.ID, TagId: a.TagID, TagName: a.TagName, ClientName: a.ClientName, Month: a.Month, Kind: a.Kind, Amount: a.Amount.String(), Notes: a.Notes})
	}
	return out
}

func (s *periodEndServer) CreateProjectAllocation(ctx context.Context, r *financev1.ProjectAllocation) (*financev1.ListProjectAllocationsResponse, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	amount, err := money("amount", orZero(r.Amount))
	if err != nil {
		return nil, err
	}
	rows, err := s.svc.ProjectTags.CreateAllocation(ctx, p.UserID, domain.ProjectAllocation{TagID: r.TagId, Month: r.Month, Kind: r.Kind, Amount: amount, Notes: r.Notes})
	if err != nil {
		return nil, err
	}
	return allocationsPB(rows), nil
}

func (s *periodEndServer) DeleteProjectAllocation(ctx context.Context, r *financev1.IdRequest) (*emptypb.Empty, error) {
	if _, err := approve(ctx); err != nil {
		return nil, err
	}
	return empty, s.svc.ProjectTags.DeleteAllocation(ctx, r.Id)
}
