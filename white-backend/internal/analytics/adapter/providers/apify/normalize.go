package apify

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Actor schemas drift between versions, so every reader accepts several
// field names, coerces numbers and falls back to a sensible default.

type rec map[string]any

func asRec(v any) rec {
	switch x := v.(type) {
	case map[string]any:
		return x
	case json.RawMessage:
		var m map[string]any
		if json.Unmarshal(x, &m) == nil {
			return m
		}
	}
	return rec{}
}

func str(v any) *string {
	switch x := v.(type) {
	case string:
		if x != "" {
			return &x
		}
	case float64:
		s := strconv.FormatFloat(x, 'f', -1, 64)
		return &s
	}
	return nil
}

var nonDigit = regexp.MustCompile(`[^\d.-]`)

func num(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case string:
		f, err := strconv.ParseFloat(nonDigit.ReplaceAllString(x, ""), 64)
		if err == nil {
			return int(f)
		}
	}
	return 0
}

func first(o rec, keys ...string) any {
	for _, k := range keys {
		if v, ok := o[k]; ok && v != nil && v != "" {
			return v
		}
	}
	return nil
}

func orStr(v any, fallback string) string {
	if s := str(v); s != nil {
		return *s
	}
	return fallback
}

func nonZero(n int) *int {
	if n == 0 {
		return nil
	}
	return &n
}

// iso parses actor timestamps: unix seconds (< 1e12), unix millis or a
// date string; anything else becomes the epoch.
func iso(v any) time.Time {
	switch x := v.(type) {
	case float64:
		if x < 1e12 {
			return time.Unix(int64(x), 0).UTC()
		}
		return time.UnixMilli(int64(x)).UTC()
	case string:
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"} {
			if t, err := time.Parse(layout, x); err == nil {
				return t.UTC()
			}
		}
	}
	return time.Unix(0, 0).UTC()
}

func igMediaType(t any) string {
	switch strings.ToLower(orStr(t, "")) {
	case "video", "reel":
		return "VIDEO"
	case "sidecar", "carousel", "carousel_album":
		return "CAROUSEL_ALBUM"
	}
	return "IMAGE"
}

func sp(s string) *string { return &s }

func NormalizeInstagramPost(raw any) *Post {
	p := asRec(raw)
	id := str(first(p, "id", "shortCode", "code"))
	if id == nil {
		return nil
	}
	product := "FEED"
	if pt := strings.ToLower(orStr(p["productType"], "")); pt == "clips" || pt == "reels" {
		product = "REELS"
	}
	permalink := str(p["url"])
	if permalink == nil {
		if sc := str(p["shortCode"]); sc != nil {
			permalink = sp("https://www.instagram.com/p/" + *sc + "/")
		}
	}
	return &Post{
		ExternalID: *id, Caption: orStr(p["caption"], ""), MediaType: igMediaType(p["type"]), ProductType: &product, Permalink: permalink,
		MediaURL: str(first(p, "videoUrl", "displayUrl")), ThumbnailURL: str(first(p, "displayUrl", "thumbnailUrl")),
		PublishedAt: iso(first(p, "timestamp", "takenAt", "taken_at")),
		Likes:       num(first(p, "likesCount", "likes")), Comments: num(first(p, "commentsCount", "comments")),
		Shares: num(first(p, "sharesCount", "reshareCount")), Saves: num(first(p, "savesCount")), Views: num(first(p, "videoPlayCount", "videoViewCount", "playCount")),
	}
}

func NormalizeInstagramProfile(raw any, fallback string) Profile {
	p := asRec(raw)
	return Profile{
		Platform: "INSTAGRAM", ExternalID: orStr(first(p, "id", "pk"), "ig_"+fallback),
		Username: strings.ToLower(orStr(p["username"], fallback)), DisplayName: orStr(first(p, "fullName", "full_name"), fallback),
		Biography: str(p["biography"]), AvatarURL: str(first(p, "profilePicUrlHD", "profilePicUrl", "profile_pic_url")),
		Followers: num(first(p, "followersCount", "followers")), Following: nonZero(num(first(p, "followsCount", "following"))),
		MediaCount: num(first(p, "postsCount", "mediaCount")),
	}
}

