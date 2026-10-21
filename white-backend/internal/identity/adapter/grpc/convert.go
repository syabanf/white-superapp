// Package grpcapi exposes the identity use cases over gRPC. Handlers only
// translate between proto and domain; rules live in app.
package grpcapi

import (
	"time"

	commonv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/common/v1"
	identityv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/identity/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/identity/app"
	"github.com/syabanf/white-superapp/white-backend/internal/identity/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ts(t time.Time) *timestamppb.Timestamp { return timestamppb.New(t) }

func tsPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func userRole(r domain.UserRole) identityv1.UserRole {
	if r == domain.RoleAdmin {
		return identityv1.UserRole_ADMIN
	}
	return identityv1.UserRole_MEMBER
}

func userRoleFrom(r identityv1.UserRole) domain.UserRole {
	switch r {
	case identityv1.UserRole_ADMIN:
		return domain.RoleAdmin
	case identityv1.UserRole_MEMBER:
		return domain.RoleMember
	}
	return ""
}

func memberRole(r domain.MemberRole) identityv1.ClientMemberRole {
	if r == domain.MemberManager {
		return identityv1.ClientMemberRole_MANAGER
	}
	return identityv1.ClientMemberRole_VIEWER
}

func memberRoleFrom(r identityv1.ClientMemberRole) domain.MemberRole {
	switch r {
	case identityv1.ClientMemberRole_MANAGER:
		return domain.MemberManager
	case identityv1.ClientMemberRole_VIEWER:
		return domain.MemberViewer
	}
	return ""
}

func accessRole(r domain.AccessRole) identityv1.ClientAccess_AccessRole {
	switch r {
	case domain.AccessAdmin:
		return identityv1.ClientAccess_ADMIN_OVERRIDE
	case domain.AccessManager:
		return identityv1.ClientAccess_MANAGER
	}
	return identityv1.ClientAccess_VIEWER
}

func modules(keys []string) []commonv1.ModuleKey {
	out := make([]commonv1.ModuleKey, 0, len(keys))
	for _, k := range keys {
		if v, ok := commonv1.ModuleKey_value[k]; ok {
			out = append(out, commonv1.ModuleKey(v))
		}
	}
	return out
}

func modulesFrom(in []commonv1.ModuleKey) []string {
	out := make([]string, 0, len(in))
	for _, m := range in {
		if m != commonv1.ModuleKey_MODULE_KEY_UNSPECIFIED {
			out = append(out, m.String())
		}
	}
	return out
}

func userPB(u *domain.User) *identityv1.User {
	return &identityv1.User{Id: u.ID, Email: u.Email, Name: u.Name, Role: userRole(u.Role), IsActive: u.IsActive, CreatedAt: ts(u.CreatedAt)}
}

func clientPB(c *domain.Client) *identityv1.Client {
	return &identityv1.Client{
		Id: c.ID, Name: c.Name, Slug: c.Slug, Description: c.Description,
		LogoUrl: c.LogoURL, WebsiteUrl: c.WebsiteURL, Industry: c.Industry,
		Currency: c.Currency, Timezone: c.Timezone,
		ShareEnabled: c.ShareEnabled, SharePinSet: c.SharePinHash != nil,
		ShareModules: modules(c.ShareModules), CreatedAt: ts(c.CreatedAt),
	}
}

func accessPB(a *domain.ClientAccess) *identityv1.ClientAccess {
	return &identityv1.ClientAccess{Client: clientPB(&a.Client), Role: accessRole(a.Role), CanManage: a.CanManage}
}

func tokensPB(t *app.Tokens) *identityv1.TokenPair {
	return &identityv1.TokenPair{
		AccessToken: t.AccessToken, AccessExpiresAt: ts(t.AccessExpiresAt),
		RefreshToken: t.RefreshToken, RefreshExpiresAt: ts(t.RefreshExpiresAt),
		User: userPB(&t.User),
	}
}

func membershipsFrom(in []*identityv1.Membership) []domain.Membership {
	out := make([]domain.Membership, 0, len(in))
	for _, m := range in {
		out = append(out, domain.Membership{ClientID: m.ClientId, Role: memberRoleFrom(m.Role)})
	}
	return out
}

func profileFrom(p *identityv1.ClientProfile) app.ClientProfile {
	if p == nil {
		return app.ClientProfile{}
	}
	return app.ClientProfile{Name: p.Name, Slug: p.Slug, Description: p.Description, Industry: p.Industry, WebsiteURL: p.WebsiteUrl, Currency: p.Currency, Timezone: p.Timezone}
}
