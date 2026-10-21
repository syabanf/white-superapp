package domain

import (
	"math"
	"sort"
	"time"
)

// Posting-time metrics ported from lib/metrics/publishing.ts and
// postingHeatmap in lib/metrics/social.ts.

type HeatCell struct {
	Day           int
	Hour          int
	Posts         int
	AvgEngagement float64
}

type BestTime struct {
	Day   int
	Hour  int
	Score float64
}

// DefaultBestTimes: Mon 11:00, Wed 19:00, Fri 12:00.
var DefaultBestTimes = []BestTime{{Day: 0, Hour: 11}, {Day: 2, Hour: 19}, {Day: 4, Hour: 12}}

// EngagedPost is a synced social post reduced to what the heatmap needs.
type EngagedPost struct {
	PublishedAt time.Time
	Engagements float64
}

// PostingHeatmap buckets posts into 7 x 24 cells by client-tz weekday and hour.
func PostingHeatmap(posts []EngagedPost, timeZone string) []HeatCell {
	type agg struct {
		n int
		e float64
	}
	sums := map[[2]int]*agg{}
	for _, p := range posts {
		z := ToZoned(p.PublishedAt, timeZone)
		key := [2]int{z.Weekday, z.Hour}
		a := sums[key]
		if a == nil {
			a = &agg{}
			sums[key] = a
		}
		a.n++
		a.e += p.Engagements
	}
	out := make([]HeatCell, 0, 7*24)
	for d := 0; d < 7; d++ {
		for h := 0; h < 24; h++ {
			c := HeatCell{Day: d, Hour: h}
			if a := sums[[2]int{d, h}]; a != nil {
				c.Posts = a.n
				c.AvgEngagement = a.e / float64(a.n)
			}
			out = append(out, c)
		}
	}
	return out
}

// BestPostingTimes ranks cells by avgEngagement * ln(1 + posts), ignores
// cells under minPosts, prefers distinct weekdays, then tops up with
// same-day slots and finally the defaults. n = 3 and minPosts = 2 are the
// production values.
func BestPostingTimes(cells []HeatCell, n, minPosts int) []BestTime {
	var ranked []BestTime
	for _, c := range cells {
		if c.Posts >= minPosts && c.AvgEngagement > 0 {
			ranked = append(ranked, BestTime{Day: c.Day, Hour: c.Hour, Score: c.AvgEngagement * math.Log(1+float64(c.Posts))})
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		a, b := ranked[i], ranked[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.Day != b.Day {
			return a.Day < b.Day
		}
		return a.Hour < b.Hour
	})
	out := make([]BestTime, 0, n)
	usedDays := map[int]bool{}
	has := func(day, hour int) bool {
		for _, o := range out {
			if o.Day == day && o.Hour == hour {
				return true
			}
		}
		return false
	}
	for _, r := range ranked {
		if len(out) >= n {
			break
		}
		if usedDays[r.Day] {
			continue
		}
		usedDays[r.Day] = true
		out = append(out, r)
	}
	for _, r := range ranked {
		if len(out) >= n {
			break
		}
		if !has(r.Day, r.Hour) {
			out = append(out, r)
		}
	}
	for _, d := range DefaultBestTimes {
		if len(out) >= n {
			break
		}
		if !has(d.Day, d.Hour) {
			out = append(out, d)
		}
	}
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// PostStatusCounts counts posts per status with every status present.
func PostStatusCounts(statuses []PostStatus) map[PostStatus]int32 {
	out := make(map[PostStatus]int32, len(StatusOrder))
	for _, s := range StatusOrder {
		out[s] = 0
	}
	for _, s := range statuses {
		if _, ok := out[s]; ok {
			out[s]++
		}
	}
	return out
}

// FailureRate is FAILED / (PUBLISHED + FAILED) * 100, nil when nothing finished.
func FailureRate(statuses []PostStatus) *float64 {
	done, failed := 0, 0
	for _, s := range statuses {
		if s == StatusPublished || s == StatusFailed {
			done++
			if s == StatusFailed {
				failed++
			}
		}
	}
	if done == 0 {
		return nil
	}
	v := float64(failed) / float64(done) * 100
	return &v
}

// PublishingSummary is the report block built from published targets.
type PublishingSummary struct {
	Posts       int32
	PerPlatform []PlatformCount
	Failed      int32
	// "engagement" when any target matched a synced post, else "recent".
	Mode string
	Top  []PublishedTarget
}

type PlatformCount struct {
	Platform Platform
	Count    int32
}

// SummarizePublishing mirrors reports/extras.ts summarizePublishing. Targets
// carry Engagements when a synced post matched them.
func SummarizePublishing(published []PublishedTarget, failed int32, n int) PublishingSummary {
	counts := map[Platform]int32{}
	posts := map[string]bool{}
	var matched, all []PublishedTarget
	for _, t := range published {
		counts[t.Platform]++
		posts[t.PostID] = true
		all = append(all, t)
		if t.Engagements != nil {
			matched = append(matched, t)
		}
	}
	per := make([]PlatformCount, 0, len(counts))
	for p, c := range counts {
		per = append(per, PlatformCount{Platform: p, Count: c})
	}
	sort.Slice(per, func(i, j int) bool {
		if per[i].Count != per[j].Count {
			return per[i].Count > per[j].Count
		}
		return per[i].Platform < per[j].Platform
	})
	byRecency := func(a, b PublishedTarget) bool { return a.PublishedAt.After(b.PublishedAt) }
	mode := "recent"
	top := all
	if len(matched) > 0 {
		mode = "engagement"
		top = matched
		sort.SliceStable(top, func(i, j int) bool {
			if *top[i].Engagements != *top[j].Engagements {
				return *top[i].Engagements > *top[j].Engagements
			}
			return byRecency(top[i], top[j])
		})
	} else {
		sort.SliceStable(top, func(i, j int) bool { return byRecency(top[i], top[j]) })
	}
	if len(top) > n {
		top = top[:n]
	}
	return PublishingSummary{Posts: int32(len(posts)), PerPlatform: per, Failed: failed, Mode: mode, Top: top}
}
