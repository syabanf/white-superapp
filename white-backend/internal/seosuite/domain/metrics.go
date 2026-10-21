package domain

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/metrics"
)

// Formulas ported from lib/metrics/seo-suite.ts and lib/metrics/seo.ts.

var expectedCtrByPosition = map[int]float64{1: 28, 2: 15, 3: 11, 4: 8, 5: 7, 6: 5, 7: 4, 8: 3, 9: 3, 10: 2.5}

// ExpectedCtr is the benchmark organic CTR curve in percent.
func ExpectedCtr(position float64) float64 {
	p := int(math.Max(1, math.Round(position)))
	if p <= 10 {
		return expectedCtrByPosition[p]
	}
	if p <= 20 {
		return 1.5
	}
	return 1
}

func ranked(p *int32) bool { return p != nil && *p <= 100 }

// Visibility is the mean of ExpectedCtr over positions; unranked counts 0.
func Visibility(positions []*int32) float64 {
	if len(positions) == 0 {
		return 0
	}
	var sum float64
	for _, p := range positions {
		if ranked(p) {
			sum += ExpectedCtr(float64(*p))
		}
	}
	return sum / float64(len(positions))
}

// AvgPosition averages ranked positions only; nil when none rank.
func AvgPosition(positions []*int32) *float64 {
	var sum float64
	var n int
	for _, p := range positions {
		if ranked(p) {
			sum += float64(*p)
			n++
		}
	}
	if n == 0 {
		return nil
	}
	v := sum / float64(n)
	return &v
}

func CountTop(positions []*int32, n int32) int32 {
	var c int32
	for _, p := range positions {
		if p != nil && *p <= n {
			c++
		}
	}
	return c
}

var RankBuckets = []string{"1-3", "4-10", "11-20", "21-50", "51-100", "unranked"}

func RankBucket(position *int32) string {
	switch {
	case position == nil || *position > 100 || *position < 1:
		return "unranked"
	case *position <= 3:
		return "1-3"
	case *position <= 10:
		return "4-10"
	case *position <= 20:
		return "11-20"
	case *position <= 50:
		return "21-50"
	}
	return "51-100"
}

type BucketCount struct {
	Bucket string
	Count  int32
}

func RankDistribution(positions []*int32) []BucketCount {
	acc := map[string]int32{}
	for _, p := range positions {
		acc[RankBucket(p)]++
	}
	out := make([]BucketCount, 0, len(RankBuckets))
	for _, b := range RankBuckets {
		out = append(out, BucketCount{Bucket: b, Count: acc[b]})
	}
	return out
}

type RankChangeKind string

const (
	ChangeUp   RankChangeKind = "up"
	ChangeDown RankChangeKind = "down"
	ChangeFlat RankChangeKind = "flat"
	ChangeNew  RankChangeKind = "new"
	ChangeLost RankChangeKind = "lost"
	ChangeNone RankChangeKind = "none"
)

type RankChange struct {
	Kind RankChangeKind
	// Delta is prev - cur, so positive means improvement.
	Delta *int32
}

// ClassifyRankChange compares two positions; anything above 100 counts as unranked.
func ClassifyRankChange(prev, cur *int32) RankChange {
	p, c := prev, cur
	if !ranked(p) {
		p = nil
	}
	if !ranked(c) {
		c = nil
	}
	switch {
	case p == nil && c == nil:
		return RankChange{Kind: ChangeNone}
	case p == nil:
		return RankChange{Kind: ChangeNew}
	case c == nil:
		return RankChange{Kind: ChangeLost}
	}
	d := *p - *c
	kind := ChangeFlat
	if d > 0 {
		kind = ChangeUp
	} else if d < 0 {
		kind = ChangeDown
	}
	return RankChange{Kind: kind, Delta: &d}
}

// IsRankDropAlert: left the top 100, dropped more than 5 places, or fell out of the top 10.
func IsRankDropAlert(prev, cur *int32) bool {
	ch := ClassifyRankChange(prev, cur)
	if ch.Kind == ChangeLost {
		return true
	}
	if ch.Kind != ChangeDown {
		return false
	}
	p, c := *prev, *cur
	return c-p > 5 || (p <= 10 && c > 10)
}

// PositionToScore is the sparkline transform: 101 - clamp(p, 1, 100), 0 when unranked.
func PositionToScore(position *int32) int32 {
	if !ranked(position) {
		return 0
	}
	return 101 - int32(metrics.Clamp(float64(*position), 1, 100))
}

type ShareOfVoiceRow struct {
	Domain string
	Weight float64
	Share  float64
}

// ShareOfVoice weights each top-10 appearance by ExpectedCtr and folds the
// tail after n domains into otherLabel.
func ShareOfVoice(lists [][]TopResult, n int, otherLabel string) []ShareOfVoiceRow {
	acc := map[string]float64{}
	order := []string{}
	for _, list := range lists {
		for _, r := range list {
			if r.Domain == "" {
				continue
			}
			if _, seen := acc[r.Domain]; !seen {
				order = append(order, r.Domain)
			}
			acc[r.Domain] += ExpectedCtr(float64(r.Position))
		}
	}
	var total float64
	for _, w := range acc {
		total += w
	}
	sort.SliceStable(order, func(i, j int) bool { return acc[order[i]] > acc[order[j]] })
	share := func(w float64) float64 {
		if total == 0 {
			return 0
		}
		return w / total * 100
	}
	rows := []ShareOfVoiceRow{}
	var rest float64
	for i, d := range order {
		if i < n {
			rows = append(rows, ShareOfVoiceRow{Domain: d, Weight: acc[d], Share: share(acc[d])})
		} else {
			rest += acc[d]
		}
	}
	if rest > 0 {
		rows = append(rows, ShareOfVoiceRow{Domain: otherLabel, Weight: rest, Share: share(rest)})
	}
	return rows
}

