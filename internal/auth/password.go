package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword returns a bcrypt hash of the given plaintext password.
func HashPassword(plain string) (string, error) {
	if plain == "" {
		return "", errors.New("password must not be empty")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// VerifyPassword reports whether plain matches the given bcrypt hash.
func VerifyPassword(hash, plain string) bool {
	if hash == "" || plain == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
