package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Quotes covers the service catalog, quotations with versioning, the
// discount approval chain, PDF rendering and the generated drafts.
type Quotes struct {
	*Deps
	auto *Automation
}

func NewQuotes(d *Deps, auto *Automation) *Quotes { return &Quotes{Deps: d, auto: auto} }

// ─── catalog (ADMIN) ─────────────────────────────────────────

func (q *Quotes) ListCatalog(ctx context.Context, activeOnly bool) ([]domain.CatalogItem, error) {
	return q.Store.ListCatalog(ctx, activeOnly)
}

func (q *Quotes) SaveCatalogItem(ctx context.Context, c domain.CatalogItem) (*domain.CatalogItem, error) {
	errs := map[string]string{}
	if domain.JSTrim(c.Name) == "" {
		errs["name"] = "Nama layanan wajib diisi"
	}
	if !domain.FormSlugRe.MatchString(c.Code) {
		errs["code"] = "Kode hanya huruf kecil, angka, dan tanda hubung"
	}
	if !domain.Contains(domain.ServiceLines, c.ServiceLine) {
		errs["service_line"] = "Service line tidak valid"
	}
	if c.DefaultPrice < 0 {
		errs["default_price"] = "Harga tidak boleh negatif"
	}
	seen := map[string]bool{}
	for _, t := range c.Tiers {
		if !domain.Contains(domain.PackageTiers, t.Tier) || seen[t.Tier] || t.Price < 0 {
			errs["tiers"] = "Paket tidak valid"
		}
		seen[t.Tier] = true
	}
	if err := fieldErrs(errs); err != nil {
		return nil, err
	}
	if c.ID == "" {
		c.ID = ids.New()
	}
	if c.Unit == "" {
		c.Unit = "paket"
	}
	err := q.Store.WithTx(ctx, func(st domain.Store) error { return st.SaveCatalogItem(ctx, c) })
	if err != nil {
		var u *domain.ErrUnique
		if asUnique(err, &u) {
			return nil, conflict("Kode layanan sudah dipakai")
		}
		return nil, err
	}
	return q.Store.GetCatalogItem(ctx, c.ID)
}

func (q *Quotes) ListRetainerDiscounts(ctx context.Context) ([]domain.RetainerDiscount, error) {
	return q.Store.ListRetainerDiscounts(ctx)
}

func (q *Quotes) SaveRetainerDiscounts(ctx context.Context, rules []domain.RetainerDiscount) ([]domain.RetainerDiscount, error) {
	seen := map[int]bool{}
	for _, r := range rules {
		if r.MinMonths < 1 || r.MinMonths > 36 || seen[r.MinMonths] || r.DiscountPercent < 0 || r.DiscountPercent > 100 {
			return nil, invalid("Aturan diskon retainer tidak valid")
		}
		seen[r.MinMonths] = true
	}
	if err := q.Store.WithTx(ctx, func(st domain.Store) error { return st.ReplaceRetainerDiscounts(ctx, rules) }); err != nil {
		return nil, err
	}
	return q.Store.ListRetainerDiscounts(ctx)
}

// ─── quotations ──────────────────────────────────────────────

type QuotationItemInput struct {
	ItemType, CatalogID, Description, Scope, Unit string
	Qty                                           float64
	UnitPrice                                     string
}

type QuotationTermInput struct {
	Label   string
	Percent float64
	DueDate string
}

type QuotationInput struct {
	ID, DealID                  string
	UsePpn                      bool
	PpnPercent, DiscountPercent float64
	Notes, ValidUntil           string
	Items                       []QuotationItemInput
	Terms                       []QuotationTermInput
}

