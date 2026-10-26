package grpc

import (
	"encoding/json"
	"time"

	salesv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/sales/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/app"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ts(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func tsPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func timePtr(t *timestamppb.Timestamp) *time.Time {
	if t == nil {
		return nil
	}
	v := t.AsTime().UTC()
	return &v
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func moneyPtr(m *domain.Money) string {
	if m == nil {
		return ""
	}
	return m.String()
}

func i32(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}

func structOf(raw []byte) *structpb.Struct {
	var m map[string]any
	if len(raw) == 0 || json.Unmarshal(raw, &m) != nil || m == nil {
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	s, err := structpb.NewStruct(m)
	if err != nil {
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	return s
}

func rawOf(s *structpb.Struct) json.RawMessage {
	if s == nil {
		return nil
	}
	b, err := json.Marshal(s.AsMap())
	if err != nil {
		return nil
	}
	return b
}

func valueOf(v any) *structpb.Value {
	if v == nil {
		return nil
	}
	pv, err := structpb.NewValue(v)
	if err != nil {
		return nil
	}
	return pv
}

func anyOf(v *structpb.Value) any {
	if v == nil {
		return nil
	}
	return v.AsInterface()
}

func attributionPB(a domain.Attribution) *salesv1.Attribution {
	return &salesv1.Attribution{UtmSource: str(a.UtmSource), UtmMedium: str(a.UtmMedium), UtmCampaign: str(a.UtmCampaign), UtmTerm: str(a.UtmTerm), UtmContent: str(a.UtmContent), LandingPage: str(a.LandingPage), Referrer: str(a.Referrer)}
}

func attributionFrom(a *salesv1.Attribution) domain.Attribution {
	if a == nil {
		return domain.Attribution{}
	}
	return domain.Attribution{UtmSource: domain.TrimmedOrNil(a.UtmSource), UtmMedium: domain.TrimmedOrNil(a.UtmMedium), UtmCampaign: domain.TrimmedOrNil(a.UtmCampaign), UtmTerm: domain.TrimmedOrNil(a.UtmTerm), UtmContent: domain.TrimmedOrNil(a.UtmContent), LandingPage: domain.TrimmedOrNil(a.LandingPage), Referrer: domain.TrimmedOrNil(a.Referrer)}
}

func leadPB(l domain.Lead) *salesv1.Lead {
	out := &salesv1.Lead{Id: l.ID, OrgName: l.OrgName, OrgType: l.OrgType, PicName: l.PicName, PicTitle: str(l.PicTitle), PicPhone: str(l.PicPhone), PicEmail: str(l.PicEmail), City: str(l.City), Source: l.Source, Temperature: l.Temperature, Status: l.Status, Notes: str(l.Notes), OwnerUserId: str(l.OwnerUserID), AccountId: str(l.AccountID), ContactId: str(l.ContactID), Custom: structOf(l.Custom), Score: int32(l.Score), Attribution: attributionPB(l.Attribution), CreatedAt: ts(l.CreatedAt), UpdatedAt: ts(l.UpdatedAt)}
	for _, s := range l.ScoreBreakdown {
		item := &salesv1.ScoreItem{RuleId: s.RuleID, Name: s.Name, Points: int32(s.Points)}
		if s.Count != nil {
			item.Count = int32(*s.Count)
		}
		out.ScoreBreakdown = append(out.ScoreBreakdown, item)
	}
	return out
}

func leadInput(r *salesv1.LeadInput) app.LeadInput {
	return app.LeadInput{ID: r.Id, OrgName: r.OrgName, OrgType: r.OrgType, PicName: r.PicName, PicTitle: r.PicTitle, PicPhone: r.PicPhone, PicEmail: r.PicEmail, City: r.City, Source: r.Source, Temperature: r.Temperature, Status: r.Status, Notes: r.Notes, OwnerUserID: r.OwnerUserId, AccountID: r.AccountId, ContactID: r.ContactId, Custom: rawOf(r.Custom), Attribution: attributionFrom(r.Attribution)}
}

func accountPB(a domain.Account) *salesv1.Account {
	return &salesv1.Account{Id: a.ID, Name: a.Name, AccountType: a.AccountType, Industry: str(a.Industry), Address: str(a.Address), City: str(a.City), Phone: str(a.Phone), Email: str(a.Email), Website: str(a.Website), Npwp: str(a.Npwp), Notes: str(a.Notes), OwnerUserId: str(a.OwnerUserID), Custom: structOf(a.Custom), CreatedAt: ts(a.CreatedAt), UpdatedAt: ts(a.UpdatedAt)}
}

func contactPB(c domain.Contact) *salesv1.Contact {
	return &salesv1.Contact{Id: c.ID, AccountId: str(c.AccountID), Name: c.Name, Title: str(c.Title), Phone: str(c.Phone), Email: str(c.Email), IsPrimary: c.IsPrimary, Notes: str(c.Notes), Custom: structOf(c.Custom), CreatedAt: ts(c.CreatedAt)}
}

func customFieldPB(f domain.CustomField) *salesv1.CustomField {
	return &salesv1.CustomField{Id: f.ID, Object: f.Object, Key: f.Key, Label: f.Label, FieldType: f.FieldType, Options: f.Options, IsRequired: f.IsRequired, ShowInList: f.ShowInList, SortOrder: int32(f.SortOrder), IsActive: f.IsActive}
}

func stagePB(s domain.Stage) *salesv1.Stage {
	return &salesv1.Stage{Id: s.ID, PipelineId: s.PipelineID, Code: s.Code, Name: s.Name, SortOrder: int32(s.SortOrder), Probability: int32(s.Probability), IsWon: s.IsWon, IsLost: s.IsLost, StuckThresholdDays: i32(s.StuckThresholdDays), IsActive: s.IsActive}
}

func pipelinePB(p domain.Pipeline) *salesv1.Pipeline {
	out := &salesv1.Pipeline{Id: p.ID, Code: p.Code, Name: p.Name, IsDefault: p.IsDefault, SortOrder: int32(p.SortOrder)}
	for _, s := range p.Stages {
		out.Stages = append(out.Stages, stagePB(s))
	}
	return out
}

func dealPB(d domain.Deal) *salesv1.Deal {
	return &salesv1.Deal{Id: d.ID, LeadId: str(d.LeadID), AccountId: str(d.AccountID), Title: d.Title, ServiceLines: d.ServiceLines, PipelineId: d.PipelineID, StageId: d.StageID, StageName: d.StageName, StageCode: d.StageCode, IsWon: d.IsWon, IsLost: d.IsLost, Probability: int32(d.Probability), ValueEstimate: moneyPtr(d.ValueEstimate), ValueFinal: moneyPtr(d.ValueFinal), OwnerUserId: str(d.OwnerUserID), LostReasonId: str(d.LostReasonID), ExpectedCloseDate: domain.FormatDay(d.ExpectedCloseDate), Notes: str(d.Notes), ForecastCategory: d.ForecastCategory, EnteredStageAt: ts(d.EnteredStageAt), ClosedAt: tsPtr(d.ClosedAt), Custom: structOf(d.Custom), ClientName: d.ClientName(), PicName: d.PicName, CreatedAt: ts(d.CreatedAt), UpdatedAt: ts(d.UpdatedAt)}
}

func dealsPB(ds []domain.Deal) []*salesv1.Deal {
	out := make([]*salesv1.Deal, 0, len(ds))
	for _, d := range ds {
		out = append(out, dealPB(d))
	}
	return out
}

func dealInput(r *salesv1.DealInput) app.DealInput {
	return app.DealInput{ID: r.Id, LeadID: r.LeadId, AccountID: r.AccountId, Title: r.Title, PipelineID: r.PipelineId, StageID: r.StageId, ServiceLines: r.ServiceLines, ValueEstimate: r.ValueEstimate, OwnerUserID: r.OwnerUserId, ExpectedCloseDate: r.ExpectedCloseDate, Notes: r.Notes, ForecastCategory: r.ForecastCategory, Custom: rawOf(r.Custom)}
}

func membersPB(ms []domain.DealMember) []*salesv1.DealMember {
	out := make([]*salesv1.DealMember, 0, len(ms))
	for _, m := range ms {
		out = append(out, &salesv1.DealMember{UserId: m.UserID, Role: m.Role, SplitPercent: m.SplitPercent})
	}
	return out
}

func recurrencePB(raw []byte) *salesv1.Recurrence {
	r, ok := domain.ParseRecurrence(raw)
	if !ok {
		return nil
	}
	return &salesv1.Recurrence{Freq: r.Freq, Interval: int32(r.Interval), Until: str(r.Until)}
}

func activityPB(a domain.Activity) *salesv1.Activity {
	return &salesv1.Activity{Id: a.ID, LeadId: str(a.LeadID), DealId: str(a.DealID), SubjectType: str(a.SubjectType), SubjectId: str(a.SubjectID), ActivityType: a.ActivityType, Title: a.Title, Notes: str(a.Notes), Priority: a.Priority, Status: a.Status, DueAt: tsPtr(a.DueAt), DoneAt: tsPtr(a.DoneAt), ReminderAt: tsPtr(a.ReminderAt), Recurrence: recurrencePB(a.Recurrence), OwnerUserId: str(a.OwnerUserID), ParentTaskId: str(a.ParentTaskID), CreatedAt: ts(a.CreatedAt)}
}

func activitiesPB(as []domain.Activity) []*salesv1.Activity {
	out := make([]*salesv1.Activity, 0, len(as))
	for _, a := range as {
		out = append(out, activityPB(a))
	}
	return out
}

func activityInput(r *salesv1.ActivityInput) app.ActivityInput {
	in := app.ActivityInput{ID: r.Id, LeadID: r.LeadId, DealID: r.DealId, SubjectType: r.SubjectType, SubjectID: r.SubjectId, ActivityType: r.ActivityType, Title: r.Title, Notes: r.Notes, Priority: r.Priority, Status: r.Status, DueAt: timePtr(r.DueAt), ReminderAt: timePtr(r.ReminderAt), OwnerUserID: r.OwnerUserId}
	if rc := r.Recurrence; rc != nil && rc.Freq != "" {
		in.Recurrence = &domain.Recurrence{Freq: rc.Freq, Interval: int(rc.Interval), Until: domain.TrimmedOrNil(rc.Until)}
		if in.Recurrence.Interval == 0 {
			in.Recurrence.Interval = 1
		}
	}
	return in
}

func catalogPB(c domain.CatalogItem) *salesv1.CatalogItem {
	out := &salesv1.CatalogItem{Id: c.ID, Code: c.Code, Name: c.Name, ServiceLine: c.ServiceLine, Unit: c.Unit, DefaultPrice: c.DefaultPrice.String(), Description: c.Description, IsRetainer: c.IsRetainer, IsDefaultForLine: c.IsDefaultForLine, IsActive: c.IsActive, SortOrder: int32(c.SortOrder)}
	for _, t := range c.Tiers {
		out.Tiers = append(out.Tiers, &salesv1.CatalogTier{Tier: t.Tier, Price: t.Price.String(), Description: t.Description})
	}
	return out
}

func quotationPB(q domain.Quotation) *salesv1.Quotation {
	out := &salesv1.Quotation{Id: q.ID, DealId: q.DealID, DealTitle: q.DealTitle, ClientName: q.ClientName, QuoteNumber: q.QuoteNumber, Version: int32(q.Version), ParentQuotationId: str(q.ParentQuotationID), Status: q.Status, UsePpn: q.UsePpn, PpnPercent: q.PpnPercent, Subtotal: q.Subtotal.String(), DiscountPercent: q.DiscountPercent, DiscountNominal: q.DiscountNominal.String(), PpnNominal: q.PpnNominal.String(), Total: q.Total.String(), Notes: str(q.Notes), ValidUntil: domain.FormatDay(q.ValidUntil), ApprovalStatus: q.ApprovalStatus, ApprovalRequestId: str(q.ApprovalRequestID), GeneratedBy: str(q.GeneratedBy), SentAt: tsPtr(q.SentAt), CreatedAt: ts(q.CreatedAt), UpdatedAt: ts(q.UpdatedAt)}
	for _, it := range q.Items {
		out.Items = append(out.Items, &salesv1.QuotationItem{Id: it.ID, ItemType: it.ItemType, CatalogId: str(it.CatalogID), Description: it.Description, Scope: it.Scope, Unit: it.Unit, Qty: it.Qty, UnitPrice: it.UnitPrice.String(), LineTotal: it.LineTotal.String()})
	}
	percents := make([]float64, len(q.Terms))
	for i, t := range q.Terms {
		percents[i] = t.Percent
	}
	amounts := domain.AllocateTermAmounts(q.Total, percents)
	for i, t := range q.Terms {
		out.Terms = append(out.Terms, &salesv1.QuotationTerm{Id: t.ID, Label: t.Label, Percent: t.Percent, DueDate: domain.FormatDay(t.DueDate), Amount: amounts[i].String()})
	}
	return out
}

func quotationsPB(qs []domain.Quotation) []*salesv1.Quotation {
	out := make([]*salesv1.Quotation, 0, len(qs))
	for _, q := range qs {
		out = append(out, quotationPB(q))
	}
	return out
}

func quotationInput(r *salesv1.QuotationInput) app.QuotationInput {
	in := app.QuotationInput{ID: r.Id, DealID: r.DealId, UsePpn: r.UsePpn, PpnPercent: r.PpnPercent, DiscountPercent: r.DiscountPercent, Notes: r.Notes, ValidUntil: r.ValidUntil}
	for _, it := range r.Items {
		in.Items = append(in.Items, app.QuotationItemInput{ItemType: it.ItemType, CatalogID: it.CatalogId, Description: it.Description, Scope: it.Scope, Unit: it.Unit, Qty: it.Qty, UnitPrice: it.UnitPrice})
	}
	for _, t := range r.Terms {
		in.Terms = append(in.Terms, app.QuotationTermInput{Label: t.Label, Percent: t.Percent, DueDate: t.DueDate})
	}
	return in
}

func approvalRulePB(r domain.ApprovalRule) *salesv1.ApprovalRule {
	return &salesv1.ApprovalRule{Id: r.ID, Name: r.Name, Level: int32(r.Level), MinDiscountPercent: r.MinDiscount, ApproverRole: str(r.ApproverRole), ApproverUserId: str(r.ApproverUserID), IsActive: r.IsActive}
}

func approvalPB(v app.ApprovalView) *salesv1.ApprovalRequest {
	out := &salesv1.ApprovalRequest{Id: v.ID, QuotationId: v.SubjectID, QuoteNumber: v.QuoteNumber, DealId: v.DealID, DealTitle: v.DealTitle, ClientName: v.ClientName, Status: v.Status, CurrentLevel: int32(v.CurrentLevel), DiscountPercent: v.DiscountPercent, Amount: v.Amount.String(), RequestedBy: str(v.RequestedBy), CreatedAt: ts(v.CreatedAt), ResolvedAt: tsPtr(v.ResolvedAt), CanDecide: v.CanDecide}
	for _, s := range v.Steps {
		out.Steps = append(out.Steps, &salesv1.ApprovalStep{Level: int32(s.Level), Status: s.Status, ApproverRole: str(s.ApproverRole), ApproverUserId: str(s.ApproverUserID), DecidedBy: str(s.DecidedBy), DecidedAt: tsPtr(s.DecidedAt), Comment: str(s.Comment)})
	}
	return out
}

func invoicePB(i domain.Invoice) *salesv1.Invoice {
	return &salesv1.Invoice{Id: i.ID, DealId: i.DealID, DealTitle: i.DealTitle, ClientName: i.ClientName, QuotationId: str(i.QuotationID), TermId: str(i.TermID), InvoiceNumber: i.InvoiceNumber, Label: i.Label, Amount: i.Amount.String(), DueDate: domain.FormatDay(i.DueDate), Status: i.Status, SentAt: tsPtr(i.SentAt), IssuedBy: str(i.IssuedBy), CancelReason: str(i.CancelReason), PaidTotal: i.PaidTotal.String(), CreatedAt: ts(i.CreatedAt)}
}

func invoicesPB(is []domain.Invoice) []*salesv1.Invoice {
	out := make([]*salesv1.Invoice, 0, len(is))
	for _, i := range is {
		out = append(out, invoicePB(i))
	}
	return out
}

func paymentPB(p domain.Payment) *salesv1.Payment {
	return &salesv1.Payment{Id: p.ID, DealId: p.DealID, InvoiceId: str(p.InvoiceID), Amount: p.Amount.String(), Method: p.Method, PaidOn: p.PaidOn.Format("2006-01-02"), Note: str(p.Note), CreatedBy: str(p.CreatedBy), CreatedAt: ts(p.CreatedAt)}
}

func paymentsPB(ps []domain.Payment) []*salesv1.Payment {
	out := make([]*salesv1.Payment, 0, len(ps))
	for _, p := range ps {
		out = append(out, paymentPB(p))
	}
	return out
}

func contractPB(c domain.Contract) *salesv1.Contract {
	out := &salesv1.Contract{Id: c.ID, ContractNumber: c.ContractNumber, TemplateId: str(c.TemplateID), QuotationId: str(c.QuotationID), DealId: c.DealID, Type: c.Type, Title: c.Title, ClientName: c.ClientName, Body: c.Body, Status: c.Status, StartDate: domain.FormatDay(c.StartDate), EndDate: domain.FormatDay(c.EndDate), SentAt: tsPtr(c.SentAt), SignedAt: tsPtr(c.SignedAt), SignerName: str(c.SignerName), SignerIp: str(c.SignerIP), ActivatedAt: tsPtr(c.ActivatedAt), EndedAt: tsPtr(c.EndedAt), CancelledAt: tsPtr(c.CancelledAt), CancelReason: str(c.CancelReason), CreatedAt: ts(c.CreatedAt)}
	if c.Status != domain.ContractDraft && c.Status != domain.ContractCancelled {
		out.SignPath = app.SignPath(c.SignToken)
	}
	return out
}

func contractsPB(cs []domain.Contract) []*salesv1.Contract {
	out := make([]*salesv1.Contract, 0, len(cs))
	for _, c := range cs {
		out = append(out, contractPB(c))
	}
	return out
}

func publicContractPB(c domain.Contract) *salesv1.PublicContract {
	return &salesv1.PublicContract{ContractNumber: c.ContractNumber, Title: c.Title, ClientName: c.ClientName, Body: c.Body, Status: c.Status, StartDate: domain.FormatDay(c.StartDate), EndDate: domain.FormatDay(c.EndDate), SignedAt: tsPtr(c.SignedAt), SignerName: str(c.SignerName)}
}

func templatePB(t domain.ContractTemplate) *salesv1.ContractTemplate {
	return &salesv1.ContractTemplate{Id: t.ID, Name: t.Name, Type: t.Type, Body: t.Body, IsActive: t.IsActive, UnknownPlaceholders: domain.UnknownPlaceholders(t.Body)}
}

func deckPB(d domain.PitchDeck) *salesv1.PitchDeck {
	out := &salesv1.PitchDeck{Id: d.ID, DealId: str(d.DealID), LeadId: str(d.LeadID), Title: d.Title, Version: int32(d.Version), Options: &salesv1.DeckOptions{Tone: d.Options.Tone, IncludePricing: d.Options.IncludePricing, IncludeCaseStudies: d.Options.IncludeCaseStudies, ExtraNotes: d.Options.ExtraNotes}, Model: d.Model, SizeBytes: d.SizeBytes, CreatedBy: str(d.CreatedBy), CreatedAt: ts(d.CreatedAt)}
	for _, s := range d.Slides {
		out.Slides = append(out.Slides, &salesv1.Slide{Kind: s.Kind, Title: s.Title, Subtitle: s.Subtitle, Bullets: s.Bullets})
	}
	return out
}

func caseStudyPB(c domain.CaseStudy) *salesv1.CaseStudy {
	return &salesv1.CaseStudy{Id: c.ID, Title: c.Title, ClientName: c.ClientName, Industry: c.Industry, ServiceLines: c.ServiceLines, Challenge: c.Challenge, Solution: c.Solution, Results: c.Results, SortOrder: int32(c.SortOrder), IsActive: c.IsActive}
}

func onboardingPB(o domain.Onboarding) *salesv1.Onboarding {
	out := &salesv1.Onboarding{Id: o.ID, DealId: o.DealID, ClientName: o.ClientName, Status: o.Status, PortalPath: app.PortalPath(o.PortalToken), Brief: o.Brief, BriefSubmittedAt: tsPtr(o.BriefSubmittedAt), MarketingClientId: str(o.MarketingClientID), Progress: int32(o.Progress), CreatedAt: ts(o.CreatedAt), CompletedAt: tsPtr(o.CompletedAt)}
	for _, it := range o.Items {
		out.Items = append(out.Items, &salesv1.OnboardingItem{Id: it.ID, Key: it.Key, Kind: it.Kind, Title: it.Title, Description: it.Description, OwnerUserId: str(it.OwnerUserID), DueDate: domain.FormatDay(it.DueDate), Status: it.Status, DoneAt: tsPtr(it.DoneAt), DoneBy: str(it.DoneBy)})
	}
	out.Assets = assetsPB(o.Assets)
	return out
}

func assetsPB(as []domain.OnboardingAsset) []*salesv1.OnboardingAsset {
	out := make([]*salesv1.OnboardingAsset, 0, len(as))
	for _, a := range as {
		out = append(out, &salesv1.OnboardingAsset{Id: a.ID, FileName: a.FileName, ContentType: a.ContentType, SizeBytes: a.SizeBytes, CreatedAt: ts(a.CreatedAt)})
	}
	return out
}

func formFieldsPB(fs []domain.FormField) []*salesv1.FormField {
	out := make([]*salesv1.FormField, 0, len(fs))
	for _, f := range fs {
		out = append(out, &salesv1.FormField{Key: f.Key, Label: f.Label, Type: f.Type, Required: f.Required, Options: f.Options, Help: f.Help})
	}
	return out
}

func publicOnboardingPB(o domain.Onboarding) *salesv1.PublicOnboarding {
	out := &salesv1.PublicOnboarding{ClientName: o.ClientName, Progress: int32(o.Progress), BriefFields: formFieldsPB(domain.BriefFields), Brief: o.Brief, BriefSubmittedAt: tsPtr(o.BriefSubmittedAt), Assets: assetsPB(o.Assets)}
	for _, it := range o.Items {
		out.Steps = append(out.Steps, &salesv1.PublicOnboardingStep{Title: it.Title, Status: it.Status})
	}
	return out
}

func formPB(f domain.Form) *salesv1.LeadForm {
	return &salesv1.LeadForm{Id: f.ID, Slug: f.Slug, Name: f.Name, Description: f.Description, Fields: formFieldsPB(f.Fields), SubmitLabel: f.SubmitLabel, SuccessMessage: f.SuccessMessage, RedirectUrl: str(f.RedirectURL), DefaultSource: f.DefaultSource, LeadTemperature: f.LeadTemperature, NotifyUserIds: f.NotifyUserIDs, IsActive: f.IsActive, SubmissionCount: int32(f.SubmissionCount)}
}

func conversationPB(c domain.Conversation) *salesv1.Conversation {
	return &salesv1.Conversation{Id: c.ID, Channel: c.Channel, ExternalId: c.ExternalID, LeadId: str(c.LeadID), DisplayName: str(c.DisplayName), Phone: str(c.Phone), Email: str(c.Email), Status: c.Status, Priority: c.Priority, AssignedUserId: str(c.AssignedUserID), BotActive: c.BotActive, HandoffReason: str(c.HandoffReason), HandoffAt: tsPtr(c.HandoffAt), IsComplaint: c.IsComplaint, UnreadCount: int32(c.UnreadCount), LastMessageAt: tsPtr(c.LastMessageAt), LastMessagePreview: str(c.LastMessagePreview), AwaitingSince: tsPtr(c.AwaitingSince), FirstResponseSeconds: i32(c.FirstResponseSeconds), SlaResponseBreached: c.SLAResponseBreached, ResolvedAt: tsPtr(c.ResolvedAt), CreatedAt: ts(c.CreatedAt)}
}

func conversationsPB(cs []domain.Conversation) []*salesv1.Conversation {
	out := make([]*salesv1.Conversation, 0, len(cs))
	for _, c := range cs {
		out = append(out, conversationPB(c))
	}
	return out
}

func messagesPB(ms []domain.Message) []*salesv1.ChatMessage {
	out := make([]*salesv1.ChatMessage, 0, len(ms))
	for _, m := range ms {
		out = append(out, &salesv1.ChatMessage{Id: m.ID, Direction: m.Direction, Sender: m.Sender, Body: m.Body, Status: m.Status, ErrorReason: str(m.ErrorReason), SentByUserId: str(m.SentByUserID), CreatedAt: ts(m.CreatedAt)})
	}
	return out
}

func insightPB(in *domain.Insight, status string) *salesv1.ConversationInsight {
	if in == nil {
		return &salesv1.ConversationInsight{Status: status}
	}
	return &salesv1.ConversationInsight{Summary: in.Summary, Topic: in.Topic, Sentiment: in.Sentiment, IsComplaint: in.IsComplaint, Keywords: in.Keywords, Model: in.Model, AnalyzedAt: ts(in.AnalyzedAt), Status: status}
}

func settingsPB(s domain.ChatbotSettings, waEnabled bool) *salesv1.ChatbotSettings {
	return &salesv1.ChatbotSettings{Enabled: s.Enabled, BotName: s.BotName, Persona: s.Persona, ServicesOffered: s.ServicesOffered, PricingHints: s.PricingHints, BusinessHoursStart: int32(s.BusinessHoursStart), BusinessHoursEnd: int32(s.BusinessHoursEnd), HandoffRules: s.HandoffRules, Greeting: s.Greeting, MinConfidence: s.MinConfidence, UpdatedAt: ts(s.UpdatedAt), WhatsappEnabled: waEnabled, HandoffUserIds: s.HandoffUserIDs}
}

func publicChatPB(p *app.PublicChat) *salesv1.PublicChat {
	return &salesv1.PublicChat{Token: p.Token, BotName: p.BotName, Messages: messagesPB(p.Messages), HumanTakeover: p.HumanTakeover}
}

func forecastRowPB(r *domain.ForecastRow) *salesv1.ForecastRow {
	return &salesv1.ForecastRow{UserId: str(r.UserID), UserName: r.UserName, TargetValue: r.TargetValue.String(), TargetDeals: i32(r.TargetDeals), WonValue: r.WonValue.String(), WonDeals: int32(r.WonDeals), CommitValue: r.CommitValue.String(), BestCaseValue: r.BestCaseValue.String(), PipelineValue: r.PipelineValue.String(), WeightedValue: r.WeightedValue.String(), OpenDeals: int32(r.OpenDeals), AttainmentPercent: r.AttainmentPercent, Gap: r.Gap.String()}
}

func targetsPB(month string, ts []domain.Target) *salesv1.ListTargetsResponse {
	out := &salesv1.ListTargetsResponse{Month: month}
	for _, t := range ts {
		out.Targets = append(out.Targets, &salesv1.SalesTarget{UserId: str(t.UserID), TargetValue: t.TargetValue.String(), TargetDeals: i32(t.TargetDeals)})
	}
	return out
}

func workflowRulePB(r domain.WorkflowRule) *salesv1.WorkflowRule {
	cfg, _ := structpb.NewStruct(r.TriggerConfig)
	out := &salesv1.WorkflowRule{Id: r.ID, Name: r.Name, Object: r.Object, TriggerType: r.TriggerType, TriggerConfig: cfg, RunOncePerRecord: r.RunOncePerRecord, IsActive: r.IsActive, RunCount: int32(r.RunCount), LastRunAt: tsPtr(r.LastRunAt)}
	for _, c := range r.Conditions {
		out.Conditions = append(out.Conditions, &salesv1.WorkflowCondition{Field: c.Field, Op: c.Op, Value: valueOf(c.Value)})
	}
	for _, a := range r.Actions {
		pa := &salesv1.WorkflowAction{Type: a.Type, Title: a.Title, ActivityType: a.ActivityType, Notes: a.Notes, Priority: a.Priority, AssignTo: a.AssignTo, UserId: a.UserID, Strategy: a.Strategy, UserIds: a.UserIDs, OnlyIfEmpty: a.OnlyIfEmpty, Field: a.Field, Value: valueOf(a.Value), To: a.To, Message: a.Message, Role: a.Role, Hours: a.Hours, Days: a.Days}
		if a.DueInDays != nil {
			pa.DueInDays = *a.DueInDays
		}
		out.Actions = append(out.Actions, pa)
	}
	return out
}

func workflowRuleFrom(r *salesv1.WorkflowRule) domain.WorkflowRule {
	out := domain.WorkflowRule{ID: r.Id, Name: r.Name, Object: r.Object, TriggerType: r.TriggerType, RunOncePerRecord: r.RunOncePerRecord, IsActive: r.IsActive, TriggerConfig: map[string]any{}}
	if r.TriggerConfig != nil {
		out.TriggerConfig = r.TriggerConfig.AsMap()
	}
	for _, c := range r.Conditions {
		out.Conditions = append(out.Conditions, domain.Condition{Field: c.Field, Op: c.Op, Value: anyOf(c.Value)})
	}
	for _, a := range r.Actions {
		wa := domain.WorkflowAction{Type: a.Type, Title: a.Title, ActivityType: a.ActivityType, Notes: a.Notes, Priority: a.Priority, AssignTo: a.AssignTo, UserID: a.UserId, Strategy: a.Strategy, UserIDs: a.UserIds, OnlyIfEmpty: a.OnlyIfEmpty, Field: a.Field, Value: anyOf(a.Value), To: a.To, Message: a.Message, Role: a.Role, Hours: a.Hours, Days: a.Days}
		if a.DueInDays != 0 {
			d := a.DueInDays
			wa.DueInDays = &d
		}
		out.Actions = append(out.Actions, wa)
	}
	return out
}

func scoringRulePB(r domain.ScoringRule) *salesv1.ScoringRule {
	out := &salesv1.ScoringRule{Id: r.ID, Name: r.Name, Kind: r.Kind, Field: r.Field, Operator: r.Operator, Value: valueOf(r.Value), EventType: r.EventType, MaxCount: int32(r.MaxCount), Points: int32(r.Points), IsActive: r.IsActive}
	if r.WindowDays != nil {
		out.WindowDays = int32(*r.WindowDays)
	}
	return out
}

func scoringRuleFrom(r *salesv1.ScoringRule) domain.ScoringRule {
	out := domain.ScoringRule{ID: r.Id, Name: r.Name, Kind: r.Kind, Field: r.Field, Operator: r.Operator, Value: anyOf(r.Value), EventType: r.EventType, MaxCount: int(r.MaxCount), Points: int(r.Points), IsActive: r.IsActive}
	if r.WindowDays > 0 {
		w := int(r.WindowDays)
		out.WindowDays = &w
	}
	return out
}

func runsPB(rs []domain.WorkflowRun) []*salesv1.WorkflowRun {
	out := make([]*salesv1.WorkflowRun, 0, len(rs))
	for _, r := range rs {
		list := make([]any, 0, len(r.ActionsResult))
		for _, a := range r.ActionsResult {
			list = append(list, a)
		}
		lv, _ := structpb.NewList(list)
		out = append(out, &salesv1.WorkflowRun{Id: r.ID, SubjectType: r.SubjectType, SubjectId: r.SubjectID, Status: r.Status, ActionsResult: lv, Error: str(r.Error), CreatedAt: ts(r.CreatedAt)})
	}
	return out
}

func pageArgs(page, size int32) (limit, offset int) {
	if size <= 0 || size > 200 {
		size = 50
	}
	if page < 1 {
		page = 1
	}
	return int(size), int((page - 1) * size)
}
