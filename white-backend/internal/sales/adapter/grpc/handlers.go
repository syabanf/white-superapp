// Package grpc exposes the sales use cases over gRPC (grpc-gateway maps
// them to /v1/sales/... and /v1/public/sales/...). Handlers check the
// permission keys, convert proto to Go and back; no business rules here.
package grpc

import (
	"context"
	"strings"

	salesv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/sales/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/app"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Services are the use cases the handlers call.
type Services struct {
	Leads      *app.Leads
	Pipeline   *app.Pipeline
	Quotes     *app.Quotes
	Billing    *app.Billing
	Insights   *app.Insights
	Dashboard  *app.Dashboard
	Automation *app.Automation
	Forms      *app.Forms
	Contracts  *app.Contracts
	Decks      *app.Decks
	Onboarding *app.Onboarding
	Chat       *app.Chat
}

func Register(s *grpc.Server, svc Services) {
	salesv1.RegisterLeadServiceServer(s, &leadServer{svc: svc})
	salesv1.RegisterPipelineServiceServer(s, &pipelineServer{svc: svc})
	salesv1.RegisterQuotationServiceServer(s, &quotationServer{svc: svc})
	salesv1.RegisterBillingServiceServer(s, &billingServer{svc: svc})
	salesv1.RegisterInsightServiceServer(s, &insightServer{svc: svc})
	salesv1.RegisterAutomationServiceServer(s, &automationServer{svc: svc})
	salesv1.RegisterFormServiceServer(s, &formServer{svc: svc})
	salesv1.RegisterContractServiceServer(s, &contractServer{svc: svc})
	salesv1.RegisterPitchDeckServiceServer(s, &deckServer{svc: svc})
	salesv1.RegisterOnboardingServiceServer(s, &onboardingServer{svc: svc})
	salesv1.RegisterChatServiceServer(s, &chatServer{svc: svc})
}

// actor checks the permission and returns the caller.
func actor(ctx context.Context, perm string) (app.Actor, error) {
	p, err := authn.RequirePerm(ctx, perm)
	if err != nil {
		return app.Actor{}, err
	}
	return app.Actor{UserID: p.UserID, Admin: p.IsAdmin(), Approver: p.Can(authn.PermSalesApprove)}, nil
}

func reader(ctx context.Context) (app.Actor, error) { return actor(ctx, authn.PermSalesRead) }
func writer(ctx context.Context) (app.Actor, error) { return actor(ctx, authn.PermSalesWrite) }

func admin(ctx context.Context) (app.Actor, error) {
	p, err := authn.RequireAdmin(ctx)
	if err != nil {
		return app.Actor{}, err
	}
	return app.Actor{UserID: p.UserID, Admin: true, Approver: true}, nil
}

func header(ctx context.Context, key string) string {
	md, _ := metadata.FromIncomingContext(ctx)
	if v := md.Get(key); len(v) > 0 {
		return v[0]
	}
	return ""
}

func empty() *emptypb.Empty { return &emptypb.Empty{} }

func file(name, contentType string, data []byte) *salesv1.File {
	return &salesv1.File{FileName: name, ContentType: contentType, Data: data}
}

// ─── leads ───────────────────────────────────────────────────

type leadServer struct {
	salesv1.UnimplementedLeadServiceServer
	svc Services
}

func (s *leadServer) ListLeads(ctx context.Context, r *salesv1.ListLeadsRequest) (*salesv1.ListLeadsResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	from, to, bad := domain.ParseDayRange(r.DateFrom, r.DateTo)
	if bad != "" {
		return nil, apperr.Field(bad, "Tanggal tidak valid")
	}
	limit, offset := pageArgs(r.Page, r.PageSize)
	leads, total, err := s.svc.Leads.ListLeads(ctx, domain.LeadFilter{Query: strings.TrimSpace(r.Q), Status: r.Status, Temperature: r.Temperature, Source: r.Source, OwnerUserID: r.OwnerUserId,
		From: from, To: to, ScoreMin: r.ScoreMin, ScoreMax: r.ScoreMax, Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	out := &salesv1.ListLeadsResponse{Total: int32(total)}
	for _, l := range leads {
		out.Leads = append(out.Leads, leadPB(l))
	}
	return out, nil
}

func (s *leadServer) GetLead(ctx context.Context, r *salesv1.IdRequest) (*salesv1.LeadDetail, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	d, err := s.svc.Leads.GetLead(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	return &salesv1.LeadDetail{Lead: leadPB(d.Lead), Deals: dealsPB(d.Deals), Activities: activitiesPB(d.Activities), Conversations: conversationsPB(d.Conversations)}, nil
}

func (s *leadServer) CreateLead(ctx context.Context, r *salesv1.LeadInput) (*salesv1.Lead, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	l, err := s.svc.Leads.CreateLead(ctx, a, leadInput(r))
	if err != nil {
		return nil, err
	}
	return leadPB(*l), nil
}

func (s *leadServer) UpdateLead(ctx context.Context, r *salesv1.LeadInput) (*salesv1.Lead, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	l, err := s.svc.Leads.UpdateLead(ctx, a, leadInput(r))
	if err != nil {
		return nil, err
	}
	return leadPB(*l), nil
}

func (s *leadServer) DeleteLead(ctx context.Context, r *salesv1.IdRequest) (*emptypb.Empty, error) {
	if _, err := writer(ctx); err != nil {
		return nil, err
	}
	return empty(), s.svc.Leads.DeleteLead(ctx, r.Id)
}

func (s *leadServer) ImportLeads(ctx context.Context, r *salesv1.ImportLeadsRequest) (*salesv1.ImportLeadsResponse, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	res, err := s.svc.Leads.ImportLeads(ctx, a, r.FileName, r.Data)
	if err != nil {
		return nil, err
	}
	out := &salesv1.ImportLeadsResponse{Imported: int32(res.Imported), Skipped: int32(res.Skipped)}
	for _, e := range res.Errors {
		out.Errors = append(out.Errors, &salesv1.ImportRowError{Row: int32(e.Row), Message: e.Message})
	}
	return out, nil
}

func (s *leadServer) ListOwners(ctx context.Context, _ *emptypb.Empty) (*salesv1.ListOwnersResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	users, err := s.svc.Leads.ListOwners(ctx)
	if err != nil {
		return nil, err
	}
	out := &salesv1.ListOwnersResponse{}
	for _, u := range users {
		out.Owners = append(out.Owners, &salesv1.Owner{Id: u.ID, Name: u.Name, Email: u.Email})
	}
	return out, nil
}

func accountInput(r *salesv1.AccountInput) app.AccountInput {
	return app.AccountInput{ID: r.Id, Name: r.Name, AccountType: r.AccountType, Industry: r.Industry, Address: r.Address, City: r.City, Phone: r.Phone, Email: r.Email, Website: r.Website, Npwp: r.Npwp, Notes: r.Notes, OwnerUserID: r.OwnerUserId, Custom: rawOf(r.Custom)}
}

func (s *leadServer) ListAccounts(ctx context.Context, r *salesv1.ListAccountsRequest) (*salesv1.ListAccountsResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	as, err := s.svc.Leads.ListAccounts(ctx, strings.TrimSpace(r.Q), int(r.Limit))
	if err != nil {
		return nil, err
	}
	out := &salesv1.ListAccountsResponse{}
	for _, a := range as {
		out.Accounts = append(out.Accounts, accountPB(a))
	}
	return out, nil
}

func (s *leadServer) GetAccount(ctx context.Context, r *salesv1.IdRequest) (*salesv1.AccountDetail, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	d, err := s.svc.Leads.GetAccount(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	out := &salesv1.AccountDetail{Account: accountPB(d.Account), Deals: dealsPB(d.Deals)}
	for _, c := range d.Contacts {
		out.Contacts = append(out.Contacts, contactPB(c))
	}
	return out, nil
}

func (s *leadServer) CreateAccount(ctx context.Context, r *salesv1.AccountInput) (*salesv1.Account, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	acc, err := s.svc.Leads.CreateAccount(ctx, a, accountInput(r))
	if err != nil {
		return nil, err
	}
	return accountPB(*acc), nil
}

func (s *leadServer) UpdateAccount(ctx context.Context, r *salesv1.AccountInput) (*salesv1.Account, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	acc, err := s.svc.Leads.UpdateAccount(ctx, a, accountInput(r))
	if err != nil {
		return nil, err
	}
	return accountPB(*acc), nil
}

func (s *leadServer) DeleteAccount(ctx context.Context, r *salesv1.IdRequest) (*emptypb.Empty, error) {
	if _, err := writer(ctx); err != nil {
		return nil, err
	}
	return empty(), s.svc.Leads.DeleteAccount(ctx, r.Id)
}

func contactInput(r *salesv1.ContactInput) app.ContactInput {
	return app.ContactInput{ID: r.Id, AccountID: r.AccountId, Name: r.Name, Title: r.Title, Phone: r.Phone, Email: r.Email, Notes: r.Notes, IsPrimary: r.IsPrimary, Custom: rawOf(r.Custom)}
}

func (s *leadServer) ListContacts(ctx context.Context, r *salesv1.ListContactsRequest) (*salesv1.ListContactsResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	cs, err := s.svc.Leads.ListContacts(ctx, r.AccountId, strings.TrimSpace(r.Q), int(r.Limit))
	if err != nil {
		return nil, err
	}
	out := &salesv1.ListContactsResponse{}
	for _, c := range cs {
		out.Contacts = append(out.Contacts, contactPB(c))
	}
	return out, nil
}

func (s *leadServer) CreateContact(ctx context.Context, r *salesv1.ContactInput) (*salesv1.Contact, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Leads.CreateContact(ctx, a, contactInput(r))
	if err != nil {
		return nil, err
	}
	return contactPB(*c), nil
}

func (s *leadServer) UpdateContact(ctx context.Context, r *salesv1.ContactInput) (*salesv1.Contact, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Leads.UpdateContact(ctx, a, contactInput(r))
	if err != nil {
		return nil, err
	}
	return contactPB(*c), nil
}

func (s *leadServer) DeleteContact(ctx context.Context, r *salesv1.IdRequest) (*emptypb.Empty, error) {
	if _, err := writer(ctx); err != nil {
		return nil, err
	}
	return empty(), s.svc.Leads.DeleteContact(ctx, r.Id)
}

func (s *leadServer) ListCustomFields(ctx context.Context, r *salesv1.ListCustomFieldsRequest) (*salesv1.ListCustomFieldsResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	fs, err := s.svc.Leads.ListCustomFields(ctx, r.Object)
	if err != nil {
		return nil, err
	}
	out := &salesv1.ListCustomFieldsResponse{}
	for _, f := range fs {
		out.Fields = append(out.Fields, customFieldPB(f))
	}
	return out, nil
}

func (s *leadServer) SaveCustomField(ctx context.Context, r *salesv1.CustomField) (*salesv1.CustomField, error) {
	if _, err := admin(ctx); err != nil {
		return nil, err
	}
	f, err := s.svc.Leads.SaveCustomField(ctx, domain.CustomField{ID: r.Id, Object: r.Object, Key: r.Key, Label: r.Label, FieldType: r.FieldType, Options: r.Options, IsRequired: r.IsRequired, ShowInList: r.ShowInList, SortOrder: int(r.SortOrder), IsActive: r.IsActive})
	if err != nil {
		return nil, err
	}
	return customFieldPB(*f), nil
}

func (s *leadServer) DeleteCustomField(ctx context.Context, r *salesv1.IdRequest) (*emptypb.Empty, error) {
	if _, err := admin(ctx); err != nil {
		return nil, err
	}
	return empty(), s.svc.Leads.DeleteCustomField(ctx, r.Id)
}

// ─── pipeline ────────────────────────────────────────────────

type pipelineServer struct {
	salesv1.UnimplementedPipelineServiceServer
	svc Services
}

func (s *pipelineServer) ListPipelines(ctx context.Context, _ *emptypb.Empty) (*salesv1.ListPipelinesResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	ps, err := s.svc.Pipeline.ListPipelines(ctx)
	if err != nil {
		return nil, err
	}
	out := &salesv1.ListPipelinesResponse{}
	for _, p := range ps {
		out.Pipelines = append(out.Pipelines, pipelinePB(p))
	}
	return out, nil
}

func (s *pipelineServer) SavePipeline(ctx context.Context, r *salesv1.Pipeline) (*salesv1.Pipeline, error) {
	if _, err := admin(ctx); err != nil {
		return nil, err
	}
	p, err := s.svc.Pipeline.SavePipeline(ctx, domain.Pipeline{ID: r.Id, Code: r.Code, Name: r.Name, IsDefault: r.IsDefault, SortOrder: int(r.SortOrder)})
	if err != nil {
		return nil, err
	}
	return pipelinePB(*p), nil
}

func (s *pipelineServer) SaveStage(ctx context.Context, r *salesv1.Stage) (*salesv1.Stage, error) {
	if _, err := admin(ctx); err != nil {
		return nil, err
	}
	st := domain.Stage{ID: r.Id, PipelineID: r.PipelineId, Code: r.Code, Name: r.Name, SortOrder: int(r.SortOrder), Probability: int(r.Probability), IsWon: r.IsWon, IsLost: r.IsLost, IsActive: r.IsActive}
	if r.StuckThresholdDays > 0 {
		st.StuckThresholdDays = &r.StuckThresholdDays
	}
	out, err := s.svc.Pipeline.SaveStage(ctx, st)
	if err != nil {
		return nil, err
	}
	return stagePB(*out), nil
}

func (s *pipelineServer) ListLostReasons(ctx context.Context, _ *emptypb.Empty) (*salesv1.ListLostReasonsResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	rs, err := s.svc.Pipeline.ListLostReasons(ctx)
	if err != nil {
		return nil, err
	}
	out := &salesv1.ListLostReasonsResponse{}
	for _, r := range rs {
		out.Reasons = append(out.Reasons, &salesv1.LostReason{Id: r.ID, Code: r.Code, Name: r.Name})
	}
	return out, nil
}

func (s *pipelineServer) ListDeals(ctx context.Context, r *salesv1.ListDealsRequest) (*salesv1.ListDealsResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	limit, offset := pageArgs(r.Page, r.PageSize)
	if r.PageSize == 0 {
		limit = 500
	}
	from, to, bad := domain.ParseDayRange(r.DateFrom, r.DateTo)
	if bad != "" {
		return nil, apperr.Field(bad, "Tanggal tidak valid")
	}
	ds, total, err := s.svc.Pipeline.ListDeals(ctx, domain.DealFilter{PipelineID: r.PipelineId, StageID: r.StageId, OwnerUserID: r.OwnerUserId, LeadID: r.LeadId, Status: r.Status, Query: strings.TrimSpace(r.Q),
		ServiceLine: r.ServiceLine, Industry: r.Industry, Source: r.Source, LostReason: r.LostReasonId, From: from, To: to, ByClosed: r.DateField == "closed", Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	return &salesv1.ListDealsResponse{Deals: dealsPB(ds), Total: int32(total)}, nil
}

func (s *pipelineServer) GetDeal(ctx context.Context, r *salesv1.IdRequest) (*salesv1.DealDetail, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	d, err := s.svc.Pipeline.GetDeal(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	out := &salesv1.DealDetail{Deal: dealPB(d.Deal), Members: membersPB(d.Members), Quotations: quotationsPB(d.Quotations), Invoices: invoicesPB(d.Invoices), Payments: paymentsPB(d.Payments), ReferenceTotal: d.ReferenceTotal.String(), PaidTotal: d.PaidTotal.String(), PaymentStatus: d.PaymentStatus, Contracts: contractsPB(d.Contracts)}
	for _, h := range d.History {
		out.StageHistory = append(out.StageHistory, &salesv1.StageHistoryEntry{StageId: h.StageID, StageName: h.StageName, EnteredAt: ts(h.EnteredAt), MovedBy: str(h.MovedBy)})
	}
	for _, t := range d.TermProgress {
		out.TermProgress = append(out.TermProgress, &salesv1.TermProgress{TermId: t.TermID, Label: t.Label, DueDate: domain.FormatDay(t.DueDate), Percent: t.Percent, Amount: t.Amount.String(), Paid: t.Paid.String(), Status: t.Status})
	}
	if d.Onboarding != nil {
		out.Onboarding = onboardingPB(*d.Onboarding)
	}
	return out, nil
}

func (s *pipelineServer) CreateDeal(ctx context.Context, r *salesv1.DealInput) (*salesv1.Deal, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.svc.Pipeline.CreateDeal(ctx, a, dealInput(r))
	if err != nil {
		return nil, err
	}
	return dealPB(*d), nil
}

func (s *pipelineServer) UpdateDeal(ctx context.Context, r *salesv1.DealInput) (*salesv1.Deal, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.svc.Pipeline.UpdateDeal(ctx, a, dealInput(r))
	if err != nil {
		return nil, err
	}
	return dealPB(*d), nil
}

func (s *pipelineServer) DeleteDeal(ctx context.Context, r *salesv1.IdRequest) (*emptypb.Empty, error) {
	if _, err := writer(ctx); err != nil {
		return nil, err
	}
	return empty(), s.svc.Pipeline.DeleteDeal(ctx, r.Id)
}

func (s *pipelineServer) MoveDealStage(ctx context.Context, r *salesv1.MoveDealStageRequest) (*salesv1.Deal, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.svc.Pipeline.MoveDealStage(ctx, a, r.Id, r.StageId, r.ValueFinal, r.LostReasonId)
	if err != nil {
		return nil, err
	}
	return dealPB(*d), nil
}

func (s *pipelineServer) SetDealMembers(ctx context.Context, r *salesv1.SetDealMembersRequest) (*salesv1.SetDealMembersResponse, error) {
	if _, err := writer(ctx); err != nil {
		return nil, err
	}
	members := make([]domain.DealMember, 0, len(r.Members))
	for _, m := range r.Members {
		members = append(members, domain.DealMember{UserID: m.UserId, Role: m.Role, SplitPercent: m.SplitPercent})
	}
	out, err := s.svc.Pipeline.SetMembers(ctx, r.Id, members)
	if err != nil {
		return nil, err
	}
	return &salesv1.SetDealMembersResponse{Members: membersPB(out)}, nil
}

func (s *pipelineServer) ListActivities(ctx context.Context, r *salesv1.ListActivitiesRequest) (*salesv1.ListActivitiesResponse, error) {
	a, err := reader(ctx)
	if err != nil {
		return nil, err
	}
	owner := r.OwnerUserId
	if r.Mine {
		owner = a.UserID
	}
	as, err := s.svc.Pipeline.ListActivities(ctx, domain.ActivityFilter{LeadID: r.LeadId, DealID: r.DealId, OwnerUserID: owner, Status: r.Status, OpenOnly: r.OpenOnly, Limit: int(r.Limit)})
	if err != nil {
		return nil, err
	}
	return &salesv1.ListActivitiesResponse{Activities: activitiesPB(as)}, nil
}

func (s *pipelineServer) CreateActivity(ctx context.Context, r *salesv1.ActivityInput) (*salesv1.Activity, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	act, err := s.svc.Pipeline.CreateActivity(ctx, a, activityInput(r))
	if err != nil {
		return nil, err
	}
	return activityPB(*act), nil
}

func (s *pipelineServer) UpdateActivity(ctx context.Context, r *salesv1.ActivityInput) (*salesv1.Activity, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	act, err := s.svc.Pipeline.UpdateActivity(ctx, a, activityInput(r))
	if err != nil {
		return nil, err
	}
	return activityPB(*act), nil
}

func (s *pipelineServer) DeleteActivity(ctx context.Context, r *salesv1.IdRequest) (*emptypb.Empty, error) {
	if _, err := writer(ctx); err != nil {
		return nil, err
	}
	return empty(), s.svc.Pipeline.DeleteActivity(ctx, r.Id)
}

// ─── quotations ──────────────────────────────────────────────

type quotationServer struct {
	salesv1.UnimplementedQuotationServiceServer
	svc Services
}

func (s *quotationServer) ListCatalog(ctx context.Context, r *salesv1.ListCatalogRequest) (*salesv1.ListCatalogResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	items, err := s.svc.Quotes.ListCatalog(ctx, !r.IncludeInactive)
	if err != nil {
		return nil, err
	}
	out := &salesv1.ListCatalogResponse{}
	for _, c := range items {
		out.Items = append(out.Items, catalogPB(c))
	}
	return out, nil
}

func (s *quotationServer) SaveCatalogItem(ctx context.Context, r *salesv1.CatalogItem) (*salesv1.CatalogItem, error) {
	if _, err := admin(ctx); err != nil {
		return nil, err
	}
	price, err := domain.ParseMoney(r.DefaultPrice)
	if err != nil {
		return nil, invalidField("defaultPrice", "Harga tidak valid")
	}
	c := domain.CatalogItem{ID: r.Id, Code: r.Code, Name: r.Name, ServiceLine: r.ServiceLine, Unit: r.Unit, DefaultPrice: price, Description: r.Description, IsRetainer: r.IsRetainer, IsDefaultForLine: r.IsDefaultForLine, IsActive: r.IsActive, SortOrder: int(r.SortOrder)}
	for _, t := range r.Tiers {
		p, err := domain.ParseMoney(t.Price)
		if err != nil {
			return nil, invalidField("tiers", "Harga paket tidak valid")
		}
		c.Tiers = append(c.Tiers, domain.CatalogTier{Tier: t.Tier, Price: p, Description: t.Description})
	}
	out, err := s.svc.Quotes.SaveCatalogItem(ctx, c)
	if err != nil {
		return nil, err
	}
	return catalogPB(*out), nil
}

func discountsPB(rs []domain.RetainerDiscount) *salesv1.RetainerDiscounts {
	out := &salesv1.RetainerDiscounts{}
	for _, r := range rs {
		out.Rules = append(out.Rules, &salesv1.RetainerDiscount{MinMonths: int32(r.MinMonths), DiscountPercent: r.DiscountPercent})
	}
	return out
}

func (s *quotationServer) ListRetainerDiscounts(ctx context.Context, _ *emptypb.Empty) (*salesv1.RetainerDiscounts, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	rs, err := s.svc.Quotes.ListRetainerDiscounts(ctx)
	if err != nil {
		return nil, err
	}
	return discountsPB(rs), nil
}

func (s *quotationServer) SaveRetainerDiscounts(ctx context.Context, r *salesv1.RetainerDiscounts) (*salesv1.RetainerDiscounts, error) {
	if _, err := admin(ctx); err != nil {
		return nil, err
	}
	rules := make([]domain.RetainerDiscount, 0, len(r.Rules))
	for _, x := range r.Rules {
		rules = append(rules, domain.RetainerDiscount{MinMonths: int(x.MinMonths), DiscountPercent: x.DiscountPercent})
	}
	rs, err := s.svc.Quotes.SaveRetainerDiscounts(ctx, rules)
	if err != nil {
		return nil, err
	}
	return discountsPB(rs), nil
}

func (s *quotationServer) ListQuotations(ctx context.Context, r *salesv1.ListQuotationsRequest) (*salesv1.ListQuotationsResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	qs, err := s.svc.Quotes.ListQuotations(ctx, domain.QuotationFilter{DealID: r.DealId, Status: r.Status, Limit: int(r.Limit)})
	if err != nil {
		return nil, err
	}
	return &salesv1.ListQuotationsResponse{Quotations: quotationsPB(qs)}, nil
}

func (s *quotationServer) GetQuotation(ctx context.Context, r *salesv1.IdRequest) (*salesv1.Quotation, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	q, err := s.svc.Quotes.GetQuotation(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	return quotationPB(*q), nil
}

func (s *quotationServer) CreateQuotation(ctx context.Context, r *salesv1.QuotationInput) (*salesv1.Quotation, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	q, err := s.svc.Quotes.CreateQuotation(ctx, a, quotationInput(r))
	if err != nil {
		return nil, err
	}
	return quotationPB(*q), nil
}

func (s *quotationServer) UpdateQuotation(ctx context.Context, r *salesv1.QuotationInput) (*salesv1.Quotation, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	q, err := s.svc.Quotes.UpdateQuotation(ctx, a, quotationInput(r))
	if err != nil {
		return nil, err
	}
	return quotationPB(*q), nil
}

func (s *quotationServer) SetQuotationStatus(ctx context.Context, r *salesv1.SetQuotationStatusRequest) (*salesv1.Quotation, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	q, err := s.svc.Quotes.SetStatus(ctx, a, r.Id, r.Status)
	if err != nil {
		return nil, err
	}
	return quotationPB(*q), nil
}

func (s *quotationServer) ReviseQuotation(ctx context.Context, r *salesv1.IdRequest) (*salesv1.Quotation, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	q, err := s.svc.Quotes.Revise(ctx, a, r.Id)
	if err != nil {
		return nil, err
	}
	return quotationPB(*q), nil
}

func (s *quotationServer) DeleteQuotation(ctx context.Context, r *salesv1.IdRequest) (*emptypb.Empty, error) {
	if _, err := writer(ctx); err != nil {
		return nil, err
	}
	return empty(), s.svc.Quotes.DeleteQuotation(ctx, r.Id)
}

func (s *quotationServer) GenerateQuotationDraft(ctx context.Context, r *salesv1.GenerateQuotationDraftRequest) (*salesv1.GenerateQuotationDraftResponse, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	res, err := s.svc.Quotes.GenerateDraft(ctx, a, r.DealId, app.DraftOptions{Tier: r.Tier, RetainerMonths: int(r.RetainerMonths), UseAI: r.UseAi, UsePpn: r.UsePpn, StartDate: r.StartDate})
	if err != nil {
		return nil, err
	}
	return &salesv1.GenerateQuotationDraftResponse{Quotation: quotationPB(*res.Quotation), UnpricedServiceLines: res.Unpriced, Model: res.Model}, nil
}

func (s *quotationServer) RenderQuotationPdf(ctx context.Context, r *salesv1.IdRequest) (*salesv1.File, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	name, data, err := s.svc.Quotes.RenderPDF(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	return file(name, "application/pdf", data), nil
}

func (s *quotationServer) ListApprovalRules(ctx context.Context, _ *emptypb.Empty) (*salesv1.ListApprovalRulesResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	rs, err := s.svc.Quotes.ListApprovalRules(ctx)
	if err != nil {
		return nil, err
	}
	out := &salesv1.ListApprovalRulesResponse{}
	for _, r := range rs {
		out.Rules = append(out.Rules, approvalRulePB(r))
	}
	return out, nil
}

func (s *quotationServer) SaveApprovalRule(ctx context.Context, r *salesv1.ApprovalRule) (*salesv1.ApprovalRule, error) {
	if _, err := admin(ctx); err != nil {
		return nil, err
	}
	out, err := s.svc.Quotes.SaveApprovalRule(ctx, domain.ApprovalRule{ID: r.Id, Name: r.Name, Level: int(r.Level), MinDiscount: r.MinDiscountPercent, ApproverRole: &r.ApproverRole, ApproverUserID: &r.ApproverUserId, IsActive: r.IsActive})
	if err != nil {
		return nil, err
	}
	return approvalRulePB(*out), nil
}

func (s *quotationServer) DeleteApprovalRule(ctx context.Context, r *salesv1.IdRequest) (*emptypb.Empty, error) {
	if _, err := admin(ctx); err != nil {
		return nil, err
	}
	return empty(), s.svc.Quotes.DeleteApprovalRule(ctx, r.Id)
}

func (s *quotationServer) ListApprovals(ctx context.Context, r *salesv1.ListApprovalsRequest) (*salesv1.ListApprovalsResponse, error) {
	a, err := reader(ctx)
	if err != nil {
		return nil, err
	}
	vs, err := s.svc.Quotes.ListApprovals(ctx, a, r.View, r.Status)
	if err != nil {
		return nil, err
	}
	out := &salesv1.ListApprovalsResponse{}
	for _, v := range vs {
		out.Requests = append(out.Requests, approvalPB(v))
	}
	return out, nil
}

// DecideApproval needs sales.read; the step decides who may act (a named
// approver, sales.approve holders, or ADMIN).
func (s *quotationServer) DecideApproval(ctx context.Context, r *salesv1.DecideApprovalRequest) (*salesv1.DecideApprovalResponse, error) {
	a, err := reader(ctx)
	if err != nil {
		return nil, err
	}
	res, err := s.svc.Quotes.Decide(ctx, a, r.Id, r.Decision, r.Comment)
	if err != nil {
		return nil, err
	}
	return &salesv1.DecideApprovalResponse{Status: res.Status, NextLevel: int32(res.NextLevel), Message: res.Message}, nil
}

// ─── billing ─────────────────────────────────────────────────

type billingServer struct {
	salesv1.UnimplementedBillingServiceServer
	svc Services
}

// salesOrFinance lets finance read invoices without sales access.
func salesOrFinance(ctx context.Context) error {
	p, err := authn.Require(ctx)
	if err != nil {
		return err
	}
	if !p.Can(authn.PermSalesRead) && !p.Can(authn.PermFinanceRead) {
		return apperr.New(apperr.Unauthorized, "")
	}
	return nil
}

// finance is finance.write (invoice issue and cancel are finance's call).
func finance(ctx context.Context) (app.Actor, error) { return actor(ctx, authn.PermFinanceWrite) }

func (s *billingServer) ListInvoices(ctx context.Context, r *salesv1.ListInvoicesRequest) (*salesv1.ListInvoicesResponse, error) {
	if err := salesOrFinance(ctx); err != nil {
		return nil, err
	}
	from, to, bad := domain.ParseDayRange(r.DateFrom, r.DateTo)
	if bad != "" {
		return nil, apperr.Field(bad, "Tanggal tidak valid")
	}
	is, err := s.svc.Billing.ListInvoices(ctx, domain.InvoiceFilter{DealID: r.DealId, Status: r.Status, From: from, To: to, Limit: int(r.Limit)})
	if err != nil {
		return nil, err
	}
	return &salesv1.ListInvoicesResponse{Invoices: invoicesPB(is)}, nil
}

func (s *billingServer) SubmitInvoice(ctx context.Context, r *salesv1.SubmitInvoiceRequest) (*salesv1.Invoice, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	inv, err := s.svc.Billing.SubmitInvoice(ctx, a, app.InvoiceInput{DealID: r.DealId, QuotationID: r.QuotationId, TermID: r.TermId, Label: r.Label, Amount: r.Amount, DueDate: r.DueDate})
	if err != nil {
		return nil, err
	}
	return invoicePB(*inv), nil
}

func (s *billingServer) SubmitInvoicesFromTerms(ctx context.Context, r *salesv1.IdRequest) (*salesv1.ListInvoicesResponse, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	is, err := s.svc.Billing.SubmitFromTerms(ctx, a, r.Id)
	if err != nil {
		return nil, err
	}
	return &salesv1.ListInvoicesResponse{Invoices: invoicesPB(is)}, nil
}

func (s *billingServer) IssueInvoice(ctx context.Context, r *salesv1.IdRequest) (*salesv1.Invoice, error) {
	a, err := finance(ctx)
	if err != nil {
		return nil, err
	}
	inv, err := s.svc.Billing.IssueInvoice(ctx, a, r.Id)
	if err != nil {
		return nil, err
	}
	return invoicePB(*inv), nil
}

func (s *billingServer) CancelInvoice(ctx context.Context, r *salesv1.CancelInvoiceRequest) (*salesv1.Invoice, error) {
	a, err := finance(ctx)
	if err != nil {
		return nil, err
	}
	inv, err := s.svc.Billing.CancelInvoice(ctx, a, r.Id, r.Reason)
	if err != nil {
		return nil, err
	}
	return invoicePB(*inv), nil
}

func (s *billingServer) RenderInvoicePdf(ctx context.Context, r *salesv1.IdRequest) (*salesv1.File, error) {
	if err := salesOrFinance(ctx); err != nil {
		return nil, err
	}
	name, data, err := s.svc.Billing.RenderInvoicePDF(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	return file(name, "application/pdf", data), nil
}

func (s *billingServer) ListPayments(ctx context.Context, r *salesv1.ListPaymentsRequest) (*salesv1.ListPaymentsResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	ps, total, err := s.svc.Billing.ListPayments(ctx, r.DealId)
	if err != nil {
		return nil, err
	}
	return &salesv1.ListPaymentsResponse{Payments: paymentsPB(ps), PaidTotal: total.String()}, nil
}

func (s *billingServer) RecordPayment(ctx context.Context, r *salesv1.RecordPaymentRequest) (*salesv1.Payment, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	p, err := s.svc.Billing.RecordPayment(ctx, a, app.PaymentInput{DealID: r.DealId, InvoiceID: r.InvoiceId, Amount: r.Amount, Method: r.Method, PaidOn: r.PaidOn, Note: r.Note})
	if err != nil {
		return nil, err
	}
	return paymentPB(*p), nil
}

func (s *billingServer) DeletePayment(ctx context.Context, r *salesv1.IdRequest) (*emptypb.Empty, error) {
	if _, err := writer(ctx); err != nil {
		return nil, err
	}
	return empty(), s.svc.Billing.DeletePayment(ctx, r.Id)
}

// ─── insights ────────────────────────────────────────────────

type insightServer struct {
	salesv1.UnimplementedInsightServiceServer
	svc Services
}

func (s *insightServer) ListTargets(ctx context.Context, r *salesv1.ListTargetsRequest) (*salesv1.ListTargetsResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	m, ts, err := s.svc.Insights.ListTargets(ctx, r.Month)
	if err != nil {
		return nil, err
	}
	return targetsPB(m, ts), nil
}

func (s *insightServer) SaveTargets(ctx context.Context, r *salesv1.SaveTargetsRequest) (*salesv1.ListTargetsResponse, error) {
	a, err := actor(ctx, authn.PermSalesApprove)
	if err != nil {
		return nil, err
	}
	targets := make([]domain.Target, 0, len(r.Targets))
	for _, t := range r.Targets {
		v, err := domain.ParseMoney(t.TargetValue)
		if err != nil {
			return nil, invalidField("targets", "Nilai target tidak valid")
		}
		dt := domain.Target{UserID: domain.TrimmedOrNil(t.UserId), TargetValue: v}
		if t.TargetDeals > 0 {
			n := t.TargetDeals
			dt.TargetDeals = &n
		}
		targets = append(targets, dt)
	}
	m, ts, err := s.svc.Insights.SaveTargets(ctx, a, r.Month, targets)
	if err != nil {
		return nil, err
	}
	return targetsPB(m, ts), nil
}

func (s *insightServer) GetForecast(ctx context.Context, r *salesv1.GetForecastRequest) (*salesv1.Forecast, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	f, err := s.svc.Insights.Forecast(ctx, r.Month)
	if err != nil {
		return nil, err
	}
	out := &salesv1.Forecast{Month: f.Month, Total: forecastRowPB(f.Total), CompanyTargetSet: f.Total.CompanyTargetSet, AllocatedTargetValue: moneyPtr(f.Total.AllocatedTargetValue)}
	for _, row := range f.Rows {
		out.Rows = append(out.Rows, forecastRowPB(row))
	}
	return out, nil
}

func (s *insightServer) GetFunnelReport(ctx context.Context, r *salesv1.GetFunnelReportRequest) (*salesv1.FunnelReport, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	rep, from, to, err := s.svc.Insights.Funnel(ctx, r.From, r.To)
	if err != nil {
		return nil, err
	}
	out := &salesv1.FunnelReport{From: from.Format("2006-01-02"), To: to.Format("2006-01-02"), Leads: int32(rep.Leads), QualifiedLeads: int32(rep.QualifiedLeads), DealsCreated: int32(rep.DealsCreated), DealsWon: int32(rep.DealsWon), DealsLost: int32(rep.DealsLost), WonValue: rep.WonValue.String(), OpenValue: rep.OpenValue.String(), WinRatePercent: rep.WinRatePercent, AvgCycleDays: rep.AvgCycleDays, AvgDealValue: rep.AvgDealValue.String(), ServiceLineWins: map[string]int32{}}
	for _, st := range rep.Stages {
		out.Stages = append(out.Stages, &salesv1.FunnelStage{StageId: st.StageID, StageName: st.StageName, Deals: int32(st.Deals), Value: st.Value.String()})
	}
	for _, src := range rep.Sources {
		out.Sources = append(out.Sources, &salesv1.FunnelSource{Source: src.Source, Leads: int32(src.Leads), Converted: int32(src.Converted)})
	}
	for _, lr := range rep.LostReasons {
		out.LostReasons = append(out.LostReasons, &salesv1.FunnelReason{ReasonId: lr.ReasonID, Name: lr.Name, Deals: int32(lr.Deals)})
	}
	for k, v := range rep.ServiceLineWins {
		out.ServiceLineWins[k] = int32(v)
	}
	return out, nil
}

// ─── automation (ADMIN) ──────────────────────────────────────

type automationServer struct {
	salesv1.UnimplementedAutomationServiceServer
	svc Services
}

func (s *automationServer) ListWorkflowRules(ctx context.Context, _ *emptypb.Empty) (*salesv1.ListWorkflowRulesResponse, error) {
	if _, err := admin(ctx); err != nil {
		return nil, err
	}
	rs, err := s.svc.Automation.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	out := &salesv1.ListWorkflowRulesResponse{}
	for _, r := range rs {
		out.Rules = append(out.Rules, workflowRulePB(r))
	}
	return out, nil
}

func (s *automationServer) SaveWorkflowRule(ctx context.Context, r *salesv1.WorkflowRule) (*salesv1.WorkflowRule, error) {
	a, err := admin(ctx)
	if err != nil {
		return nil, err
	}
	out, err := s.svc.Automation.SaveRule(ctx, a, workflowRuleFrom(r))
	if err != nil {
		return nil, err
	}
	return workflowRulePB(*out), nil
}

func (s *automationServer) DeleteWorkflowRule(ctx context.Context, r *salesv1.IdRequest) (*emptypb.Empty, error) {
	if _, err := admin(ctx); err != nil {
		return nil, err
	}
	return empty(), s.svc.Automation.DeleteRule(ctx, r.Id)
}

func (s *automationServer) ListWorkflowRuns(ctx context.Context, r *salesv1.IdRequest) (*salesv1.ListWorkflowRunsResponse, error) {
	if _, err := admin(ctx); err != nil {
		return nil, err
	}
	runs, err := s.svc.Automation.ListRuns(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	return &salesv1.ListWorkflowRunsResponse{Runs: runsPB(runs)}, nil
}

func (s *automationServer) ListScoringRules(ctx context.Context, _ *emptypb.Empty) (*salesv1.ListScoringRulesResponse, error) {
	if _, err := admin(ctx); err != nil {
		return nil, err
	}
	rs, err := s.svc.Automation.ListScoringRules(ctx)
	if err != nil {
		return nil, err
	}
	out := &salesv1.ListScoringRulesResponse{}
	for _, r := range rs {
		out.Rules = append(out.Rules, scoringRulePB(r))
	}
	return out, nil
}

func (s *automationServer) SaveScoringRule(ctx context.Context, r *salesv1.ScoringRule) (*salesv1.ScoringRule, error) {
	if _, err := admin(ctx); err != nil {
		return nil, err
	}
	out, err := s.svc.Automation.SaveScoringRule(ctx, scoringRuleFrom(r))
	if err != nil {
		return nil, err
	}
	return scoringRulePB(out), nil
}

func (s *automationServer) DeleteScoringRule(ctx context.Context, r *salesv1.IdRequest) (*emptypb.Empty, error) {
	if _, err := admin(ctx); err != nil {
		return nil, err
	}
	return empty(), s.svc.Automation.DeleteScoringRule(ctx, r.Id)
}

func (s *automationServer) RecalculateScores(ctx context.Context, _ *emptypb.Empty) (*salesv1.RecalculateScoresResponse, error) {
	if _, err := admin(ctx); err != nil {
		return nil, err
	}
	total, changed, err := s.svc.Automation.RecalculateAll(ctx)
	if err != nil {
		return nil, err
	}
	return &salesv1.RecalculateScoresResponse{Leads: int32(total), Changed: int32(changed)}, nil
}

func remoteIP(ctx context.Context) string { return grpcx.RemoteIP(ctx) }