func NormalizeTikTokItems(items []json.RawMessage, fallback string) (*Profile, []Post) {
	var profile *Profile
	posts := []Post{}
	for _, raw := range items {
		it := asRec(raw)
		a := asRec(first(it, "authorMeta", "author"))
		if profile == nil && len(a) > 0 {
			profile = &Profile{
				Platform: "TIKTOK", ExternalID: orStr(a["id"], "tt_"+fallback),
				Username: strings.ToLower(orStr(first(a, "name", "uniqueId"), fallback)), DisplayName: orStr(first(a, "nickName", "nickname"), fallback),
				Biography: str(a["signature"]), AvatarURL: str(first(a, "avatar", "avatarLarger", "avatarThumb")),
				Followers: num(first(a, "fans", "followerCount")), Following: nonZero(num(first(a, "following", "followingCount"))),
				MediaCount: num(first(a, "video", "videoCount")), TotalLikes: nonZero(num(first(a, "heart", "heartCount"))),
			}
		}
		id := str(it["id"])
		if id == nil {
			continue
		}
		video := asRec(it["videoMeta"])
		thumb := str(first(video, "coverUrl", "originalCoverUrl"))
		if thumb == nil {
			thumb = str(first(it, "cover", "coverUrl"))
		}
		posts = append(posts, Post{
			ExternalID: *id, Caption: orStr(first(it, "text", "desc"), ""), MediaType: "VIDEO", ProductType: sp("REELS"),
			Permalink: str(first(it, "webVideoUrl", "url")), MediaURL: str(first(video, "downloadAddr", "playAddr")), ThumbnailURL: thumb,
			PublishedAt: iso(first(it, "createTimeISO", "createTime")),
			Likes:       num(first(it, "diggCount", "likes")), Comments: num(first(it, "commentCount", "comments")),
			Shares: num(first(it, "shareCount", "shares")), Saves: num(first(it, "collectCount", "saves")), Views: num(first(it, "playCount", "views")),
		})
	}
	return profile, posts
}

func NormalizeFacebookPage(raw any, fallback string) Profile {
	p := asRec(raw)
	return Profile{
		Platform: "FACEBOOK", ExternalID: orStr(first(p, "pageId", "facebookId", "id"), "fb_"+fallback),
		Username: strings.ToLower(orStr(first(p, "pageName", "username"), fallback)), DisplayName: orStr(first(p, "title", "name", "pageName"), fallback),
		Biography: str(first(p, "intro", "about", "description")), AvatarURL: str(first(p, "profilePictureUrl", "profilePhoto", "pageLogo")),
		Followers: num(first(p, "followers", "followersCount")), Following: nil, MediaCount: num(first(p, "postsCount")),
		TotalLikes: nonZero(num(first(p, "likes", "likesCount"))),
	}
}

var videoRe = regexp.MustCompile(`(?i)video`)

func NormalizeFacebookPost(raw any) *Post {
	p := asRec(raw)
	id := str(first(p, "postId", "id", "legacyId"))
	if id == nil {
		return nil
	}
	mediaList, _ := p["media"].([]any)
	media := rec{}
	if len(mediaList) > 0 {
		media = asRec(mediaList[0])
	}
	mediaType := "IMAGE"
	hasVideo := false
	for _, m := range mediaList {
		mr := asRec(m)
		if videoRe.MatchString(orStr(first(mr, "__typename", "type"), "")) {
			hasVideo = true
		}
	}
	if hasVideo {
		mediaType = "VIDEO"
	} else if len(mediaList) > 1 {
		mediaType = "CAROUSEL_ALBUM"
	}
	return &Post{
		ExternalID: *id, Caption: orStr(first(p, "text", "message"), ""), MediaType: mediaType, ProductType: sp("FEED"),
		Permalink: str(first(p, "url", "postUrl", "topLevelUrl")), MediaURL: str(first(media, "url", "photo_image", "image")), ThumbnailURL: str(first(media, "thumbnail", "photo_image", "image")),
		PublishedAt: iso(first(p, "time", "timestamp", "publishedAt")),
		Likes:       num(first(p, "likes", "likesCount", "reactionsCount")), Comments: num(first(p, "comments", "commentsCount")),
		Shares: num(first(p, "shares", "sharesCount")), Saves: 0, Views: num(first(p, "viewsCount", "videoViewCount")),
	}
}
