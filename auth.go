package main

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"

	"github.com/owulveryck/goMarkableStream/internal/jwtutil"
)

// isPublicPath returns true if the path should be accessible without authentication.
// This includes the login endpoint, main page, and static assets.
func isPublicPath(path string) bool {
	// Login endpoint must be public
	if path == "/login" {
		return true
	}

	// Main page serves the login modal
	if path == "/" {
		return true
	}

	// Static assets are public (JS, CSS, images, fonts, etc.)
	publicExtensions := []string{".js", ".css", ".ico", ".png", ".jpg", ".jpeg", ".svg", ".woff", ".woff2", ".ttf"}
	for _, ext := range publicExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}

	// Library files
	if strings.HasPrefix(path, "/lib/") {
		return true
	}

	return false
}

// authRoleKey is the context key under which AuthMiddleware stores the role
// of the authenticated caller.
type authRoleKey struct{}

// requestRole returns the role of the caller. Requests that did not go through
// AuthMiddleware (the -unsafe mode, where authentication is disabled on
// purpose) are treated as admin.
func requestRole(r *http.Request) string {
	if role, ok := r.Context().Value(authRoleKey{}).(string); ok {
		return role
	}
	return jwtutil.RoleAdmin
}

// authorizeToken validates a JWT and returns the role it grants.
// Funnel tokens are only honoured while the credentials they were issued for
// are still the active ones: disabling Funnel, or rotating its credentials,
// revokes every token handed out to temporary viewers.
func authorizeToken(jwtMgr *jwtutil.Manager, raw string) (string, bool) {
	if jwtMgr == nil || raw == "" {
		return "", false
	}
	token, err := jwtMgr.ValidateToken(raw)
	if err != nil {
		return "", false
	}
	switch token.Claims.Role {
	case jwtutil.RoleAdmin:
		return jwtutil.RoleAdmin, true
	case jwtutil.RoleFunnel:
		if funnelCreds != nil && funnelCreds.IsCurrent(token.Claims.FunnelGen) {
			return jwtutil.RoleFunnel, true
		}
	}
	// Tokens without a known role (including those issued before roles
	// existed, which could belong to Funnel viewers) are refused.
	return "", false
}

// AuthMiddleware validates JWT tokens for protected endpoints.
// Public paths (login, main page, static assets) are allowed without authentication.
// For SSE endpoints, also checks the token query parameter.
func AuthMiddleware(next http.Handler, jwtMgr *jwtutil.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow public paths without authentication
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Bearer token in Authorization header, or token query parameter
		// (EventSource and fetch-streaming cannot always set headers).
		raw, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		role, ok := authorizeToken(jwtMgr, raw)
		if !ok {
			role, ok = authorizeToken(jwtMgr, r.URL.Query().Get("token"))
		}
		if ok {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authRoleKey{}, role)))
			return
		}

		// Authentication failed
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, "Unauthorized")
	})
}

// defaultPassword is the out-of-the-box RK_SERVER_PASSWORD.
const defaultPassword = "password"

// usingDefaultCredentials reports whether the owner password is still the
// shipped default (or empty), which must never be reachable from the internet.
func usingDefaultCredentials(cfg *configuration) bool {
	return cfg.Password == "" || cfg.Password == defaultPassword
}

// checkCredentials validates the username and password against configuration
// and returns the role to grant. For Funnel credentials it also returns the
// credential generation to embed in the token.
// Used by the /login endpoint.
func checkCredentials(username, password string) (role string, funnelGen uint64, ok bool) {
	// Compare in constant time, and always evaluate both fields, so response
	// timing does not reveal how much of a guess was right.
	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(c.Username)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(password), []byte(c.Password)) == 1
	if userOK && passOK {
		return jwtutil.RoleAdmin, 0, true
	}
	if funnelCreds != nil {
		if gen, ok := funnelCreds.ValidateGen(username, password); ok {
			return jwtutil.RoleFunnel, gen, true
		}
	}
	return "", 0, false
}
