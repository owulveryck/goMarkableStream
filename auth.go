package main

import (
	"fmt"
	"net/http"

	"github.com/owulveryck/goMarkableStream/internal/remarkable"
)

// BasicAuthMiddleware is a middleware function that adds basic authentication to a handler
func BasicAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		skipAuth := remarkable.HostsIP(r.RemoteAddr)
		// Check if the request is authenticated
		user, pass, ok := r.BasicAuth()
		if !skipAuth && (!ok || !checkCredentials(user, pass)) {
			// Authentication failed, send a 401 Unauthorized response
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintln(w, "Unauthorized")
			return
		}

		// Authentication succeeded, call the next handler
		next.ServeHTTP(w, r)
	})
}

// checkCredentials is a dummy function to validate the username and password
func checkCredentials(username, password string) bool {
	// Add your custom logic here to validate the credentials against your storage (e.g., database, file)
	// This is a basic example, so we're using hard-coded credentials for demonstration purposes.
	return username == c.Username && password == c.Password
}
