package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

const (
	providerName = "crawler"
	hardCap      = 60
	maxRedirects = 5
)

var binaryRe = regexp.MustCompile(`(?i)\.(png|jpe?g|gif|webp|avif|svg|ico|css|js|mjs|json|pdf|zip|rar|7z|gz|tar|mp3|mp4|webm|mov|avi|wmv|woff2?|ttf|otf|eot|txt|xml|rss|atom|exe|dmg|apk)(\?.*)?$`)

type Options struct {
	// MaxPages defaults to 30 and is capped at 60.
	MaxPages int
	// TimeBudget defaults to 45 s.
	TimeBudget     time.Duration
	Concurrency    int
	RequestTimeout time.Duration
}

type Crawler struct {
	Transport http.RoundTripper
}

func normalizeStart(input string) (string, error) {
	raw := strings.TrimSpace(input)
	if strings.HasPrefix(raw, "sc-domain:") {
		raw = "https://" + strings.TrimPrefix(raw, "sc-domain:") + "/"
	}
	if !regexp.MustCompile(`(?i)^https?://`).MatchString(raw) {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("invalid url")
	}
	u.Fragment = ""
	return u.String(), nil
}

func normalizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	u.Fragment = ""
	return u.String()
}

func origin(u *url.URL) string { return u.Scheme + "://" + u.Host }

type fetchOutcome struct {
	status     int
	finalURL   string
	hops       int
	loadMs     float64
	html       *string
	xRobotsTag string
}

