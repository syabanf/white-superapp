// Package app holds the sales use cases: leads and accounts, the pipeline,
// quotations with approvals, billing, insights, the workflow engine, public
// forms, contracts, pitch decks, client onboarding and the lead chatbot.
package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/mail"
)

// Notification types sent through identity.
const (
	NotifLeadNew        = "SALES_LEAD_NEW"
	NotifTaskReminder   = "SALES_TASK_REMINDER"
	NotifApproval       = "SALES_APPROVAL"
	NotifQuoteExpiry    = "SALES_QUOTATION_EXPIRY"
	NotifContractSigned = "SALES_CONTRACT_SIGNED"
	NotifOnboarding     = "SALES_ONBOARDING"
	NotifChatHandoff    = "SALES_CHAT_HANDOFF"
	NotifWorkflow       = "SALES_WORKFLOW"
	NotifInvoice        = "SALES_INVOICE"
)

// Notification is one in-app alert.
type Notification struct {
	UserIDs []string
	Type    string
	Title   string
	Body    string
	Href    string
}

// Notifier delivers in-app notifications (identity NotificationService).
// WhatsApp and email delivery plug in behind the same port later.
type Notifier interface {
	Notify(ctx context.Context, n Notification) error
}

// User is a workspace login as identity lists it. Admin is filled only
// by Approvers.
type User struct {
	ID, Name, Email string
	Admin           bool
}

// Directory resolves users through identity: ListUsers for owner pickers
// (any signed-in caller), Approvers for the users holding sales.approve or
// ADMIN.
type Directory interface {
	ListUsers(ctx context.Context) ([]User, error)
	Approvers(ctx context.Context) ([]User, error)
}

// ClientInput creates a client in the marketing platform.
type ClientInput struct {
	Name, Slug, WebsiteURL, Industry string
}

// TaxInfo is a finance VAT code as a quotation uses it.
type TaxInfo struct {
	Code, Name       string
	EffectivePercent float64
}

// TaxCodes reads the finance tax master. Nil leaves quotations on a typed rate.
type TaxCodes interface {
	VATByCode(ctx context.Context, code string) (*TaxInfo, error)
	DefaultSaleVAT(ctx context.Context) (*TaxInfo, error)
}

// MarketingClients creates clients through identity ClientService.
type MarketingClients interface {
	CreateClient(ctx context.Context, in ClientInput) (string, error)
}

// AIMessage is one chat completion message.
type AIMessage struct {
	Role    string
	Content string
}

// AIRequest is one completion.
type AIRequest struct {
	Messages    []AIMessage
	JSON        bool
	Temperature float64
	MaxTokens   int
}

// AIResult carries the content and the model id. Mock is true when no
// OpenRouter key is configured: callers then use their deterministic mock.
type AIResult struct {
	Content string
	Model   string
	Mock    bool
}

// MockModel is the model id stored for deterministic output.
const MockModel = "mock"

// AI is OpenRouter through integrations credentials.
type AI interface {
	Complete(ctx context.Context, req AIRequest) (AIResult, error)
}

// WhatsApp is the WA gateway port (NüHabit services/wa-gateway HTTP
// contract). Disabled unless WA_GATEWAY_URL and WA_GATEWAY_TOKEN are set.
type WhatsApp interface {
	Enabled() bool
	VerifyToken(token string) bool
	Send(ctx context.Context, phone, text string) (messageID string, err error)
}

