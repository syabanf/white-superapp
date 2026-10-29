package app

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Automation is the CRM event bus, workflow engine and lead scoring of
// crm/advance (engine.go, actions.go, scoring.go). Everything triggered by
// an event is best effort: failures are logged and never fail the action
// that raised the event.
type Automation struct{ *Deps }

func NewAutomation(d *Deps) *Automation { return &Automation{d} }

// Actor is the signed-in caller as the use cases need it.
type Actor struct {
	UserID   string
	Admin    bool
	Approver bool
}

func (a Actor) ref() *string {
	if a.UserID == "" {
		return nil
	}
	return &a.UserID
}

// Event is one CRM event ("lead.created", "deal.stage_changed", ...).
type Event struct {
	Type        string
	SubjectType string
	SubjectID   string
	Payload     map[string]any
	Actor       *string
	Changes     map[string]domain.Change
}

// Scoring signals counted by event rules (crm_events on the lead).
const (
	SignalTaskDone      = "task_done"
	SignalChatInbound   = "chat_inbound"
	SignalDealCreated   = "deal_created"
	SignalQuotationSent = "quotation_sent"
)

// Emit records the event, runs matching workflow rules and rescores the
// affected lead (emitting lead.score_changed when it moved).
func (a *Automation) Emit(ctx context.Context, ev Event) {
	if err := a.process(ctx, ev); err != nil {
		a.warn("crm event failed", "event", ev.Type, "err", err)
	}
}

func (a *Automation) process(ctx context.Context, ev Event) error {
	eventID := ids.New()
	if err := a.Store.InsertCrmEvent(ctx, domain.CrmEvent{ID: eventID, EventType: ev.Type, SubjectType: ev.SubjectType, SubjectID: ev.SubjectID, Payload: ev.Payload, ActorUserID: ev.Actor, CreatedAt: a.now()}); err != nil {
		return err
	}
	a.runEvent(ctx, ev, &eventID)
	leadID, err := a.affectedLead(ctx, ev.SubjectType, ev.SubjectID)
	if err != nil || leadID == "" {
		return err
	}
	changed, prev, score, err := a.RecalculateLead(ctx, leadID)
	if err != nil || !changed {
		return err
	}
	scoreEv := Event{Type: "lead.score_changed", SubjectType: "lead", SubjectID: leadID, Actor: ev.Actor,
		Payload: map[string]any{"previous_score": float64(prev), "score": float64(score), "source_event": ev.Type}}
	scoreID := ids.New()
	if err := a.Store.InsertCrmEvent(ctx, domain.CrmEvent{ID: scoreID, EventType: scoreEv.Type, SubjectType: "lead", SubjectID: leadID, Payload: scoreEv.Payload, ActorUserID: ev.Actor, CreatedAt: a.now()}); err != nil {
		return err
	}
	a.runEvent(ctx, scoreEv, &scoreID)
	return nil
}

// Signal records a scoring signal on a lead and rescores it.
func (a *Automation) Signal(ctx context.Context, leadID, signal, subType string) {
	if leadID == "" {
		return
	}
	payload := map[string]any{}
	if subType != "" {
		payload["sub_type"] = subType
	}
	if err := a.Store.InsertCrmEvent(ctx, domain.CrmEvent{ID: ids.New(), EventType: signal, SubjectType: "lead", SubjectID: leadID, Payload: payload, CreatedAt: a.now()}); err != nil {
		a.warn("scoring signal failed", "signal", signal, "err", err)
		return
	}
	if _, _, _, err := a.RecalculateLead(ctx, leadID); err != nil {
		a.warn("rescore failed", "lead", leadID, "err", err)
	}
}

