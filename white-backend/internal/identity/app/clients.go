package app

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/identity/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/password"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

const (
	msgClientNameRequired = "Nama klien wajib diisi"
	msgSlugRequired       = "Slug wajib diisi"
	msgSlugInvalid        = "Slug hanya boleh huruf kecil, angka, dan tanda hubung"
	msgSlugTaken          = "Slug sudah dipakai"
	msgWebsiteInvalid     = "URL website tidak valid (awali dengan https://)"
	msgCurrencyInvalid    = "Mata uang tidak valid"
	msgTimezoneInvalid    = "Zona waktu tidak valid"
	msgPinInvalid         = "PIN harus 6 digit angka"
	msgModulesMin         = "Pilih minimal satu modul"
	msgModuleInvalid      = "Modul tidak valid"
	msgPickOneChannel     = "Pilih minimal satu kanal."
)

var (
	slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	pinRe  = regexp.MustCompile(`^\d{6}$`)
)

type ClientProfile struct {
	Name        string
	Slug        string
	Description string
	Industry    string
	WebsiteURL  string
	Currency    string
	Timezone    string
}

func (p *ClientProfile) normalize() {
	p.Name = strings.TrimSpace(p.Name)
	p.Slug = strings.ToLower(strings.TrimSpace(p.Slug))
	p.Description = strings.TrimSpace(p.Description)
	p.Industry = strings.TrimSpace(p.Industry)
	p.WebsiteURL = strings.TrimSpace(p.WebsiteURL)
}

func (p ClientProfile) validate() error {
	v := validate.New()
	v.Required("name", p.Name, msgClientNameRequired).MaxLen("name", p.Name, 80, msgTooLong)
	v.MinLen("slug", p.Slug, 2, msgSlugRequired).MaxLen("slug", p.Slug, 60, msgTooLong).Match("slug", p.Slug, slugRe, msgSlugInvalid)
	v.MaxLen("description", p.Description, 500, msgTooLong)
	v.MaxLen("industry", p.Industry, 60, msgTooLong)
	v.HTTPURL("websiteUrl", p.WebsiteURL, msgWebsiteInvalid)
	v.OneOf("currency", p.Currency, domain.Currencies, msgCurrencyInvalid)
	v.OneOf("timezone", p.Timezone, domain.Timezones, msgTimezoneInvalid)
	return v.Err()
}

type ClientSummary struct {
	Client       domain.Client
	MembersCount int32
	MyRole       domain.AccessRole
}

type Clients struct {
	store domain.Store
	bus   events.Publisher
}

func NewClients(store domain.Store, bus events.Publisher) *Clients {
	return &Clients{store: store, bus: bus}
}

// Access resolves what a user may do with a client. Admins see everything;
// members need a membership row. A missing client is NOT_FOUND, a missing
// membership UNAUTHORIZED.
func (s *Clients) Access(ctx context.Context, userID string, isAdmin bool, clientID, slug string) (*domain.ClientAccess, error) {
	var c *domain.Client
	var err error
	if clientID != "" {
		c, err = s.store.GetClient(ctx, clientID)
	} else {
		c, err = s.store.GetClientBySlug(ctx, slug)
	}
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, apperr.New(apperr.NotFound, "Klien tidak ditemukan.")
	}
	if isAdmin {
		return &domain.ClientAccess{Client: *c, Role: domain.AccessAdmin, CanManage: true}, nil
	}
	m, err := s.store.GetMembership(ctx, userID, c.ID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	role := domain.AccessViewer
	if m.Role == domain.MemberManager {
		role = domain.AccessManager
	}
	return &domain.ClientAccess{Client: *c, Role: role, CanManage: role == domain.AccessManager}, nil
}

