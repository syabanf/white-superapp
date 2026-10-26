package metamarketing

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type rewrite struct{ target *url.URL }

func (r rewrite) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme, req.URL.Host = r.target.Scheme, r.target.Host
	return http.DefaultTransport.RoundTrip(req)
}

func s(v string) *string { return &v }

func TestPickResultPriority(t *testing.T) {
	acts := []rawAction{{s("link_click"), s("40")}, {s("onsite_conversion.lead_grouped"), s("3")}, {s("offsite_conversion.fb_pixel_purchase"), s("2")}, {s("purchase"), s("1")}}
	n, typ := PickResult(acts)
	if *typ != "purchase" || n != 3 {
		t.Fatalf("%v %v", n, *typ)
	}
	n, typ = PickResult(acts[:2])
	if *typ != "lead" || n != 3 {
		t.Fatalf("lead %v", n)
	}
	n, typ = PickResult([]rawAction{{s("video_view"), s("9")}})
	if typ != nil || n != 0 {
		t.Fatal("unknown actions give no result")
	}
}

func TestInsightsPaging(t *testing.T) {
	var srvURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			io.WriteString(w, `{"data":[{"campaign_id":"c","adset_id":"s","ad_id":"a2","spend":"5","impressions":"50","date_start":"2026-09-02","actions":[{"action_type":"link_click","value":"4"}]}]}`)
			return
		}
		if r.URL.Query().Get("level") != "ad" || r.URL.Query().Get("time_increment") != "1" {
			t.Errorf("query %v", r.URL.Query())
		}
		fmt.Fprintf(w, `{"data":[{"campaign_id":"c","adset_id":"s","ad_id":"a1","spend":"10.5","impressions":"100","inline_link_clicks":"7","date_start":"2026-09-01","actions":[{"action_type":"purchase","value":"2"}],"action_values":[{"action_type":"purchase","value":"150000"}]}],"paging":{"next":"%s/v21.0/act_1/insights?page=2"}}`, srvURL)
	}))
	defer srv.Close()
	srvURL = srv.URL
	u, _ := url.Parse(srv.URL)
	r := NewReal("tok")
	r.Client.HTTP.HTTP.Transport = rewrite{u}
	rows, err := r.GetInsights(context.Background(), "act_1", InsightsQuery{Since: "2026-09-01", Until: "2026-09-02"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("paging not followed: %d", len(rows))
	}
	if rows[0].Spend != 10.5 || rows[0].LinkClicks != 7 || *rows[0].ResultType != "purchase" || *rows[0].PurchaseValue != 150000 {
		t.Fatalf("%+v", rows[0])
	}
	if *rows[1].ResultType != "link_click" || rows[1].PurchaseValue != nil {
		t.Fatalf("%+v", rows[1])
	}
}

func TestMockBlueprint(t *testing.T) {
	c, _ := Mock{}.ListCampaigns(context.Background(), "act_1")
	ads, _ := Mock{}.ListAds(context.Background(), "act_1")
	if len(c) != 3 || len(ads) != 7 || c[1].Objective != "OUTCOME_SALES" {
		t.Fatalf("%d campaigns %d ads", len(c), len(ads))
	}
	rows, _ := Mock{}.GetInsights(context.Background(), "act_1", InsightsQuery{Since: "2026-09-01", Until: "2026-09-02"})
	again, _ := Mock{}.GetInsights(context.Background(), "act_1", InsightsQuery{Since: "2026-09-01", Until: "2026-09-02"})
	if len(rows) != 14 || rows[3].Spend != again[3].Spend {
		t.Fatal("mock not deterministic")
	}
	demo, _ := Mock{}.GetInsights(context.Background(), "act_1", InsightsQuery{Since: "2026-09-01", Until: "2026-09-01", Breakdowns: []string{"age", "gender"}})
	if len(demo) != 7*18 || *demo[0].Age != "18-24" {
		t.Fatalf("%d", len(demo))
	}
}
