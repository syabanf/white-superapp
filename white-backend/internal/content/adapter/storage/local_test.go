package storage

import (
	"context"
	"testing"
)

func TestLocalRoundTripAndSafety(t *testing.T) {
	ctx := context.Background()
	l := NewLocal(t.TempDir())
	obj, err := l.Put(ctx, "client_a", "png", []byte("data"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := l.Get(ctx, obj.Key)
	if err != nil || string(got) != "data" || obj.SizeBytes != 4 {
		t.Fatalf("got %q err %v", got, err)
	}
	for _, bad := range []string{"../etc/passwd", "client_a/../../x", "a/b/c", ""} {
		if b, err := l.Get(ctx, bad); b != nil || err != nil {
			t.Fatalf("%q: %v %v", bad, b, err)
		}
	}
	if _, err := l.Put(ctx, "../evil", "png", []byte("x")); err == nil {
		t.Fatal("escaping client id accepted")
	}
	if err := l.DeleteClient(ctx, "client_a"); err != nil {
		t.Fatal(err)
	}
	if b, _ := l.Get(ctx, obj.Key); b != nil {
		t.Fatal("file survived client purge")
	}
}
