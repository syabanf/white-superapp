// Package storage keeps CVs and psikotes drawings on local disk (RECRUITMENT_UPLOAD_DIR).
// Swapping in S3 or R2 means another implementation of app.FileStore.
package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var errInvalidKey = errors.New("kunci file tidak valid")

type Local struct{ Root string }

func NewLocal(root string) *Local {
	if root == "" {
		root = ".uploads/recruitment"
	}
	return &Local{Root: root}
}

// path resolves a key below Root and refuses anything that escapes it.
func (l *Local) path(key string) (string, error) {
	root, err := filepath.Abs(l.Root)
	if err != nil {
		return "", err
	}
	if key == "" || strings.Contains(key, "..") || filepath.IsAbs(key) {
		return "", errInvalidKey
	}
	abs := filepath.Join(root, filepath.FromSlash(key))
	if !strings.HasPrefix(abs, root+string(filepath.Separator)) {
		return "", errInvalidKey
	}
	return abs, nil
}

func (l *Local) Put(_ context.Context, key string, data []byte) error {
	abs, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	return os.WriteFile(abs, data, 0o644)
}

func (l *Local) Get(_ context.Context, key string) ([]byte, error) {
	abs, err := l.path(key)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(abs)
}

func (l *Local) Delete(_ context.Context, key string) error {
	abs, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(abs); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
