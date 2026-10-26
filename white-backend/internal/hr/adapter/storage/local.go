// Package storage keeps hr files (attendance selfies, employee documents)
// on local disk under HR_UPLOAD_DIR. Files are private: they are served
// only through the authorized hr RPCs, never by public URL.
package storage

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

type Local struct{ Root string }

func NewLocal(root string) *Local {
	if root == "" {
		root = ".uploads/hr"
	}
	return &Local{Root: root}
}

var extensions = map[string]string{"image/jpeg": "jpg", "image/png": "png", "image/webp": "webp", "application/pdf": "pdf"}

// Sniff reads the magic bytes of the types hr accepts ("" for others).
func Sniff(b []byte) string {
	switch {
	case len(b) >= 3 && b[0] == 0xff && b[1] == 0xd8 && b[2] == 0xff:
		return "image/jpeg"
	case len(b) >= 8 && bytes.Equal(b[:8], []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}):
		return "image/png"
	case len(b) >= 12 && string(b[:4]) == "RIFF" && string(b[8:12]) == "WEBP":
		return "image/webp"
	case len(b) >= 5 && string(b[:5]) == "%PDF-":
		return "application/pdf"
	}
	return ""
}

var keyRe = regexp.MustCompile(`^[a-z]+/[A-Za-z0-9-]+/[A-Za-z0-9-]+\.[a-z]+$`)

// path resolves a key below Root and refuses anything that escapes it.
func (l *Local) path(key string) (string, error) {
	if !keyRe.MatchString(key) {
		return "", os.ErrNotExist
	}
	root, err := filepath.Abs(l.Root)
	if err != nil {
		return "", err
	}
	abs := filepath.Join(root, filepath.FromSlash(key))
	if !strings.HasPrefix(abs, root+string(filepath.Separator)) {
		return "", os.ErrNotExist
	}
	return abs, nil
}

func (l *Local) Put(_ context.Context, prefix string, data []byte, allowed []string, maxBytes int) (string, string, error) {
	ct := Sniff(data)
	if ct == "" || !slices.Contains(allowed, ct) {
		return "", "", app.ErrFileType
	}
	if len(data) > maxBytes {
		return "", "", app.ErrFileLarge
	}
	key := prefix + "/" + ids.New() + "." + extensions[ct]
	abs, err := l.path(key)
	if err != nil {
		return "", "", app.ErrFileType
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(abs, data, 0o640); err != nil {
		return "", "", err
	}
	return key, ct, nil
}

func (l *Local) Get(_ context.Context, key string) ([]byte, string, error) {
	abs, err := l.path(key)
	if err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, "", err
	}
	return data, Sniff(data), nil
}

func (l *Local) Delete(_ context.Context, key string) error {
	abs, err := l.path(key)
	if err != nil {
		return nil
	}
	if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
