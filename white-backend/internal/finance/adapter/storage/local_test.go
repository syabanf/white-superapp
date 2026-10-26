package storage

import (
	"context"
	"testing"
)

func TestLocalRoundTripAndKeySafety(t *testing.T) {
	ctx := context.Background()
	l := NewLocal(t.TempDir())
	if err := l.Put(ctx, "reimbursements/a.png", []byte("x")); err != nil {
		t.Fatal(err)
	}
	data, etag, err := l.Get(ctx, "reimbursements/a.png")
	if err != nil || string(data) != "x" || etag == "" {
		t.Fatalf("%q %q %v", data, etag, err)
	}
	if err := l.Put(ctx, "../escape.png", []byte("x")); err != ErrInvalidKey {
		t.Fatalf("escape: %v", err)
	}
	if err := l.Delete(ctx, "reimbursements/a.png"); err != nil {
		t.Fatal(err)
	}
	if data, _, _ := l.Get(ctx, "reimbursements/a.png"); data != nil {
		t.Fatal("deleted")
	}
	if err := l.Delete(ctx, "reimbursements/missing.png"); err != nil {
		t.Fatal("missing file is fine")
	}
}
