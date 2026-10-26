// Package storage keeps reimbursement receipts on local disk
// (FINANCE_UPLOAD_DIR, default ./.uploads/finance), served back through the
// internal GetReceiptFile RPC. Another backend (S3, R2) implements app.Storage.
package storage

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
)

var ErrInvalidKey = errors.New("INVALID_KEY")

type Local struct{ Root string }

func NewLocal(root string) *Local {
	if root == "" {
		root = filepath.Join(".uploads", "finance")
	}
	return &Local{Root: root}
}

// path resolves a key below Root and refuses anything that escapes it.
func (l *Local) path(key string) (string, error) {
	if !domain.IsSafeKey(key) {
		return "", ErrInvalidKey
	}
	root, err := filepath.Abs(l.Root)
	if err != nil {
		return "", err
	}
	abs := filepath.Join(root, filepath.FromSlash(key))
	if !strings.HasPrefix(abs, root+string(filepath.Separator)) {
		return "", ErrInvalidKey
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

// Get returns nil data for a missing object.
func (l *Local) Get(_ context.Context, key string) ([]byte, string, error) {
	abs, err := l.path(key)
	if err != nil {
		return nil, "", nil
	}
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		return nil, "", nil
	}
	body, err := os.ReadFile(abs)
	if err != nil {
		return nil, "", err
	}
	sum := sha1.Sum([]byte(fmt.Sprintf("%s-%d-%d", key, info.Size(), info.ModTime().UnixMilli())))
	return body, hex.EncodeToString(sum[:]), nil
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
