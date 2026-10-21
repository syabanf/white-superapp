// Package apify reads public Instagram, TikTok and Facebook profiles and
// posts through Apify actors. Real needs APIFY_TOKEN; the mock is
// deterministic and shares seeds with the TS mock.
package apify

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

type Profile struct {
	Platform    string
	ExternalID  string
	Username    string
	DisplayName string
	Biography   *string
	AvatarURL   *string
	Followers   int
	Following   *int
	MediaCount  int
	TotalLikes  *int
}

type Post struct {
	ExternalID   string
	Caption      string
	MediaType    string
	ProductType  *string
	Permalink    *string
	MediaURL     *string
	ThumbnailURL *string
	PublishedAt  time.Time
	Likes        int
	Comments     int
	Shares       int
	Saves        int
	Views        int
}

type PublicSocial interface {
	Profile(ctx context.Context, platform, username string) (*Profile, error)
	Posts(ctx context.Context, platform, username string, limit int) ([]Post, error)
}

// Actors are the actor ids per job; the setup wizard can override them.
type Actors struct {
	Serp             string
	InstagramProfile string
	InstagramPosts   string
	TikTok           string
	FacebookPage     string
	FacebookPosts    string
}

var DefaultActors = Actors{
	Serp:             "apify/google-search-scraper",
	InstagramProfile: "apify/instagram-profile-scraper",
	InstagramPosts:   "apify/instagram-scraper",
	TikTok:           "clockworks/tiktok-scraper",
	FacebookPage:     "apify/facebook-pages-scraper",
	FacebookPosts:    "apify/facebook-posts-scraper",
}

var actorIDRe = regexp.MustCompile(`^[\w.-]+/[\w.-]+$`)

// ParseActorOverrides merges a JSON object like {"serp":"owner/name"} over
// the defaults. Unknown keys are ignored; invalid JSON keeps the defaults.
func ParseActorOverrides(raw string) Actors {
	out := DefaultActors
	if strings.TrimSpace(raw) == "" {
		return out
	}
	var o map[string]any
	if err := json.Unmarshal([]byte(raw), &o); err != nil {
		return out
	}
	set := func(dst *string, key string) {
		if v, ok := o[key].(string); ok && actorIDRe.MatchString(v) {
			*dst = v
		}
	}
	set(&out.Serp, "serp")
	set(&out.InstagramProfile, "instagramProfile")
	set(&out.InstagramPosts, "instagramPosts")
	set(&out.TikTok, "tiktok")
	set(&out.FacebookPage, "facebookPage")
	set(&out.FacebookPosts, "facebookPosts")
	return out
}

// Pick returns the real adapter when a token is set. actorsJSON is the
// public half of the APIFY integration.
func Pick(token, actorsJSON string) PublicSocial {
	if token != "" {
		return NewReal(token, ParseActorOverrides(actorsJSON))
	}
	return Mock{Now: time.Now}
}

func cleanUsername(u string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(u, "@")))
}

// ProfileURL of a public account.
func ProfileURL(platform, username string) string {
	u := strings.TrimPrefix(username, "@")
	switch platform {
	case "INSTAGRAM":
		return "https://www.instagram.com/" + u + "/"
	case "TIKTOK":
		return "https://www.tiktok.com/@" + u
	}
	return "https://www.facebook.com/" + u
}