// manage is the ADMIN-or-MANAGER gate. Like the old manageContext it answers
// UNAUTHORIZED for an unknown client.
func (s *Clients) manage(ctx context.Context, userID string, isAdmin bool, clientID string) (*domain.Client, error) {
	acc, err := s.Access(ctx, userID, isAdmin, clientID, "")
	if err != nil {
		if apperr.Is(err, apperr.NotFound) {
			return nil, apperr.New(apperr.Unauthorized, "")
		}
		return nil, err
	}
	if !acc.CanManage {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	return &acc.Client, nil
}

func (s *Clients) List(ctx context.Context, userID string, isAdmin bool) ([]ClientSummary, error) {
	var clients []domain.Client
	var err error
	if isAdmin {
		clients, err = s.store.ListAllClients(ctx)
	} else {
		clients, err = s.store.ListClientsForUser(ctx, userID)
	}
	if err != nil {
		return nil, err
	}
	clientIDs := make([]string, 0, len(clients))
	for _, c := range clients {
		clientIDs = append(clientIDs, c.ID)
	}
	counts, err := s.store.CountMembers(ctx, clientIDs)
	if err != nil {
		return nil, err
	}
	roles := map[string]domain.MemberRole{}
	if !isAdmin {
		ms, err := s.store.ListMembershipsForUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		for _, m := range ms {
			roles[m.ClientID] = m.Role
		}
	}
	out := make([]ClientSummary, 0, len(clients))
	for _, c := range clients {
		role := domain.AccessAdmin
		if !isAdmin {
			role = domain.AccessViewer
			if roles[c.ID] == domain.MemberManager {
				role = domain.AccessManager
			}
		}
		out = append(out, ClientSummary{Client: c, MembersCount: counts[c.ID], MyRole: role})
	}
	return out, nil
}

func (s *Clients) ListAll(ctx context.Context) ([]domain.Client, error) {
	return s.store.ListAllClients(ctx)
}

func (s *Clients) ListAccessibleIDs(ctx context.Context, userID string, isAdmin bool) ([]string, error) {
	if isAdmin {
		all, err := s.store.ListAllClients(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, len(all))
		for _, c := range all {
			out = append(out, c.ID)
		}
		return out, nil
	}
	return s.store.ListClientIDsForUser(ctx, userID)
}

// Create is ADMIN only. `modules` comes from the wizard; empty means all.
func (s *Clients) Create(ctx context.Context, actorID string, p ClientProfile, modules []string) (*domain.Client, error) {
	p.normalize()
	if err := p.validate(); err != nil {
		return nil, err
	}
	if len(modules) == 0 {
		modules = domain.AllModules
	}
	for _, m := range modules {
		if !contains(domain.AllModules, m) {
			return nil, apperr.New(apperr.Validation, msgPickOneChannel)
		}
	}
	if taken, err := s.store.GetClientBySlug(ctx, p.Slug); err != nil {
		return nil, err
	} else if taken != nil {
		return nil, apperr.Field("slug", msgSlugTaken)
	}
	c, err := s.store.InsertClient(ctx, domain.Client{
		ID: ids.New(), Name: p.Name, Slug: p.Slug, Description: p.Description,
		Industry: validate.EmptyToNil(p.Industry), WebsiteURL: validate.EmptyToNil(p.WebsiteURL),
		Currency: p.Currency, Timezone: p.Timezone, ShareModules: modules, CreatedByID: &actorID,
	})
	if err != nil {
		return nil, slugUnique(err)
	}
	// Analytics creates the SEO property from this when SEO is on and a website is set.
	_ = s.bus.Publish(ctx, events.ClientCreated, events.ClientCreatedPayload{ClientID: c.ID, Slug: c.Slug, WebsiteURL: p.WebsiteURL, Modules: modules})
	return c, nil
}

func (s *Clients) Update(ctx context.Context, userID string, isAdmin bool, clientID string, p ClientProfile) (*domain.Client, error) {
	existing, err := s.manage(ctx, userID, isAdmin, clientID)
	if err != nil {
		return nil, err
	}
	p.normalize()
	if err := p.validate(); err != nil {
		return nil, err
	}
	if p.Slug != existing.Slug {
		if taken, err := s.store.GetClientBySlug(ctx, p.Slug); err != nil {
			return nil, err
		} else if taken != nil {
			return nil, apperr.Field("slug", msgSlugTaken)
		}
	}
	c, err := s.store.UpdateClient(ctx, domain.Client{
		ID: clientID, Name: p.Name, Slug: p.Slug, Description: p.Description,
		Industry: validate.EmptyToNil(p.Industry), WebsiteURL: validate.EmptyToNil(p.WebsiteURL),
		Currency: p.Currency, Timezone: p.Timezone,
	})
	if err != nil {
		return nil, slugUnique(err)
	}
	return c, nil
}

// Delete is ADMIN only. Other services purge their rows on the event.
func (s *Clients) Delete(ctx context.Context, clientID string) error {
	c, err := s.store.GetClient(ctx, clientID)
	if err != nil {
		return err
	}
	if c == nil {
		return apperr.New(apperr.NotFound, "")
	}
	if _, err := s.store.DeleteClient(ctx, clientID); err != nil {
		return err
	}
	return s.bus.Publish(ctx, events.ClientDeleted, events.ClientDeletedPayload{ClientID: c.ID, Slug: c.Slug})
}

func (s *Clients) ListMembers(ctx context.Context, userID string, isAdmin bool, clientID string) ([]domain.ClientMember, error) {
	if _, err := s.Access(ctx, userID, isAdmin, clientID, ""); err != nil {
		return nil, err
	}
	return s.store.ListClientMembers(ctx, clientID)
}

func (s *Clients) UpsertMember(ctx context.Context, userID string, isAdmin bool, clientID, memberID string, role domain.MemberRole) error {
	if _, err := s.manage(ctx, userID, isAdmin, clientID); err != nil {
		return err
	}
	if memberID == "" || (role != domain.MemberManager && role != domain.MemberViewer) {
		return apperr.New(apperr.Validation, "")
	}
	u, err := s.store.GetUser(ctx, memberID)
	if err != nil {
		return err
	}
	if u == nil {
		return apperr.New(apperr.NotFound, "")
	}
	return s.store.UpsertMembership(ctx, domain.Membership{UserID: memberID, ClientID: clientID, Role: role})
}

func (s *Clients) RemoveMember(ctx context.Context, userID string, isAdmin bool, clientID, memberID string) error {
	if _, err := s.manage(ctx, userID, isAdmin, clientID); err != nil {
		return err
	}
	ok, err := s.store.DeleteMembership(ctx, memberID, clientID)
	if err != nil {
		return err
	}
	if !ok {
		return apperr.New(apperr.NotFound, "")
	}
	return nil
}

// UpdateShare sets the public dashboard flags. An empty PIN clears the PIN;
// any PIN is re-hashed, which invalidates issued unlock cookies.
func (s *Clients) UpdateShare(ctx context.Context, userID string, isAdmin bool, clientID string, enabled bool, pin string, modules []string) (*domain.Client, error) {
	if _, err := s.manage(ctx, userID, isAdmin, clientID); err != nil {
		return nil, err
	}
	v := validate.New()
	if pin != "" {
		v.Match("pin", pin, pinRe, msgPinInvalid)
	}
	if len(modules) == 0 {
		v.Add("modules", msgModulesMin)
	}
	for _, m := range modules {
		v.OneOf("modules", m, domain.AllModules, msgModuleInvalid)
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	var hash *string
	if pin != "" {
		h, err := password.Hash(pin)
		if err != nil {
			return nil, err
		}
		hash = &h
	}
	return s.store.UpdateShareSettings(ctx, clientID, enabled, hash, modules)
}

func slugUnique(err error) error {
	var u *domain.ErrUnique
	if errors.As(err, &u) {
		return apperr.Field("slug", msgSlugTaken)
	}
	return err
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
