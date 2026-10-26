package pagespeed

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

func TestRunMapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Query().Get("url") == "https://quota.id/" {
			w.WriteHeader(429)
			io.WriteString(w, `{"error":{"message":"quota"}}`)
			return
		}
		if len(q["category"]) != 4 || q.Get("key") != "k" || q.Get("strategy") != "mobile" {
			t.Errorf("query %v", q)
		}
		io.WriteString(w, `{"lighthouseResult":{"finalDisplayedUrl":"https://kopi.id/","categories":{"performance":{"score":0.874},"seo":{"score":1},"accessibility":{"score":0.9},"best-practices":{"score":null}},"audits":{"largest-contentful-paint":{"numericValue":2400.5,"displayValue":"2,4 s"},"experimental-interaction-to-next-paint":{"numericValue":180},"cumulative-layout-shift":{"numericValue":0.05}}}}`)
	}))
	defer srv.Close()
	c := New("k")
	c.Endpoint = srv.URL
	r, err := c.Run(context.Background(), "https://kopi.id", "mobile")
	if err != nil {
		t.Fatal(err)
	}
	if r.Scores.Performance != 87 || r.Scores.Seo != 100 || r.Scores.BestPractices != 0 || r.FinalURL != "https://kopi.id/" {
		t.Fatalf("%+v", r.Scores)
	}
	if *r.Metrics.LcpMs != 2400.5 || *r.Metrics.InpMs != 180 || r.Metrics.TtfbMs != nil {
		t.Fatalf("%+v", r.Metrics)
	}
	var raw map[string]map[string]any
	_ = json.Unmarshal(r.Raw, &raw)
	if raw["categories"]["performance"].(float64) != 87 || raw["audits"]["largest-contentful-paint"] == nil {
		t.Fatalf("raw %s", r.Raw)
	}
	_, err = c.Run(context.Background(), "https://quota.id/", "mobile")
	if e := apperr.As(err); e.Code != apperr.RateLimit || e.Message != "Kuota PageSpeed Insights tercapai. Coba lagi beberapa menit. (quota)" {
		t.Fatalf("%+v", e)
	}
}
