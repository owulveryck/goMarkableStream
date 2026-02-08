package main

import (
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

		// Check for Bearer token in Authorization header
		authHeader := r.Header.Get("Authorization")
		if token, found := strings.CutPrefix(authHeader, "Bearer "); found {
			if jwtMgr != nil {
				if _, err := jwtMgr.ValidateToken(token); err == nil {
					next.ServeHTTP(w, r)
					return
				}
			}
		}

		// Check for token query parameter (for SSE endpoints)
		tokenParam := r.URL.Query().Get("token")
		if tokenParam != "" && jwtMgr != nil {
			if _, err := jwtMgr.ValidateToken(tokenParam); err == nil {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Authentication failed
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, "Unauthorized")
	})
}

// checkCredentials validates the username and password against configuration.
// Used by the /login endpoint.
func checkCredentials(username, password string) bool {
	return username == c.Username && password == c.Password
}
