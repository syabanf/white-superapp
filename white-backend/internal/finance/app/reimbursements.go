package app

import (
	"context"
	"fmt"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// Reimbursements is the employee claim flow (new for WHITE):
//
//	DRAFT -> SUBMITTED -> APPROVED_MANAGER -> APPROVED_FINANCE -> PAID
//
// The manager step needs the claimant's manager (hr manager_id) or
// hr.approve; the finance step needs finance.approve. A claimant without a
// manager skips the manager step, and a claim within the limits of a
// policy with auto_approve skips the finance step. Claims above the limits
// always wait for a finance.approve holder.
type Reimbursements struct{ Deps }

func NewReimbursements(d Deps) *Reimbursements { return &Reimbursements{d} }

// Notification types and history actions.
const (
	NotifClaimSubmitted = "REIMBURSEMENT_SUBMITTED"
	NotifClaimApproved  = "REIMBURSEMENT_APPROVED"
	NotifClaimRejected  = "REIMBURSEMENT_REJECTED"
	NotifClaimPaid      = "REIMBURSEMENT_PAID"
	NotifClaimReminder  = "REIMBURSEMENT_REMINDER"
	maxReceipts         = 10
)

func claimHref(id string) string { return "/erp/finance/reimbursements/" + id }

// ─── access rules ───────────────────────────────────────────

func isOwner(p authn.Principal, c *domain.Claim) bool { return c.EmployeeUserID == p.UserID }

func isManagerOf(p authn.Principal, c *domain.Claim) bool {
	return c.ManagerUserID != "" && c.ManagerUserID == p.UserID
}

// CanApprove reports whether p may act on the claim's current step.
func CanApprove(p authn.Principal, c *domain.Claim) bool {
	if isOwner(p, c) && !p.IsAdmin() {
		return false
	}
	switch c.Status {
	case domain.ClaimSubmitted:
		return isManagerOf(p, c) || p.Can(authn.PermHRApprove)
	case domain.ClaimApprovedManager:
		return p.Can(authn.PermFinanceApprove)
	}
	return false
}

func canView(p authn.Principal, c *domain.Claim) bool {
	return isOwner(p, c) || isManagerOf(p, c) || p.Can(authn.PermFinanceRead) || p.Can(authn.PermHRApprove)
}

// CanEditClaim: the owner edits drafts.
func CanEditClaim(p authn.Principal, c *domain.Claim) bool {
	return isOwner(p, c) && c.Status == domain.ClaimDraft
}

func (r *Reimbursements) load(ctx context.Context, p authn.Principal, id string) (*domain.Claim, error) {
	c, err := r.Store.GetClaim(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil || !canView(p, c) {
		return nil, domain.NotFound(domain.MsgClaimNotFound)
	}
	return c, nil
}

func (r *Reimbursements) loadOwnDraft(ctx context.Context, p authn.Principal, id string) (*domain.Claim, error) {
	c, err := r.load(ctx, p, id)
	if err != nil {
		return nil, err
	}
	if !isOwner(p, c) {
		return nil, domain.NotFound(domain.MsgClaimNotFound)
	}
	if c.Status != domain.ClaimDraft {
		return nil, domain.Bad(domain.MsgClaimNotEditable)
	}
	return c, nil
}

func (r *Reimbursements) Get(ctx context.Context, p authn.Principal, id string) (*domain.Claim, error) {
	return r.load(ctx, p, id)
}

func (r *Reimbursements) actorName(ctx context.Context, p authn.Principal) string {
	if r.Identity == nil {
		return ""
	}
	return r.Identity.UserName(ctx, p.UserID)
}

func (r *Reimbursements) history(ctx context.Context, s domain.Store, p authn.Principal, claimID, action, note string) error {
	return s.InsertClaimEvent(ctx, domain.ClaimEvent{ClaimID: claimID, Action: action, ActorID: p.UserID, ActorName: r.actorName(ctx, p), Note: note})
}

// ─── self-service ───────────────────────────────────────────

func (r *Reimbursements) MyClaims(ctx context.Context, p authn.Principal, status domain.ClaimStatus) ([]domain.Claim, int, domain.Money, error) {
	f := domain.ClaimFilter{EmployeeUserID: p.UserID}
	if status != "" {
		f.Statuses = []domain.ClaimStatus{status}
	}
	return r.Store.ListClaims(ctx, f)
}

// ClaimInput is the claim form.
type ClaimInput struct {
	Category, TransactionDate, Description, ClientID, Project string
	Amount                                                    domain.Money
}

func (r *Reimbursements) normalize(in *ClaimInput) error {
	in.Description, in.Project, in.ClientID = trim(in.Description), trim(in.Project), trim(in.ClientID)
	v := validate.New().Required("description", in.Description, "Keterangan wajib diisi").MaxLen("description", in.Description, 500, "Keterangan maksimal 500 karakter")
	if !domain.IsClaimCategory(in.Category) {
		v.Add("category", "Kategori tidak valid")
	}
	if in.Amount <= 0 {
		v.Add("amount", "Nominal harus lebih dari 0")
	}
	if err := v.Err(); err != nil {
		return err
	}
	day, err := parseDay("transaction_date", in.TransactionDate)
	if err != nil {
		return err
	}
	in.TransactionDate = day
	if day > r.today() {
		return apperr.Field("transaction_date", domain.MsgFutureDate)
	}
	return nil
}

// claimant resolves the caller's employee record and their manager's user.
func (r *Reimbursements) claimant(ctx context.Context, userID string) (*Employee, string, error) {
	emp, err := r.HR.EmployeeByUser(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	if emp == nil {
		return nil, "", domain.Bad(domain.MsgNoEmployee)
	}
	managerUser := ""
	if emp.ManagerID != "" {
		if mgr, err := r.HR.Employee(ctx, emp.ManagerID); err == nil && mgr != nil {
			managerUser = mgr.UserID
		} else if err != nil && r.Logger != nil {
			r.Logger.Warn("manager lookup failed", "employee", emp.ID, "err", err)
		}
	}
	return emp, managerUser, nil
}

func (r *Reimbursements) Create(ctx context.Context, p authn.Principal, in ClaimInput) (*domain.Claim, error) {
	if err := r.normalize(&in); err != nil {
		return nil, err
	}
	emp, managerUser, err := r.claimant(ctx, p.UserID)
	if err != nil {
		return nil, err
	}
	c := domain.Claim{
		ID: ids.New(), EmployeeID: emp.ID, EmployeeUserID: p.UserID, EmployeeName: emp.Name, DepartmentName: emp.DepartmentName,
		ManagerEmployeeID: emp.ManagerID, ManagerUserID: managerUser, Category: in.Category, TransactionDate: in.TransactionDate,
		Amount: in.Amount, Description: in.Description, ClientID: in.ClientID, Project: in.Project, Status: domain.ClaimDraft, CreatedBy: p.UserID,
	}
	err = r.Store.WithTx(ctx, func(tx domain.Store) error {
		prefix := "RB-" + r.now().In(Jakarta).Format("200601") + "-"
		n, err := tx.NextSequence(ctx, prefix)
		if err != nil {
			return err
		}
		c.ClaimNo = domain.SequenceNo(prefix, n)
		if err := tx.InsertClaim(ctx, c); err != nil {
			return err
		}
		return r.history(ctx, tx, p, c.ID, "CREATED", "")
	})
	if err != nil {
		return nil, err
	}
	return r.Store.GetClaim(ctx, c.ID)
}

func (r *Reimbursements) Update(ctx context.Context, p authn.Principal, id string, in ClaimInput) (*domain.Claim, error) {
	c, err := r.loadOwnDraft(ctx, p, id)
	if err != nil {
		return nil, err
	}
	if err := r.normalize(&in); err != nil {
		return nil, err
	}
	c.Category, c.TransactionDate, c.Amount, c.Description = in.Category, in.TransactionDate, in.Amount, in.Description
	c.ClientID, c.Project, c.UpdatedBy = in.ClientID, in.Project, p.UserID
	if err := r.Store.UpdateClaim(ctx, *c); err != nil {
		return nil, err
	}
	return r.Store.GetClaim(ctx, id)
}

// UploadReceipt stores an image or PDF (magic bytes checked, 10 MB max).
func (r *Reimbursements) UploadReceipt(ctx context.Context, p authn.Principal, id, filename string, data []byte) (*domain.Claim, error) {
	c, err := r.loadOwnDraft(ctx, p, id)
	if err != nil {
		return nil, err
	}
	if len(data) > domain.MaxReceiptBytes {
		return nil, apperr.Field("data", domain.MsgReceiptTooLarge)
	}
	mime, ext := domain.SniffReceipt(data)
	if mime == "" {
		return nil, apperr.Field("data", domain.MsgReceiptType)
	}
	if len(c.Receipts) >= maxReceipts {
		return nil, domain.Bad(fmt.Sprintf("Maksimal %d bukti per klaim.", maxReceipts))
	}
	rec := domain.Receipt{ID: ids.New(), ClaimID: id, Filename: domain.CleanFilename(filename), ContentType: mime, SizeBytes: int64(len(data)), CreatedBy: p.UserID}
	rec.StorageKey = "reimbursements/" + rec.ID + "." + ext
	if err := r.Storage.Put(ctx, rec.StorageKey, data); err != nil {
		return nil, err
	}
	if err := r.Store.InsertReceipt(ctx, rec); err != nil {
		_ = r.Storage.Delete(ctx, rec.StorageKey)
		return nil, err
	}
	return r.Store.GetClaim(ctx, id)
}

func (r *Reimbursements) DeleteReceipt(ctx context.Context, p authn.Principal, id, receiptID string) (*domain.Claim, error) {
	if _, err := r.loadOwnDraft(ctx, p, id); err != nil {
		return nil, err
	}
	rec, err := r.Store.GetReceipt(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	if rec == nil || rec.ClaimID != id {
		return nil, domain.NotFound("Bukti tidak ditemukan.")
	}
	if err := r.Store.DeleteReceipt(ctx, receiptID); err != nil {
		return nil, err
	}
	if err := r.Storage.Delete(ctx, rec.StorageKey); err != nil && r.Logger != nil {
		r.Logger.Warn("receipt file delete failed", "key", rec.StorageKey, "err", err)
	}
	return r.Store.GetClaim(ctx, id)
}

// ReceiptFile returns the bytes of a receipt the caller may view.
func (r *Reimbursements) ReceiptFile(ctx context.Context, p authn.Principal, receiptID string) (*domain.Receipt, []byte, string, error) {
	rec, err := r.Store.GetReceipt(ctx, receiptID)
	if err != nil {
		return nil, nil, "", err
	}
	if rec == nil {
		return nil, nil, "", domain.NotFound("Bukti tidak ditemukan.")
	}
	if _, err := r.load(ctx, p, rec.ClaimID); err != nil {
		return nil, nil, "", err
	}
	data, etag, err := r.Storage.Get(ctx, rec.StorageKey)
	if err != nil {
		return nil, nil, "", err
	}
	if data == nil {
		return nil, nil, "", domain.NotFound("File bukti tidak ditemukan.")
	}
	return rec, data, etag, nil
}

// evaluateLimit flags a claim above its category policy.
func (r *Reimbursements) evaluateLimit(ctx context.Context, c *domain.Claim) (*domain.Policy, error) {
	policy, err := r.Store.GetPolicy(ctx, c.Category)
	if err != nil {
		return nil, err
	}
	monthStart := c.TransactionDate[:8] + "01"
	t, _ := time.Parse(time.DateOnly, monthStart)
	monthEnd := t.AddDate(0, 1, -1).Format(time.DateOnly)
	mtd, err := r.Store.MonthToDate(ctx, c.EmployeeID, c.Category, monthStart, monthEnd, c.ID)
	if err != nil {
		return nil, err
	}
	c.OverLimit, c.LimitNote = domain.EvaluateLimit(policy, c.Amount, mtd)
	return policy, nil
}

// Submit checks the 30-day rule and the receipts, flags policy limits and
// routes the claim to the manager (or straight to finance without one).
func (r *Reimbursements) Submit(ctx context.Context, p authn.Principal, id string) (*domain.Claim, error) {
	c, err := r.loadOwnDraft(ctx, p, id)
	if err != nil {
		return nil, err
	}
	if len(c.Receipts) == 0 {
		return nil, domain.Bad(domain.MsgNoReceipt)
	}
	trx, _ := time.Parse(time.DateOnly, c.TransactionDate)
	today, _ := time.Parse(time.DateOnly, r.today())
	if err := domain.CheckClaimAge(trx, today); err != nil {
		return nil, err
	}
	if emp, managerUser, err := r.claimant(ctx, p.UserID); err == nil {
		c.EmployeeName, c.DepartmentName, c.ManagerEmployeeID, c.ManagerUserID = emp.Name, emp.DepartmentName, emp.ManagerID, managerUser
	} else if apperr.As(err).Code == apperr.Validation {
		return nil, err
	}
	policy, err := r.evaluateLimit(ctx, c)
	if err != nil {
		return nil, err
	}
	now := r.now()
	c.Status, c.SubmittedAt, c.UpdatedBy = domain.ClaimSubmitted, &now, p.UserID
	notes := []string{c.LimitNote}
	if c.ManagerEmployeeID == "" {
		c.Status, c.ManagerApprovedAt = domain.ClaimApprovedManager, &now
		notes = append(notes, "Tanpa atasan langsung, klaim langsung ke finance.")
		r.autoApproveFinance(c, policy, now, &notes)
	}
	err = r.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.UpdateClaim(ctx, *c); err != nil {
			return err
		}
		return r.history(ctx, tx, p, id, "SUBMITTED", joinNotes(notes))
	})
	if err != nil {
		return nil, err
	}
	r.notifyNextApprover(ctx, c)
	return r.Store.GetClaim(ctx, id)
}

// autoApproveFinance moves an APPROVED_MANAGER claim within the limits of
// an auto_approve policy to APPROVED_FINANCE.
func (r *Reimbursements) autoApproveFinance(c *domain.Claim, policy *domain.Policy, now time.Time, notes *[]string) {
	if policy == nil || !policy.IsActive || !policy.AutoApprove || c.OverLimit {
		return
	}
	c.Status, c.FinanceApprovedAt = domain.ClaimApprovedFinance, &now
	*notes = append(*notes, "Disetujui otomatis sesuai kebijakan "+policy.Label+".")
}

func joinNotes(notes []string) string {
	out := ""
	for _, n := range notes {
		if n == "" {
			continue
		}
		if out != "" {
			out += " "
		}
		out += n
	}
	return out
}

func (r *Reimbursements) notifyNextApprover(ctx context.Context, c *domain.Claim) {
	title := fmt.Sprintf("Klaim %s menunggu persetujuan", c.ClaimNo)
	body := fmt.Sprintf("%s mengajukan reimbursement %s Rp%s.", c.EmployeeName, domain.CategoryLabel(c.Category), domain.FormatRupiah(c.Amount))
	switch c.Status {
	case domain.ClaimSubmitted:
		r.notify(ctx, NotifyInput{UserIDs: []string{c.ManagerUserID}, Type: NotifClaimSubmitted, Title: title, Body: body, Href: claimHref(c.ID)})
	case domain.ClaimApprovedManager:
		r.notify(ctx, NotifyInput{UserIDs: r.financeApprovers(ctx), Type: NotifClaimSubmitted, Title: title, Body: body, Href: claimHref(c.ID)})
	}
}

func (r *Reimbursements) Cancel(ctx context.Context, p authn.Principal, id string) (*domain.Claim, error) {
	c, err := r.load(ctx, p, id)
	if err != nil {
		return nil, err
	}
	if !isOwner(p, c) {
		return nil, domain.NotFound(domain.MsgClaimNotFound)
	}
	if !domain.CanCancel(c.Status) {
		return nil, domain.Bad(domain.MsgClaimTransition)
	}
	now := r.now()
	c.Status, c.CancelledAt, c.UpdatedBy = domain.ClaimCancelled, &now, p.UserID
	err = r.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.UpdateClaim(ctx, *c); err != nil {
			return err
		}
		return r.history(ctx, tx, p, id, "CANCELLED", "")
	})
	if err != nil {
		return nil, err
	}
	return r.Store.GetClaim(ctx, id)
}

