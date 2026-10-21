package app

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/integrations/adapter/providers/googleoauth"
	"github.com/syabanf/white-superapp/white-backend/internal/integrations/adapter/providers/metaoauth"
	"github.com/syabanf/white-superapp/white-backend/internal/integrations/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/crypto"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/providers/google"
	"github.com/syabanf/white-superapp/white-backend/pkg/providers/meta"
)

const (
	msgTokenExpired = "Token koneksi kedaluwarsa, hubungkan ulang di Pengaturan Klien."
	demoDisplayName = "Akun demo"
	demoTokenTTL    = 60 * 24 * time.Hour
)

// AccessChecker resolves what a user may do with a client (identity service).
// Exactly one of clientID or slug is set.
type AccessChecker interface {
	Check(ctx context.Context, userID, clientID, slug string) (*domain.ClientAccess, error)
}

// CredentialSource returns the effective provider-app credentials.
type CredentialSource interface {
	Credentials(ctx context.Context, key domain.IntegrationKey) (domain.ProviderCredentials, error)
}

type GoogleOAuth interface {
	ExchangeCode(ctx context.Context, clientID, clientSecret, redirectURI, code string) (*googleoauth.Tokens, error)
	Userinfo(ctx context.Context, accessToken string) (*googleoauth.Userinfo, error)
	Refresh(ctx context.Context, clientID, clientSecret, refreshToken string) (*googleoauth.Refreshed, error)
}

type MetaOAuth interface {
	ExchangeCode(ctx context.Context, appID, appSecret, redirectURI, code string) (*metaoauth.Token, error)
	ExchangeLongLived(ctx context.Context, appID, appSecret, accessToken string) (*metaoauth.Token, error)
	Me(ctx context.Context, accessToken string) (*metaoauth.Me, error)
}

type ConnectionsDeps struct {
	Store  domain.Store
	Cipher *crypto.Cipher
	Creds  CredentialSource
	Access AccessChecker
	Google GoogleOAuth
	Meta   MetaOAuth
	Bus    events.Publisher
	Clock  clock.Clock
	// StateSecret signs the OAuth state (AUTH_SECRET).
	StateSecret string
	// AppBaseURL is the fallback when a request carries no base URL.
	AppBaseURL string
	Logger     *slog.Logger
}

type Connections struct {
	d ConnectionsDeps
}

func NewConnections(d ConnectionsDeps) *Connections {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	return &Connections{d: d}
}

func (s *Connections) List(ctx context.Context, userID, clientID string) ([]domain.Connection, error) {
	if _, err := s.d.Access.Check(ctx, userID, clientID, ""); err != nil {
		return nil, err
	}
	return s.d.Store.ListConnections(ctx, clientID)
}

