package apify

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestActorOverrides(t *testing.T) {
	a := ParseActorOverrides(`{"tiktok":"me/tt","serp":"bad id","unknown":"x/y"}`)
	if a.TikTok != "me/tt" || a.Serp != DefaultActors.Serp {
		t.Fatalf("%+v", a)
	}
	if ParseActorOverrides("{nope") != DefaultActors {
		t.Fatal("invalid json keeps defaults")
	}
}

func TestNormalizers(t *testing.T) {
	p := NormalizeInstagramPost(map[string]any{"shortCode": "abc", "type": "Sidecar", "likesCount": "1,234", "timestamp": float64(1757000000)})
	if p.MediaType != "CAROUSEL_ALBUM" || p.Likes != 1234 || *p.Permalink != "https://www.instagram.com/p/abc/" || p.PublishedAt.Year() != 2025 {
		t.Fatalf("%+v", p)
	}
	items := []json.RawMessage{json.RawMessage(`{"id":"v1","authorMeta":{"name":"Kopi","fans":900,"heart":0},"diggCount":5,"createTimeISO":"2026-09-01T00:00:00Z"}`)}
	prof, posts := NormalizeTikTokItems(items, "kopi")
	if prof.Username != "kopi" || prof.Followers != 900 || prof.TotalLikes != nil || len(posts) != 1 || posts[0].MediaType != "VIDEO" {
		t.Fatalf("%+v %+v", prof, posts)
	}
}

func TestMockDeterministic(t *testing.T) {
	m := Mock{Now: func() time.Time { return time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC) }}
	a, _ := m.Profile(context.Background(), "FACEBOOK", "@Kopi")
	b, _ := m.Profile(context.Background(), "FACEBOOK", "kopi")
	if !reflect.DeepEqual(a, b) || a.Following != nil {
		t.Fatal("profile")
	}
	pa, _ := m.Posts(context.Background(), "TIKTOK", "kopi", 5)
	pb, _ := m.Posts(context.Background(), "TIKTOK", "kopi", 5)
	if !reflect.DeepEqual(pa, pb) || pa[0].MediaType != "VIDEO" {
		t.Fatal("posts")
	}
}
