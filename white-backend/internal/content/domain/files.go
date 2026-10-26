package domain

import (
	"bytes"
	"path"
	"regexp"
	"strings"
)

// MaxUploadBytes caps one file; uploads travel as bytes inside JSON/gRPC.
const MaxUploadBytes = 25 * 1024 * 1024

type fileType struct {
	mime string
	ext  string
	// media: publishing accepts it as post media.
	media bool
}

var fileTypes = []fileType{
	{"image/jpeg", "jpg", true},
	{"image/png", "png", true},
	{"image/webp", "webp", true},
	{"image/gif", "gif", true},
	{"video/mp4", "mp4", true},
	{"video/quicktime", "mov", true},
	{"application/pdf", "pdf", false},
	{"image/svg+xml", "svg", false},
	{"font/ttf", "ttf", false},
	{"font/otf", "otf", false},
	{"font/woff", "woff", false},
	{"font/woff2", "woff2", false},
	{"application/zip", "zip", false},
}

// SniffFile reads the leading bytes and returns the content type and
// extension, or "" when the file is outside the allowlist. The filename
// is only a claim; the bytes decide.
func SniffFile(b []byte) (mime, ext string) {
	head := b
	if len(head) > 512 {
		head = head[:512]
	}
	switch {
	case len(b) >= 3 && b[0] == 0xff && b[1] == 0xd8 && b[2] == 0xff:
		return "image/jpeg", "jpg"
	case len(b) >= 8 && bytes.Equal(b[:8], []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}):
		return "image/png", "png"
	case len(b) >= 6 && (string(b[:6]) == "GIF87a" || string(b[:6]) == "GIF89a"):
		return "image/gif", "gif"
	case len(b) >= 12 && string(b[:4]) == "RIFF" && string(b[8:12]) == "WEBP":
		return "image/webp", "webp"
	case len(b) >= 12 && string(b[4:8]) == "ftyp":
		if strings.HasPrefix(string(b[8:12]), "qt") {
			return "video/quicktime", "mov"
		}
		return "video/mp4", "mp4"
	case len(b) >= 5 && string(b[:5]) == "%PDF-":
		return "application/pdf", "pdf"
	case len(b) >= 4 && string(b[:4]) == "wOFF":
		return "font/woff", "woff"
	case len(b) >= 4 && string(b[:4]) == "wOF2":
		return "font/woff2", "woff2"
	case len(b) >= 4 && string(b[:4]) == "OTTO":
		return "font/otf", "otf"
	case len(b) >= 4 && bytes.Equal(b[:4], []byte{0, 1, 0, 0}):
		return "font/ttf", "ttf"
	case len(b) >= 4 && bytes.Equal(b[:4], []byte{'P', 'K', 3, 4}):
		return "application/zip", "zip"
	case bytes.Contains(bytes.ToLower(head), []byte("<svg")):
		return "image/svg+xml", "svg"
	}
	return "", ""
}

// IsMedia reports whether publishing accepts the content type as post media.
func IsMedia(mime string) bool {
	for _, t := range fileTypes {
		if t.mime == mime {
			return t.media
		}
	}
	return false
}

// MimeForKey derives the content type from a stored key's extension.
func MimeForKey(key string) string {
	ext := strings.TrimPrefix(path.Ext(key), ".")
	for _, t := range fileTypes {
		if t.ext == ext {
			return t.mime
		}
	}
	return "application/octet-stream"
}

var safeKeyRe = regexp.MustCompile(`^[A-Za-z0-9_-]+/[A-Za-z0-9_.-]+$`)

// IsSafeKey allows only `<segment>/<segment>` in a whitelisted charset.
func IsSafeKey(key string) bool { return safeKeyRe.MatchString(key) }

// CleanFilename keeps the base name, at most 200 characters, "file" when empty.
func CleanFilename(name string) string {
	name = strings.TrimSpace(path.Base(strings.ReplaceAll(name, "\\", "/")))
	if name == "" || name == "." || name == "/" {
		name = "file"
	}
	r := []rune(name)
	if len(r) > 200 {
		name = string(r[:200])
	}
	return name
}
