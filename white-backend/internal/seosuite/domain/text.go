package domain

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

var (
	spacesRe    = regexp.MustCompile(`\s+`)
	lineSplitRe = regexp.MustCompile(`[\n,]+`)
)

// NormalizeKeyword trims, collapses whitespace and lowercases.
func NormalizeKeyword(v string) string {
	return strings.ToLower(spacesRe.ReplaceAllString(strings.TrimSpace(v), " "))
}

// ParseKeywordLines turns a textarea (newline or comma separated) into a
// unique list capped at max; entries longer than 120 characters are dropped.
func ParseKeywordLines(text string, max int) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, raw := range lineSplitRe.Split(text, -1) {
		k := NormalizeKeyword(raw)
		if k == "" || utf8.RuneCountInString(k) > 120 || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, k)
		if len(out) >= max {
			break
		}
	}
	return out
}

// ParseTags: "Brand, produk ,, brand" -> ["brand", "produk"], max 10 of 30 chars.
func ParseTags(text string, max int) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, raw := range lineSplitRe.Split(text, -1) {
		tag := NormalizeKeyword(raw)
		if tag == "" || utf8.RuneCountInString(tag) > 30 || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
		if len(out) >= max {
			break
		}
	}
	return out
}

type LabelCount struct {
	Label string
	Count int32
}

// TopCounts returns the n most frequent values; blanks count as "(kosong)".
func TopCounts(values []string, n int) []LabelCount {
	acc := map[string]int32{}
	order := []string{}
	for _, v := range values {
		k := strings.TrimSpace(v)
		if k == "" {
			k = "(kosong)"
		}
		if _, seen := acc[k]; !seen {
			order = append(order, k)
		}
		acc[k]++
	}
	sort.SliceStable(order, func(i, j int) bool { return acc[order[i]] > acc[order[j]] })
	if len(order) > n {
		order = order[:n]
	}
	out := make([]LabelCount, 0, len(order))
	for _, k := range order {
		out = append(out, LabelCount{Label: k, Count: acc[k]})
	}
	return out
}

// PickSeriesDomains: own domain first, then the top n others by latest value.
func PickSeriesDomains(own string, latestByDomain map[string]int32, n int) []string {
	others := []string{}
	for d := range latestByDomain {
		if d != own {
			others = append(others, d)
		}
	}
	sort.SliceStable(others, func(i, j int) bool {
		if latestByDomain[others[i]] != latestByDomain[others[j]] {
			return latestByDomain[others[i]] > latestByDomain[others[j]]
		}
		return others[i] < others[j]
	})
	if len(others) > n {
		others = others[:n]
	}
	return append([]string{own}, others...)
}
