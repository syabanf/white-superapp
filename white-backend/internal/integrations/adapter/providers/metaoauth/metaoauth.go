// Package metaoauth runs the Meta login flow for the connections use case:
// code to short-lived token, short-lived to long-lived token, then /me.
// GraphBase is a field so tests can point at httptest.
package metaoauth

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/httpx"
	"github.com/syabanf/white-superapp/white-backend/pkg/providers/meta"
)

const provider = "meta"

// Timeout covers every exchange call (the Next.js app had none).
const Timeout = 15 * time.Second

type Token struct {
	AccessToken string
	// ExpiresIn is nil when Meta omits it.
	ExpiresIn *int
}

type Me struct {
	ID   string
	Name string
}

type Client struct {
	HTTP      *httpx.Client
	GraphBase string
}

func New() *Client {
	return &Client{HTTP: httpx.New(Timeout), GraphBase: meta.GraphBase}
}

// AuthURL builds the Facebook login dialog URL (scopes comma-joined).
func AuthURL(appID, redirectURI, state string) string {
	q := url.Values{}
	q.Set("client_id", appID)
	q.Set("redirect_uri", redirectURI)
	q.Set("state", state)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(meta.Scopes, ","))
	return meta.Dialog + "?" + q.Encode()
}

func (c *Client) ExchangeCode(ctx context.Context, appID, appSecret, redirectURI, code string) (*Token, error) {
	return c.token(ctx, url.Values{
		"client_id":     {appID},
		"client_secret": {appSecret},
		"redirect_uri":  {redirectURI},
		"code":          {code},
	})
}

// ExchangeLongLived upgrades a short-lived user token to one valid for about 60 days.
func (c *Client) ExchangeLongLived(ctx context.Context, appID, appSecret, accessToken string) (*Token, error) {
	return c.token(ctx, url.Values{
		"grant_type":        {"fb_exchange_token"},
		"client_id":         {appID},
		"client_secret":     {appSecret},
		"fb_exchange_token": {accessToken},
	})
}

func (c *Client) token(ctx context.Context, q url.Values) (*Token, error) {
	body, err := c.get(ctx, "/oauth/access_token", q)
	if err != nil {
		return nil, err
	}
	var t struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   *int   `json:"expires_in"`
	}
	_ = json.Unmarshal(body, &t)
	if t.AccessToken == "" {
		return nil, apperr.FromProvider(provider, apperr.Unknown, "Token tidak ditemukan pada respons Meta", nil)
	}
	return &Token{AccessToken: t.AccessToken, ExpiresIn: t.ExpiresIn}, nil
}

// Me returns the token owner, used as the connection's account id and name.
func (c *Client) Me(ctx context.Context, accessToken string) (*Me, error) {
	body, err := c.get(ctx, "/me", url.Values{"fields": {"id,name"}, "access_token": {accessToken}})
	if err != nil {
		return nil, err
	}
	var me struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	_ = json.Unmarshal(body, &me)
	if me.ID == "" {
		return nil, apperr.FromProvider(provider, apperr.Unknown, "Respons /me tidak valid", nil)
	}
	if me.Name == "" {
		me.Name = me.ID
	}
	return &Me{ID: me.ID, Name: me.Name}, nil
}

func (c *Client) get(ctx context.Context, path string, q url.Values) ([]byte, error) {
	res, err := c.HTTP.Get(ctx, httpx.WithQuery(c.GraphBase+path, q), nil)
	if err != nil {
		return nil, apperr.FromProvider(provider, apperr.Network, "Tidak dapat menghubungi Graph API", err)
	}
	if res.Status >= 400 {
		return nil, meta.MapError(provider, res.Status, res.Body)
	}
	if len(res.Body) == 0 {
		return nil, apperr.FromProvider(provider, apperr.Unknown, "Respons Graph API kosong", nil)
	}
	return res.Body, nil
}
