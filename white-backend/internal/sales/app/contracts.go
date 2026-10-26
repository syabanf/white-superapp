package app

import (
	"context"
	"math"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Contracts is the contract automation: Markdown templates with
// placeholders, contracts generated from accepted quotations, a public
// signing link and the signed PDF.
type Contracts struct {
	*Deps
	onboarding *Onboarding
	limiter    *RateLimiter
}

func NewContracts(d *Deps, onboarding *Onboarding) *Contracts {
	return &Contracts{Deps: d, onboarding: onboarding, limiter: NewRateLimiter(d.Clock, 10*time.Minute, 20)}
}

// AgencyName is printed on documents.
const AgencyName = "PT WHITE Digital Kreatif"

var contractTypeLabels = map[string]string{"retainer": "Retainer", "project": "Proyek", "nda": "Kerahasiaan (NDA)"}

// SignPath is the public signing route for a token.
func SignPath(token string) string { return "/v1/public/sales/contracts/" + token }

func (c *Contracts) ListTemplates(ctx context.Context) ([]domain.ContractTemplate, error) {
	return c.Store.ListContractTemplates(ctx, false)
}

func (c *Contracts) SaveTemplate(ctx context.Context, actor Actor, t domain.ContractTemplate) (*domain.ContractTemplate, error) {
	errs := map[string]string{}
	if domain.JSTrim(t.Name) == "" {
		errs["name"] = "Nama template wajib diisi"
	}
	if !domain.Contains(domain.ContractTypes, t.Type) {
		errs["type"] = "Jenis kontrak tidak valid"
	}
	if domain.JSTrim(t.Body) == "" {
		errs["body"] = "Isi template wajib diisi"
	}
	if err := fieldErrs(errs); err != nil {
		return nil, err
	}
	now := c.now()
	if t.ID == "" {
		t.ID, t.CreatedBy, t.CreatedAt = ids.New(), actor.ref(), now
	} else {
		cur, err := c.Store.GetContractTemplate(ctx, t.ID)
		if err != nil {
			return nil, err
		}
		if cur == nil {
			return nil, notFound(msgTemplateNotFound)
		}
		t.CreatedBy, t.CreatedAt = cur.CreatedBy, cur.CreatedAt
	}
	t.UpdatedBy, t.UpdatedAt = actor.ref(), now
	if err := c.Store.SaveContractTemplate(ctx, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (c *Contracts) DeleteTemplate(ctx context.Context, id string) error {
	return c.Store.DeleteContractTemplate(ctx, id)
}

func (c *Contracts) List(ctx context.Context, f domain.ContractFilter) ([]domain.Contract, error) {
	return c.Store.ListContracts(ctx, f)
}

func (c *Contracts) load(ctx context.Context, id string) (*domain.Contract, error) {
	ct, err := c.Store.GetContract(ctx, id)
	if err != nil {
		return nil, err
	}
	if ct == nil {
		return nil, notFound(msgContractNotFound)
	}
	return ct, nil
}

func (c *Contracts) Get(ctx context.Context, id string) (*domain.Contract, error) {
	return c.load(ctx, id)
}

// GenerateInput steers Generate.
type GenerateInput struct {
	QuotationID, TemplateID, StartDate, EndDate, Title string
}

// defaultEnd is the contract length when none is given: the retainer
// months of the quotation, one month for a project, a year for an NDA.
func defaultEnd(kind string, start time.Time, q *domain.Quotation) time.Time {
	switch kind {
	case "nda":
		return start.AddDate(1, 0, -1)
	case "retainer":
		months := 0.0
		for _, it := range q.Items {
			if it.Unit == "bulan" {
				months = math.Max(months, it.Qty)
			}
		}
		if months < 1 {
			months = 3
		}
		return start.AddDate(0, int(months), -1)
	}
	return start.AddDate(0, 1, -1)
}

// Generate fills a template from an accepted quotation into a numbered
// draft contract (KTR-YYMM-####).
func (c *Contracts) Generate(ctx context.Context, actor Actor, in GenerateInput) (*domain.Contract, error) {
	q, err := c.Store.GetQuotation(ctx, in.QuotationID)
	if err != nil {
		return nil, err
	}
	if q == nil {
		return nil, notFound(msgQuoteNotFound)
	}
	if q.Status != domain.QuoteAccepted {
		return nil, conflict(msgQuoteNeedAccepted)
	}
	tpl, err := c.Store.GetContractTemplate(ctx, in.TemplateID)
	if err != nil {
		return nil, err
	}
	if tpl == nil || !tpl.IsActive {
		return nil, notFound(msgTemplateNotFound)
	}
	deal, err := c.Store.GetDeal(ctx, q.DealID)
	if err != nil {
		return nil, err
	}
	if deal == nil {
		return nil, notFound(msgDealNotFound)
	}
	start, err := domain.ParseDate(in.StartDate)
	if err != nil {
		return nil, apperr.Field("startDate", "Tanggal mulai tidak valid")
	}
	end, err := domain.ParseDate(in.EndDate)
	if err != nil {
		return nil, apperr.Field("endDate", "Tanggal selesai tidak valid")
	}
	now := c.now()
	if start == nil {
		d := domain.DueDate(now, 0)
		start = &d
	}
	if end == nil {
		e := defaultEnd(tpl.Type, *start, q)
		end = &e
	}
	if end.Before(*start) {
		return nil, apperr.Field("endDate", "Tanggal selesai harus setelah tanggal mulai")
	}
	cctx := domain.ContractContext{AgencyName: AgencyName, Today: now, ClientName: deal.ClientName(), PicName: deal.PicName, PicEmail: deal.PicEmail, PicPhone: deal.PicPhone, DealTitle: deal.Title, Quotation: q, StartDate: start, EndDate: end}
	if deal.AccountID != nil {
		if acc, err := c.Store.GetAccount(ctx, *deal.AccountID); err != nil {
			return nil, err
		} else if acc != nil {
			cctx.ClientAddress = deref(acc.Address)
			if city := deref(acc.City); city != "" && cctx.ClientAddress != "" {
				cctx.ClientAddress += ", " + city
			} else if city != "" {
				cctx.ClientAddress = city
			}
		}
	}
	if deal.LeadID != nil {
		if lead, err := c.Store.GetLead(ctx, *deal.LeadID); err == nil && lead != nil {
			cctx.PicTitle = deref(lead.PicTitle)
		}
	}
	ct := &domain.Contract{ID: ids.New(), TemplateID: &tpl.ID, QuotationID: &q.ID, DealID: deal.ID, Type: tpl.Type, ClientName: deal.ClientName(), Status: domain.ContractDraft, StartDate: start, EndDate: end, SignToken: NewToken(),
		Audit: domain.Audit{CreatedBy: actor.ref(), UpdatedBy: actor.ref(), CreatedAt: now, UpdatedAt: now}}
	ct.Title = domain.JSTrim(in.Title)
	if ct.Title == "" {
		ct.Title = "Perjanjian " + contractTypeLabels[tpl.Type] + " " + ct.ClientName
	}
	err = c.Store.WithTx(ctx, func(st domain.Store) error {
		prefix := domain.DocumentPrefix(domain.NumberContract, now)
		n, err := st.NextNumber(ctx, prefix)
		if err != nil {
			return err
		}
		ct.ContractNumber = domain.DocumentNumber(prefix, n)
		cctx.ContractNumber = ct.ContractNumber
		ct.Body = domain.RenderContract(tpl.Body, domain.ContractValues(cctx))
		return st.SaveContract(ctx, ct)
	})
	if err != nil {
		return nil, err
	}
	return ct, nil
}

type ContractUpdate struct {
	ID, Title, Body, StartDate, EndDate string
}

func (c *Contracts) Update(ctx context.Context, actor Actor, in ContractUpdate) (*domain.Contract, error) {
	ct, err := c.load(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if ct.Status != domain.ContractDraft {
		return nil, conflict("Hanya kontrak draft yang bisa diubah")
	}
	if t := domain.JSTrim(in.Title); t != "" {
		ct.Title = t
	}
	if b := domain.JSTrim(in.Body); b != "" {
		ct.Body = in.Body
	}
	if in.StartDate != "" {
		if ct.StartDate, err = domain.ParseDate(in.StartDate); err != nil {
			return nil, apperr.Field("startDate", "Tanggal mulai tidak valid")
		}
	}
	if in.EndDate != "" {
		if ct.EndDate, err = domain.ParseDate(in.EndDate); err != nil {
			return nil, apperr.Field("endDate", "Tanggal selesai tidak valid")
		}
	}
	ct.UpdatedBy, ct.UpdatedAt = actor.ref(), c.now()
	return ct, c.Store.SaveContract(ctx, ct)
}

// Send moves draft -> sent; the signing link becomes valid.
func (c *Contracts) Send(ctx context.Context, actor Actor, id string) (*domain.Contract, error) {
	ct, err := c.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if !domain.ContractTransitionAllowed(ct.Status, domain.ContractSent) {
		return nil, conflict(msgStatusTransition)
	}
	now := c.now()
	ct.Status, ct.SentAt, ct.UpdatedBy, ct.UpdatedAt = domain.ContractSent, &now, actor.ref(), now
	return ct, c.Store.SaveContract(ctx, ct)
}

// SetStatus handles signed -> active -> ended and cancellation.
func (c *Contracts) SetStatus(ctx context.Context, actor Actor, id, status, reason string) (*domain.Contract, error) {
	ct, err := c.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if status != domain.ContractActive && status != domain.ContractEnded && status != domain.ContractCancelled {
		return nil, apperr.Field("status", "Status tidak valid")
	}
	if !domain.ContractTransitionAllowed(ct.Status, status) {
		return nil, conflict(msgStatusTransition)
	}
	now := c.now()
	switch status {
	case domain.ContractActive:
		ct.ActivatedAt = &now
	case domain.ContractEnded:
		ct.EndedAt = &now
	case domain.ContractCancelled:
		if domain.JSTrim(reason) == "" {
			return nil, apperr.Field("reason", "Alasan pembatalan wajib diisi")
		}
		ct.CancelledAt, ct.CancelReason = &now, domain.TrimmedOrNil(reason)
	}
	ct.Status, ct.UpdatedBy, ct.UpdatedAt = status, actor.ref(), now
	return ct, c.Store.SaveContract(ctx, ct)
}

// byToken returns a contract the client may see: sent or later, never a
// draft or a cancelled one.
func (c *Contracts) byToken(ctx context.Context, token string) (*domain.Contract, error) {
	if len(token) < 32 {
		return nil, notFound(msgLinkInvalid)
	}
	ct, err := c.Store.GetContractByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if ct == nil || ct.Status == domain.ContractDraft || ct.Status == domain.ContractCancelled {
		return nil, notFound(msgLinkInvalid)
	}
	return ct, nil
}

func (c *Contracts) PublicGet(ctx context.Context, token, ip string) (*domain.Contract, error) {
	if err := c.limiter.Check("contract-view:" + ip); err != nil {
		return nil, err
	}
	return c.byToken(ctx, token)
}

// SignInput is the client's signature.
type SignInput struct {
	Token, SignerName, SignaturePNG string
	Agree                           bool
	IP, UserAgent                   string
}

// Sign stores the typed name, the drawn signature (PNG), IP, user agent
// and time; the deal owner is notified and onboarding moves on.
func (c *Contracts) Sign(ctx context.Context, in SignInput) (*domain.Contract, error) {
	if err := c.limiter.Check("contract-sign:" + in.IP); err != nil {
		return nil, err
	}
	ct, err := c.byToken(ctx, in.Token)
	if err != nil {
		return nil, err
	}
	if ct.Status != domain.ContractSent {
		return nil, conflict("Kontrak sudah ditandatangani")
	}
	errs := map[string]string{}
	name := domain.JSTrim(in.SignerName)
	if len([]rune(name)) < 3 {
		errs["signerName"] = "Ketik nama lengkap Anda"
	}
	if !in.Agree {
		errs["agree"] = "Centang persetujuan untuk menandatangani"
	}
	png, sigErr := domain.DecodeSignature(in.SignaturePNG)
	if sigErr != nil {
		errs["signaturePng"] = sigErr.Error()
	}
	if err := fieldErrs(errs); err != nil {
		return nil, err
	}
	now := c.now()
	ct.Status, ct.SignedAt, ct.SignerName, ct.SignaturePNG = domain.ContractSigned, &now, &name, png
	ct.SignerIP, ct.SignerUserAgent, ct.UpdatedAt = domain.TrimmedOrNil(in.IP), domain.TrimmedOrNil(domain.SliceUTF16(in.UserAgent, 400)), now
	if err := c.Store.SaveContract(ctx, ct); err != nil {
		return nil, err
	}
	recipients := []string{}
	if ct.CreatedBy != nil {
		recipients = append(recipients, *ct.CreatedBy)
	}
	if deal, err := c.Store.GetDeal(ctx, ct.DealID); err == nil && deal != nil && deal.OwnerUserID != nil && *deal.OwnerUserID != deref(ct.CreatedBy) {
		recipients = append(recipients, *deal.OwnerUserID)
	}
	c.notify(ctx, Notification{UserIDs: recipients, Type: NotifContractSigned, Title: "Kontrak " + ct.ContractNumber + " ditandatangani", Body: ct.ClientName + " · ditandatangani oleh " + name, Href: "/erp/sales/contracts/" + ct.ID})
	c.onboarding.OnContractSigned(ctx, ct.DealID)
	return ct, nil
}

// RenderPDF renders by id (internal) or token (public download).
func (c *Contracts) RenderPDF(ctx context.Context, id, token string) (string, []byte, error) {
	var ct *domain.Contract
	var err error
	if token != "" {
		ct, err = c.byToken(ctx, token)
	} else {
		ct, err = c.load(ctx, id)
	}
	if err != nil {
		return "", nil, err
	}
	data, err := c.Renderer.ContractPDF(*ct)
	if err != nil {
		return "", nil, err
	}
	return fileName(ct.ContractNumber, ct.ClientName, "pdf"), data, nil
}
