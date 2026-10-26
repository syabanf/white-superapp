package gateway

import (
	"context"
	"encoding/json"
	"net/http"

	identityv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/identity/v1"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc/metadata"
)

// internalCtx forwards the shared secret; identity validates it.
func internalCtx(r *http.Request) (context.Context, bool) {
	secret := r.Header.Get("X-Internal-Secret")
	if secret == "" {
		return nil, false
	}
	return metadata.AppendToOutgoingContext(r.Context(), "x-internal-secret", secret), true
}

// internalSyncUser mirrors a Next.js user row into identity.
func (b *BrowserRoutes) internalSyncUser(w http.ResponseWriter, r *http.Request) {
	ctx, ok := internalCtx(r)
	if !ok {
		grpcx.WriteError(w, apperr.New(apperr.Unauthorized, ""))
		return
	}
	var body struct {
		ID           string `json:"id"`
		Email        string `json:"email"`
		Name         string `json:"name"`
		PasswordHash string `json:"passwordHash"`
		Role         string `json:"role"`
		IsActive     bool   `json:"isActive"`
		Deleted      bool   `json:"deleted"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&body); err != nil || body.ID == "" {
		grpcx.WriteError(w, apperr.New(apperr.Validation, ""))
		return
	}
	users := identityv1.NewUserServiceClient(b.up.Identity)
	if body.Deleted {
		if _, err := users.RemoveLegacyUser(ctx, &identityv1.DeleteUserRequest{UserId: body.ID}); err != nil {
			grpcx.WriteError(w, grpcx.FromStatus(err))
			return
		}
		writeJSON(w, map[string]any{"ok": true})
		return
	}
	role := identityv1.UserRole_MEMBER
	if body.Role == "ADMIN" {
		role = identityv1.UserRole_ADMIN
	}
	if _, err := users.SyncLegacyUser(ctx, &identityv1.SyncLegacyUserRequest{Id: body.ID, Email: body.Email, Name: body.Name, PasswordHash: body.PasswordHash, Role: role, IsActive: body.IsActive}); err != nil {
		grpcx.WriteError(w, grpcx.FromStatus(err))
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

// internalSession lets the Next.js server, which owns the browser session,
// trade its shared secret plus a user id for a short-lived access token.
// Identity checks the secret; the gateway only forwards it.
func (b *BrowserRoutes) internalSession(w http.ResponseWriter, r *http.Request) {
	ctx, ok := internalCtx(r)
	if !ok {
		grpcx.WriteError(w, apperr.New(apperr.Unauthorized, ""))
		return
	}
	var body struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&body); err != nil || body.UserID == "" {
		grpcx.WriteError(w, apperr.New(apperr.Validation, ""))
		return
	}
	res, err := identityv1.NewAuthServiceClient(b.up.Identity).IssueServiceToken(ctx, &identityv1.IssueServiceTokenRequest{UserId: body.UserID})
	if err != nil {
		grpcx.WriteError(w, grpcx.FromStatus(err))
		return
	}
	writeJSON(w, map[string]any{"accessToken": res.AccessToken, "expiresAt": res.AccessExpiresAt.AsTime(), "user": map[string]any{"id": res.User.Id, "role": res.User.Role.String(), "permissions": res.User.Permissions}})
}
