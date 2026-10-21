package metaoauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/httpx"
)

func TestAuthURL(t *testing.T) {
	u, err := url.Parse(AuthURL("app", "https://app/api/connections/meta/callback", "st"))
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if u.Host != "www.facebook.com" || u.Path != "/v21.0/dialog/oauth" || q.Get("client_id") != "app" || q.Get("response_type") != "code" || q.Get("state") != "st" {
		t.Fatalf("%s", u)
	}
	if q.Get("scope") != "pages_show_list,pages_read_engagement,instagram_basic,instagram_manage_insights,business_management,ads_read" {
		t.Fatalf("scope %q", q.Get("scope"))
	}
}

func TestLoginFlow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case r.URL.Path == "/oauth/access_token" && q.Get("code") == "c0de":
			if q.Get("client_id") != "app" || q.Get("client_secret") != "sec" || q.Get("redirect_uri") != "https://app/cb" {
				w.WriteHeader(400)
				_, _ = w.Write([]byte(`{"error":{"message":"Invalid redirect","code":100}}`))
				return
			}
			_, _ = w.Write([]byte(`{"access_token":"short","token_type":"bearer","expires_in":5000}`))
		case r.URL.Path == "/oauth/access_token" && q.Get("grant_type") == "fb_exchange_token":
			if q.Get("fb_exchange_token") != "short" {
				w.WriteHeader(400)
				_, _ = w.Write([]byte(`{"error":{"message":"Bad token","code":190,"type":"OAuthException"}}`))
				return
			}
			_, _ = w.Write([]byte(`{"access_token":"long","expires_in":5183944}`))
		case r.URL.Path == "/me":
			if q.Get("access_token") != "long" || q.Get("fields") != "id,name" {
				w.WriteHeader(400)
				_, _ = w.Write([]byte(`{"error":{"message":"Bad token","code":190,"type":"OAuthException"}}`))
				return
			}
			_, _ = w.Write([]byte(`{"id":"777","name":"White Agency"}`))
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)
	c := &Client{HTTP: httpx.New(2 * time.Second), GraphBase: srv.URL}
	ctx := context.Background()

	short, err := c.ExchangeCode(ctx, "app", "sec", "https://app/cb", "c0de")
	if err != nil || short.AccessToken != "short" || *short.ExpiresIn != 5000 {
		t.Fatalf("%+v %v", short, err)
	}
	long, err := c.ExchangeLongLived(ctx, "app", "sec", short.AccessToken)
	if err != nil || long.AccessToken != "long" {
		t.Fatalf("%+v %v", long, err)
	}
	me, err := c.Me(ctx, long.AccessToken)
	if err != nil || me.ID != "777" || me.Name != "White Agency" {
		t.Fatalf("%+v %v", me, err)
	}

	_, err = c.ExchangeCode(ctx, "app", "wrong", "https://app/cb", "c0de")
	if e := apperr.As(err); e.Code != apperr.NotFound || e.Provider != "meta" || e.Message != "Invalid redirect" {
		t.Fatalf("%v", err)
	}
	if _, err := c.Me(ctx, "expired"); !apperr.Is(err, apperr.TokenExpired) {
		t.Fatalf("%v", err)
	}
}

func TestEmptyAndMissingToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/me" {
			return
		}
		_, _ = w.Write([]byte(`{"token_type":"bearer"}`))
	}))
	t.Cleanup(srv.Close)
	c := &Client{HTTP: httpx.New(time.Second), GraphBase: srv.URL}
	if _, err := c.ExchangeCode(context.Background(), "a", "b", "c", "d"); apperr.As(err).Message != "Token tidak ditemukan pada respons Meta" {
		t.Fatalf("%v", err)
	}
	if _, err := c.Me(context.Background(), "x"); apperr.As(err).Message != "Respons Graph API kosong" {
		t.Fatalf("%v", err)
	}
}
