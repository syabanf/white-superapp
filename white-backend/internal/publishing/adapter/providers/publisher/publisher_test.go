package publisher

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/publishing/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// rewrite sends every Graph call to the test server instead of graph.facebook.com.
type rewrite struct{ target *url.URL }

func (r rewrite) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = r.target.Scheme
	req.URL.Host = r.target.Host
	return http.DefaultTransport.RoundTrip(req)
}

type call struct {
	Method string
	Path   string
	Form   url.Values
}

type graphStub struct {
	mu        sync.Mutex
	calls     []call
	responses map[string][]string // key "METHOD path" -> queued JSON bodies
	status    map[string]int
}

func newStub() *graphStub {
	return &graphStub{responses: map[string][]string{}, status: map[string]int{}}
}

func (g *graphStub) on(method, path string, bodies ...string) {
	g.responses[method+" "+path] = append(g.responses[method+" "+path], bodies...)
}

func (g *graphStub) handler(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	defer g.mu.Unlock()
	_ = r.ParseForm()
	path := strings.TrimPrefix(r.URL.Path, "/v21.0")
	g.calls = append(g.calls, call{Method: r.Method, Path: path, Form: r.Form})
	key := r.Method + " " + path
	queue := g.responses[key]
	if len(queue) == 0 {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"error":{"message":"unexpected call ` + key + `","code":100}}`))
		return
	}
	body := queue[0]
	if len(queue) > 1 {
		g.responses[key] = queue[1:]
	}
	if st := g.status[key]; st != 0 {
		w.WriteHeader(st)
	}
	_, _ = w.Write([]byte(body))
}

func newReal(t *testing.T, stub *graphStub) *Real {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(stub.handler))
	t.Cleanup(srv.Close)
	target, _ := url.Parse(srv.URL)
	r := NewReal()
	r.Client.HTTP.HTTP.Transport = rewrite{target}
	r.PollInterval = time.Millisecond
	r.PollMax = 3
	return r
}

func TestInstagramSingleImage(t *testing.T) {
	stub := newStub()
	stub.on("POST", "/178/media", `{"id":"c1"}`)
	stub.on("POST", "/178/media_publish", `{"id":"m1"}`)
	stub.on("GET", "/m1", `{"permalink":"https://www.instagram.com/p/abc/"}`)
	stub.on("POST", "/m1/comments", `{"id":"cm"}`)
	res, err := newReal(t, stub).Publish(context.Background(), domain.PublishInput{Platform: domain.PlatformInstagram, AccessToken: "tok", AccountExternalID: "178", Text: "halo", Media: []domain.PublishMedia{{URL: "https://x/a.jpg", Kind: domain.MediaImage}}, FirstComment: " #tag "})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExternalID != "m1" || res.Permalink == nil || *res.Permalink != "https://www.instagram.com/p/abc/" {
		t.Fatalf("%+v", res)
	}
	first := stub.calls[0]
	if first.Form.Get("image_url") != "https://x/a.jpg" || first.Form.Get("caption") != "halo" || first.Form.Get("access_token") != "tok" {
		t.Fatalf("%+v", first.Form)
	}
	if stub.calls[1].Form.Get("creation_id") != "c1" || stub.calls[3].Form.Get("message") != "#tag" {
		t.Fatalf("%+v", stub.calls)
	}
}

func TestInstagramCarouselWithVideoPolling(t *testing.T) {
	stub := newStub()
	stub.on("POST", "/178/media", `{"id":"img"}`, `{"id":"vid"}`, `{"id":"car"}`)
	stub.on("GET", "/vid", `{"status_code":"IN_PROGRESS"}`, `{"status_code":"FINISHED"}`)
	stub.on("POST", "/178/media_publish", `{"id":"m2"}`)
	stub.on("GET", "/m2", `{"permalink":"https://www.instagram.com/p/def/"}`)
	res, err := newReal(t, stub).Publish(context.Background(), domain.PublishInput{Platform: domain.PlatformInstagram, AccessToken: "tok", AccountExternalID: "178", Text: "caro", Media: []domain.PublishMedia{{URL: "https://x/a.jpg", Kind: domain.MediaImage}, {URL: "https://x/b.mp4", Kind: domain.MediaVideo}}})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExternalID != "m2" {
		t.Fatal(res.ExternalID)
	}
	if stub.calls[0].Form.Get("is_carousel_item") != "true" || stub.calls[1].Form.Get("media_type") != "VIDEO" || stub.calls[1].Form.Get("video_url") != "https://x/b.mp4" {
		t.Fatalf("%+v", stub.calls[:2])
	}
	polls := 0
	for _, c := range stub.calls {
		if c.Method == "GET" && c.Path == "/vid" {
			polls++
		}
	}
	if polls != 2 {
		t.Fatalf("polls = %d", polls)
	}
	var carousel call
	for _, c := range stub.calls {
		if c.Form.Get("media_type") == "CAROUSEL" {
			carousel = c
		}
	}
	if carousel.Form.Get("children") != "img,vid" || carousel.Form.Get("caption") != "caro" {
		t.Fatalf("%+v", carousel.Form)
	}
}

func TestInstagramReelsErrorsAndTimeout(t *testing.T) {
	stub := newStub()
	stub.on("POST", "/178/media", `{"id":"v"}`)
	stub.on("GET", "/v", `{"status_code":"ERROR"}`)
	_, err := newReal(t, stub).Publish(context.Background(), domain.PublishInput{Platform: domain.PlatformInstagram, AccountExternalID: "178", Media: []domain.PublishMedia{{URL: "u", Kind: domain.MediaVideo}}})
	if e := apperr.As(err); e.Code != apperr.Unknown || !strings.Contains(e.Message, "ERROR") {
		t.Fatalf("%v", err)
	}
	if stub.calls[0].Form.Get("media_type") != "REELS" {
		t.Fatal("single video should be REELS")
	}

	stub = newStub()
	stub.on("POST", "/178/media", `{"id":"v"}`)
	stub.on("GET", "/v", `{"status_code":"IN_PROGRESS"}`)
	_, err = newReal(t, stub).Publish(context.Background(), domain.PublishInput{Platform: domain.PlatformInstagram, AccountExternalID: "178", Media: []domain.PublishMedia{{URL: "u", Kind: domain.MediaVideo}}})
	if e := apperr.As(err); e.Code != apperr.Network {
		t.Fatalf("%v", err)
	}

	_, err = newReal(t, newStub()).Publish(context.Background(), domain.PublishInput{Platform: domain.PlatformInstagram, AccountExternalID: "178"})
	if e := apperr.As(err); e.Code != apperr.Unknown || e.Provider != ProviderName {
		t.Fatalf("%v", err)
	}
}

func TestFacebookVariants(t *testing.T) {
	ctx := context.Background()
	// Video.
	stub := newStub()
	stub.on("POST", "/p1/videos", `{"id":"v1"}`)
	stub.on("GET", "/v1", `{"permalink_url":"https://fb/v1"}`)
	res, err := newReal(t, stub).Publish(ctx, domain.PublishInput{Platform: domain.PlatformFacebook, AccountExternalID: "p1", Text: "desc", Media: []domain.PublishMedia{{URL: "u.mp4", Kind: domain.MediaVideo}}})
	if err != nil || res.ExternalID != "v1" || *res.Permalink != "https://fb/v1" || stub.calls[0].Form.Get("description") != "desc" || stub.calls[0].Form.Get("file_url") != "u.mp4" {
		t.Fatalf("%+v %v", res, err)
	}
	// One photo: post_id wins over id; permalink lookup failure is swallowed.
	stub = newStub()
	stub.on("POST", "/p1/photos", `{"id":"ph","post_id":"p1_post"}`)
	res, err = newReal(t, stub).Publish(ctx, domain.PublishInput{Platform: domain.PlatformFacebook, AccountExternalID: "p1", Text: "msg", Media: []domain.PublishMedia{{URL: "u.jpg", Kind: domain.MediaImage}}})
	if err != nil || res.ExternalID != "p1_post" || res.Permalink != nil || stub.calls[0].Form.Get("message") != "msg" {
		t.Fatalf("%+v %v", res, err)
	}
	// Several photos: unpublished uploads then feed with attached_media.
	stub = newStub()
	stub.on("POST", "/p1/photos", `{"id":"a"}`, `{"id":"b"}`)
	stub.on("POST", "/p1/feed", `{"id":"feed1"}`)
	stub.on("GET", "/feed1", `{"permalink_url":"https://fb/feed1"}`)
	stub.on("POST", "/feed1/comments", `{"id":"c"}`)
	res, err = newReal(t, stub).Publish(ctx, domain.PublishInput{Platform: domain.PlatformFacebook, AccountExternalID: "p1", Text: "multi", FirstComment: "hi", Media: []domain.PublishMedia{{URL: "a.jpg", Kind: domain.MediaImage}, {URL: "b.jpg", Kind: domain.MediaImage}}})
	if err != nil || res.ExternalID != "feed1" {
		t.Fatalf("%+v %v", res, err)
	}
	if stub.calls[0].Form.Get("published") != "false" {
		t.Fatal("photos should be unpublished")
	}
	feed := stub.calls[2].Form
	var att map[string]string
	_ = json.Unmarshal([]byte(feed.Get("attached_media[1]")), &att)
	if feed.Get("message") != "multi" || att["media_fbid"] != "b" {
		t.Fatalf("%+v", feed)
	}
	if stub.calls[4].Path != "/feed1/comments" || stub.calls[4].Form.Get("message") != "hi" {
		t.Fatalf("%+v", stub.calls[4])
	}
	// Text with link.
	stub = newStub()
	stub.on("POST", "/p1/feed", `{"id":"t1"}`)
	res, err = newReal(t, stub).Publish(ctx, domain.PublishInput{Platform: domain.PlatformFacebook, AccountExternalID: "p1", Text: "txt", LinkURL: "https://kopi.id"})
	if err != nil || res.ExternalID != "t1" || stub.calls[0].Form.Get("link") != "https://kopi.id" {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestErrorMappingAndTikTok(t *testing.T) {
	stub := newStub()
	stub.status["POST /p1/feed"] = 400
	stub.on("POST", "/p1/feed", `{"error":{"message":"Error validating access token","type":"OAuthException","code":190}}`)
	_, err := newReal(t, stub).Publish(context.Background(), domain.PublishInput{Platform: domain.PlatformFacebook, AccountExternalID: "p1", Text: "x"})
	e := apperr.As(err)
	if e.Code != apperr.TokenExpired || !e.RequiresReconnect || e.Provider != ProviderName {
		t.Fatalf("%+v", e)
	}
	stub = newStub()
	stub.status["POST /p1/feed"] = 403
	stub.on("POST", "/p1/feed", `{"error":{"message":"(#200) Requires pages_manage_posts","code":200}}`)
	_, err = newReal(t, stub).Publish(context.Background(), domain.PublishInput{Platform: domain.PlatformFacebook, AccountExternalID: "p1", Text: "x"})
	if apperr.As(err).Code != apperr.Permission {
		t.Fatal(err)
	}
	_, err = newReal(t, newStub()).Publish(context.Background(), domain.PublishInput{Platform: domain.PlatformTikTok})
	if e := apperr.As(err); e.Code != apperr.Permission || e.Provider != "tiktok" {
		t.Fatal(err)
	}
}

func TestMock(t *testing.T) {
	in := domain.PublishInput{Platform: domain.PlatformInstagram, AccountExternalID: "178", Text: "halo", Media: []domain.PublishMedia{{URL: "u", Kind: domain.MediaImage}}}
	a, err := Mock{}.Publish(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Mock{}.Publish(context.Background(), in)
	if a.ExternalID != b.ExternalID || *a.Permalink != *b.Permalink || !strings.HasPrefix(*a.Permalink, "https://www.instagram.com/p/") || !strings.HasPrefix(a.ExternalID, "mock_instagram_") {
		t.Fatalf("%+v", a)
	}
	in.Platform = domain.PlatformFacebook
	fb, _ := Mock{}.Publish(context.Background(), in)
	if !strings.Contains(*fb.Permalink, "facebook.com/178/posts/") {
		t.Fatal(*fb.Permalink)
	}
	_, err = Mock{}.Publish(context.Background(), domain.PublishInput{Platform: domain.PlatformFacebook, Text: "x " + MockFailMarker})
	if e := apperr.As(err); e.Provider != "mock-publisher" || e.Code != apperr.Unknown {
		t.Fatal(err)
	}
	if Pick(&Real{}, Mock{}, true, true) == (Mock{}) || Pick(&Real{}, Mock{}, true, false) != (Mock{}) || Pick(&Real{}, Mock{}, false, true) != (Mock{}) {
		t.Fatal("pick")
	}
}
