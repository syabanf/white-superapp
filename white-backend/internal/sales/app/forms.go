package app

import (
	"context"
	"encoding/json"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Forms is the web-to-lead forms of crm/marketing (admin side) and
// crm/publicforms (the public endpoint).
type Forms struct {
	*Deps
	auto    *Automation
	limiter *RateLimiter
}

func NewForms(d *Deps, auto *Automation) *Forms {
	return &Forms{Deps: d, auto: auto, limiter: NewRateLimiter(d.Clock, 10*time.Minute, 10)}
}

func (f *Forms) ListForms(ctx context.Context) ([]domain.Form, error) { return f.Store.ListForms(ctx) }

func (f *Forms) SaveForm(ctx context.Context, actor Actor, in domain.Form) (*domain.Form, error) {
	errs := map[string]string{}
	if domain.JSTrim(in.Name) == "" {
		errs["name"] = "Nama form wajib diisi"
	}
	if !domain.FormSlugRe.MatchString(in.Slug) {
		errs["slug"] = "Slug hanya huruf kecil, angka, dan tanda hubung"
	}
	if msg := domain.ValidateFormFields(in.Fields); msg != "" {
		errs["fields"] = msg
	}
	if in.DefaultSource == "" {
		in.DefaultSource = "form"
	} else if !domain.Contains(domain.LeadSources, in.DefaultSource) {
		errs["defaultSource"] = "Sumber lead tidak valid"
	}
	if in.LeadTemperature == "" {
		in.LeadTemperature = "hangat"
	} else if !domain.Contains(domain.LeadTemperatures, in.LeadTemperature) {
		errs["leadTemperature"] = "Suhu lead tidak valid"
	}
	if err := fieldErrs(errs); err != nil {
		return nil, err
	}
	now := f.now()
	if in.SubmitLabel == "" {
		in.SubmitLabel = "Kirim"
	}
	if in.SuccessMessage == "" {
		in.SuccessMessage = "Terima kasih, tim kami akan segera menghubungi Anda."
	}
	if in.ID == "" {
		in.ID, in.CreatedBy, in.CreatedAt = ids.New(), actor.ref(), now
	} else {
		cur, err := f.Store.GetForm(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		if cur == nil {
			return nil, notFound(msgFormNotFound)
		}
		in.CreatedBy, in.CreatedAt = cur.CreatedBy, cur.CreatedAt
	}
	in.UpdatedBy, in.UpdatedAt = actor.ref(), now
	if err := f.Store.SaveForm(ctx, &in); err != nil {
		var u *domain.ErrUnique
		if asUnique(err, &u) {
			return nil, conflict("Slug sudah dipakai form lain")
		}
		return nil, err
	}
	return f.Store.GetForm(ctx, in.ID)
}

func (f *Forms) DeleteForm(ctx context.Context, id string) error { return f.Store.DeleteForm(ctx, id) }

func (f *Forms) ListSubmissions(ctx context.Context, formID string) ([]domain.FormSubmission, error) {
	return f.Store.ListFormSubmissions(ctx, formID, 200)
}

// PublicForm returns an active form by slug.
func (f *Forms) PublicForm(ctx context.Context, slug string) (*domain.Form, error) {
	form, err := f.Store.GetFormBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if form == nil || !form.IsActive {
		return nil, notFound(msgFormNotFound)
	}
	return form, nil
}

// SubmitResult is the public answer.
type SubmitResult struct {
	OK          bool
	Message     string
	RedirectURL string
	Errors      []domain.FieldError
}

// Submit is POST /api/public/crm/forms/{slug}: spam checks, validation,
// a new lead with attribution, the submission log and sales alerts.
func (f *Forms) Submit(ctx context.Context, slug, ip string, payload map[string]any) (*SubmitResult, error) {
	if err := f.limiter.Check("form:" + ip); err != nil {
		return nil, err
	}
	form, err := f.PublicForm(ctx, slug)
	if err != nil {
		return nil, err
	}
	now := f.now()
	attr := domain.ParseAttribution(payload)
	raw, _ := json.Marshal(payload)
	utm, _ := json.Marshal(attr)
	sub := domain.FormSubmission{ID: ids.New(), FormID: form.ID, Payload: raw, Utm: utm, IPHash: domain.HashIP(ip), CreatedAt: now}
	if bot, reason := domain.IsLikelyBot(payload, now); bot {
		sub.Status, sub.Reason = "spam", &reason
		if err := f.Store.InsertFormSubmission(ctx, sub); err != nil {
			return nil, err
		}
		// Bots get the same answer as people.
		return &SubmitResult{OK: true, Message: form.SuccessMessage, RedirectURL: deref(form.RedirectURL)}, nil
	}
	v := domain.ValidateSubmission(form.Fields, payload)
	if !v.OK() {
		reason := v.Errors[0].Message
		sub.Status, sub.Reason = "invalid", &reason
		if err := f.Store.InsertFormSubmission(ctx, sub); err != nil {
			return nil, err
		}
		return &SubmitResult{OK: false, Message: reason, Errors: v.Errors}, nil
	}
	phone := ""
	if raw := v.Lead["pic_phone"]; raw != "" {
		if p := domain.NormalizePhone(raw); domain.IsValidNormalizedPhone(p) {
			phone = p
		}
	}
	orgType := v.Lead["org_type"]
	if !domain.Contains(domain.LeadOrgTypes, orgType) {
		orgType = "corporate"
	}
	custom, _ := json.Marshal(v.Custom)
	picName := v.Lead["pic_name"]
	if picName == "" {
		picName = domain.LeadOrgName(v.Lead)
	}
	lead := &domain.Lead{ID: ids.New(), OrgName: domain.LeadOrgName(v.Lead), OrgType: orgType, PicName: picName, PicPhone: domain.TrimmedOrNil(phone), PicEmail: domain.TrimmedOrNil(v.Lead["pic_email"]), City: domain.TrimmedOrNil(v.Lead["city"]), Notes: domain.TrimmedOrNil(v.Lead["notes"]),
		Source: domain.SourceFromAttribution(attr, form.DefaultSource), Temperature: form.LeadTemperature, Status: "baru", Custom: custom, Attribution: attr, Audit: domain.Audit{CreatedAt: now, UpdatedAt: now}}
	sub.Status, sub.LeadID = "accepted", &lead.ID
	err = f.Store.WithTx(ctx, func(st domain.Store) error {
		if err := st.SaveLead(ctx, lead); err != nil {
			return err
		}
		return st.InsertFormSubmission(ctx, sub)
	})
	if err != nil {
		return nil, err
	}
	f.notify(ctx, Notification{UserIDs: form.NotifyUserIDs, Type: NotifLeadNew, Title: "Lead baru dari form " + form.Name, Body: domain.BuildLeadAlert(form.Name, v.Lead, attr), Href: "/erp/sales/leads/" + lead.ID})
	f.auto.Emit(ctx, Event{Type: "lead.created", SubjectType: "lead", SubjectID: lead.ID, Payload: map[string]any{"form": form.Slug}})
	return &SubmitResult{OK: true, Message: form.SuccessMessage, RedirectURL: deref(form.RedirectURL)}, nil
}
