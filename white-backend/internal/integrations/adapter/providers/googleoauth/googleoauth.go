// Package googleoauth exchanges and refreshes Google OAuth tokens for the
// connections use case. Endpoints are fields so tests can point at httptest.
package googleoauth

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/httpx"
	"github.com/syabanf/white-superapp/white-backend/pkg/providers/google"
)

const provider = "google"

// Timeout covers every exchange call. The Next.js app had none on the code
// exchange and userinfo; 15 s matches its refresh timeout.
const Timeout = 15 * time.Second

type Tokens struct {
	AccessToken  string
	RefreshToken *string
	// ExpiresIn is nil when Google omits it.
	ExpiresIn *int
	Scopes    []string
}

type Userinfo struct {
	Sub   string
	Email *string
	Name  *string
}

type Refreshed struct {
	AccessToken string
	ExpiresIn   *int
}

type Client struct {
	HTTP        *httpx.Client
	TokenURL    string
	UserinfoURL string
}

func New() *Client {
	return &Client{HTTP: httpx.New(Timeout), TokenURL: google.TokenURL, UserinfoURL: google.UserinfoURL}
}

// AuthURL builds the consent URL. Offline access with a forced consent
// prompt makes Google return a refresh token every time.
func AuthURL(clientID, redirectURI, state string) string {
	q := url.Values{}
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("access_type", "offline")
	q.Set("prompt", "consent")
	q.Set("include_granted_scopes", "true")
	q.Set("scope", strings.Join(google.Scopes, " "))
	q.Set("state", state)
	return google.AuthURL + "?" + q.Encode()
}

type tokenBody struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    *int   `json:"expires_in"`
	Scope        string `json:"scope"`
	Error        string `json:"error"`
}

func (c *Client) ExchangeCode(ctx context.Context, clientID, clientSecret, redirectURI, code string) (*Tokens, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURI},
		"code":          {code},
	}
	res, err := c.HTTP.PostForm(ctx, c.TokenURL, form, nil)
	if err != nil {
		return nil, apperr.FromProvider(provider, apperr.Network, "Tidak dapat menghubungi server token Google", err)
	}
	if res.Status >= 400 {
		return nil, google.MapError(provider, res.Status, res.Body)
	}
	var body tokenBody
	_ = json.Unmarshal(res.Body, &body)
	if body.AccessToken == "" {
		return nil, apperr.FromProvider(provider, apperr.Unknown, "Token tidak ditemukan pada respons Google", nil)
	}
	t := &Tokens{AccessToken: body.AccessToken, ExpiresIn: body.ExpiresIn, Scopes: strings.Fields(body.Scope)}
	if body.RefreshToken != "" {
		t.RefreshToken = &body.RefreshToken
	}
	return t, nil
}

func (c *Client) Userinfo(ctx context.Context, accessToken string) (*Userinfo, error) {
	res, err := c.HTTP.Get(ctx, c.UserinfoURL, map[string]string{"Authorization": "Bearer " + accessToken})
	if err != nil {
		return nil, apperr.FromProvider(provider, apperr.Network, "Tidak dapat menghubungi userinfo Google", err)
	}
	if res.Status >= 400 {
		return nil, google.MapError(provider, res.Status, res.Body)
	}
	var body struct {
		Sub   string  `json:"sub"`
		Email *string `json:"email"`
		Name  *string `json:"name"`
	}
	_ = json.Unmarshal(res.Body, &body)
	if body.Sub == "" {
		return nil, apperr.FromProvider(provider, apperr.Unknown, "Respons userinfo tidak valid", nil)
	}
	return &Userinfo{Sub: body.Sub, Email: body.Email, Name: body.Name}, nil
}

// Refresh trades the stored refresh token for a new access token. Google
// does not rotate the refresh token. invalid_grant means the user revoked
// access (TOKEN_EXPIRED); anything else is a PERMISSION problem.
func (c *Client) Refresh(ctx context.Context, clientID, clientSecret, refreshToken string) (*Refreshed, error) {
	form := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}
	res, err := c.HTTP.PostForm(ctx, c.TokenURL, form, nil)
	if err != nil {
		return nil, apperr.FromProvider(provider, apperr.Network, "Tidak dapat menghubungi server token Google", err)
	}
	var body tokenBody
	_ = json.Unmarshal(res.Body, &body)
	if res.Status >= 400 || body.AccessToken == "" {
		code := apperr.Permission
		reason := body.Error
		if reason == "invalid_grant" {
			code = apperr.TokenExpired
		}
		if reason == "" {
			reason = strconv.Itoa(res.Status)
		}
		return nil, apperr.FromProvider(provider, code, "Google menolak refresh token ("+reason+").", nil)
	}
	return &Refreshed{AccessToken: body.AccessToken, ExpiresIn: body.ExpiresIn}, nil
}