// ─── approvers ──────────────────────────────────────────────

// Inbox lists the claims waiting for the caller: their reports' SUBMITTED
// claims (all of them with hr.approve) and APPROVED_MANAGER claims for
// finance.approve holders. Own claims are left out.
func (r *Reimbursements) Inbox(ctx context.Context, p authn.Principal) ([]domain.Claim, error) {
	managerFilter := domain.ClaimFilter{Statuses: []domain.ClaimStatus{domain.ClaimSubmitted}, ManagerUserID: p.UserID}
	if p.Can(authn.PermHRApprove) {
		managerFilter.ManagerUserID = ""
	}
	waiting, _, _, err := r.Store.ListClaims(ctx, managerFilter)
	if err != nil {
		return nil, err
	}
	if p.Can(authn.PermFinanceApprove) {
		finance, _, _, err := r.Store.ListClaims(ctx, domain.ClaimFilter{Statuses: []domain.ClaimStatus{domain.ClaimApprovedManager}})
		if err != nil {
			return nil, err
		}
		waiting = append(waiting, finance...)
	}
	out := []domain.Claim{}
	for _, c := range waiting {
		if CanApprove(p, &c) {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *Reimbursements) Approve(ctx context.Context, p authn.Principal, id, note string) (*domain.Claim, error) {
	c, err := r.load(ctx, p, id)
	if err != nil {
		return nil, err
	}
	next := domain.NextApproval(c.Status)
	if next == "" {
		return nil, domain.Bad(domain.MsgClaimTransition)
	}
	if isOwner(p, c) && !p.IsAdmin() {
		return nil, domain.Bad(domain.MsgSelfApprove)
	}
	if !CanApprove(p, c) {
		if c.Status == domain.ClaimApprovedManager && c.OverLimit {
			return nil, apperr.New(apperr.Unauthorized, domain.MsgOverLimitApprove)
		}
		return nil, apperr.New(apperr.Unauthorized, domain.MsgNotApprover)
	}
	now := r.now()
	notes := []string{trim(note)}
	action := "APPROVED_FINANCE"
	if next == domain.ClaimApprovedManager {
		action = "APPROVED_MANAGER"
		c.ManagerApprovedAt, c.ManagerApprovedBy = &now, p.UserID
		policy, err := r.Store.GetPolicy(ctx, c.Category)
		if err != nil {
			return nil, err
		}
		c.Status = next
		r.autoApproveFinance(c, policy, now, &notes)
	} else {
		c.Status, c.FinanceApprovedAt, c.FinanceApprovedBy = next, &now, p.UserID
	}
	c.UpdatedBy = p.UserID
	err = r.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.UpdateClaim(ctx, *c); err != nil {
			return err
		}
		return r.history(ctx, tx, p, id, action, joinNotes(notes))
	})
	if err != nil {
		return nil, err
	}
	r.notify(ctx, NotifyInput{UserIDs: []string{c.EmployeeUserID}, Type: NotifClaimApproved,
		Title: fmt.Sprintf("Klaim %s disetujui", c.ClaimNo), Body: approvedBody(c), Href: claimHref(c.ID)})
	r.notifyNextApprover(ctx, c)
	return r.Store.GetClaim(ctx, id)
}

