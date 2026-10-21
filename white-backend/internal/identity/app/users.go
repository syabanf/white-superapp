package app

import (
	"context"
	"errors"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/identity/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/password"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

const (
	msgNameRequired      = "Nama wajib diisi"
	msgEmailInvalid      = "Email tidak valid"
	msgEmailTaken        = "Email sudah terdaftar"
	msgPasswordMin       = "Kata sandi minimal 8 karakter"
	msgCannotEditSelf    = "Anda tidak dapat menonaktifkan atau menurunkan peran akun sendiri."
	msgCannotDeleteSelf  = "Anda tidak dapat menghapus akun sendiri."
	msgTooLong           = "Terlalu panjang"
	msgRoleInvalid       = "Peran tidak valid"
	msgMemberRoleInvalid = "Peran anggota tidak valid"
)

type UserInput struct {
	Name        string
	Email       string
	Password    string // empty keeps the current password on update
	Role        domain.UserRole
	IsActive    bool
	Memberships []domain.Membership
}

type UserWithClients struct {
	User    domain.User
	Clients []domain.MembershipWithClient
}

// Users is the admin user management use case.
type Users struct {
	store domain.Store
	bus   events.Publisher
}

func NewUsers(store domain.Store, bus events.Publisher) *Users {
	return &Users{store: store, bus: bus}
}

func (in *UserInput) normalize() {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
}

func (in UserInput) validate(create bool) error {
	v := validate.New()
	v.Required("name", in.Name, msgNameRequired).MaxLen("name", in.Name, 80, msgTooLong)
	v.Email("email", in.Email, msgEmailInvalid)
	v.OneOf("role", string(in.Role), []string{string(domain.RoleAdmin), string(domain.RoleMember)}, msgRoleInvalid)
	if create || in.Password != "" {
		v.MinLen("password", in.Password, 8, msgPasswordMin).MaxLen("password", in.Password, 100, msgTooLong)
	}
	for _, m := range in.Memberships {
		if m.ClientID == "" {
			v.Add("memberships", "Klien wajib dipilih")
		}
		v.OneOf("memberships", string(m.Role), []string{string(domain.MemberManager), string(domain.MemberViewer)}, msgMemberRoleInvalid)
	}
	return v.Err()
}

func (s *Users) List(ctx context.Context) ([]UserWithClients, error) {
	users, err := s.store.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	idsList := make([]string, 0, len(users))
	for _, u := range users {
		idsList = append(idsList, u.ID)
	}
	ms, err := s.store.ListMembershipsOfUsers(ctx, idsList)
	if err != nil {
		return nil, err
	}
	byUser := map[string][]domain.MembershipWithClient{}
	for _, m := range ms {
		byUser[m.UserID] = append(byUser[m.UserID], m)
	}
	out := make([]UserWithClients, 0, len(users))
	for _, u := range users {
		out = append(out, UserWithClients{User: u, Clients: byUser[u.ID]})
	}
	return out, nil
}

func (s *Users) ListOptions(ctx context.Context) ([]domain.User, error) {
	return s.store.ListActiveUserOptions(ctx)
}

func (s *Users) Create(ctx context.Context, in UserInput) (*domain.User, error) {
	in.normalize()
	if err := in.validate(true); err != nil {
		return nil, err
	}
	if existing, err := s.store.GetUserByEmail(ctx, in.Email); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, apperr.Field("email", msgEmailTaken)
	}
	hash, err := password.Hash(in.Password)
	if err != nil {
		return nil, err
	}
	var created *domain.User
	err = s.store.WithTx(ctx, func(tx domain.Store) error {
		u, err := tx.InsertUser(ctx, domain.User{ID: ids.New(), Email: in.Email, Name: in.Name, PasswordHash: hash, Role: in.Role, IsActive: in.IsActive})
		if err != nil {
			return err
		}
		created = u
		return tx.ReplaceMemberships(ctx, u.ID, in.Memberships)
	})
	if err != nil {
		return nil, uniqueToField(err)
	}
	return created, nil
}

// Update replaces profile, password (when given) and every membership in
// one transaction. Admins cannot deactivate or demote themselves.
func (s *Users) Update(ctx context.Context, actorID, userID string, in UserInput) (*domain.User, error) {
	in.normalize()
	if err := in.validate(false); err != nil {
		return nil, err
	}
	existing, err := s.store.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, apperr.New(apperr.NotFound, "")
	}
	if userID == actorID && (!in.IsActive || in.Role != domain.RoleAdmin) {
		return nil, apperr.New(apperr.Validation, msgCannotEditSelf)
	}
	if other, err := s.store.GetUserByEmail(ctx, in.Email); err != nil {
		return nil, err
	} else if other != nil && other.ID != userID {
		return nil, apperr.Field("email", msgEmailTaken)
	}
	var hash *string
	if in.Password != "" {
		h, err := password.Hash(in.Password)
		if err != nil {
			return nil, err
		}
		hash = &h
	}
	var updated *domain.User
	err = s.store.WithTx(ctx, func(tx domain.Store) error {
		u, err := tx.UpdateUser(ctx, domain.User{ID: userID, Name: in.Name, Email: in.Email, Role: in.Role, IsActive: in.IsActive}, hash)
		if err != nil {
			return err
		}
		updated = u
		if err := tx.ReplaceMemberships(ctx, userID, in.Memberships); err != nil {
			return err
		}
		if !in.IsActive || hash != nil {
			return tx.RevokeUserSessions(ctx, userID)
		}
		return nil
	})
	if err != nil {
		return nil, uniqueToField(err)
	}
	return updated, nil
}

func (s *Users) SetActive(ctx context.Context, actorID, userID string, active bool) (*domain.User, error) {
	if userID == actorID && !active {
		return nil, apperr.New(apperr.Validation, msgCannotEditSelf)
	}
	u, err := s.store.SetUserActive(ctx, userID, active)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, apperr.New(apperr.NotFound, "")
	}
	if !active {
		if err := s.store.RevokeUserSessions(ctx, userID); err != nil {
			return nil, err
		}
	}
	return u, nil
}

func (s *Users) Delete(ctx context.Context, actorID, userID string) error {
	if userID == actorID {
		return apperr.New(apperr.Validation, msgCannotDeleteSelf)
	}
	ok, err := s.store.DeleteUser(ctx, userID)
	if err != nil {
		return err
	}
	if !ok {
		return apperr.New(apperr.NotFound, "")
	}
	// Other services null out createdBy/uploadedBy references on this event.
	return s.bus.Publish(ctx, events.UserDeleted, events.UserDeletedPayload{UserID: userID})
}

func uniqueToField(err error) error {
	var u *domain.ErrUnique
	if errors.As(err, &u) {
		return apperr.Field("email", msgEmailTaken)
	}
	return err
}
