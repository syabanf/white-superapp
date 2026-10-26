package postgres

import (
	"context"
	"strconv"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

// ─── catalog ─────────────────────────────────────────────────

func (s *Store) catalog(ctx context.Context, rows []sqlc.SalesServiceCatalog) ([]domain.CatalogItem, error) {
	tiers, err := s.q.ListCatalogTiers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.CatalogItem, 0, len(rows))
	for _, r := range rows {
		c := domain.CatalogItem{ID: r.ID, Code: r.Code, Name: r.Name, ServiceLine: r.ServiceLine, Unit: r.Unit, DefaultPrice: domain.MustMoney(r.DefaultPrice), Description: r.Description, IsRetainer: r.IsRetainer, IsDefaultForLine: r.IsDefaultForLine, IsActive: r.IsActive, SortOrder: int(r.SortOrder)}
		for _, t := range tiers {
			if t.CatalogID == r.ID {
				c.Tiers = append(c.Tiers, domain.CatalogTier{Tier: t.Tier, Price: domain.MustMoney(t.Price), Description: t.Description})
			}
		}
		out = append(out, c)
	}
	return out, nil
}

func (s *Store) ListCatalog(ctx context.Context, activeOnly bool) ([]domain.CatalogItem, error) {
	rows, err := s.q.ListCatalog(ctx, activeOnly)
	if err != nil {
		return nil, err
	}
	return s.catalog(ctx, rows)
}

func (s *Store) GetCatalogItem(ctx context.Context, id string) (*domain.CatalogItem, error) {
	r, err := s.q.GetCatalogItem(ctx, id)
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	items, err := s.catalog(ctx, []sqlc.SalesServiceCatalog{r})
	if err != nil {
		return nil, err
	}
	return &items[0], nil
}

func (s *Store) SaveCatalogItem(ctx context.Context, c domain.CatalogItem) error {
	if err := s.q.UpsertCatalogItem(ctx, sqlc.UpsertCatalogItemParams{ID: c.ID, Code: c.Code, Name: c.Name, ServiceLine: c.ServiceLine, Unit: c.Unit, DefaultPrice: c.DefaultPrice.String(), Description: c.Description, IsRetainer: c.IsRetainer, IsDefaultForLine: c.IsDefaultForLine, IsActive: c.IsActive, SortOrder: int32(c.SortOrder)}); err != nil {
		return wrap(err)
	}
	if err := s.q.DeleteCatalogTiers(ctx, c.ID); err != nil {
		return err
	}
	for _, t := range c.Tiers {
		if err := s.q.InsertCatalogTier(ctx, sqlc.InsertCatalogTierParams{CatalogID: c.ID, Tier: t.Tier, Price: t.Price.String(), Description: t.Description}); err != nil {
			return wrap(err)
		}
	}
	return nil
}

func (s *Store) ListRetainerDiscounts(ctx context.Context) ([]domain.RetainerDiscount, error) {
	rows, err := s.q.ListRetainerDiscounts(ctx)
	return many(rows, err, func(r sqlc.SalesRetainerDiscount) domain.RetainerDiscount {
		return domain.RetainerDiscount{MinMonths: int(r.MinMonths), DiscountPercent: domain.ParsePercent(r.DiscountPercent)}
	})
}

func (s *Store) ReplaceRetainerDiscounts(ctx context.Context, rules []domain.RetainerDiscount) error {
	if err := s.q.DeleteRetainerDiscounts(ctx); err != nil {
		return err
	}
	for _, r := range rules {
		if err := s.q.InsertRetainerDiscount(ctx, sqlc.InsertRetainerDiscountParams{MinMonths: int32(r.MinMonths), DiscountPercent: domain.PercentString(r.DiscountPercent)}); err != nil {
			return wrap(err)
		}
	}
	return nil
}

// ─── quotations ──────────────────────────────────────────────

