// Package meta is the Graph API transport shared by every service that
// talks to Facebook or Instagram: base URL, token placement and the error
// mapping the old adapters agreed on.
package meta

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/httpx"
)

const (
	Version   = "v21.0"
	GraphBase = "https://graph.facebook.com/" + Version
	Dialog    = "https://www.facebook.com/" + Version + "/dialog/oauth"
)

// Scopes requested at connect time. Publishing needs pages_manage_posts and
// instagram_content_publish on top; those are tracked in the review checklist.
var Scopes = []string{"pages_show_list", "pages_read_engagement", "instagram_basic", "instagram_manage_insights", "business_management", "ads_read"}

type graphError struct {
	Error struct {
		Message   string `json:"message"`
		Type      string `json:"type"`
		Code      int    `json:"code"`
		Subcode   int    `json:"error_subcode"`
		FBTraceID string `json:"fbtrace_id"`
	} `json:"error"`
}

// MapError turns a Graph response into the shared error vocabulary.
// Provider names the caller ("meta", "meta-marketing", "meta-publisher").
func MapError(provider string, status int, body []byte) *apperr.Error {
	var ge graphError
	_ = json.Unmarshal(body, &ge)
	code := ge.Error.Code
	msg := ge.Error.Message
	if msg == "" {
		msg = httpx.Describe(&httpx.Response{Status: status, Body: body})
	}
	var c apperr.Code
	switch {
	case status == 429 || code == 4 || code == 17 || code == 32 || code == 613:
		c = apperr.RateLimit
	case code == 190 || ge.Error.Type == "OAuthException":
		c = apperr.TokenExpired
	case status == 403 || code == 10 || code == 272 || (code >= 200 && code <= 299):
		c = apperr.Permission
	case status == 404 || code == 803 || code == 100:
		c = apperr.NotFound
	default:
		c = apperr.Unknown
	}
	return apperr.FromProvider(provider, c, msg, nil)
}

// Client performs Graph calls with the token in the query string (GET) or
// the form body (POST), the way the Graph API expects.
type Client struct {
	HTTP     *httpx.Client
	Provider string
}

func NewClient(provider string, timeout time.Duration) *Client {
	return &Client{HTTP: httpx.New(timeout), Provider: provider}
}

// Get fetches `path` (e.g. "/me") with params and decodes JSON into out.
// A body carrying `error` is an error even on HTTP 200.
func (c *Client) Get(ctx context.Context, token, path string, params url.Values, out any) error {
	if params == nil {
		params = url.Values{}
	}
	if token != "" {
		params.Set("access_token", token)
	}
	res, err := c.HTTP.Get(ctx, httpx.WithQuery(GraphBase+path, params), nil)
	if err != nil {
		return apperr.FromProvider(c.Provider, apperr.Network, "Tidak bisa menghubungi Meta Graph API.", err)
	}
	return c.decode(res, out)
}

func (c *Client) Post(ctx context.Context, token, path string, form url.Values, out any) error {
	if form == nil {
		form = url.Values{}
	}
	if token != "" {
		form.Set("access_token", token)
	}
	res, err := c.HTTP.PostForm(ctx, GraphBase+path, form, nil)
	if err != nil {
		return apperr.FromProvider(c.Provider, apperr.Network, "Tidak bisa menghubungi Meta Graph API.", err)
	}
	return c.decode(res, out)
}

func (c *Client) decode(res *httpx.Response, out any) error {
	var probe graphError
	_ = json.Unmarshal(res.Body, &probe)
	if res.Status >= 400 || probe.Error.Message != "" || probe.Error.Code != 0 {
		return MapError(c.Provider, res.Status, res.Body)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(res.Body, out); err != nil {
		return apperr.FromProvider(c.Provider, apperr.Unknown, "Respons Meta tidak bisa dibaca.", err)
	}
	return nil
}

// Paged is the envelope of list endpoints.
type Paged[T any] struct {
	Data   []T `json:"data"`
	Paging struct {
		Next string `json:"next"`
	} `json:"paging"`
}

// GetAll follows paging.next up to maxPages times.
func GetAll[T any](ctx context.Context, c *Client, token, path string, params url.Values, maxPages int) ([]T, error) {
	var out []T
	var page Paged[T]
	if err := c.Get(ctx, token, path, params, &page); err != nil {
		return nil, err
	}
	out = append(out, page.Data...)
	next := page.Paging.Next
	for i := 1; i < maxPages && next != ""; i++ {
		res, err := c.HTTP.Get(ctx, next, nil)
		if err != nil {
			return out, apperr.FromProvider(c.Provider, apperr.Network, "Tidak bisa menghubungi Meta Graph API.", err)
		}
		page = Paged[T]{}
		if err := c.decode(res, &page); err != nil {
			return out, err
		}
		out = append(out, page.Data...)
		next = page.Paging.Next
	}
	return out, nil
}
