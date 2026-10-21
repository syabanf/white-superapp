package verify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/httpx"
)

// fake routes every verifier at one server; the handler picks the reply by path.
func fake(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &Client{HTTP: httpx.New(2 * time.Second), MetaGraphBase: srv.URL + "/meta", GoogleTokenURL: srv.URL + "/google/token",
		DataForSeoBase: srv.URL + "/dfs", OpenRouterBase: srv.URL + "/or", ApifyBase: srv.URL + "/apify"}
}

func reply(w http.ResponseWriter, status int, body string) {
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func TestMetaApp(t *testing.T) {
	c := fake(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/meta/oauth/access_token" || q.Get("grant_type") != "client_credentials" {
			reply(w, 404, "")
			return
		}
		if q.Get("client_secret") == "good" {
			reply(w, 200, `{"access_token":"app|tok"}`)
		} else if q.Get("client_secret") == "noerr" {
			reply(w, 400, `{}`)
		} else {
			reply(w, 400, `{"error":{"message":"Error validating client secret."}}`)
		}
	})
	ctx := context.Background()
	if r := c.MetaApp(ctx, "1", "good"); !r.OK || r.Message != "Kredensial aplikasi Meta valid." {
		t.Fatalf("%+v", r)
	}
	if r := c.MetaApp(ctx, "1", "bad"); r.OK || r.Message != "Meta menolak: Error validating client secret." {
		t.Fatalf("%+v", r)
	}
	if r := c.MetaApp(ctx, "1", "noerr"); r.OK || r.Message != "App ID atau App Secret tidak cocok." {
		t.Fatalf("%+v", r)
	}
}

func TestGoogleClient(t *testing.T) {
	var errCode string
	c := fake(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("code") != "white-analytics-probe" || r.Form.Get("redirect_uri") != "http://localhost/probe" {
			t.Errorf("probe body %v", r.Form)
		}
		if errCode == "" {
			reply(w, 200, `{"access_token":"?"}`)
			return
		}
		reply(w, 400, `{"error":"`+errCode+`"}`)
	})
	ctx := context.Background()
	cases := []struct {
		code, msg string
		ok        bool
	}{
		{"invalid_grant", "Client ID & secret Google dikenali.", true},
		{"redirect_uri_mismatch", "Client ID & secret Google dikenali.", true},
		{"invalid_client", "Client ID atau Client Secret Google salah.", false},
		{"unauthorized_client", "Client ID atau Client Secret Google salah.", false},
		{"weird", "Respons tak terduga dari Google (weird).", false},
		{"", "Respons tak terduga dari Google (tanpa kode).", false},
	}
	for _, tc := range cases {
		errCode = tc.code
		if r := c.GoogleClient(ctx, "id", "sec"); r.OK != tc.ok || r.Message != tc.msg {
			t.Errorf("%s: %+v", tc.code, r)
		}
	}
}

func TestDataForSeo(t *testing.T) {
	var status int
	var body string
	c := fake(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dfs/appendix/user_data" || r.Header.Get("Authorization") != "Basic bG9naW46cHc=" {
			reply(w, 404, "")
			return
		}
		reply(w, status, body)
	})
	ctx := context.Background()
	status, body = 200, `{"tasks":[{"status_code":20000,"result":[{"money":{"balance":12.5}}]}]}`
	if r := c.DataForSeo(ctx, "login", "pw"); !r.OK || r.Message != "Terhubung. Saldo akun: $12.50." || r.Detail["balanceUsd"] != "12.5" {
		t.Fatalf("%+v", r)
	}
	status, body = 200, `{"tasks":[{"status_code":20000,"result":[]}]}`
	if r := c.DataForSeo(ctx, "login", "pw"); !r.OK || r.Message != "Terhubung ke DataForSEO." || r.Detail != nil {
		t.Fatalf("%+v", r)
	}
	status, body = 401, ``
	if r := c.DataForSeo(ctx, "login", "pw"); r.OK || r.Message != "Login atau password DataForSEO salah." {
		t.Fatalf("%+v", r)
	}
	status, body = 200, `{"tasks":[{"status_code":40100}]}`
	if r := c.DataForSeo(ctx, "login", "pw"); r.OK || r.Message != "DataForSEO menjawab status 200." {
		t.Fatalf("%+v", r)
	}
}

