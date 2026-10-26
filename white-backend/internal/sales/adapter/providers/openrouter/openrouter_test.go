package openrouter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/app"
)

func creds(key string) CredentialsFunc {
	return func(context.Context) (Credentials, error) { return Credentials{APIKey: key}, nil }
}

func TestMockWithoutKey(t *testing.T) {
	res, err := NewSource(creds(""), Options{}).Complete(context.Background(), app.AIRequest{})
	if err != nil || !res.Mock || res.Model != app.MockModel {
		t.Fatalf("got %+v %v", res, err)
	}
}

func TestRealRetriesWithoutJSONMode(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if r.Header.Get("Authorization") != "Bearer k" {
			t.Errorf("auth header %q", r.Header.Get("Authorization"))
		}
		if _, json := body["response_format"]; json {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":" {\"reply\":\"ok\"} "}}]}`))
	}))
	defer srv.Close()
	res, err := NewSource(creds("k"), Options{Endpoint: srv.URL}).Complete(context.Background(), app.AIRequest{JSON: true, Messages: []app.AIMessage{{Role: "user", Content: "hi"}}})
	if err != nil || res.Content != `{"reply":"ok"}` || res.Mock || calls != 2 {
		t.Fatalf("got %+v %v calls=%d", res, err, calls)
	}
}

func TestRealMapsErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) }))
	defer srv.Close()
	if _, err := NewSource(creds("k"), Options{Endpoint: srv.URL}).Complete(context.Background(), app.AIRequest{}); err == nil {
		t.Fatal("expected provider error")
	}
}
