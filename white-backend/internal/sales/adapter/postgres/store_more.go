package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

// ─── automation ──────────────────────────────────────────────

func (s *Store) InsertCrmEvent(ctx context.Context, e domain.CrmEvent) error {
	payload := e.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	return s.q.InsertCrmEvent(ctx, sqlc.InsertCrmEventParams{ID: e.ID, EventType: e.EventType, SubjectType: e.SubjectType, SubjectID: e.SubjectID, Payload: mustJSON(payload), ActorUserID: e.ActorUserID, CreatedAt: e.CreatedAt})
}

// CountSubjectEvents counts events per type and per "type:sub_type".
func (s *Store) CountSubjectEvents(ctx context.Context, subjectIDs []string, since *time.Time) (map[string]int, error) {
	rows, err := s.q.CountSubjectEvents(ctx, sqlc.CountSubjectEventsParams{Ids: subjectIDs, Since: since})
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, r := range rows {
		out[r.EventType] += int(r.N)
		if r.SubType != "" {
			out[r.EventType+":"+r.SubType] += int(r.N)
		}
	}
	return out, nil
}

func toWorkflowRule(r sqlc.SalesWorkflowRule) domain.WorkflowRule {
	w := domain.WorkflowRule{ID: r.ID, Name: r.Name, Object: r.Object, TriggerType: r.TriggerType, RunOncePerRecord: r.RunOncePerRecord, IsActive: r.IsActive, RunCount: int(r.RunCount), LastRunAt: r.LastRunAt, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
	_ = json.Unmarshal(r.TriggerConfig, &w.TriggerConfig)
	_ = json.Unmarshal(r.Conditions, &w.Conditions)
	_ = json.Unmarshal(r.Actions, &w.Actions)
	if w.TriggerConfig == nil {
		w.TriggerConfig = map[string]any{}
	}
	return w
}

func (s *Store) ListWorkflowRules(ctx context.Context, object, trigger string, activeOnly bool) ([]domain.WorkflowRule, error) {
	rows, err := s.q.ListWorkflowRules(ctx, sqlc.ListWorkflowRulesParams{Object: nz(object), Trigger: nz(trigger), ActiveOnly: activeOnly})
	return many(rows, err, toWorkflowRule)
}

func (s *Store) GetWorkflowRule(ctx context.Context, id string) (*domain.WorkflowRule, error) {
	r, err := s.q.GetWorkflowRule(ctx, id)
	return one(r, err, toWorkflowRule)
}

func (s *Store) SaveWorkflowRule(ctx context.Context, r domain.WorkflowRule) error {
	conds, actions := r.Conditions, r.Actions
	if conds == nil {
		conds = []domain.Condition{}
	}
	if actions == nil {
		actions = []domain.WorkflowAction{}
	}
	cfg := r.TriggerConfig
	if cfg == nil {
		cfg = map[string]any{}
	}
	return wrap(s.q.UpsertWorkflowRule(ctx, sqlc.UpsertWorkflowRuleParams{ID: r.ID, Name: r.Name, Object: r.Object, TriggerType: r.TriggerType, TriggerConfig: mustJSON(cfg), Conditions: mustJSON(conds), Actions: mustJSON(actions), RunOncePerRecord: r.RunOncePerRecord, IsActive: r.IsActive, CreatedBy: r.CreatedBy}))
}

func (s *Store) DeleteWorkflowRule(ctx context.Context, id string) error {
	return s.q.DeleteWorkflowRule(ctx, id)
}

func (s *Store) MarkRuleRun(ctx context.Context, id string, at time.Time) error {
	return s.q.MarkRuleRun(ctx, sqlc.MarkRuleRunParams{ID: id, LastRunAt: &at})
}

func (s *Store) HasRuleRun(ctx context.Context, ruleID, subjectID string, since *time.Time) (bool, error) {
	return s.q.HasRuleRun(ctx, sqlc.HasRuleRunParams{RuleID: ruleID, SubjectID: subjectID, Since: since})
}

func (s *Store) CountRuleRuns(ctx context.Context, ruleID string) (int, error) {
	n, err := s.q.CountRuleRuns(ctx, ruleID)
	return int(n), err
}

func (s *Store) InsertWorkflowRun(ctx context.Context, r domain.WorkflowRun) error {
	return s.q.InsertWorkflowRun(ctx, sqlc.InsertWorkflowRunParams{ID: r.ID, RuleID: r.RuleID, EventID: r.EventID, SubjectType: r.SubjectType, SubjectID: r.SubjectID, Status: r.Status, ActionsResult: mustJSON(nonNil(r.ActionsResult)), Error: r.Error, CreatedAt: r.CreatedAt})
}

func nonNil(v []map[string]any) []map[string]any {
	if v == nil {
		return []map[string]any{}
	}
	return v
}

func (s *Store) UpdateWorkflowRun(ctx context.Context, r domain.WorkflowRun) error {
	return s.q.UpdateWorkflowRun(ctx, sqlc.UpdateWorkflowRunParams{ID: r.ID, Status: r.Status, ActionsResult: mustJSON(nonNil(r.ActionsResult)), Error: r.Error})
}

func (s *Store) ListWorkflowRuns(ctx context.Context, ruleID string, limit int) ([]domain.WorkflowRun, error) {
	rows, err := s.q.ListWorkflowRuns(ctx, sqlc.ListWorkflowRunsParams{RuleID: ruleID, Limit: limitOr(limit, 100)})
	return many(rows, err, func(r sqlc.SalesWorkflowRun) domain.WorkflowRun {
		w := domain.WorkflowRun{ID: r.ID, RuleID: r.RuleID, EventID: r.EventID, SubjectType: r.SubjectType, SubjectID: r.SubjectID, Status: r.Status, Error: r.Error, CreatedAt: r.CreatedAt}
		_ = json.Unmarshal(r.ActionsResult, &w.ActionsResult)
		return w
	})
}

func (s *Store) InsertScheduledAction(ctx context.Context, a domain.ScheduledAction) error {
	return s.q.InsertScheduledAction(ctx, sqlc.InsertScheduledActionParams{ID: a.ID, RuleID: a.RuleID, RunID: a.RunID, SubjectType: a.SubjectType, SubjectID: a.SubjectID, Actions: mustJSON(a.Actions), Context: mustJSON(a.Context), RunAt: a.RunAt})
}

func (s *Store) DueScheduledActions(ctx context.Context, now time.Time, limit int) ([]domain.ScheduledAction, error) {
	rows, err := s.q.DueScheduledActions(ctx, sqlc.DueScheduledActionsParams{RunAt: now, Limit: limitOr(limit, 50)})
	return many(rows, err, func(r sqlc.SalesScheduledAction) domain.ScheduledAction {
		a := domain.ScheduledAction{ID: r.ID, RuleID: r.RuleID, RunID: r.RunID, SubjectType: r.SubjectType, SubjectID: r.SubjectID, RunAt: r.RunAt, Status: r.Status, Attempts: int(r.Attempts)}
		_ = json.Unmarshal(r.Actions, &a.Actions)
		_ = json.Unmarshal(r.Context, &a.Context)
		return a
	})
}

func (s *Store) FinishScheduledAction(ctx context.Context, id, status string, errText *string) error {
	return s.q.FinishScheduledAction(ctx, sqlc.FinishScheduledActionParams{ID: id, Status: status, LastError: errText})
}

func (s *Store) ListScoringRules(ctx context.Context, activeOnly bool) ([]domain.ScoringRule, error) {
	rows, err := s.q.ListScoringRules(ctx, activeOnly)
	return many(rows, err, func(r sqlc.SalesScoringRule) domain.ScoringRule {
		sr := domain.ScoringRule{ID: r.ID, Name: r.Name, Kind: r.Kind, MaxCount: int(r.MaxCount), Points: int(r.Points), IsActive: r.IsActive}
		if r.Field != nil {
			sr.Field = *r.Field
		}
		if r.Operator != nil {
			sr.Operator = *r.Operator
		}
		if r.EventType != nil {
			sr.EventType = *r.EventType
		}
		if r.WindowDays != nil {
			w := int(*r.WindowDays)
			sr.WindowDays = &w
		}
		if len(r.Value) > 0 {
			_ = json.Unmarshal(r.Value, &sr.Value)
		}
		return sr
	})
}

func (s *Store) SaveScoringRule(ctx context.Context, r domain.ScoringRule) error {
	var window *int32
	if r.WindowDays != nil {
		w := int32(*r.WindowDays)
		window = &w
	}
	var value []byte
	if r.Value != nil {
		value = mustJSON(r.Value)
	}
	return wrap(s.q.UpsertScoringRule(ctx, sqlc.UpsertScoringRuleParams{ID: r.ID, Name: r.Name, Kind: r.Kind, Field: nz(r.Field), Operator: nz(r.Operator), Value: value, EventType: nz(r.EventType), WindowDays: window, MaxCount: int32(max(1, r.MaxCount)), Points: int32(r.Points), IsActive: r.IsActive}))
}

func (s *Store) DeleteScoringRule(ctx context.Context, id string) error {
	return s.q.DeleteScoringRule(ctx, id)
}

// ─── forms ───────────────────────────────────────────────────

func toForm(r sqlc.SalesForm) domain.Form {
	f := domain.Form{ID: r.ID, Slug: r.Slug, Name: r.Name, Description: r.Description, SubmitLabel: r.SubmitLabel, SuccessMessage: r.SuccessMessage, RedirectURL: r.RedirectUrl, DefaultSource: r.DefaultSource, LeadTemperature: r.LeadTemperature, NotifyUserIDs: r.NotifyUserIds, IsActive: r.IsActive, SubmissionCount: int(r.SubmissionCount),
		Audit: domain.Audit{CreatedBy: r.CreatedBy, UpdatedBy: r.UpdatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}}
	_ = json.Unmarshal(r.Fields, &f.Fields)
	return f
}

func (s *Store) GetForm(ctx context.Context, id string) (*domain.Form, error) {
	r, err := s.q.GetForm(ctx, id)
	return one(r, err, toForm)
}

func (s *Store) GetFormBySlug(ctx context.Context, slug string) (*domain.Form, error) {
	r, err := s.q.GetFormBySlug(ctx, slug)
	return one(r, err, toForm)
}

func (s *Store) ListForms(ctx context.Context) ([]domain.Form, error) {
	rows, err := s.q.ListForms(ctx)
	return many(rows, err, toForm)
}

func (s *Store) SaveForm(ctx context.Context, f *domain.Form) error {
	notify := f.NotifyUserIDs
	if notify == nil {
		notify = []string{}
	}
	return wrap(s.q.UpsertForm(ctx, sqlc.UpsertFormParams{ID: f.ID, Slug: f.Slug, Name: f.Name, Description: f.Description, Fields: mustJSON(f.Fields), SubmitLabel: f.SubmitLabel, SuccessMessage: f.SuccessMessage, RedirectUrl: f.RedirectURL, DefaultSource: f.DefaultSource, LeadTemperature: f.LeadTemperature, NotifyUserIds: notify, IsActive: f.IsActive, CreatedBy: f.CreatedBy, UpdatedBy: f.UpdatedBy, CreatedAt: f.CreatedAt, UpdatedAt: f.UpdatedAt}))
}

func (s *Store) DeleteForm(ctx context.Context, id string) error { return s.q.DeleteForm(ctx, id) }

func (s *Store) InsertFormSubmission(ctx context.Context, sub domain.FormSubmission) error {
	if err := s.q.InsertFormSubmission(ctx, sqlc.InsertFormSubmissionParams{ID: sub.ID, FormID: sub.FormID, LeadID: sub.LeadID, Payload: jsonOr(sub.Payload, "{}"), Utm: jsonOr(sub.Utm, "{}"), IpHash: sub.IPHash, Status: sub.Status, Reason: sub.Reason, CreatedAt: sub.CreatedAt}); err != nil {
		return err
	}
	if sub.Status == "accepted" {
		return s.q.IncrementFormSubmissions(ctx, sub.FormID)
	}
	return nil
}

func (s *Store) ListFormSubmissions(ctx context.Context, formID string, limit int) ([]domain.FormSubmission, error) {
	rows, err := s.q.ListFormSubmissions(ctx, sqlc.ListFormSubmissionsParams{FormID: formID, Limit: limitOr(limit, 200)})
	return many(rows, err, func(r sqlc.SalesFormSubmission) domain.FormSubmission {
		return domain.FormSubmission{ID: r.ID, FormID: r.FormID, LeadID: r.LeadID, Payload: r.Payload, Utm: r.Utm, IPHash: r.IpHash, Status: r.Status, Reason: r.Reason, CreatedAt: r.CreatedAt}
	})
}

// ─── contracts ───────────────────────────────────────────────

func toTemplate(r sqlc.SalesContractTemplate) domain.ContractTemplate {
	return domain.ContractTemplate{ID: r.ID, Name: r.Name, Type: r.Type, Body: r.Body, IsActive: r.IsActive, Audit: domain.Audit{CreatedBy: r.CreatedBy, UpdatedBy: r.UpdatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}}
}

func (s *Store) ListContractTemplates(ctx context.Context, activeOnly bool) ([]domain.ContractTemplate, error) {
	rows, err := s.q.ListContractTemplates(ctx, activeOnly)
	return many(rows, err, toTemplate)
}

func (s *Store) GetContractTemplate(ctx context.Context, id string) (*domain.ContractTemplate, error) {
	r, err := s.q.GetContractTemplate(ctx, id)
	return one(r, err, toTemplate)
}

func (s *Store) SaveContractTemplate(ctx context.Context, t *domain.ContractTemplate) error {
	return wrap(s.q.UpsertContractTemplate(ctx, sqlc.UpsertContractTemplateParams{ID: t.ID, Name: t.Name, Type: t.Type, Body: t.Body, IsActive: t.IsActive, CreatedBy: t.CreatedBy, UpdatedBy: t.UpdatedBy, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}))
}

func (s *Store) DeleteContractTemplate(ctx context.Context, id string) error {
	return s.q.DeleteContractTemplate(ctx, id)
}

func toContract(r sqlc.SalesContract) domain.Contract {
	return domain.Contract{ID: r.ID, ContractNumber: r.ContractNumber, TemplateID: r.TemplateID, QuotationID: r.QuotationID, DealID: r.DealID, Type: r.Type, Title: r.Title, ClientName: r.ClientName, Body: r.Body, Status: r.Status, StartDate: r.StartDate, EndDate: r.EndDate, SignToken: r.SignToken, SentAt: r.SentAt, SignedAt: r.SignedAt, SignerName: r.SignerName, SignerIP: r.SignerIp, SignerUserAgent: r.SignerUserAgent, SignaturePNG: r.SignaturePng, ActivatedAt: r.ActivatedAt, EndedAt: r.EndedAt, CancelledAt: r.CancelledAt, CancelReason: r.CancelReason,
		Audit: domain.Audit{CreatedBy: r.CreatedBy, UpdatedBy: r.UpdatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}}
}

func (s *Store) GetContract(ctx context.Context, id string) (*domain.Contract, error) {
	r, err := s.q.GetContract(ctx, id)
	return one(r, err, toContract)
}

func (s *Store) GetContractByToken(ctx context.Context, token string) (*domain.Contract, error) {
	r, err := s.q.GetContractByToken(ctx, token)
	return one(r, err, toContract)
}

func (s *Store) ListContracts(ctx context.Context, f domain.ContractFilter) ([]domain.Contract, error) {
	rows, err := s.q.ListContracts(ctx, sqlc.ListContractsParams{Deal: nz(f.DealID), Status: nz(f.Status), Lim: limitOr(f.Limit, 200)})
	return many(rows, err, toContract)
}

func (s *Store) SaveContract(ctx context.Context, c *domain.Contract) error {
	return wrap(s.q.UpsertContract(ctx, sqlc.UpsertContractParams{ID: c.ID, ContractNumber: c.ContractNumber, TemplateID: c.TemplateID, QuotationID: c.QuotationID, DealID: c.DealID, Type: c.Type, Title: c.Title, ClientName: c.ClientName, Body: c.Body, Status: c.Status, StartDate: c.StartDate, EndDate: c.EndDate, SignToken: c.SignToken, SentAt: c.SentAt, SignedAt: c.SignedAt, SignerName: c.SignerName, SignerIp: c.SignerIP, SignerUserAgent: c.SignerUserAgent, SignaturePng: c.SignaturePNG, ActivatedAt: c.ActivatedAt, EndedAt: c.EndedAt, CancelledAt: c.CancelledAt, CancelReason: c.CancelReason, CreatedBy: c.CreatedBy, UpdatedBy: c.UpdatedBy, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt}))
}

// ─── pitch decks ─────────────────────────────────────────────

func (s *Store) ListCaseStudies(ctx context.Context, activeOnly bool) ([]domain.CaseStudy, error) {
	rows, err := s.q.ListCaseStudies(ctx, activeOnly)
	return many(rows, err, func(r sqlc.SalesCaseStudy) domain.CaseStudy {
		return domain.CaseStudy{ID: r.ID, Title: r.Title, ClientName: r.ClientName, Industry: r.Industry, ServiceLines: r.ServiceLines, Challenge: r.Challenge, Solution: r.Solution, Results: r.Results, SortOrder: int(r.SortOrder), IsActive: r.IsActive}
	})
}

func (s *Store) SaveCaseStudy(ctx context.Context, c domain.CaseStudy) error {
	lines := c.ServiceLines
	if lines == nil {
		lines = []string{}
	}
	return wrap(s.q.UpsertCaseStudy(ctx, sqlc.UpsertCaseStudyParams{ID: c.ID, Title: c.Title, ClientName: c.ClientName, Industry: c.Industry, ServiceLines: lines, Challenge: c.Challenge, Solution: c.Solution, Results: c.Results, SortOrder: int32(c.SortOrder), IsActive: c.IsActive}))
}

func (s *Store) DeleteCaseStudy(ctx context.Context, id string) error {
	return s.q.DeleteCaseStudy(ctx, id)
}

func (s *Store) ListPitchDecks(ctx context.Context, dealID, leadID string) ([]domain.PitchDeck, error) {
	rows, err := s.q.ListPitchDecks(ctx, sqlc.ListPitchDecksParams{Deal: nz(dealID), Lead: nz(leadID)})
	return many(rows, err, func(r sqlc.ListPitchDecksRow) domain.PitchDeck {
		d := domain.PitchDeck{ID: r.ID, DealID: r.DealID, LeadID: r.LeadID, Title: r.Title, Version: int(r.Version), Model: r.Model, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, SizeBytes: r.SizeBytes}
		_ = json.Unmarshal(r.Options, &d.Options)
		_ = json.Unmarshal(r.Slides, &d.Slides)
		return d
	})
}

func (s *Store) GetPitchDeck(ctx context.Context, id string) (*domain.PitchDeck, error) {
	r, err := s.q.GetPitchDeck(ctx, id)
	return one(r, err, func(r sqlc.SalesPitchDeck) domain.PitchDeck {
		d := domain.PitchDeck{ID: r.ID, DealID: r.DealID, LeadID: r.LeadID, Title: r.Title, Version: int(r.Version), File: r.File, SizeBytes: int64(len(r.File)), Model: r.Model, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt}
		_ = json.Unmarshal(r.Options, &d.Options)
		_ = json.Unmarshal(r.Slides, &d.Slides)
		return d
	})
}

func (s *Store) InsertPitchDeck(ctx context.Context, d domain.PitchDeck) error {
	return wrap(s.q.InsertPitchDeck(ctx, sqlc.InsertPitchDeckParams{ID: d.ID, DealID: d.DealID, LeadID: d.LeadID, Title: d.Title, Version: int32(d.Version), Options: mustJSON(d.Options), Slides: mustJSON(d.Slides), File: d.File, Model: d.Model, CreatedBy: d.CreatedBy, CreatedAt: d.CreatedAt}))
}

func (s *Store) NextDeckVersion(ctx context.Context, dealID, leadID string) (int, error) {
	n, err := s.q.NextDeckVersion(ctx, sqlc.NextDeckVersionParams{Deal: nz(dealID), Lead: nz(leadID)})
	return int(n), err
}

// ─── onboarding ──────────────────────────────────────────────

func (s *Store) ListOnboardingTemplates(ctx context.Context) ([]domain.OnboardingTemplate, error) {
	rows, err := s.q.ListOnboardingTemplates(ctx)
	return many(rows, err, func(r sqlc.SalesOnboardingTemplate) domain.OnboardingTemplate {
		t := domain.OnboardingTemplate{ID: r.ID, Name: r.Name, IsDefault: r.IsDefault, UpdatedBy: r.UpdatedBy, UpdatedAt: r.UpdatedAt}
		_ = json.Unmarshal(r.Items, &t.Items)
		return t
	})
}

func (s *Store) SaveOnboardingTemplate(ctx context.Context, t domain.OnboardingTemplate) error {
	if err := s.q.UpsertOnboardingTemplate(ctx, sqlc.UpsertOnboardingTemplateParams{ID: t.ID, Name: t.Name, IsDefault: t.IsDefault, Items: mustJSON(t.Items), UpdatedBy: t.UpdatedBy}); err != nil {
		return wrap(err)
	}
	if t.IsDefault {
		return s.q.ClearDefaultOnboardingTemplate(ctx, t.ID)
	}
	return nil
}

func toOnboarding(r sqlc.SalesOnboarding) domain.Onboarding {
	o := domain.Onboarding{ID: r.ID, DealID: r.DealID, TemplateID: r.TemplateID, ClientName: r.ClientName, Status: r.Status, PortalToken: r.PortalToken, BriefSubmittedAt: r.BriefSubmittedAt, MarketingClientID: r.MarketingClientID, Progress: int(r.Progress), CompletedAt: r.CompletedAt, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, Brief: map[string]string{}}
	_ = json.Unmarshal(r.Brief, &o.Brief)
	return o
}

func toOnboardingItem(r sqlc.SalesOnboardingItem) domain.OnboardingItem {
	return domain.OnboardingItem{ID: r.ID, OnboardingID: r.OnboardingID, Key: r.Key, Kind: r.Kind, Title: r.Title, Description: r.Description, OwnerUserID: r.OwnerUserID, DueDate: r.DueDate, Status: r.Status, DoneAt: r.DoneAt, DoneBy: r.DoneBy, RemindedAt: r.RemindedAt, SortOrder: int(r.SortOrder)}
}

func toAsset(r sqlc.SalesOnboardingAsset) domain.OnboardingAsset {
	return domain.OnboardingAsset{ID: r.ID, OnboardingID: r.OnboardingID, FileName: r.FileName, ContentType: r.ContentType, SizeBytes: r.SizeBytes, StorageKey: r.StorageKey, UploaderIP: r.UploaderIp, CreatedAt: r.CreatedAt}
}

// withChildren loads items and assets.
func (s *Store) withChildren(ctx context.Context, os []domain.Onboarding) ([]domain.Onboarding, error) {
	if len(os) == 0 {
		return os, nil
	}
	idList := make([]string, len(os))
	index := map[string]int{}
	for i, o := range os {
		idList[i] = o.ID
		index[o.ID] = i
	}
	items, err := s.q.ListOnboardingItems(ctx, idList)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		o := &os[index[it.OnboardingID]]
		o.Items = append(o.Items, toOnboardingItem(it))
	}
	assets, err := s.q.ListOnboardingAssets(ctx, idList)
	if err != nil {
		return nil, err
	}
	for _, a := range assets {
		o := &os[index[a.OnboardingID]]
		o.Assets = append(o.Assets, toAsset(a))
	}
	return os, nil
}

