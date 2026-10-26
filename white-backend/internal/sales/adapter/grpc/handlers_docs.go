package grpc

import (
	"context"
	"time"

	salesv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/sales/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/app"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"google.golang.org/protobuf/types/known/emptypb"
)

func invalidField(field, msg string) error { return apperr.Field(field, msg) }

// ─── forms ───────────────────────────────────────────────────

type formServer struct {
	salesv1.UnimplementedFormServiceServer
	svc Services
}

func (s *formServer) ListForms(ctx context.Context, _ *emptypb.Empty) (*salesv1.ListFormsResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	fs, err := s.svc.Forms.ListForms(ctx)
	if err != nil {
		return nil, err
	}
	out := &salesv1.ListFormsResponse{}
	for _, f := range fs {
		out.Forms = append(out.Forms, formPB(f))
	}
	return out, nil
}

func (s *formServer) SaveForm(ctx context.Context, r *salesv1.LeadForm) (*salesv1.LeadForm, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	f := domain.Form{ID: r.Id, Slug: r.Slug, Name: r.Name, Description: r.Description, SubmitLabel: r.SubmitLabel, SuccessMessage: r.SuccessMessage, RedirectURL: domain.TrimmedOrNil(r.RedirectUrl), DefaultSource: r.DefaultSource, LeadTemperature: r.LeadTemperature, NotifyUserIDs: r.NotifyUserIds, IsActive: r.IsActive}
	for _, fl := range r.Fields {
		f.Fields = append(f.Fields, domain.FormField{Key: fl.Key, Label: fl.Label, Type: fl.Type, Required: fl.Required, Options: fl.Options, Help: fl.Help})
	}
	out, err := s.svc.Forms.SaveForm(ctx, a, f)
	if err != nil {
		return nil, err
	}
	return formPB(*out), nil
}

func (s *formServer) DeleteForm(ctx context.Context, r *salesv1.IdRequest) (*emptypb.Empty, error) {
	if _, err := writer(ctx); err != nil {
		return nil, err
	}
	return empty(), s.svc.Forms.DeleteForm(ctx, r.Id)
}

func (s *formServer) ListFormSubmissions(ctx context.Context, r *salesv1.IdRequest) (*salesv1.ListFormSubmissionsResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	subs, err := s.svc.Forms.ListSubmissions(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	out := &salesv1.ListFormSubmissionsResponse{}
	for _, sub := range subs {
		out.Submissions = append(out.Submissions, &salesv1.FormSubmission{Id: sub.ID, LeadId: str(sub.LeadID), Payload: structOf(sub.Payload), Status: sub.Status, Reason: str(sub.Reason), CreatedAt: ts(sub.CreatedAt)})
	}
	return out, nil
}

func (s *formServer) GetPublicForm(ctx context.Context, r *salesv1.PublicFormRequest) (*salesv1.PublicForm, error) {
	f, err := s.svc.Forms.PublicForm(ctx, r.Slug)
	if err != nil {
		return nil, err
	}
	return &salesv1.PublicForm{Slug: f.Slug, Name: f.Name, Description: f.Description, Fields: formFieldsPB(f.Fields), SubmitLabel: f.SubmitLabel, SuccessMessage: f.SuccessMessage, RedirectUrl: str(f.RedirectURL)}, nil
}

func (s *formServer) SubmitPublicForm(ctx context.Context, r *salesv1.SubmitPublicFormRequest) (*salesv1.SubmitPublicFormResponse, error) {
	payload := map[string]any{}
	if r.Payload != nil {
		payload = r.Payload.AsMap()
	}
	res, err := s.svc.Forms.Submit(ctx, r.Slug, remoteIP(ctx), payload)
	if err != nil {
		return nil, err
	}
	out := &salesv1.SubmitPublicFormResponse{Ok: res.OK, Message: res.Message, RedirectUrl: res.RedirectURL}
	for _, e := range res.Errors {
		out.Errors = append(out.Errors, &salesv1.SubmitPublicFormFieldError{Key: e.Key, Message: e.Message})
	}
	return out, nil
}

// ─── contracts ───────────────────────────────────────────────

type contractServer struct {
	salesv1.UnimplementedContractServiceServer
	svc Services
}

func (s *contractServer) ListContractTemplates(ctx context.Context, _ *emptypb.Empty) (*salesv1.ListContractTemplatesResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	ts, err := s.svc.Contracts.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}
	out := &salesv1.ListContractTemplatesResponse{Placeholders: domain.ContractPlaceholders}
	for _, t := range ts {
		out.Templates = append(out.Templates, templatePB(t))
	}
	return out, nil
}

