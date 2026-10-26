package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/app"
)

var (
	png  = []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0x0d}
	jpeg = []byte{0xff, 0xd8, 0xff, 0xe0, 0, 0x10}
)

func TestPutGetChecksMagicBytes(t *testing.T) {
	l := NewLocal(t.TempDir())
	ctx := context.Background()
	images := []string{"image/jpeg", "image/png", "image/webp"}
	key, ct, err := l.Put(ctx, "attendance/emp-1", png, images, 1024)
	if err != nil || ct != "image/png" {
		t.Fatal(key, ct, err)
	}
	data, got, err := l.Get(ctx, key)
	if err != nil || got != "image/png" || len(data) != len(png) {
		t.Fatal(got, err)
	}
	if _, _, err := l.Put(ctx, "attendance/emp-1", []byte("<?php"), images, 1024); !errors.Is(err, app.ErrFileType) {
		t.Fatal("text must be refused", err)
	}
	if _, _, err := l.Put(ctx, "attendance/emp-1", []byte("%PDF-1.7"), images, 1024); !errors.Is(err, app.ErrFileType) {
		t.Fatal("pdf is not an image", err)
	}
	if _, _, err := l.Put(ctx, "attendance/emp-1", jpeg, images, 3); !errors.Is(err, app.ErrFileLarge) {
		t.Fatal("size limit", err)
	}
	if _, _, err := l.Get(ctx, "../../etc/passwd"); err == nil {
		t.Fatal("traversal must fail")
	}
}
