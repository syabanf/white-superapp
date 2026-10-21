package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/publishing/domain"
)

var pngBytes = []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0x0d, 'I', 'H', 'D', 'R'}

func TestLocalRoundTrip(t *testing.T) {
	ctx := context.Background()
	l := NewLocal(t.TempDir(), "https://app.white.id/")
	obj, err := l.Put(ctx, "client1", "image/png", pngBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(obj.Key, "client1/") || !strings.HasSuffix(obj.Key, ".png") || obj.URL != "https://app.white.id/api/media/"+obj.Key || obj.SizeBytes != int64(len(pngBytes)) {
		t.Fatalf("%+v", obj)
	}
	got, err := l.Get(ctx, obj.Key)
	if err != nil || got == nil || got.ContentType != "image/png" || len(got.ETag) != 40 || string(got.Body) != string(pngBytes) {
		t.Fatalf("%+v %v", got, err)
	}
	if missing, _ := l.Get(ctx, "client1/nope.png"); missing != nil {
		t.Fatal("missing file returned")
	}
	if bad, _ := l.Get(ctx, "../etc/passwd"); bad != nil {
		t.Fatal("traversal served")
	}
	if dir, _ := l.Get(ctx, "client1/.."); dir != nil {
		t.Fatal("parent directory served")
	}
	if err := l.Delete(ctx, obj.Key); err != nil {
		t.Fatal(err)
	}
	if err := l.Delete(ctx, obj.Key); err != nil {
		t.Fatal("second delete should be silent")
	}
}

func TestLocalRejects(t *testing.T) {
	ctx := context.Background()
	l := NewLocal(t.TempDir(), "")
	if _, err := l.Put(ctx, "c", "image/png", []byte("<svg/>")); err != domain.ErrUnsupportedType {
		t.Fatal(err)
	}
	if _, err := l.Put(ctx, "c", "image/jpeg", pngBytes); err != domain.ErrUnsupportedType {
		t.Fatal("mismatched declared type accepted")
	}
	if _, err := l.Put(ctx, "../c", "image/png", pngBytes); err != domain.ErrInvalidKey {
		t.Fatal(err)
	}
}

func TestDeleteClient(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	l := NewLocal(root, "")
	obj, err := l.Put(ctx, "c2", "image/png", pngBytes)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.DeleteClient(ctx, "c2"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, obj.Key)); !os.IsNotExist(err) {
		t.Fatal("file survived")
	}
	if err := l.DeleteClient(ctx, ".."); err != domain.ErrInvalidKey {
		t.Fatal("parent removal accepted")
	}
}