func toQuotation(r sqlc.SalesQuotation) domain.Quotation {
	return domain.Quotation{ID: r.ID, DealID: r.DealID, QuoteNumber: r.QuoteNumber, Version: int(r.Version), ParentQuotationID: r.ParentQuotationID, Status: r.Status, UsePpn: r.UsePpn, PpnPercent: domain.ParsePercent(r.PpnPercent), Subtotal: domain.MustMoney(r.Subtotal), DiscountPercent: domain.ParsePercent(r.DiscountPercent), DiscountNominal: domain.MustMoney(r.DiscountNominal), PpnNominal: domain.MustMoney(r.PpnNominal), Total: domain.MustMoney(r.Total), Notes: r.Notes, ValidUntil: r.ValidUntil, ApprovalStatus: r.ApprovalStatus, ApprovalRequestID: r.ApprovalRequestID, GeneratedBy: r.GeneratedBy, SentAt: r.SentAt, DecidedAt: r.DecidedAt, SupersededAt: r.SupersededAt, ExpiryRemindedAt: r.ExpiryRemindedAt,
		Audit: domain.Audit{CreatedBy: r.CreatedBy, UpdatedBy: r.UpdatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}}
}

func quotationRow(r sqlc.GetQuotationRow) domain.Quotation {
	q := toQuotation(r.SalesQuotation)
	q.DealTitle, q.ClientName, q.OwnerUserID = r.DealTitle, r.ClientName, r.OwnerUserID
	return q
}

// withLines loads items and terms for the given quotations.
func (s *Store) withLines(ctx context.Context, qs []domain.Quotation) ([]domain.Quotation, error) {
	if len(qs) == 0 {
		return qs, nil
	}
	idList := make([]string, len(qs))
	index := map[string]int{}
	for i, q := range qs {
		idList[i] = q.ID
		index[q.ID] = i
	}
	items, err := s.q.ListQuotationItems(ctx, idList)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		q := &qs[index[it.QuotationID]]
		q.Items = append(q.Items, domain.QuotationItem{ID: it.ID, QuotationID: it.QuotationID, ItemType: it.ItemType, CatalogID: it.CatalogID, Description: it.Description, Scope: it.Scope, Unit: it.Unit, Qty: domain.ParsePercent(it.Qty), UnitPrice: domain.MustMoney(it.UnitPrice), LineTotal: domain.MustMoney(it.LineTotal), SortOrder: int(it.SortOrder)})
	}
	terms, err := s.q.ListQuotationTerms(ctx, idList)
	if err != nil {
		return nil, err
	}
	for _, t := range terms {
		q := &qs[index[t.QuotationID]]
		q.Terms = append(q.Terms, domain.QuotationTerm{ID: t.ID, QuotationID: t.QuotationID, Label: t.Label, Percent: domain.ParsePercent(t.Percent), DueDate: t.DueDate, SortOrder: int(t.SortOrder)})
	}
	return qs, nil
}

func (s *Store) GetQuotation(ctx context.Context, id string) (*domain.Quotation, error) {
	r, err := s.q.GetQuotation(ctx, id)
	q, err := one(r, err, quotationRow)
	if err != nil || q == nil {
		return q, err
	}
	qs, err := s.withLines(ctx, []domain.Quotation{*q})
	if err != nil {
		return nil, err
	}
	return &qs[0], nil
}

func (s *Store) ListQuotations(ctx context.Context, f domain.QuotationFilter) ([]domain.Quotation, error) {
	rows, err := s.q.ListQuotations(ctx, sqlc.ListQuotationsParams{Deal: nz(f.DealID), Status: nz(f.Status), Lim: limitOr(f.Limit, 200)})
	qs, err := many(rows, err, func(r sqlc.ListQuotationsRow) domain.Quotation { return quotationRow(sqlc.GetQuotationRow(r)) })
	if err != nil {
		return nil, err
	}
	return s.withLines(ctx, qs)
}

