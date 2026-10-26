package app

import (
	"context"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Onboarding is the client onboarding started when a deal is won: a
// checklist with owners and due dates, the client portal (brief and asset
// uploads) and the step that creates the client in the marketing platform.
type Onboarding struct {
	*Deps
	limiter *RateLimiter
}

func NewOnboarding(d *Deps) *Onboarding {
	return &Onboarding{Deps: d, limiter: NewRateLimiter(d.Clock, 10*time.Minute, 30)}
}

// PortalPath is the public portal route for a token.
func PortalPath(token string) string { return "/v1/public/sales/onboarding/" + token }

func (o *Onboarding) template(ctx context.Context) (*string, []domain.OnboardingTemplateItem, error) {
	tpls, err := o.Store.ListOnboardingTemplates(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, t := range tpls {
		if t.IsDefault && len(t.Items) > 0 {
			id := t.ID
			return &id, t.Items, nil
		}
	}
	return nil, domain.DefaultOnboardingItems, nil
}

// Start creates the onboarding of a won deal; idempotent.
func (o *Onboarding) Start(ctx context.Context, actor Actor, dealID string) (*domain.Onboarding, error) {
	if cur, err := o.Store.GetOnboardingByDeal(ctx, dealID); err != nil || cur != nil {
		return cur, err
	}
	deal, err := o.Store.GetDeal(ctx, dealID)
	if err != nil {
		return nil, err
	}
	if deal == nil {
		return nil, notFound(msgDealNotFound)
	}
	if !deal.IsWon {
		return nil, conflict("Onboarding hanya untuk deal yang sudah menang")
	}
	tplID, items, err := o.template(ctx)
	if err != nil {
		return nil, err
	}
	now := o.now()
	ob := domain.Onboarding{ID: ids.New(), DealID: deal.ID, TemplateID: tplID, ClientName: deal.ClientName(), Status: domain.OnboardingActive, PortalToken: NewToken(), Brief: map[string]string{}, CreatedBy: actor.ref(), CreatedAt: now, UpdatedAt: now}
	for i, it := range items {
		due := domain.DueDate(now, it.DueDays)
		ob.Items = append(ob.Items, domain.OnboardingItem{ID: ids.New(), OnboardingID: ob.ID, Key: it.Key, Kind: domain.ItemKind(it.Key, it.Kind), Title: it.Title, Description: it.Description, OwnerUserID: deal.OwnerUserID, DueDate: &due, Status: domain.ItemTodo, SortOrder: i})
	}
	created, err := o.Store.CreateOnboarding(ctx, ob)
	if err != nil {
		return nil, err
	}
	if created {
		if deal.OwnerUserID != nil {
			o.notify(ctx, Notification{UserIDs: []string{*deal.OwnerUserID}, Type: NotifOnboarding, Title: "Onboarding " + ob.ClientName + " dimulai", Body: "Checklist onboarding dibuat. Bagikan tautan portal ke klien untuk mengisi brief dan mengunggah aset.", Href: "/erp/sales/onboarding/" + ob.ID})
		}
		o.OnContractSigned(ctx, dealID)
		o.OnPayment(ctx, dealID)
	}
	return o.Store.GetOnboardingByDeal(ctx, dealID)
}

func (o *Onboarding) List(ctx context.Context, status string) ([]domain.Onboarding, error) {
	return o.Store.ListOnboardings(ctx, status)
}

func (o *Onboarding) Get(ctx context.Context, id string) (*domain.Onboarding, error) {
	ob, err := o.Store.GetOnboarding(ctx, id)
	if err != nil {
		return nil, err
	}
	if ob == nil {
		return nil, notFound(msgOnboardNotFound)
	}
	return ob, nil
}

// refresh recomputes progress and closes the onboarding at 100%.
func (o *Onboarding) refresh(ctx context.Context, id string) (*domain.Onboarding, error) {
	ob, err := o.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	statuses := make([]string, len(ob.Items))
	for i, it := range ob.Items {
		statuses[i] = it.Status
	}
	now := o.now()
	ob.Progress, ob.UpdatedAt = domain.OnboardingProgress(statuses), now
	switch {
	case ob.Progress == 100 && ob.Status == domain.OnboardingActive:
		ob.Status, ob.CompletedAt = domain.OnboardingDone, &now
	case ob.Progress < 100 && ob.Status == domain.OnboardingDone:
		ob.Status, ob.CompletedAt = domain.OnboardingActive, nil
	}
	if err := o.Store.SaveOnboarding(ctx, ob); err != nil {
		return nil, err
	}
	return ob, nil
}

// ItemUpdate edits one checklist item.
type ItemUpdate struct {
	ID          string
	Status      string
	OwnerUserID *string
	DueDate     *string
}

func (o *Onboarding) UpdateItem(ctx context.Context, actor Actor, in ItemUpdate) (*domain.Onboarding, error) {
	it, err := o.Store.GetOnboardingItem(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if it == nil {
		return nil, notFound("Item checklist tidak ditemukan")
	}
	if in.Status != "" {
		if !domain.Contains([]string{domain.ItemTodo, domain.ItemDone, domain.ItemSkipped}, in.Status) {
			return nil, apperr.Field("status", "Status tidak valid")
		}
		if in.Status != it.Status {
			it.Status = in.Status
			if in.Status == domain.ItemTodo {
				it.DoneAt, it.DoneBy = nil, nil
			} else {
				now := o.now()
				it.DoneAt, it.DoneBy = &now, actor.ref()
			}
		}
	}
	if in.OwnerUserID != nil {
		it.OwnerUserID = domain.TrimmedOrNil(*in.OwnerUserID)
	}
	if in.DueDate != nil {
		due, err := domain.ParseDate(*in.DueDate)
		if err != nil {
			return nil, apperr.Field("dueDate", "Tanggal tidak valid")
		}
		it.DueDate, it.RemindedAt = due, nil
	}
	if err := o.Store.SaveOnboardingItem(ctx, *it); err != nil {
		return nil, err
	}
	return o.refresh(ctx, it.OnboardingID)
}

// complete marks the open items of a kind done (automation hooks).
func (o *Onboarding) complete(ctx context.Context, ob *domain.Onboarding, kind string, by *string) error {
	changed := false
	now := o.now()
	for _, it := range ob.Items {
		if it.Kind != kind || it.Status != domain.ItemTodo {
			continue
		}
		it.Status, it.DoneAt, it.DoneBy = domain.ItemDone, &now, by
		if err := o.Store.SaveOnboardingItem(ctx, it); err != nil {
			return err
		}
		changed = true
	}
	if changed {
		_, err := o.refresh(ctx, ob.ID)
		return err
	}
	return nil
}

// hook completes a kind for the deal's onboarding, best effort.
func (o *Onboarding) hook(ctx context.Context, dealID, kind string, ready func(*domain.Onboarding) (bool, error)) {
	ob, err := o.Store.GetOnboardingByDeal(ctx, dealID)
	if err != nil || ob == nil || ob.Status != domain.OnboardingActive {
		return
	}
	ok, err := ready(ob)
	if err == nil && ok {
		err = o.complete(ctx, ob, kind, nil)
	}
	if err != nil {
		o.warn("onboarding hook failed", "kind", kind, "err", err)
	}
}

// OnContractSigned completes "sign contract" once any contract of the
// deal is signed or active.
func (o *Onboarding) OnContractSigned(ctx context.Context, dealID string) {
	o.hook(ctx, dealID, domain.ItemContract, func(*domain.Onboarding) (bool, error) {
		cs, err := o.Store.ListContracts(ctx, domain.ContractFilter{DealID: dealID, Limit: 50})
		for _, c := range cs {
			if c.Status == domain.ContractSigned || c.Status == domain.ContractActive || c.Status == domain.ContractEnded {
				return true, err
			}
		}
		return false, err
	})
}

// OnPayment completes "first invoice paid" once payments cover the
// earliest active invoice (or any payment when nothing was invoiced).
func (o *Onboarding) OnPayment(ctx context.Context, dealID string) {
	o.hook(ctx, dealID, domain.ItemInvoicePaid, func(*domain.Onboarding) (bool, error) {
		paid, err := o.Store.PaidTotal(ctx, dealID)
		if err != nil || paid <= 0 {
			return false, err
		}
		invs, err := o.Store.ListInvoices(ctx, domain.InvoiceFilter{DealID: dealID, Limit: 500})
		if err != nil {
			return false, err
		}
		var first *domain.Invoice
		for i := range invs {
			inv := &invs[i]
			if inv.Status != InvoiceCancelled && (first == nil || inv.CreatedAt.Before(first.CreatedAt)) {
				first = inv
			}
		}
		return first == nil || paid >= first.Amount, nil
	})
}

// CreateMarketingClient registers the client in the marketing platform
// through identity (the caller must be ADMIN) and completes that step.
func (o *Onboarding) CreateMarketingClient(ctx context.Context, actor Actor, id string, in ClientInput) (*domain.Onboarding, error) {
	ob, err := o.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if ob.MarketingClientID != nil {
		return nil, conflict("Klien sudah dibuat di platform marketing")
	}
	if domain.JSTrim(in.Name) == "" {
		in.Name = ob.ClientName
	}
	if in.Slug == "" {
		in.Slug = slugify(in.Name)
	}
	if in.WebsiteURL == "" {
		in.WebsiteURL = ob.Brief["website"]
	}
	clientID, err := o.Clients.CreateClient(ctx, in)
	if err != nil {
		return nil, err
	}
	ob.MarketingClientID, ob.UpdatedAt = &clientID, o.now()
	if err := o.Store.SaveOnboarding(ctx, ob); err != nil {
		return nil, err
	}
	if err := o.complete(ctx, ob, domain.ItemCreateClient, actor.ref()); err != nil {
		return nil, err
	}
	return o.Get(ctx, id)
}

func slugify(s string) string {
	out := make([]rune, 0, len(s))
	dash := false
	for _, r := range []rune(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out, dash = append(out, r), false
		case r >= 'A' && r <= 'Z':
			out, dash = append(out, r+32), false
		case !dash && len(out) > 0:
			out, dash = append(out, '-'), true
		}
	}
	for len(out) > 0 && out[len(out)-1] == '-' {
		out = out[:len(out)-1]
	}
	if len(out) > 48 {
		out = out[:48]
	}
	return string(out)
}

func (o *Onboarding) ListTemplates(ctx context.Context) ([]domain.OnboardingTemplate, error) {
	return o.Store.ListOnboardingTemplates(ctx)
}

func (o *Onboarding) SaveTemplate(ctx context.Context, actor Actor, t domain.OnboardingTemplate) (*domain.OnboardingTemplate, error) {
	if domain.JSTrim(t.Name) == "" {
		return nil, apperr.Field("name", "Nama template wajib diisi")
	}
	if len(t.Items) == 0 {
		return nil, apperr.Field("items", "Template minimal punya 1 item")
	}
	seen := map[string]bool{}
	for i, it := range t.Items {
		if !domain.FieldKeyRe.MatchString(it.Key) || seen[it.Key] || domain.JSTrim(it.Title) == "" || it.DueDays < 0 {
			return nil, apperr.Field("items", "Item checklist tidak valid")
		}
		seen[it.Key] = true
		if it.Kind == "" {
			t.Items[i].Kind = domain.ItemKind(it.Key, "")
		}
	}
	if t.ID == "" {
		t.ID = ids.New()
	}
	t.UpdatedBy = actor.ref()
	return &t, o.Store.SaveOnboardingTemplate(ctx, t)
}

// Asset returns an uploaded asset's bytes.
func (o *Onboarding) Asset(ctx context.Context, id string) (*domain.OnboardingAsset, []byte, error) {
	a, err := o.Store.GetOnboardingAsset(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if a == nil {
		return nil, nil, notFound("Aset tidak ditemukan")
	}
	data, err := o.Files.Get(ctx, a.StorageKey)
	if err != nil {
		return nil, nil, err
	}
	return a, data, nil
}

// ─── public portal ───────────────────────────────────────────

func (o *Onboarding) byToken(ctx context.Context, token, ip string) (*domain.Onboarding, error) {
	if err := o.limiter.Check("onboarding:" + ip); err != nil {
		return nil, err
	}
	if len(token) < 32 {
		return nil, notFound(msgLinkInvalid)
	}
	ob, err := o.Store.GetOnboardingByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if ob == nil || ob.Status == domain.OnboardingCancelled {
		return nil, notFound(msgLinkInvalid)
	}
	return ob, nil
}

func (o *Onboarding) PublicGet(ctx context.Context, token, ip string) (*domain.Onboarding, error) {
	return o.byToken(ctx, token, ip)
}

func (o *Onboarding) ownerOf(ctx context.Context, ob *domain.Onboarding) []string {
	deal, err := o.Store.GetDeal(ctx, ob.DealID)
	if err != nil || deal == nil || deal.OwnerUserID == nil {
		return nil
	}
	return []string{*deal.OwnerUserID}
}

// SubmitBrief stores the client brief from the portal.
func (o *Onboarding) SubmitBrief(ctx context.Context, token, ip string, brief map[string]string) (*domain.Onboarding, error) {
	ob, err := o.byToken(ctx, token, ip)
	if err != nil {
		return nil, err
	}
	clean, errs := domain.CleanBrief(brief)
	if err := fieldErrs(errs); err != nil {
		return nil, err
	}
	now := o.now()
	ob.Brief, ob.BriefSubmittedAt, ob.UpdatedAt = clean, &now, now
	if err := o.Store.SaveOnboarding(ctx, ob); err != nil {
		return nil, err
	}
	if err := o.complete(ctx, ob, domain.ItemBrief, nil); err != nil {
		return nil, err
	}
	o.notify(ctx, Notification{UserIDs: o.ownerOf(ctx, ob), Type: NotifOnboarding, Title: ob.ClientName + " mengisi brief", Body: "Brief onboarding sudah dikirim klien.", Href: "/erp/sales/onboarding/" + ob.ID})
	return o.Store.GetOnboarding(ctx, ob.ID)
}

// UploadAsset stores a client asset after checking its magic bytes.
func (o *Onboarding) UploadAsset(ctx context.Context, token, ip, name string, data []byte) (*domain.Onboarding, error) {
	ob, err := o.byToken(ctx, token, ip)
	if err != nil {
		return nil, err
	}
	if len(ob.Assets) >= 50 {
		return nil, invalid("Batas 50 file per onboarding tercapai")
	}
	ct, ext, err := domain.SniffAsset(data, name)
	if err != nil {
		return nil, invalid(err.Error())
	}
	a := domain.OnboardingAsset{ID: ids.New(), OnboardingID: ob.ID, FileName: domain.SafeFileName(name), ContentType: ct, SizeBytes: int64(len(data)), UploaderIP: domain.TrimmedOrNil(ip), CreatedAt: o.now()}
	a.StorageKey = "onboarding/" + ob.ID + "/" + a.ID + "." + ext
	if err := o.Files.Put(ctx, a.StorageKey, data); err != nil {
		return nil, err
	}
	if err := o.Store.InsertOnboardingAsset(ctx, a); err != nil {
		return nil, err
	}
	if err := o.complete(ctx, ob, domain.ItemAssets, nil); err != nil {
		return nil, err
	}
	o.notify(ctx, Notification{UserIDs: o.ownerOf(ctx, ob), Type: NotifOnboarding, Title: ob.ClientName + " mengunggah aset", Body: a.FileName, Href: "/erp/sales/onboarding/" + ob.ID})
	return o.Store.GetOnboarding(ctx, ob.ID)
}

// SendReminders notifies owners of overdue or due-today checklist items
// (ERP daily tick), at most once per day per item.
func (o *Onboarding) SendReminders(ctx context.Context) (int, error) {
	items, err := o.Store.DueOnboardingItems(ctx, domain.DueDate(o.now(), 0))
	if err != nil {
		return 0, err
	}
	sent := 0
	for _, it := range items {
		if it.OwnerUserID == nil {
			continue
		}
		o.notify(ctx, Notification{UserIDs: []string{*it.OwnerUserID}, Type: NotifOnboarding, Title: "Onboarding " + it.ClientName + ": " + it.Title, Body: "Jatuh tempo " + domain.FormatDateID(*it.DueDate), Href: "/erp/sales/onboarding/" + it.OnboardingID})
		if err := o.Store.MarkOnboardingItemReminded(ctx, it.ID, o.now()); err != nil {
			return sent, err
		}
		sent++
	}
	return sent, nil
}
