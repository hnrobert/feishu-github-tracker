package panel

import (
	"context"
	"net/http"
	"strings"

	"github.com/hnrobert/feishu-github-tracker/internal/auth"
)

type ctxKey string

const (
	ctxUsername ctxKey = "username"
)

// withAuthContext parses the session (cookie or Bearer token) and injects the
// username into the request context. Applied to the whole mux.
func (a *App) withAuthContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if username := a.readAuth(r); username != "" {
			ctx := context.WithValue(r.Context(), ctxUsername, username)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// readAuth returns the authenticated username from a cookie or Authorization
// header, or "" if absent/invalid.
func (a *App) readAuth(r *http.Request) string {
	if c, err := r.Cookie(a.cookieName); err == nil && c.Value != "" {
		if cl, err := auth.ParseHS256(a.secret, c.Value); err == nil {
			return cl.Username
		}
	}
	authz := r.Header.Get("Authorization")
	if authz != "" {
		parts := strings.SplitN(authz, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			if cl, err := auth.ParseHS256(a.secret, strings.TrimSpace(parts[1])); err == nil {
				return cl.Username
			}
		}
	}
	return ""
}

func usernameFrom(r *http.Request) string {
	if v := r.Context().Value(ctxUsername); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// requireAuth redirects unauthenticated requests to /login.
func (a *App) requireAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if usernameFrom(r) == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		h(w, r)
	}
}

func (a *App) issueCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     a.cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
		MaxAge:   int(sessionTTL.Seconds()),
	})
}

func (a *App) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     a.cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
		MaxAge:   -1,
	})
}