func approvedBody(c *domain.Claim) string {
	if c.Status == domain.ClaimApprovedFinance {
		return "Klaim disetujui finance dan menunggu pembayaran."
	}
	return "Klaim disetujui atasan dan diteruskan ke finance."
}

func (r *Reimbursements) Reject(ctx context.Context, p authn.Principal, id, reason string) (*domain.Claim, error) {
	reason = trim(reason)
	if reason == "" {
		return nil, apperr.Field("reason", domain.MsgRejectReason)
	}
	c, err := r.load(ctx, p, id)
	if err != nil {
		return nil, err
	}
	if !domain.CanReject(c.Status) {
		return nil, domain.Bad(domain.MsgClaimTransition)
	}
	if !CanApprove(p, c) {
		return nil, apperr.New(apperr.Unauthorized, domain.MsgNotApprover)
	}
	now := r.now()
	c.RejectedStage, c.Status, c.RejectedAt, c.RejectedBy, c.RejectReason, c.UpdatedBy = string(c.Status), domain.ClaimRejected, &now, p.UserID, reason, p.UserID
	err = r.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.UpdateClaim(ctx, *c); err != nil {
			return err
		}
		return r.history(ctx, tx, p, id, "REJECTED", reason)
	})
	if err != nil {
		return nil, err
	}
	r.notify(ctx, NotifyInput{UserIDs: []string{c.EmployeeUserID}, Type: NotifClaimRejected,
		Title: fmt.Sprintf("Klaim %s ditolak", c.ClaimNo), Body: "Alasan: " + reason, Href: claimHref(c.ID)})
	return r.Store.GetClaim(ctx, id)
}