func (q *Quotes) payload(ctx context.Context, in QuotationInput) (domain.QuotationPayload, error) {
	p := domain.QuotationPayload{UsePpn: in.UsePpn, PpnPercent: in.PpnPercent, DiscountPercent: in.DiscountPercent, Notes: domain.TrimmedOrNil(in.Notes)}
	if p.UsePpn && p.PpnPercent == 0 {
		p.PpnPercent = 11
	}
	vu, err := domain.ParseDate(in.ValidUntil)
	if err != nil {
		return p, apperr.Field("validUntil", "Tanggal berlaku tidak valid")
	}
	p.ValidUntil = vu
	for _, it := range in.Items {
		price, err := domain.ParseMoney(it.UnitPrice)
		if err != nil {
			return p, apperr.Field("items", "Harga item tidak valid")
		}
		itemType := it.ItemType
		if itemType == "" {
			itemType = "bebas"
		}
		if itemType != "katalog" && itemType != "bebas" {
			return p, apperr.Field("items", "Jenis item tidak valid")
		}
		line := domain.QuotationLine{ItemType: itemType, CatalogID: domain.TrimmedOrNil(it.CatalogID), Description: domain.SliceUTF16(domain.JSTrim(it.Description), 300), Scope: domain.SliceUTF16(domain.JSTrim(it.Scope), 2000), Unit: domain.JSTrim(it.Unit), Qty: it.Qty, UnitPrice: price}
		if itemType == "katalog" && line.CatalogID != nil {
			c, err := q.Store.GetCatalogItem(ctx, *line.CatalogID)
			if err != nil {
				return p, err
			}
			if c == nil {
				return p, apperr.Field("items", "Layanan katalog tidak ditemukan")
			}
			if line.Description == "" {
				line.Description = c.Name
			}
			if line.Unit == "" {
				line.Unit = c.Unit
			}
		}
		if line.Unit == "" {
			line.Unit = "paket"
		}
		p.Items = append(p.Items, line)
	}
	for _, t := range in.Terms {
		due, err := domain.ParseDate(t.DueDate)
		if err != nil {
			return p, apperr.Field("terms", "Tanggal termin tidak valid")
		}
		p.Terms = append(p.Terms, domain.QuotationTermInput{Label: domain.JSTrim(t.Label), Percent: t.Percent, DueDate: due})
	}
	return p, fieldErrs(domain.ValidateQuotation(p))
}

// applyPayload writes totals, items and terms onto the quotation.
func applyPayload(qt *domain.Quotation, p domain.QuotationPayload) {
	t := domain.ComputeTotals(&p)
	qt.UsePpn, qt.PpnPercent, qt.DiscountPercent, qt.Notes, qt.ValidUntil = p.UsePpn, p.PpnPercent, p.DiscountPercent, p.Notes, p.ValidUntil
	if !p.UsePpn {
		qt.PpnPercent = 0
	}
	qt.Subtotal, qt.DiscountNominal, qt.PpnNominal, qt.Total = t.Subtotal, t.DiscountNominal, t.PpnNominal, t.Total
	qt.Items = make([]domain.QuotationItem, len(p.Items))
	for i, it := range p.Items {
		qt.Items[i] = domain.QuotationItem{ItemType: it.ItemType, CatalogID: it.CatalogID, Description: it.Description, Scope: it.Scope, Unit: it.Unit, Qty: it.Qty, UnitPrice: it.UnitPrice, LineTotal: it.LineTotal}
	}
	qt.Terms = make([]domain.QuotationTerm, len(p.Terms))
	for i, t := range p.Terms {
		qt.Terms[i] = domain.QuotationTerm{Label: t.Label, Percent: t.Percent, DueDate: t.DueDate}
	}
}

func (q *Quotes) number(ctx context.Context, st domain.Store, kind string) (string, error) {
	prefix := domain.DocumentPrefix(kind, q.now())
	n, err := st.NextNumber(ctx, prefix)
	if err != nil {
		return "", err
	}
	return domain.DocumentNumber(prefix, n), nil
}

