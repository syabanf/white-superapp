package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Leads covers leads, accounts, contacts and custom fields (salesfunnel
// leads.go, accounts.go, leadimport.go).
type Leads struct {
	*Deps
	auto *Automation
}

func NewLeads(d *Deps, auto *Automation) *Leads { return &Leads{Deps: d, auto: auto} }

// LeadInput is the create/update payload.
type LeadInput struct {
	ID, OrgName, OrgType, PicName, PicTitle, PicPhone, PicEmail, City string
	Source, Temperature, Status, Notes, OwnerUserID, AccountID        string
	ContactID                                                         string
	Custom                                                            json.RawMessage
	Attribution                                                       domain.Attribution
}

func (l *Leads) validateCustom(ctx context.Context, object string, raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 || string(raw) == "null" {
		raw = json.RawMessage("{}")
	}
	var values map[string]any
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, apperr.Field("custom", "Field tambahan tidak valid")
	}
	fields, err := l.Store.ListCustomFields(ctx, object)
	if err != nil {
		return nil, err
	}
	for _, f := range fields {
		if !f.IsActive || !f.IsRequired {
			continue
		}
		if v, ok := values[f.Key]; !ok || v == nil || v == "" {
			return nil, apperr.Field("custom", f.Label+" wajib diisi")
		}
	}
	return raw, nil
}

// normalizeLead validates the payload and writes it onto l.
func (l *Leads) normalizeLead(ctx context.Context, in LeadInput, lead *domain.Lead) error {
	errs := map[string]string{}
	if domain.JSTrim(in.OrgName) == "" {
		errs["org_name"] = "Nama instansi wajib diisi"
	}
	if domain.JSTrim(in.PicName) == "" {
		errs["pic_name"] = "Nama PIC wajib diisi"
	}
	phone := ""
	if raw := domain.JSTrim(in.PicPhone); raw != "" {
		phone = domain.NormalizePhone(raw)
		if !domain.IsValidNormalizedPhone(phone) {
			errs["pic_phone"] = "No. WA tidak valid"
		}
	}
	email := strings.ToLower(domain.JSTrim(in.PicEmail))
	if email != "" && !domain.IsEmail(email) {
		errs["pic_email"] = "Email tidak valid"
	}
	if phone == "" && email == "" {
		errs["pic_phone"] = "Isi No. WA atau email PIC"
	}
	pick := func(field, v string, allowed []string, def string) string {
		if v == "" {
			return def
		}
		if !domain.Contains(allowed, v) {
			errs[field] = "Pilihan tidak valid"
		}
		return v
	}
	orgType := pick("org_type", in.OrgType, domain.LeadOrgTypes, "corporate")
	source := pick("source", in.Source, domain.LeadSources, "lainnya")
	temp := pick("temperature", in.Temperature, domain.LeadTemperatures, "hangat")
	status := pick("status", in.Status, domain.LeadStatuses, "baru")
	if err := fieldErrs(errs); err != nil {
		return err
	}
	custom, err := l.validateCustom(ctx, "lead", in.Custom)
	if err != nil {
		return err
	}
	if in.AccountID != "" {
		if a, err := l.Store.GetAccount(ctx, in.AccountID); err != nil {
			return err
		} else if a == nil {
			return notFound(msgAccountNotFound)
		}
	}
	lead.OrgName, lead.OrgType, lead.PicName = domain.SliceUTF16(domain.JSTrim(in.OrgName), 200), orgType, domain.SliceUTF16(domain.JSTrim(in.PicName), 150)
	lead.PicTitle, lead.PicPhone, lead.PicEmail = domain.TrimmedOrNil(in.PicTitle), domain.TrimmedOrNil(phone), domain.TrimmedOrNil(email)
	lead.City, lead.Source, lead.Temperature, lead.Status = domain.TrimmedOrNil(in.City), source, temp, status
	lead.Notes, lead.OwnerUserID = domain.TrimmedOrNil(in.Notes), domain.TrimmedOrNil(in.OwnerUserID)
	lead.AccountID, lead.ContactID, lead.Custom = domain.TrimmedOrNil(in.AccountID), domain.TrimmedOrNil(in.ContactID), custom
	return nil
}

