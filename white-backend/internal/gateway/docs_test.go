package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDocsRoutes(t *testing.T) {
	mux := http.NewServeMux()
	mountDocs(mux)
	mux.HandleFunc("/", http.NotFound)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/openapi.json", nil))
	var spec struct {
		Swagger string         `json:"swagger"`
		Paths   map[string]any `json:"paths"`
	}
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &spec) != nil || spec.Swagger != "2.0" || spec.Paths["/v1/auth/login"] == nil {
		t.Fatalf("openapi: %d %s", rec.Code, rec.Body.String()[:min(80, rec.Body.Len())])
	}
	for _, p := range []string{"/", "/docs"} {
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", p, nil))
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), "SwaggerUIBundle") {
			t.Fatalf("%s: %d", p, rec.Code)
		}
	}
	// Unknown paths still fall through to the API mux.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/nope", nil))
	if rec.Code != 404 {
		t.Fatalf("/nope: %d", rec.Code)
	}
}