// CreateQuotation is createQuotation: a numbered draft, the deal estimate
// follows the total, and the discount approval chain starts when a rule
// requires it.
func (q *Quotes) CreateQuotation(ctx context.Context, actor Actor, in QuotationInput) (*domain.Quotation, error) {
	return q.create(ctx, actor, in, nil)
}

func (q *Quotes) create(ctx context.Context, actor Actor, in QuotationInput, generatedBy *string) (*domain.Quotation, error) {
	deal, err := q.Store.GetDeal(ctx, in.DealID)
	if err != nil {
		return nil, err
	}
	if deal == nil {
		return nil, notFound(msgDealNotFound)
	}
	p, err := q.payload(ctx, in)
	if err != nil {
		return nil, err
	}
	now := q.now()
	qt := &domain.Quotation{ID: ids.New(), DealID: deal.ID, Version: 1, Status: domain.QuoteDraft, ApprovalStatus: domain.ApprovalNone, GeneratedBy: generatedBy, Audit: domain.Audit{CreatedBy: actor.ref(), UpdatedBy: actor.ref(), CreatedAt: now, UpdatedAt: now}}
	applyPayload(qt, p)
	var approvers *pendingApproval
	err = q.Store.WithTx(ctx, func(st domain.Store) error {
		if qt.QuoteNumber, err = q.number(ctx, st, domain.NumberQuotation); err != nil {
			return err
		}
		if err := st.SaveQuotation(ctx, qt); err != nil {
			return err
		}
		if approvers, err = q.requestApproval(ctx, st, qt, actor); err != nil {
			return err
		}
		deal.ValueEstimate = &qt.Total
		deal.UpdatedAt = now
		return st.SaveDeal(ctx, deal)
	})
	if err != nil {
		return nil, err
	}
	q.notifyApprovers(ctx, approvers)
	q.auto.Emit(ctx, Event{Type: "quotation.created", SubjectType: "quotation", SubjectID: qt.ID, Actor: actor.ref()})
	return q.Store.GetQuotation(ctx, qt.ID)
}

func (q *Quotes) load(ctx context.Context, id string) (*domain.Quotation, error) {
	qt, err := q.Store.GetQuotation(ctx, id)
	if err != nil {
		return nil, err
	}
	if qt == nil {
		return nil, notFound(msgQuoteNotFound)
	}
	return qt, nil
}

func (q *Quotes) GetQuotation(ctx context.Context, id string) (*domain.Quotation, error) {
	return q.load(ctx, id)
}

func (q *Quotes) ListQuotations(ctx context.Context, f domain.QuotationFilter) ([]domain.Quotation, error) {
	return q.Store.ListQuotations(ctx, f)
}

