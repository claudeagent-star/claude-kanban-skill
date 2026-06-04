package main

import (
	"crypto/subtle"
	"net/http"
)

// requireBasicAuth wraps next with HTTP Basic auth.
// If both user and pass are empty, auth is disabled and requests pass through.
// Comparison is constant-time to avoid timing attacks.
func requireBasicAuth(user, pass string, next http.Handler) http.Handler {
	if user == "" && pass == "" {
		return next
	}
	expectedUser := []byte(user)
	expectedPass := []byte(pass)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, ok := r.BasicAuth()
		if !ok {
			unauthorised(w)
			return
		}
		userOK := subtle.ConstantTimeCompare([]byte(gotUser), expectedUser) == 1
		passOK := subtle.ConstantTimeCompare([]byte(gotPass), expectedPass) == 1
		if !userOK || !passOK {
			unauthorised(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func unauthorised(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="kanban"`)
	http.Error(w, "unauthorised", http.StatusUnauthorized)
}
