package tlsserver

import (
	"net/http"
)

// BasicAuthenticator decodes the authentication method used in the request and authenticates the user
type BasicAuthenticator struct {
	// the password verification handler
	verifyUsernamePassword func(username, password string) bool
}

// AuthenticateRequest
// Checks in order: client certificate, JWT bearer, Basic
// Returns the authenticated userID or an error if authentication failed
func (bauth *BasicAuthenticator) AuthenticateRequest(resp http.ResponseWriter, req *http.Request) (userID string, match bool) {
	username, password, ok := req.BasicAuth()
	if !ok {
		return username, false
	}
	ok = bauth.verifyUsernamePassword(username, password)
	if !ok {
		return username, false
	}
	return username, true
}

// NewBasicAuthenticator creates a new HTTP Basic authenticator
//
//	verifyUsernamePassword is the handler that validates the loginID and secret
func NewBasicAuthenticator(verifyUsernamePassword func(loginID, secret string) bool) *BasicAuthenticator {
	ba := &BasicAuthenticator{
		verifyUsernamePassword: verifyUsernamePassword,
	}
	return ba
}
