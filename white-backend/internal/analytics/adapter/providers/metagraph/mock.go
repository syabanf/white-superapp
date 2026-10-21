package metagraph

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/providers/mockrand"
)

// Mock is the deterministic Meta Graph used without app credentials. Seeds
// match the TS mock (FNV-1a + mulberry32) so demo data stays identical.
type Mock struct {
	Now func() time.Time
}

func (m Mock) now() time.Time {
	if m.Now != nil {
		return m.Now()
	}
	return time.Now()
}

func jsRound(x float64) float64 { return math.Floor(x + 0.5) }

func round(x float64) int { return int(jsRound(x)) }

func base36(n uint32) string { return strconv.FormatUint(uint64(n), 36) }

var captions = []string{
	"Koleksi terbaru sudah tersedia! Cek link di bio ✨",
	"Terima kasih untuk semua pelanggan setia bulan ini 🙏",
	"Promo spesial akhir pekan — jangan sampai kehabisan!",
	"Behind the scenes dari tim kami hari ini 📸",
	"Tips singkat untuk kamu yang baru mulai. Simpan dulu!",
	"Giveaway time! Tag 3 temanmu di kolom komentar 🎁",
	"Produk favorit pelanggan minggu ini — sudah coba?",
	"Cerita di balik layar. Geser untuk lihat prosesnya →",
}

const hashtags = "#promo #indonesia #lokal #umkm"

