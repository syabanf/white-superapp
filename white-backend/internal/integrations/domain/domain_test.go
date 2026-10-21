package domain

import (
	"encoding/base64"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"
)

var now = time.Date(2026, 10, 21, 8, 0, 0, 0, time.UTC)

func TestOAuthStateRoundTrip(t *testing.T) {
	st, err := SignOAuthState("acme-id", "abc123", "secret", now)
	if err != nil {
		t.Fatal(err)
	}
	got := VerifyOAuthState(st, "secret", now.Add(5*time.Minute))
	if got == nil || got.Slug != "acme-id" || got.Nonce != "abc123" || got.Ts != now.UnixMilli() {
		t.Fatalf("got %+v", got)
	}
	if _, err := SignOAuthState("acme", "n", "", now); err != ErrNoStateSecret {
		t.Fatalf("expected ErrNoStateSecret, got %v", err)
	}
}

func TestOAuthStateRejects(t *testing.T) {
	st, _ := SignOAuthState("acme", "nonce", "secret", now)
	body, sig, _ := strings.Cut(st, ".")
	cases := map[string]struct {
		state  string
		secret string
		at     time.Time
	}{
		"expired":        {st, "secret", now.Add(OAuthStateMaxAge + time.Second)},
		"future":         {st, "secret", now.Add(-61 * time.Second)},
		"wrong secret":   {st, "other", now},
		"tampered body":  {body + "x." + sig, "secret", now},
		"tampered sig":   {body + "." + sig[:len(sig)-2] + "AA", "secret", now},
		"no dot":         {body, "secret", now},
		"trailing dot":   {body + ".", "secret", now},
		"empty":          {"", "secret", now},
		"empty secret":   {st, "", now},
		"bad slug":       {mustSign(t, OAuthState{Slug: "Acme Corp", Nonce: "n", Ts: now.UnixMilli()}, "secret"), "secret", now},
		"empty nonce":    {mustSign(t, OAuthState{Slug: "acme", Nonce: "", Ts: now.UnixMilli()}, "secret"), "secret", now},
		"non-json body":  {signRaw("not json", "secret"), "secret", now},
		"non-base64url":  {signRaw("%%%", "secret"), "secret", now},
		"within max age": {st, "secret", now.Add(OAuthStateMaxAge)},
		"within skew":    {st, "secret", now.Add(-59 * time.Second)},
	}
	for name, c := range cases {
		got := VerifyOAuthState(c.state, c.secret, c.at)
		wantOK := name == "within max age" || name == "within skew"
		if (got != nil) != wantOK {
			t.Errorf("%s: got %+v, want ok=%v", name, got, wantOK)
		}
	}
}

func mustSign(t *testing.T, s OAuthState, secret string) string {
	t.Helper()
	raw := `{"slug":"` + s.Slug + `","nonce":"` + s.Nonce + `","ts":` + strconv.FormatInt(s.Ts, 10) + `}`
	body := base64.RawURLEncoding.EncodeToString([]byte(raw))
	return body + "." + stateMAC(body, secret)
}

func signRaw(body, secret string) string { return body + "." + stateMAC(body, secret) }

