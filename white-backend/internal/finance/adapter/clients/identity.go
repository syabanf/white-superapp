package clients

import (
	"context"
	"time"

	identityv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/identity/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

type Identity struct {
	access identityv1.AccessServiceClient
	notify identityv1.NotificationServiceClient
	names  *ttlCache[string]
}

func NewIdentity(conn grpc.ClientConnInterface) *Identity {
	return &Identity{
		access: identityv1.NewAccessServiceClient(conn),
		notify: identityv1.NewNotificationServiceClient(conn),
		names:  newCache[string](5 * time.Minute),
	}
}

func (c *Identity) Notify(ctx context.Context, in app.NotifyInput) error {
	_, err := c.notify.Notify(ctx, &identityv1.NotifyRequest{UserIds: in.UserIDs, Type: in.Type, Title: in.Title, Body: in.Body, Href: in.Href})
	if err != nil {
		return grpcx.FromStatus(err)
	}
	return nil
}

// UserName is "" when the lookup fails.
func (c *Identity) UserName(ctx context.Context, userID string) string {
	if userID == "" {
		return ""
	}
	if name, ok := c.names.get(userID); ok {
		return name
	}
	res, err := c.access.GetUser(ctx, &identityv1.GetUserRequest{UserId: userID})
	if err != nil {
		return ""
	}
	c.names.put(userID, res.Name)
	return res.Name
}

func (c *Identity) PermissionHolders(ctx context.Context, perm string) ([]string, error) {
	res, err := c.access.ListPermissionHolders(ctx, &identityv1.ListPermissionHoldersRequest{Permission: perm})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	ids := make([]string, 0, len(res.Users))
	for _, u := range res.Users {
		ids = append(ids, u.Id)
	}
	return ids, nil
}