// UpdateQuotation edits a draft; a changed discount restarts approval.
func (q *Quotes) UpdateQuotation(ctx context.Context, actor Actor, in QuotationInput) (*domain.Quotation, error) {
	qt, err := q.load(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if qt.Status != domain.QuoteDraft {
		return nil, conflict(msgQuoteFrozen)
	}
	in.DealID = qt.DealID
	p, err := q.payload(ctx, in)
	if err != nil {
		return nil, err
	}
	prevDiscount := qt.DiscountPercent
	applyPayload(qt, p)
	now := q.now()
	qt.UpdatedBy, qt.UpdatedAt = actor.ref(), now
	var approvers *pendingApproval
	err = q.Store.WithTx(ctx, func(st domain.Store) error {
		if prevDiscount != qt.DiscountPercent || qt.ApprovalStatus == domain.ApprovalRejected {
			if err := st.CancelPendingApprovals(ctx, qt.ID, now); err != nil {
				return err
			}
			qt.ApprovalStatus, qt.ApprovalRequestID = domain.ApprovalNone, nil
			if err := st.SaveQuotation(ctx, qt); err != nil {
				return err
			}
			if approvers, err = q.requestApproval(ctx, st, qt, actor); err != nil {
				return err
			}
			return nil
		}
		return st.SaveQuotation(ctx, qt)
	})
	if err != nil {
		return nil, err
	}
	q.notifyApprovers(ctx, approvers)
	return q.Store.GetQuotation(ctx, qt.ID)
}

// SetStatus moves draft -> terkirim (release gate) and terkirim ->
// diterima | ditolak.
func (q *Quotes) SetStatus(ctx context.Context, actor Actor, id, status string) (*domain.Quotation, error) {
	qt, err := q.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if !domain.QuoteTransitionAllowed(qt.Status, status) {
		return nil, conflict(msgStatusTransition)
	}
	if status == domain.QuoteSent && !domain.CanRelease(qt.ApprovalStatus) {
		return nil, conflict(msgApprovalRequired)
	}
	if status == domain.QuoteAccepted && !domain.CanRelease(qt.ApprovalStatus) {
		return nil, conflict(msgApprovalRequired)
	}
	now := q.now()
	from := qt.Status
	qt.Status = status
	switch status {
	case domain.QuoteSent:
		qt.SentAt = &now
	default:
		qt.DecidedAt = &now
	}
	qt.UpdatedBy, qt.UpdatedAt = actor.ref(), now
	if err := q.Store.SaveQuotation(ctx, qt); err != nil {
		return nil, err
	}
	q.auto.Emit(ctx, Event{Type: "quotation.status_changed", SubjectType: "quotation", SubjectID: qt.ID, Actor: actor.ref(), Changes: map[string]domain.Change{"status": {From: from, To: status}}})
	if status == domain.QuoteSent {
		if lead, err := q.auto.affectedLead(ctx, "quotation", qt.ID); err == nil {
			q.auto.Signal(ctx, lead, SignalQuotationSent, "")
		}
	}
	return q.Store.GetQuotation(ctx, qt.ID)
}

// Revise copies items and terms into version + 1 (same parent) as a new
// draft; the old version becomes superseded.
func (q *Quotes) Revise(ctx context.Context, actor Actor, id string) (*domain.Quotation, error) {
	old, err := q.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if old.Status == domain.QuoteSuperseded {
		return nil, conflict("Quotation ini sudah direvisi")
	}
	if old.Status == domain.QuoteAccepted {
		return nil, conflict("Quotation yang sudah diterima tidak bisa direvisi")
	}
	now := q.now()
	parent := old.ID
	if old.ParentQuotationID != nil {
		parent = *old.ParentQuotationID
	}
	base := old.QuoteNumber
	if i := strings.Index(base, "-R"); i > 0 {
		base = base[:i]
	}
	next := *old
	next.ID, next.Version, next.ParentQuotationID, next.Status = ids.New(), old.Version+1, &parent, domain.QuoteDraft
	next.QuoteNumber = fmt.Sprintf("%s-R%d", base, old.Version)
	next.ApprovalStatus, next.ApprovalRequestID = domain.ApprovalNone, nil
	next.SentAt, next.DecidedAt, next.SupersededAt, next.ExpiryRemindedAt = nil, nil, nil, nil
	next.Audit = domain.Audit{CreatedBy: actor.ref(), UpdatedBy: actor.ref(), CreatedAt: now, UpdatedAt: now}
	next.Items = append([]domain.QuotationItem(nil), old.Items...)
	next.Terms = append([]domain.QuotationTerm(nil), old.Terms...)
	for i := range next.Items {
		next.Items[i].ID = ""
	}
	old.Status, old.SupersededAt, old.UpdatedAt = domain.QuoteSuperseded, &now, now
	var approvers *pendingApproval
	err = q.Store.WithTx(ctx, func(st domain.Store) error {
		if err := st.CancelPendingApprovals(ctx, old.ID, now); err != nil {
			return err
		}
		if err := st.SaveQuotation(ctx, old); err != nil {
			return err
		}
		if err := st.SaveQuotation(ctx, &next); err != nil {
			return err
		}
		approvers, err = q.requestApproval(ctx, st, &next, actor)
		return err
	})
	if err != nil {
		return nil, err
	}
	q.notifyApprovers(ctx, approvers)
	q.auto.Emit(ctx, Event{Type: "quotation.created", SubjectType: "quotation", SubjectID: next.ID, Actor: actor.ref()})
	return q.Store.GetQuotation(ctx, next.ID)
}

func (q *Quotes) DeleteQuotation(ctx context.Context, id string) error {
	qt, err := q.load(ctx, id)
	if err != nil {
		return err
	}
	if qt.Status == domain.QuoteAccepted {
		return conflict("Quotation yang sudah diterima tidak bisa dihapus")
	}
	return q.Store.WithTx(ctx, func(st domain.Store) error {
		if err := st.CancelPendingApprovals(ctx, id, q.now()); err != nil {
			return err
		}
		return st.DeleteQuotation(ctx, id, q.now())
	})
}

// RenderPDF renders the quotation with the Mixist letterhead.
func (q *Quotes) RenderPDF(ctx context.Context, id string) (string, []byte, error) {
	qt, err := q.load(ctx, id)
	if err != nil {
		return "", nil, err
	}
	deal, err := q.Store.GetDeal(ctx, qt.DealID)
	if err != nil {
		return "", nil, err
	}
	if deal == nil {
		return "", nil, notFound(msgDealNotFound)
	}
	data, err := q.Renderer.QuotationPDF(*qt, *deal)
	if err != nil {
		return "", nil, err
	}
	return fileName(qt.QuoteNumber, qt.ClientName, "pdf"), data, nil
}

// fileName is "{number}-{org}.{ext}" with a safe organisation slug.
func fileName(number, org, ext string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(org) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case b.Len() > 0 && !strings.HasSuffix(b.String(), "-"):
			b.WriteByte('-')
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return number + "." + ext
	}
	return number + "-" + domain.SliceUTF16(slug, 40) + "." + ext
}

