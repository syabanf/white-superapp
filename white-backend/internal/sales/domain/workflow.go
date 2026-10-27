package domain

import (
	"encoding/json"
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// The workflow engine rules of lib/crm/workflow.ts (crm/advance/domain).
// Records are JSON snapshots: string, float64, bool, nil, []any or
// map[string]any values keyed by snake_case column names.

var (
	WorkflowObjects  = []string{"lead", "deal", "account", "contact", "task", "quotation"}
	WorkflowTriggers = []string{"created", "updated", "stage_changed", "status_changed", "score_reached", "inactive_days", "due_soon"}
	ConditionOps     = []string{"eq", "neq", "in", "not_in", "contains", "gt", "gte", "lt", "lte", "is_empty", "not_empty", "changed", "changed_to"}
	// WorkflowActions drops send_wa and webhook until a provider is
	// configured; the WhatsApp provider port stays in app.
	WorkflowActions = []string{"create_task", "assign_owner", "update_field", "notify_in_app", "wait"}
)

// UpdatableFields whitelists the fields update_field may write per object.
var UpdatableFields = map[string][]string{
	"lead":      {"temperature", "status", "source", "notes", "city"},
	"deal":      {"forecast_category", "notes"},
	"account":   {"account_type", "industry", "city", "notes"},
	"contact":   {"title", "notes"},
	"task":      {"priority", "status", "notes"},
	"quotation": {"notes"},
}

// CanUpdateField reports whether update_field may write field on object.
func CanUpdateField(object, field string) bool {
	return slices.Contains(UpdatableFields[object], field)
}

// Condition is one rule condition.
type Condition struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value any    `json:"value,omitempty"`
}

// Change is one entry of the updated-record change map.
type Change struct{ From, To any }

// WorkflowAction is one stored action.
type WorkflowAction struct {
	Type         string   `json:"type"`
	Title        string   `json:"title,omitempty"`
	ActivityType string   `json:"activity_type,omitempty"`
	Notes        string   `json:"notes,omitempty"`
	DueInDays    *float64 `json:"due_in_days,omitempty"`
	Priority     string   `json:"priority,omitempty"`
	AssignTo     string   `json:"assign_to,omitempty"`
	UserID       string   `json:"user_id,omitempty"`
	Strategy     string   `json:"strategy,omitempty"`
	UserIDs      []string `json:"user_ids,omitempty"`
	OnlyIfEmpty  bool     `json:"only_if_empty,omitempty"`
	Field        string   `json:"field,omitempty"`
	Value        any      `json:"value,omitempty"`
	To           string   `json:"to,omitempty"`
	Message      string   `json:"message,omitempty"`
	Role         string   `json:"role,omitempty"`
	Hours        float64  `json:"hours,omitempty"`
	Days         float64  `json:"days,omitempty"`
}

// ValueString is String(v) for a JSON value (null and missing are "").
func ValueString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = ValueString(e)
		}
		return strings.Join(parts, ",")
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// ValueNumber is Number(v): blank is 0, non-numeric is NaN.
func ValueNumber(v any) float64 {
	switch x := v.(type) {
	case nil:
		return 0
	case float64:
		return x
	case bool:
		if x {
			return 1
		}
		return 0
	case string:
		s := JSTrim(x)
		if s == "" {
			return 0
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return math.NaN()
		}
		return f
	}
	return math.NaN()
}

func norm(v any) string { return strings.ToLower(JSTrim(ValueString(v))) }

func inList(list any, a string) bool {
	items, ok := list.([]any)
	if !ok {
		return false
	}
	for _, it := range items {
		if norm(it) == a {
			return true
		}
	}
	return false
}

// EvaluateCondition mirrors evaluateCondition in lib/crm/workflow.ts.
func EvaluateCondition(c Condition, record map[string]any, changes map[string]Change) bool {
	actual := record[c.Field]
	a, v := norm(actual), norm(c.Value)
	switch c.Op {
	case "eq":
		return a == v
	case "neq":
		return a != v
	case "in":
		return inList(c.Value, a)
	case "not_in":
		return !inList(c.Value, a)
	case "contains":
		return len(v) > 0 && strings.Contains(a, v)
	case "gt":
		return ValueNumber(actual) > ValueNumber(c.Value)
	case "gte":
		return ValueNumber(actual) >= ValueNumber(c.Value)
	case "lt":
		return ValueNumber(actual) < ValueNumber(c.Value)
	case "lte":
		return ValueNumber(actual) <= ValueNumber(c.Value)
	case "is_empty":
		return len(a) == 0
	case "not_empty":
		return len(a) > 0
	case "changed":
		_, ok := changes[c.Field]
		return ok
	case "changed_to":
		ch, ok := changes[c.Field]
		return ok && norm(ch.To) == v
	}
	return false
}

// EvaluateConditions is AND over every condition; none always matches.
func EvaluateConditions(conds []Condition, record map[string]any, changes map[string]Change) bool {
	for _, c := range conds {
		if !EvaluateCondition(c, record, changes) {
			return false
		}
	}
	return true
}

