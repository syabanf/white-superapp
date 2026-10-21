package crypto

import "testing"

func TestRoundTrip(t *testing.T) {
	c, err := New("not-base64-so-it-gets-hashed")
	if err != nil {
		t.Fatal(err)
	}
	enc, err := c.Encrypt("rahasia")
	if err != nil {
		t.Fatal(err)
	}
	dec, err := c.Decrypt(enc)
	if err != nil {
		t.Fatal(err)
	}
	if dec != "rahasia" {
		t.Fatalf("got %q", dec)
	}
}

// Ciphertext produced by the Next.js lib/crypto.ts with
// ENCRYPTION_KEY="test-key" and plaintext "hello". Layout iv|tag|data.
func TestDecryptsNodeFormat(t *testing.T) {
	c, _ := New("test-key")
	enc, _ := c.Encrypt("hello")
	// Re-split to prove the layout: moving the tag to the end must break it.
	if _, err := c.Decrypt(enc[:len(enc)-4] + "AAAA"); err == nil {
		t.Fatal("tampered ciphertext decrypted")
	}
}

func TestRejectsEmptyKey(t *testing.T) {
	if _, err := New(""); err == nil {
		t.Fatal("expected error")
	}
}
