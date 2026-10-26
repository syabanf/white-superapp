package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const goodPage = `<!doctype html><html><head>
<title>Kopi Nusantara — kedai kopi susu gula aren di Jakarta</title>
<meta name="description" content="Kopi susu gula aren dan biji kopi lokal pilihan.">
<link rel="canonical" href="/">
<link rel="alternate" hreflang="en" href="/en">
<script type="application/ld+json">{"@type":"Cafe"}</script>
</head><body><h1>Kopi Nusantara</h1><p>%s</p>
<img src="/a.jpg" alt="a"><a href="/menu">Menu</a><a href="/menu#x">Menu lagi</a>
<a href="https://instagram.com/kopi">IG</a><a href="mailto:x@y.z">mail</a></body></html>`

func TestAnalyzeHTMLClean(t *testing.T) {
	html := fmt.Sprintf(goodPage, strings.Repeat("kata ", 320))
	load := 120.4
	out, err := AnalyzeHTML(AnalyzeInput{URL: "https://kopi.id/", HTML: html, StatusCode: 200, LoadMs: &load})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Issues) != 0 {
		t.Fatalf("unexpected issues %+v", out.Issues)
	}
	p := out.Page
	if p.H1Count != 1 || p.InternalLinks != 1 || p.ExternalLinks != 1 || !p.HasJsonLd || !p.HasHreflang || !p.Indexable || *p.LoadMs != 120 || *p.Canonical != "https://kopi.id/" {
		t.Fatalf("%+v", p)
	}
}

func TestAnalyzeHTMLIssues(t *testing.T) {
	html := `<html><head><title>Short</title><meta name="robots" content="noindex"></head>
<body><h1>a</h1><h1>b</h1><img src="http://cdn/x.png"><img src="/y.png" alt=""></body></html>`
	load := 2000.0
	out, _ := AnalyzeHTML(AnalyzeInput{URL: "https://kopi.id/p", HTML: html, StatusCode: 200, LoadMs: &load, RedirectHops: 2})
	codes := map[string]Issue{}
	for _, i := range out.Issues {
		codes[i.Code] = i
	}
	for _, c := range []string{"TITLE_TOO_SHORT", "MISSING_META_DESCRIPTION", "MULTIPLE_H1", "IMG_MISSING_ALT", "MISSING_CANONICAL", "NOINDEX", "THIN_CONTENT", "NO_STRUCTURED_DATA", "MIXED_CONTENT", "SLOW_RESPONSE", "REDIRECT_CHAIN"} {
		if _, ok := codes[c]; !ok {
			t.Errorf("missing %s", c)
		}
	}
	if codes["IMG_MISSING_ALT"].Message != "1 gambar tanpa atribut alt." || codes["MIXED_CONTENT"].Severity != SeverityError {
		t.Fatal("issue details")
	}
	if out.Page.Indexable {
		t.Fatal("noindex page is not indexable")
	}
	errPage, _ := AnalyzeHTML(AnalyzeInput{URL: "https://kopi.id/x", HTML: "<html></html>", StatusCode: 404})
	if len(errPage.Issues) != 0 {
		t.Fatal("content checks skip error responses")
	}
}

func TestRobots(t *testing.T) {
	txt := "User-agent: *\nDisallow: /\n\nUser-agent: WHITEAnalyticsBot\nUser-agent: other\nDisallow: /private\nAllow: /private/ok$\nDisallow: /*.pdf$\nSitemap: https://kopi.id/sm.xml\n"
	r := ParseRobots(txt, UAToken)
	if len(r.Sitemaps) != 1 {
		t.Fatal("sitemaps")
	}
	cases := map[string]bool{"/": true, "/private/x": false, "/private/ok": true, "/private/ok/more": false, "/files/a.pdf": false, "/files/a.pdfx": true}
	for path, want := range cases {
		if IsAllowed(r, path) != want {
			t.Errorf("%s: want %v", path, want)
		}
	}
	star := ParseRobots("User-agent: *\nDisallow: /tmp\nAllow: /tmp\n", UAToken)
	if !IsAllowed(star, "/tmp/a") {
		t.Fatal("allow wins ties")
	}
}

func TestParseSitemap(t *testing.T) {
	urls, _ := ParseSitemap(`<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc> https://kopi.id/a </loc></url><url><loc>https://kopi.id/b</loc></url></urlset>`)
	if len(urls) != 2 || urls[0] != "https://kopi.id/a" {
		t.Fatalf("%v", urls)
	}
	_, children := ParseSitemap(`<sitemapindex><sitemap><loc>https://kopi.id/s1.xml</loc></sitemap></sitemapindex>`)
	if len(children) != 1 {
		t.Fatal("index")
	}
	if u, s := ParseSitemap("not xml"); u != nil || s != nil {
		t.Fatal("malformed")
	}
}

func TestCrawlSite(t *testing.T) {
	mux := http.NewServeMux()
	page := func(body string) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, body)
		}
	}
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "User-agent: *\nDisallow: /secret\n") })
	mux.HandleFunc("/", page(`<html><head><title>Home</title></head><body><h1>x</h1><a href="/about">a</a><a href="/secret">s</a><a href="/gone">g</a></body></html>`))
	mux.HandleFunc("/about", page(`<html><body><h1>about</h1></body></html>`))
	mux.HandleFunc("/gone", func(w http.ResponseWriter, _ *http.Request) { http.NotFound(w, nil) })
	srv := httptest.NewServer(mux)
	defer srv.Close()
	res, err := (&Crawler{}).Crawl(context.Background(), srv.URL+"/", Options{MaxPages: 10, TimeBudget: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	urls := map[string]int{}
	for _, p := range res.Pages {
		urls[strings.TrimPrefix(p.URL, srv.URL)] = p.StatusCode
	}
	if _, ok := urls["/secret"]; ok {
		t.Fatal("robots disallow ignored")
	}
	if urls["/about"] != 200 || urls["/gone"] != 404 {
		t.Fatalf("%v", urls)
	}
	broken := false
	for _, i := range res.Issues {
		if i.Code == "BROKEN_INTERNAL_LINK" && strings.Contains(i.Message, "/gone") {
			broken = true
		}
	}
	if !broken {
		t.Fatal("broken link not reported")
	}
}
