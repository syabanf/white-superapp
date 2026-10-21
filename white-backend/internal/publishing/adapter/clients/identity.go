package clients

import (
	"context"
	"time"

	identityv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/identity/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/publishing/app"
	"github.com/syabanf/white-superapp/white-backend/internal/publishing/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Identity struct {
	access  identityv1.AccessServiceClient
	notify  identityv1.NotificationServiceClient
	acc     *ttlCache[*app.Access]
	clients *ttlCache[map[string]*app.ClientInfo]
	names   *ttlCache[string]
}

func NewIdentity(conn *grpc.ClientConn) *Identity {
	return &Identity{
		access:  identityv1.NewAccessServiceClient(conn),
		notify:  identityv1.NewNotificationServiceClient(conn),
		acc:     newCache[*app.Access](30 * time.Second),
		clients: newCache[map[string]*app.ClientInfo](60 * time.Second),
		names:   newCache[string](5 * time.Minute),
	}
}

func roleFrom(r identityv1.ClientAccess_AccessRole) domain.PostRole {
	switch r {
	case identityv1.ClientAccess_ADMIN_OVERRIDE:
		return domain.RoleAdmin
	case identityv1.ClientAccess_MANAGER:
		return domain.RoleManager
	}
	return domain.RoleViewer
}

func (c *Identity) CheckAccess(ctx context.Context, userID, clientID string) (*app.Access, error) {
	key := userID + "|" + clientID
	if a, ok := c.acc.get(key); ok {
		return a, nil
	}
	res, err := c.access.CheckClientAccess(ctx, &identityv1.CheckClientAccessRequest{UserId: userID, Client: &identityv1.CheckClientAccessRequest_ClientId{ClientId: clientID}})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	a := &app.Access{UserID: userID, Role: roleFrom(res.Role), ClientID: res.Client.GetId(), Slug: res.Client.GetSlug(), Timezone: res.Client.GetTimezone()}
	c.acc.put(key, a)
	return a, nil
}

func (c *Identity) ClientInfo(ctx context.Context, clientID string) (*app.ClientInfo, error) {
	all, ok := c.clients.get("all")
	if !ok {
		res, err := c.access.ListAllClients(ctx, &emptypb.Empty{})
		if err != nil {
			return nil, grpcx.FromStatus(err)
		}
		all = map[string]*app.ClientInfo{}
		for _, cl := range res.Clients {
			all[cl.Id] = &app.ClientInfo{ID: cl.Id, Slug: cl.Slug, Timezone: cl.Timezone}
		}
		c.clients.put("all", all)
	}
	return all[clientID], nil
}

func (c *Identity) UserName(ctx context.Context, userID string) string {
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

func (c *Identity) Notify(ctx context.Context, in app.NotifyInput) error {
	aud := identityv1.NotifyRequest_AUDIENCE_UNSPECIFIED
	switch in.Audience {
	case app.AudienceClientAll:
		aud = identityv1.NotifyRequest_CLIENT_ALL
	case app.AudienceClientApprovers:
		aud = identityv1.NotifyRequest_CLIENT_APPROVERS
	}
	_, err := c.notify.Notify(ctx, &identityv1.NotifyRequest{UserIds: in.UserIDs, ClientId: in.ClientID, Audience: aud, ExceptUserId: in.ExceptUserID, Type: in.Type, Title: in.Title, Body: in.Body, Href: in.Href})
	if err != nil {
		return grpcx.FromStatus(err)
	}
	return nil
}
