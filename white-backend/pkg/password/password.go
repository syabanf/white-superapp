// Package password hashes user passwords and share PINs with bcrypt cost 10,
// the same parameters the Next.js app used, so existing hashes keep working.
package password

import "golang.org/x/crypto/bcrypt"

const Cost = 10

func Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), Cost)
	return string(b), err
}

func Verify(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