// SaveQuotation upserts the header; items and terms are replaced only
// while the quotation is a draft, so invoices keep their term ids.
func (s *Store) SaveQuotation(ctx context.Context, q *domain.Quotation) error {
	err := s.q.UpsertQuotation(ctx, sqlc.UpsertQuotationParams{ID: q.ID, DealID: q.DealID, QuoteNumber: q.QuoteNumber, Version: int32(q.Version), ParentQuotationID: q.ParentQuotationID, Status: q.Status, UsePpn: q.UsePpn, PpnPercent: domain.PercentString(q.PpnPercent), Subtotal: q.Subtotal.String(), DiscountPercent: domain.PercentString(q.DiscountPercent), DiscountNominal: q.DiscountNominal.String(), PpnNominal: q.PpnNominal.String(), Total: q.Total.String(), Notes: q.Notes, ValidUntil: q.ValidUntil, ApprovalStatus: q.ApprovalStatus, ApprovalRequestID: q.ApprovalRequestID, GeneratedBy: q.GeneratedBy, SentAt: q.SentAt, DecidedAt: q.DecidedAt, SupersededAt: q.SupersededAt, ExpiryRemindedAt: q.ExpiryRemindedAt, CreatedBy: q.CreatedBy, UpdatedBy: q.UpdatedBy, CreatedAt: q.CreatedAt, UpdatedAt: q.UpdatedAt})
	if err != nil || q.Status != domain.QuoteDraft {
		return wrap(err)
	}
	if err := s.q.DeleteQuotationItems(ctx, q.ID); err != nil {
		return err
	}
	for i := range q.Items {
		it := &q.Items[i]
		if it.ID == "" {
			it.ID = ids.New()
		}
		it.QuotationID, it.SortOrder = q.ID, i
		if err := s.q.InsertQuotationItem(ctx, sqlc.InsertQuotationItemParams{ID: it.ID, QuotationID: q.ID, ItemType: it.ItemType, CatalogID: it.CatalogID, Description: it.Description, Scope: it.Scope, Unit: it.Unit, Qty: strconv.FormatFloat(it.Qty, 'f', -1, 64), UnitPrice: it.UnitPrice.String(), LineTotal: it.LineTotal.String(), SortOrder: int32(i)}); err != nil {
			return wrap(err)
		}
	}
	if err := s.q.DeleteQuotationTerms(ctx, q.ID); err != nil {
		return err
	}
	for i := range q.Terms {
		t := &q.Terms[i]
		t.ID, t.QuotationID, t.SortOrder = ids.New(), q.ID, i
		if err := s.q.InsertQuotationTerm(ctx, sqlc.InsertQuotationTermParams{ID: t.ID, QuotationID: q.ID, Label: t.Label, Percent: domain.PercentString(t.Percent), DueDate: t.DueDate, SortOrder: int32(i)}); err != nil {
			return wrap(err)
		}
	}
	return nil
}

func (s *Store) SetQuotationApproval(ctx context.Context, id, status string, requestID *string) error {
	return s.q.SetQuotationApproval(ctx, sqlc.SetQuotationApprovalParams{ID: id, ApprovalStatus: status, ApprovalRequestID: requestID})
}

func (s *Store) DeleteQuotation(ctx context.Context, id string, at time.Time) error {
	return s.q.DeleteQuotation(ctx, sqlc.DeleteQuotationParams{ID: id, DeletedAt: &at})
}

func (s *Store) LatestQuotation(ctx context.Context, dealID string) (*domain.Quotation, error) {
	id, err := s.q.LatestQuotationID(ctx, dealID)
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.GetQuotation(ctx, id)
}

func (s *Store) ExpiringQuotations(ctx context.Context, until time.Time) ([]domain.Quotation, error) {
	idList, err := s.q.ExpiringQuotationIDs(ctx, &until)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Quotation, 0, len(idList))
	for _, id := range idList {
		q, err := s.GetQuotation(ctx, id)
		if err != nil {
			return nil, err
		}
		if q != nil {
			out = append(out, *q)
		}
	}
	return out, nil
}

