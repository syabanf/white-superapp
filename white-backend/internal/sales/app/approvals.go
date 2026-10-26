package app

import (
	"context"
	"fmt"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// The quotation discount approval flow of crm/advance approvals.go.

// pendingApproval is who must be told after the transaction commits.
type pendingApproval struct {
	requestID string
	level     int
}

// requestApproval builds the approval request from the active rules
// (RequiredApprovalLevels). No rule matched means no approval needed.
func (q *Quotes) requestApproval(ctx context.Context, st domain.Store, qt *domain.Quotation, actor Actor) (*pendingApproval, error) {
	rules, err := st.ListApprovalRules(ctx, true)
	if err != nil {
		return nil, err
	}
	levels := domain.RequiredApprovalLevels(qt.DiscountPercent, rules)
	if len(levels) == 0 {
		return nil, nil
	}
	req := domain.ApprovalRequest{ID: ids.New(), Object: "quotation", SubjectID: qt.ID, Status: "pending", CurrentLevel: levels[0].Level, DiscountPercent: qt.DiscountPercent, Amount: qt.Total, RequestedBy: actor.ref(), CreatedAt: q.now()}
	for _, r := range levels {
		req.Steps = append(req.Steps, domain.ApprovalStep{ID: ids.New(), RequestID: req.ID, Level: r.Level, ApproverRole: r.ApproverRole, ApproverUserID: r.ApproverUserID, Status: "pending"})
	}
	if err := st.CreateApprovalRequest(ctx, req); err != nil {
		return nil, err
	}
	if err := st.SetQuotationApproval(ctx, qt.ID, domain.ApprovalPending, &req.ID); err != nil {
		return nil, err
	}
	qt.ApprovalStatus, qt.ApprovalRequestID = domain.ApprovalPending, &req.ID
	return &pendingApproval{requestID: req.ID, level: req.CurrentLevel}, nil
}

// notifyApprovers tells the approvers of a level (in-app).
func (q *Quotes) notifyApprovers(ctx context.Context, pa *pendingApproval) {
	if pa == nil {
		return
	}
	req, err := q.Store.GetApprovalRequest(ctx, pa.requestID, false)
	if err != nil || req == nil {
		return
	}
	var recipients []string
	for _, st := range req.Steps {
		if st.Level != pa.level {
			continue
		}
		if st.ApproverUserID != nil {
			recipients = []string{*st.ApproverUserID}
			break
		}
		if q.Directory != nil {
			users, err := q.Directory.Approvers(ctx)
			if err != nil {
				q.warn("list approvers failed", "err", err)
			}
			for _, u := range users {
				if deref(st.ApproverRole) == domain.ApproverPerm || u.Admin {
					recipients = append(recipients, u.ID)
				}
			}
		}
	}
	title := fmt.Sprintf("Approval diskon %s%% - %s", domain.FormatNumber(req.DiscountPercent, 2), req.QuoteNumber)
	body := fmt.Sprintf("%s · %s · total %s (tingkat %d)", req.ClientName, req.DealTitle, req.Amount.Rupiah(), pa.level)
	q.notify(ctx, Notification{UserIDs: recipients, Type: NotifApproval, Title: title, Body: body, Href: "/erp/sales/approvals?request=" + req.ID})
}

func (q *Quotes) ListApprovalRules(ctx context.Context) ([]domain.ApprovalRule, error) {
	return q.Store.ListApprovalRules(ctx, false)
}

func (q *Quotes) SaveApprovalRule(ctx context.Context, r domain.ApprovalRule) (*domain.ApprovalRule, error) {
	errs := map[string]string{}
	if domain.JSTrim(r.Name) == "" {
		errs["name"] = "Nama aturan wajib diisi"
	}
	if r.Level < 1 || r.Level > 5 {
		errs["level"] = "Tingkat harus 1-5"
	}
	if r.MinDiscount < 0 || r.MinDiscount > 100 {
		errs["minDiscountPercent"] = "Batas diskon harus 0-100%"
	}
	role := deref(r.ApproverRole)
	if deref(r.ApproverUserID) == "" && role != domain.ApproverPerm && role != domain.ApproverAdmin {
		errs["approverRole"] = "Pilih approver: sales.approve, ADMIN, atau pengguna tertentu"
	}
	if err := fieldErrs(errs); err != nil {
		return nil, err
	}
	if r.ID == "" {
		r.ID = ids.New()
	}
	r.ApproverRole, r.ApproverUserID = domain.TrimmedOrNil(role), domain.TrimmedOrNil(deref(r.ApproverUserID))
	return &r, q.Store.SaveApprovalRule(ctx, r)
}

func (q *Quotes) DeleteApprovalRule(ctx context.Context, id string) error {
	return q.Store.DeleteApprovalRule(ctx, id)
}

// ApprovalView is one inbox row with the caller's right to decide.
type ApprovalView struct {
	domain.ApprovalRequest
	CanDecide bool
}

func canDecide(req domain.ApprovalRequest, actor Actor) bool {
	if req.Status != "pending" {
		return false
	}
	for _, st := range req.Steps {
		if st.Level == req.CurrentLevel && st.Status == "pending" {
			return domain.CanDecideStep(st.ApproverRole, st.ApproverUserID, actor.UserID, actor.Admin, actor.Approver)
		}
	}
	return false
}

// ListApprovals is the approval inbox. view=mine (default): waiting on me
// at the current level, or requested by me; view=all for approvers.
func (q *Quotes) ListApprovals(ctx context.Context, actor Actor, view, status string) ([]ApprovalView, error) {
	if status == "" {
		status = "pending"
	}
	if status == "all" {
		status = ""
	}
	reqs, err := q.Store.ListApprovalRequests(ctx, status, 200)
	if err != nil {
		return nil, err
	}
	all := view == "all" && (actor.Admin || actor.Approver)
	out := []ApprovalView{}
	for _, r := range reqs {
		can := canDecide(r, actor)
		if !all && !can && deref(r.RequestedBy) != actor.UserID {
			continue
		}
		out = append(out, ApprovalView{ApprovalRequest: r, CanDecide: can})
	}
	return out, nil
}

// DecisionResult is decideApproval's answer.
type DecisionResult struct {
	Status    string
	NextLevel int
	Message   string
}

// Decide records an approver's decision on the current level; the request
// row is locked so two approvers cannot decide the same level at once.
func (q *Quotes) Decide(ctx context.Context, actor Actor, id, decision, comment string) (*DecisionResult, error) {
	if decision != "approve" && decision != "reject" {
		return nil, apperr.Field("decision", "Keputusan harus approve atau reject")
	}
	var res DecisionResult
	var requester *string
	var quoteNumber, dealID, subjectID string
	err := q.Store.WithTx(ctx, func(st domain.Store) error {
		req, err := st.GetApprovalRequest(ctx, id, true)
		if err != nil {
			return err
		}
		if req == nil {
			return notFound("Permintaan approval tidak ditemukan")
		}
		if req.Status != "pending" {
			return conflict("Permintaan sudah diputuskan")
		}
		var step *domain.ApprovalStep
		for i := range req.Steps {
			if req.Steps[i].Level == req.CurrentLevel && req.Steps[i].Status == "pending" {
				step = &req.Steps[i]
			}
		}
		if step == nil {
			return conflict("Tingkat approval tidak ditemukan")
		}
		if !domain.CanDecideStep(step.ApproverRole, step.ApproverUserID, actor.UserID, actor.Admin, actor.Approver) {
			return apperr.New(apperr.Unauthorized, "Anda bukan approver tingkat ini")
		}
		now := q.now()
		step.Status, step.DecidedBy, step.DecidedAt, step.Comment = "rejected", actor.ref(), &now, domain.TrimmedOrNil(comment)
		if decision == "approve" {
			step.Status = "approved"
		}
		if err := st.SaveApprovalStep(ctx, *step); err != nil {
			return err
		}
		requester, quoteNumber, dealID, subjectID = req.RequestedBy, req.QuoteNumber, req.DealID, req.SubjectID
		resolve := func(status string) error {
			req.Status, req.ResolvedBy, req.ResolvedAt = status, actor.ref(), &now
			if err := st.SaveApprovalRequest(ctx, *req); err != nil {
				return err
			}
			res.Status = status
			return st.SetQuotationApproval(ctx, req.SubjectID, status, &req.ID)
		}
		if decision == "reject" {
			res.Message = "Ditolak"
			return resolve("rejected")
		}
		next := 0
		for _, s := range req.Steps {
			if s.Status == "pending" && s.Level > req.CurrentLevel && (next == 0 || s.Level < next) {
				next = s.Level
			}
		}
		if next == 0 {
			res.Message = "Disetujui, quotation boleh dikirim"
			return resolve("approved")
		}
		req.CurrentLevel = next
		res.Status, res.NextLevel = "pending", next
		res.Message = fmt.Sprintf("Disetujui tingkat ini, lanjut ke tingkat %d", next)
		return st.SaveApprovalRequest(ctx, *req)
	})
	if err != nil {
		return nil, err
	}
	if res.NextLevel > 0 {
		q.notifyApprovers(ctx, &pendingApproval{requestID: id, level: res.NextLevel})
	} else if requester != nil {
		title, body := "Diskon "+quoteNumber+" DITOLAK", "Ditolak oleh approver"
		if res.Status == "approved" {
			title, body = "Diskon "+quoteNumber+" DISETUJUI", "Quotation boleh dikirim"
		}
		if c := domain.JSTrim(comment); c != "" {
			body = c
		}
		q.notify(ctx, Notification{UserIDs: []string{*requester}, Type: NotifApproval, Title: title, Body: body, Href: "/erp/sales/pipeline?deal=" + dealID})
	}
	q.auto.Emit(ctx, Event{Type: "quotation.status_changed", SubjectType: "quotation", SubjectID: subjectID, Actor: actor.ref(), Payload: map[string]any{"approval": res.Status, "decision": decision}})
	return &res, nil
}