func TestOpenRouter(t *testing.T) {
	var status int
	var body string
	c := fake(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/or/auth/key" || r.Header.Get("Authorization") != "Bearer k" {
			reply(w, 404, "")
			return
		}
		reply(w, status, body)
	})
	ctx := context.Background()
	status, body = 200, `{"data":{"label":"white"}}`
	if r := c.OpenRouter(ctx, "k"); !r.OK || r.Message != "Kunci OpenRouter valid (white)." {
		t.Fatalf("%+v", r)
	}
	status, body = 200, `{"data":{}}`
	if r := c.OpenRouter(ctx, "k"); !r.OK || r.Message != "Kunci OpenRouter valid." {
		t.Fatalf("%+v", r)
	}
	status, body = 401, ``
	if r := c.OpenRouter(ctx, "k"); r.OK || r.Message != "Kunci OpenRouter ditolak." {
		t.Fatalf("%+v", r)
	}
	status, body = 503, ``
	if r := c.OpenRouter(ctx, "k"); r.OK || r.Message != "OpenRouter menjawab status 503." {
		t.Fatalf("%+v", r)
	}
}

func TestApify(t *testing.T) {
	var limitsStatus, meStatus int
	c := fake(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("token") != "tok" {
			reply(w, 401, "")
			return
		}
		switch r.URL.Path {
		case "/apify/users/me":
			reply(w, meStatus, `{"data":{"username":"white","plan":{"id":"FREE"}}}`)
		case "/apify/users/me/limits":
			reply(w, limitsStatus, `{"data":{"current":{"monthlyUsageUsd":1.5},"limits":{"monthlyUsageUsd":5}}}`)
		default:
			reply(w, 404, "")
		}
	})
	ctx := context.Background()
	meStatus, limitsStatus = 200, 200
	if r := c.Apify(ctx, "tok"); !r.OK || r.Message != "Token Apify valid (white, FREE). Pemakaian bulan ini $1.50 dari $5.00." || r.Detail["monthlyUsageUsd"] != "1.5" {
		t.Fatalf("%+v", r)
	}
	limitsStatus = 500
	if r := c.Apify(ctx, "tok"); !r.OK || r.Message != "Token Apify valid (white, FREE)." || r.Detail != nil {
		t.Fatalf("limits optional: %+v", r)
	}
	if r := c.Apify(ctx, "wrong"); r.OK || r.Message != "Token Apify ditolak." {
		t.Fatalf("%+v", r)
	}
	meStatus = 402
	if r := c.Apify(ctx, "tok"); r.OK || r.Message != "Kuota/kredit Apify habis (402)." {
		t.Fatalf("%+v", r)
	}
	meStatus = 500
	if r := c.Apify(ctx, "tok"); r.OK || r.Message != `Apify menjawab 500: {"data":{"username":"white","plan":{"id":"FREE"}}}` {
		t.Fatalf("%+v", r)
	}
}

func TestNetworkFailures(t *testing.T) {
	dead := "http://127.0.0.1:1"
	c := &Client{HTTP: httpx.New(time.Second), MetaGraphBase: dead, GoogleTokenURL: dead, DataForSeoBase: dead, OpenRouterBase: dead, ApifyBase: dead}
	ctx := context.Background()
	prefixes := map[string]string{
		c.MetaApp(ctx, "a", "b").Message:      "Tidak bisa menghubungi Meta: ",
		c.GoogleClient(ctx, "a", "b").Message: "Tidak bisa menghubungi Google: ",
		c.DataForSeo(ctx, "a", "b").Message:   "Tidak bisa menghubungi DataForSEO: ",
		c.OpenRouter(ctx, "a").Message:        "Tidak bisa menghubungi OpenRouter: ",
		c.Apify(ctx, "a").Message:             "Tidak bisa menghubungi Apify: ",
	}
	for msg, prefix := range prefixes {
		if len(msg) <= len(prefix) || msg[:len(prefix)] != prefix {
			t.Errorf("%q should start with %q", msg, prefix)
		}
	}
}
