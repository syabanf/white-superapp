package googleoauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/httpx"
)

func newTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &Client{HTTP: httpx.New(2 * time.Second), TokenURL: srv.URL + "/token", UserinfoURL: srv.URL + "/userinfo"}
}

func TestAuthURL(t *testing.T) {
	u, err := url.Parse(AuthURL("cid", "https://app/api/connections/google/callback", "st.sig"))
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if u.Host != "accounts.google.com" || q.Get("access_type") != "offline" || q.Get("prompt") != "consent" || q.Get("include_granted_scopes") != "true" || q.Get("response_type") != "code" {
		t.Fatalf("%s", u)
	}
	if !strings.Contains(q.Get("scope"), "webmasters.readonly ") || q.Get("state") != "st.sig" || q.Get("client_id") != "cid" {
		t.Fatalf("%s", u)
	}
}

func TestExchangeCodeAndUserinfo(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			_ = r.ParseForm()
			if r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code") != "c0de" || r.Form.Get("redirect_uri") != "https://app/cb" || r.Form.Get("client_secret") != "sec" {
				w.WriteHeader(400)
				_, _ = w.Write([]byte(`{"error":"invalid_request"}`))
				return
			}
			_, _ = w.Write([]byte(`{"access_token":"at","refresh_token":"rt","expires_in":3599,"scope":"openid email"}`))
		case "/userinfo":
			if r.Header.Get("Authorization") != "Bearer at" {
				w.WriteHeader(401)
				return
			}
			_, _ = w.Write([]byte(`{"sub":"123","email":"a@b.id","name":"A"}`))
		}
	})
	ctx := context.Background()
	tok, err := c.ExchangeCode(ctx, "cid", "sec", "https://app/cb", "c0de")
	if err != nil {
		t.Fatal(err)
	}
	if tok.AccessToken != "at" || *tok.RefreshToken != "rt" || *tok.ExpiresIn != 3599 || len(tok.Scopes) != 2 {
		t.Fatalf("%+v", tok)
	}
	info, err := c.Userinfo(ctx, "at")
	if err != nil {
		t.Fatal(err)
	}
	if info.Sub != "123" || *info.Email != "a@b.id" {
		t.Fatalf("%+v", info)
	}
	if _, err := c.Userinfo(ctx, "bad"); !apperr.Is(err, apperr.TokenExpired) {
		t.Fatalf("401 should map to TOKEN_EXPIRED, got %v", err)
	}
	if _, err := c.ExchangeCode(ctx, "cid", "wrong", "https://app/cb", "c0de"); !apperr.Is(err, apperr.Unknown) {
		t.Fatalf("400 invalid_request should be UNKNOWN, got %v", err)
	}
}

func TestExchangeCodeMissingToken(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{}`)) })
	_, err := c.ExchangeCode(context.Background(), "cid", "sec", "u", "c")
	if e := apperr.As(err); e.Code != apperr.Unknown || e.Message != "Token tidak ditemukan pada respons Google" {
		t.Fatalf("%v", err)
	}
}

func TestRefresh(t *testing.T) {
	var reply string
	var status int
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "rt" {
			t.Errorf("bad form %v", r.Form)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	})
	ctx := context.Background()

	status, reply = 200, `{"access_token":"new","expires_in":3600}`
	got, err := c.Refresh(ctx, "cid", "sec", "rt")
	if err != nil || got.AccessToken != "new" || *got.ExpiresIn != 3600 {
		t.Fatalf("%+v %v", got, err)
	}

	status, reply = 400, `{"error":"invalid_grant"}`
	_, err = c.Refresh(ctx, "cid", "sec", "rt")
	if e := apperr.As(err); e.Code != apperr.TokenExpired || !e.RequiresReconnect || e.Message != "Google menolak refresh token (invalid_grant)." {
		t.Fatalf("%v", err)
	}

	status, reply = 500, ``
	_, err = c.Refresh(ctx, "cid", "sec", "rt")
	if e := apperr.As(err); e.Code != apperr.Permission || e.Message != "Google menolak refresh token (500)." {
		t.Fatalf("%v", err)
	}
}

func TestNetworkError(t *testing.T) {
	c := &Client{HTTP: httpx.New(time.Second), TokenURL: "http://127.0.0.1:1/token", UserinfoURL: "http://127.0.0.1:1/u"}
	if _, err := c.Refresh(context.Background(), "a", "b", "c"); !apperr.Is(err, apperr.Network) {
		t.Fatalf("%v", err)
	}
}