func (s *Store) MarkQuotationReminded(ctx context.Context, id string, at time.Time) error {
	return s.q.MarkQuotationReminded(ctx, sqlc.MarkQuotationRemindedParams{ID: id, ExpiryRemindedAt: &at})
}

// ─── approvals ───────────────────────────────────────────────

func (s *Store) ListApprovalRules(ctx context.Context, activeOnly bool) ([]domain.ApprovalRule, error) {
	rows, err := s.q.ListApprovalRules(ctx, activeOnly)
	return many(rows, err, func(r sqlc.SalesApprovalRule) domain.ApprovalRule {
		return domain.ApprovalRule{ID: r.ID, Name: r.Name, Level: int(r.Level), MinDiscount: domain.ParsePercent(r.MinDiscountPercent), ApproverRole: r.ApproverRole, ApproverUserID: r.ApproverUserID, IsActive: r.IsActive}
	})
}

func (s *Store) SaveApprovalRule(ctx context.Context, r domain.ApprovalRule) error {
	return wrap(s.q.UpsertApprovalRule(ctx, sqlc.UpsertApprovalRuleParams{ID: r.ID, Name: r.Name, Level: int32(r.Level), MinDiscountPercent: domain.PercentString(r.MinDiscount), ApproverRole: r.ApproverRole, ApproverUserID: r.ApproverUserID, IsActive: r.IsActive}))
}

func (s *Store) DeleteApprovalRule(ctx context.Context, id string) error {
	return s.q.DeleteApprovalRule(ctx, id)
}

func (s *Store) CreateApprovalRequest(ctx context.Context, r domain.ApprovalRequest) error {
	if err := s.q.InsertApprovalRequest(ctx, sqlc.InsertApprovalRequestParams{ID: r.ID, Object: r.Object, SubjectID: r.SubjectID, Status: r.Status, CurrentLevel: int32(r.CurrentLevel), DiscountPercent: domain.PercentString(r.DiscountPercent), Amount: r.Amount.String(), Note: r.Note, RequestedBy: r.RequestedBy, CreatedAt: r.CreatedAt}); err != nil {
		return wrap(err)
	}
	for _, st := range r.Steps {
		if err := s.q.InsertApprovalStep(ctx, sqlc.InsertApprovalStepParams{ID: st.ID, RequestID: r.ID, Level: int32(st.Level), ApproverRole: st.ApproverRole, ApproverUserID: st.ApproverUserID, Status: st.Status}); err != nil {
			return wrap(err)
		}
	}
	return nil
}

func approvalRow(r sqlc.ListApprovalRequestsRow) domain.ApprovalRequest {
	return domain.ApprovalRequest{ID: r.ID, Object: r.Object, SubjectID: r.SubjectID, Status: r.Status, CurrentLevel: int(r.CurrentLevel), DiscountPercent: domain.ParsePercent(r.DiscountPercent), Amount: domain.MustMoney(r.Amount), Note: r.Note, RequestedBy: r.RequestedBy, ResolvedBy: r.ResolvedBy, ResolvedAt: r.ResolvedAt, CreatedAt: r.CreatedAt, QuoteNumber: r.QuoteNumber, DealID: r.DealID, DealTitle: r.DealTitle, ClientName: r.ClientName}
}

func (s *Store) withSteps(ctx context.Context, reqs []domain.ApprovalRequest) ([]domain.ApprovalRequest, error) {
	if len(reqs) == 0 {
		return reqs, nil
	}
	idList := make([]string, len(reqs))
	index := map[string]int{}
	for i, r := range reqs {
		idList[i] = r.ID
		index[r.ID] = i
	}
	steps, err := s.q.ListApprovalSteps(ctx, idList)
	if err != nil {
		return nil, err
	}
	for _, st := range steps {
		r := &reqs[index[st.RequestID]]
		r.Steps = append(r.Steps, domain.ApprovalStep{ID: st.ID, RequestID: st.RequestID, Level: int(st.Level), ApproverRole: st.ApproverRole, ApproverUserID: st.ApproverUserID, Status: st.Status, DecidedBy: st.DecidedBy, DecidedAt: st.DecidedAt, Comment: st.Comment})
	}
	return reqs, nil
}

