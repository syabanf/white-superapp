package grpcapi

import (
	"context"

	integrationsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/integrations/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/integrations/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Services struct {
	Settings    *app.Settings
	Connections *app.Connections
}

func Register(s *grpc.Server, svc Services) {
	integrationsv1.RegisterConnectionServiceServer(s, &connectionServer{svc: svc})
	integrationsv1.RegisterSettingsServiceServer(s, &settingsServer{svc: svc})
}

// ─── ConnectionService ───────────────────────────────────────

type connectionServer struct {
	integrationsv1.UnimplementedConnectionServiceServer
	svc Services
}

func (s *connectionServer) ListConnections(ctx context.Context, req *integrationsv1.ListConnectionsRequest) (*integrationsv1.ListConnectionsResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.svc.Connections.List(ctx, p.UserID, req.ClientId)
	if err != nil {
		return nil, err
	}
	out := &integrationsv1.ListConnectionsResponse{}
	for i := range rows {
		out.Connections = append(out.Connections, connectionPB(&rows[i]))
	}
	return out, nil
}

func (s *connectionServer) DisconnectConnection(ctx context.Context, req *integrationsv1.DisconnectConnectionRequest) (*emptypb.Empty, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.svc.Connections.Disconnect(ctx, p.UserID, req.ClientId, req.ConnectionId)
}

func (s *connectionServer) SimulateConnection(ctx context.Context, req *integrationsv1.SimulateConnectionRequest) (*integrationsv1.Connection, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Connections.Simulate(ctx, p.UserID, req.ClientId, providerFrom(req.Provider))
	if err != nil {
		return nil, err
	}
	return connectionPB(c), nil
}

func (s *connectionServer) StartOAuth(ctx context.Context, req *integrationsv1.StartOAuthRequest) (*integrationsv1.StartOAuthResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.Connections.StartOAuth(ctx, p.UserID, req.ClientId, providerFrom(req.Provider), req.AppBaseUrl)
	if err != nil {
		return nil, err
	}
	outcome := integrationsv1.StartOAuthResponse_NOT_CONFIGURED
	if r.Configured {
		outcome = integrationsv1.StartOAuthResponse_REDIRECT_TO_PROVIDER
	}
	return &integrationsv1.StartOAuthResponse{Outcome: outcome, RedirectUrl: r.RedirectURL, Nonce: r.Nonce}, nil
}

// CompleteOAuth is called by the gateway's callback handler. The browser
// arrives from the provider with cookies only, so the gateway resolves the
// user and passes user_id; a bearer principal, when present, wins.
func (s *connectionServer) CompleteOAuth(ctx context.Context, req *integrationsv1.CompleteOAuthRequest) (*integrationsv1.CompleteOAuthResponse, error) {
	userID := req.UserId
	if p, ok := authn.FromContext(ctx); ok {
		userID = p.UserID
	}
	r, err := s.svc.Connections.CompleteOAuth(ctx, app.CompleteOAuthInput{
		Provider: providerFrom(req.Provider), Code: req.Code, State: req.State, ProviderError: req.ProviderError,
		CookieNonce: req.CookieNonce, AppBaseURL: req.AppBaseUrl, UserID: userID,
	})
	if err != nil {
		return nil, err
	}
	return &integrationsv1.CompleteOAuthResponse{RedirectUrl: r.RedirectURL, Connection: connectionPB(r.Connection)}, nil
}

func (s *connectionServer) GetAccessToken(ctx context.Context, req *integrationsv1.GetAccessTokenRequest) (*integrationsv1.AccessToken, error) {
	t, err := s.svc.Connections.AccessToken(ctx, req.ConnectionId)
	if err != nil {
		return nil, err
	}
	return &integrationsv1.AccessToken{ConnectionId: t.ConnectionID, Token: t.Token, Provider: providerPB(t.Provider)}, nil
}

func (s *connectionServer) FindClientConnection(ctx context.Context, req *integrationsv1.FindClientConnectionRequest) (*integrationsv1.FindClientConnectionResponse, error) {
	c, err := s.svc.Connections.Find(ctx, req.ClientId, providerFrom(req.Provider))
	if err != nil {
		return nil, err
	}
	return &integrationsv1.FindClientConnectionResponse{Connection: connectionPB(c)}, nil
}

// ─── SettingsService (ADMIN) ─────────────────────────────────

type settingsServer struct {
	integrationsv1.UnimplementedSettingsServiceServer
	svc Services
}

func (s *settingsServer) ListIntegrationStatuses(ctx context.Context, _ *emptypb.Empty) (*integrationsv1.ListIntegrationStatusesResponse, error) {
	if _, err := authn.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	ov, err := s.svc.Settings.ListStatuses(ctx)
	if err != nil {
		return nil, err
	}
	out := &integrationsv1.ListIntegrationStatusesResponse{MetaReview: reviewPB(ov.MetaReview)}
	for _, st := range ov.Statuses {
		out.Statuses = append(out.Statuses, statusPB(st))
	}
	return out, nil
}

func (s *settingsServer) SaveCredential(ctx context.Context, req *integrationsv1.SaveCredentialRequest) (*integrationsv1.SaveCredentialResponse, error) {
	p, err := authn.RequireAdmin(ctx)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.Settings.SaveCredential(ctx, p.UserID, app.SaveCredentialInput{Key: keyFrom(req.Key), PublicID: req.PublicId, Secret: req.Secret, Verify: req.Verify})
	if err != nil {
		return nil, err
	}
	return &integrationsv1.SaveCredentialResponse{Status: statusPB(r.Status), Verified: verifyPB(r.Verified)}, nil
}

func (s *settingsServer) ClearCredential(ctx context.Context, req *integrationsv1.ClearCredentialRequest) (*emptypb.Empty, error) {
	if _, err := authn.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.svc.Settings.ClearCredential(ctx, keyFrom(req.Key))
}

func (s *settingsServer) SaveMetaReview(ctx context.Context, req *integrationsv1.SaveMetaReviewRequest) (*integrationsv1.IntegrationStatus, error) {
	p, err := authn.RequireAdmin(ctx)
	if err != nil {
		return nil, err
	}
	st, err := s.svc.Settings.SaveMetaReview(ctx, p.UserID, app.MetaReviewInput{Permissions: req.Permissions, AppLive: req.AppLive, BusinessVerified: req.BusinessVerified})
	if err != nil {
		return nil, err
	}
	return statusPB(*st), nil
}

func (s *settingsServer) EstimateCosts(ctx context.Context, req *integrationsv1.EstimateCostsRequest) (*integrationsv1.EstimateCostsResponse, error) {
	if _, err := authn.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	e := s.svc.Settings.EstimateCosts(costInput(req))
	return &integrationsv1.EstimateCostsResponse{Dataforseo: estimatePB(e.DataForSeo), Apify: estimatePB(e.Apify)}, nil
}

func (s *settingsServer) GetProviderCredentials(ctx context.Context, req *integrationsv1.GetProviderCredentialsRequest) (*integrationsv1.ProviderCredentials, error) {
	c, err := s.svc.Settings.Credentials(ctx, keyFrom(req.Key))
	if err != nil {
		return nil, err
	}
	return &integrationsv1.ProviderCredentials{Key: keyPB(c.Key), Configured: c.Configured, PublicId: c.PublicID, Secret: c.Secret}, nil
}
