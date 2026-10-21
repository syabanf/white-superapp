package domain

import (
	"bytes"
	"errors"
	"path"
	"regexp"
	"strings"
)

// Storage failures the use cases translate into validation messages.
var (
	ErrInvalidKey      = errors.New("INVALID_KEY")
	ErrUnsupportedType = errors.New("UNSUPPORTED_TYPE")
	ErrTooLarge        = errors.New("TOO_LARGE")
)

// Pure parts of lib/storage.ts: the MIME allowlist, magic-byte sniffing and
// key safety. The disk adapter lives in adapter/storage.

const MaxUploadBytes = 25 * 1024 * 1024

// allowedTypes maps a content type to its file extension.
var allowedTypes = []struct{ mime, ext string }{
	{"image/jpeg", "jpg"},
	{"image/png", "png"},
	{"image/webp", "webp"},
	{"image/gif", "gif"},
	{"video/mp4", "mp4"},
	{"video/quicktime", "mov"},
}

// ExtensionFor returns "" for a content type outside the allowlist.
func ExtensionFor(contentType string) string {
	for _, a := range allowedTypes {
		if a.mime == contentType {
			return a.ext
		}
	}
	return ""
}

// ContentTypeForExtension is the inverse, "application/octet-stream" when unknown.
func ContentTypeForExtension(ext string) string {
	for _, a := range allowedTypes {
		if a.ext == ext {
			return a.mime
		}
	}
	return "application/octet-stream"
}

// SniffContentType reads the leading bytes. The browser MIME is a claim;
// this is the evidence. Returns "" for anything outside the allowlist.
func SniffContentType(b []byte) string {
	switch {
	case len(b) >= 3 && b[0] == 0xff && b[1] == 0xd8 && b[2] == 0xff:
		return "image/jpeg"
	case len(b) >= 8 && bytes.Equal(b[:8], []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}):
		return "image/png"
	case len(b) >= 6 && (string(b[:6]) == "GIF87a" || string(b[:6]) == "GIF89a"):
		return "image/gif"
	case len(b) >= 12 && string(b[:4]) == "RIFF" && string(b[8:12]) == "WEBP":
		return "image/webp"
	case len(b) >= 12 && string(b[4:8]) == "ftyp":
		if strings.HasPrefix(string(b[8:12]), "qt") {
			return "video/quicktime"
		}
		return "video/mp4"
	}
	return ""
}

var safeKeyRe = regexp.MustCompile(`^[A-Za-z0-9_-]+/[A-Za-z0-9_.-]+$`)

// IsSafeKey allows only `<segment>/<segment>` in a whitelisted charset.
func IsSafeKey(key string) bool { return safeKeyRe.MatchString(key) }

// MediaKey builds `{clientId}/{id}.{ext}`.
func MediaKey(clientID, id, ext string) string { return clientID + "/" + id + "." + ext }

// PublicMediaURL is `{base}/api/media/{key}` with the trailing slash of base removed.
func PublicMediaURL(base, key string) string {
	return strings.TrimRight(base, "/") + "/api/media/" + key
}

var keyFromURLRe = regexp.MustCompile(`/api/media/([A-Za-z0-9_-]+/[A-Za-z0-9_.-]+)$`)

// KeyFromURL extracts the storage key from a URL produced by
// PublicMediaURL, "" for an external URL.
func KeyFromURL(url string) string {
	m := keyFromURLRe.FindStringSubmatch(url)
	if m == nil {
		return ""
	}
	return m[1]
}

// KindForMime is VIDEO for video/*, IMAGE otherwise.
func KindForMime(contentType string) MediaKind {
	if strings.HasPrefix(contentType, "video/") {
		return MediaVideo
	}
	return MediaImage
}

// ExternalFilename is the last path segment of an https URL, URI-decoded,
// capped at 200 characters, "media" when empty.
func ExternalFilename(rawPath string) string {
	name := path.Base(rawPath)
	if name == "/" || name == "." || name == "" {
		name = "media"
	}
	return TruncateRunes(name, 200)
}

// ExternalMime derives a content type from kind and extension the way
// actions-media.ts did; the remote file is never fetched.
func ExternalMime(kind MediaKind, filename string) string {
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(filename), "."))
	if kind == MediaVideo {
		if ext == "mov" {
			return "video/quicktime"
		}
		return "video/mp4"
	}
	switch ext {
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	case "gif":
		return "image/gif"
	}
	return "image/jpeg"
}

// TruncateRunes is slice(0, n) on code points.
func TruncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
