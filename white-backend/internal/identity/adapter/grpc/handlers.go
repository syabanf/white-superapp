package grpcapi

import (
	"context"

	identityv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/identity/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/identity/app"
	"github.com/syabanf/white-superapp/white-backend/internal/identity/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Services struct {
	Auth          *app.Auth
	Users         *app.Users
	Clients       *app.Clients
	Notifications *app.Notifications
	Share         *app.Share
	Workspace     *app.Workspace
}

func Register(s *grpc.Server, svc Services) {
	identityv1.RegisterAuthServiceServer(s, &authServer{svc: svc})
	identityv1.RegisterUserServiceServer(s, &userServer{svc: svc})
	identityv1.RegisterClientServiceServer(s, &clientServer{svc: svc})
	identityv1.RegisterAccessServiceServer(s, &accessServer{svc: svc})
	identityv1.RegisterNotificationServiceServer(s, &notificationServer{svc: svc})
	identityv1.RegisterShareServiceServer(s, &shareServer{svc: svc})
	identityv1.RegisterWorkspaceServiceServer(s, &workspaceServer{svc: svc})
}

func userAgent(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get("user-agent"); len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

// ─── AuthService ─────────────────────────────────────────────

type authServer struct {
	identityv1.UnimplementedAuthServiceServer
	svc Services
}

func (s *authServer) Login(ctx context.Context, req *identityv1.LoginRequest) (*identityv1.TokenPair, error) {
	t, err := s.svc.Auth.Login(ctx, req.Email, req.Password, userAgent(ctx), grpcx.RemoteIP(ctx))
	if err != nil {
		return nil, err
	}
	return tokensPB(t), nil
}

func (s *authServer) Refresh(ctx context.Context, req *identityv1.RefreshRequest) (*identityv1.TokenPair, error) {
	t, err := s.svc.Auth.Refresh(ctx, req.RefreshToken, userAgent(ctx), grpcx.RemoteIP(ctx))
	if err != nil {
		return nil, err
	}
	return tokensPB(t), nil
}

func (s *authServer) Logout(ctx context.Context, req *identityv1.LogoutRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.svc.Auth.Logout(ctx, req.RefreshToken)
}

func (s *authServer) Me(ctx context.Context, _ *emptypb.Empty) (*identityv1.User, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	u, err := s.svc.Auth.Me(ctx, p.UserID)
	if err != nil {
		return nil, err
	}
	return userPB(u), nil
}

// ─── UserService ─────────────────────────────────────────────

type userServer struct {
	identityv1.UnimplementedUserServiceServer
	svc Services
}

func userInput(name, email, pw string, role identityv1.UserRole, active bool, ms []*identityv1.Membership) app.UserInput {
	return app.UserInput{Name: name, Email: email, Password: pw, Role: userRoleFrom(role), IsActive: active, Memberships: membershipsFrom(ms)}
}

func (s *userServer) ListUsers(ctx context.Context, _ *emptypb.Empty) (*identityv1.ListUsersResponse, error) {
	if _, err := authn.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Users.List(ctx)
	if err != nil {
		return nil, err
	}
	out := &identityv1.ListUsersResponse{}
	for _, r := range rows {
		u := &identityv1.UserWithClients{User: userPB(&r.User)}
		for _, c := range r.Clients {
			u.Clients = append(u.Clients, &identityv1.UserWithClients_UserClient{Id: c.ClientID, Name: c.ClientName, Slug: c.ClientSlug, Role: memberRole(c.Role)})
		}
		out.Users = append(out.Users, u)
	}
	return out, nil
}

func (s *userServer) ListUserOptions(ctx context.Context, _ *emptypb.Empty) (*identityv1.ListUserOptionsResponse, error) {
	if _, err := authn.Require(ctx); err != nil {
		return nil, err
	}
	users, err := s.svc.Users.ListOptions(ctx)
	if err != nil {
		return nil, err
	}
	out := &identityv1.ListUserOptionsResponse{}
	for _, u := range users {
		out.Users = append(out.Users, &identityv1.UserOption{Id: u.ID, Name: u.Name, Email: u.Email})
	}
	return out, nil
}

func (s *userServer) CreateUser(ctx context.Context, req *identityv1.CreateUserRequest) (*identityv1.User, error) {
	if _, err := authn.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	u, err := s.svc.Users.Create(ctx, userInput(req.Name, req.Email, req.Password, req.Role, req.IsActive, req.Memberships))
	if err != nil {
		return nil, err
	}
	return userPB(u), nil
}

func (s *userServer) UpdateUser(ctx context.Context, req *identityv1.UpdateUserRequest) (*identityv1.User, error) {
	p, err := authn.RequireAdmin(ctx)
	if err != nil {
		return nil, err
	}
	u, err := s.svc.Users.Update(ctx, p.UserID, req.UserId, userInput(req.Name, req.Email, req.Password, req.Role, req.IsActive, req.Memberships))
	if err != nil {
		return nil, err
	}
	return userPB(u), nil
}

func (s *userServer) SetUserActive(ctx context.Context, req *identityv1.SetUserActiveRequest) (*identityv1.User, error) {
	p, err := authn.RequireAdmin(ctx)
	if err != nil {
		return nil, err
	}
	u, err := s.svc.Users.SetActive(ctx, p.UserID, req.UserId, req.IsActive)
	if err != nil {
		return nil, err
	}
	return userPB(u), nil
}

func (s *userServer) DeleteUser(ctx context.Context, req *identityv1.DeleteUserRequest) (*emptypb.Empty, error) {
	p, err := authn.RequireAdmin(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.svc.Users.Delete(ctx, p.UserID, req.UserId)
}

// ─── ClientService ───────────────────────────────────────────

type clientServer struct {
	identityv1.UnimplementedClientServiceServer
	svc Services
}

func (s *clientServer) ListClients(ctx context.Context, _ *emptypb.Empty) (*identityv1.ListClientsResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.svc.Clients.List(ctx, p.UserID, p.IsAdmin())
	if err != nil {
		return nil, err
	}
	out := &identityv1.ListClientsResponse{}
	for _, r := range rows {
		out.Clients = append(out.Clients, &identityv1.ClientSummary{Client: clientPB(&r.Client), MembersCount: r.MembersCount, MyRole: accessRole(r.MyRole)})
	}
	return out, nil
}

func (s *clientServer) GetClient(ctx context.Context, req *identityv1.GetClientRequest) (*identityv1.ClientAccess, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	a, err := s.svc.Clients.Access(ctx, p.UserID, p.IsAdmin(), "", req.Slug)
	if err != nil {
		return nil, err
	}
	return accessPB(a), nil
}

func (s *clientServer) CreateClient(ctx context.Context, req *identityv1.CreateClientRequest) (*identityv1.Client, error) {
	p, err := authn.RequireAdmin(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Clients.Create(ctx, p.UserID, profileFrom(req.Profile), modulesFrom(req.Modules))
	if err != nil {
		return nil, err
	}
	return clientPB(c), nil
}

func (s *clientServer) UpdateClient(ctx context.Context, req *identityv1.UpdateClientRequest) (*identityv1.Client, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Clients.Update(ctx, p.UserID, p.IsAdmin(), req.ClientId, profileFrom(req.Profile))
	if err != nil {
		return nil, err
	}
	return clientPB(c), nil
}

func (s *clientServer) DeleteClient(ctx context.Context, req *identityv1.DeleteClientRequest) (*emptypb.Empty, error) {
	if _, err := authn.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.svc.Clients.Delete(ctx, req.ClientId)
}

func (s *clientServer) ListMembers(ctx context.Context, req *identityv1.ListMembersRequest) (*identityv1.ListMembersResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.svc.Clients.ListMembers(ctx, p.UserID, p.IsAdmin(), req.ClientId)
	if err != nil {
		return nil, err
	}
	out := &identityv1.ListMembersResponse{}
	for _, m := range rows {
		out.Members = append(out.Members, &identityv1.ClientMember{UserId: m.UserID, Name: m.Name, Email: m.Email, IsActive: m.IsActive, Role: memberRole(m.Role), CreatedAt: ts(m.CreatedAt)})
	}
	return out, nil
}

func (s *clientServer) UpsertMember(ctx context.Context, req *identityv1.UpsertMemberRequest) (*emptypb.Empty, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.svc.Clients.UpsertMember(ctx, p.UserID, p.IsAdmin(), req.ClientId, req.UserId, memberRoleFrom(req.Role))
}

func (s *clientServer) RemoveMember(ctx context.Context, req *identityv1.RemoveMemberRequest) (*emptypb.Empty, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.svc.Clients.RemoveMember(ctx, p.UserID, p.IsAdmin(), req.ClientId, req.UserId)
}

func (s *clientServer) UpdateShareSettings(ctx context.Context, req *identityv1.UpdateShareSettingsRequest) (*identityv1.Client, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Clients.UpdateShare(ctx, p.UserID, p.IsAdmin(), req.ClientId, req.ShareEnabled, req.Pin, modulesFrom(req.Modules))
	if err != nil {
		return nil, err
	}
	return clientPB(c), nil
}

// ─── AccessService (internal) ────────────────────────────────

type accessServer struct {
	identityv1.UnimplementedAccessServiceServer
	svc Services
}

func (s *accessServer) CheckClientAccess(ctx context.Context, req *identityv1.CheckClientAccessRequest) (*identityv1.ClientAccess, error) {
	u, err := s.svc.Auth.Me(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	a, err := s.svc.Clients.Access(ctx, u.ID, u.Role == domain.RoleAdmin, req.GetClientId(), req.GetSlug())
	if err != nil {
		return nil, err
	}
	return accessPB(a), nil
}

func (s *accessServer) GetUser(ctx context.Context, req *identityv1.GetUserRequest) (*identityv1.User, error) {
	u, err := s.svc.Auth.Me(ctx, req.UserId)
	if err != nil {
		if apperr.Is(err, apperr.Unauthenticated) {
			return nil, apperr.New(apperr.NotFound, "")
		}
		return nil, err
	}
	return userPB(u), nil
}

func (s *accessServer) ListAccessibleClientIds(ctx context.Context, req *identityv1.ListAccessibleClientIdsRequest) (*identityv1.ListAccessibleClientIdsResponse, error) {
	u, err := s.svc.Auth.Me(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	ids, err := s.svc.Clients.ListAccessibleIDs(ctx, u.ID, u.Role == domain.RoleAdmin)
	if err != nil {
		return nil, err
	}
	return &identityv1.ListAccessibleClientIdsResponse{ClientIds: ids}, nil
}

func (s *accessServer) ListAllClients(ctx context.Context, _ *emptypb.Empty) (*identityv1.ListAllClientsResponse, error) {
	rows, err := s.svc.Clients.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	out := &identityv1.ListAllClientsResponse{}
	for i := range rows {
		out.Clients = append(out.Clients, clientPB(&rows[i]))
	}
	return out, nil
}

// ─── NotificationService ─────────────────────────────────────

type notificationServer struct {
	identityv1.UnimplementedNotificationServiceServer
	svc Services
}

func (s *notificationServer) ListNotifications(ctx context.Context, req *identityv1.ListNotificationsRequest) (*identityv1.NotificationFeed, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	items, unread, err := s.svc.Notifications.Feed(ctx, p.UserID, req.Limit)
	if err != nil {
		return nil, err
	}
	out := &identityv1.NotificationFeed{Unread: unread}
	for _, n := range items {
		out.Items = append(out.Items, &identityv1.Notification{Id: n.ID, Type: n.Type, Title: n.Title, Body: n.Body, Href: n.Href, Read: n.ReadAt != nil, CreatedAt: ts(n.CreatedAt)})
	}
	return out, nil
}

func (s *notificationServer) MarkNotificationRead(ctx context.Context, req *identityv1.MarkNotificationReadRequest) (*emptypb.Empty, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.svc.Notifications.MarkRead(ctx, p.UserID, req.NotificationId)
}

func (s *notificationServer) MarkAllNotificationsRead(ctx context.Context, _ *emptypb.Empty) (*identityv1.MarkAllNotificationsReadResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	n, err := s.svc.Notifications.MarkAllRead(ctx, p.UserID)
	if err != nil {
		return nil, err
	}
	return &identityv1.MarkAllNotificationsReadResponse{Count: n}, nil
}

func (s *notificationServer) Notify(ctx context.Context, req *identityv1.NotifyRequest) (*identityv1.NotifyResponse, error) {
	aud := app.AudienceNone
	switch req.Audience {
	case identityv1.NotifyRequest_CLIENT_ALL:
		aud = app.AudienceClientAll
	case identityv1.NotifyRequest_CLIENT_APPROVERS:
		aud = app.AudienceClientApprovers
	}
	n, err := s.svc.Notifications.Notify(ctx, app.NotifyInput{
		UserIDs: req.UserIds, ClientID: req.ClientId, Audience: aud, ExceptUserID: req.ExceptUserId,
		Type: req.Type, Title: req.Title, Body: req.Body, Href: req.Href,
	})
	if err != nil {
		return nil, err
	}
	return &identityv1.NotifyResponse{Count: n}, nil
}

// ─── ShareService ────────────────────────────────────────────

type shareServer struct {
	identityv1.UnimplementedShareServiceServer
	svc Services
}

func (s *shareServer) GetShareContext(ctx context.Context, req *identityv1.GetShareContextRequest) (*identityv1.ShareContext, error) {
	sc, err := s.svc.Share.Context(ctx, req.Slug, req.UnlockToken)
	if err != nil {
		return nil, err
	}
	out := &identityv1.ShareContext{}
	switch sc.Status {
	case app.ShareOK:
		out.Status = identityv1.ShareContext_OK
	case app.ShareLocked:
		out.Status = identityv1.ShareContext_LOCKED
	case app.ShareDisabled:
		out.Status = identityv1.ShareContext_DISABLED
	default:
		out.Status = identityv1.ShareContext_NOT_FOUND
	}
	if sc.Client != nil {
		out.Client = clientPB(sc.Client)
	}
	return out, nil
}

func (s *shareServer) VerifySharePin(ctx context.Context, req *identityv1.VerifySharePinRequest) (*identityv1.VerifySharePinResponse, error) {
	ip := req.RemoteIp
	if ip == "" {
		ip = grpcx.RemoteIP(ctx)
	}
	clientID, token, err := s.svc.Share.VerifyPin(ctx, req.Slug, req.Pin, ip)
	if err != nil {
		return nil, err
	}
	return &identityv1.VerifySharePinResponse{ClientId: clientID, UnlockToken: token, MaxAgeSeconds: int32(app.ShareCookieMaxAge.Seconds())}, nil
}

// ─── WorkspaceService ────────────────────────────────────────

type workspaceServer struct {
	identityv1.UnimplementedWorkspaceServiceServer
	svc Services
}

func setupPB(st *app.SetupState) *identityv1.SetupState {
	return &identityv1.SetupState{CompletedAt: tsPtr(st.CompletedAt), SkippedAt: tsPtr(st.SkippedAt), Clients: st.Clients}
}

func (s *workspaceServer) GetSetupState(ctx context.Context, _ *emptypb.Empty) (*identityv1.SetupState, error) {
	if _, err := authn.Require(ctx); err != nil {
		return nil, err
	}
	st, err := s.svc.Workspace.SetupState(ctx)
	if err != nil {
		return nil, err
	}
	return setupPB(st), nil
}

func (s *workspaceServer) MarkSetup(ctx context.Context, req *identityv1.MarkSetupRequest) (*identityv1.SetupState, error) {
	if _, err := authn.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	if req.Kind == identityv1.MarkSetupRequest_KIND_UNSPECIFIED {
		return nil, apperr.New(apperr.Validation, "")
	}
	st, err := s.svc.Workspace.MarkSetup(ctx, req.Kind == identityv1.MarkSetupRequest_COMPLETED)
	if err != nil {
		return nil, err
	}
	return setupPB(st), nil
}
