package tlsserver

import (
	"crypto/ecdsa"
	"net/http"
)

// HttpAuthenticator chains the selected authenticators
type HttpAuthenticator struct {
	BasicAuth *BasicAuthenticator
	CertAuth  *CertAuthenticator
	JwtAuth   *JWTAuthenticator
}

// AuthenticateRequest
// Checks in order: client certificate, JWT bearer, Basic
// Returns the authenticated userID or an error if authentication failed
func (hauth *HttpAuthenticator) AuthenticateRequest(resp http.ResponseWriter, req *http.Request) (userID string, match bool) {
	if hauth.CertAuth != nil {
		// FIXME: how to differentiate between cert auth and other for authorization
		// workaround: plugin certificates do not have a name
		userID, match = hauth.CertAuth.AuthenticateRequest(resp, req)
		if match {
			return userID, match
		}
	}
	if hauth.JwtAuth != nil {
		userID, match = hauth.JwtAuth.AuthenticateRequest(resp, req)
		if match {
			return userID, match
		}
	}
	if hauth.BasicAuth != nil {
		userID, match = hauth.BasicAuth.AuthenticateRequest(resp, req)
		if match {
			return userID, match
		}
	}
	return userID, false
}

// GetClientOU returns the authorization OU of the requester's client certificate, if any.
// Returns OUNone if the request has no client certificate or the certificate has no OU
func (hauth *HttpAuthenticator) GetClientOU(request *http.Request) string {
	return hauth.CertAuth.GetClientOU(request)
}

// EnableBasicAuth enables BASIC authentication
// Basic auth is a legacy authentication scheme and not recommended as it requires each service to
// have access to the credentials store. Use of JwtAuth is preferred.
//
// validateCredentials is the function that verifies the given credentials
func (hauth *HttpAuthenticator) EnableBasicAuth(validateCredentials func(loginName string, password string) bool) {
	hauth.BasicAuth = NewBasicAuthenticator(validateCredentials)
}

// EnableJwtAuth enables JWT authentication using asymmetric keys
// JWT tokens are included in the request header authorization field and signed by an issuing authentication
// server using the server's private key. The provided verification key is the server's public key needed
// to verify that signature.
func (hauth *HttpAuthenticator) EnableJwtAuth(verificationKey *ecdsa.PublicKey) {
	hauth.JwtAuth = NewJWTAuthenticator(verificationKey)
}

// NewHttpAuthenticator creates a container to apply HTTP request authenticators
// By default the certificate authenticator is enabled. Additional authenticators can be enabled using the Enable... functions
//
// Use .AuthenticateRequest() to authenticate the incoming request
func NewHttpAuthenticator() *HttpAuthenticator {
	ha := &HttpAuthenticator{
		CertAuth: NewCertAuthenticator(),
	}
	return ha
}