// affectedLead is the lead behind a subject (the lead itself, a deal's or
// quotation's lead, a task's lead).
func (a *Automation) affectedLead(ctx context.Context, subjectType, subjectID string) (string, error) {
	switch subjectType {
	case "lead":
		return subjectID, nil
	case "deal":
		d, err := a.Store.GetDeal(ctx, subjectID)
		if err != nil || d == nil {
			return "", err
		}
		return deref(d.LeadID), nil
	case "quotation":
		q, err := a.Store.GetQuotation(ctx, subjectID)
		if err != nil || q == nil {
			return "", err
		}
		return a.affectedLead(ctx, "deal", q.DealID)
	case "task":
		t, err := a.Store.GetActivity(ctx, subjectID)
		if err != nil || t == nil {
			return "", err
		}
		if t.LeadID != nil {
			return *t.LeadID, nil
		}
		if t.DealID != nil {
			return a.affectedLead(ctx, "deal", *t.DealID)
		}
	}
	return "", nil
}

func (a *Automation) runEvent(ctx context.Context, ev Event, eventID *string) {
	object, trigger, ok := domain.TriggerForEvent(ev.Type)
	if !ok {
		return
	}
	rules, err := a.Store.ListWorkflowRules(ctx, object, trigger, true)
	if err != nil {
		a.warn("load workflow rules failed", "err", err)
		return
	}
	payload := ev.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	for _, rule := range rules {
		err := a.Store.WithTx(ctx, func(st domain.Store) error {
			return a.runRule(ctx, st, rule, ev.SubjectID, eventID, ev.Actor, payload, ev.Changes, nil)
		})
		if err != nil {
			a.warn("workflow rule failed", "rule", rule.Name, "err", err)
		}
	}
}

