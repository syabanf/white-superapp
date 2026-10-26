package wagateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if r.URL.Path != "/send" || r.Header.Get("x-gateway-token") != "tok" || body["target"] != "6281234567890" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"success":false,"error":"Unauthorized"}`))
			return
		}
		_, _ = w.Write([]byte(`{"success":true,"messageId":"ABC"}`))
	}))
	defer srv.Close()
	g := New(srv.URL+"/", "tok", nil)
	id, err := g.Send(context.Background(), "6281234567890", "Halo")
	if err != nil || id != "ABC" {
		t.Fatalf("got %q %v", id, err)
	}
	if _, err := New(srv.URL, "salah", nil).Send(context.Background(), "6281234567890", "x"); err == nil {
		t.Fatal("expected error")
	}
	if New("", "", nil).Enabled() || !g.VerifyToken("tok") || g.VerifyToken("tok2") {
		t.Fatal("enabled/verify")
	}
}