// manage is the ADMIN-or-MANAGER gate. Like the old manageContext it answers
// UNAUTHORIZED for an unknown client.
func (s *Connections) manage(ctx context.Context, userID, clientID, slug string) (*domain.ClientAccess, error) {
	acc, err := s.d.Access.Check(ctx, userID, clientID, slug)
	if err != nil {
		if apperr.Is(err, apperr.NotFound) {
			return nil, apperr.New(apperr.Unauthorized, "")
		}
		return nil, err
	}
	if !acc.CanManage {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	return acc, nil
}

func (s *Connections) Disconnect(ctx context.Context, userID, clientID, connectionID string) error {
	if _, err := s.manage(ctx, userID, clientID, ""); err != nil {
		return err
	}
	c, err := s.d.Store.GetConnection(ctx, connectionID)
	if err != nil {
		return err
	}
	if c == nil || c.ClientID != clientID {
		return apperr.New(apperr.NotFound, "")
	}
	if _, err := s.d.Store.DeleteConnection(ctx, connectionID); err != nil {
		return err
	}
	return s.d.Bus.Publish(ctx, events.ConnectionDeleted, events.ConnectionDeletedPayload{ClientID: clientID, ConnectionID: connectionID})
}

// Simulate stores a mock token so sync paths behave as "connected" (demo mode).
func (s *Connections) Simulate(ctx context.Context, userID, clientID string, provider domain.Provider) (*domain.Connection, error) {
	if _, err := s.manage(ctx, userID, clientID, ""); err != nil {
		return nil, err
	}
	if !provider.Valid() {
		return nil, apperr.New(apperr.Validation, "")
	}
	enc, err := s.d.Cipher.Encrypt("mock")
	if err != nil {
		return nil, err
	}
	exp := s.d.Clock.Now().Add(demoTokenTTL)
	return s.d.Store.UpsertConnection(ctx, domain.Connection{
		ID: ids.New(), ClientID: clientID, Provider: provider, AccountID: "demo-" + provider.Slug(),
		DisplayName: demoDisplayName, AccessTokenEnc: enc, ExpiresAt: &exp, Scopes: scopesFor(provider),
	}, false)
}

func scopesFor(p domain.Provider) []string {
	if p == domain.ProviderMeta {
		return append([]string(nil), meta.Scopes...)
	}
	return append([]string(nil), google.Scopes...)
}

// ─── OAuth ───────────────────────────────────────────────────

type OAuthStart struct {
	// Configured is false when the provider app has no credentials; the
	// redirect then points at the settings page in demo mode.
	Configured  bool
	RedirectURL string
	Nonce       string
}

func (s *Connections) baseURL(appBaseURL string) string {
	b := strings.TrimRight(strings.TrimSpace(appBaseURL), "/")
	if b == "" {
		b = strings.TrimRight(s.d.AppBaseURL, "/")
	}
	return b
}

func settingsURL(base, slug string) string { return base + "/clients/" + slug + "/settings" }

func callbackURL(base string, p domain.Provider) string {
	return base + "/api/connections/" + p.Slug() + "/callback"
}

func (s *Connections) StartOAuth(ctx context.Context, userID, clientID string, provider domain.Provider, appBaseURL string) (*OAuthStart, error) {
	if !provider.Valid() {
		return nil, apperr.New(apperr.Validation, "")
	}
	acc, err := s.manage(ctx, userID, clientID, "")
	if err != nil {
		return nil, err
	}
	base := s.baseURL(appBaseURL)
	creds, err := s.d.Creds.Credentials(ctx, provider.IntegrationKey())
	if err != nil {
		return nil, err
	}
	if !creds.Configured {
		return &OAuthStart{RedirectURL: settingsURL(base, acc.Slug) + "?demo=1&provider=" + provider.Slug()}, nil
	}
	nonce := domain.RandomNonce()
	state, err := domain.SignOAuthState(acc.Slug, nonce, s.d.StateSecret, s.d.Clock.Now())
	if err != nil {
		return nil, err
	}
	redirectURI := callbackURL(base, provider)
	var consent string
	if provider == domain.ProviderMeta {
		consent = metaoauth.AuthURL(creds.PublicID, redirectURI, state)
	} else {
		consent = googleoauth.AuthURL(creds.PublicID, redirectURI, state)
	}
	return &OAuthStart{Configured: true, RedirectURL: consent, Nonce: nonce}, nil
}

type CompleteOAuthInput struct {
	Provider      domain.Provider
	Code          string
	State         string
	ProviderError string
	CookieNonce   string
	AppBaseURL    string
	UserID        string
}

type OAuthResult struct {
	RedirectURL string
	Connection  *domain.Connection
}

// CompleteOAuth verifies the state, exchanges the code and upserts the
// connection. Every failure is reported through the redirect URL, the way
// the old callback routes did; only infrastructure errors are returned.
func (s *Connections) CompleteOAuth(ctx context.Context, in CompleteOAuthInput) (*OAuthResult, error) {
	base := s.baseURL(in.AppBaseURL)
	missing := &OAuthResult{RedirectURL: base + "/clients?missing=1"}
	if !in.Provider.Valid() {
		return missing, nil
	}
	st := domain.VerifyOAuthState(in.State, s.d.StateSecret, s.d.Clock.Now())
	if st == nil {
		return missing, nil
	}
	settings := settingsURL(base, st.Slug)
	if in.CookieNonce == "" || in.CookieNonce != st.Nonce {
		return &OAuthResult{RedirectURL: settings + "?error=oauth_state"}, nil
	}
	if in.UserID == "" {
		return &OAuthResult{RedirectURL: base + "/login"}, nil
	}
	acc, err := s.manage(ctx, in.UserID, "", st.Slug)
	if err != nil {
		if apperr.As(err).Code == apperr.Unauthorized {
			return missing, nil
		}
		return nil, err
	}
	creds, err := s.d.Creds.Credentials(ctx, in.Provider.IntegrationKey())
	if err != nil {
		return nil, err
	}
	if !creds.Configured {
		return &OAuthResult{RedirectURL: settings + "?demo=1&provider=" + in.Provider.Slug()}, nil
	}
	if in.ProviderError != "" || in.Code == "" {
		return &OAuthResult{RedirectURL: settings + "?error=oauth_denied"}, nil
	}

	var conn *domain.Connection
	redirectURI := callbackURL(base, in.Provider)
	if in.Provider == domain.ProviderMeta {
		conn, err = s.connectMeta(ctx, acc.ClientID, creds, redirectURI, in.Code)
	} else {
		conn, err = s.connectGoogle(ctx, acc.ClientID, creds, redirectURI, in.Code)
	}
	if err != nil {
		e := apperr.As(err)
		s.d.Logger.Warn("oauth exchange failed", "provider", in.Provider, "client", acc.ClientID, "code", e.Code, "err", err)
		return &OAuthResult{RedirectURL: settings + "?error=" + string(e.Code)}, nil
	}
	return &OAuthResult{RedirectURL: settings + "?connected=" + in.Provider.Slug(), Connection: conn}, nil
}

func (s *Connections) expiry(expiresIn *int) *time.Time {
	if expiresIn == nil || *expiresIn == 0 {
		return nil
	}
	t := s.d.Clock.Now().Add(time.Duration(*expiresIn) * time.Second)
	return &t
}

func (s *Connections) connectGoogle(ctx context.Context, clientID string, creds domain.ProviderCredentials, redirectURI, code string) (*domain.Connection, error) {
	tokens, err := s.d.Google.ExchangeCode(ctx, creds.PublicID, creds.Secret, redirectURI, code)
	if err != nil {
		return nil, err
	}
	info, err := s.d.Google.Userinfo(ctx, tokens.AccessToken)
	if err != nil {
		return nil, err
	}
	accessEnc, err := s.d.Cipher.Encrypt(tokens.AccessToken)
	if err != nil {
		return nil, err
	}
	var refreshEnc *string
	if tokens.RefreshToken != nil {
		enc, err := s.d.Cipher.Encrypt(*tokens.RefreshToken)
		if err != nil {
			return nil, err
		}
		refreshEnc = &enc
	}
	display := "Akun Google"
	if info.Email != nil {
		display = *info.Email
	} else if info.Name != nil {
		display = *info.Name
	}
	scopes := tokens.Scopes
	if len(scopes) == 0 {
		scopes = scopesFor(domain.ProviderGoogle)
	}
	// Keep the stored refresh token when Google does not return a new one.
	return s.d.Store.UpsertConnection(ctx, domain.Connection{
		ID: ids.New(), ClientID: clientID, Provider: domain.ProviderGoogle, AccountID: info.Sub, DisplayName: display,
		AccessTokenEnc: accessEnc, RefreshTokenEnc: refreshEnc, ExpiresAt: s.expiry(tokens.ExpiresIn), Scopes: scopes,
	}, true)
}

func (s *Connections) connectMeta(ctx context.Context, clientID string, creds domain.ProviderCredentials, redirectURI, code string) (*domain.Connection, error) {
	short, err := s.d.Meta.ExchangeCode(ctx, creds.PublicID, creds.Secret, redirectURI, code)
	if err != nil {
		return nil, err
	}
	long, err := s.d.Meta.ExchangeLongLived(ctx, creds.PublicID, creds.Secret, short.AccessToken)
	if err != nil {
		return nil, err
	}
	me, err := s.d.Meta.Me(ctx, long.AccessToken)
	if err != nil {
		return nil, err
	}
	accessEnc, err := s.d.Cipher.Encrypt(long.AccessToken)
	if err != nil {
		return nil, err
	}
	return s.d.Store.UpsertConnection(ctx, domain.Connection{
		ID: ids.New(), ClientID: clientID, Provider: domain.ProviderMeta, AccountID: me.ID, DisplayName: me.Name,
		AccessTokenEnc: accessEnc, ExpiresAt: s.expiry(long.ExpiresIn), Scopes: scopesFor(domain.ProviderMeta),
	}, false)
}

// ─── Tokens for other services ───────────────────────────────

type AccessToken struct {
	ConnectionID string
	Provider     domain.Provider
	Token        string
}

// AccessToken returns the decrypted token, refreshing Google tokens that
// expire within RefreshSkew. NOT_FOUND when the provider app is not
// configured or the row is missing; TOKEN_EXPIRED when it cannot be refreshed.
func (s *Connections) AccessToken(ctx context.Context, connectionID string) (*AccessToken, error) {
	if connectionID == "" {
		return nil, apperr.New(apperr.NotFound, "")
	}
	c, err := s.d.Store.GetConnection(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, apperr.New(apperr.NotFound, "")
	}
	creds, err := s.d.Creds.Credentials(ctx, c.Provider.IntegrationKey())
	if err != nil {
		return nil, err
	}
	if !creds.Configured {
		return nil, apperr.New(apperr.NotFound, "")
	}
	if !c.Expiring(s.d.Clock.Now()) {
		token, err := s.d.Cipher.Decrypt(c.AccessTokenEnc)
		if err != nil {
			return nil, err
		}
		return &AccessToken{ConnectionID: c.ID, Provider: c.Provider, Token: token}, nil
	}
	if c.Provider == domain.ProviderGoogle && c.RefreshTokenEnc != nil {
		refresh, err := s.d.Cipher.Decrypt(*c.RefreshTokenEnc)
		if err != nil {
			return nil, err
		}
		fresh, err := s.d.Google.Refresh(ctx, creds.PublicID, creds.Secret, refresh)
		if err != nil {
			return nil, err
		}
		enc, err := s.d.Cipher.Encrypt(fresh.AccessToken)
		if err != nil {
			return nil, err
		}
		if err := s.d.Store.UpdateAccessToken(ctx, c.ID, enc, s.expiry(fresh.ExpiresIn)); err != nil {
			return nil, err
		}
		return &AccessToken{ConnectionID: c.ID, Provider: c.Provider, Token: fresh.AccessToken}, nil
	}
	return nil, apperr.FromProvider(strings.ToLower(string(c.Provider)), apperr.TokenExpired, msgTokenExpired, nil)
}

// Find returns the first connection of a provider for a client, if any.
func (s *Connections) Find(ctx context.Context, clientID string, provider domain.Provider) (*domain.Connection, error) {
	if !provider.Valid() {
		return nil, apperr.New(apperr.Validation, "")
	}
	return s.d.Store.FindConnection(ctx, clientID, provider)
}
