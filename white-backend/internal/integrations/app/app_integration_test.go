package app_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/integrations"
	"github.com/syabanf/white-superapp/white-backend/internal/integrations/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/integrations/adapter/providers/googleoauth"
	"github.com/syabanf/white-superapp/white-backend/internal/integrations/adapter/providers/metaoauth"
	"github.com/syabanf/white-superapp/white-backend/internal/integrations/adapter/providers/verify"
	"github.com/syabanf/white-superapp/white-backend/internal/integrations/app"
	"github.com/syabanf/white-superapp/white-backend/internal/integrations/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/crypto"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/httpx"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

const (
	adminID   = "user-admin"
	viewerID  = "user-viewer"
	clientID  = "client-1"
	clientSlg = "acme"
)

// stubAccess plays identity: admin manages, viewer only reads, others are unknown.
type stubAccess struct{}

func (stubAccess) Check(_ context.Context, userID, id, slug string) (*domain.ClientAccess, error) {
	if (id != "" && id != clientID) || (slug != "" && slug != clientSlg) {
		return nil, apperr.New(apperr.NotFound, "Klien tidak ditemukan.")
	}
	switch userID {
	case adminID:
		return &domain.ClientAccess{ClientID: clientID, Slug: clientSlg, CanManage: true}, nil
	case viewerID:
		return &domain.ClientAccess{ClientID: clientID, Slug: clientSlg}, nil
	}
	return nil, apperr.New(apperr.Unauthorized, "")
}

type recorder struct {
	mu       sync.Mutex
	subjects []string
}

func (r *recorder) Publish(_ context.Context, subject string, _ any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subjects = append(r.subjects, subject)
	return nil
}

type fixture struct {
	store    *postgres.Store
	settings *app.Settings
	conns    *app.Connections
	cipher   *crypto.Cipher
	clock    *clock.Fixed
	env      map[string]string
	bus      *recorder
	// provider is the fake Google and Meta server.
	provider *fakeProvider
}

type fakeProvider struct {
	srv *httptest.Server
	mu  sync.Mutex
	// refreshes counts token refresh calls; refreshError forces invalid_grant.
	refreshes    int
	refreshError bool
}

