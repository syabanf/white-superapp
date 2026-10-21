// Package storage is the media object store. Local disk today (UPLOAD_DIR,
// default ./.uploads) served back through the gateway's GET /api/media/{key}.
// Swapping in S3 or R2 means another implementation of domain.Storage.
package storage

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/publishing/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

type Local struct {
	Root    string
	BaseURL string
}

func NewLocal(root, baseURL string) *Local {
	if root == "" {
		root = ".uploads"
	}
	return &Local{Root: root, BaseURL: baseURL}
}

// path resolves a key below Root and refuses anything that escapes it.
func (l *Local) path(key string) (string, error) {
	if !domain.IsSafeKey(key) {
		return "", domain.ErrInvalidKey
	}
	root, err := filepath.Abs(l.Root)
	if err != nil {
		return "", err
	}
	abs := filepath.Join(root, filepath.FromSlash(key))
	if !strings.HasPrefix(abs, root+string(filepath.Separator)) {
		return "", domain.ErrInvalidKey
	}
	return abs, nil
}

func (l *Local) Put(_ context.Context, clientID, contentType string, data []byte) (*domain.StoredObject, error) {
	ext := domain.ExtensionFor(contentType)
	if ext == "" || domain.SniffContentType(data) != contentType {
		return nil, domain.ErrUnsupportedType
	}
	if len(data) > domain.MaxUploadBytes {
		return nil, domain.ErrTooLarge
	}
	key := domain.MediaKey(clientID, ids.New(), ext)
	abs, err := l.path(key)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(abs, data, 0o644); err != nil {
		return nil, err
	}
	return &domain.StoredObject{Key: key, URL: domain.PublicMediaURL(l.BaseURL, key), SizeBytes: int64(len(data)), ContentType: contentType}, nil
}

func (l *Local) Get(_ context.Context, key string) (*domain.Object, error) {
	abs, err := l.path(key)
	if err != nil {
		return nil, nil
	}
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		return nil, nil
	}
	body, err := os.ReadFile(abs)
	if err != nil {
		return nil, nil
	}
	ext := strings.TrimPrefix(filepath.Ext(key), ".")
	sum := sha1.Sum([]byte(fmt.Sprintf("%d-%d", info.Size(), info.ModTime().UnixMilli())))
	return &domain.Object{Body: body, ContentType: domain.ContentTypeForExtension(ext), ETag: hex.EncodeToString(sum[:])}, nil
}

// Delete ignores a missing file.
func (l *Local) Delete(_ context.Context, key string) error {
	abs, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// DeleteClient removes the client's whole directory.
func (l *Local) DeleteClient(_ context.Context, clientID string) error {
	if !domain.IsSafeKey(clientID + "/x") {
		return domain.ErrInvalidKey
	}
	root, err := filepath.Abs(l.Root)
	if err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(root, clientID))
}
