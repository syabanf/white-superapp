package storage

import (
	"context"
	"testing"
)

func TestLocalRoundTripAndTraversal(t *testing.T) {
	l := NewLocal(t.TempDir())
	ctx := context.Background()
	if err := l.Put(ctx, "cv/c1/a.pdf", []byte("%PDF-1.7")); err != nil {
		t.Fatal(err)
	}
	data, err := l.Get(ctx, "cv/c1/a.pdf")
	if err != nil || string(data) != "%PDF-1.7" {
		t.Fatal(string(data), err)
	}
	if err := l.Delete(ctx, "cv/c1/a.pdf"); err != nil {
		t.Fatal(err)
	}
	if err := l.Delete(ctx, "cv/c1/a.pdf"); err != nil {
		t.Fatal("deleting a missing file is fine", err)
	}
	for _, key := range []string{"../../etc/passwd", "/etc/passwd", ""} {
		if _, err := l.Get(ctx, key); err == nil {
			t.Fatalf("%q must be refused", key)
		}
	}
}
