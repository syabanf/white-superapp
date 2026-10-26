package googlesearch

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

func TestRealGscAndGa4(t *testing.T) {
	var gscBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("auth header %q", r.Header.Get("Authorization"))
		}
		switch {
		case strings.Contains(r.URL.EscapedPath(), "/sites/sc-domain%3Akopi.id/searchAnalytics/query"):
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &gscBody)
			io.WriteString(w, `{"rows":[{"keys":["2026-09-01"],"clicks":12,"impressions":400,"ctr":0.03,"position":4.5}]}`)
		case r.URL.Path == "/properties/123:runReport":
			io.WriteString(w, `{"rows":[{"dimensionValues":[{"value":"20260901"},{"value":"Organic Search"}],"metricValues":[{"value":"10"},{"value":"7"},{"value":"9"},{"value":"1"},{"value":"300"}]}]}`)
		case r.URL.Path == "/properties/403:runReport":
			w.WriteHeader(403)
			io.WriteString(w, `{"error":{"code":403,"message":"denied","status":"PERMISSION_DENIED"}}`)
		default:
			t.Errorf("unexpected %s", r.URL.String())
		}
	}))
	defer srv.Close()
	r := NewReal()
	r.GSCBase, r.GA4Base = srv.URL, srv.URL
	rows, err := r.QuerySearchAnalytics(context.Background(), "tok", GscQuery{SiteURL: "sc-domain:kopi.id", StartDate: "2026-09-01", EndDate: "2026-09-01", Dimensions: []string{"date"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].CTR != 3 || rows[0].Clicks != 12 {
		t.Fatalf("ctr must be x100: %+v", rows)
	}
	if gscBody["rowLimit"].(float64) != 5000 {
		t.Fatalf("default rowLimit %v", gscBody["rowLimit"])
	}
	ga, err := r.RunGa4Report(context.Background(), "tok", Ga4Request{PropertyID: "123", Metrics: []string{"sessions"}, Dimensions: []string{"date"}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ga[0].Dimensions, []string{"20260901", "Organic Search"}) || ga[0].Metrics[4] != 300 {
		t.Fatalf("%+v", ga)
	}
	_, err = r.RunGa4Report(context.Background(), "tok", Ga4Request{PropertyID: "properties/403"})
	if e := apperr.As(err); e.Code != apperr.Permission || !e.RequiresReconnect || !strings.Contains(e.Message, "(denied)") {
		t.Fatalf("%+v", e)
	}
}

func TestMockDeterministic(t *testing.T) {
	q := GscQuery{SiteURL: "sc-domain:kopi.id", StartDate: "2026-09-01", EndDate: "2026-09-03", Dimensions: []string{"date", "page"}, RowLimit: 5000}
	a, _ := Mock{}.QuerySearchAnalytics(context.Background(), "", q)
	b, _ := Mock{}.QuerySearchAnalytics(context.Background(), "", q)
	if len(a) != 30 || !reflect.DeepEqual(a, b) {
		t.Fatalf("rows %d", len(a))
	}
	if a[0].Keys[1] != "https://kopi.id/" {
		t.Fatalf("page key %v", a[0].Keys)
	}
	if xmur3("gsc") == xmur3("gsd") {
		t.Fatal("hash")
	}
	ga, _ := Mock{}.RunGa4Report(context.Background(), "", Ga4Request{PropertyID: "p", StartDate: "2026-09-01", EndDate: "2026-09-02", Metrics: []string{"sessions", "nope"}, Dimensions: []string{"date", "sessionDefaultChannelGroup"}})
	if len(ga) != 8 || ga[0].Dimensions[0] != "20260901" || ga[0].Metrics[1] != 0 {
		t.Fatalf("%+v", ga)
	}
}
