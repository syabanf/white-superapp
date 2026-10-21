package app

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/syabanf/white-superapp/white-backend/internal/publishing/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

func trim(s string) string { return strings.TrimSpace(s) }

type Media struct{ Deps }

func NewMedia(d Deps) *Media { return &Media{d} }

type UploadFile struct {
	Filename string
	Content  []byte
}

func (m *Media) List(ctx context.Context, userID, clientID string) ([]domain.MediaAsset, error) {
	if _, err := m.access(ctx, userID, clientID); err != nil {
		return nil, err
	}
	return m.Store.ListMedia(ctx, clientID)
}

// Upload validates every file (size, magic bytes) before storing any. The
// browser MIME type is ignored; the sniffed type is authoritative.
func (m *Media) Upload(ctx context.Context, userID, clientID string, files []UploadFile) ([]domain.MediaAsset, error) {
	if clientID == "" || len(files) == 0 {
		return nil, apperr.New(apperr.Validation, "")
	}
	if _, err := m.manage(ctx, userID, clientID); err != nil {
		return nil, err
	}
	types := make([]string, len(files))
	for i, f := range files {
		if len(f.Content) > domain.MaxUploadBytes {
			return nil, apperr.New(apperr.Validation, msgFileTooLarge+" ("+f.Filename+")")
		}
		types[i] = domain.SniffContentType(f.Content)
		if types[i] == "" {
			return nil, apperr.New(apperr.Validation, msgFileTypeNotAllowed+" ("+f.Filename+")")
		}
	}
	out := make([]domain.MediaAsset, 0, len(files))
	for i, f := range files {
		obj, err := m.Storage.Put(ctx, clientID, types[i], f.Content)
		if err != nil {
			if errors.Is(err, domain.ErrUnsupportedType) || errors.Is(err, domain.ErrTooLarge) {
				return out, apperr.New(apperr.Validation, msgFileTypeNotAllowed+" ("+f.Filename+")")
			}
			return out, err
		}
		asset, err := m.Store.InsertMedia(ctx, domain.MediaAsset{
			ID: ids.New(), ClientID: clientID, Kind: domain.KindForMime(types[i]), URL: obj.URL,
			Filename: domain.TruncateRunes(f.Filename, 200), MimeType: types[i], SizeBytes: int32(obj.SizeBytes),
			Tags: []string{}, UploadedByID: &userID,
		})
		if err != nil {
			return out, err
		}
		out = append(out, *asset)
	}
	return out, nil
}

type ExternalMediaInput struct {
	URL     string
	Kind    domain.MediaKind
	AltText string
	Tags    []string
}

// AddExternal registers an https URL without fetching it.
func (m *Media) AddExternal(ctx context.Context, userID, clientID string, in ExternalMediaInput) (*domain.MediaAsset, error) {
	u, err := url.Parse(in.URL)
	if err != nil || u.Scheme != "https" || u.Host == "" || utf8.RuneCountInString(in.URL) > 2000 || (in.Kind != domain.MediaImage && in.Kind != domain.MediaVideo) {
		return nil, apperr.New(apperr.Validation, msgInvalidHTTPSURL)
	}
	if utf8.RuneCountInString(in.AltText) > 500 {
		return nil, apperr.New(apperr.Validation, msgInvalidHTTPSURL)
	}
	tags, err := normalizeTags(in.Tags)
	if err != nil {
		return nil, apperr.New(apperr.Validation, msgInvalidHTTPSURL)
	}
	if _, err := m.manage(ctx, userID, clientID); err != nil {
		return nil, err
	}
	decoded, derr := url.PathUnescape(u.Path)
	if derr != nil {
		decoded = u.Path
	}
	filename := domain.ExternalFilename(decoded)
	return m.Store.InsertMedia(ctx, domain.MediaAsset{
		ID: ids.New(), ClientID: clientID, Kind: in.Kind, URL: in.URL, Filename: filename,
		MimeType: domain.ExternalMime(in.Kind, filename), AltText: validate.EmptyToNil(in.AltText), Tags: tags, UploadedByID: &userID,
	})
}

// normalizeTags trims, requires 1..40 characters each, caps at 20 and dedupes.
func normalizeTags(tags []string) ([]string, error) {
	if len(tags) > 20 {
		return nil, errors.New("tags")
	}
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = trim(t)
		if n := utf8.RuneCountInString(t); n < 1 || n > 40 {
			return nil, errors.New("tags")
		}
		out = append(out, t)
	}
	return dedupe(out), nil
}

// Update changes alt text (nil = unchanged, "" = clear) and tags (nil = unchanged).
func (m *Media) Update(ctx context.Context, userID, clientID, mediaID string, altText *string, tags []string) (*domain.MediaAsset, error) {
	if altText != nil && utf8.RuneCountInString(*altText) > 500 {
		return nil, apperr.New(apperr.Validation, "")
	}
	var newTags []string
	if tags != nil {
		var err error
		if newTags, err = normalizeTags(tags); err != nil {
			return nil, apperr.New(apperr.Validation, "")
		}
	}
	existing, err := m.Store.GetMedia(ctx, mediaID)
	if err != nil {
		return nil, err
	}
	if existing == nil || existing.ClientID != clientID {
		return nil, apperr.New(apperr.NotFound, "")
	}
	if _, err := m.manage(ctx, userID, clientID); err != nil {
		return nil, err
	}
	alt := existing.AltText
	if altText != nil {
		alt = trimToNil(altText)
	}
	if tags == nil {
		newTags = existing.Tags
	}
	return m.Store.UpdateMedia(ctx, mediaID, alt, newTags)
}

// Delete refuses assets still referenced by a post, then removes the row
// and the stored file (external URLs have no file).
func (m *Media) Delete(ctx context.Context, userID, clientID, mediaID string) error {
	existing, err := m.Store.GetMedia(ctx, mediaID)
	if err != nil {
		return err
	}
	if existing == nil || existing.ClientID != clientID {
		return apperr.New(apperr.NotFound, "")
	}
	if _, err := m.manage(ctx, userID, clientID); err != nil {
		return err
	}
	if existing.UsageCount > 0 {
		return apperr.New(apperr.Validation, msgMediaInUse)
	}
	if _, err := m.Store.DeleteMedia(ctx, mediaID); err != nil {
		return err
	}
	if key := domain.KeyFromURL(existing.URL); key != "" {
		if err := m.Storage.Delete(ctx, key); err != nil {
			m.log().Warn("media file not removed", "key", key, "err", err)
		}
	}
	return nil
}

// Object serves a stored file for the gateway; nil when missing.
func (m *Media) Object(ctx context.Context, key string) (*domain.Object, error) {
	if !domain.IsSafeKey(key) {
		return nil, nil
	}
	return m.Storage.Get(ctx, key)
}
