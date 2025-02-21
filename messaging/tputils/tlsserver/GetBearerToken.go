package tlsserver

import (
	"fmt"
	"net/http"
	"strings"
)

// GetBearerToken returns the bearer token from the HTTP request authorization header
// Returns an error if no token present or token isn't a bearer token
func GetBearerToken(req *http.Request) (string, error) {
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("tlsserver: no Authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 {
		return "", fmt.Errorf("tlsserver: invalid Authorization header")
	}
	authType := strings.ToLower(parts[0])
	authTokenString := parts[1]
	if authType != "bearer" {
		return "", fmt.Errorf("tlsserver: not a bearer token")
	}
	return authTokenString, nil
}