func (s *Store) oneOnboarding(ctx context.Context, r sqlc.SalesOnboarding, err error) (*domain.Onboarding, error) {
	o, err := one(r, err, toOnboarding)
	if err != nil || o == nil {
		return o, err
	}
	os, err := s.withChildren(ctx, []domain.Onboarding{*o})
	if err != nil {
		return nil, err
	}
	return &os[0], nil
}

func (s *Store) GetOnboarding(ctx context.Context, id string) (*domain.Onboarding, error) {
	r, err := s.q.GetOnboarding(ctx, id)
	return s.oneOnboarding(ctx, r, err)
}

func (s *Store) GetOnboardingByDeal(ctx context.Context, dealID string) (*domain.Onboarding, error) {
	r, err := s.q.GetOnboardingByDeal(ctx, dealID)
	return s.oneOnboarding(ctx, r, err)
}

func (s *Store) GetOnboardingByToken(ctx context.Context, token string) (*domain.Onboarding, error) {
	r, err := s.q.GetOnboardingByToken(ctx, token)
	return s.oneOnboarding(ctx, r, err)
}

func (s *Store) ListOnboardings(ctx context.Context, status string) ([]domain.Onboarding, error) {
	rows, err := s.q.ListOnboardings(ctx, nz(status))
	os, err := many(rows, err, toOnboarding)
	if err != nil {
		return nil, err
	}
	return s.withChildren(ctx, os)
}