var (
	placeholder = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.]+)\s*\}\}`)
	spaceRuns   = regexp.MustCompile(`[ \t]{2,}`)
)

// RenderTemplate fills {{path.to.value}} placeholders from a nested
// context: unknown keys render empty, numbers use the id-ID format.
func RenderTemplate(tpl string, ctx map[string]any) string {
	out := placeholder.ReplaceAllStringFunc(tpl, func(m string) string {
		var value any = ctx
		for _, key := range strings.Split(placeholder.FindStringSubmatch(m)[1], ".") {
			obj, ok := value.(map[string]any)
			if !ok {
				return ""
			}
			value = obj[key]
		}
		if f, ok := value.(float64); ok {
			return FormatNumber(f, 3)
		}
		return ValueString(value)
	})
	return JSTrim(spaceRuns.ReplaceAllString(out, " "))
}

// TriggerForEvent maps an internal event type ("deal.stage_changed") to the
// rule object and trigger it fires.
func TriggerForEvent(eventType string) (object, trigger string, ok bool) {
	obj, evt, _ := strings.Cut(eventType, ".")
	evt, _, _ = strings.Cut(evt, ".")
	if !slices.Contains(WorkflowObjects, obj) {
		return "", "", false
	}
	switch evt {
	case "created", "updated", "stage_changed", "status_changed":
		return obj, evt, true
	case "done":
		return obj, "status_changed", true
	case "score_changed":
		return obj, "score_reached", true
	}
	return "", "", false
}

// TriggerConfigMatches applies the trigger-specific filters: the target
// stage or status, and score_reached only when the score crosses the bar.
func TriggerConfigMatches(trigger string, cfg, record, payload map[string]any) bool {
	switch trigger {
	case "stage_changed":
		if to, _ := cfg["to_stage"].(string); to != "" && ValueString(record["stage_code"]) != to {
			return false
		}
	case "status_changed":
		if to, _ := cfg["to_status"].(string); to != "" && ValueString(record["status"]) != to {
			return false
		}
	case "score_reached":
		bar, ok := cfg["score"].(float64)
		if !ok {
			return true
		}
		score := ValueNumber(record["score"])
		prev := math.Inf(-1)
		if p, ok := payload["previous_score"]; ok && p != nil {
			prev = ValueNumber(p)
		}
		if score < bar || prev >= bar {
			return false
		}
	}
	return true
}

// SplitAtWait splits actions around the first wait: the actions before it
// run now, the ones after it run (days * 24 + hours) hours later.
func SplitAtWait(actions []WorkflowAction) (now, later []WorkflowAction, delayHours float64) {
	for i, a := range actions {
		if a.Type == "wait" {
			return actions[:i], actions[i+1:], a.Days*24 + a.Hours
		}
	}
	return actions, nil, 0
}

// PickRoundRobin picks userIDs[previousRuns % len]; false when empty.
func PickRoundRobin(userIDs []string, previousRuns int) (string, bool) {
	if len(userIDs) == 0 {
		return "", false
	}
	return userIDs[previousRuns%len(userIDs)], true
}

// RunStatus is the workflow run status after its immediate actions.
func RunStatus(results, failed int, scheduled bool) string {
	switch {
	case failed == 0 && scheduled:
		return "scheduled"
	case failed == 0:
		return "success"
	case failed == results:
		return "failed"
	}
	return "partial"
}

// ValidateWorkflowRule checks a rule definition before it is stored.
func ValidateWorkflowRule(name, object, trigger string, conds []Condition, actions []WorkflowAction) map[string]string {
	errs := map[string]string{}
	if JSTrim(name) == "" {
		errs["name"] = "Nama aturan wajib diisi"
	}
	if !slices.Contains(WorkflowObjects, object) {
		errs["object"] = "Objek tidak valid"
	}
	if !slices.Contains(WorkflowTriggers, trigger) {
		errs["triggerType"] = "Pemicu tidak valid"
	}
	for _, c := range conds {
		if c.Field == "" || !slices.Contains(ConditionOps, c.Op) {
			errs["conditions"] = "Kondisi tidak valid"
		}
	}
	if len(actions) == 0 {
		errs["actions"] = "Minimal 1 aksi"
	}
	for _, a := range actions {
		switch {
		case !slices.Contains(WorkflowActions, a.Type):
			errs["actions"] = "Aksi tidak dikenal: " + a.Type
		case a.Type == "create_task" && JSTrim(a.Title) == "":
			errs["actions"] = "Judul tugas wajib diisi"
		case a.Type == "update_field" && !CanUpdateField(object, a.Field):
			errs["actions"] = "Field tidak boleh diubah otomatis: " + a.Field
		case a.Type == "wait" && a.Days*24+a.Hours <= 0:
			errs["actions"] = "Durasi tunggu harus lebih dari 0"
		}
	}
	return errs
}

// ScoringFields, ScoringOperators and ScoringEventTypes are the enums of
// scoringRuleSchema (chat_inbound replaces wa_inbound: any channel counts).
var (
	ScoringFields     = []string{"source", "org_type", "temperature", "status", "city", "pic_email", "pic_title", "account_type", "industry"}
	ScoringOperators  = []string{"eq", "neq", "in", "contains", "not_empty", "gt", "lt"}
	ScoringEventTypes = []string{"task_done", "chat_inbound", "deal_created", "quotation_sent"}
)

// ScoringRule is one active scoring rule.
type ScoringRule struct {
	ID, Name, Kind string
	Field          string
	Operator       string
	Value          any
	EventType      string
	WindowDays     *int
	MaxCount       int
	Points         int
	IsActive       bool
}

// ScoreItem is one entry of leads.score_breakdown.
type ScoreItem struct {
	RuleID string `json:"rule_id"`
	Name   string `json:"name"`
	Points int    `json:"points"`
	Count  *int   `json:"count,omitempty"`
}

// MatchFieldRule evaluates a field rule against the lead snapshot.
func MatchFieldRule(r ScoringRule, snapshot map[string]string) bool {
	if r.Field == "" || r.Operator == "" {
		return false
	}
	raw, present := snapshot[r.Field]
	actual := strings.ToLower(JSTrim(raw))
	switch r.Operator {
	case "not_empty":
		return len(actual) > 0
	case "eq":
		return actual == norm(r.Value)
	case "neq":
		return actual != norm(r.Value)
	case "in":
		return inList(r.Value, actual)
	case "contains":
		v := norm(r.Value)
		return len(v) > 0 && strings.Contains(actual, v)
	case "gt", "lt":
		var a any
		if present {
			a = raw
		}
		if r.Operator == "gt" {
			return ValueNumber(a) > ValueNumber(r.Value)
		}
		return ValueNumber(a) < ValueNumber(r.Value)
	}
	return false
}

// EventCountKey is "<event_type>" or "<event_type>:<sub-type>" when the
// rule value is a non-blank string.
func EventCountKey(eventType string, value any) string {
	if s, ok := value.(string); ok && JSTrim(s) != "" {
		return eventType + ":" + strings.ToLower(JSTrim(s))
	}
	return eventType
}

// ComputeLeadScore sums the points of matching field rules and of event
// rules times their (max_count capped) occurrences.
func ComputeLeadScore(rules []ScoringRule, snapshot map[string]string, counts map[string]int) (int, []ScoreItem) {
	breakdown := []ScoreItem{}
	for _, r := range rules {
		if r.Kind == "field" {
			if MatchFieldRule(r, snapshot) {
				breakdown = append(breakdown, ScoreItem{RuleID: r.ID, Name: r.Name, Points: r.Points})
			}
			continue
		}
		if r.EventType == "" {
			continue
		}
		count := min(counts[EventCountKey(r.EventType, r.Value)], max(1, r.MaxCount))
		if count > 0 {
			c := count
			breakdown = append(breakdown, ScoreItem{RuleID: r.ID, Name: r.Name, Points: r.Points * count, Count: &c})
		}
	}
	score := 0
	for _, b := range breakdown {
		score += b.Points
	}
	return score, breakdown
}

// MaxWindow is the widest window_days among the event rules of eventType,
// or nil when any of them has no window (count everything).
func MaxWindow(rules []ScoringRule, eventType string) *int {
	var widest *int
	for _, r := range rules {
		if r.Kind != "event" || r.EventType != eventType {
			continue
		}
		if r.WindowDays == nil || *r.WindowDays == 0 {
			return nil
		}
		if widest == nil || *r.WindowDays > *widest {
			w := *r.WindowDays
			widest = &w
		}
	}
	return widest
}

// Approver roles a rule can name (NüHabit used IAM roles; Mixist has
// permission keys and the ADMIN role).
const (
	ApproverPerm  = "sales.approve"
	ApproverAdmin = "ADMIN"
)

// CanDecideStep mirrors canDecideStep: the named user, or the role (ADMIN
// may stand in for any role, as super_admin did).
func CanDecideStep(approverRole, approverUserID *string, userID string, isAdmin, canApprove bool) bool {
	if approverUserID != nil && *approverUserID != "" {
		return *approverUserID == userID || isAdmin
	}
	if approverRole != nil && *approverRole != "" {
		if isAdmin {
			return true
		}
		return *approverRole == ApproverPerm && canApprove
	}
	return false
}

// ApprovalRule is one active approval rule.
type ApprovalRule struct {
	ID             string
	Name           string
	Level          int
	MinDiscount    float64
	ApproverRole   *string
	ApproverUserID *string
	IsActive       bool
}

// RequiredApprovalLevels mirrors requiredApprovalLevels: the rules whose
// bar the discount passes, one per level (the highest bar passed).
func RequiredApprovalLevels(discount float64, rules []ApprovalRule) []ApprovalRule {
	byLevel := map[int]ApprovalRule{}
	for _, r := range rules {
		if !(discount > r.MinDiscount) {
			continue
		}
		if cur, ok := byLevel[r.Level]; !ok || r.MinDiscount > cur.MinDiscount {
			byLevel[r.Level] = r
		}
	}
	out := make([]ApprovalRule, 0, len(byLevel))
	for _, r := range byLevel {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Level < out[j].Level })
	return out
}