// snapshot is the JSON record conditions and templates read.
func snapshot(ctx context.Context, st domain.Store, object, id string) (map[string]any, error) {
	var v any
	var err error
	switch object {
	case "lead":
		v, err = nilable(st.GetLead(ctx, id))
	case "deal":
		v, err = nilable(st.GetDeal(ctx, id))
	case "account":
		v, err = nilable(st.GetAccount(ctx, id))
	case "contact":
		v, err = nilable(st.GetContact(ctx, id))
	case "task":
		v, err = nilable(st.GetActivity(ctx, id))
	case "quotation":
		v, err = nilable(st.GetQuotation(ctx, id))
	}
	if err != nil || v == nil {
		return nil, err
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	return out, json.Unmarshal(raw, &out)
}

// nilable keeps a typed nil pointer from becoming a non-nil interface.
func nilable[T any](v *T, err error) (any, error) {
	if v == nil {
		return nil, err
	}
	return v, err
}

// runRule evaluates one rule for one subject. since limits the
// run-once-per-record check for time-based triggers.
func (a *Automation) runRule(ctx context.Context, st domain.Store, rule domain.WorkflowRule, subjectID string, eventID, actor *string, payload map[string]any, changes map[string]domain.Change, since *time.Time) error {
	if rule.RunOncePerRecord || since != nil {
		ran, err := st.HasRuleRun(ctx, rule.ID, subjectID, since)
		if err != nil || ran {
			return err
		}
	}
	rec, err := snapshot(ctx, st, rule.Object, subjectID)
	if err != nil || rec == nil {
		return err
	}
	if !domain.TriggerConfigMatches(rule.TriggerType, rule.TriggerConfig, rec, payload) {
		return nil
	}
	now := a.now()
	if !domain.EvaluateConditions(rule.Conditions, rec, changes) {
		return st.InsertWorkflowRun(ctx, domain.WorkflowRun{ID: ids.New(), RuleID: rule.ID, EventID: eventID, SubjectType: rule.Object, SubjectID: subjectID, Status: "skipped", CreatedAt: now})
	}
	run := domain.WorkflowRun{ID: ids.New(), RuleID: rule.ID, EventID: eventID, SubjectType: rule.Object, SubjectID: subjectID, Status: "scheduled", CreatedAt: now}
	if err := st.InsertWorkflowRun(ctx, run); err != nil {
		return err
	}
	ac := &actionCtx{st: st, ruleID: &rule.ID, runID: &run.ID, object: rule.Object, subjectID: subjectID, rec: rec, actor: actor, payload: payload}
	results, scheduled, failed, err := a.runActions(ctx, rule.Actions, ac)
	if err != nil {
		return err
	}
	run.Status = domain.RunStatus(len(results), failed, scheduled)
	run.ActionsResult = results
	if failed > 0 {
		run.Error = ptr(strconv.Itoa(failed) + " aksi gagal")
	}
	if err := st.UpdateWorkflowRun(ctx, run); err != nil {
		return err
	}
	return st.MarkRuleRun(ctx, rule.ID, now)
}

type actionCtx struct {
	st        domain.Store
	ruleID    *string
	runID     *string
	object    string
	subjectID string
	rec       map[string]any
	actor     *string
	payload   map[string]any
}

func (ac *actionCtx) template() map[string]any {
	stage := map[string]any{"name": ac.rec["stage_name"], "code": ac.rec["stage_code"]}
	ctx := map[string]any{"score": ac.rec["score"], "stage": stage, "event": ac.payload, ac.object: ac.rec,
		"owner": map[string]any{"id": ac.rec["owner_user_id"]}}
	if ac.object == "deal" {
		ctx["lead"] = map[string]any{"org_name": ac.rec["org_name"], "pic_name": ac.rec["pic_name"], "pic_phone": ac.rec["pic_phone"]}
	}
	return ctx
}

func (ac *actionCtx) owner() *string {
	if s, ok := ac.rec["owner_user_id"].(string); ok && s != "" {
		return &s
	}
	return nil
}

// runActions runs the actions before the first wait (one failure does not
// stop the rest) and queues the ones after it.
func (a *Automation) runActions(ctx context.Context, actions []domain.WorkflowAction, ac *actionCtx) ([]map[string]any, bool, int, error) {
	now, later, delayHours := domain.SplitAtWait(actions)
	results := []map[string]any{}
	failed := 0
	for _, act := range now {
		res, err := a.execute(ctx, act, ac)
		if err != nil {
			res = map[string]any{"type": act.Type, "ok": false, "reason": err.Error()}
		}
		if res["ok"] != true {
			failed++
		}
		results = append(results, res)
	}
	if len(later) == 0 {
		return results, false, failed, nil
	}
	ctxMap := map[string]any{"payload": ac.payload}
	if ac.actor != nil {
		ctxMap["actor_user_id"] = *ac.actor
	}
	err := ac.st.InsertScheduledAction(ctx, domain.ScheduledAction{ID: ids.New(), RuleID: ac.ruleID, RunID: ac.runID, SubjectType: ac.object, SubjectID: ac.subjectID, Actions: later, Context: ctxMap, RunAt: a.now().Add(time.Duration(delayHours * float64(time.Hour)))})
	return results, err == nil, failed, err
}

func (a *Automation) execute(ctx context.Context, act domain.WorkflowAction, ac *actionCtx) (map[string]any, error) {
	res := map[string]any{"type": act.Type}
	tpl := ac.template()
	switch act.Type {
	case "wait":
		res["ok"] = true
	case "create_task":
		var owner *string
		switch act.AssignTo {
		case "", "owner":
			owner = ac.owner()
		case "creator":
			owner = ac.actor
		default:
			owner = ptr(act.AssignTo)
		}
		days := 1.0
		if act.DueInDays != nil {
			days = *act.DueInDays
		}
		due := a.now().Add(time.Duration(days * float64(24*time.Hour))).In(domain.Jakarta)
		due = time.Date(due.Year(), due.Month(), due.Day(), 9, 0, 0, 0, domain.Jakarta).UTC()
		activityType := act.ActivityType
		if !domain.Contains(domain.ActivityTypes, activityType) {
			activityType = "tugas"
		}
		priority := act.Priority
		if !domain.Contains(domain.TaskPriorities, priority) {
			priority = "normal"
		}
		t := domain.Activity{ID: ids.New(), ActivityType: activityType, Title: domain.SliceUTF16(domain.RenderTemplate(act.Title, tpl), 200), Priority: priority, Status: "open", DueAt: &due, OwnerUserID: owner,
			Audit: domain.Audit{CreatedBy: ac.actor, UpdatedBy: ac.actor, CreatedAt: a.now(), UpdatedAt: a.now()}}
		switch ac.object {
		case "lead":
			t.LeadID = &ac.subjectID
		case "deal":
			t.DealID = &ac.subjectID
		}
		if ac.object == "lead" || ac.object == "deal" || ac.object == "account" || ac.object == "contact" {
			t.SubjectType, t.SubjectID = &ac.object, &ac.subjectID
		}
		if act.Notes != "" {
			t.Notes = ptr(domain.RenderTemplate(act.Notes, tpl))
		}
		if err := ac.st.SaveActivity(ctx, &t); err != nil {
			return nil, err
		}
		res["ok"], res["task_id"] = true, t.ID
	case "assign_owner":
		if act.OnlyIfEmpty && ac.owner() != nil {
			res["ok"], res["skipped"] = true, "sudah ada owner"
			return res, nil
		}
		userID := act.UserID
		if act.Strategy == "round_robin" && len(act.UserIDs) > 0 {
			n, err := ac.st.CountRuleRuns(ctx, deref(ac.ruleID))
			if err != nil {
				return nil, err
			}
			userID, _ = domain.PickRoundRobin(act.UserIDs, n)
		}
		if userID == "" {
			res["ok"], res["reason"] = false, "tidak ada kandidat owner"
			return res, nil
		}
		if err := setSubjectField(ctx, ac.st, ac.object, ac.subjectID, "owner_user_id", userID, a.now()); err != nil {
			return nil, err
		}
		ac.rec["owner_user_id"] = userID
		res["ok"], res["owner_user_id"] = true, userID
	case "update_field":
		if !domain.CanUpdateField(ac.object, act.Field) {
			res["ok"], res["reason"] = false, "field tidak diizinkan"
			return res, nil
		}
		value := domain.ValueString(act.Value)
		if err := setSubjectField(ctx, ac.st, ac.object, ac.subjectID, act.Field, value, a.now()); err != nil {
			return nil, err
		}
		ac.rec[act.Field] = value
		res["ok"], res["field"] = true, act.Field
	case "notify_in_app":
		var to []string
		switch {
		case act.To == "user" && act.UserID != "":
			to = []string{act.UserID}
		case ac.owner() != nil:
			to = []string{*ac.owner()}
		case ac.actor != nil:
			to = []string{*ac.actor}
		}
		if len(to) == 0 {
			res["ok"], res["reason"] = false, "tidak ada penerima"
			return res, nil
		}
		a.notify(ctx, Notification{UserIDs: to, Type: NotifWorkflow, Title: domain.SliceUTF16(domain.RenderTemplate(act.Title, tpl), 150), Body: domain.SliceUTF16(domain.RenderTemplate(act.Message, tpl), 1000), Href: subjectLink(ac.object, ac.subjectID, ac.rec)})
		res["ok"], res["notified"] = true, len(to)
	default:
		res["ok"], res["reason"] = false, "aksi tidak didukung"
	}
	return res, nil
}

// setSubjectField writes one whitelisted field (or the owner) of a record.
func setSubjectField(ctx context.Context, st domain.Store, object, id, field, value string, now time.Time) error {
	v := domain.TrimmedOrNil(value)
	switch object {
	case "lead":
		l, err := st.GetLead(ctx, id)
		if err != nil || l == nil {
			return err
		}
		switch field {
		case "owner_user_id":
			l.OwnerUserID = v
		case "temperature":
			if domain.Contains(domain.LeadTemperatures, value) {
				l.Temperature = value
			}
		case "status":
			if domain.Contains(domain.LeadStatuses, value) {
				l.Status = value
			}
		case "source":
			if domain.Contains(domain.LeadSources, value) {
				l.Source = value
			}
		case "notes":
			l.Notes = v
		case "city":
			l.City = v
		}
		l.UpdatedAt = now
		return st.SaveLead(ctx, l)
	case "deal":
		d, err := st.GetDeal(ctx, id)
		if err != nil || d == nil {
			return err
		}
		switch field {
		case "owner_user_id":
			d.OwnerUserID = v
		case "forecast_category":
			if domain.Contains(domain.ForecastCategories, value) && !d.IsWon && !d.IsLost {
				d.ForecastCategory = value
			}
		case "notes":
			d.Notes = v
		}
		d.UpdatedAt = now
		return st.SaveDeal(ctx, d)
	case "account":
		ac, err := st.GetAccount(ctx, id)
		if err != nil || ac == nil {
			return err
		}
		switch field {
		case "owner_user_id":
			ac.OwnerUserID = v
		case "account_type":
			if domain.Contains(domain.AccountTypes, value) {
				ac.AccountType = value
			}
		case "industry":
			ac.Industry = v
		case "city":
			ac.City = v
		case "notes":
			ac.Notes = v
		}
		ac.UpdatedAt = now
		return st.SaveAccount(ctx, ac)
	case "contact":
		c, err := st.GetContact(ctx, id)
		if err != nil || c == nil {
			return err
		}
		switch field {
		case "title":
			c.Title = v
		case "notes":
			c.Notes = v
		}
		c.UpdatedAt = now
		return st.SaveContact(ctx, c)
	case "task":
		t, err := st.GetActivity(ctx, id)
		if err != nil || t == nil {
			return err
		}
		switch field {
		case "owner_user_id":
			t.OwnerUserID = v
		case "priority":
			if domain.Contains(domain.TaskPriorities, value) {
				t.Priority = value
			}
		case "status":
			if domain.Contains(domain.TaskStatuses, value) {
				t.Status = value
			}
		case "notes":
			t.Notes = v
		}
		t.UpdatedAt = now
		return st.SaveActivity(ctx, t)
	case "quotation":
		q, err := st.GetQuotation(ctx, id)
		if err != nil || q == nil {
			return err
		}
		if field == "notes" {
			q.Notes = v
		}
		q.UpdatedAt = now
		return st.SaveQuotation(ctx, q)
	}
	return nil
}

func subjectLink(object, id string, rec map[string]any) string {
	switch object {
	case "lead":
		return "/erp/sales/leads/" + id
	case "deal":
		return "/erp/sales/pipeline/" + id
	case "account":
		return "/erp/sales/leads/accounts/" + id
	case "task":
		// Tasks have no page of their own: open the deal or lead they belong to.
		if d := domain.ValueString(rec["deal_id"]); d != "" {
			return "/erp/sales/pipeline/" + d
		}
		if l := domain.ValueString(rec["lead_id"]); l != "" {
			return "/erp/sales/leads/" + l
		}
		return "/erp/sales"
	case "quotation":
		return "/erp/sales/pipeline/" + domain.ValueString(rec["deal_id"])
	}
	return "/erp/sales"
}

// RunDueScheduledActions runs queued steps after a wait (workflow-watcher).
func (a *Automation) RunDueScheduledActions(ctx context.Context) (int, error) {
	due, err := a.Store.DueScheduledActions(ctx, a.now(), 50)
	if err != nil {
		return 0, err
	}
	for _, sa := range due {
		err := a.Store.WithTx(ctx, func(st domain.Store) error {
			rec, err := snapshot(ctx, st, sa.SubjectType, sa.SubjectID)
			if err != nil {
				return err
			}
			if rec == nil {
				return st.FinishScheduledAction(ctx, sa.ID, "cancelled", ptr("record sudah dihapus"))
			}
			payload, _ := sa.Context["payload"].(map[string]any)
			var actor *string
			if s, ok := sa.Context["actor_user_id"].(string); ok && s != "" {
				actor = &s
			}
			ac := &actionCtx{st: st, ruleID: sa.RuleID, runID: sa.RunID, object: sa.SubjectType, subjectID: sa.SubjectID, rec: rec, actor: actor, payload: payload}
			_, _, failed, err := a.runActions(ctx, sa.Actions, ac)
			if err != nil {
				return err
			}
			status := "done"
			var errText *string
			if failed > 0 {
				status, errText = "failed", ptr(strconv.Itoa(failed)+" aksi gagal")
			}
			return st.FinishScheduledAction(ctx, sa.ID, status, errText)
		})
		if err != nil {
			a.warn("scheduled action failed", "id", sa.ID, "err", err)
		}
	}
	return len(due), nil
}

// ScanTimeBasedRules fires inactive_days (lead, deal) and due_soon (task)
// rules (scanTimeBasedRules in workflow-watcher).
func (a *Automation) ScanTimeBasedRules(ctx context.Context) error {
	now := a.now()
	for _, trigger := range []string{"inactive_days", "due_soon"} {
		rules, err := a.Store.ListWorkflowRules(ctx, "", trigger, true)
		if err != nil {
			return err
		}
		for _, rule := range rules {
			var subjects []string
			var since *time.Time
			switch {
			case trigger == "inactive_days" && (rule.Object == "lead" || rule.Object == "deal"):
				days := domain.ValueNumber(rule.TriggerConfig["days"])
				if !(days > 0) {
					days = 7
				}
				before := now.Add(-time.Duration(days * float64(24*time.Hour)))
				since = &before
				if rule.Object == "lead" {
					subjects, err = a.Store.StaleLeadIDs(ctx, before)
				} else {
					subjects, err = a.Store.StaleDealIDs(ctx, before)
				}
			case trigger == "due_soon" && rule.Object == "task":
				hours := domain.ValueNumber(rule.TriggerConfig["hours"])
				if !(hours > 0) {
					hours = 24
				}
				subjects, err = a.Store.TasksDueBetween(ctx, now, now.Add(time.Duration(hours*float64(time.Hour))))
				epoch := time.Unix(0, 0)
				since = &epoch
			}
			if err != nil {
				return err
			}
			for _, id := range subjects {
				err := a.Store.WithTx(ctx, func(st domain.Store) error {
					return a.runRule(ctx, st, rule, id, nil, nil, map[string]any{}, nil, since)
				})
				if err != nil {
					a.warn("time rule failed", "rule", rule.Name, "err", err)
				}
			}
		}
	}
	return nil
}

// ─── scoring ─────────────────────────────────────────────────

// RecalculateLead rescores one lead (recalculateLeadScore).
func (a *Automation) RecalculateLead(ctx context.Context, leadID string) (changed bool, prev, score int, err error) {
	lead, err := a.Store.GetLead(ctx, leadID)
	if err != nil || lead == nil {
		return false, 0, 0, err
	}
	rules, err := a.Store.ListScoringRules(ctx, true)
	if err != nil {
		return false, 0, 0, err
	}
	snap := map[string]string{"source": lead.Source, "org_type": lead.OrgType, "temperature": lead.Temperature, "status": lead.Status, "city": deref(lead.City), "pic_email": deref(lead.PicEmail), "pic_title": deref(lead.PicTitle)}
	if lead.AccountID != nil {
		if acc, err := a.Store.GetAccount(ctx, *lead.AccountID); err != nil {
			return false, 0, 0, err
		} else if acc != nil {
			snap["account_type"], snap["industry"] = acc.AccountType, deref(acc.Industry)
		}
	}
	counts := map[string]int{}
	needs := map[string]bool{}
	for _, r := range rules {
		if r.Kind == "event" && r.EventType != "" {
			needs[r.EventType] = true
		}
	}
	types := make([]string, 0, len(needs))
	for t := range needs {
		types = append(types, t)
	}
	sort.Strings(types)
	for _, t := range types {
		var since *time.Time
		if w := domain.MaxWindow(rules, t); w != nil {
			s := a.now().AddDate(0, 0, -*w)
			since = &s
		}
		got, err := a.Store.CountSubjectEvents(ctx, []string{leadID}, since)
		if err != nil {
			return false, 0, 0, err
		}
		for k, n := range got {
			if k == t || len(k) > len(t) && k[:len(t)+1] == t+":" {
				counts[k] = n
			}
		}
	}
	score, breakdown := domain.ComputeLeadScore(rules, snap, counts)
	if err := a.Store.UpdateLeadScore(ctx, leadID, score, breakdown, a.now()); err != nil {
		return false, 0, 0, err
	}
	return score != lead.Score, lead.Score, score, nil
}

// RecalculateAll rescores every lead.
func (a *Automation) RecalculateAll(ctx context.Context) (total, changed int, err error) {
	leadIDs, err := a.Store.ListLeadIDs(ctx)
	if err != nil {
		return 0, 0, err
	}
	for _, id := range leadIDs {
		c, _, _, err := a.RecalculateLead(ctx, id)
		if err != nil {
			return 0, 0, err
		}
		if c {
			changed++
		}
	}
	return len(leadIDs), changed, nil
}

// ─── rule CRUD (ADMIN) ───────────────────────────────────────

func (a *Automation) ListRules(ctx context.Context) ([]domain.WorkflowRule, error) {
	return a.Store.ListWorkflowRules(ctx, "", "", false)
}

func (a *Automation) SaveRule(ctx context.Context, actor Actor, r domain.WorkflowRule) (*domain.WorkflowRule, error) {
	if err := fieldErrs(domain.ValidateWorkflowRule(r.Name, r.Object, r.TriggerType, r.Conditions, r.Actions)); err != nil {
		return nil, err
	}
	if r.ID == "" {
		r.ID = ids.New()
		r.CreatedBy = actor.ref()
	} else if cur, err := a.Store.GetWorkflowRule(ctx, r.ID); err != nil {
		return nil, err
	} else if cur == nil {
		return nil, notFound(msgRuleNotFound)
	}
	if err := a.Store.SaveWorkflowRule(ctx, r); err != nil {
		return nil, err
	}
	return a.Store.GetWorkflowRule(ctx, r.ID)
}

func (a *Automation) DeleteRule(ctx context.Context, id string) error {
	return a.Store.DeleteWorkflowRule(ctx, id)
}

func (a *Automation) ListRuns(ctx context.Context, ruleID string) ([]domain.WorkflowRun, error) {
	return a.Store.ListWorkflowRuns(ctx, ruleID, 100)
}

func (a *Automation) ListScoringRules(ctx context.Context) ([]domain.ScoringRule, error) {
	return a.Store.ListScoringRules(ctx, false)
}

func (a *Automation) SaveScoringRule(ctx context.Context, r domain.ScoringRule) (domain.ScoringRule, error) {
	errs := map[string]string{}
	if domain.JSTrim(r.Name) == "" {
		errs["name"] = "Nama aturan wajib diisi"
	}
	switch r.Kind {
	case "field":
		if !domain.Contains(domain.ScoringFields, r.Field) {
			errs["field"] = "Field tidak valid"
		}
		if !domain.Contains(domain.ScoringOperators, r.Operator) {
			errs["operator"] = "Operator tidak valid"
		}
	case "event":
		if !domain.Contains(domain.ScoringEventTypes, r.EventType) {
			errs["eventType"] = "Jenis event tidak valid"
		}
	default:
		errs["kind"] = "Jenis aturan tidak valid"
	}
	if r.Points < -1000 || r.Points > 1000 {
		errs["points"] = "Poin harus antara -1000 dan 1000"
	}
	if err := fieldErrs(errs); err != nil {
		return r, err
	}
	if r.ID == "" {
		r.ID = ids.New()
	}
	r.MaxCount = max(1, r.MaxCount)
	return r, a.Store.SaveScoringRule(ctx, r)
}

func (a *Automation) DeleteScoringRule(ctx context.Context, id string) error {
	return a.Store.DeleteScoringRule(ctx, id)
}
