package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/publishing/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/providers/meta"
)

const (
	ProviderName = "meta-publisher"
	// Video containers are polled every 3 s up to 40 times (about 2 minutes).
	DefaultPollInterval = 3 * time.Second
	DefaultPollMax      = 40
)

// Real is the Meta Graph v21.0 publisher. Instagram: container(s) then
// media_publish; Facebook Page: /feed, /photos or /videos. TikTok is refused
// until the Content Posting API is approved.
type Real struct {
	Client       *meta.Client
	PollInterval time.Duration
	PollMax      int
}

func NewReal() *Real {
	return &Real{Client: meta.NewClient(ProviderName, 60*time.Second), PollInterval: DefaultPollInterval, PollMax: DefaultPollMax}
}

func (r *Real) Publish(ctx context.Context, in domain.PublishInput) (*domain.PublishResult, error) {
	story := in.Format == domain.FormatStory
	switch {
	case in.Platform == domain.PlatformInstagram && story:
		return r.instagramStory(ctx, in)
	case in.Platform == domain.PlatformInstagram:
		return r.instagram(ctx, in)
	case in.Platform == domain.PlatformFacebook && story:
		return r.facebookStory(ctx, in)
	case in.Platform == domain.PlatformFacebook:
		return r.facebook(ctx, in)
	}
	return nil, apperr.FromProvider("tiktok", apperr.Permission, "Publikasi TikTok belum tersedia — menunggu persetujuan Content Posting API", nil)
}

type idResponse struct {
	ID     string `json:"id"`
	PostID string `json:"post_id"`
}

func (r *Real) post(ctx context.Context, token, path string, form url.Values) (idResponse, error) {
	var out idResponse
	err := r.Client.Post(ctx, token, path, form, &out)
	return out, err
}

func (r *Real) waitForContainer(ctx context.Context, token, containerID string) error {
	for i := 0; i < r.PollMax; i++ {
		var st struct {
			StatusCode string `json:"status_code"`
		}
		if err := r.Client.Get(ctx, token, "/"+containerID, url.Values{"fields": {"status_code"}}, &st); err != nil {
			return err
		}
		switch st.StatusCode {
		case "FINISHED":
			return nil
		case "ERROR", "EXPIRED":
			return apperr.FromProvider(ProviderName, apperr.Unknown, fmt.Sprintf("Kontainer media Instagram gagal diproses (%s)", st.StatusCode), nil)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(r.PollInterval):
		}
	}
	return apperr.FromProvider(ProviderName, apperr.Network, "Menunggu pemrosesan video Instagram terlalu lama", nil)
}

func (r *Real) igContainer(ctx context.Context, token, igUserID string, m domain.PublishMedia, extra url.Values) (string, error) {
	form := url.Values{}
	for k, v := range extra {
		form[k] = v
	}
	if m.Kind == domain.MediaVideo {
		mediaType := "REELS"
		if extra.Get("is_carousel_item") != "" {
			mediaType = "VIDEO"
		}
		form.Set("media_type", mediaType)
		form.Set("video_url", m.URL)
	} else {
		form.Set("image_url", m.URL)
	}
	res, err := r.post(ctx, token, "/"+igUserID+"/media", form)
	if err != nil {
		return "", err
	}
	if m.Kind == domain.MediaVideo {
		if err := r.waitForContainer(ctx, token, res.ID); err != nil {
			return "", err
		}
	}
	return res.ID, nil
}

func (r *Real) instagram(ctx context.Context, in domain.PublishInput) (*domain.PublishResult, error) {
	token, ig := in.AccessToken, in.AccountExternalID
	if len(in.Media) == 0 {
		return nil, apperr.FromProvider(ProviderName, apperr.Unknown, "Instagram membutuhkan minimal satu media", nil)
	}
	var creationID string
	if len(in.Media) == 1 {
		id, err := r.igContainer(ctx, token, ig, in.Media[0], url.Values{"caption": {in.Text}})
		if err != nil {
			return nil, err
		}
		creationID = id
	} else {
		children := make([]string, 0, len(in.Media))
		for _, m := range in.Media {
			id, err := r.igContainer(ctx, token, ig, m, url.Values{"is_carousel_item": {"true"}})
			if err != nil {
				return nil, err
			}
			children = append(children, id)
		}
		res, err := r.post(ctx, token, "/"+ig+"/media", url.Values{"media_type": {"CAROUSEL"}, "children": {strings.Join(children, ",")}, "caption": {in.Text}})
		if err != nil {
			return nil, err
		}
		creationID = res.ID
	}
	published, err := r.post(ctx, token, "/"+ig+"/media_publish", url.Values{"creation_id": {creationID}})
	if err != nil {
		return nil, err
	}
	var link struct {
		Permalink string `json:"permalink"`
	}
	if err := r.Client.Get(ctx, token, "/"+published.ID, url.Values{"fields": {"permalink"}}, &link); err != nil {
		return nil, err
	}
	r.firstComment(ctx, token, published.ID, in.FirstComment)
	return &domain.PublishResult{ExternalID: published.ID, Permalink: nilIfEmpty(link.Permalink)}, nil
}

