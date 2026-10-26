package gateway

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"strings"

	commonv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/common/v1"
	identityv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/identity/v1"
	integrationsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/integrations/v1"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc/metadata"
)

// BrowserRoutes are the endpoints the browser hits directly (redirects and
// cookies), kept outside grpc-gateway on purpose.
type BrowserRoutes struct {
	up     *Upstreams
	logger *slog.Logger
}

func NewBrowserRoutes(up *Upstreams, logger *slog.Logger) *BrowserRoutes {
	return &BrowserRoutes{up: up, logger: logger}
}

var slugRe = regexp.MustCompile(`^[a-z0-9-]+$`)

const nonceCookieMaxAge = 600

func nonceCookieName(provider string) string { return provider + "_oauth_nonce" }

func (b *BrowserRoutes) Mount(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/connections/{provider}/start", b.oauthStart)
	mux.HandleFunc("GET /api/connections/{provider}/callback", b.oauthCallback)
	mux.HandleFunc("POST /api/share/{slug}/unlock", b.shareUnlock)
	mux.HandleFunc("GET /api/share/{slug}/context", b.shareContext)
	mux.HandleFunc("POST /internal/session", b.internalSession)
	mux.HandleFunc("POST /internal/users/sync", b.internalSyncUser)
	b.mountFiles(mux)
	b.mountReports(mux)
}

// forwardAuth copies the browser's Authorization header to the gRPC call.
func forwardAuth(r *http.Request) metadata.MD {
	md := metadata.Pairs(grpcx.EdgeHeader, "1")
	if v := r.Header.Get("Authorization"); v != "" {
		md.Set(authn.MetadataHeader, v)
	}
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		md.Set("x-forwarded-for", v)
	} else if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		// Same fallback grpc-gateway applies to transcoded routes.
		md.Set("x-forwarded-for", host)
	}
	if v := r.Header.Get("User-Agent"); v != "" {
		md.Set("user-agent", v)
	}
	return md
}

func providerOf(name string) (commonv1.ConnectionProvider, bool) {
	switch name {
	case "google":
		return commonv1.ConnectionProvider_GOOGLE, true
	case "meta":
		return commonv1.ConnectionProvider_META, true
	}
	return commonv1.ConnectionProvider_CONNECTION_PROVIDER_UNSPECIFIED, false
}

// oauthStart: /api/connections/{google|meta}/start?client=<slug>
// The browser arrives here from a link, so the access token travels in the
// `white_access` cookie set by the frontend, or in the Authorization header.
func (b *BrowserRoutes) oauthStart(w http.ResponseWriter, r *http.Request) {
	provider, ok := providerOf(r.PathValue("provider"))
	slug := r.URL.Query().Get("client")
	if !ok || !slugRe.MatchString(slug) {
		http.Redirect(w, r, appBaseURL(r)+"/clients?missing=1", http.StatusFound)
		return
	}
	ctx := metadata.NewOutgoingContext(r.Context(), authWithCookie(r))
	access, err := identityv1.NewClientServiceClient(b.up.Identity).GetClient(ctx, &identityv1.GetClientRequest{Slug: slug})
	if err != nil {
		e := grpcx.FromStatus(err)
		if e.Code == apperr.Unauthenticated {
			http.Redirect(w, r, appBaseURL(r)+"/login?callbackUrl="+r.URL.RequestURI(), http.StatusFound)
			return
		}
		http.Redirect(w, r, appBaseURL(r)+"/clients?missing=1", http.StatusFound)
		return
	}
	if !access.CanManage {
		http.Redirect(w, r, appBaseURL(r)+"/clients?missing=1", http.StatusFound)
		return
	}
	res, err := integrationsv1.NewConnectionServiceClient(b.up.Integrations).StartOAuth(ctx, &integrationsv1.StartOAuthRequest{
		ClientId: access.Client.Id, Provider: provider, AppBaseUrl: appBaseURL(r),
	})
	if err != nil {
		grpcx.WriteError(w, grpcx.FromStatus(err))
		return
	}
	if res.Nonce != "" {
		http.SetCookie(w, &http.Cookie{Name: nonceCookieName(r.PathValue("provider")), Value: res.Nonce, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: cookieSecure(r), MaxAge: nonceCookieMaxAge})
	}
	http.Redirect(w, r, res.RedirectUrl, http.StatusFound)
}