// ─── quotation automation ────────────────────────────────────

// DraftOptions are the knobs of GenerateDraft.
type DraftOptions struct {
	Tier           string
	RetainerMonths int
	UseAI          bool
	UsePpn         *bool
	StartDate      string
}

// DraftResult is the generated quotation and what the catalog missed.
type DraftResult struct {
	Quotation *domain.Quotation
	Unpriced  []string
	Model     string
}

// GenerateDraft builds a draft from the deal's service lines, the catalog,
// package tiers and retainer discounts; with UseAI OpenRouter (or the
// mock) writes each item's scope from the deal notes.
func (q *Quotes) GenerateDraft(ctx context.Context, actor Actor, dealID string, opts DraftOptions) (*DraftResult, error) {
	deal, err := q.Store.GetDeal(ctx, dealID)
	if err != nil {
		return nil, err
	}
	if deal == nil {
		return nil, notFound(msgDealNotFound)
	}
	if len(deal.ServiceLines) == 0 {
		return nil, invalid("Deal belum punya layanan. Pilih layanan di deal terlebih dahulu.")
	}
	catalog, err := q.Store.ListCatalog(ctx, true)
	if err != nil {
		return nil, err
	}
	discounts, err := q.Store.ListRetainerDiscounts(ctx)
	if err != nil {
		return nil, err
	}
	start, err := domain.ParseDate(opts.StartDate)
	if err != nil {
		return nil, apperr.Field("startDate", "Tanggal mulai tidak valid")
	}
	usePpn := opts.UsePpn == nil || *opts.UsePpn
	do := domain.DraftOptions{Tier: opts.Tier, RetainerMonths: opts.RetainerMonths, UsePpn: usePpn, PpnPercent: 11}
	if start != nil {
		do.StartDate = *start
	}
	do = do.Normalize(q.now())
	p, missing := domain.BuildQuotationDraft(deal.ServiceLines, catalog, discounts, do, q.now())
	model := ""
	generatedBy := "template"
	if opts.UseAI && deal.Notes != nil && domain.JSTrim(*deal.Notes) != "" {
		model, err = q.writeScopes(ctx, deal, &p)
		if err != nil {
			return nil, err
		}
		generatedBy = "ai"
	}
	in := QuotationInput{DealID: deal.ID, UsePpn: p.UsePpn, PpnPercent: p.PpnPercent, DiscountPercent: p.DiscountPercent, Notes: deref(p.Notes), ValidUntil: domain.FormatDay(p.ValidUntil)}
	for _, it := range p.Items {
		in.Items = append(in.Items, QuotationItemInput{ItemType: it.ItemType, CatalogID: deref(it.CatalogID), Description: it.Description, Scope: it.Scope, Unit: it.Unit, Qty: it.Qty, UnitPrice: it.UnitPrice.String()})
	}
	for _, t := range p.Terms {
		in.Terms = append(in.Terms, QuotationTermInput{Label: t.Label, Percent: t.Percent, DueDate: domain.FormatDay(t.DueDate)})
	}
	qt, err := q.create(ctx, actor, in, &generatedBy)
	if err != nil {
		return nil, err
	}
	if missing == nil {
		missing = []string{}
	}
	return &DraftResult{Quotation: qt, Unpriced: missing, Model: model}, nil
}