func (c *Crawler) client(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:       timeout,
		Transport:     c.Transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// fetchPage follows redirects by hand (max 5) and reads the body only for HTML.
func (c *Crawler) fetchPage(ctx context.Context, target string, timeout time.Duration, method string) fetchOutcome {
	started := time.Now()
	current := target
	hops := 0
	cl := c.client(timeout)
	for {
		req, err := http.NewRequestWithContext(ctx, method, current, nil)
		if err != nil {
			return fetchOutcome{status: 0, finalURL: current, hops: hops}
		}
		req.Header.Set("User-Agent", UserAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*;q=0.8")
		res, err := cl.Do(req)
		if err != nil {
			return fetchOutcome{status: 0, finalURL: current, hops: hops, loadMs: ms(started)}
		}
		if res.StatusCode >= 300 && res.StatusCode < 400 {
			loc := res.Header.Get("Location")
			res.Body.Close()
			if loc == "" || hops >= maxRedirects {
				return fetchOutcome{status: res.StatusCode, finalURL: current, hops: hops, loadMs: ms(started), xRobotsTag: res.Header.Get("X-Robots-Tag")}
			}
			base, _ := url.Parse(current)
			ref, err := url.Parse(loc)
			if err != nil || base == nil {
				return fetchOutcome{status: res.StatusCode, finalURL: current, hops: hops, loadMs: ms(started)}
			}
			current = base.ResolveReference(ref).String()
			hops++
			continue
		}
		ct := res.Header.Get("Content-Type")
		isHTML := strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml")
		var html *string
		if method == http.MethodGet && isHTML {
			if b, err := io.ReadAll(io.LimitReader(res.Body, 4<<20)); err == nil {
				s := string(b)
				html = &s
			}
		}
		res.Body.Close()
		return fetchOutcome{status: res.StatusCode, finalURL: current, hops: hops, loadMs: ms(started), html: html, xRobotsTag: res.Header.Get("X-Robots-Tag")}
	}
}

func ms(since time.Time) float64 { return float64(time.Since(since).Microseconds()) / 1000 }

func (c *Crawler) fetchText(ctx context.Context, target string, timeout time.Duration) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", UserAgent)
	res, err := c.client(timeout).Do(req)
	if err != nil {
		return ""
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return ""
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return ""
	}
	return string(b)
}

// collectSeeds reads up to 4 sitemap documents (3 children each) for same-origin URLs.
func (c *Crawler) collectSeeds(ctx context.Context, org string, rules Rules, cap int) []string {
	seen := map[string]bool{}
	var queue []string
	for _, s := range append(append([]string{}, rules.Sitemaps...), org+"/sitemap.xml") {
		if !seen[s] {
			seen[s] = true
			queue = append(queue, s)
		}
	}
	if len(queue) > 4 {
		queue = queue[:4]
	}
	var urls []string
	fetched := 0
	for len(queue) > 0 && fetched < 4 && len(urls) < cap {
		sm := queue[0]
		queue = queue[1:]
		fetched++
		x := c.fetchText(ctx, sm, 5*time.Second)
		if x == "" {
			continue
		}
		pageURLs, children := ParseSitemap(x)
		if len(children) > 3 {
			children = children[:3]
		}
		queue = append(queue, children...)
		for _, u := range pageURLs {
			if n := normalizeURL(u); n != "" && strings.HasPrefix(n, org) {
				urls = append(urls, n)
			}
			if len(urls) >= cap {
				break
			}
		}
	}
	return urls
}

// Crawl runs a BFS over one origin until maxPages or the time budget.
func (c *Crawler) Crawl(ctx context.Context, startURL string, opt Options) (*Result, error) {
	started := time.Now()
	maxPages := opt.MaxPages
	if maxPages <= 0 {
		maxPages = 30
	}
	if maxPages > hardCap {
		maxPages = hardCap
	}
	budget := opt.TimeBudget
	if budget == 0 {
		budget = 45 * time.Second
	}
	concurrency := opt.Concurrency
	if concurrency <= 0 {
		concurrency = 4
	}
	reqTimeout := opt.RequestTimeout
	if reqTimeout == 0 {
		reqTimeout = 10 * time.Second
	}
	timeLeft := func() time.Duration { return budget - time.Since(started) }

	start, err := normalizeStart(startURL)
	if err != nil {
		return nil, apperr.FromProvider(providerName, apperr.Unknown, "URL situs tidak valid: "+startURL, nil)
	}
	startU, _ := url.Parse(start)
	org := origin(startU)

	rules := AllowAll
	if txt := c.fetchText(ctx, org+"/robots.txt", 5*time.Second); txt != "" {
		rules = ParseRobots(txt, UAToken)
	}
	var seeds []string
	if timeLeft() > 5*time.Second {
		seeds = c.collectSeeds(ctx, org, rules, maxPages*2)
	}

	var mu sync.Mutex
	queue := []string{}
	queued := map[string]bool{}
	enqueue := func(u string) {
		if queued[u] || len(queue) > maxPages*6 {
			return
		}
		queued[u] = true
		queue = append(queue, u)
	}
	enqueue(start)
	for _, s := range seeds {
		enqueue(s)
	}

	pages := []Page{}
	issues := []Issue{}
	issueKeys := map[string]bool{}
	pushIssue := func(i Issue) {
		k := i.Code + "|" + i.URL + "|" + i.Message
		if issueKeys[k] {
			return
		}
		issueKeys[k] = true
		issues = append(issues, i)
	}
	statusByURL := map[string]int{}
	linkTargets := map[string]string{}
	var linkOrder []string
	active := 0

	processURL := func(target string) {
		u, err := url.Parse(target)
		if err != nil || origin(u) != org || binaryRe.MatchString(u.Path) || !IsAllowed(rules, u.RequestURI()) {
			return
		}
		out := c.fetchPage(ctx, target, reqTimeout, http.MethodGet)
		if out.status == 0 {
			return
		}
		mu.Lock()
		statusByURL[target] = out.status
		statusByURL[out.finalURL] = out.status
		mu.Unlock()
		finalU := normalizeURL(out.finalURL)
		if finalU == "" || !strings.HasPrefix(finalU, org) {
			return
		}
		mu.Lock()
		full := len(pages) >= maxPages
		mu.Unlock()
		if full {
			return
		}
		if out.html == nil {
			if out.status >= 400 {
				load := int(out.loadMs + 0.5)
				mu.Lock()
				pages = append(pages, Page{URL: finalU, StatusCode: out.status, Indexable: false, LoadMs: &load})
				mu.Unlock()
			}
			return
		}
		load := out.loadMs
		analyzed, err := AnalyzeHTML(AnalyzeInput{URL: finalU, HTML: *out.html, StatusCode: out.status, LoadMs: &load, XRobotsTag: out.xRobotsTag, RedirectHops: out.hops})
		if err != nil {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		pages = append(pages, analyzed.Page)
		for _, i := range analyzed.Issues {
			pushIssue(i)
		}
		for _, link := range analyzed.InternalURLs {
			if _, ok := linkTargets[link]; !ok {
				linkTargets[link] = finalU
				linkOrder = append(linkOrder, link)
			}
			if !binaryRe.MatchString(link) {
				enqueue(link)
			}
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if timeLeft() < 1500*time.Millisecond || ctx.Err() != nil {
					return
				}
				mu.Lock()
				if len(pages) >= maxPages {
					mu.Unlock()
					return
				}
				if len(queue) == 0 {
					idle := active == 0
					mu.Unlock()
					if idle {
						return
					}
					time.Sleep(60 * time.Millisecond)
					continue
				}
				next := queue[0]
				queue = queue[1:]
				active++
				mu.Unlock()
				processURL(next)
				mu.Lock()
				active--
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if len(pages) == 0 {
		return nil, apperr.FromProvider(providerName, apperr.Network, fmt.Sprintf("Tidak ada halaman yang bisa di-crawl dari %s. Periksa URL dan koneksi.", org), nil)
	}

	brokenFor := func(target, referrer string, status int) {
		pushIssue(Issue{Code: "BROKEN_INTERNAL_LINK", Severity: IssueDefs["BROKEN_INTERNAL_LINK"].Severity, Message: fmt.Sprintf("Tautan internal mengarah ke halaman %d: %s", status, target), URL: referrer})
	}
	var unchecked [][2]string
	for _, target := range linkOrder {
		referrer := linkTargets[target]
		if known, ok := statusByURL[target]; ok {
			if known >= 400 {
				brokenFor(target, referrer, known)
			}
		} else if !binaryRe.MatchString(target) {
			unchecked = append(unchecked, [2]string{target, referrer})
		}
	}
	if len(unchecked) > 15 {
		unchecked = unchecked[:15]
	}
	idx := 0
	workers := len(unchecked)
	if workers > 4 {
		workers = 4
	}
	var cwg sync.WaitGroup
	for i := 0; i < workers; i++ {
		cwg.Add(1)
		go func() {
			defer cwg.Done()
			for {
				mu.Lock()
				if idx >= len(unchecked) || timeLeft() <= 1500*time.Millisecond {
					mu.Unlock()
					return
				}
				pair := unchecked[idx]
				idx++
				mu.Unlock()
				out := c.fetchPage(ctx, pair[0], 5*time.Second, http.MethodHead)
				if out.status == 405 || out.status == 501 {
					out = c.fetchPage(ctx, pair[0], 5*time.Second, http.MethodGet)
				}
				if out.status >= 400 {
					mu.Lock()
					brokenFor(pair[0], pair[1], out.status)
					mu.Unlock()
				}
			}
		}()
	}
	cwg.Wait()

	return &Result{Pages: pages, Issues: issues, PagesCrawled: len(pages), DurationMs: time.Since(started).Milliseconds()}, nil
}