func (s *Store) GetApprovalRequest(ctx context.Context, id string, forUpdate bool) (*domain.ApprovalRequest, error) {
	if forUpdate {
		if err := s.q.LockApprovalRequest(ctx, id); err != nil && !pg.IsNoRows(err) {
			return nil, err
		}
	}
	r, err := s.q.GetApprovalRequest(ctx, id)
	req, err := one(r, err, func(r sqlc.GetApprovalRequestRow) domain.ApprovalRequest {
		return approvalRow(sqlc.ListApprovalRequestsRow(r))
	})
	if err != nil || req == nil {
		return req, err
	}
	reqs, err := s.withSteps(ctx, []domain.ApprovalRequest{*req})
	if err != nil {
		return nil, err
	}
	return &reqs[0], nil
}

func (s *Store) ListApprovalRequests(ctx context.Context, status string, limit int) ([]domain.ApprovalRequest, error) {
	rows, err := s.q.ListApprovalRequests(ctx, sqlc.ListApprovalRequestsParams{Status: nz(status), Lim: limitOr(limit, 200)})
	reqs, err := many(rows, err, approvalRow)
	if err != nil {
		return nil, err
	}
	return s.withSteps(ctx, reqs)
}

func (s *Store) SaveApprovalRequest(ctx context.Context, r domain.ApprovalRequest) error {
	return s.q.UpdateApprovalRequest(ctx, sqlc.UpdateApprovalRequestParams{ID: r.ID, Status: r.Status, CurrentLevel: int32(r.CurrentLevel), ResolvedBy: r.ResolvedBy, ResolvedAt: r.ResolvedAt})
}

func (s *Store) SaveApprovalStep(ctx context.Context, st domain.ApprovalStep) error {
	return s.q.UpdateApprovalStep(ctx, sqlc.UpdateApprovalStepParams{ID: st.ID, Status: st.Status, DecidedBy: st.DecidedBy, DecidedAt: st.DecidedAt, Comment: st.Comment})
}

func (s *Store) CancelPendingApprovals(ctx context.Context, subjectID string, at time.Time) error {
	return s.q.CancelPendingApprovals(ctx, sqlc.CancelPendingApprovalsParams{SubjectID: subjectID, ResolvedAt: &at})
}

// ─── billing ─────────────────────────────────────────────────

func invoiceRow(r sqlc.GetInvoiceRow) domain.Invoice {
	i := r.SalesInvoice
	return domain.Invoice{ID: i.ID, DealID: i.DealID, QuotationID: i.QuotationID, TermID: i.TermID, InvoiceNumber: i.InvoiceNumber, Label: i.Label, Amount: domain.MustMoney(i.Amount), DueDate: i.DueDate, Status: i.Status, SentAt: i.SentAt, IssuedBy: i.IssuedBy, CancelledAt: i.CancelledAt, CancelledBy: i.CancelledBy, CancelReason: i.CancelReason,
		Audit:     domain.Audit{CreatedBy: i.CreatedBy, UpdatedBy: i.UpdatedBy, CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt},
		DealTitle: r.DealTitle, ClientName: r.ClientName, PaidTotal: domain.MustMoney(r.PaidTotal)}
}

func (s *Store) GetInvoice(ctx context.Context, id string) (*domain.Invoice, error) {
	r, err := s.q.GetInvoice(ctx, id)
	return one(r, err, invoiceRow)
}