func (l *Leads) CreateLead(ctx context.Context, actor Actor, in LeadInput) (*domain.Lead, error) {
	now := l.now()
	lead := &domain.Lead{ID: ids.New(), Attribution: in.Attribution, Audit: domain.Audit{CreatedBy: actor.ref(), UpdatedBy: actor.ref(), CreatedAt: now, UpdatedAt: now}}
	if err := l.normalizeLead(ctx, in, lead); err != nil {
		return nil, err
	}
	if lead.OwnerUserID == nil && !actor.Admin {
		lead.OwnerUserID = actor.ref()
	}
	if err := l.Store.SaveLead(ctx, lead); err != nil {
		return nil, err
	}
	l.auto.Emit(ctx, Event{Type: "lead.created", SubjectType: "lead", SubjectID: lead.ID, Actor: actor.ref()})
	return l.Store.GetLead(ctx, lead.ID)
}

func (l *Leads) UpdateLead(ctx context.Context, actor Actor, in LeadInput) (*domain.Lead, error) {
	lead, err := l.Store.GetLead(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if lead == nil {
		return nil, notFound(msgLeadNotFound)
	}
	before := *lead
	if err := l.normalizeLead(ctx, in, lead); err != nil {
		return nil, err
	}
	lead.UpdatedBy, lead.UpdatedAt = actor.ref(), l.now()
	if err := l.Store.SaveLead(ctx, lead); err != nil {
		return nil, err
	}
	changes := map[string]domain.Change{}
	track := func(field, from, to string) {
		if from != to {
			changes[field] = domain.Change{From: from, To: to}
		}
	}
	track("status", before.Status, lead.Status)
	track("temperature", before.Temperature, lead.Temperature)
	track("source", before.Source, lead.Source)
	track("owner_user_id", deref(before.OwnerUserID), deref(lead.OwnerUserID))
	l.auto.Emit(ctx, Event{Type: "lead.updated", SubjectType: "lead", SubjectID: lead.ID, Actor: actor.ref(), Changes: changes})
	if _, ok := changes["status"]; ok {
		l.auto.Emit(ctx, Event{Type: "lead.status_changed", SubjectType: "lead", SubjectID: lead.ID, Actor: actor.ref(), Changes: changes, Payload: map[string]any{"from": before.Status, "to": lead.Status}})
	}
	return l.Store.GetLead(ctx, lead.ID)
}

func (l *Leads) DeleteLead(ctx context.Context, id string) error {
	lead, err := l.Store.GetLead(ctx, id)
	if err != nil {
		return err
	}
	if lead == nil {
		return notFound(msgLeadNotFound)
	}
	return l.Store.DeleteLead(ctx, id, l.now())
}

func (l *Leads) ListLeads(ctx context.Context, f domain.LeadFilter) ([]domain.Lead, int, error) {
	return l.Store.ListLeads(ctx, f)
}

// LeadDetail is the lead with its deals, open tasks and conversations.
type LeadDetail struct {
	Lead          domain.Lead
	Deals         []domain.Deal
	Activities    []domain.Activity
	Conversations []domain.Conversation
}

func (l *Leads) GetLead(ctx context.Context, id string) (*LeadDetail, error) {
	lead, err := l.Store.GetLead(ctx, id)
	if err != nil {
		return nil, err
	}
	if lead == nil {
		return nil, notFound(msgLeadNotFound)
	}
	deals, _, err := l.Store.ListDeals(ctx, domain.DealFilter{LeadID: id, Limit: 100})
	if err != nil {
		return nil, err
	}
	acts, err := l.Store.ListActivities(ctx, domain.ActivityFilter{LeadID: id, Limit: 100})
	if err != nil {
		return nil, err
	}
	convs, err := l.Store.ListConversations(ctx, domain.ConversationFilter{LeadID: id, Limit: 20})
	if err != nil {
		return nil, err
	}
	return &LeadDetail{Lead: *lead, Deals: deals, Activities: acts, Conversations: convs}, nil
}

// ImportRowError reports one skipped or warned row (row 1 is the header).
type ImportRowError struct {
	Row     int
	Message string
}

// ImportResult is leadImportResult.
type ImportResult struct {
	Imported, Skipped int
	Errors            []ImportRowError
}

// ImportLeads is POST /leads/import: CSV or XLSX rows become leads. Rows
// that fail are skipped and reported by row number.
func (l *Leads) ImportLeads(ctx context.Context, actor Actor, fileName string, data []byte) (*ImportResult, error) {
	if len(data) == 0 {
		return nil, invalid("File tidak ditemukan")
	}
	if len(data) > domain.LeadImportMaxBytes {
		return nil, invalid("Ukuran file maksimal 5 MB")
	}
	rows, err := domain.SpreadsheetMatrix(data, fileName)
	if err != nil {
		return nil, invalid(err.Error())
	}
	if len(rows) < 2 {
		return nil, invalid("File harus berisi baris header dan minimal satu baris data")
	}
	if len(rows)-1 > domain.LeadImportMaxRows {
		return nil, invalid(fmt.Sprintf("Maksimal %d baris per import", domain.LeadImportMaxRows))
	}
	headers := make([]string, len(rows[0]))
	for i, h := range rows[0] {
		headers[i] = domain.NormalizeLeadHeader(h)
	}
	used, err := l.Store.LeadDedupKeys(ctx)
	if err != nil {
		return nil, err
	}
	var owner *string
	if !actor.Admin {
		owner = actor.ref()
	}
	res := &ImportResult{Errors: []ImportRowError{}}
	for i := 1; i < len(rows); i++ {
		row := i + 1
		m := domain.MapLeadImportRow(headers, rows[i])
		switch m.Kind {
		case domain.LeadRowEmpty:
			continue
		case domain.LeadRowInvalid:
			res.Errors = append(res.Errors, ImportRowError{row, m.Message})
			res.Skipped++
			continue
		}
		key := domain.LeadDedupKey(m.Lead.PicPhone, m.Lead.OrgName)
		if used[key] {
			res.Errors = append(res.Errors, ImportRowError{row, "Duplikat instansi + no. WA: " + m.Lead.OrgName + " / " + m.RawPhone})
			res.Skipped++
			continue
		}
		if m.Warning != "" {
			res.Errors = append(res.Errors, ImportRowError{row, m.Warning})
		}
		now := l.now()
		r := m.Lead
		lead := &domain.Lead{ID: ids.New(), OrgName: r.OrgName, OrgType: r.OrgType, PicName: r.PicName, PicTitle: r.PicTitle, PicPhone: &r.PicPhone, PicEmail: r.PicEmail, City: r.City, Source: r.Source, Temperature: r.Temperature, Status: "baru", Notes: r.Notes, OwnerUserID: owner, Custom: json.RawMessage("{}"),
			Audit: domain.Audit{CreatedBy: actor.ref(), UpdatedBy: actor.ref(), CreatedAt: now, UpdatedAt: now}}
		if err := l.Store.SaveLead(ctx, lead); err != nil {
			res.Errors = append(res.Errors, ImportRowError{row, "Gagal menyimpan baris"})
			res.Skipped++
			continue
		}
		used[key] = true
		res.Imported++
	}
	return res, nil
}

func (l *Leads) ListOwners(ctx context.Context) ([]User, error) {
	if l.Directory == nil {
		return []User{}, nil
	}
	return l.Directory.ListUsers(ctx)
}

// ─── accounts ────────────────────────────────────────────────

type AccountInput struct {
	ID, Name, AccountType, Industry, Address, City, Phone, Email, Website, Npwp, Notes, OwnerUserID string
	Custom                                                                                          json.RawMessage
}

func (l *Leads) applyAccount(ctx context.Context, in AccountInput, a *domain.Account) error {
	errs := map[string]string{}
	if domain.JSTrim(in.Name) == "" {
		errs["name"] = "Nama akun wajib diisi"
	}
	accType := in.AccountType
	if accType == "" {
		accType = "corporate"
	} else if !domain.Contains(domain.AccountTypes, accType) {
		errs["account_type"] = "Jenis akun tidak valid"
	}
	email := strings.ToLower(domain.JSTrim(in.Email))
	if email != "" && !domain.IsEmail(email) {
		errs["email"] = "Email tidak valid"
	}
	if err := fieldErrs(errs); err != nil {
		return err
	}
	custom, err := l.validateCustom(ctx, "account", in.Custom)
	if err != nil {
		return err
	}
	a.Name, a.AccountType, a.Industry, a.Address = domain.JSTrim(in.Name), accType, domain.TrimmedOrNil(in.Industry), domain.TrimmedOrNil(in.Address)
	a.City, a.Phone, a.Email, a.Website = domain.TrimmedOrNil(in.City), domain.TrimmedOrNil(in.Phone), domain.TrimmedOrNil(email), domain.TrimmedOrNil(in.Website)
	a.Npwp, a.Notes, a.OwnerUserID, a.Custom = domain.TrimmedOrNil(in.Npwp), domain.TrimmedOrNil(in.Notes), domain.TrimmedOrNil(in.OwnerUserID), custom
	return nil
}

func (l *Leads) CreateAccount(ctx context.Context, actor Actor, in AccountInput) (*domain.Account, error) {
	now := l.now()
	a := &domain.Account{ID: ids.New(), Audit: domain.Audit{CreatedBy: actor.ref(), UpdatedBy: actor.ref(), CreatedAt: now, UpdatedAt: now}}
	if err := l.applyAccount(ctx, in, a); err != nil {
		return nil, err
	}
	if err := l.Store.SaveAccount(ctx, a); err != nil {
		return nil, err
	}
	l.auto.Emit(ctx, Event{Type: "account.created", SubjectType: "account", SubjectID: a.ID, Actor: actor.ref()})
	return a, nil
}

func (l *Leads) UpdateAccount(ctx context.Context, actor Actor, in AccountInput) (*domain.Account, error) {
	a, err := l.Store.GetAccount(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, notFound(msgAccountNotFound)
	}
	if err := l.applyAccount(ctx, in, a); err != nil {
		return nil, err
	}
	a.UpdatedBy, a.UpdatedAt = actor.ref(), l.now()
	if err := l.Store.SaveAccount(ctx, a); err != nil {
		return nil, err
	}
	l.auto.Emit(ctx, Event{Type: "account.updated", SubjectType: "account", SubjectID: a.ID, Actor: actor.ref()})
	return a, nil
}

func (l *Leads) DeleteAccount(ctx context.Context, id string) error {
	return l.Store.DeleteAccount(ctx, id, l.now())
}

func (l *Leads) ListAccounts(ctx context.Context, q string, limit int) ([]domain.Account, error) {
	return l.Store.ListAccounts(ctx, q, limit)
}

type AccountDetail struct {
	Account  domain.Account
	Contacts []domain.Contact
	Deals    []domain.Deal
}

func (l *Leads) GetAccount(ctx context.Context, id string) (*AccountDetail, error) {
	a, err := l.Store.GetAccount(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, notFound(msgAccountNotFound)
	}
	contacts, err := l.Store.ListContacts(ctx, id, "", 200)
	if err != nil {
		return nil, err
	}
	deals, _, err := l.Store.ListDeals(ctx, domain.DealFilter{AccountID: id, Limit: 200})
	if err != nil {
		return nil, err
	}
	return &AccountDetail{Account: *a, Contacts: contacts, Deals: deals}, nil
}

// ─── contacts ────────────────────────────────────────────────

type ContactInput struct {
	ID, AccountID, Name, Title, Phone, Email, Notes string
	IsPrimary                                       bool
	Custom                                          json.RawMessage
}

func (l *Leads) applyContact(ctx context.Context, in ContactInput, c *domain.Contact) error {
	errs := map[string]string{}
	if domain.JSTrim(in.Name) == "" {
		errs["name"] = "Nama kontak wajib diisi"
	}
	phone := ""
	if raw := domain.JSTrim(in.Phone); raw != "" {
		if phone = domain.NormalizePhone(raw); !domain.IsValidNormalizedPhone(phone) {
			errs["phone"] = "No. telepon tidak valid"
		}
	}
	email := strings.ToLower(domain.JSTrim(in.Email))
	if email != "" && !domain.IsEmail(email) {
		errs["email"] = "Email tidak valid"
	}
	if err := fieldErrs(errs); err != nil {
		return err
	}
	if in.AccountID != "" {
		if a, err := l.Store.GetAccount(ctx, in.AccountID); err != nil {
			return err
		} else if a == nil {
			return notFound(msgAccountNotFound)
		}
	}
	custom, err := l.validateCustom(ctx, "contact", in.Custom)
	if err != nil {
		return err
	}
	c.AccountID, c.Name, c.Title = domain.TrimmedOrNil(in.AccountID), domain.JSTrim(in.Name), domain.TrimmedOrNil(in.Title)
	c.Phone, c.Email, c.IsPrimary, c.Notes, c.Custom = domain.TrimmedOrNil(phone), domain.TrimmedOrNil(email), in.IsPrimary, domain.TrimmedOrNil(in.Notes), custom
	return nil
}

func (l *Leads) CreateContact(ctx context.Context, actor Actor, in ContactInput) (*domain.Contact, error) {
	now := l.now()
	c := &domain.Contact{ID: ids.New(), Audit: domain.Audit{CreatedBy: actor.ref(), UpdatedBy: actor.ref(), CreatedAt: now, UpdatedAt: now}}
	if err := l.applyContact(ctx, in, c); err != nil {
		return nil, err
	}
	if err := l.Store.SaveContact(ctx, c); err != nil {
		return nil, err
	}
	l.auto.Emit(ctx, Event{Type: "contact.created", SubjectType: "contact", SubjectID: c.ID, Actor: actor.ref()})
	return c, nil
}

func (l *Leads) UpdateContact(ctx context.Context, actor Actor, in ContactInput) (*domain.Contact, error) {
	c, err := l.Store.GetContact(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, notFound(msgContactNotFound)
	}
	if err := l.applyContact(ctx, in, c); err != nil {
		return nil, err
	}
	c.UpdatedBy, c.UpdatedAt = actor.ref(), l.now()
	if err := l.Store.SaveContact(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (l *Leads) DeleteContact(ctx context.Context, id string) error {
	return l.Store.DeleteContact(ctx, id, l.now())
}

func (l *Leads) ListContacts(ctx context.Context, accountID, q string, limit int) ([]domain.Contact, error) {
	return l.Store.ListContacts(ctx, accountID, q, limit)
}

// ─── custom fields (ADMIN) ───────────────────────────────────

var customFieldTypes = []string{"text", "textarea", "number", "date", "boolean", "picklist", "multipicklist", "url", "email", "phone"}

func (l *Leads) ListCustomFields(ctx context.Context, object string) ([]domain.CustomField, error) {
	return l.Store.ListCustomFields(ctx, object)
}

func (l *Leads) SaveCustomField(ctx context.Context, f domain.CustomField) (*domain.CustomField, error) {
	errs := map[string]string{}
	if !domain.Contains([]string{"lead", "deal", "account", "contact"}, f.Object) {
		errs["object"] = "Objek tidak valid"
	}
	if !domain.FieldKeyRe.MatchString(f.Key) {
		errs["key"] = "Key hanya huruf kecil, angka, dan garis bawah (2-40 karakter)"
	}
	if domain.JSTrim(f.Label) == "" {
		errs["label"] = "Label wajib diisi"
	}
	if !domain.Contains(customFieldTypes, f.FieldType) {
		errs["field_type"] = "Tipe field tidak valid"
	}
	if (f.FieldType == "picklist" || f.FieldType == "multipicklist") && len(f.Options) == 0 {
		errs["options"] = "Pilihan wajib diisi untuk picklist"
	}
	if err := fieldErrs(errs); err != nil {
		return nil, err
	}
	if f.ID == "" {
		f.ID = ids.New()
	}
	if err := l.Store.SaveCustomField(ctx, f); err != nil {
		var u *domain.ErrUnique
		if asUnique(err, &u) {
			return nil, conflict("Key sudah dipakai")
		}
		return nil, err
	}
	return &f, nil
}

func (l *Leads) DeleteCustomField(ctx context.Context, id string) error {
	return l.Store.DeleteCustomField(ctx, id)
}
