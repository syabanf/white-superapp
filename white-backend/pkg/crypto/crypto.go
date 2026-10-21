// Package crypto encrypts secrets at rest (OAuth tokens, integration keys)
// with AES-256-GCM. The wire format matches the Next.js app so rows written
// by either side decrypt on both: base64(iv[12] || tag[16] || ciphertext).
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
)

const (
	ivLen  = 12
	tagLen = 16
)

type Cipher struct {
	key []byte
}

// New derives the key the way lib/crypto.ts did: a base64 value that decodes
// to exactly 32 bytes is used as-is, anything else is SHA-256 hashed.
func New(encryptionKey string) (*Cipher, error) {
	if encryptionKey == "" {
		return nil, errors.New("ENCRYPTION_KEY belum diset")
	}
	if raw, err := base64.StdEncoding.DecodeString(encryptionKey); err == nil && len(raw) == 32 {
		return &Cipher{key: raw}, nil
	}
	sum := sha256.Sum256([]byte(encryptionKey))
	return &Cipher{key: sum[:]}, nil
}

func (c *Cipher) gcm() (cipher.AEAD, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func (c *Cipher) Encrypt(plain string) (string, error) {
	g, err := c.gcm()
	if err != nil {
		return "", err
	}
	iv := make([]byte, ivLen)
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}
	// Go appends the tag after the ciphertext; the TS format puts it before.
	sealed := g.Seal(nil, iv, []byte(plain), nil)
	data, tag := sealed[:len(sealed)-tagLen], sealed[len(sealed)-tagLen:]
	out := make([]byte, 0, ivLen+tagLen+len(data))
	out = append(out, iv...)
	out = append(out, tag...)
	out = append(out, data...)
	return base64.StdEncoding.EncodeToString(out), nil
}

func (c *Cipher) Decrypt(payload string) (string, error) {
	buf, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("ciphertext bukan base64: %w", err)
	}
	if len(buf) < ivLen+tagLen {
		return "", errors.New("ciphertext terlalu pendek")
	}
	g, err := c.gcm()
	if err != nil {
		return "", err
	}
	iv, tag, data := buf[:ivLen], buf[ivLen:ivLen+tagLen], buf[ivLen+tagLen:]
	sealed := make([]byte, 0, len(data)+tagLen)
	sealed = append(sealed, data...)
	sealed = append(sealed, tag...)
	plain, err := g.Open(nil, iv, sealed, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
