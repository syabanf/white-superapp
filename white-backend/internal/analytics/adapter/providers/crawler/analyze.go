package crawler

import (
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	titleMax    = 60
	titleMin    = 30
	metaDescMax = 160
	thinWords   = 300
	slowMs      = 1500
)

type AnalyzeInput struct {
	URL        string
	HTML       string
	StatusCode int
	// LoadMs is the request duration including redirects; nil when unknown.
	LoadMs *float64
	// XRobotsTag response header, may contain "noindex".
	XRobotsTag   string
	RedirectHops int
}

type AnalyzeOutput struct {
	Page         Page
	Issues       []Issue
	InternalURLs []string
	ExternalURLs []string
}

func makeIssue(code, pageURL, message string) Issue {
	def := IssueDefs[code]
	if message == "" {
		message = def.Message
	}
	return Issue{Code: code, Severity: def.Severity, Message: message, URL: pageURL}
}

var skipSchemeRe = regexp.MustCompile(`(?i)^(mailto:|tel:|javascript:|data:|ftp:)`)

// resolveLink resolves and normalises an href; nil for non-http(s) targets.
func resolveLink(href string, base *url.URL) *url.URL {
	t := strings.TrimSpace(href)
	if t == "" || strings.HasPrefix(t, "#") || skipSchemeRe.MatchString(t) {
		return nil
	}
	ref, err := url.Parse(t)
	if err != nil {
		return nil
	}
	u := base.ResolveReference(ref)
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil
	}
	u.Fragment = ""
	return u
}

var spaceRe = regexp.MustCompile(`\s+`)

