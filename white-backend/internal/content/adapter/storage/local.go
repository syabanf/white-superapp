// Package storage keeps design files and brand assets on local disk
// (CONTENT_UPLOAD_DIR, default ./.uploads/content). Files are served only
// through the authorized download RPCs, never by public URL.
package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

type Local struct{ Root string }

func NewLocal(root string) *Local {
	if root == "" {
		root = filepath.Join(".uploads", "content")
	}
	return &Local{Root: root}
}

// path resolves a key below Root and refuses anything that escapes it.
func (l *Local) path(key string) (string, bool) {
	if !domain.IsSafeKey(key) {
		return "", false
	}
	root, err := filepath.Abs(l.Root)
	if err != nil {
		return "", false
	}
	abs := filepath.Join(root, filepath.FromSlash(key))
	return abs, strings.HasPrefix(abs, root+string(filepath.Separator))
}

func (l *Local) Put(_ context.Context, clientID, ext string, data []byte) (*domain.StoredFile, error) {
	key := clientID + "/" + ids.New() + "." + ext
	abs, ok := l.path(key)
	if !ok {
		return nil, os.ErrInvalid
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(abs, data, 0o644); err != nil {
		return nil, err
	}
	return &domain.StoredFile{Key: key, SizeBytes: int64(len(data))}, nil
}

// Get returns nil for a missing or invalid key.
func (l *Local) Get(_ context.Context, key string) ([]byte, error) {
	abs, ok := l.path(key)
	if !ok {
		return nil, nil
	}
	data, err := os.ReadFile(abs)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

// DeleteClient removes the client's whole directory.
func (l *Local) DeleteClient(_ context.Context, clientID string) error {
	abs, ok := l.path(clientID + "/x")
	if !ok {
		return os.ErrInvalid
	}
	return os.RemoveAll(filepath.Dir(abs))
}
