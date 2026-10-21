// Package verify runs the cheap credential checks behind the setup wizard:
// one call per provider that fails fast on a wrong id or secret. Base URLs
// are fields so tests can point at httptest.
package verify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/integrations/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/httpx"
	"github.com/syabanf/white-superapp/white-backend/pkg/providers/google"
	"github.com/syabanf/white-superapp/white-backend/pkg/providers/meta"
)

const Timeout = 12 * time.Second

type Client struct {
	HTTP           *httpx.Client
	MetaGraphBase  string
	GoogleTokenURL string
	DataForSeoBase string
	OpenRouterBase string
	ApifyBase      string
}

func New() *Client {
	return &Client{
		HTTP:           httpx.New(Timeout),
		MetaGraphBase:  meta.GraphBase,
		GoogleTokenURL: google.TokenURL,
		DataForSeoBase: "https://api.dataforseo.com/v3",
		OpenRouterBase: "https://openrouter.ai/api/v1",
		ApifyBase:      "https://api.apify.com/v2",
	}
}

func fail(msg string) domain.VerifyResult { return domain.VerifyResult{OK: false, Message: msg} }

func ok(msg string) domain.VerifyResult { return domain.VerifyResult{OK: true, Message: msg} }

// MetaApp asks for an app access token via client_credentials, which proves
// app id and secret match.
func (c *Client) MetaApp(ctx context.Context, appID, appSecret string) domain.VerifyResult {
	q := url.Values{"client_id": {appID}, "client_secret": {appSecret}, "grant_type": {"client_credentials"}}
	res, err := c.HTTP.Get(ctx, httpx.WithQuery(c.MetaGraphBase+"/oauth/access_token", q), nil)
	if err != nil {
		return fail("Tidak bisa menghubungi Meta: " + err.Error())
	}
	var body struct {
		AccessToken string `json:"access_token"`
		Error       struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(res.Body, &body)
	if res.Status == 200 && body.AccessToken != "" {
		return ok("Kredensial aplikasi Meta valid.")
	}
	if body.Error.Message != "" {
		return fail("Meta menolak: " + body.Error.Message)
	}
	return fail("App ID atau App Secret tidak cocok.")
}

// GoogleClient has no "check client" endpoint; a bogus code exchange tells
// invalid_client (wrong id or secret) from invalid_grant (credentials fine).
func (c *Client) GoogleClient(ctx context.Context, clientID, clientSecret string) domain.VerifyResult {
	form := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"grant_type":    {"authorization_code"},
		"code":          {"white-analytics-probe"},
		"redirect_uri":  {"http://localhost/probe"},
	}
	res, err := c.HTTP.PostForm(ctx, c.GoogleTokenURL, form, nil)
	if err != nil {
		return fail("Tidak bisa menghubungi Google: " + err.Error())
	}
	var body struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(res.Body, &body)
	switch body.Error {
	case "invalid_grant", "redirect_uri_mismatch":
		return ok("Client ID & secret Google dikenali.")
	case "invalid_client", "unauthorized_client":
		return fail("Client ID atau Client Secret Google salah.")
	}
	code := body.Error
	if code == "" {
		code = "tanpa kode"
	}
	return fail(fmt.Sprintf("Respons tak terduga dari Google (%s).", code))
}

func (c *Client) DataForSeo(ctx context.Context, login, password string) domain.VerifyResult {
	auth := base64.StdEncoding.EncodeToString([]byte(login + ":" + password))
	res, err := c.HTTP.Get(ctx, c.DataForSeoBase+"/appendix/user_data", map[string]string{"Authorization": "Basic " + auth})
	if err != nil {
		return fail("Tidak bisa menghubungi DataForSEO: " + err.Error())
	}
	if res.Status == 401 {
		return fail("Login atau password DataForSEO salah.")
	}
	var body struct {
		Tasks []struct {
			StatusCode int `json:"status_code"`
			Result     []struct {
				Money struct {
					Balance *float64 `json:"balance"`
				} `json:"money"`
			} `json:"result"`
		} `json:"tasks"`
	}
	_ = json.Unmarshal(res.Body, &body)
	if res.Status == 200 && len(body.Tasks) > 0 && body.Tasks[0].StatusCode == 20000 {
		task := body.Tasks[0]
		if len(task.Result) > 0 && task.Result[0].Money.Balance != nil {
			balance := *task.Result[0].Money.Balance
			r := ok(fmt.Sprintf("Terhubung. Saldo akun: $%.2f.", balance))
			r.Detail = map[string]string{"balanceUsd": fmt.Sprintf("%g", balance)}
			return r
		}
		return ok("Terhubung ke DataForSEO.")
	}
	return fail(fmt.Sprintf("DataForSEO menjawab status %d.", res.Status))
}