// CreateOnboarding inserts the onboarding with its items; false when the
// deal already has one.
func (s *Store) CreateOnboarding(ctx context.Context, o domain.Onboarding) (bool, error) {
	n, err := s.q.InsertOnboarding(ctx, sqlc.InsertOnboardingParams{ID: o.ID, DealID: o.DealID, TemplateID: o.TemplateID, ClientName: o.ClientName, Status: o.Status, PortalToken: o.PortalToken, Brief: mustJSON(o.Brief), Progress: int32(o.Progress), CreatedBy: o.CreatedBy, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt})
	if err != nil || n == 0 {
		return false, wrap(err)
	}
	for _, it := range o.Items {
		if err := s.SaveOnboardingItem(ctx, it); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (s *Store) SaveOnboarding(ctx context.Context, o *domain.Onboarding) error {
	brief := o.Brief
	if brief == nil {
		brief = map[string]string{}
	}
	return s.q.UpdateOnboarding(ctx, sqlc.UpdateOnboardingParams{ID: o.ID, ClientName: o.ClientName, Status: o.Status, Brief: mustJSON(brief), BriefSubmittedAt: o.BriefSubmittedAt, MarketingClientID: o.MarketingClientID, Progress: int32(o.Progress), CompletedAt: o.CompletedAt, UpdatedAt: o.UpdatedAt})
}

func (s *Store) GetOnboardingItem(ctx context.Context, id string) (*domain.OnboardingItem, error) {
	r, err := s.q.GetOnboardingItem(ctx, id)
	return one(r, err, toOnboardingItem)
}

func (s *Store) SaveOnboardingItem(ctx context.Context, it domain.OnboardingItem) error {
	return wrap(s.q.UpsertOnboardingItem(ctx, sqlc.UpsertOnboardingItemParams{ID: it.ID, OnboardingID: it.OnboardingID, Key: it.Key, Kind: it.Kind, Title: it.Title, Description: it.Description, OwnerUserID: it.OwnerUserID, DueDate: it.DueDate, Status: it.Status, DoneAt: it.DoneAt, DoneBy: it.DoneBy, RemindedAt: it.RemindedAt, SortOrder: int32(it.SortOrder)}))
}

func (s *Store) InsertOnboardingAsset(ctx context.Context, a domain.OnboardingAsset) error {
	return s.q.InsertOnboardingAsset(ctx, sqlc.InsertOnboardingAssetParams{ID: a.ID, OnboardingID: a.OnboardingID, FileName: a.FileName, ContentType: a.ContentType, SizeBytes: a.SizeBytes, StorageKey: a.StorageKey, UploaderIp: a.UploaderIP, CreatedAt: a.CreatedAt})
}

func (s *Store) GetOnboardingAsset(ctx context.Context, id string) (*domain.OnboardingAsset, error) {
	r, err := s.q.GetOnboardingAsset(ctx, id)
	return one(r, err, toAsset)
}

func (s *Store) DueOnboardingItems(ctx context.Context, until time.Time) ([]domain.OnboardingItem, error) {
	rows, err := s.q.DueOnboardingItems(ctx, &until)
	return many(rows, err, func(r sqlc.DueOnboardingItemsRow) domain.OnboardingItem {
		it := toOnboardingItem(sqlc.SalesOnboardingItem{ID: r.ID, OnboardingID: r.OnboardingID, Key: r.Key, Kind: r.Kind, Title: r.Title, Description: r.Description, OwnerUserID: r.OwnerUserID, DueDate: r.DueDate, Status: r.Status, DoneAt: r.DoneAt, DoneBy: r.DoneBy, RemindedAt: r.RemindedAt, SortOrder: r.SortOrder})
		it.ClientName = r.ClientName
		return it
	})
}

func (s *Store) MarkOnboardingItemReminded(ctx context.Context, id string, at time.Time) error {
	return s.q.MarkOnboardingItemReminded(ctx, sqlc.MarkOnboardingItemRemindedParams{ID: id, RemindedAt: &at})
}

// ─── conversations ───────────────────────────────────────────

func toConversation(r sqlc.SalesLeadConversation) domain.Conversation {
	return domain.Conversation{ID: r.ID, Channel: r.Channel, ExternalID: r.ExternalID, VisitorToken: r.VisitorToken, LeadID: r.LeadID, DisplayName: r.DisplayName, Phone: r.Phone, Email: r.Email, Status: r.Status, Priority: r.Priority, AssignedUserID: r.AssignedUserID, BotActive: r.BotActive, HandoffReason: r.HandoffReason, HandoffAt: r.HandoffAt, IsComplaint: r.IsComplaint, UnreadCount: int(r.UnreadCount), LastMessageAt: r.LastMessageAt, LastMessagePreview: r.LastMessagePreview, AwaitingSince: r.AwaitingSince, FirstResponseAt: r.FirstResponseAt, FirstResponseSeconds: r.FirstResponseSeconds, SLAResponseBreached: r.SlaResponseBreached, ResolvedAt: r.ResolvedAt, ResolutionSeconds: r.ResolutionSeconds, PageURL: r.PageUrl, Utm: jsonOr(r.Utm, "{}"), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func (s *Store) GetConversation(ctx context.Context, id string) (*domain.Conversation, error) {
	r, err := s.q.GetConversation(ctx, id)
	return one(r, err, toConversation)
}

func (s *Store) GetConversationByToken(ctx context.Context, token string) (*domain.Conversation, error) {
	r, err := s.q.GetConversationByToken(ctx, &token)
	return one(r, err, toConversation)
}

func (s *Store) GetConversationByExternal(ctx context.Context, channel, externalID string) (*domain.Conversation, error) {
	r, err := s.q.GetConversationByExternal(ctx, sqlc.GetConversationByExternalParams{Channel: channel, ExternalID: externalID})
	return one(r, err, toConversation)
}

func (s *Store) ListConversations(ctx context.Context, f domain.ConversationFilter) ([]domain.Conversation, error) {
	rows, err := s.q.ListConversations(ctx, sqlc.ListConversationsParams{Status: nz(f.Status), Channel: nz(f.Channel), Assigned: nz(f.AssignedUserID), Lead: nz(f.LeadID), Q: nz(f.Query), Lim: limitOr(f.Limit, 100)})
	return many(rows, err, toConversation)
}

func (s *Store) SaveConversation(ctx context.Context, c *domain.Conversation) error {
	return wrap(s.q.UpsertConversation(ctx, sqlc.UpsertConversationParams{ID: c.ID, Channel: c.Channel, ExternalID: c.ExternalID, VisitorToken: c.VisitorToken, LeadID: c.LeadID, DisplayName: c.DisplayName, Phone: c.Phone, Email: c.Email, Status: c.Status, Priority: c.Priority, AssignedUserID: c.AssignedUserID, BotActive: c.BotActive, HandoffReason: c.HandoffReason, HandoffAt: c.HandoffAt, IsComplaint: c.IsComplaint, UnreadCount: int32(c.UnreadCount), LastMessageAt: c.LastMessageAt, LastMessagePreview: c.LastMessagePreview, AwaitingSince: c.AwaitingSince, FirstResponseAt: c.FirstResponseAt, FirstResponseSeconds: c.FirstResponseSeconds, SlaResponseBreached: c.SLAResponseBreached, ResolvedAt: c.ResolvedAt, ResolutionSeconds: c.ResolutionSeconds, PageUrl: c.PageURL, Utm: jsonOr(c.Utm, "{}"), CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt}))
}

// InsertMessage reports false for a duplicate provider message.
func (s *Store) InsertMessage(ctx context.Context, m domain.Message) (bool, error) {
	n, err := s.q.InsertMessage(ctx, sqlc.InsertMessageParams{ID: m.ID, ConversationID: m.ConversationID, Direction: m.Direction, Sender: m.Sender, Body: m.Body, Status: m.Status, Provider: m.Provider, ProviderMessageID: m.ProviderMessageID, ErrorReason: m.ErrorReason, SentByUserID: m.SentByUserID, CreatedAt: m.CreatedAt})
	return n == 1, err
}

func toMessage(r sqlc.SalesLeadMessage) domain.Message {
	return domain.Message{ID: r.ID, ConversationID: r.ConversationID, Direction: r.Direction, Sender: r.Sender, Body: r.Body, Status: r.Status, Provider: r.Provider, ProviderMessageID: r.ProviderMessageID, ErrorReason: r.ErrorReason, SentByUserID: r.SentByUserID, CreatedAt: r.CreatedAt}
}

func (s *Store) ListMessages(ctx context.Context, conversationID string, after time.Time, limit int) ([]domain.Message, error) {
	rows, err := s.q.ListMessages(ctx, sqlc.ListMessagesParams{ConversationID: conversationID, CreatedAt: after, Limit: limitOr(limit, 500)})
	return many(rows, err, toMessage)
}

func (s *Store) RecentMessages(ctx context.Context, conversationID string, limit int) ([]domain.Message, error) {
	rows, err := s.q.RecentMessages(ctx, sqlc.RecentMessagesParams{ConversationID: conversationID, Limit: limitOr(limit, 30)})
	return many(rows, err, toMessage)
}

func (s *Store) InsertNote(ctx context.Context, n domain.ConversationNote) error {
	return s.q.InsertNote(ctx, sqlc.InsertNoteParams{ID: n.ID, ConversationID: n.ConversationID, AuthorUserID: n.AuthorUserID, Body: n.Body, CreatedAt: n.CreatedAt})
}

func (s *Store) ListNotes(ctx context.Context, conversationID string) ([]domain.ConversationNote, error) {
	rows, err := s.q.ListNotes(ctx, conversationID)
	return many(rows, err, func(r sqlc.SalesConversationNote) domain.ConversationNote {
		return domain.ConversationNote{ID: r.ID, ConversationID: r.ConversationID, AuthorUserID: r.AuthorUserID, Body: r.Body, CreatedAt: r.CreatedAt}
	})
}

func (s *Store) ListReplyTemplates(ctx context.Context, activeOnly bool) ([]domain.ReplyTemplate, error) {
	rows, err := s.q.ListReplyTemplates(ctx, activeOnly)
	return many(rows, err, func(r sqlc.SalesReplyTemplate) domain.ReplyTemplate {
		return domain.ReplyTemplate{ID: r.ID, Title: r.Title, Body: r.Body, IsActive: r.IsActive, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
	})
}

func (s *Store) SaveReplyTemplate(ctx context.Context, t domain.ReplyTemplate) error {
	return wrap(s.q.UpsertReplyTemplate(ctx, sqlc.UpsertReplyTemplateParams{ID: t.ID, Title: t.Title, Body: t.Body, IsActive: t.IsActive, CreatedBy: t.CreatedBy}))
}

func (s *Store) DeleteReplyTemplate(ctx context.Context, id string) error {
	return s.q.DeleteReplyTemplate(ctx, id)
}

func (s *Store) GetInsight(ctx context.Context, conversationID string) (*domain.Insight, error) {
	r, err := s.q.GetInsight(ctx, conversationID)
	return one(r, err, func(r sqlc.SalesConversationInsight) domain.Insight {
		in := domain.Insight{ConversationID: r.ConversationID, Summary: r.Summary, Topic: r.Topic, Sentiment: r.Sentiment, IsComplaint: r.IsComplaint, Fingerprint: r.Fingerprint, Model: r.Model, AnalyzedAt: r.AnalyzedAt}
		_ = json.Unmarshal(r.Keywords, &in.Keywords)
		return in
	})
}

func (s *Store) SaveInsight(ctx context.Context, in domain.Insight) error {
	kw := in.Keywords
	if kw == nil {
		kw = []string{}
	}
	return s.q.UpsertInsight(ctx, sqlc.UpsertInsightParams{ConversationID: in.ConversationID, Summary: in.Summary, Topic: in.Topic, Sentiment: in.Sentiment, IsComplaint: in.IsComplaint, Keywords: mustJSON(kw), Fingerprint: in.Fingerprint, Model: in.Model, AnalyzedAt: in.AnalyzedAt})
}

func (s *Store) GetChatbotSettings(ctx context.Context) (*domain.ChatbotSettings, error) {
	r, err := s.q.GetChatbotSettings(ctx)
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.ChatbotSettings{Enabled: r.Enabled, BotName: r.BotName, Persona: r.Persona, ServicesOffered: r.ServicesOffered, PricingHints: r.PricingHints, BusinessHoursStart: int(r.BusinessHoursStart), BusinessHoursEnd: int(r.BusinessHoursEnd), HandoffRules: r.HandoffRules, Greeting: r.Greeting, MinConfidence: domain.ParsePercent(r.MinConfidence), HandoffUserIDs: r.HandoffUserIds, UpdatedBy: r.UpdatedBy, UpdatedAt: r.UpdatedAt}, nil
}

func (s *Store) SaveChatbotSettings(ctx context.Context, c domain.ChatbotSettings) error {
	if c.HandoffUserIDs == nil {
		c.HandoffUserIDs = []string{}
	}
	return s.q.UpsertChatbotSettings(ctx, sqlc.UpsertChatbotSettingsParams{HandoffUserIds: c.HandoffUserIDs, Enabled: c.Enabled, BotName: c.BotName, Persona: c.Persona, ServicesOffered: c.ServicesOffered, PricingHints: c.PricingHints, BusinessHoursStart: int32(c.BusinessHoursStart), BusinessHoursEnd: int32(c.BusinessHoursEnd), HandoffRules: c.HandoffRules, Greeting: c.Greeting, MinConfidence: domain.PercentString(c.MinConfidence), UpdatedBy: c.UpdatedBy, UpdatedAt: c.UpdatedAt})
}

func (s *Store) MarkSLABreaches(ctx context.Context, before time.Time) ([]domain.Conversation, error) {
	rows, err := s.q.MarkSLABreaches(ctx, &before)
	return many(rows, err, toConversation)
}