// AnalyzeHTML extracts the on-page fields of one crawled page, emits issue
// rows and returns the discovered links. No network.
func AnalyzeHTML(in AnalyzeInput) (*AnalyzeOutput, error) {
	pageURL, err := url.Parse(in.URL)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(in.HTML))
	if err != nil {
		return nil, err
	}
	issues := []Issue{}

	title := strings.TrimSpace(doc.Find("head title").First().Text())
	if title == "" {
		title = strings.TrimSpace(doc.Find("title").First().Text())
	}

	var metaDescription *string
	robotsMeta := ""
	doc.Find("meta").Each(func(_ int, s *goquery.Selection) {
		name := strings.ToLower(strings.TrimSpace(s.AttrOr("name", "")))
		if name == "description" && metaDescription == nil {
			v := strings.TrimSpace(s.AttrOr("content", ""))
			metaDescription = &v
		}
		if name == "robots" {
			robotsMeta += " " + strings.ToLower(s.AttrOr("content", ""))
		}
	})
	if metaDescription != nil && *metaDescription == "" {
		metaDescription = nil
	}

	var canonical *string
	if href := strings.TrimSpace(doc.Find(`link[rel="canonical"]`).First().AttrOr("href", "")); href != "" {
		c := href
		if ref, err := url.Parse(href); err == nil {
			c = pageURL.ResolveReference(ref).String()
		}
		canonical = &c
	}

	noindex := strings.Contains(robotsMeta, "noindex") || strings.Contains(strings.ToLower(in.XRobotsTag), "noindex")
	indexable := !noindex && in.StatusCode < 400
	h1Count := doc.Find("h1").Length()

	body := doc.Find("body").Clone()
	body.Find("script,style,noscript,template,svg").Remove()
	text := strings.TrimSpace(spaceRe.ReplaceAllString(body.Text(), " "))
	wordCount := 0
	if text != "" {
		wordCount = len(strings.Fields(text))
	}

	imagesTotal, imagesMissingAlt := 0, 0
	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		imagesTotal++
		if _, ok := s.Attr("alt"); !ok {
			imagesMissingAlt++
		}
	})

	internal := map[string]bool{}
	external := map[string]bool{}
	var internalOrder, externalOrder []string
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		u := resolveLink(s.AttrOr("href", ""), pageURL)
		if u == nil {
			return
		}
		key := u.String()
		if sameOrigin(u, pageURL) {
			if key != pageURL.String() && !internal[key] {
				internal[key] = true
				internalOrder = append(internalOrder, key)
			}
		} else if !external[key] {
			external[key] = true
			externalOrder = append(externalOrder, key)
		}
	})

	hasJsonLd := doc.Find(`script[type="application/ld+json"]`).Length() > 0
	hasHreflang := doc.Find(`link[rel="alternate"][hreflang]`).Length() > 0

	mixed := false
	if pageURL.Scheme == "https" {
		doc.Find(`img[src], script[src], iframe[src], video[src], audio[src], source[src], link[rel="stylesheet"][href]`).Each(func(_ int, s *goquery.Selection) {
			v := strings.TrimSpace(s.AttrOr("src", s.AttrOr("href", "")))
			if strings.HasPrefix(strings.ToLower(v), "http://") {
				mixed = true
			}
		})
	}

	if in.StatusCode < 400 {
		switch {
		case title == "":
			issues = append(issues, makeIssue("MISSING_TITLE", in.URL, ""))
		case len([]rune(title)) > titleMax:
			issues = append(issues, makeIssue("TITLE_TOO_LONG", in.URL, ""))
		case len([]rune(title)) < titleMin:
			issues = append(issues, makeIssue("TITLE_TOO_SHORT", in.URL, ""))
		}
		switch {
		case metaDescription == nil:
			issues = append(issues, makeIssue("MISSING_META_DESCRIPTION", in.URL, ""))
		case len([]rune(*metaDescription)) > metaDescMax:
			issues = append(issues, makeIssue("META_DESCRIPTION_TOO_LONG", in.URL, ""))
		}
		switch {
		case h1Count == 0:
			issues = append(issues, makeIssue("MISSING_H1", in.URL, ""))
		case h1Count > 1:
			issues = append(issues, makeIssue("MULTIPLE_H1", in.URL, ""))
		}
		if imagesMissingAlt > 0 {
			issues = append(issues, makeIssue("IMG_MISSING_ALT", in.URL, fmt.Sprintf("%d gambar tanpa atribut alt.", imagesMissingAlt)))
		}
		if canonical == nil {
			issues = append(issues, makeIssue("MISSING_CANONICAL", in.URL, ""))
		}
		if noindex {
			issues = append(issues, makeIssue("NOINDEX", in.URL, ""))
		}
		if wordCount < thinWords {
			issues = append(issues, makeIssue("THIN_CONTENT", in.URL, ""))
		}
		if !hasJsonLd {
			issues = append(issues, makeIssue("NO_STRUCTURED_DATA", in.URL, ""))
		}
		if mixed {
			issues = append(issues, makeIssue("MIXED_CONTENT", in.URL, ""))
		}
	}
	if in.LoadMs != nil && *in.LoadMs > slowMs {
		issues = append(issues, makeIssue("SLOW_RESPONSE", in.URL, ""))
	}
	if in.RedirectHops > 1 {
		issues = append(issues, makeIssue("REDIRECT_CHAIN", in.URL, ""))
	}

	page := Page{URL: in.URL, StatusCode: in.StatusCode, MetaDescription: metaDescription, H1Count: h1Count, WordCount: wordCount, Canonical: canonical, Indexable: indexable, ImagesTotal: imagesTotal, ImagesMissingAlt: imagesMissingAlt, InternalLinks: len(internal), ExternalLinks: len(external), HasJsonLd: hasJsonLd, HasHreflang: hasHreflang}
	if title != "" {
		page.Title = &title
	}
	if in.LoadMs != nil {
		ms := int(math.Round(*in.LoadMs))
		page.LoadMs = &ms
	}
	return &AnalyzeOutput{Page: page, Issues: issues, InternalURLs: internalOrder, ExternalURLs: externalOrder}, nil
}

func sameOrigin(a, b *url.URL) bool {
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Host, b.Host)
}
