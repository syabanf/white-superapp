// Package clients wraps the identity gRPC client behind the hr app ports.
package clients

import (
	"context"
	"sync"
	"time"

	identityv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/identity/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Identity struct {
	access identityv1.AccessServiceClient
	notify identityv1.NotificationServiceClient
	users  identityv1.UserServiceClient

	mu      sync.Mutex
	holders map[string]cachedIDs
}

type cachedIDs struct {
	ids []string
	at  time.Time
}

// holdersTTL bounds how stale a permission audience may be.
const holdersTTL = time.Minute

func NewIdentity(conn *grpc.ClientConn) *Identity {
	return &Identity{
		access:  identityv1.NewAccessServiceClient(conn),
		notify:  identityv1.NewNotificationServiceClient(conn),
		users:   identityv1.NewUserServiceClient(conn),
		holders: map[string]cachedIDs{},
	}
}

// PermissionHolders is AccessService.ListPermissionHolders, cached a minute
// per permission.
func (c *Identity) PermissionHolders(ctx context.Context, perm string) ([]string, error) {
	c.mu.Lock()
	hit, ok := c.holders[perm]
	c.mu.Unlock()
	if ok && time.Since(hit.at) < holdersTTL {
		return hit.ids, nil
	}
	res, err := c.access.ListPermissionHolders(ctx, &identityv1.ListPermissionHoldersRequest{Permission: perm})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	ids := make([]string, 0, len(res.GetUsers()))
	for _, u := range res.GetUsers() {
		ids = append(ids, u.GetId())
	}
	c.mu.Lock()
	c.holders[perm] = cachedIDs{ids: ids, at: time.Now()}
	c.mu.Unlock()
	return ids, nil
}

// GetUser is AccessService.GetUser (internal); nil when the user is gone.
func (c *Identity) GetUser(ctx context.Context, userID string) (*app.UserInfo, error) {
	u, err := c.access.GetUser(ctx, &identityv1.GetUserRequest{UserId: userID})
	if err != nil {
		if e := grpcx.FromStatus(err); e.Code == apperr.NotFound {
			return nil, nil
		}
		return nil, grpcx.FromStatus(err)
	}
	return &app.UserInfo{ID: u.Id, Name: u.Name, Email: u.Email, IsActive: u.IsActive}, nil
}

func (c *Identity) Notify(ctx context.Context, in app.NotifyInput) error {
	if len(in.UserIDs) == 0 {
		return nil
	}
	_, err := c.notify.Notify(ctx, &identityv1.NotifyRequest{UserIds: in.UserIDs, Type: in.Type, Title: in.Title, Body: in.Body, Href: in.Href})
	if err != nil {
		return grpcx.FromStatus(err)
	}
	return nil
}

// UserOptions is UserService.ListUserOptions; the context must carry a
// bearer token (the seeder mints one).
func (c *Identity) UserOptions(ctx context.Context) ([]app.UserInfo, error) {
	res, err := c.users.ListUserOptions(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	out := make([]app.UserInfo, len(res.Users))
	for i, u := range res.Users {
		out[i] = app.UserInfo{ID: u.Id, Name: u.Name, Email: u.Email, IsActive: true}
	}
	return out, nil
}
