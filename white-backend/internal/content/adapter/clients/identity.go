package clients

import (
	"context"
	"errors"
	"time"

	identityv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/identity/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/content/app"
	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ErrNoInternalSecret means AsUser cannot mint tokens.
var ErrNoInternalSecret = errors.New("INTERNAL_API_SECRET kosong: token layanan tidak tersedia")

type Identity struct {
	access  identityv1.AccessServiceClient
	notify  identityv1.NotificationServiceClient
	auth    identityv1.AuthServiceClient
	users   identityv1.UserServiceClient
	secret  string
	acc     *ttlCache[*app.Access]
	clients *ttlCache[map[string]*app.ClientInfo]
	names   *ttlCache[string]
	tokens  *ttlCache[string]
}

// NewIdentity takes the INTERNAL_API_SECRET used for service tokens ("" disables AsUser).
func NewIdentity(conn *grpc.ClientConn, internalSecret string) *Identity {
	return &Identity{
		access:  identityv1.NewAccessServiceClient(conn),
		notify:  identityv1.NewNotificationServiceClient(conn),
		auth:    identityv1.NewAuthServiceClient(conn),
		users:   identityv1.NewUserServiceClient(conn),
		secret:  internalSecret,
		acc:     newCache[*app.Access](30 * time.Second),
		clients: newCache[map[string]*app.ClientInfo](60 * time.Second),
		names:   newCache[string](5 * time.Minute),
		tokens:  newCache[string](10 * time.Minute),
	}
}

func roleFrom(r identityv1.ClientAccess_AccessRole) domain.Role {
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
	a := &app.Access{UserID: userID, Role: roleFrom(res.Role), ClientID: res.Client.GetId(), Slug: res.Client.GetSlug(), Name: res.Client.GetName()}
	c.acc.put(key, a)
	return a, nil
}

// Clients lists every marketing client (internal, no user needed).
func (c *Identity) Clients(ctx context.Context) ([]app.ClientInfo, error) {
	res, err := c.access.ListAllClients(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	out := make([]app.ClientInfo, 0, len(res.Clients))
	for _, cl := range res.Clients {
		out = append(out, app.ClientInfo{ID: cl.Id, Slug: cl.Slug, Name: cl.Name})
	}
	return out, nil
}

func (c *Identity) ClientInfo(ctx context.Context, clientID string) (*app.ClientInfo, error) {
	all, ok := c.clients.get("all")
	if !ok {
		list, err := c.Clients(ctx)
		if err != nil {
			return nil, err
		}
		all = map[string]*app.ClientInfo{}
		for i := range list {
			all[list[i].ID] = &list[i]
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

// UserOptions lists active users; the caller's context must carry a token.
func (c *Identity) UserOptions(ctx context.Context) ([]*identityv1.UserOption, error) {
	res, err := c.users.ListUserOptions(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	return res.Users, nil
}

func (c *Identity) Notify(ctx context.Context, in app.NotifyInput) error {
	aud := identityv1.NotifyRequest_AUDIENCE_UNSPECIFIED
	if in.Audience == app.AudienceClientApprovers {
		aud = identityv1.NotifyRequest_CLIENT_APPROVERS
	}
	_, err := c.notify.Notify(ctx, &identityv1.NotifyRequest{UserIds: in.UserIDs, ClientId: in.ClientID, Audience: aud, ExceptUserId: in.ExceptUserID, Type: in.Type, Title: in.Title, Body: in.Body, Href: in.Href})
	if err != nil {
		return grpcx.FromStatus(err)
	}
	return nil
}

// AsUser mints (and caches for 10 minutes, under the 15 minute token
// lifetime) an access token for userID and returns ctx carrying only it.
func (c *Identity) AsUser(ctx context.Context, userID string) (context.Context, error) {
	if c.secret == "" {
		return nil, ErrNoInternalSecret
	}
	tok, ok := c.tokens.get(userID)
	if !ok {
		res, err := c.auth.IssueServiceToken(metadata.NewOutgoingContext(ctx, metadata.Pairs("x-internal-secret", c.secret)), &identityv1.IssueServiceTokenRequest{UserId: userID})
		if err != nil {
			return nil, grpcx.FromStatus(err)
		}
		tok = res.AccessToken
		c.tokens.put(userID, tok)
	}
	return metadata.NewOutgoingContext(ctx, metadata.Pairs(authn.MetadataHeader, "Bearer "+tok)), nil
}
