package metagraph

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// rewrite sends every request to the test server, keeping path and query.
type rewrite struct{ target *url.URL }

func (r rewrite) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme, req.URL.Host = r.target.Scheme, r.target.Host
	return http.DefaultTransport.RoundTrip(req)
}

func testReal(t *testing.T, h http.HandlerFunc) *Real {
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	r := NewReal()
	r.Client.HTTP.HTTP.Transport = rewrite{u}
	return r
}

func TestRealMediaAndInsights(t *testing.T) {
	r := testReal(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Query().Get("access_token") != "tok" {
			t.Errorf("token missing")
		}
		switch req.URL.Path {
		case "/v21.0/ig1/media":
			io.WriteString(w, `{"data":[{"id":"m1","caption":"hi","like_count":5,"comments_count":2,"media_type":"VIDEO","media_product_type":"REELS","media_url":"u","timestamp":"2026-09-01T10:00:00+0000"},{"id":"m2","media_type":"WEIRD"}]}`)
		case "/v21.0/ig1/insights":
			io.WriteString(w, `{"data":[{"name":"reach","values":[{"value":100,"end_time":"2026-09-02T07:00:00+0000"},{"value":120,"end_time":"2026-09-01T07:00:00+0000"}]},{"name":"profile_views","values":[{"value":{"x":1},"end_time":"2026-09-01T07:00:00+0000"}]}]}`)
		case "/v21.0/bad/media":
			io.WriteString(w, `{"error":{"message":"expired","type":"OAuthException","code":190}}`)
		}
	})
	media, err := r.GetInstagramMedia(context.Background(), "tok", "ig1", 50)
	if err != nil {
		t.Fatal(err)
	}
	if media[0].MediaType != "VIDEO" || *media[0].MediaProductType != "REELS" || *media[0].ThumbnailURL != "u" || media[0].Timestamp.Hour() != 10 {
		t.Fatalf("%+v", media[0])
	}
	if media[1].MediaType != "IMAGE" || media[1].MediaProductType != nil || media[1].Timestamp.Unix() != 0 {
		t.Fatalf("%+v", media[1])
	}
	ins, err := r.GetInstagramInsights(context.Background(), "tok", "ig1", time.Now(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(ins) != 2 || ins[0].Date != "2026-09-01" || ins[0].Reach != 120 || ins[0].ProfileViews != 0 {
		t.Fatalf("%+v", ins)
	}
	_, err = r.GetInstagramMedia(context.Background(), "tok", "bad", 1)
	if apperr.As(err).Code != apperr.TokenExpired {
		t.Fatalf("%v", err)
	}
}

func TestMockDeterministic(t *testing.T) {
	now := func() time.Time { return time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC) }
	m := Mock{Now: now}
	a, _ := m.DiscoverBusiness(context.Background(), "", "", "@KopiKenangan")
	b, _ := m.DiscoverBusiness(context.Background(), "", "", "kopikenangan")
	if a.FollowersCount != b.FollowersCount || len(a.Media) != 25 || a.Media[0].ID != b.Media[0].ID {
		t.Fatal("not deterministic")
	}
	series := BuildFollowerSeries("bd:kopikenangan", 120, 50000)
	if len(series) != 120 || series[119] != 50000 {
		t.Fatalf("series must end exactly at the target: %d", series[119])
	}
	if !equal(series, BuildFollowerSeries("bd:kopikenangan", 120, 50000)) {
		t.Fatal("series not deterministic")
	}
}

func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