type DifficultyBand string

const (
	BandEasy     DifficultyBand = "easy"
	BandMedium   DifficultyBand = "medium"
	BandHard     DifficultyBand = "hard"
	BandVeryHard DifficultyBand = "very_hard"
)

func DifficultyBandOf(kd *float64) *DifficultyBand {
	if kd == nil || math.IsNaN(*kd) {
		return nil
	}
	v := metrics.Clamp(*kd, 0, 100)
	b := BandVeryHard
	switch {
	case v < 30:
		b = BandEasy
	case v < 60:
		b = BandMedium
	case v < 80:
		b = BandHard
	}
	return &b
}

// IsCacheFresh: 0 <= age < maxAgeDays. A fetchedAt in the future forces a refetch.
func IsCacheFresh(fetchedAt, now time.Time, maxAgeDays int) bool {
	age := now.Sub(fetchedAt)
	return age >= 0 && age < time.Duration(maxAgeDays)*24*time.Hour
}

type GapRow struct {
	Keyword            string
	Volume             int32
	OwnPosition        *int32
	CompetitorPosition *int32
}

type KeywordGap struct {
	Missing []GapRow
	Weak    []GapRow
	Strong  []GapRow
	Shared  []GapRow
}

// KeywordGapOf splits a competitor's keywords against our own ranked set.
func KeywordGapOf(own, competitor []DomainKeyword) KeywordGap {
	mine := map[string]DomainKeyword{}
	for _, k := range own {
		mine[strings.ToLower(k.Keyword)] = k
	}
	var g KeywordGap
	g.Missing, g.Weak, g.Strong, g.Shared = []GapRow{}, []GapRow{}, []GapRow{}, []GapRow{}
	for _, c := range competitor {
		cp := c.Position
		m, ok := mine[strings.ToLower(c.Keyword)]
		if !ok {
			g.Missing = append(g.Missing, GapRow{Keyword: c.Keyword, Volume: c.Volume, CompetitorPosition: &cp})
			continue
		}
		op := m.Position
		row := GapRow{Keyword: c.Keyword, Volume: max(c.Volume, m.Volume), OwnPosition: &op, CompetitorPosition: &cp}
		g.Shared = append(g.Shared, row)
		if m.Position > c.Position {
			g.Weak = append(g.Weak, row)
		} else {
			g.Strong = append(g.Strong, row)
		}
	}
	byVolume := func(rows []GapRow) {
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].Volume > rows[j].Volume })
	}
	byVolume(g.Missing)
	byVolume(g.Weak)
	byVolume(g.Strong)
	byVolume(g.Shared)
	return g
}

func IsToxic(spamScore *int32) bool { return spamScore != nil && *spamScore >= ToxicSpamScore }

// IsNewBacklink: live and first seen within newDays of now.
func IsNewBacklink(firstSeen time.Time, isLost bool, now time.Time, newDays int) bool {
	return !isLost && now.Sub(firstSeen) <= time.Duration(newDays)*24*time.Hour
}

func DofollowShare(dofollow, nofollow int32) float64 {
	total := dofollow + nofollow
	if total == 0 {
		return 0
	}
	return float64(dofollow) / float64(total) * 100
}

var (
	schemeRe    = regexp.MustCompile(`^[a-z][a-z0-9+.-]*://`)
	credsRe     = regexp.MustCompile(`^[^/@]*@`)
	portRe      = regexp.MustCompile(`:\d+$`)
	hostnameRe  = regexp.MustCompile(`^([a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$`)
	splitPathRe = regexp.MustCompile(`[/?#]`)
)

// NormalizeDomain reduces user input to a bare lowercase host; "" when it is
// not a plausible hostname.
func NormalizeDomain(input string) string {
	s := strings.ToLower(strings.TrimSpace(input))
	if s == "" {
		return ""
	}
	s = strings.TrimPrefix(s, "sc-domain:")
	s = schemeRe.ReplaceAllString(s, "")
	s = credsRe.ReplaceAllString(s, "")
	s = splitPathRe.Split(s, 2)[0]
	s = portRe.ReplaceAllString(s, "")
	s = strings.TrimPrefix(s, "www.")
	s = strings.TrimRight(s, ".")
	if len(s) > 253 || !hostnameRe.MatchString(s) {
		return ""
	}
	return s
}

// SiteHost is the own host of a GSC property (https://x.id/ or sc-domain:x.id).
func SiteHost(siteURL string) string {
	if h := NormalizeDomain(siteURL); h != "" {
		return h
	}
	return strings.TrimPrefix(siteURL, "sc-domain:")
}