// ─── finance ────────────────────────────────────────────────

func (r *Reimbursements) List(ctx context.Context, f domain.ClaimFilter, month string) ([]domain.Claim, int, domain.Money, error) {
	var err error
	if month != "" {
		if f.MonthFrom, f.MonthTo, err = r.monthRange(month); err != nil {
			return nil, 0, 0, err
		}
	} else {
		if f.MonthFrom, err = optionalDay("date_from", f.MonthFrom); err != nil {
			return nil, 0, 0, err
		}
		if f.MonthTo, err = optionalDay("date_to", f.MonthTo); err != nil {
			return nil, 0, 0, err
		}
	}
	f.Search, f.Department = trim(f.Search), trim(f.Department)
	return r.Store.ListClaims(ctx, f)
}

// PayInput pays APPROVED_FINANCE claims in one batch.
type PayInput struct {
	ClaimIDs      []string
	CashAccountID string
	PaidAt        string
	Notes         string
}

// Pay marks the claims PAID, posts one REIMBURSEMENT_PAYMENT journal
// (one expense line per category account, one cash credit) and publishes
// white.finance.reimbursement.paid.
func (r *Reimbursements) Pay(ctx context.Context, p authn.Principal, in PayInput) (*domain.ReimbursementBatch, error) {
	claimIDs := domain.UniqueIDs(in.ClaimIDs)
	if len(claimIDs) == 0 {
		return nil, apperr.Field("claim_ids", domain.MsgPayNothing)
	}
	paidAt := r.today()
	if trim(in.PaidAt) != "" {
		var err error
		if paidAt, err = parseDay("paid_at", in.PaidAt); err != nil {
			return nil, err
		}
	}
	if in.CashAccountID != "" {
		acc, err := r.Store.GetAccount(ctx, in.CashAccountID)
		if err != nil {
			return nil, err
		}
		if acc == nil || !acc.IsCashBank || !acc.IsPostable {
			return nil, domain.Bad("Akun kas/bank tidak valid")
		}
	}
	policies, err := r.Store.ListPolicies(ctx)
	if err != nil {
		return nil, err
	}
	expenseAccount := map[string]string{}
	for _, pol := range policies {
		expenseAccount[pol.Category] = pol.ExpenseAccountID
	}
	batch := domain.ReimbursementBatch{ID: ids.New(), PaidAt: paidAt, CashAccountID: in.CashAccountID, Notes: trim(in.Notes), CreatedBy: p.UserID}
	var claims []domain.Claim
	byCategory := map[string]domain.Money{}
	now := r.now()
	err = r.Store.WithTx(ctx, func(tx domain.Store) error {
		for _, id := range claimIDs {
			c, err := tx.GetClaim(ctx, id)
			if err != nil {
				return err
			}
			if c == nil {
				return domain.NotFound(domain.MsgClaimNotFound)
			}
			if c.Status != domain.ClaimApprovedFinance {
				return domain.Bad(domain.MsgPayNotApproved + " (" + c.ClaimNo + ")")
			}
			batch.TotalAmount += c.Amount
			byCategory[c.Category] += c.Amount
			claims = append(claims, *c)
		}
		prefix := domain.DailyNoPrefix("RBP", now.In(Jakarta))
		n, err := tx.NextSequence(ctx, prefix)
		if err != nil {
			return err
		}
		batch.BatchNo, batch.ClaimCount = domain.SequenceNo(prefix, n), len(claims)
		if err := tx.InsertBatch(ctx, batch); err != nil {
			return err
		}
		for i := range claims {
			c := &claims[i]
			c.Status, c.PaidAt, c.PaidBy, c.PaymentBatchID, c.UpdatedBy = domain.ClaimPaid, &now, p.UserID, batch.ID, p.UserID
			if err := tx.UpdateClaim(ctx, *c); err != nil {
				return err
			}
			if err := r.history(ctx, tx, p, c.ID, "PAID", batch.BatchNo); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var splits []domain.Split
	for _, cat := range domain.ClaimCategories {
		if amt := byCategory[cat.Code]; amt > 0 {
			splits = append(splits, domain.Split{AccountID: expenseAccount[cat.Code], Amount: amt})
		}
	}
	post := MappingPost{
		UserID: p.UserID, EventCode: domain.EventReimbursementPayment, DocumentType: DocReimbBatch, DocumentID: batch.ID,
		DocumentLabel: batch.BatchNo, EntryDate: paidAt, Description: fmt.Sprintf("Pembayaran reimbursement %s (%d klaim)", batch.BatchNo, len(claims)),
		Module: "REIMBURSEMENT", Amounts: domain.PaymentAmounts(batch.TotalAmount), Splits: map[string][]domain.Split{"EXPENSE": splits},
	}
	if in.CashAccountID != "" {
		post.Overrides = map[string]string{"CASH": in.CashAccountID}
	}
	res, err := PostFromMapping(ctx, r.Store, post)
	if err != nil {
		return nil, err
	}
	if err := r.Store.UpdateBatchJournal(ctx, batch.ID, res.EntryID, res.Status, res.Reason); err != nil {
		return nil, err
	}
	r.publishPaid(ctx, batch, claims)
	for _, c := range claims {
		r.notify(ctx, NotifyInput{UserIDs: []string{c.EmployeeUserID}, Type: NotifClaimPaid,
			Title: fmt.Sprintf("Klaim %s sudah dibayar", c.ClaimNo), Body: fmt.Sprintf("Reimbursement Rp%s dibayar pada %s.", domain.FormatRupiah(c.Amount), paidAt), Href: claimHref(c.ID)})
	}
	batch.JournalEntryID, batch.JournalStatus, batch.JournalReason, batch.CreatedAt = res.EntryID, res.Status, res.Reason, now
	if in.CashAccountID != "" {
		if acc, err := r.Store.GetAccount(ctx, in.CashAccountID); err == nil && acc != nil {
			batch.CashAccountName = acc.Name
		}
	}
	return &batch, nil
}

func (r *Reimbursements) publishPaid(ctx context.Context, b domain.ReimbursementBatch, claims []domain.Claim) {
	payload := domain.ReimbursementPaid{BatchID: b.ID, BatchNo: b.BatchNo, PaidAt: b.PaidAt, TotalAmount: b.TotalAmount}
	for _, c := range claims {
		payload.ClaimIDs = append(payload.ClaimIDs, c.ID)
		payload.EmployeeIDs = append(payload.EmployeeIDs, c.EmployeeID)
		payload.Amounts = append(payload.Amounts, c.Amount)
		payload.Claims = append(payload.Claims, domain.ReimbursementPaidRow{ClaimID: c.ID, ClaimNo: c.ClaimNo, EmployeeID: c.EmployeeID, Category: c.Category, Amount: c.Amount})
	}
	if err := r.Events.Publish(ctx, events.FinanceReimbursementPaid, payload); err != nil && r.Logger != nil {
		r.Logger.Error("publish reimbursement paid failed", "batch", b.ID, "err", err)
	}
}

func (r *Reimbursements) Batches(ctx context.Context, limit int) ([]domain.ReimbursementBatch, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return r.Store.ListBatches(ctx, limit)
}

// MonthlySummary per employee and category; mine (or no finance.read)
// limits it to the caller.
func (r *Reimbursements) MonthlySummary(ctx context.Context, p authn.Principal, month string, mine bool) (string, []domain.SummaryRow, error) {
	from, to, err := r.monthRange(month)
	if err != nil {
		return "", nil, err
	}
	employeeID := ""
	if mine || !p.Can(authn.PermFinanceRead) {
		emp, err := r.HR.EmployeeByUser(ctx, p.UserID)
		if err != nil {
			return "", nil, err
		}
		if emp == nil {
			return from[:7], []domain.SummaryRow{}, nil
		}
		employeeID = emp.ID
	}
	rows, err := r.Store.MonthlySummary(ctx, from, to, employeeID)
	return from[:7], rows, err
}

func (r *Reimbursements) Policies(ctx context.Context) ([]domain.Policy, error) {
	return r.Store.ListPolicies(ctx)
}

func (r *Reimbursements) UpdatePolicy(ctx context.Context, userID string, in domain.Policy) (*domain.Policy, error) {
	if !domain.IsClaimCategory(in.Category) {
		return nil, apperr.Field("category", "Kategori tidak valid")
	}
	if in.MonthlyLimit < 0 || in.PerClaimLimit < 0 {
		return nil, domain.Bad("Batas tidak boleh negatif")
	}
	if in.ExpenseAccountID != "" {
		if err := checkAccounts(ctx, r.Store, []string{in.ExpenseAccountID}, "", "Akun beban harus postable (bukan header)"); err != nil {
			return nil, err
		}
	}
	in.Label = domain.CategoryLabel(in.Category)
	if err := r.Store.UpsertPolicy(ctx, in, userID); err != nil {
		return nil, err
	}
	return r.Store.GetPolicy(ctx, in.Category)
}

// ─── scheduled work ─────────────────────────────────────────

// RemindApprovers nudges approvers about claims waiting more than two days
// at their step, at most once a day per claim.
func (r *Reimbursements) RemindApprovers(ctx context.Context, now time.Time) (int, error) {
	cutoff := now.Add(-domain.ReminderAfter)
	remindedBefore := now.Add(-24 * time.Hour)
	waiting, _, _, err := r.Store.ListClaims(ctx, domain.ClaimFilter{
		Statuses:        []domain.ClaimStatus{domain.ClaimSubmitted, domain.ClaimApprovedManager},
		SubmittedBefore: &cutoff, RemindedBefore: &remindedBefore,
	})
	if err != nil {
		return 0, err
	}
	var reminded []string
	for _, c := range waiting {
		var to []string
		switch {
		case c.Status == domain.ClaimSubmitted:
			to = []string{c.ManagerUserID}
		case c.ManagerApprovedAt != nil && c.ManagerApprovedAt.Before(cutoff):
			to = r.financeApprovers(ctx)
		}
		if len(domain.UniqueIDs(to)) == 0 {
			continue
		}
		r.notify(ctx, NotifyInput{UserIDs: to, Type: NotifClaimReminder,
			Title: fmt.Sprintf("Pengingat: klaim %s menunggu lebih dari 2 hari", c.ClaimNo),
			Body:  fmt.Sprintf("%s, %s Rp%s.", c.EmployeeName, domain.CategoryLabel(c.Category), domain.FormatRupiah(c.Amount)), Href: claimHref(c.ID)})
		reminded = append(reminded, c.ID)
	}
	if len(reminded) == 0 {
		return 0, nil
	}
	return len(reminded), r.Store.MarkReminded(ctx, reminded, now)
}

// ForgetUser drops a deleted user's draft claims and receipt files and
// clears manager references so their reports' claims fall back to hr.approve.
func (r *Reimbursements) ForgetUser(ctx context.Context, userID string) error {
	keys, err := r.Store.ForgetUser(ctx, userID)
	if err != nil {
		return err
	}
	for _, k := range keys {
		if err := r.Storage.Delete(ctx, k); err != nil && r.Logger != nil {
			r.Logger.Warn("receipt file delete failed", "key", k, "err", err)
		}
	}
	return nil
}