func TestMaskSecret(t *testing.T) {
	cases := map[string]string{
		"":                  "••••",
		"12345678":          "••••",
		"123456789":         "1234••••6789",
		"sk-or-v1-abcdwxyz": "sk-o••••wxyz",
	}
	for in, want := range cases {
		if got := MaskSecret(in); got != want {
			t.Errorf("MaskSecret(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsConfiguredPair(t *testing.T) {
	meta := Integrations[KeyMeta]
	apify := Integrations[KeyApify]
	if IsConfiguredPair(meta, "id", "") || IsConfiguredPair(meta, "", "sec") || !IsConfiguredPair(meta, "id", "sec") {
		t.Fatal("meta pair rule broken")
	}
	if IsConfiguredPair(apify, "", "") || !IsConfiguredPair(apify, "", "tok") {
		t.Fatal("apify pair rule broken")
	}
}

func TestResolveDBWinsOverEnv(t *testing.T) {
	env := func(k string) string {
		return map[string]string{"META_APP_ID": "env-id", "META_APP_SECRET": "env-secret"}[k]
	}
	def := Integrations[KeyMeta]

	st, creds := Resolve(def, nil, "", env)
	if st.Source != SourceEnv || !st.Configured || creds.PublicID != "env-id" || creds.Secret != "env-secret" {
		t.Fatalf("env fallback: %+v %+v", st, creds)
	}

	id := "db-id"
	st, creds = Resolve(def, &IntegrationSetting{PublicID: &id}, "db-secret", env)
	if st.Source != SourceDB || creds.PublicID != "db-id" || creds.Secret != "db-secret" || *st.SecretMasked != "db-s••••cret" {
		t.Fatalf("db wins: %+v %+v", st, creds)
	}

	// A row that only carries meta (review checklist) is not "from db".
	st, _ = Resolve(def, &IntegrationSetting{Meta: []byte(`{"appLive":true}`)}, "", env)
	if st.Source != SourceEnv || string(st.Meta) != `{"appLive":true}` {
		t.Fatalf("meta-only row: %+v", st)
	}

	st, creds = Resolve(def, nil, "", func(string) string { return "" })
	if st.Source != SourceNone || st.Configured || st.PublicID != nil || st.SecretMasked != nil || creds.Configured {
		t.Fatalf("none: %+v", st)
	}
}

func TestParseMetaReviewAndProgress(t *testing.T) {
	empty := ParseMetaReview(nil)
	if len(empty.Permissions) != 0 || empty.AppLive {
		t.Fatalf("%+v", empty)
	}
	raw := []byte(`{"permissions":{"pages_show_list":"APPROVED","ads_read":"BOGUS","x":1},"appLive":1,"businessVerified":""}`)
	s := ParseMetaReview(raw)
	if len(s.Permissions) != 1 || s.Permissions["pages_show_list"] != ReviewApproved || !s.AppLive || s.BusinessVerified {
		t.Fatalf("%+v", s)
	}
	p := s.Progress()
	if p.Total != 8 || p.Approved != 1 || p.PublishReady || p.AdsReady {
		t.Fatalf("%+v", p)
	}

	all := MetaReviewState{Permissions: map[string]MetaReviewStatus{}, AppLive: false}
	for _, perm := range MetaPermissions {
		all.Permissions[perm.Name] = ReviewApproved
	}
	if p := all.Progress(); p.Approved != 8 || p.PublishReady || p.AdsReady {
		t.Fatalf("app not live should block readiness: %+v", p)
	}
	all.AppLive = true
	if p := all.Progress(); !p.PublishReady || !p.AdsReady {
		t.Fatalf("%+v", p)
	}
	delete(all.Permissions, "pages_manage_posts")
	if p := all.Progress(); p.PublishReady || !p.AdsReady || p.Approved != 7 {
		t.Fatalf("%+v", p)
	}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestEstimateSeoCost(t *testing.T) {
	e := EstimateSeoCost(SeoCostInput{TrackedKeywords: 100, CompetitorDomains: 2, ResearchRunsPerMonth: 4, UsdIdr: 16000})
	// rank 100*30*0.002=6, backlinks 30*0.02=0.6, domains 3*30*0.01=0.9, research 4*0.05=0.2
	if !near(e.LinesUSD["rank"], 6) || !near(e.LinesUSD["backlinks"], 0.6) || !near(e.LinesUSD["domains"], 0.9) || !near(e.LinesUSD["research"], 0.2) {
		t.Fatalf("%+v", e.LinesUSD)
	}
	if !near(e.TotalUSD, 7.7) || !near(e.TotalIDR, 123200) {
		t.Fatalf("total %v idr %v", e.TotalUSD, e.TotalIDR)
	}
	neg := EstimateSeoCost(SeoCostInput{TrackedKeywords: -5, CompetitorDomains: -1, ResearchRunsPerMonth: -1, UsdIdr: -1})
	if !near(neg.TotalUSD, 0.9) || neg.TotalIDR != 0 {
		t.Fatalf("clamp: %+v", neg)
	}
}

func TestEstimateApifyCost(t *testing.T) {
	e := EstimateApifyCost(ApifyCostInput{TrackedKeywords: 100, SocialAccounts: 3, PostsPerAccount: 9, UsdIdr: 16000})
	// serp 100*30*0.0035=10.5, social 3*30*(10*0.0023)=2.07
	if !near(e.LinesUSD["serp"], 10.5) || !near(e.LinesUSD["social"], 2.07) || !near(e.TotalUSD, 12.57) || !near(e.TotalIDR, 201120) {
		t.Fatalf("%+v", e)
	}
}

func TestConnectionExpiring(t *testing.T) {
	in := func(d time.Duration) *time.Time { v := now.Add(d); return &v }
	cases := []struct {
		exp  *time.Time
		want bool
	}{
		{nil, false},
		{in(2 * time.Minute), false},
		{in(RefreshSkew), false},
		{in(RefreshSkew - time.Millisecond), true},
		{in(-time.Hour), true},
	}
	for i, c := range cases {
		if got := (Connection{ExpiresAt: c.exp}).Expiring(now); got != c.want {
			t.Errorf("case %d: got %v want %v", i, got, c.want)
		}
	}
}