// Files stores uploaded assets (local disk today).
type Files interface {
	Put(ctx context.Context, key string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

// Company is the letterhead and payment instruction Finance keeps as
// master data. NPWP is left off the letterhead when empty.
type Company struct {
	LegalName, Address, Email, Phone, NPWP string
	// BankLine tells clients where to transfer.
	BankLine string
}

// CompanyProfile reads the company master data; it never fails, falling
// back to the deployed defaults.
type CompanyProfile interface {
	Company(ctx context.Context) Company
}

// Renderer turns documents into bytes on the given letterhead.
type Renderer interface {
	// ownerName is the deal owner's display name ("" when unknown).
	QuotationPDF(co Company, q domain.Quotation, d domain.Deal, ownerName string) ([]byte, error)
	// quoteNumber is the source quotation's number ("" when none).
	InvoicePDF(co Company, i domain.Invoice, d domain.Deal, quoteNumber string) ([]byte, error)
	ContractPDF(co Company, c domain.Contract) ([]byte, error)
	PitchDeck(title string, slides []domain.Slide) ([]byte, error)
}

// Deps are the collaborators shared by every use case.
type Deps struct {
	Store     domain.Store
	Clock     clock.Clock
	Events    events.Publisher
	Notifier  Notifier
	Directory Directory
	Clients   MarketingClients
	AI        AI
	WhatsApp  WhatsApp
	Files     Files
	Taxes     TaxCodes
	Renderer  Renderer
	Logger    *slog.Logger
	// Mail sends client email; AppURL builds the links inside it.
	Mail   mail.Sender
	AppURL string
	// Share unlocks the client portal; Holders finds who checks proofs.
	Share   ShareGate
	Holders PermissionHolders
	Profile CompanyProfile
}

func (d *Deps) now() time.Time { return d.Clock.Now() }

func (d *Deps) company(ctx context.Context) Company {
	if d.Profile == nil {
		return Company{}
	}
	return d.Profile.Company(ctx)
}

// notify is best effort: a failed notification never fails the action.
func (d *Deps) notify(ctx context.Context, n Notification) {
	if d.Notifier == nil || len(n.UserIDs) == 0 {
		return
	}
	if err := d.Notifier.Notify(ctx, n); err != nil && d.Logger != nil {
		d.Logger.Warn("notify failed", "type", n.Type, "err", err)
	}
}

func (d *Deps) publish(ctx context.Context, subject string, payload any) error {
	if d.Events == nil {
		return nil
	}
	return d.Events.Publish(ctx, subject, payload)
}

func (d *Deps) warn(msg string, args ...any) {
	if d.Logger != nil {
		d.Logger.Warn(msg, args...)
	}
}

// Shared user-facing messages.
const (
	msgLeadNotFound      = "Lead tidak ditemukan"
	msgDealNotFound      = "Deal tidak ditemukan"
	msgAccountNotFound   = "Akun tidak ditemukan"
	msgContactNotFound   = "Kontak tidak ditemukan"
	msgActivityNotFound  = "Aktivitas tidak ditemukan"
	msgQuoteNotFound     = "Quotation tidak ditemukan"
	msgStageNotFound     = "Tahap tidak ditemukan"
	msgInvoiceNotFound   = "Invoice tidak ditemukan"
	msgContractNotFound  = "Kontrak tidak ditemukan"
	msgOnboardNotFound   = "Onboarding tidak ditemukan"
	msgConvNotFound      = "Percakapan tidak ditemukan"
	msgFormNotFound      = "Form tidak ditemukan"
	msgDeckNotFound      = "Pitch deck tidak ditemukan"
	msgTemplateNotFound  = "Template tidak ditemukan"
	msgRuleNotFound      = "Aturan tidak ditemukan"
	msgTooManyRequests   = "Terlalu banyak permintaan. Coba lagi beberapa menit."
	msgLinkInvalid       = "Tautan tidak valid atau sudah tidak berlaku."
	msgQuoteFrozen       = "Quotation yang sudah dikirim atau diputuskan tidak bisa diubah. Buat revisi."
	msgApprovalRequired  = "Diskon menunggu persetujuan. Quotation belum boleh dikirim."
	msgStatusTransition  = "Perubahan status tidak diizinkan"
	msgQuoteNeedAccepted = "Quotation harus berstatus diterima"
)

func notFound(msg string) error { return apperr.New(apperr.NotFound, msg) }

func invalid(msg string) error { return apperr.New(apperr.Validation, msg) }

func conflict(msg string) error { return apperr.New(apperr.Conflict, msg) }

// fieldErrs turns a field -> message map into a validation error.
func fieldErrs(errs map[string]string) error {
	if len(errs) == 0 {
		return nil
	}
	fields := map[string][]string{}
	first := ""
	for _, k := range sortedKeys(errs) {
		fields[k] = []string{errs[k]}
		if first == "" {
			first = errs[k]
		}
	}
	return apperr.New(apperr.Validation, first).Fields(fields)
}

func ptr[T any](v T) *T { return &v }

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
