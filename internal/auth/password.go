package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Sha256Hex returns the lowercase-hex SHA-256 digest of s.
//
// This is the "password material" exchanged with the browser: the frontend
// hashes the user's plaintext password with SHA-256 before submitting it, so
// the plaintext never travels over the wire.
func Sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// HashPlaintext returns the password_hash for a plaintext password the server
// itself holds (e.g. read from server.yaml or the PANEL_PASSWORD env var): its
// SHA-256 digest. This is exactly the value the browser sends, so what is
// stored as password_hash is compared directly at login time — no re-hashing,
// which means existing password_hash values keep working.
func HashPlaintext(plaintext string) string {
	return Sha256Hex(plaintext)
}

// VerifyPassword reports whether received (the SHA-256 digest sent by the
// browser) matches the stored password_hash. It accepts:
//   - a direct SHA-256 password_hash (constant-time compare), the current
//     scheme, and
//   - a legacy bcrypt password_hash (bcrypt-verify), so values created by the
//     earlier bcrypt(sha256) scheme still authenticate.
func VerifyPassword(stored, received string) bool {
	if stored == "" || received == "" {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(stored), []byte(received)) == 1 {
		return true
	}
	if strings.HasPrefix(stored, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(received)) == nil
	}
	return false
}