func (r *Real) facebook(ctx context.Context, in domain.PublishInput) (*domain.PublishResult, error) {
	token, page := in.AccessToken, in.AccountExternalID
	var video *domain.PublishMedia
	var images []domain.PublishMedia
	for i := range in.Media {
		m := in.Media[i]
		if m.Kind == domain.MediaVideo {
			if video == nil {
				video = &m
			}
		} else {
			images = append(images, m)
		}
	}
	var postID string
	switch {
	case video != nil:
		res, err := r.post(ctx, token, "/"+page+"/videos", url.Values{"file_url": {video.URL}, "description": {in.Text}})
		if err != nil {
			return nil, err
		}
		postID = res.ID
	case len(images) == 1:
		res, err := r.post(ctx, token, "/"+page+"/photos", url.Values{"url": {images[0].URL}, "message": {in.Text}})
		if err != nil {
			return nil, err
		}
		postID = res.PostID
		if postID == "" {
			postID = res.ID
		}
	case len(images) > 1:
		form := url.Values{"message": {in.Text}}
		for i, img := range images {
			res, err := r.post(ctx, token, "/"+page+"/photos", url.Values{"url": {img.URL}, "published": {"false"}})
			if err != nil {
				return nil, err
			}
			attached, _ := json.Marshal(map[string]string{"media_fbid": res.ID})
			form.Set(fmt.Sprintf("attached_media[%d]", i), string(attached))
		}
		res, err := r.post(ctx, token, "/"+page+"/feed", form)
		if err != nil {
			return nil, err
		}
		postID = res.ID
	default:
		form := url.Values{"message": {in.Text}}
		if in.LinkURL != "" {
			form.Set("link", in.LinkURL)
		}
		res, err := r.post(ctx, token, "/"+page+"/feed", form)
		if err != nil {
			return nil, err
		}
		postID = res.ID
	}
	var link struct {
		PermalinkURL string `json:"permalink_url"`
	}
	// Permalink lookup failures are swallowed, as before.
	_ = r.Client.Get(ctx, token, "/"+postID, url.Values{"fields": {"permalink_url"}}, &link)
	r.firstComment(ctx, token, postID, in.FirstComment)
	return &domain.PublishResult{ExternalID: postID, Permalink: nilIfEmpty(link.PermalinkURL)}, nil
}

// instagramStory publishes one image or video as a story: a STORIES
// container (stories take no caption), then media_publish.
func (r *Real) instagramStory(ctx context.Context, in domain.PublishInput) (*domain.PublishResult, error) {
	token, ig := in.AccessToken, in.AccountExternalID
	if len(in.Media) != 1 {
		return nil, apperr.FromProvider(ProviderName, apperr.Unknown, domain.MsgStoryOneMedia, nil)
	}
	m := in.Media[0]
	form := url.Values{"media_type": {"STORIES"}}
	if m.Kind == domain.MediaVideo {
		form.Set("video_url", m.URL)
	} else {
		form.Set("image_url", m.URL)
	}
	container, err := r.post(ctx, token, "/"+ig+"/media", form)
	if err != nil {
		return nil, err
	}
	if m.Kind == domain.MediaVideo {
		if err := r.waitForContainer(ctx, token, container.ID); err != nil {
			return nil, err
		}
	}
	published, err := r.post(ctx, token, "/"+ig+"/media_publish", url.Values{"creation_id": {container.ID}})
	if err != nil {
		return nil, err
	}
	var link struct {
		Permalink string `json:"permalink"`
	}
	// Stories expire after 24 hours; a missing permalink is not a failure.
	_ = r.Client.Get(ctx, token, "/"+published.ID, url.Values{"fields": {"permalink"}}, &link)
	return &domain.PublishResult{ExternalID: published.ID, Permalink: nilIfEmpty(link.Permalink)}, nil
}

// facebookStory posts a photo story: the photo is uploaded unpublished,
// then attached with /photo_stories. Video stories need the resumable
// upload flow and are refused for now.
func (r *Real) facebookStory(ctx context.Context, in domain.PublishInput) (*domain.PublishResult, error) {
	token, page := in.AccessToken, in.AccountExternalID
	if len(in.Media) != 1 {
		return nil, apperr.FromProvider(ProviderName, apperr.Unknown, domain.MsgStoryOneMedia, nil)
	}
	if in.Media[0].Kind == domain.MediaVideo {
		return nil, apperr.FromProvider(ProviderName, apperr.Permission, "Story video Facebook belum didukung. Gunakan gambar.", nil)
	}
	photo, err := r.post(ctx, token, "/"+page+"/photos", url.Values{"url": {in.Media[0].URL}, "published": {"false"}})
	if err != nil {
		return nil, err
	}
	story, err := r.post(ctx, token, "/"+page+"/photo_stories", url.Values{"photo_id": {photo.ID}})
	if err != nil {
		return nil, err
	}
	id := story.PostID
	if id == "" {
		id = story.ID
	}
	return &domain.PublishResult{ExternalID: id}, nil
}

// firstComment is best effort: errors are ignored.
func (r *Real) firstComment(ctx context.Context, token, objectID, comment string) {
	if strings.TrimSpace(comment) == "" {
		return
	}
	_, _ = r.post(ctx, token, "/"+objectID+"/comments", url.Values{"message": {strings.TrimSpace(comment)}})
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