func (b *BrowserRoutes) oauthCallback(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("provider")
	provider, ok := providerOf(name)
	if !ok {
		http.Redirect(w, r, appBaseURL(r)+"/clients?missing=1", http.StatusFound)
		return
	}
	var nonce string
	if c, err := r.Cookie(nonceCookieName(name)); err == nil {
		nonce = c.Value
	}
	http.SetCookie(w, &http.Cookie{Name: nonceCookieName(name), Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	ctx := metadata.NewOutgoingContext(r.Context(), authWithCookie(r))
	var userID string
	if me, err := identityv1.NewAuthServiceClient(b.up.Identity).Me(ctx, &emptyPB); err == nil {
		userID = me.Id
	} else {
		http.Redirect(w, r, appBaseURL(r)+"/login", http.StatusFound)
		return
	}
	q := r.URL.Query()
	res, err := integrationsv1.NewConnectionServiceClient(b.up.Integrations).CompleteOAuth(ctx, &integrationsv1.CompleteOAuthRequest{
		Provider: provider, Code: q.Get("code"), State: q.Get("state"), ProviderError: q.Get("error"),
		CookieNonce: nonce, AppBaseUrl: appBaseURL(r), UserId: userID,
	})
	if err != nil {
		e := grpcx.FromStatus(err)
		b.logger.Warn("oauth callback failed", "provider", name, "code", e.Code, "err", e.Message)
		http.Redirect(w, r, appBaseURL(r)+"/clients?missing=1", http.StatusFound)
		return
	}
	http.Redirect(w, r, res.RedirectUrl, http.StatusFound)
}

// authWithCookie accepts the access token from the Authorization header or
// the `white_access` cookie (browser navigations cannot set headers).
func authWithCookie(r *http.Request) metadata.MD {
	md := forwardAuth(r)
	if len(md.Get(authn.MetadataHeader)) == 0 {
		if c, err := r.Cookie("white_access"); err == nil && c.Value != "" {
			md.Set(authn.MetadataHeader, "Bearer "+c.Value)
		}
	}
	return md
}

func shareCookieName(clientID string) string { return "share_" + clientID }

// shareUnlock: POST {pin} -> sets the share cookie on success and returns the context.
func (b *BrowserRoutes) shareUnlock(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	var body struct {
		Pin string `json:"pin"`
	}
	if r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
		body.Pin = r.FormValue("pin")
	} else {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	ctx := metadata.NewOutgoingContext(r.Context(), forwardAuth(r))
	res, err := identityv1.NewShareServiceClient(b.up.Identity).VerifySharePin(ctx, &identityv1.VerifySharePinRequest{Slug: slug, Pin: body.Pin, RemoteIp: remoteIP(r)})
	if err != nil {
		grpcx.WriteError(w, grpcx.FromStatus(err))
		return
	}
	if res.UnlockToken != "" {
		http.SetCookie(w, &http.Cookie{Name: shareCookieName(res.ClientId), Value: res.UnlockToken, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: cookieSecure(r), MaxAge: int(res.MaxAgeSeconds)})
	}
	writeJSON(w, map[string]any{"ok": true})
}

// shareContext resolves the public dashboard using the unlock cookie.
func (b *BrowserRoutes) shareContext(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	ctx := metadata.NewOutgoingContext(r.Context(), forwardAuth(r))
	client := identityv1.NewShareServiceClient(b.up.Identity)
	first, err := client.GetShareContext(ctx, &identityv1.GetShareContextRequest{Slug: slug})
	if err != nil {
		grpcx.WriteError(w, grpcx.FromStatus(err))
		return
	}
	res := first
	if first.Status == identityv1.ShareContext_LOCKED && first.Client != nil {
		if c, err := r.Cookie(shareCookieName(first.Client.Id)); err == nil {
			if again, err := client.GetShareContext(ctx, &identityv1.GetShareContextRequest{Slug: slug, UnlockToken: c.Value}); err == nil {
				res = again
			}
		}
	}
	writeProtoJSON(w, res)
}

func remoteIP(r *http.Request) string {
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		return strings.TrimSpace(strings.Split(v, ",")[0])
	}
	if v := r.Header.Get("X-Real-IP"); v != "" {
		return v
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 {
		host = host[:i]
	}
	return host
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