type scopeAnswer struct {
	Items []struct {
		Index int    `json:"index"`
		Scope string `json:"scope"`
	} `json:"items"`
}

const scopeSystemPrompt = `Kamu adalah account manager di Mixist, digital marketing agency di Indonesia.
Tulis deskripsi ruang lingkup (scope of work) untuk setiap item penawaran berdasarkan catatan kebutuhan klien.
Aturan: Bahasa Indonesia, 2-4 kalimat per item, konkret (deliverable, frekuensi, laporan), tanpa harga, tanpa janji hasil angka.
Jawab HANYA dengan JSON tanpa pagar kode: {"items": [{"index": number, "scope": string}]}`

// writeScopes asks the model for scope text per item; the mock appends the
// client's main need to the catalog description. Returns the model id.
func (q *Quotes) writeScopes(ctx context.Context, deal *domain.Deal, p *domain.QuotationPayload) (string, error) {
	var b strings.Builder
	b.WriteString("Klien: " + deal.ClientName() + "\nCatatan kebutuhan:\n" + deref(deal.Notes) + "\n\nItem:\n")
	for i, it := range p.Items {
		fmt.Fprintf(&b, "%d. %s (%s, qty %s %s). Deskripsi katalog: %s\n", i, it.Description, it.ItemType, domain.FormatNumber(it.Qty, 3), it.Unit, strings.ReplaceAll(it.Scope, "\n", " "))
	}
	res, err := q.AI.Complete(ctx, AIRequest{Messages: []AIMessage{{Role: "system", Content: scopeSystemPrompt}, {Role: "user", Content: b.String()}}, JSON: true, Temperature: 0.3, MaxTokens: 1500})
	if err != nil {
		return "", err
	}
	if res.Mock {
		need := firstSentence(deref(deal.Notes))
		for i := range p.Items {
			p.Items[i].Scope = strings.TrimSpace(p.Items[i].Scope + "\nDisesuaikan dengan kebutuhan klien: " + need + ".")
		}
		return MockModel, nil
	}
	var ans scopeAnswer
	if err := decodeJSONAnswer(res.Content, &ans); err == nil {
		for _, a := range ans.Items {
			if a.Index >= 0 && a.Index < len(p.Items) && domain.JSTrim(a.Scope) != "" {
				p.Items[a.Index].Scope = domain.SliceUTF16(domain.JSTrim(a.Scope), 2000)
			}
		}
	}
	return res.Model, nil
}

func firstSentence(s string) string {
	s = domain.JSTrim(s)
	if i := strings.IndexAny(s, ".\n"); i > 0 {
		s = s[:i]
	}
	return domain.SliceUTF16(s, 160)
}
