package apify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/httpx"
)

const (
	providerName = "apify"
	defaultBase  = "https://api.apify.com/v2"
)

type Real struct {
	Base   string
	Token  string
	Actors Actors
	// HTTP is built per call so the timeout follows the actor timeout.
	newHTTP func(timeout time.Duration) *httpx.Client
}

func NewReal(token string, actors Actors) *Real {
	return &Real{Base: defaultBase, Token: token, Actors: actors, newHTTP: httpx.New}
}

func mapStatus(status int, body string) *apperr.Error {
	switch status {
	case 401, 403:
		return apperr.FromProvider(providerName, apperr.TokenExpired, "Token Apify ditolak.", nil)
	case 402:
		e := apperr.FromProvider(providerName, apperr.Permission, "Kuota/kredit Apify habis (402).", nil)
		e.RequiresReconnect = false
		return e
	case 404:
		return apperr.FromProvider(providerName, apperr.NotFound, "Actor Apify tidak ditemukan — periksa override actor.", nil)
	case 408:
		return apperr.FromProvider(providerName, apperr.Network, "Run Apify melebihi batas waktu sinkron (408).", nil)
	case 429:
		return apperr.FromProvider(providerName, apperr.RateLimit, "Terlalu banyak permintaan ke Apify (429).", nil)
	}
	if len(body) > 200 {
		body = body[:200]
	}
	return apperr.FromProvider(providerName, apperr.Unknown, fmt.Sprintf("Apify menjawab %d: %s", status, body), nil)
}

type runOpts struct {
	timeoutSecs int
	memoryMb    int
	maxItems    int
}

// RunActor runs an actor synchronously and returns its dataset items.
func (r *Real) RunActor(ctx context.Context, actorID string, input map[string]any, o runOpts) ([]json.RawMessage, error) {
	if r.Token == "" {
		e := apperr.FromProvider(providerName, apperr.Permission, "APIFY_TOKEN belum diatur.", nil)
		e.RequiresReconnect = false
		return nil, e
	}
	if o.timeoutSecs == 0 {
		o.timeoutSecs = 120
	}
	if o.memoryMb == 0 {
		o.memoryMb = 1024
	}
	q := url.Values{"token": {r.Token}, "timeout": {strconv.Itoa(o.timeoutSecs)}, "memory": {strconv.Itoa(o.memoryMb)}, "clean": {"true"}, "format": {"json"}}
	if o.maxItems > 0 {
		q.Set("limit", strconv.Itoa(o.maxItems))
	}
	u := fmt.Sprintf("%s/acts/%s/run-sync-get-dataset-items?%s", r.Base, strings.Replace(actorID, "/", "~", 1), q.Encode())
	client := r.newHTTP(time.Duration(o.timeoutSecs+15) * time.Second)
	res, err := client.PostJSON(ctx, u, input, nil)
	if err != nil {
		return nil, apperr.FromProvider(providerName, apperr.Network, "Tidak bisa menghubungi Apify: "+err.Error(), err)
	}
	if res.Status >= 400 {
		return nil, mapStatus(res.Status, string(res.Body))
	}
	var items []json.RawMessage
	if err := json.Unmarshal(res.Body, &items); err != nil {
		return []json.RawMessage{}, nil
	}
	return items, nil
}

func (r *Real) tiktok(ctx context.Context, username string, limit int) (*Profile, []Post, error) {
	u := cleanUsername(username)
	per := limit
	if per < 1 {
		per = 1
	}
	if per > 100 {
		per = 100
	}
	items, err := r.RunActor(ctx, r.Actors.TikTok, map[string]any{
		"profiles": []string{u}, "resultsPerPage": per, "profileSorting": "latest", "profileScrapeSections": []string{"videos"},
		"shouldDownloadVideos": false, "shouldDownloadCovers": false, "shouldDownloadSubtitles": false, "shouldDownloadSlideshowImages": false,
	}, runOpts{timeoutSecs: 180, maxItems: limit})
	if err != nil {
		return nil, nil, err
	}
	profile, posts := NormalizeTikTokItems(items, u)
	if profile == nil {
		return nil, nil, apperr.FromProvider(providerName, apperr.NotFound, fmt.Sprintf("Akun TikTok @%s tidak ditemukan atau belum punya video publik.", u), nil)
	}
	return profile, posts, nil
}

func (r *Real) Profile(ctx context.Context, platform, username string) (*Profile, error) {
	u := cleanUsername(username)
	switch platform {
	case "INSTAGRAM":
		items, err := r.RunActor(ctx, r.Actors.InstagramProfile, map[string]any{"usernames": []string{u}}, runOpts{timeoutSecs: 120, maxItems: 1})
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			return nil, apperr.FromProvider(providerName, apperr.NotFound, fmt.Sprintf("Akun Instagram @%s tidak ditemukan.", u), nil)
		}
		p := NormalizeInstagramProfile(items[0], u)
		return &p, nil
	case "TIKTOK":
		p, _, err := r.tiktok(ctx, u, 1)
		return p, err
	}
	items, err := r.RunActor(ctx, r.Actors.FacebookPage, map[string]any{"startUrls": []map[string]string{{"url": ProfileURL("FACEBOOK", u)}}}, runOpts{timeoutSecs: 120, maxItems: 1})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, apperr.FromProvider(providerName, apperr.NotFound, fmt.Sprintf("Halaman Facebook %s tidak ditemukan.", u), nil)
	}
	p := NormalizeFacebookPage(items[0], u)
	return &p, nil
}

func (r *Real) Posts(ctx context.Context, platform, username string, limit int) ([]Post, error) {
	u := cleanUsername(username)
	if limit <= 0 {
		limit = 30
	}
	switch platform {
	case "INSTAGRAM":
		items, err := r.RunActor(ctx, r.Actors.InstagramPosts, map[string]any{"directUrls": []string{ProfileURL("INSTAGRAM", u)}, "resultsType": "posts", "resultsLimit": limit, "addParentData": false}, runOpts{timeoutSecs: 180, maxItems: limit})
		if err != nil {
			return nil, err
		}
		out := []Post{}
		for _, it := range items {
			if p := NormalizeInstagramPost(it); p != nil {
				out = append(out, *p)
			}
		}
		return out, nil
	case "TIKTOK":
		_, posts, err := r.tiktok(ctx, u, limit)
		return posts, err
	}
	items, err := r.RunActor(ctx, r.Actors.FacebookPosts, map[string]any{"startUrls": []map[string]string{{"url": ProfileURL("FACEBOOK", u)}}, "resultsLimit": limit}, runOpts{timeoutSecs: 180, maxItems: limit})
	if err != nil {
		return nil, err
	}
	out := []Post{}
	for _, it := range items {
		if p := NormalizeFacebookPost(it); p != nil {
			out = append(out, *p)
		}
	}
	return out, nil
}
