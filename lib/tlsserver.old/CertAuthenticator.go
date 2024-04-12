package tlsserver_old

import (
	"github.com/hiveot/hub/lib/certs"
	"net/http"
)

// CertAuthenticator verifies the client certificate authentication is used
// This simply checks if a client certificate is active and assumes that having one is sufficient to pass auth
type CertAuthenticator struct {
}

// AuthenticateRequest
// The real check happens by the TLS server that verifies it is signed by the CA.
// If the certificate is a plugin, then no userID is returned
// Returns the userID of the certificate (CN) or an error if no client certificate is used
func (hauth *CertAuthenticator) AuthenticateRequest(resp http.ResponseWriter, req *http.Request) (userID string, ok bool) {
	if len(req.TLS.PeerCertificates) == 0 {
		return "", false
	}
	cert := req.TLS.PeerCertificates[0]
	userID = cert.Subject.CommonName
	// a plugin is not a username
	if cert.Subject.CommonName == "plugin" {
		userID = ""
	}

	return userID, true
}

// GetClientOU returns the authorization OU of the client certificate, if any.
// Returns OUNone if the request has no client certificate or the certificate has no OU
// client certificate.
func (hauth *CertAuthenticator) GetClientOU(request *http.Request) (certOU string) {
	certOU = certs.OUNone
	if len(request.TLS.PeerCertificates) > 0 {
		cert := request.TLS.PeerCertificates[0]
		if len(cert.Subject.OrganizationalUnit) > 0 {
			certOU = cert.Subject.OrganizationalUnit[0]
		}
	}
	return certOU
}

// NewCertAuthenticator creates a new HTTP authenticator
// Use .AuthenticateRequest() to authenticate the incoming request
func NewCertAuthenticator() *CertAuthenticator {
	ca := &CertAuthenticator{}
	return ca
}
