package crawler

import (
	"encoding/xml"
	"regexp"
	"strings"
)

type Rules struct {
	Disallow []string
	Allow    []string
	Sitemaps []string
}

var AllowAll = Rules{}

// ParseRobots reads robots.txt for one user-agent token (case-insensitive
// substring match either way). Groups naming the specific UA win over `*`.
func ParseRobots(txt, uaToken string) Rules {
	type group struct {
		agents   []string
		disallow []string
		allow    []string
	}
	var groups []*group
	var sitemaps []string
	var current *group
	lastWasAgent := false
	for _, raw := range strings.Split(txt, "\n") {
		line := raw
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		field := strings.ToLower(strings.TrimSpace(line[:idx]))
		value := strings.TrimSpace(line[idx+1:])
		switch field {
		case "sitemap":
			if value != "" {
				sitemaps = append(sitemaps, value)
			}
			continue
		case "user-agent":
			if current == nil || !lastWasAgent {
				current = &group{}
				groups = append(groups, current)
			}
			current.agents = append(current.agents, strings.ToLower(value))
			lastWasAgent = true
			continue
		}
		lastWasAgent = false
		if current == nil {
			continue
		}
		if field == "disallow" && value != "" {
			current.disallow = append(current.disallow, value)
		}
		if field == "allow" && value != "" {
			current.allow = append(current.allow, value)
		}
	}
	token := strings.ToLower(uaToken)
	var chosen []*group
	for _, g := range groups {
		for _, a := range g.agents {
			if a != "*" && (strings.Contains(token, a) || strings.Contains(a, token)) {
				chosen = append(chosen, g)
				break
			}
		}
	}
	if len(chosen) == 0 {
		for _, g := range groups {
			for _, a := range g.agents {
				if a == "*" {
					chosen = append(chosen, g)
					break
				}
			}
		}
	}
	r := Rules{Sitemaps: sitemaps}
	for _, g := range chosen {
		r.Disallow = append(r.Disallow, g.disallow...)
		r.Allow = append(r.Allow, g.allow...)
	}
	return r
}

// matchLength is the pattern length when it matches the path, else -1.
// Supports `*` and a trailing `$`.
func matchLength(pattern, path string) int {
	p := pattern
	anchored := false
	if strings.HasSuffix(p, "$") {
		anchored = true
		p = p[:len(p)-1]
	}
	parts := strings.Split(p, "*")
	for i, part := range parts {
		parts[i] = regexp.QuoteMeta(part)
	}
	expr := "^" + strings.Join(parts, ".*")
	if anchored {
		expr += "$"
	}
	re, err := regexp.Compile(expr)
	if err != nil || !re.MatchString(path) {
		return -1
	}
	return len(p)
}

// IsAllowed applies longest-match; Allow wins ties.
func IsAllowed(rules Rules, path string) bool {
	best := -1
	for _, d := range rules.Disallow {
		if n := matchLength(d, path); n > best {
			best = n
		}
	}
	if best < 0 {
		return true
	}
	bestAllow := -1
	for _, a := range rules.Allow {
		if n := matchLength(a, path); n > bestAllow {
			bestAllow = n
		}
	}
	return bestAllow >= best
}

// ParseSitemap extracts page URLs (urlset) and child sitemaps (sitemapindex).
func ParseSitemap(data string) (urls []string, sitemaps []string) {
	var doc struct {
		XMLName xml.Name
		URLs    []struct {
			Loc string `xml:"loc"`
		} `xml:"url"`
		Sitemaps []struct {
			Loc string `xml:"loc"`
		} `xml:"sitemap"`
	}
	if err := xml.Unmarshal([]byte(data), &doc); err != nil {
		return nil, nil
	}
	if doc.XMLName.Local == "urlset" {
		for _, u := range doc.URLs {
			if v := strings.TrimSpace(u.Loc); v != "" {
				urls = append(urls, v)
			}
		}
	}
	if doc.XMLName.Local == "sitemapindex" {
		for _, s := range doc.Sitemaps {
			if v := strings.TrimSpace(s.Loc); v != "" {
				sitemaps = append(sitemaps, v)
			}
		}
	}
	return urls, sitemaps
}