func (s *contractServer) SaveContractTemplate(ctx context.Context, r *salesv1.ContractTemplate) (*salesv1.ContractTemplate, error) {
	a, err := admin(ctx)
	if err != nil {
		return nil, err
	}
	t, err := s.svc.Contracts.SaveTemplate(ctx, a, domain.ContractTemplate{ID: r.Id, Name: r.Name, Type: r.Type, Body: r.Body, IsActive: r.IsActive})
	if err != nil {
		return nil, err
	}
	return templatePB(*t), nil
}

func (s *contractServer) DeleteContractTemplate(ctx context.Context, r *salesv1.IdRequest) (*emptypb.Empty, error) {
	if _, err := admin(ctx); err != nil {
		return nil, err
	}
	return empty(), s.svc.Contracts.DeleteTemplate(ctx, r.Id)
}

func (s *contractServer) ListContracts(ctx context.Context, r *salesv1.ListContractsRequest) (*salesv1.ListContractsResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	cs, err := s.svc.Contracts.List(ctx, domain.ContractFilter{DealID: r.DealId, Status: r.Status, Limit: int(r.Limit)})
	if err != nil {
		return nil, err
	}
	return &salesv1.ListContractsResponse{Contracts: contractsPB(cs)}, nil
}

func (s *contractServer) GetContract(ctx context.Context, r *salesv1.IdRequest) (*salesv1.Contract, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	c, err := s.svc.Contracts.Get(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	return contractPB(*c), nil
}

func (s *contractServer) GenerateContract(ctx context.Context, r *salesv1.GenerateContractRequest) (*salesv1.Contract, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Contracts.Generate(ctx, a, app.GenerateInput{QuotationID: r.QuotationId, TemplateID: r.TemplateId, StartDate: r.StartDate, EndDate: r.EndDate, Title: r.Title})
	if err != nil {
		return nil, err
	}
	return contractPB(*c), nil
}

func (s *contractServer) UpdateContract(ctx context.Context, r *salesv1.UpdateContractRequest) (*salesv1.Contract, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Contracts.Update(ctx, a, app.ContractUpdate{ID: r.Id, Title: r.Title, Body: r.Body, StartDate: r.StartDate, EndDate: r.EndDate})
	if err != nil {
		return nil, err
	}
	return contractPB(*c), nil
}

func (s *contractServer) SendContract(ctx context.Context, r *salesv1.IdRequest) (*salesv1.Contract, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Contracts.Send(ctx, a, r.Id)
	if err != nil {
		return nil, err
	}
	return contractPB(*c), nil
}

func (s *contractServer) SetContractStatus(ctx context.Context, r *salesv1.SetContractStatusRequest) (*salesv1.Contract, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Contracts.SetStatus(ctx, a, r.Id, r.Status, r.Reason)
	if err != nil {
		return nil, err
	}
	return contractPB(*c), nil
}

// RenderContractPdf serves by sign token without auth, by id with
// sales.read.
func (s *contractServer) RenderContractPdf(ctx context.Context, r *salesv1.RenderContractPdfRequest) (*salesv1.File, error) {
	if r.Token == "" {
		if _, err := reader(ctx); err != nil {
			return nil, err
		}
	}
	name, data, err := s.svc.Contracts.RenderPDF(ctx, r.Id, r.Token)
	if err != nil {
		return nil, err
	}
	return file(name, "application/pdf", data), nil
}

func (s *contractServer) GetPublicContract(ctx context.Context, r *salesv1.PublicContractRequest) (*salesv1.PublicContract, error) {
	c, err := s.svc.Contracts.PublicGet(ctx, r.Token, remoteIP(ctx))
	if err != nil {
		return nil, err
	}
	return publicContractPB(*c), nil
}

func (s *contractServer) SignPublicContract(ctx context.Context, r *salesv1.SignPublicContractRequest) (*salesv1.PublicContract, error) {
	c, err := s.svc.Contracts.Sign(ctx, app.SignInput{Token: r.Token, SignerName: r.SignerName, SignaturePNG: r.SignaturePng, Agree: r.Agree, IP: remoteIP(ctx), UserAgent: header(ctx, "user-agent")})
	if err != nil {
		return nil, err
	}
	return publicContractPB(*c), nil
}

// ─── pitch decks ─────────────────────────────────────────────

type deckServer struct {
	salesv1.UnimplementedPitchDeckServiceServer
	svc Services
}

func (s *deckServer) ListPitchDecks(ctx context.Context, r *salesv1.ListPitchDecksRequest) (*salesv1.ListPitchDecksResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	ds, err := s.svc.Decks.List(ctx, r.DealId, r.LeadId)
	if err != nil {
		return nil, err
	}
	out := &salesv1.ListPitchDecksResponse{}
	for _, d := range ds {
		out.Decks = append(out.Decks, deckPB(d))
	}
	return out, nil
}

func (s *deckServer) GetPitchDeck(ctx context.Context, r *salesv1.IdRequest) (*salesv1.PitchDeck, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	d, err := s.svc.Decks.Get(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	return deckPB(*d), nil
}

func (s *deckServer) GeneratePitchDeck(ctx context.Context, r *salesv1.GeneratePitchDeckRequest) (*salesv1.PitchDeck, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	opts := domain.DeckOptions{IncludePricing: true, IncludeCaseStudies: true}
	if o := r.Options; o != nil {
		opts = domain.DeckOptions{Tone: o.Tone, IncludePricing: o.IncludePricing, IncludeCaseStudies: o.IncludeCaseStudies, ExtraNotes: o.ExtraNotes}
	}
	d, err := s.svc.Decks.Generate(ctx, a, r.DealId, r.LeadId, opts)
	if err != nil {
		return nil, err
	}
	return deckPB(*d), nil
}

func (s *deckServer) GetPitchDeckFile(ctx context.Context, r *salesv1.IdRequest) (*salesv1.File, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	name, data, err := s.svc.Decks.File(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	return file(name, "application/vnd.openxmlformats-officedocument.presentationml.presentation", data), nil
}

func (s *deckServer) ListCaseStudies(ctx context.Context, _ *emptypb.Empty) (*salesv1.ListCaseStudiesResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	cs, err := s.svc.Decks.ListCaseStudies(ctx)
	if err != nil {
		return nil, err
	}
	out := &salesv1.ListCaseStudiesResponse{}
	for _, c := range cs {
		out.CaseStudies = append(out.CaseStudies, caseStudyPB(c))
	}
	return out, nil
}

func (s *deckServer) SaveCaseStudy(ctx context.Context, r *salesv1.CaseStudy) (*salesv1.CaseStudy, error) {
	if _, err := admin(ctx); err != nil {
		return nil, err
	}
	c, err := s.svc.Decks.SaveCaseStudy(ctx, domain.CaseStudy{ID: r.Id, Title: r.Title, ClientName: r.ClientName, Industry: r.Industry, ServiceLines: r.ServiceLines, Challenge: r.Challenge, Solution: r.Solution, Results: r.Results, SortOrder: int(r.SortOrder), IsActive: r.IsActive})
	if err != nil {
		return nil, err
	}
	return caseStudyPB(*c), nil
}

func (s *deckServer) DeleteCaseStudy(ctx context.Context, r *salesv1.IdRequest) (*emptypb.Empty, error) {
	if _, err := admin(ctx); err != nil {
		return nil, err
	}
	return empty(), s.svc.Decks.DeleteCaseStudy(ctx, r.Id)
}

// ─── onboarding ──────────────────────────────────────────────

type onboardingServer struct {
	salesv1.UnimplementedOnboardingServiceServer
	svc Services
}

func (s *onboardingServer) ListOnboardings(ctx context.Context, r *salesv1.ListOnboardingsRequest) (*salesv1.ListOnboardingsResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	os, err := s.svc.Onboarding.List(ctx, r.Status)
	if err != nil {
		return nil, err
	}
	out := &salesv1.ListOnboardingsResponse{}
	for _, o := range os {
		out.Onboardings = append(out.Onboardings, onboardingPB(o))
	}
	return out, nil
}

func (s *onboardingServer) GetOnboarding(ctx context.Context, r *salesv1.IdRequest) (*salesv1.Onboarding, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	o, err := s.svc.Onboarding.Get(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	return onboardingPB(*o), nil
}

func (s *onboardingServer) StartOnboarding(ctx context.Context, r *salesv1.StartOnboardingRequest) (*salesv1.Onboarding, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	o, err := s.svc.Onboarding.Start(ctx, a, r.DealId)
	if err != nil {
		return nil, err
	}
	return onboardingPB(*o), nil
}

func (s *onboardingServer) UpdateOnboardingItem(ctx context.Context, r *salesv1.UpdateOnboardingItemRequest) (*salesv1.Onboarding, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	o, err := s.svc.Onboarding.UpdateItem(ctx, a, app.ItemUpdate{ID: r.Id, Status: r.Status, OwnerUserID: r.OwnerUserId, DueDate: r.DueDate})
	if err != nil {
		return nil, err
	}
	return onboardingPB(*o), nil
}

// CreateMarketingClient is ADMIN only: identity CreateClient is.
func (s *onboardingServer) CreateMarketingClient(ctx context.Context, r *salesv1.CreateMarketingClientRequest) (*salesv1.Onboarding, error) {
	a, err := admin(ctx)
	if err != nil {
		return nil, err
	}
	o, err := s.svc.Onboarding.CreateMarketingClient(ctx, a, r.Id, app.ClientInput{Name: r.Name, Slug: r.Slug, WebsiteURL: r.WebsiteUrl, Industry: r.Industry})
	if err != nil {
		return nil, err
	}
	return onboardingPB(*o), nil
}

func (s *onboardingServer) ListOnboardingTemplates(ctx context.Context, _ *emptypb.Empty) (*salesv1.ListOnboardingTemplatesResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	ts, err := s.svc.Onboarding.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}
	out := &salesv1.ListOnboardingTemplatesResponse{}
	for _, t := range ts {
		out.Templates = append(out.Templates, onboardingTemplatePB(t))
	}
	return out, nil
}

func onboardingTemplatePB(t domain.OnboardingTemplate) *salesv1.OnboardingTemplate {
	out := &salesv1.OnboardingTemplate{Id: t.ID, Name: t.Name, IsDefault: t.IsDefault}
	for _, it := range t.Items {
		out.Items = append(out.Items, &salesv1.OnboardingTemplateItem{Key: it.Key, Title: it.Title, Description: it.Description, DueDays: int32(it.DueDays), Kind: it.Kind})
	}
	return out
}

func (s *onboardingServer) SaveOnboardingTemplate(ctx context.Context, r *salesv1.OnboardingTemplate) (*salesv1.OnboardingTemplate, error) {
	a, err := admin(ctx)
	if err != nil {
		return nil, err
	}
	t := domain.OnboardingTemplate{ID: r.Id, Name: r.Name, IsDefault: r.IsDefault}
	for _, it := range r.Items {
		t.Items = append(t.Items, domain.OnboardingTemplateItem{Key: it.Key, Title: it.Title, Description: it.Description, DueDays: int(it.DueDays), Kind: it.Kind})
	}
	out, err := s.svc.Onboarding.SaveTemplate(ctx, a, t)
	if err != nil {
		return nil, err
	}
	return onboardingTemplatePB(*out), nil
}

func (s *onboardingServer) GetOnboardingAsset(ctx context.Context, r *salesv1.IdRequest) (*salesv1.File, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	a, data, err := s.svc.Onboarding.Asset(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	return file(a.FileName, a.ContentType, data), nil
}

func (s *onboardingServer) GetPublicOnboarding(ctx context.Context, r *salesv1.PublicOnboardingRequest) (*salesv1.PublicOnboarding, error) {
	o, err := s.svc.Onboarding.PublicGet(ctx, r.Token, remoteIP(ctx))
	if err != nil {
		return nil, err
	}
	return publicOnboardingPB(*o), nil
}

func (s *onboardingServer) SubmitPublicBrief(ctx context.Context, r *salesv1.SubmitPublicBriefRequest) (*salesv1.PublicOnboarding, error) {
	o, err := s.svc.Onboarding.SubmitBrief(ctx, r.Token, remoteIP(ctx), r.Brief)
	if err != nil {
		return nil, err
	}
	return publicOnboardingPB(*o), nil
}

func (s *onboardingServer) UploadPublicAsset(ctx context.Context, r *salesv1.UploadPublicAssetRequest) (*salesv1.PublicOnboarding, error) {
	o, err := s.svc.Onboarding.UploadAsset(ctx, r.Token, remoteIP(ctx), r.FileName, r.Data)
	if err != nil {
		return nil, err
	}
	return publicOnboardingPB(*o), nil
}

// ─── chat ────────────────────────────────────────────────────

type chatServer struct {
	salesv1.UnimplementedChatServiceServer
	svc Services
}

func (s *chatServer) ListConversations(ctx context.Context, r *salesv1.ListConversationsRequest) (*salesv1.ListConversationsResponse, error) {
	a, err := reader(ctx)
	if err != nil {
		return nil, err
	}
	f := domain.ConversationFilter{Status: r.Status, Channel: r.Channel, Query: r.Q, Limit: int(r.Limit)}
	if r.Mine {
		f.AssignedUserID = a.UserID
	}
	cs, err := s.svc.Chat.List(ctx, f)
	if err != nil {
		return nil, err
	}
	return &salesv1.ListConversationsResponse{Conversations: conversationsPB(cs)}, nil
}

func detailPB(d *app.ConversationDetail) *salesv1.ConversationDetail {
	out := &salesv1.ConversationDetail{Conversation: conversationPB(d.Conversation), Messages: messagesPB(d.Messages)}
	for _, n := range d.Notes {
		out.Notes = append(out.Notes, &salesv1.ConversationNote{Id: n.ID, AuthorUserId: n.AuthorUserID, Body: n.Body, CreatedAt: ts(n.CreatedAt)})
	}
	if d.Insight != nil {
		out.Insight = insightPB(d.Insight, "cache")
	}
	return out
}

func (s *chatServer) GetConversation(ctx context.Context, r *salesv1.IdRequest) (*salesv1.ConversationDetail, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	d, err := s.svc.Chat.Get(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	return detailPB(d), nil
}

func (s *chatServer) ReplyConversation(ctx context.Context, r *salesv1.ReplyConversationRequest) (*salesv1.ConversationDetail, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.svc.Chat.Reply(ctx, a, r.Id, r.Body)
	if err != nil {
		return nil, err
	}
	return detailPB(d), nil
}

func (s *chatServer) ConversationAction(ctx context.Context, r *salesv1.ConversationActionRequest) (*salesv1.Conversation, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Chat.Action(ctx, a, r.Id, r.Action, r.Status, r.Value)
	if err != nil {
		return nil, err
	}
	return conversationPB(*c), nil
}

func (s *chatServer) AddConversationNote(ctx context.Context, r *salesv1.AddConversationNoteRequest) (*salesv1.ConversationNote, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	n, err := s.svc.Chat.AddNote(ctx, a, r.Id, r.Body)
	if err != nil {
		return nil, err
	}
	return &salesv1.ConversationNote{Id: n.ID, AuthorUserId: n.AuthorUserID, Body: n.Body, CreatedAt: ts(n.CreatedAt)}, nil
}

func (s *chatServer) AnalyzeConversation(ctx context.Context, r *salesv1.IdRequest) (*salesv1.ConversationInsight, error) {
	if _, err := writer(ctx); err != nil {
		return nil, err
	}
	in, status, err := s.svc.Chat.Analyze(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	return insightPB(in, status), nil
}

func (s *chatServer) ListReplyTemplates(ctx context.Context, _ *emptypb.Empty) (*salesv1.ListReplyTemplatesResponse, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	ts, err := s.svc.Chat.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}
	out := &salesv1.ListReplyTemplatesResponse{}
	for _, t := range ts {
		out.Templates = append(out.Templates, &salesv1.ReplyTemplate{Id: t.ID, Title: t.Title, Body: t.Body, IsActive: t.IsActive})
	}
	return out, nil
}

func (s *chatServer) SaveReplyTemplate(ctx context.Context, r *salesv1.ReplyTemplate) (*salesv1.ReplyTemplate, error) {
	a, err := writer(ctx)
	if err != nil {
		return nil, err
	}
	t, err := s.svc.Chat.SaveTemplate(ctx, a, domain.ReplyTemplate{ID: r.Id, Title: r.Title, Body: r.Body, IsActive: r.IsActive})
	if err != nil {
		return nil, err
	}
	return &salesv1.ReplyTemplate{Id: t.ID, Title: t.Title, Body: t.Body, IsActive: t.IsActive}, nil
}

func (s *chatServer) DeleteReplyTemplate(ctx context.Context, r *salesv1.IdRequest) (*emptypb.Empty, error) {
	if _, err := writer(ctx); err != nil {
		return nil, err
	}
	return empty(), s.svc.Chat.DeleteTemplate(ctx, r.Id)
}

func (s *chatServer) GetChatbotSettings(ctx context.Context, _ *emptypb.Empty) (*salesv1.ChatbotSettings, error) {
	if _, err := reader(ctx); err != nil {
		return nil, err
	}
	st, err := s.svc.Chat.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	return settingsPB(st, s.svc.Chat.WhatsAppEnabled()), nil
}

func (s *chatServer) SaveChatbotSettings(ctx context.Context, r *salesv1.ChatbotSettings) (*salesv1.ChatbotSettings, error) {
	a, err := admin(ctx)
	if err != nil {
		return nil, err
	}
	st, err := s.svc.Chat.SaveSettings(ctx, a, domain.ChatbotSettings{Enabled: r.Enabled, BotName: r.BotName, Persona: r.Persona, ServicesOffered: r.ServicesOffered, PricingHints: r.PricingHints, BusinessHoursStart: int(r.BusinessHoursStart), BusinessHoursEnd: int(r.BusinessHoursEnd), HandoffRules: r.HandoffRules, Greeting: r.Greeting, MinConfidence: r.MinConfidence, HandoffUserIDs: r.HandoffUserIds})
	if err != nil {
		return nil, err
	}
	return settingsPB(st, s.svc.Chat.WhatsAppEnabled()), nil
}

func (s *chatServer) StartPublicChat(ctx context.Context, r *salesv1.StartPublicChatRequest) (*salesv1.PublicChat, error) {
	p, err := s.svc.Chat.StartPublic(ctx, remoteIP(ctx), app.StartInput{Name: r.Name, Email: r.Email, Phone: r.Phone, PageURL: r.PageUrl, Attribution: attributionFrom(r.Attribution)})
	if err != nil {
		return nil, err
	}
	return publicChatPB(p), nil
}

func (s *chatServer) SendPublicChat(ctx context.Context, r *salesv1.SendPublicChatRequest) (*salesv1.PublicChat, error) {
	p, err := s.svc.Chat.SendPublic(ctx, remoteIP(ctx), r.Token, r.Body)
	if err != nil {
		return nil, err
	}
	return publicChatPB(p), nil
}

func (s *chatServer) PollPublicChat(ctx context.Context, r *salesv1.PollPublicChatRequest) (*salesv1.PublicChat, error) {
	var after time.Time
	if r.After != nil {
		after = r.After.AsTime()
	}
	p, err := s.svc.Chat.PollPublic(ctx, remoteIP(ctx), r.Token, after)
	if err != nil {
		return nil, err
	}
	return publicChatPB(p), nil
}

// ReceiveWhatsApp takes the gateway token from x-gateway-token (forwarded
// as metadata) or the body field.
func (s *chatServer) ReceiveWhatsApp(ctx context.Context, r *salesv1.ReceiveWhatsAppRequest) (*salesv1.ReceiveWhatsAppResponse, error) {
	token := header(ctx, "x-gateway-token")
	if token == "" {
		token = r.GatewayToken
	}
	msgs := make([]app.WAInbound, 0, len(r.Messages))
	for _, m := range r.Messages {
		msgs = append(msgs, app.WAInbound{RemoteJid: m.RemoteJid, MessageID: m.MessageId, Text: m.Text, MediaType: m.MediaType, PushName: m.PushName, FromMe: m.FromMe, Timestamp: m.Timestamp})
	}
	stored, skipped, err := s.svc.Chat.ReceiveWhatsApp(ctx, remoteIP(ctx), token, msgs)
	if err != nil {
		return nil, err
	}
	return &salesv1.ReceiveWhatsAppResponse{Success: true, Stored: int32(stored), Skipped: int32(skipped)}, nil
}