func titleCase(username string) string {
	parts := strings.FieldsFunc(username, func(r rune) bool { return r == '.' || r == '_' })
	for i, w := range parts {
		parts[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	if out := strings.Join(parts, " "); out != "" {
		return out
	}
	return username
}

func baseFollowers(seedKey string, min, span uint32) int {
	return int(min + mockrand.Seed(seedKey)%span)
}

// BuildFollowerSeries is a plausible daily follower series of `days` values
// ending exactly at endFollowers. It seeds competitor history.
func BuildFollowerSeries(seedKey string, days, endFollowers int) []int {
	rand := mockrand.New("series:" + seedKey)
	gain := math.Max(1, jsRound(float64(endFollowers)*(0.0004+rand.Float64()*0.0008)))
	f := math.Max(10, float64(endFollowers)-gain*float64(days))
	out := make([]float64, 0, days)
	for i := 0; i < days; i++ {
		spike := 1.0
		if rand.Float64() < 0.02 {
			spike = 4 + rand.Float64()*3
		}
		f = math.Max(0, jsRound(f+gain*(0.7+rand.Float64()*0.6)*spike+(rand.Float64()-0.5)*gain))
		out = append(out, f)
	}
	diff := float64(endFollowers) - out[len(out)-1]
	res := make([]int, 0, days)
	for i, v := range out {
		res = append(res, int(math.Max(0, v+jsRound(diff*float64(i+1)/float64(days)))))
	}
	return res
}

func (m Mock) mockIgMedia(seedKey string, count, followers int, withInsights bool) []IgMedia {
	rand := mockrand.New("media:" + seedKey)
	end := dates.Day(m.now())
	seed := mockrand.Seed(seedKey)
	out := make([]IgMedia, 0, count)
	for i := 0; i < count; i++ {
		daysAgo := int(math.Min(115, jsRound(float64(i)*(90/math.Max(1, float64(count)))+rand.Float64()*3)))
		hourWib := 8 + int(math.Floor(rand.Float64()*13))
		ts := dates.AddDays(end, -daysAgo).Add(time.Duration(hourWib-7)*time.Hour + time.Duration(math.Floor(rand.Float64()*60))*time.Minute)
		roll := rand.Float64()
		mediaType := "IMAGE"
		er := 0.018
		switch {
		case roll < 0.42:
			mediaType, er = "VIDEO", 0.03
		case roll < 0.72:
			mediaType, er = "CAROUSEL_ALBUM", 0.024
		}
		er *= 0.5 + rand.Float64()
		engagements := int(math.Max(8, jsRound(float64(followers)*er)))
		likeCount := round(float64(engagements) * (0.82 + rand.Float64()*0.08))
		commentsCount := int(math.Max(1, jsRound(float64(engagements)*(0.05+rand.Float64()*0.04))))
		itemSeed := mockrand.Seed(fmt.Sprintf("%s:%d", seedKey, i))
		product := "FEED"
		if mediaType == "VIDEO" {
			product = "REELS"
		}
		permalink := fmt.Sprintf("https://www.instagram.com/p/mock%s/", base36(itemSeed))
		mediaURL := fmt.Sprintf("https://picsum.photos/seed/mg-%d-%d/800/800", seed%9973, i)
		thumbURL := fmt.Sprintf("https://picsum.photos/seed/mg-%d-%d/400/400", seed%9973, i)
		media := IgMedia{
			ID:               "mock_" + base36(itemSeed),
			Caption:          captions[int((seed+uint32(i))%uint32(len(captions)))] + " " + hashtags,
			MediaType:        mediaType,
			MediaProductType: &product,
			Permalink:        &permalink,
			MediaURL:         &mediaURL,
			ThumbnailURL:     &thumbURL,
			Timestamp:        ts,
			LikeCount:        likeCount,
			CommentsCount:    commentsCount,
		}
		if withInsights {
			shares := round(float64(engagements) * (0.04 + rand.Float64()*0.05))
			saved := int(math.Max(0, float64(engagements-likeCount-commentsCount-shares)))
			share := 0.22
			if mediaType == "VIDEO" {
				share = 0.4
			}
			reach := round(float64(followers) * share * (0.6 + rand.Float64()*0.9))
			impressions := round(float64(reach) * (1.3 + rand.Float64()*0.5))
			views := 0
			if mediaType == "VIDEO" {
				views = round(float64(reach) * (1.4 + rand.Float64()))
			}
			media.Shares, media.Saved, media.Reach, media.Impressions, media.Views = &shares, &saved, &reach, &impressions, &views
		}
		out = append(out, media)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Timestamp.After(out[j].Timestamp) })
	return out
}

func dicebear(seed, bg, fg string) *string {
	s := fmt.Sprintf("https://api.dicebear.com/9.x/initials/svg?seed=%s&backgroundColor=%s&textColor=%s", url.QueryEscape(seed), bg, fg)
	return &s
}

func (m Mock) GetInstagramAccount(_ context.Context, _, igUserID string) (*IgAccount, error) {
	key := "ig:" + igUserID
	username := "akun_" + base36(mockrand.Seed(key)%46655)
	bio := "Akun demo Instagram Business."
	return &IgAccount{
		ID: igUserID, Username: username, Name: titleCase(username), Biography: &bio,
		ProfilePictureURL: dicebear(username, "1c1c1e", "ffffff"),
		FollowersCount:    baseFollowers(key, 12000, 180000),
		FollowsCount:      180 + int(mockrand.Seed(key)%240),
		MediaCount:        150 + int(mockrand.Seed(key)%900),
	}, nil
}

func (m Mock) GetInstagramInsights(_ context.Context, _, igUserID string, since, until time.Time) ([]IgDailyInsight, error) {
	followers := float64(baseFollowers("ig:"+igUserID, 12000, 180000))
	out := []IgDailyInsight{}
	dates.Range{From: dates.Day(since), To: dates.Day(until)}.Each(func(d time.Time) {
		r := mockrand.New("ins:" + igUserID + ":" + dates.Format(d))
		reach := jsRound(followers * (0.08 + r.Float64()*0.06))
		out = append(out, IgDailyInsight{
			Date: dates.Format(d), Reach: int(reach),
			Impressions:   round(reach * (1.4 + r.Float64()*0.5)),
			ProfileViews:  round(reach * (0.05 + r.Float64()*0.04)),
			FollowerCount: round(followers * 0.0008 * (0.4 + r.Float64())),
		})
	})
	return out, nil
}

func (m Mock) GetInstagramMedia(_ context.Context, _, igUserID string, limit int) ([]IgMedia, error) {
	if limit <= 0 {
		limit = 30
	}
	return m.mockIgMedia("ig:"+igUserID, limit, baseFollowers("ig:"+igUserID, 12000, 180000), true), nil
}

func (m Mock) DiscoverBusiness(_ context.Context, _, _, username string) (*IgBusinessDiscovery, error) {
	u := strings.ToLower(strings.TrimLeft(strings.TrimSpace(username), "@"))
	key := "bd:" + u
	followers := baseFollowers(key, 8000, 290000)
	name := titleCase(u)
	bio := fmt.Sprintf("Akun resmi %s. Demo Business Discovery.", name)
	return &IgBusinessDiscovery{
		Username: u, Name: name, Biography: &bio, ProfilePictureURL: dicebear(name, "8e8e93", "ffffff"),
		FollowersCount: followers, MediaCount: 150 + int(mockrand.Seed(key)%1400),
		Media: m.mockIgMedia(key, 25, followers, false),
	}, nil
}

func (m Mock) GetFacebookPage(_ context.Context, _, pageID string) (*FbPage, error) {
	key := "fb:" + pageID
	followers := baseFollowers(key, 5000, 120000)
	username := "page_" + base36(mockrand.Seed(key)%46655)
	return &FbPage{ID: pageID, Name: titleCase(username), Username: &username, FanCount: round(float64(followers) * 0.94), FollowersCount: followers}, nil
}

func (m Mock) GetPageInsights(_ context.Context, _, pageID string, since, until time.Time) ([]FbPageDailyInsight, error) {
	followers := float64(baseFollowers("fb:"+pageID, 5000, 120000))
	out := []FbPageDailyInsight{}
	dates.Range{From: dates.Day(since), To: dates.Day(until)}.Each(func(d time.Time) {
		r := mockrand.New("fbi:" + pageID + ":" + dates.Format(d))
		unique := jsRound(followers * (0.05 + r.Float64()*0.05))
		out = append(out, FbPageDailyInsight{Date: dates.Format(d), ImpressionsUnique: int(unique), PostEngagements: round(unique * (0.03 + r.Float64()*0.03))})
	})
	return out, nil
}

func (m Mock) GetPagePosts(_ context.Context, _, pageID string, limit int) ([]FbPost, error) {
	if limit <= 0 {
		limit = 25
	}
	key := "fb:" + pageID
	media := m.mockIgMedia(key, limit, baseFollowers(key, 5000, 120000), false)
	out := make([]FbPost, 0, len(media))
	for i, md := range media {
		link := fmt.Sprintf("https://www.facebook.com/%s/posts/%d", pageID, mockrand.Seed(fmt.Sprintf("%s:%d", key, i)))
		out = append(out, FbPost{ID: fmt.Sprintf("%s_post_%d", pageID, i), Message: md.Caption, CreatedTime: md.Timestamp, PermalinkURL: &link, Shares: round(float64(md.LikeCount) * 0.05), Reactions: md.LikeCount, Comments: md.CommentsCount})
	}
	return out, nil
}
