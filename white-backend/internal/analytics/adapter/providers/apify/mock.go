package apify

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/providers/mockrand"
)

// Mock mirrors the Apify adapter shape with deterministic numbers so the
// competitor and sync flows behave the same with or without a token.
type Mock struct {
	Now func() time.Time
}

var mockCaptions = []string{
	"Promo minggu ini — cek highlight kami ✨",
	"Terima kasih sudah mampir hari ini 🙏",
	"Behind the scenes tim kami ☕",
	"Tips singkat untuk kamu yang baru mulai",
	"Koleksi baru sudah tersedia!",
	"Giveaway! Baca syaratnya di caption",
	"Testimoni pelanggan minggu ini ❤️",
	"Jadwal buka libur panjang — simpan ya",
}

func jsRound(x float64) int { return int(math.Floor(x + 0.5)) }

func displayName(u string) string {
	words := strings.Fields(strings.NewReplacer(".", " ", "_", " ").Replace(u))
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

func (m Mock) now() time.Time {
	if m.Now != nil {
		return m.Now()
	}
	return time.Now()
}

func (m Mock) Profile(_ context.Context, platform, username string) (*Profile, error) {
	u := cleanUsername(username)
	rand := mockrand.New("pub:" + platform + ":" + u)
	followers := jsRound(2000 + rand.Float64()*180000)
	bio := fmt.Sprintf("Akun resmi %s. DM untuk kerja sama.", u)
	avatar := "https://api.dicebear.com/9.x/initials/svg?seed=" + url.QueryEscape(u)
	p := &Profile{Platform: platform, ExternalID: strings.ToLower(platform[:2]) + "_pub_" + u, Username: u, DisplayName: displayName(u), Biography: &bio, AvatarURL: &avatar, Followers: followers}
	if platform != "FACEBOOK" {
		f := jsRound(50 + rand.Float64()*900)
		p.Following = &f
	}
	p.MediaCount = jsRound(40 + rand.Float64()*600)
	if platform == "TIKTOK" {
		t := jsRound(float64(followers) * (3 + rand.Float64()*12))
		p.TotalLikes = &t
	}
	return p, nil
}

func (m Mock) Posts(_ context.Context, platform, username string, limit int) ([]Post, error) {
	u := cleanUsername(username)
	if limit <= 0 {
		limit = 30
	}
	rand := mockrand.New("pubposts:" + platform + ":" + u)
	base := 2000 + rand.Float64()*180000
	now := m.now()
	out := make([]Post, 0, limit)
	for i := 0; i < limit; i++ {
		daysAgo := float64(i)*(1+rand.Float64()*2) + rand.Float64()
		isVideo := platform == "TIKTOK" || rand.Float64() < 0.35
		er := 0.01 + rand.Float64()*0.06
		likes := jsRound(base * er)
		mediaType := "IMAGE"
		product := "FEED"
		if isVideo {
			mediaType, product = "VIDEO", "REELS"
		} else if rand.Float64() < 0.3 {
			mediaType = "CAROUSEL_ALBUM"
		}
		sep := "p/"
		if platform == "TIKTOK" {
			sep = "/video/"
		}
		permalink := fmt.Sprintf("%s%s%d", ProfileURL(platform, u), sep, i+1000)
		thumb := fmt.Sprintf("https://picsum.photos/seed/%s-%s-%d/400/400", platform, u, i)
		p := Post{
			ExternalID: fmt.Sprintf("%s_%s_%d", strings.ToLower(platform[:2]), u, i), Caption: mockCaptions[i%len(mockCaptions)],
			MediaType: mediaType, ProductType: &product, Permalink: &permalink, ThumbnailURL: &thumb,
			PublishedAt: now.Add(-time.Duration(daysAgo * 24 * float64(time.Hour))).UTC(),
			Likes:       likes, Comments: jsRound(float64(likes) * (0.02 + rand.Float64()*0.06)),
			Shares: jsRound(float64(likes) * (0.01 + rand.Float64()*0.05)), Saves: jsRound(float64(likes) * (0.02 + rand.Float64()*0.08)),
		}
		if isVideo {
			p.Views = jsRound(float64(likes) * (8 + rand.Float64()*30))
		}
		out = append(out, p)
	}
	return out, nil
}