func newFakeProvider(t *testing.T) *fakeProvider {
	f := &fakeProvider{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		q := r.URL.Query()
		switch {
		case r.URL.Path == "/google/token" && r.Form.Get("grant_type") == "authorization_code":
			_, _ = w.Write([]byte(`{"access_token":"g-access","refresh_token":"g-refresh","expires_in":3600,"scope":"openid email"}`))
		case r.URL.Path == "/google/token" && r.Form.Get("grant_type") == "refresh_token":
			f.mu.Lock()
			f.refreshes++
			bad := f.refreshError
			f.mu.Unlock()
			if bad || r.Form.Get("refresh_token") != "g-refresh" {
				w.WriteHeader(400)
				_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
				return
			}
			_, _ = w.Write([]byte(`{"access_token":"g-access-2","expires_in":3600}`))
		case r.URL.Path == "/google/userinfo":
			_, _ = w.Write([]byte(`{"sub":"g-sub","email":"owner@acme.id"}`))
		case r.URL.Path == "/meta/oauth/access_token" && q.Get("grant_type") == "fb_exchange_token":
			_, _ = w.Write([]byte(`{"access_token":"m-long","expires_in":5184000}`))
		case r.URL.Path == "/meta/oauth/access_token":
			if q.Get("code") == "denied" {
				w.WriteHeader(400)
				_, _ = w.Write([]byte(`{"error":{"message":"Bad code","code":100,"type":"OAuthException"}}`))
				return
			}
			_, _ = w.Write([]byte(`{"access_token":"m-short","expires_in":5000}`))
		case r.URL.Path == "/meta/me":
			_, _ = w.Write([]byte(`{"id":"m-777","name":"White Agency"}`))
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func setup(t *testing.T) *fixture {
	t.Helper()
	pool := pg.TestPool(t, integrations.Migrations(), "migrations", integrations.Schema)
	pg.TruncateSchema(t, pool, integrations.Schema)
	cipher, err := crypto.New("test-encryption-key")
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{
		store:    postgres.NewStore(pool),
		cipher:   cipher,
		clock:    &clock.Fixed{T: time.Date(2026, 10, 21, 8, 0, 0, 0, time.UTC)},
		env:      map[string]string{},
		bus:      &recorder{},
		provider: newFakeProvider(t),
	}
	// Every verifier points at the fake server so no test call leaves the machine.
	dead := f.provider.srv.URL + "/none"
	ver := &verify.Client{HTTP: httpx.New(time.Second), MetaGraphBase: f.provider.srv.URL + "/meta", GoogleTokenURL: dead, DataForSeoBase: dead, OpenRouterBase: dead, ApifyBase: dead}
	f.settings = app.NewSettings(f.store, cipher, ver, f.bus, f.clock).WithEnv(func(k string) string { return f.env[k] })
	f.conns = app.NewConnections(app.ConnectionsDeps{
		Store: f.store, Cipher: cipher, Creds: f.settings, Access: stubAccess{},
		Google: &googleoauth.Client{HTTP: httpx.New(time.Second), TokenURL: f.provider.srv.URL + "/google/token", UserinfoURL: f.provider.srv.URL + "/google/userinfo"},
		Meta:   &metaoauth.Client{HTTP: httpx.New(time.Second), GraphBase: f.provider.srv.URL + "/meta"},
		Bus:    f.bus, Clock: f.clock, StateSecret: "test-secret", AppBaseURL: "https://app.test",
	})
	return f
}

func code(t *testing.T, err error, want apperr.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s, got nil", want)
	}
	if e := apperr.As(err); e.Code != want {
		t.Fatalf("expected %s, got %s (%s)", want, e.Code, e.Message)
	}
}

func str(s string) *string { return &s }

func TestIntegrationCredentialRoundTrip(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	f.env["GOOGLE_CLIENT_ID"] = "env-id"
	f.env["GOOGLE_CLIENT_SECRET"] = "env-secret-value"

	// Env fallback before anything is stored.
	ov, err := f.settings.ListStatuses(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ov.Statuses) != 5 || ov.Statuses[1].Key != domain.KeyGoogle || ov.Statuses[1].Source != domain.SourceEnv || *ov.Statuses[1].SecretMasked != "env-••••alue" {
		t.Fatalf("%+v", ov.Statuses[1])
	}
	if ov.Statuses[0].Source != domain.SourceNone || ov.Statuses[0].Configured {
		t.Fatalf("META should be unconfigured: %+v", ov.Statuses[0])
	}

	// DB wins over env; the secret is stored encrypted and reported masked.
	res, err := f.settings.SaveCredential(ctx, adminID, app.SaveCredentialInput{Key: domain.KeyGoogle, PublicID: str(" db-id "), Secret: str("db-secret-value")})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status.Source != domain.SourceDB || *res.Status.PublicID != "db-id" || *res.Status.SecretMasked != "db-s••••alue" || res.Verified != nil {
		t.Fatalf("%+v", res.Status)
	}
	row, err := f.store.GetSetting(ctx, domain.KeyGoogle)
	if err != nil || row == nil || *row.UpdatedByID != adminID {
		t.Fatalf("%+v %v", row, err)
	}
	if *row.SecretEnc == "db-secret-value" {
		t.Fatal("secret stored in plain text")
	}
	if plain, _ := f.cipher.Decrypt(*row.SecretEnc); plain != "db-secret-value" {
		t.Fatalf("decrypt: %q", plain)
	}
	creds, err := f.settings.Credentials(ctx, domain.KeyGoogle)
	if err != nil || creds.PublicID != "db-id" || creds.Secret != "db-secret-value" || !creds.Configured {
		t.Fatalf("%+v %v", creds, err)
	}

	// Blank secret keeps the stored one; absent publicId keeps it; "" clears it.
	res, err = f.settings.SaveCredential(ctx, adminID, app.SaveCredentialInput{Key: domain.KeyGoogle, Secret: str("")})
	if err != nil || *res.Status.PublicID != "db-id" || *res.Status.SecretMasked != "db-s••••alue" {
		t.Fatalf("%+v %v", res.Status, err)
	}
	res, err = f.settings.SaveCredential(ctx, adminID, app.SaveCredentialInput{Key: domain.KeyGoogle, PublicID: str("")})
	if err != nil || *res.Status.PublicID != "env-id" || res.Status.Source != domain.SourceDB {
		t.Fatalf("cleared public id should fall back to env: %+v %v", res.Status, err)
	}

	// Verify against the fake Meta server stores verifiedAt.
	res, err = f.settings.SaveCredential(ctx, adminID, app.SaveCredentialInput{Key: domain.KeyMeta, PublicID: str("app"), Secret: str("app-secret"), Verify: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Verified == nil || !res.Verified.OK || res.Verified.Message != "Kredensial aplikasi Meta valid." || res.Status.VerifiedAt == nil || !res.Status.VerifiedAt.Equal(f.clock.T) {
		t.Fatalf("fake graph hands out an app token: %+v %+v", res.Verified, res.Status)
	}
	res, err = f.settings.SaveCredential(ctx, adminID, app.SaveCredentialInput{Key: domain.KeyApify, Secret: str("tok"), Verify: true})
	if err != nil || res.Verified.OK || res.Verified.Message != "Actor Apify tidak ditemukan, periksa override actor." {
		t.Fatalf("%+v %v", res.Verified, err)
	}
	res, err = f.settings.SaveCredential(ctx, adminID, app.SaveCredentialInput{Key: domain.KeyDataForSeo, Secret: str("pw"), Verify: true})
	if err != nil || res.Verified.OK || res.Verified.Message != "Login (email akun)" {
		t.Fatalf("missing public id message: %+v %v", res.Verified, err)
	}

	// Validation.
	_, err = f.settings.SaveCredential(ctx, adminID, app.SaveCredentialInput{Key: "SLACK"})
	code(t, err, apperr.Validation)
	_, err = f.settings.SaveCredential(ctx, adminID, app.SaveCredentialInput{Key: domain.KeyApify, Secret: str(strings.Repeat("x", 2001))})
	code(t, err, apperr.Validation)

	// Clear returns to the env fallback and publishes.
	if err := f.settings.ClearCredential(ctx, domain.KeyGoogle); err != nil {
		t.Fatal(err)
	}
	creds, _ = f.settings.Credentials(ctx, domain.KeyGoogle)
	if creds.PublicID != "env-id" || creds.Secret != "env-secret-value" {
		t.Fatalf("%+v", creds)
	}
	if n := len(f.bus.subjects); n != 7 || f.bus.subjects[n-1] != events.IntegrationUpdated {
		t.Fatalf("events: %v", f.bus.subjects)
	}
}

func TestIntegrationMetaReview(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	_, err := f.settings.SaveMetaReview(ctx, adminID, app.MetaReviewInput{Permissions: map[string]string{"ads_read": "MAYBE"}})
	code(t, err, apperr.Validation)

	st, err := f.settings.SaveMetaReview(ctx, adminID, app.MetaReviewInput{Permissions: map[string]string{"ads_read": "APPROVED", "business_management": "APPROVED"}, AppLive: true})
	if err != nil {
		t.Fatal(err)
	}
	if st.Source != domain.SourceNone || st.Configured {
		t.Fatalf("review row must not count as credentials: %+v", st)
	}
	ov, err := f.settings.ListStatuses(ctx)
	if err != nil {
		t.Fatal(err)
	}
	p := ov.MetaReview.Progress()
	if !ov.MetaReview.AppLive || p.Approved != 2 || !p.AdsReady || p.PublishReady {
		t.Fatalf("%+v %+v", ov.MetaReview, p)
	}
	// The permissions map is replaced wholesale; other meta keys survive.
	if _, err := f.settings.SaveMetaReview(ctx, adminID, app.MetaReviewInput{Permissions: map[string]string{"ads_read": "SUBMITTED"}}); err != nil {
		t.Fatal(err)
	}
	ov, _ = f.settings.ListStatuses(ctx)
	if len(ov.MetaReview.Permissions) != 1 || ov.MetaReview.AppLive || ov.MetaReview.Progress().AdsReady {
		t.Fatalf("%+v", ov.MetaReview)
	}
}

func TestIntegrationSimulateListDisconnect(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	_, err := f.conns.Simulate(ctx, viewerID, clientID, domain.ProviderMeta)
	code(t, err, apperr.Unauthorized)
	_, err = f.conns.Simulate(ctx, adminID, "nope", domain.ProviderMeta)
	code(t, err, apperr.Unauthorized)
	_, err = f.conns.Simulate(ctx, adminID, clientID, "TIKTOK")
	code(t, err, apperr.Validation)

	c, err := f.conns.Simulate(ctx, adminID, clientID, domain.ProviderMeta)
	if err != nil {
		t.Fatal(err)
	}
	if c.AccountID != "demo-meta" || c.DisplayName != "Akun demo" || len(c.Scopes) != 6 || !c.ExpiresAt.Equal(f.clock.T.Add(60*24*time.Hour)) {
		t.Fatalf("%+v", c)
	}
	if plain, _ := f.cipher.Decrypt(c.AccessTokenEnc); plain != "mock" {
		t.Fatalf("token %q", plain)
	}
	again, err := f.conns.Simulate(ctx, adminID, clientID, domain.ProviderMeta)
	if err != nil || again.ID != c.ID {
		t.Fatalf("simulate must upsert on the same row: %v %v", again, err)
	}
	if _, err := f.conns.Simulate(ctx, adminID, clientID, domain.ProviderGoogle); err != nil {
		t.Fatal(err)
	}

	list, err := f.conns.List(ctx, viewerID, clientID)
	if err != nil || len(list) != 2 || list[0].Provider != domain.ProviderMeta {
		t.Fatalf("%v %v", list, err)
	}
	_, err = f.conns.List(ctx, "stranger", clientID)
	code(t, err, apperr.Unauthorized)

	found, err := f.conns.Find(ctx, clientID, domain.ProviderGoogle)
	if err != nil || found == nil || found.AccountID != "demo-google" {
		t.Fatalf("%v %v", found, err)
	}

	// Demo tokens resolve only when the provider app is configured.
	_, err = f.conns.AccessToken(ctx, c.ID)
	code(t, err, apperr.NotFound)
	f.env["META_APP_ID"], f.env["META_APP_SECRET"] = "app", "sec"
	tok, err := f.conns.AccessToken(ctx, c.ID)
	if err != nil || tok.Token != "mock" || tok.Provider != domain.ProviderMeta {
		t.Fatalf("%+v %v", tok, err)
	}

	code(t, f.conns.Disconnect(ctx, viewerID, clientID, c.ID), apperr.Unauthorized)
	code(t, f.conns.Disconnect(ctx, adminID, "other-client", c.ID), apperr.Unauthorized)
	code(t, f.conns.Disconnect(ctx, adminID, clientID, "missing"), apperr.NotFound)
	if err := f.conns.Disconnect(ctx, adminID, clientID, c.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ = f.conns.List(ctx, adminID, clientID); len(list) != 1 {
		t.Fatalf("%v", list)
	}
	if f.bus.subjects[len(f.bus.subjects)-1] != events.ConnectionDeleted {
		t.Fatalf("%v", f.bus.subjects)
	}
	_, err = f.conns.AccessToken(ctx, c.ID)
	code(t, err, apperr.NotFound)
}

func TestIntegrationOAuthGoogleAndRefresh(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	// Not configured: demo redirect.
	start, err := f.conns.StartOAuth(ctx, adminID, clientID, domain.ProviderGoogle, "https://app.test/")
	if err != nil || start.Configured || start.RedirectURL != "https://app.test/clients/acme/settings?demo=1&provider=google" || start.Nonce != "" {
		t.Fatalf("%+v %v", start, err)
	}
	_, err = f.conns.StartOAuth(ctx, viewerID, clientID, domain.ProviderGoogle, "")
	code(t, err, apperr.Unauthorized)

	f.env["GOOGLE_CLIENT_ID"], f.env["GOOGLE_CLIENT_SECRET"] = "gid", "gsecret"
	start, err = f.conns.StartOAuth(ctx, adminID, clientID, domain.ProviderGoogle, "")
	if err != nil || !start.Configured || len(start.Nonce) != 32 {
		t.Fatalf("%+v %v", start, err)
	}
	if !strings.HasPrefix(start.RedirectURL, "https://accounts.google.com/o/oauth2/v2/auth?") || !strings.Contains(start.RedirectURL, "redirect_uri=https%3A%2F%2Fapp.test%2Fapi%2Fconnections%2Fgoogle%2Fcallback") {
		t.Fatalf("%s", start.RedirectURL)
	}
	state := stateParam(t, start.RedirectURL)
	if domain.VerifyOAuthState(state, "test-secret", f.clock.T).Slug != clientSlg {
		t.Fatalf("state %s", state)
	}

	complete := func(in app.CompleteOAuthInput) *app.OAuthResult {
		t.Helper()
		if in.Provider == "" {
			in.Provider = domain.ProviderGoogle
		}
		r, err := f.conns.CompleteOAuth(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	settings := "https://app.test/clients/acme/settings"
	if r := complete(app.CompleteOAuthInput{State: "bogus", CookieNonce: start.Nonce, UserID: adminID}); r.RedirectURL != "https://app.test/clients?missing=1" {
		t.Fatalf("%+v", r)
	}
	if r := complete(app.CompleteOAuthInput{State: state, CookieNonce: "other", UserID: adminID}); r.RedirectURL != settings+"?error=oauth_state" {
		t.Fatalf("%+v", r)
	}
	if r := complete(app.CompleteOAuthInput{State: state, CookieNonce: start.Nonce}); r.RedirectURL != "https://app.test/login" {
		t.Fatalf("%+v", r)
	}
	if r := complete(app.CompleteOAuthInput{State: state, CookieNonce: start.Nonce, UserID: viewerID, Code: "c"}); r.RedirectURL != "https://app.test/clients?missing=1" {
		t.Fatalf("%+v", r)
	}
	if r := complete(app.CompleteOAuthInput{State: state, CookieNonce: start.Nonce, UserID: adminID, ProviderError: "access_denied"}); r.RedirectURL != settings+"?error=oauth_denied" {
		t.Fatalf("%+v", r)
	}
	r := complete(app.CompleteOAuthInput{State: state, CookieNonce: start.Nonce, UserID: adminID, Code: "c0de"})
	if r.RedirectURL != settings+"?connected=google" || r.Connection == nil {
		t.Fatalf("%+v", r)
	}
	c := r.Connection
	if c.AccountID != "g-sub" || c.DisplayName != "owner@acme.id" || c.RefreshTokenEnc == nil || !c.ExpiresAt.Equal(f.clock.T.Add(time.Hour)) || strings.Join(c.Scopes, " ") != "openid email" {
		t.Fatalf("%+v", c)
	}

	// Fresh token: no refresh call.
	tok, err := f.conns.AccessToken(ctx, c.ID)
	if err != nil || tok.Token != "g-access" || f.provider.refreshes != 0 {
		t.Fatalf("%+v %v refreshes=%d", tok, err, f.provider.refreshes)
	}
	// 30 s before expiry: refresh and persist.
	f.clock.T = f.clock.T.Add(time.Hour - 30*time.Second)
	tok, err = f.conns.AccessToken(ctx, c.ID)
	if err != nil || tok.Token != "g-access-2" || f.provider.refreshes != 1 {
		t.Fatalf("%+v %v refreshes=%d", tok, err, f.provider.refreshes)
	}
	stored, _ := f.store.GetConnection(ctx, c.ID)
	if plain, _ := f.cipher.Decrypt(stored.AccessTokenEnc); plain != "g-access-2" || !stored.ExpiresAt.Equal(f.clock.T.Add(time.Hour)) {
		t.Fatalf("%+v", stored)
	}
	// Revoked refresh token: TOKEN_EXPIRED with reconnect.
	f.clock.T = f.clock.T.Add(time.Hour)
	f.provider.refreshError = true
	_, err = f.conns.AccessToken(ctx, c.ID)
	code(t, err, apperr.TokenExpired)
	if !apperr.As(err).RequiresReconnect {
		t.Fatal("reconnect flag missing")
	}
}

func TestIntegrationOAuthMeta(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	f.env["META_APP_ID"], f.env["META_APP_SECRET"] = "app", "sec"

	start, err := f.conns.StartOAuth(ctx, adminID, clientID, domain.ProviderMeta, "")
	if err != nil || !strings.HasPrefix(start.RedirectURL, "https://www.facebook.com/v21.0/dialog/oauth?") {
		t.Fatalf("%+v %v", start, err)
	}
	state := stateParam(t, start.RedirectURL)
	in := app.CompleteOAuthInput{Provider: domain.ProviderMeta, State: state, CookieNonce: start.Nonce, UserID: adminID, Code: "denied"}
	r, err := f.conns.CompleteOAuth(ctx, in)
	if err != nil || r.RedirectURL != "https://app.test/clients/acme/settings?error=TOKEN_EXPIRED" {
		t.Fatalf("provider failure must surface its code: %+v %v", r, err)
	}
	in.Code = "c0de"
	r, err = f.conns.CompleteOAuth(ctx, in)
	if err != nil || r.RedirectURL != "https://app.test/clients/acme/settings?connected=meta" {
		t.Fatalf("%+v %v", r, err)
	}
	c := r.Connection
	if c.AccountID != "m-777" || c.DisplayName != "White Agency" || c.RefreshTokenEnc != nil || !c.ExpiresAt.Equal(f.clock.T.Add(5184000*time.Second)) || len(c.Scopes) != 6 {
		t.Fatalf("%+v", c)
	}
	if plain, _ := f.cipher.Decrypt(c.AccessTokenEnc); plain != "m-long" {
		t.Fatalf("token %q", plain)
	}
	// Meta cannot refresh: an expiring token is TOKEN_EXPIRED.
	f.clock.T = c.ExpiresAt.Add(-time.Second)
	_, err = f.conns.AccessToken(ctx, c.ID)
	code(t, err, apperr.TokenExpired)
	if e := apperr.As(err); e.Provider != "meta" || e.Message != "Token koneksi kedaluwarsa, hubungkan ulang di Pengaturan Klien." {
		t.Fatalf("%v", err)
	}

	// The worker path: purge on client deletion.
	n, err := f.store.DeleteClientConnections(ctx, clientID)
	if err != nil || n != 1 {
		t.Fatalf("%d %v", n, err)
	}
}

func stateParam(t *testing.T, rawURL string) string {
	t.Helper()
	_, q, _ := strings.Cut(rawURL, "?")
	for _, kv := range strings.Split(q, "&") {
		if k, v, _ := strings.Cut(kv, "="); k == "state" {
			return v
		}
	}
	t.Fatalf("no state in %s", rawURL)
	return ""
}