func (s *Store) ListInvoices(ctx context.Context, f domain.InvoiceFilter) ([]domain.Invoice, error) {
	rows, err := s.q.ListInvoices(ctx, sqlc.ListInvoicesParams{Deal: nz(f.DealID), Status: nz(f.Status), FromAt: f.From, ToAt: f.To, Lim: limitOr(f.Limit, 200)})
	return many(rows, err, func(r sqlc.ListInvoicesRow) domain.Invoice { return invoiceRow(sqlc.GetInvoiceRow(r)) })
}

func (s *Store) SaveInvoice(ctx context.Context, i *domain.Invoice) error {
	return wrap(s.q.UpsertInvoice(ctx, sqlc.UpsertInvoiceParams{ID: i.ID, DealID: i.DealID, QuotationID: i.QuotationID, TermID: i.TermID, InvoiceNumber: i.InvoiceNumber, Label: i.Label, Amount: i.Amount.String(), DueDate: i.DueDate, Status: i.Status, SentAt: i.SentAt, IssuedBy: i.IssuedBy, CancelledAt: i.CancelledAt, CancelledBy: i.CancelledBy, CancelReason: i.CancelReason, CreatedBy: i.CreatedBy, UpdatedBy: i.UpdatedBy, CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt}))
}

func toPayment(r sqlc.SalesPayment) domain.Payment {
	return domain.Payment{ID: r.ID, DealID: r.DealID, InvoiceID: r.InvoiceID, Amount: domain.MustMoney(r.Amount), Method: r.Method, PaidOn: r.PaidOn, Note: r.Note, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt}
}

func (s *Store) GetPayment(ctx context.Context, id string) (*domain.Payment, error) {
	r, err := s.q.GetPayment(ctx, id)
	return one(r, err, toPayment)
}

func (s *Store) ListPayments(ctx context.Context, dealID string) ([]domain.Payment, error) {
	rows, err := s.q.ListPayments(ctx, dealID)
	return many(rows, err, toPayment)
}

func (s *Store) SavePayment(ctx context.Context, p *domain.Payment) error {
	return wrap(s.q.InsertPayment(ctx, sqlc.InsertPaymentParams{ID: p.ID, DealID: p.DealID, InvoiceID: p.InvoiceID, Amount: p.Amount.String(), Method: p.Method, PaidOn: p.PaidOn, Note: p.Note, CreatedBy: p.CreatedBy, CreatedAt: p.CreatedAt}))
}

func (s *Store) DeletePayment(ctx context.Context, id string, at time.Time) error {
	return s.q.DeletePayment(ctx, sqlc.DeletePaymentParams{ID: id, DeletedAt: &at})
}

func (s *Store) PaidTotal(ctx context.Context, dealID string) (domain.Money, error) {
	v, err := s.q.PaidTotal(ctx, dealID)
	return domain.MustMoney(v), err
}

func (s *Store) InvoicePaidTotal(ctx context.Context, invoiceID string) (domain.Money, error) {
	v, err := s.q.InvoicePaidTotal(ctx, &invoiceID)
	return domain.MustMoney(v), err
}

// ─── targets ─────────────────────────────────────────────────

func (s *Store) ListTargets(ctx context.Context, month string) ([]domain.Target, error) {
	rows, err := s.q.ListTargets(ctx, month)
	return many(rows, err, func(r sqlc.SalesTarget) domain.Target {
		return domain.Target{ID: r.ID, UserID: r.UserID, Month: r.PeriodMonth, TargetValue: domain.MustMoney(r.TargetValue), TargetDeals: r.TargetDeals}
	})
}

func (s *Store) ReplaceTargets(ctx context.Context, month string, targets []domain.Target, by string) error {
	if err := s.q.DeleteTargets(ctx, month); err != nil {
		return err
	}
	for _, t := range targets {
		if err := s.q.InsertTarget(ctx, sqlc.InsertTargetParams{ID: ids.New(), UserID: t.UserID, PeriodMonth: month, TargetValue: t.TargetValue.String(), TargetDeals: t.TargetDeals, UpdatedBy: &by}); err != nil {
			return wrap(err)
		}
	}
	return nil
}
