// Package domain holds the sales entities, value sets, repository ports and
// the pure rules ported from NüHabit (salesfunnel, crm/advance, crm/inbox,
// crm/publicforms) plus WHITE's agency automations.
package domain

import (
	"context"
	"encoding/json"
	"time"
)

// MarshalJSON writes money as a decimal string so workflow snapshots and
// templates see rupiah, never sen.
func (m Money) MarshalJSON() ([]byte, error) { return json.Marshal(m.String()) }

type Audit struct {
	CreatedBy *string   `json:"created_by"`
	UpdatedBy *string   `json:"updated_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Pipeline struct {
	ID        string
	Code      string
	Name      string
	IsDefault bool
	SortOrder int
	Stages    []Stage
}

type Stage struct {
	ID                 string
	PipelineID         string
	Code               string
	Name               string
	SortOrder          int
	Probability        int
	IsWon              bool
	IsLost             bool
	StuckThresholdDays *int32
	IsActive           bool
}

func (s Stage) Flags() StageFlags {
	return StageFlags{IsWon: s.IsWon, IsLost: s.IsLost, Probability: s.Probability}
}

type LostReason struct {
	ID        string
	Code      string
	Name      string
	SortOrder int
	IsActive  bool
}

type Account struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	AccountType string          `json:"account_type"`
	Industry    *string         `json:"industry"`
	Address     *string         `json:"address"`
	City        *string         `json:"city"`
	Phone       *string         `json:"phone"`
	Email       *string         `json:"email"`
	Website     *string         `json:"website"`
	Npwp        *string         `json:"npwp"`
	Notes       *string         `json:"notes"`
	OwnerUserID *string         `json:"owner_user_id"`
	Custom      json.RawMessage `json:"custom"`
	Audit
}

type Contact struct {
	ID        string          `json:"id"`
	AccountID *string         `json:"account_id"`
	Name      string          `json:"name"`
	Title     *string         `json:"title"`
	Phone     *string         `json:"phone"`
	Email     *string         `json:"email"`
	IsPrimary bool            `json:"is_primary"`
	Notes     *string         `json:"notes"`
	Custom    json.RawMessage `json:"custom"`
	Audit
}

type Lead struct {
	ID             string          `json:"id"`
	OrgName        string          `json:"org_name"`
	OrgType        string          `json:"org_type"`
	PicName        string          `json:"pic_name"`
	PicTitle       *string         `json:"pic_title"`
	PicPhone       *string         `json:"pic_phone"`
	PicEmail       *string         `json:"pic_email"`
	City           *string         `json:"city"`
	Source         string          `json:"source"`
	Temperature    string          `json:"temperature"`
	Status         string          `json:"status"`
	Notes          *string         `json:"notes"`
	OwnerUserID    *string         `json:"owner_user_id"`
	AccountID      *string         `json:"account_id"`
	ContactID      *string         `json:"contact_id"`
	Custom         json.RawMessage `json:"custom"`
	Score          int             `json:"score"`
	ScoreBreakdown []ScoreItem     `json:"score_breakdown"`
	ScoreUpdatedAt *time.Time      `json:"score_updated_at"`
	Attribution
	Audit
}

type LeadFilter struct {
	Query       string
	Status      string
	Temperature string
	Source      string
	OwnerUserID string
	From, To    *time.Time // created_at range, [From, To)
	ScoreMin    *int32
	ScoreMax    *int32
	Limit       int
	Offset      int
}

type Deal struct {
	ID                string          `json:"id"`
	LeadID            *string         `json:"lead_id"`
	AccountID         *string         `json:"account_id"`
	Title             string          `json:"title"`
	ServiceLines      []string        `json:"service_lines"`
	PipelineID        string          `json:"pipeline_id"`
	StageID           string          `json:"stage_id"`
	ValueEstimate     *Money          `json:"value_estimate"`
	ValueFinal        *Money          `json:"value_final"`
	OwnerUserID       *string         `json:"owner_user_id"`
	LostReasonID      *string         `json:"lost_reason_id"`
	ExpectedCloseDate *time.Time      `json:"expected_close_date"`
	Notes             *string         `json:"notes"`
	ForecastCategory  string          `json:"forecast_category"`
	EnteredStageAt    time.Time       `json:"entered_stage_at"`
	ClosedAt          *time.Time      `json:"closed_at"`
	Custom            json.RawMessage `json:"custom"`
	Audit
	// Joined for lists and workflow snapshots.
	StageCode   string `json:"stage_code"`
	StageName   string `json:"stage_name"`
	IsWon       bool   `json:"is_won"`
	IsLost      bool   `json:"is_lost"`
	Probability int    `json:"probability"`
	OrgName     string `json:"org_name"`
	PicName     string `json:"pic_name"`
	PicPhone    string `json:"pic_phone"`
	PicEmail    string `json:"pic_email"`
	AccountName string `json:"account_name"`
}

// ClientName is the account name, then the lead's organisation.
func (d Deal) ClientName() string {
	if d.AccountName != "" {
		return d.AccountName
	}
	if d.OrgName != "" {
		return d.OrgName
	}
	return d.Title
}

type DealFilter struct {
	PipelineID  string
	StageID     string
	OwnerUserID string
	LeadID      string
	AccountID   string
	Status      string // open | won | lost
	Query       string
	ServiceLine string
	Industry    string     // "Lainnya" matches deals without one
	Source      string     // lead source
	LostReason  string     // a reason id or "none"
	From, To    *time.Time // [From, To) on created_at, or closed_at when ByClosed
	ByClosed    bool
	Limit       int
	Offset      int
}

type DealMember struct {
	DealID       string
	UserID       string
	Role         string
	SplitPercent float64
}

type StageHistory struct {
	ID        string
	DealID    string
	StageID   string
	StageName string
	EnteredAt time.Time
	MovedBy   *string
}

type Activity struct {
	ID             string     `json:"id"`
	LeadID         *string    `json:"lead_id"`
	DealID         *string    `json:"deal_id"`
	SubjectType    *string    `json:"subject_type"`
	SubjectID      *string    `json:"subject_id"`
	ActivityType   string     `json:"activity_type"`
	Title          string     `json:"title"`
	Notes          *string    `json:"notes"`
	Priority       string     `json:"priority"`
	Status         string     `json:"status"`
	DueAt          *time.Time `json:"due_at"`
	DoneAt         *time.Time `json:"done_at"`
	ReminderAt     *time.Time `json:"reminder_at"`
	ReminderSentAt *time.Time `json:"reminder_sent_at"`
	Recurrence     []byte     `json:"-"`
	ParentTaskID   *string    `json:"parent_task_id"`
	OwnerUserID    *string    `json:"owner_user_id"`
	Audit
}

type ActivityFilter struct {
	LeadID      string
	DealID      string
	OwnerUserID string
	Status      string
	OpenOnly    bool
	DueBefore   *time.Time
	Limit       int
}

type Quotation struct {
	ID                string          `json:"id"`
	DealID            string          `json:"deal_id"`
	QuoteNumber       string          `json:"quote_number"`
	Version           int             `json:"version"`
	ParentQuotationID *string         `json:"parent_quotation_id"`
	Status            string          `json:"status"`
	UsePpn            bool            `json:"use_ppn"`
	PpnPercent        float64         `json:"ppn_percent"`
	Subtotal          Money           `json:"subtotal"`
	DiscountPercent   float64         `json:"discount_percent"`
	DiscountNominal   Money           `json:"discount_nominal"`
	PpnNominal        Money           `json:"ppn_nominal"`
	Total             Money           `json:"total"`
	Notes             *string         `json:"notes"`
	ValidUntil        *time.Time      `json:"valid_until"`
	ApprovalStatus    string          `json:"approval_status"`
	ApprovalRequestID *string         `json:"approval_request_id"`
	GeneratedBy       *string         `json:"generated_by"`
	SentAt            *time.Time      `json:"sent_at"`
	DecidedAt         *time.Time      `json:"decided_at"`
	SupersededAt      *time.Time      `json:"superseded_at"`
	ExpiryRemindedAt  *time.Time      `json:"expiry_reminded_at"`
	Items             []QuotationItem `json:"-"`
	Terms             []QuotationTerm `json:"-"`
	Audit
	// Joined.
	DealTitle   string `json:"deal_title"`
	ClientName  string `json:"client_name"`
	OwnerUserID string `json:"owner_user_id"`
}

type QuotationItem struct {
	ID          string
	QuotationID string
	ItemType    string
	CatalogID   *string
	Description string
	Scope       string
	Unit        string
	Qty         float64
	UnitPrice   Money
	LineTotal   Money
	SortOrder   int
}

type QuotationTerm struct {
	ID          string
	QuotationID string
	Label       string
	Percent     float64
	DueDate     *time.Time
	SortOrder   int
}

type QuotationFilter struct {
	DealID string
	Status string
	Limit  int
}

type ApprovalRequest struct {
	ID              string
	Object          string
	SubjectID       string
	Status          string
	CurrentLevel    int
	DiscountPercent float64
	Amount          Money
	Note            *string
	RequestedBy     *string
	ResolvedBy      *string
	ResolvedAt      *time.Time
	CreatedAt       time.Time
	Steps           []ApprovalStep
	// Joined.
	QuoteNumber string
	DealID      string
	DealTitle   string
	ClientName  string
}

type ApprovalStep struct {
	ID             string
	RequestID      string
	Level          int
	ApproverRole   *string
	ApproverUserID *string
	Status         string
	DecidedBy      *string
	DecidedAt      *time.Time
	Comment        *string
}

type Invoice struct {
	ID            string
	DealID        string
	QuotationID   *string
	TermID        *string
	InvoiceNumber string
	Label         string
	Amount        Money
	DueDate       *time.Time
	Status        string // diajukan | terkirim | batal
	SentAt        *time.Time
	IssuedBy      *string
	CancelledAt   *time.Time
	CancelledBy   *string
	CancelReason  *string
	Audit
	// Joined.
	DealTitle  string
	ClientName string
	PaidTotal  Money
}

type InvoiceFilter struct {
	DealID   string
	Status   string
	From, To *time.Time // issue time (sent, else created), [From, To)
	Limit    int
}

type Payment struct {
	ID        string
	DealID    string
	InvoiceID *string
	Amount    Money
	Method    string
	PaidOn    time.Time
	Note      *string
	CreatedBy *string
	CreatedAt time.Time
}

type CustomField struct {
	ID         string
	Object     string
	Key        string
	Label      string
	FieldType  string
	Options    []string
	IsRequired bool
	ShowInList bool
	SortOrder  int
	IsActive   bool
}

type CrmEvent struct {
	ID          string
	EventType   string
	SubjectType string
	SubjectID   string
	Payload     map[string]any
	ActorUserID *string
	CreatedAt   time.Time
}

type WorkflowRule struct {
	ID               string
	Name             string
	Object           string
	TriggerType      string
	TriggerConfig    map[string]any
	Conditions       []Condition
	Actions          []WorkflowAction
	RunOncePerRecord bool
	IsActive         bool
	RunCount         int
	LastRunAt        *time.Time
	CreatedBy        *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type WorkflowRun struct {
	ID            string
	RuleID        string
	EventID       *string
	SubjectType   string
	SubjectID     string
	Status        string
	ActionsResult []map[string]any
	Error         *string
	CreatedAt     time.Time
}

type ScheduledAction struct {
	ID          string
	RuleID      *string
	RunID       *string
	SubjectType string
	SubjectID   string
	Actions     []WorkflowAction
	Context     map[string]any
	RunAt       time.Time
	Status      string
	Attempts    int
}

type Form struct {
	ID              string
	Slug            string
	Name            string
	Description     string
	Fields          []FormField
	SubmitLabel     string
	SuccessMessage  string
	RedirectURL     *string
	DefaultSource   string
	LeadTemperature string
	NotifyUserIDs   []string
	IsActive        bool
	SubmissionCount int
	Audit
}

type FormSubmission struct {
	ID        string
	FormID    string
	LeadID    *string
	Payload   json.RawMessage
	Utm       json.RawMessage
	IPHash    string
	Status    string // accepted | spam | invalid
	Reason    *string
	CreatedAt time.Time
}

type ContractTemplate struct {
	ID       string
	Name     string
	Type     string
	Body     string
	IsActive bool
	Audit
}

type Contract struct {
	ID              string
	ContractNumber  string
	TemplateID      *string
	QuotationID     *string
	DealID          string
	Type            string
	Title           string
	ClientName      string
	Body            string
	Status          string
	StartDate       *time.Time
	EndDate         *time.Time
	SignToken       string
	SentAt          *time.Time
	SignedAt        *time.Time
	SignerName      *string
	SignerIP        *string
	SignerUserAgent *string
	SignaturePNG    []byte
	ActivatedAt     *time.Time
	EndedAt         *time.Time
	CancelledAt     *time.Time
	CancelReason    *string
	Audit
}

type ContractFilter struct {
	DealID string
	Status string
	Limit  int
}

type PitchDeck struct {
	ID        string
	DealID    *string
	LeadID    *string
	Title     string
	Version   int
	Options   DeckOptions
	Slides    []Slide
	File      []byte
	SizeBytes int64
	Model     string
	CreatedBy *string
	CreatedAt time.Time
}

type OnboardingTemplate struct {
	ID        string
	Name      string
	IsDefault bool
	Items     []OnboardingTemplateItem
	UpdatedBy *string
	UpdatedAt time.Time
}

type Onboarding struct {
	ID                string
	DealID            string
	TemplateID        *string
	ClientName        string
	Status            string
	PortalToken       string
	Brief             map[string]string
	BriefSubmittedAt  *time.Time
	MarketingClientID *string
	Progress          int
	CompletedAt       *time.Time
	CreatedBy         *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	Items             []OnboardingItem
	Assets            []OnboardingAsset
}

type OnboardingItem struct {
	ID           string
	OnboardingID string
	Key          string
	Kind         string
	Title        string
	Description  string
	OwnerUserID  *string
	DueDate      *time.Time
	Status       string
	DoneAt       *time.Time
	DoneBy       *string
	RemindedAt   *time.Time
	SortOrder    int
	// Joined for reminders.
	ClientName string
}

type OnboardingAsset struct {
	ID           string
	OnboardingID string
	FileName     string
	ContentType  string
	SizeBytes    int64
	StorageKey   string
	UploaderIP   *string
	CreatedAt    time.Time
}

type Conversation struct {
	ID                   string
	Channel              string // web | whatsapp
	ExternalID           string
	VisitorToken         *string
	LeadID               *string
	DisplayName          *string
	Phone                *string
	Email                *string
	Status               string
	Priority             string
	AssignedUserID       *string
	BotActive            bool
	HandoffReason        *string
	HandoffAt            *time.Time
	IsComplaint          bool
	UnreadCount          int
	LastMessageAt        *time.Time
	LastMessagePreview   *string
	AwaitingSince        *time.Time
	FirstResponseAt      *time.Time
	FirstResponseSeconds *int32
	SLAResponseBreached  bool
	ResolvedAt           *time.Time
	ResolutionSeconds    *int32
	PageURL              *string
	Utm                  json.RawMessage
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type ConversationFilter struct {
	LeadID         string
	Status         string
	Channel        string
	AssignedUserID string
	Query          string
	Limit          int
}

type Message struct {
	ID                string
	ConversationID    string
	Direction         string // in | out
	Sender            string // customer | bot | agent | system
	Body              string
	Status            string
	Provider          *string
	ProviderMessageID *string
	ErrorReason       *string
	SentByUserID      *string
	CreatedAt         time.Time
}

type ConversationNote struct {
	ID             string
	ConversationID string
	AuthorUserID   string
	Body           string
	CreatedAt      time.Time
}

type ReplyTemplate struct {
	ID        string
	Title     string
	Body      string
	IsActive  bool
	CreatedBy *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Store is every repository of the service plus transactions. A nil
// pointer result means "not found"; errors are infrastructure failures.
type Store interface {
	DashboardRepo
	WithTx(ctx context.Context, fn func(Store) error) error
	// NextNumber returns the next counter value for a document prefix.
	NextNumber(ctx context.Context, prefix string) (int, error)

	ListPipelines(ctx context.Context) ([]Pipeline, error)
	SavePipeline(ctx context.Context, p Pipeline) error
	GetStage(ctx context.Context, id string) (*Stage, error)
	SaveStage(ctx context.Context, s Stage) error
	ListLostReasons(ctx context.Context) ([]LostReason, error)

	GetLead(ctx context.Context, id string) (*Lead, error)
	ListLeads(ctx context.Context, f LeadFilter) ([]Lead, int, error)
	SaveLead(ctx context.Context, l *Lead) error
	DeleteLead(ctx context.Context, id string, at time.Time) error
	FindLeadByContact(ctx context.Context, phone, email string) (*Lead, error)
	LeadDedupKeys(ctx context.Context) (map[string]bool, error)
	UpdateLeadScore(ctx context.Context, id string, score int, breakdown []ScoreItem, at time.Time) error
	ListLeadIDs(ctx context.Context) ([]string, error)

	GetAccount(ctx context.Context, id string) (*Account, error)
	ListAccounts(ctx context.Context, query string, limit int) ([]Account, error)
	SaveAccount(ctx context.Context, a *Account) error
	DeleteAccount(ctx context.Context, id string, at time.Time) error
	GetContact(ctx context.Context, id string) (*Contact, error)
	ListContacts(ctx context.Context, accountID, query string, limit int) ([]Contact, error)
	SaveContact(ctx context.Context, c *Contact) error
	DeleteContact(ctx context.Context, id string, at time.Time) error

	GetDeal(ctx context.Context, id string) (*Deal, error)
	ListDeals(ctx context.Context, f DealFilter) ([]Deal, int, error)
	SaveDeal(ctx context.Context, d *Deal) error
	DeleteDeal(ctx context.Context, id string, at time.Time) error
	ListDealMembers(ctx context.Context, dealID string) ([]DealMember, error)
	ReplaceDealMembers(ctx context.Context, dealID string, members []DealMember) error
	AddStageHistory(ctx context.Context, h StageHistory) error
	ListStageHistory(ctx context.Context, dealID string) ([]StageHistory, error)
	ForecastDeals(ctx context.Context, from, to time.Time) ([]ForecastDeal, error)
	StaleDealIDs(ctx context.Context, before time.Time) ([]string, error)
	StaleLeadIDs(ctx context.Context, before time.Time) ([]string, error)

	GetActivity(ctx context.Context, id string) (*Activity, error)
	ListActivities(ctx context.Context, f ActivityFilter) ([]Activity, error)
	SaveActivity(ctx context.Context, a *Activity) error
	DeleteActivity(ctx context.Context, id string, at time.Time) error
	DueReminders(ctx context.Context, now time.Time, limit int) ([]Activity, error)
	ClaimReminder(ctx context.Context, id string, at time.Time) (bool, error)
	TasksDueBetween(ctx context.Context, from, to time.Time) ([]string, error)

	ListCatalog(ctx context.Context, activeOnly bool) ([]CatalogItem, error)
	GetCatalogItem(ctx context.Context, id string) (*CatalogItem, error)
	SaveCatalogItem(ctx context.Context, c CatalogItem) error
	ListRetainerDiscounts(ctx context.Context) ([]RetainerDiscount, error)
	ReplaceRetainerDiscounts(ctx context.Context, rules []RetainerDiscount) error

	GetQuotation(ctx context.Context, id string) (*Quotation, error)
	ListQuotations(ctx context.Context, f QuotationFilter) ([]Quotation, error)
	SaveQuotation(ctx context.Context, q *Quotation) error
	SetQuotationApproval(ctx context.Context, id, status string, requestID *string) error
	DeleteQuotation(ctx context.Context, id string, at time.Time) error
	LatestQuotation(ctx context.Context, dealID string) (*Quotation, error)
	ExpiringQuotations(ctx context.Context, until time.Time) ([]Quotation, error)
	MarkQuotationReminded(ctx context.Context, id string, at time.Time) error

	ListApprovalRules(ctx context.Context, activeOnly bool) ([]ApprovalRule, error)
	SaveApprovalRule(ctx context.Context, r ApprovalRule) error
	DeleteApprovalRule(ctx context.Context, id string) error
	CreateApprovalRequest(ctx context.Context, r ApprovalRequest) error
	GetApprovalRequest(ctx context.Context, id string, forUpdate bool) (*ApprovalRequest, error)
	ListApprovalRequests(ctx context.Context, status string, limit int) ([]ApprovalRequest, error)
	SaveApprovalRequest(ctx context.Context, r ApprovalRequest) error
	SaveApprovalStep(ctx context.Context, s ApprovalStep) error
	CancelPendingApprovals(ctx context.Context, subjectID string, at time.Time) error

	GetInvoice(ctx context.Context, id string) (*Invoice, error)
	ListInvoices(ctx context.Context, f InvoiceFilter) ([]Invoice, error)
	SaveInvoice(ctx context.Context, i *Invoice) error
	GetPayment(ctx context.Context, id string) (*Payment, error)
	ListPayments(ctx context.Context, dealID string) ([]Payment, error)
	SavePayment(ctx context.Context, p *Payment) error
	DeletePayment(ctx context.Context, id string, at time.Time) error
	PaidTotal(ctx context.Context, dealID string) (Money, error)
	InvoicePaidTotal(ctx context.Context, invoiceID string) (Money, error)

	ListTargets(ctx context.Context, month string) ([]Target, error)
	ReplaceTargets(ctx context.Context, month string, targets []Target, by string) error

	ListCustomFields(ctx context.Context, object string) ([]CustomField, error)
	SaveCustomField(ctx context.Context, f CustomField) error
	DeleteCustomField(ctx context.Context, id string) error

	InsertCrmEvent(ctx context.Context, e CrmEvent) error
	CountSubjectEvents(ctx context.Context, subjectIDs []string, since *time.Time) (map[string]int, error)
	ListWorkflowRules(ctx context.Context, object, trigger string, activeOnly bool) ([]WorkflowRule, error)
	GetWorkflowRule(ctx context.Context, id string) (*WorkflowRule, error)
	SaveWorkflowRule(ctx context.Context, r WorkflowRule) error
	DeleteWorkflowRule(ctx context.Context, id string) error
	MarkRuleRun(ctx context.Context, id string, at time.Time) error
	HasRuleRun(ctx context.Context, ruleID, subjectID string, since *time.Time) (bool, error)
	CountRuleRuns(ctx context.Context, ruleID string) (int, error)
	InsertWorkflowRun(ctx context.Context, r WorkflowRun) error
	UpdateWorkflowRun(ctx context.Context, r WorkflowRun) error
	ListWorkflowRuns(ctx context.Context, ruleID string, limit int) ([]WorkflowRun, error)
	InsertScheduledAction(ctx context.Context, a ScheduledAction) error
	DueScheduledActions(ctx context.Context, now time.Time, limit int) ([]ScheduledAction, error)
	FinishScheduledAction(ctx context.Context, id, status string, errText *string) error
	ListScoringRules(ctx context.Context, activeOnly bool) ([]ScoringRule, error)
	SaveScoringRule(ctx context.Context, r ScoringRule) error
	DeleteScoringRule(ctx context.Context, id string) error

	GetForm(ctx context.Context, id string) (*Form, error)
	GetFormBySlug(ctx context.Context, slug string) (*Form, error)
	ListForms(ctx context.Context) ([]Form, error)
	SaveForm(ctx context.Context, f *Form) error
	DeleteForm(ctx context.Context, id string) error
	InsertFormSubmission(ctx context.Context, s FormSubmission) error
	ListFormSubmissions(ctx context.Context, formID string, limit int) ([]FormSubmission, error)

	ListContractTemplates(ctx context.Context, activeOnly bool) ([]ContractTemplate, error)
	GetContractTemplate(ctx context.Context, id string) (*ContractTemplate, error)
	SaveContractTemplate(ctx context.Context, t *ContractTemplate) error
	DeleteContractTemplate(ctx context.Context, id string) error
	GetContract(ctx context.Context, id string) (*Contract, error)
	GetContractByToken(ctx context.Context, token string) (*Contract, error)
	ListContracts(ctx context.Context, f ContractFilter) ([]Contract, error)
	SaveContract(ctx context.Context, c *Contract) error

	ListCaseStudies(ctx context.Context, activeOnly bool) ([]CaseStudy, error)
	SaveCaseStudy(ctx context.Context, c CaseStudy) error
	DeleteCaseStudy(ctx context.Context, id string) error
	ListPitchDecks(ctx context.Context, dealID, leadID string) ([]PitchDeck, error)
	GetPitchDeck(ctx context.Context, id string) (*PitchDeck, error)
	InsertPitchDeck(ctx context.Context, d PitchDeck) error
	NextDeckVersion(ctx context.Context, dealID, leadID string) (int, error)

	ListOnboardingTemplates(ctx context.Context) ([]OnboardingTemplate, error)
	SaveOnboardingTemplate(ctx context.Context, t OnboardingTemplate) error
	GetOnboarding(ctx context.Context, id string) (*Onboarding, error)
	GetOnboardingByDeal(ctx context.Context, dealID string) (*Onboarding, error)
	GetOnboardingByToken(ctx context.Context, token string) (*Onboarding, error)
	ListOnboardings(ctx context.Context, status string) ([]Onboarding, error)
	CreateOnboarding(ctx context.Context, o Onboarding) (bool, error)
	SaveOnboarding(ctx context.Context, o *Onboarding) error
	GetOnboardingItem(ctx context.Context, id string) (*OnboardingItem, error)
	SaveOnboardingItem(ctx context.Context, it OnboardingItem) error
	InsertOnboardingAsset(ctx context.Context, a OnboardingAsset) error
	GetOnboardingAsset(ctx context.Context, id string) (*OnboardingAsset, error)
	DueOnboardingItems(ctx context.Context, until time.Time) ([]OnboardingItem, error)
	MarkOnboardingItemReminded(ctx context.Context, id string, at time.Time) error

	GetConversation(ctx context.Context, id string) (*Conversation, error)
	GetConversationByToken(ctx context.Context, token string) (*Conversation, error)
	GetConversationByExternal(ctx context.Context, channel, externalID string) (*Conversation, error)
	ListConversations(ctx context.Context, f ConversationFilter) ([]Conversation, error)
	SaveConversation(ctx context.Context, c *Conversation) error
	InsertMessage(ctx context.Context, m Message) (bool, error)
	ListMessages(ctx context.Context, conversationID string, after time.Time, limit int) ([]Message, error)
	RecentMessages(ctx context.Context, conversationID string, limit int) ([]Message, error)
	InsertNote(ctx context.Context, n ConversationNote) error
	ListNotes(ctx context.Context, conversationID string) ([]ConversationNote, error)
	ListReplyTemplates(ctx context.Context, activeOnly bool) ([]ReplyTemplate, error)
	SaveReplyTemplate(ctx context.Context, t ReplyTemplate) error
	DeleteReplyTemplate(ctx context.Context, id string) error
	GetInsight(ctx context.Context, conversationID string) (*Insight, error)
	SaveInsight(ctx context.Context, in Insight) error
	GetChatbotSettings(ctx context.Context) (*ChatbotSettings, error)
	SaveChatbotSettings(ctx context.Context, s ChatbotSettings) error
	MarkSLABreaches(ctx context.Context, before time.Time) ([]Conversation, error)
}

// ErrUnique reports a unique constraint violation.
type ErrUnique struct{ Constraint string }

func (e *ErrUnique) Error() string { return "unique violation: " + e.Constraint }