func (c *Client) OpenRouter(ctx context.Context, apiKey string) domain.VerifyResult {
	res, err := c.HTTP.Get(ctx, c.OpenRouterBase+"/auth/key", map[string]string{"Authorization": "Bearer " + apiKey})
	if err != nil {
		return fail("Tidak bisa menghubungi OpenRouter: " + err.Error())
	}
	switch res.Status {
	case 200:
		var body struct {
			Data struct {
				Label string `json:"label"`
			} `json:"data"`
		}
		_ = json.Unmarshal(res.Body, &body)
		if body.Data.Label != "" {
			return ok(fmt.Sprintf("Kunci OpenRouter valid (%s).", body.Data.Label))
		}
		return ok("Kunci OpenRouter valid.")
	case 401:
		return fail("Kunci OpenRouter ditolak.")
	}
	return fail(fmt.Sprintf("OpenRouter menjawab status %d.", res.Status))
}

// Apify reports who the token belongs to and this month's spend.
func (c *Client) Apify(ctx context.Context, token string) domain.VerifyResult {
	u, err := c.ApifyUsage(ctx, token)
	if err != nil {
		return fail(apperr.As(err).Message)
	}
	who := ""
	if u.Username != nil {
		who = " (" + *u.Username
		if u.Plan != nil {
			who += ", " + *u.Plan
		}
		who += ")"
	}
	usage := ""
	if u.MonthlyUsageUSD != nil && u.MonthlyLimitUSD != nil {
		usage = fmt.Sprintf(" Pemakaian bulan ini $%.2f dari $%.2f.", *u.MonthlyUsageUSD, *u.MonthlyLimitUSD)
	}
	r := ok(fmt.Sprintf("Token Apify valid%s.%s", who, usage))
	if u.MonthlyUsageUSD != nil {
		r.Detail = map[string]string{"monthlyUsageUsd": fmt.Sprintf("%g", *u.MonthlyUsageUSD)}
	}
	return r
}

type ApifyUsage struct {
	Username        *string
	Plan            *string
	MonthlyUsageUSD *float64
	MonthlyLimitUSD *float64
}

// ApifyUsage calls /users/me and, when available, /users/me/limits.
func (c *Client) ApifyUsage(ctx context.Context, token string) (*ApifyUsage, error) {
	me, err := c.apifyGet(ctx, "/users/me", token)
	if err != nil {
		return nil, err
	}
	var meBody struct {
		Data struct {
			Username *string `json:"username"`
			Plan     struct {
				ID *string `json:"id"`
			} `json:"plan"`
		} `json:"data"`
	}
	_ = json.Unmarshal(me, &meBody)
	out := &ApifyUsage{Username: meBody.Data.Username, Plan: meBody.Data.Plan.ID}
	// The limits endpoint is optional.
	if limits, err := c.apifyGet(ctx, "/users/me/limits", token); err == nil {
		var lim struct {
			Data struct {
				Current struct {
					MonthlyUsageUSD *float64 `json:"monthlyUsageUsd"`
				} `json:"current"`
				Limits struct {
					MonthlyUsageUSD *float64 `json:"monthlyUsageUsd"`
				} `json:"limits"`
			} `json:"data"`
		}
		_ = json.Unmarshal(limits, &lim)
		out.MonthlyUsageUSD = lim.Data.Current.MonthlyUsageUSD
		out.MonthlyLimitUSD = lim.Data.Limits.MonthlyUsageUSD
	}
	return out, nil
}

func (c *Client) apifyGet(ctx context.Context, path, token string) ([]byte, error) {
	res, err := c.HTTP.Get(ctx, httpx.WithQuery(c.ApifyBase+path, url.Values{"token": {token}}), nil)
	if err != nil {
		return nil, apperr.FromProvider("apify", apperr.Network, "Tidak bisa menghubungi Apify: "+err.Error(), err)
	}
	if res.Status >= 400 {
		return nil, mapApifyStatus(res)
	}
	return res.Body, nil
}

func mapApifyStatus(res *httpx.Response) *apperr.Error {
	switch res.Status {
	case 401, 403:
		return apperr.FromProvider("apify", apperr.TokenExpired, "Token Apify ditolak.", nil)
	case 402:
		e := apperr.FromProvider("apify", apperr.Permission, "Kuota/kredit Apify habis (402).", nil)
		e.RequiresReconnect = false
		return e
	case 404:
		return apperr.FromProvider("apify", apperr.NotFound, "Actor Apify tidak ditemukan, periksa override actor.", nil)
	case 408:
		return apperr.FromProvider("apify", apperr.Network, "Run Apify melebihi batas waktu sinkron (408).", nil)
	case 429:
		return apperr.FromProvider("apify", apperr.RateLimit, "Terlalu banyak permintaan ke Apify (429).", nil)
	}
	return apperr.FromProvider("apify", apperr.Unknown, fmt.Sprintf("Apify menjawab %d: %s", res.Status, res.Snippet(200)), nil)
}
