package auth

import (
	"fmt"
	"net/http"
	"strings"
)

var ourDumbAuthToken = "security"

// BasicAuthMiddleware validates that the user passed a static token
//
// The purpose here is to protect against automated traffic without
// having to implement a full token management system while we try
// to get a full release out the door + deployed.
func BasicAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authToken := r.Header.Get("Authorization")

		if strings.Compare(authToken, ourDumbAuthToken) != 0 {
			fmt.Println("Token is not valid:", authToken)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
		} else {
			next.ServeHTTP(w, r)
		}
	})
}
