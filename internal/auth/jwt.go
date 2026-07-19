package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// DefaultCookieName is the name of the browser cookie holding the session JWT.
	DefaultCookieName = "fgt_panel_token"
	DefaultIssuer     = "feishu-github-tracker"
)

// Claims describes the authenticated admin session.
type Claims struct {
	Username string `json:"sub"`
	Admin    bool   `json:"admin"`
	jwt.RegisteredClaims
}

// NewRandomSecretB64 returns n random bytes encoded with raw-url base64, suitable
// for use as an ephemeral JWT signing secret.
func NewRandomSecretB64(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// SignHS256 issues an HS256 JWT for the given admin subject, valid for ttl.
func SignHS256(secret []byte, username string, admin bool, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		Username: username,
		Admin:    admin,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    DefaultIssuer,
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(secret)
}

// ParseHS256 validates an HS256 JWT and returns its claims. Allows 30s of
// clock skew to tolerate minor drift between issuer and verifier.
func ParseHS256(secret []byte, tokenString string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	}, jwt.WithLeeway(30*time.Second))
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
