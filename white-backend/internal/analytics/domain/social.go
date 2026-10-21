package domain

import (
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/metrics"
)

// PostEngagements = likes + comments + shares + saves.
func PostEngagements(p SocialPost) int { return p.Likes + p.Comments + p.Shares + p.Saves }

// PostEngagementRate is ER by followers in percent.
func PostEngagementRate(p SocialPost, followers int) float64 {
	return metrics.SafeDiv(float64(PostEngagements(p)), float64(followers)) * 100
}

// AvgEngagementRate is the mean of per-post ER; 0 without posts or followers.
func AvgEngagementRate(posts []SocialPost, followers int) float64 {
	if len(posts) == 0 || followers <= 0 {
		return 0
	}
	xs := make([]float64, 0, len(posts))
	for _, p := range posts {
		xs = append(xs, PostEngagementRate(p, followers))
	}
	return metrics.Mean(xs)
}

type Growth struct {
	Abs int
	Pct *float64
}

func FollowerGrowth(start, end int) Growth {
	g := Growth{Abs: end - start}
	if start != 0 {
		pct := float64(g.Abs) / float64(start) * 100
		g.Pct = &pct
	}
	return g
}

func PostsPerWeek(posts, days int) float64 {
	if days <= 0 {
		return 0
	}
	return float64(posts) / float64(days) * 7
}

// ContentType labels a post: Reels, Story, Video, Carousel or Foto.
func ContentType(mediaType string, productType *string) string {
	pt := ""
	if productType != nil {
		pt = strings.ToUpper(*productType)
	}
	mt := strings.ToUpper(mediaType)
	switch {
	case pt == "REELS":
		return "Reels"
	case pt == "STORY":
		return "Story"
	case mt == "VIDEO":
		return "Video"
	case mt == "CAROUSEL_ALBUM":
		return "Carousel"
	}
	return "Foto"
}

type HeatCell struct {
	Day           int
	Hour          int
	Posts         int
	AvgEngagement float64
}

var jakarta = mustLocation("Asia/Jakarta")

func mustLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.FixedZone("WIB", 7*3600)
	}
	return loc
}

// PostingHeatmap is 7x24 cells (Monday = 0) of average engagements per
// publish slot in Asia/Jakarta.
func PostingHeatmap(posts []SocialPost) []HeatCell {
	type acc struct {
		n int
		e int
	}
	sums := map[[2]int]*acc{}
	for _, p := range posts {
		t := p.PublishedAt.In(jakarta)
		day := (int(t.Weekday()) + 6) % 7
		key := [2]int{day, t.Hour()}
		a := sums[key]
		if a == nil {
			a = &acc{}
			sums[key] = a
		}
		a.n++
		a.e += PostEngagements(p)
	}
	out := make([]HeatCell, 0, 7*24)
	for d := 0; d < 7; d++ {
		for h := 0; h < 24; h++ {
			c := HeatCell{Day: d, Hour: h}
			if a := sums[[2]int{d, h}]; a != nil {
				c.Posts = a.n
				c.AvgEngagement = float64(a.e) / float64(a.n)
			}
			out = append(out, c)
		}
	}
	return out
}

// TopPosts sorts by engagements descending (stable) and keeps n.
func TopPosts(posts []SocialPost, n int) []SocialPost {
	out := append([]SocialPost(nil), posts...)
	sort.SliceStable(out, func(i, j int) bool { return PostEngagements(out[i]) > PostEngagements(out[j]) })
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// FollowersAt is the last snapshot on or before the day (snapshots sorted
// ascending); 0 before the first snapshot.
func FollowersAt(snaps []SocialSnapshot, day time.Time) int {
	f := 0
	for _, s := range snaps {
		if s.Date.After(day) {
			break
		}
		f = s.Followers
	}
	return f
}

// FollowersAtAcrossAccounts sums, per account, the latest snapshot on or
// before the day. Accounts without a snapshot yet contribute nothing.
func FollowersAtAcrossAccounts(snaps []SocialSnapshot, day time.Time) int {
	last := map[string]int{}
	for _, s := range snaps {
		if !s.Date.After(day) {
			last[s.AccountID] = s.Followers
		}
	}
	sum := 0
	for _, v := range last {
		sum += v
	}
	return sum
}

type DailySocialPoint struct {
	Date        string
	Followers   *int
	Reach       int
	Impressions int
}

// DailyFollowerSeries forward-fills followers from the last snapshot before
// the range (nil when there is none); reach and impressions are 0 when missing.
func DailyFollowerSeries(r dates.Range, snaps []SocialSnapshot) []DailySocialPoint {
	byDay := map[string]SocialSnapshot{}
	var carry *int
	for _, s := range snaps {
		if s.Date.Before(r.From) {
			v := s.Followers
			carry = &v
		} else {
			byDay[dates.Format(s.Date)] = s
		}
	}
	out := make([]DailySocialPoint, 0, r.Days())
	r.Each(func(d time.Time) {
		k := dates.Format(d)
		p := DailySocialPoint{Date: k}
		if s, ok := byDay[k]; ok {
			v := s.Followers
			carry = &v
			p.Followers = &v
			if s.Reach != nil {
				p.Reach = *s.Reach
			}
			if s.Impressions != nil {
				p.Impressions = *s.Impressions
			}
		} else {
			p.Followers = carry
		}
		out = append(out, p)
	})
	return out
}

// IndexTo100 bases the series on the first value > 0; nil or <= 0 stays nil.
func IndexTo100(values []*int) []*float64 {
	out := make([]*float64, len(values))
	var base *int
	for _, v := range values {
		if v != nil && *v > 0 {
			base = v
			break
		}
	}
	if base == nil {
		return out
	}
	for i, v := range values {
		if v == nil || *v <= 0 {
			continue
		}
		x := float64(*v) / float64(*base) * 100
		out[i] = &x
	}
	return out
}

type ErByType struct {
	Type  string
	AvgEr float64
	Count int
}

// ErByContentType averages ER per content type, sorted by ER descending.
func ErByContentType(posts []SocialPost, followers int) []ErByType {
	groups := map[string][]SocialPost{}
	order := []string{}
	for _, p := range posts {
		k := ContentType(p.MediaType, p.ProductType)
		if _, ok := groups[k]; !ok {
			order = append(order, k)
		}
		groups[k] = append(groups[k], p)
	}
	out := make([]ErByType, 0, len(order))
	for _, k := range order {
		out = append(out, ErByType{Type: k, AvgEr: AvgEngagementRate(groups[k], followers), Count: len(groups[k])})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].AvgEr > out[j].AvgEr })
	return out
}

var igUsernameRe = regexp.MustCompile(`^[a-z0-9._]+$`)

// NormalizeIgUsername trims, strips leading "@" and lowercases. Valid: 2 to
// 30 chars of [a-z0-9._], not starting or ending with ".". Empty when invalid.
func NormalizeIgUsername(raw string) string {
	u := strings.ToLower(strings.TrimLeft(strings.TrimSpace(raw), "@"))
	n := utf8.RuneCountInString(u)
	if n < 2 || n > 30 || !igUsernameRe.MatchString(u) || strings.HasPrefix(u, ".") || strings.HasSuffix(u, ".") {
		return ""
	}
	return u
}

// Truncate mirrors lib/format.ts truncate: cut at max-1 runes plus an ellipsis.
func Truncate(text string, max int) string {
	if utf8.RuneCountInString(text) <= max {
		return text
	}
	runes := []rune(text)
	return strings.TrimRight(string(runes[:max-1]), " \t\n") + "…"
}
