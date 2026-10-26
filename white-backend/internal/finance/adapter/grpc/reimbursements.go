package grpcapi

import (
	"context"

	financev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/finance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ─── ReimbursementService ───────────────────────────────────

type reimbursementServer struct {
	financev1.UnimplementedReimbursementServiceServer
	svc Services
}

func selfService(ctx context.Context) (authn.Principal, error) {
	return authn.RequirePerm(ctx, authn.PermSelfService)
}

// approver: any of the approval paths (manager rights are checked per claim).
func approver(ctx context.Context) (authn.Principal, error) { return authn.Require(ctx) }

func claimPB(p authn.Principal, c *domain.Claim) *financev1.Claim {
	out := &financev1.Claim{
		Id: c.ID, ClaimNo: c.ClaimNo, EmployeeId: c.EmployeeID, EmployeeUserId: c.EmployeeUserID, EmployeeName: c.EmployeeName,
		DepartmentName: c.DepartmentName, ManagerEmployeeId: c.ManagerEmployeeID, Category: categoryPB(c.Category), TransactionDate: c.TransactionDate,
		Amount: c.Amount.String(), Description: c.Description, ClientId: c.ClientID, Project: c.Project, Status: claimStatusPB(c.Status),
		OverLimit: c.OverLimit, LimitNote: c.LimitNote, RejectReason: c.RejectReason, RejectedStage: c.RejectedStage, SubmittedAt: tsPtr(c.SubmittedAt),
		PaidAt: tsPtr(c.PaidAt), PaymentBatchId: c.PaymentBatchID, CreatedAt: ts(c.CreatedAt), UpdatedAt: ts(c.UpdatedAt),
		CanEdit: app.CanEditClaim(p, c), CanApprove: app.CanApprove(p, c),
	}
	for _, r := range c.Receipts {
		out.Receipts = append(out.Receipts, &financev1.ClaimReceipt{Id: r.ID, Filename: r.Filename, ContentType: r.ContentType, SizeBytes: r.SizeBytes, CreatedAt: ts(r.CreatedAt)})
	}
	for _, h := range c.History {
		out.History = append(out.History, &financev1.ClaimHistory{Action: h.Action, ActorId: h.ActorID, ActorName: h.ActorName, Note: h.Note, CreatedAt: ts(h.CreatedAt)})
	}
	return out
}

func claimsPB(p authn.Principal, rows []domain.Claim, total int, sum domain.Money) *financev1.ListClaimsResponse {
	out := &financev1.ListClaimsResponse{Total: int32(total), TotalAmount: sum.String()}
	for i := range rows {
		out.Claims = append(out.Claims, claimPB(p, &rows[i]))
	}
	return out
}

func (s *reimbursementServer) ListMyClaims(ctx context.Context, req *financev1.ListMyClaimsRequest) (*financev1.ListClaimsResponse, error) {
	p, err := selfService(ctx)
	if err != nil {
		return nil, err
	}
	rows, total, sum, err := s.svc.Reimbursements.MyClaims(ctx, p, claimStatusFrom(req.Status))
	if err != nil {
		return nil, err
	}
	return claimsPB(p, rows, total, sum), nil
}

func (s *reimbursementServer) CreateClaim(ctx context.Context, req *financev1.ClaimInput) (*financev1.Claim, error) {
	p, err := selfService(ctx)
	if err != nil {
		return nil, err
	}
	in, err := claimFrom(req)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Reimbursements.Create(ctx, p, in)
	if err != nil {
		return nil, err
	}
	return claimPB(p, c), nil
}

func (s *reimbursementServer) UpdateClaim(ctx context.Context, req *financev1.UpdateClaimRequest) (*financev1.Claim, error) {
	p, err := selfService(ctx)
	if err != nil {
		return nil, err
	}
	in, err := claimFrom(req.Input)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Reimbursements.Update(ctx, p, req.Id, in)
	if err != nil {
		return nil, err
	}
	return claimPB(p, c), nil
}

func (s *reimbursementServer) UploadReceipt(ctx context.Context, req *financev1.UploadReceiptRequest) (*financev1.Claim, error) {
	p, err := selfService(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Reimbursements.UploadReceipt(ctx, p, req.Id, req.Filename, req.Data)
	if err != nil {
		return nil, err
	}
	return claimPB(p, c), nil
}

func (s *reimbursementServer) DeleteReceipt(ctx context.Context, req *financev1.DeleteReceiptRequest) (*financev1.Claim, error) {
	p, err := selfService(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Reimbursements.DeleteReceipt(ctx, p, req.Id, req.ReceiptId)
	if err != nil {
		return nil, err
	}
	return claimPB(p, c), nil
}

func (s *reimbursementServer) SubmitClaim(ctx context.Context, req *financev1.IdRequest) (*financev1.Claim, error) {
	p, err := selfService(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Reimbursements.Submit(ctx, p, req.Id)
	if err != nil {
		return nil, err
	}
	return claimPB(p, c), nil
}

func (s *reimbursementServer) CancelClaim(ctx context.Context, req *financev1.IdRequest) (*financev1.Claim, error) {
	p, err := selfService(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Reimbursements.Cancel(ctx, p, req.Id)
	if err != nil {
		return nil, err
	}
	return claimPB(p, c), nil
}

func (s *reimbursementServer) GetClaim(ctx context.Context, req *financev1.IdRequest) (*financev1.Claim, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Reimbursements.Get(ctx, p, req.Id)
	if err != nil {
		return nil, err
	}
	return claimPB(p, c), nil
}

func (s *reimbursementServer) ListApprovalInbox(ctx context.Context, _ *emptypb.Empty) (*financev1.ListClaimsResponse, error) {
	p, err := approver(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.svc.Reimbursements.Inbox(ctx, p)
	if err != nil {
		return nil, err
	}
	var sum domain.Money
	for _, c := range rows {
		sum += c.Amount
	}
	return claimsPB(p, rows, len(rows), sum), nil
}

func (s *reimbursementServer) ApproveClaim(ctx context.Context, req *financev1.ApproveClaimRequest) (*financev1.Claim, error) {
	p, err := approver(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Reimbursements.Approve(ctx, p, req.Id, req.Note)
	if err != nil {
		return nil, err
	}
	return claimPB(p, c), nil
}

func (s *reimbursementServer) RejectClaim(ctx context.Context, req *financev1.RejectClaimRequest) (*financev1.Claim, error) {
	p, err := approver(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Reimbursements.Reject(ctx, p, req.Id, req.Reason)
	if err != nil {
		return nil, err
	}
	return claimPB(p, c), nil
}

func (s *reimbursementServer) ListClaims(ctx context.Context, req *financev1.ListClaimsRequest) (*financev1.ListClaimsResponse, error) {
	p, err := read(ctx)
	if err != nil {
		return nil, err
	}
	limit, offset := app.Page(int(req.Page), int(req.PageSize))
	f := domain.ClaimFilter{Category: categoryFrom(req.Category), EmployeeID: req.EmployeeId, Search: req.Search, Department: req.Department,
		MonthFrom: req.DateFrom, MonthTo: req.DateTo, Limit: limit, Offset: offset}
	if st := claimStatusFrom(req.Status); st != "" {
		f.Statuses = []domain.ClaimStatus{st}
	}
	rows, total, sum, err := s.svc.Reimbursements.List(ctx, f, req.Month)
	if err != nil {
		return nil, err
	}
	return claimsPB(p, rows, total, sum), nil
}

func (s *reimbursementServer) PayClaims(ctx context.Context, req *financev1.PayClaimsRequest) (*financev1.ReimbursementBatch, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	b, err := s.svc.Reimbursements.Pay(ctx, p, app.PayInput{ClaimIDs: req.ClaimIds, CashAccountID: req.CashAccountId, PaidAt: req.PaidAt, Notes: req.Notes})
	if err != nil {
		return nil, err
	}
	return batchPB(b), nil
}

func (s *reimbursementServer) ListPaymentBatches(ctx context.Context, req *financev1.ListPaymentBatchesRequest) (*financev1.ListPaymentBatchesResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Reimbursements.Batches(ctx, int(req.Limit))
	if err != nil {
		return nil, err
	}
	out := &financev1.ListPaymentBatchesResponse{}
	for i := range rows {
		out.Batches = append(out.Batches, batchPB(&rows[i]))
	}
	return out, nil
}

func (s *reimbursementServer) GetMonthlySummary(ctx context.Context, req *financev1.MonthlySummaryRequest) (*financev1.MonthlySummary, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	if !p.Can(authn.PermFinanceRead) && !p.Can(authn.PermSelfService) {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	month, rows, err := s.svc.Reimbursements.MonthlySummary(ctx, p, req.Month, req.Mine)
	if err != nil {
		return nil, err
	}
	out := &financev1.MonthlySummary{Month: month}
	var sub, appr, paid domain.Money
	for _, r := range rows {
		sub, appr, paid = sub+r.SubmittedAmount, appr+r.ApprovedAmount, paid+r.PaidAmount
		out.Rows = append(out.Rows, &financev1.MonthlySummaryRow{EmployeeId: r.EmployeeID, EmployeeName: r.EmployeeName, DepartmentName: r.DepartmentName,
			Category: categoryPB(r.Category), ClaimCount: int32(r.ClaimCount), SubmittedAmount: r.SubmittedAmount.String(),
			ApprovedAmount: r.ApprovedAmount.String(), PaidAmount: r.PaidAmount.String()})
	}
	out.TotalSubmitted, out.TotalApproved, out.TotalPaid = sub.String(), appr.String(), paid.String()
	return out, nil
}

func (s *reimbursementServer) ListPolicies(ctx context.Context, _ *emptypb.Empty) (*financev1.ListPoliciesResponse, error) {
	if _, err := authn.Require(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Reimbursements.Policies(ctx)
	if err != nil {
		return nil, err
	}
	names, err := s.accountNames(ctx)
	if err != nil {
		return nil, err
	}
	out := &financev1.ListPoliciesResponse{}
	for _, pol := range rows {
		out.Policies = append(out.Policies, policyPB(pol, names[pol.ExpenseAccountID]))
	}
	return out, nil
}

func (s *reimbursementServer) accountNames(ctx context.Context) (map[string]string, error) {
	rows, err := s.svc.Accounts.ListAccounts(ctx, domain.AccountFilter{})
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, a := range rows {
		out[a.ID] = a.Name
	}
	return out, nil
}

func policyPB(p domain.Policy, accountName string) *financev1.ReimbursementPolicy {
	return &financev1.ReimbursementPolicy{Category: categoryPB(p.Category), Label: p.Label, MonthlyLimit: optionalMoney(p.MonthlyLimit),
		PerClaimLimit: optionalMoney(p.PerClaimLimit), ExpenseAccountId: p.ExpenseAccountID, ExpenseAccountName: accountName,
		IsActive: p.IsActive, AutoApprove: p.AutoApprove}
}

func (s *reimbursementServer) UpdatePolicy(ctx context.Context, req *financev1.PolicyInput) (*financev1.ReimbursementPolicy, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	monthly, err := money("monthly_limit", req.MonthlyLimit)
	if err != nil {
		return nil, err
	}
	perClaim, err := money("per_claim_limit", req.PerClaimLimit)
	if err != nil {
		return nil, err
	}
	pol, err := s.svc.Reimbursements.UpdatePolicy(ctx, p.UserID, domain.Policy{Category: categoryFrom(req.Category), MonthlyLimit: monthly, PerClaimLimit: perClaim,
		ExpenseAccountID: req.ExpenseAccountId, AutoApprove: req.AutoApprove, IsActive: req.IsActive})
	if err != nil {
		return nil, err
	}
	names, err := s.accountNames(ctx)
	if err != nil {
		return nil, err
	}
	return policyPB(*pol, names[pol.ExpenseAccountID]), nil
}

func (s *reimbursementServer) GetReceiptFile(ctx context.Context, req *financev1.GetReceiptFileRequest) (*financev1.ReceiptFile, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	rec, data, etag, err := s.svc.Reimbursements.ReceiptFile(ctx, p, req.ReceiptId)
	if err != nil {
		return nil, err
	}
	return &financev1.ReceiptFile{Data: data, ContentType: rec.ContentType, Filename: rec.Filename, Etag: etag}, nil
}
