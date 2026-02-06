package authclient

import (
	"crypto/x509"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/hiveot/hub/lib/clients/tlsclient"
	"github.com/hiveot/hub/lib/messaging"
	"github.com/hiveot/hub/lib/servers/httpbasic"
	jsoniter "github.com/json-iterator/go"
)

// AuthClient is a client for handling authentication using http endpoints
// This creates a http TLS client.
type AuthClient struct {
	tlsClient *tlsclient.TLSClient
}

// LoginWithPassword invokes the remote password authenticator.
//
// authURL is the http address to invoke. If a path is omitted, the default path
// (HttpPostLoginPath) defined in the http server will be used.
//
// This creates a temporary TLS client identified with the loginID and cid.
// The resulting authentication is linked to an internal session.
//
//	tlsClient is an unauthenticated client initialized with a URL, CA and cid
//	loginPath the path to post the login request. Default is HttpPostLoginPath
//	loginID to login as
//	password to use with digest auth
//
// This returns an authentication token for connecting with any of the protocols, or an error
func (cl *AuthClient) LoginWithPassword(loginID string, password string) (newToken string, err error) {

	// FIXME: support for http digest auth?
	loginMessage := map[string]string{
		"login":    loginID,
		"password": password,
	}

	loginPath := httpbasic.HttpPostLoginPath

	dataJSON, _ := jsoniter.Marshal(loginMessage)
	outputRaw, status, err := cl.tlsClient.Post(loginPath, dataJSON)

	if err == nil && status == http.StatusOK {
		err = jsoniter.Unmarshal(outputRaw, &newToken)
	}
	if err != nil {
		slog.Warn("AuthenticateWithPassword failed: " + err.Error())
	}
	return newToken, err
}

// Logout requests invalidating all client sessions.
// tlsClient is a client with an existing authenticated connection
// logoutPath is the http address to invoke. If a path is omitted, the default path
func (cl *AuthClient) Logout() (err error) {
	logoutPath := httpbasic.HttpPostLogoutPath

	_, _, err = cl.tlsClient.Post(logoutPath, nil)
	cl.tlsClient.Close()
	return err
}

// RefreshToken invokes the hub's authenticator to refresh the token.
//
// tlsClient is a client with an existing authenticated connection
// refreshPath is the http address to invoke. If a path is omitted, the default path
// (HttpPostRefreshPath) defined in the http server will be used.
//
// This returns a new authentication token, or an error
func (cl *AuthClient) RefreshToken(oldToken string) (newToken string, err error) {

	refreshPath := httpbasic.HttpPostRefreshPath
	dataJSON, _ := jsoniter.Marshal(oldToken)
	cl.tlsClient.SetAuthToken(oldToken)
	outputRaw, status, err := cl.tlsClient.Post(refreshPath, dataJSON)

	if err == nil && status == http.StatusOK {
		err = jsoniter.Unmarshal(outputRaw, &newToken)
	}

	if err != nil {
		slog.Warn("RefreshToken failed: " + err.Error())
	} else {
		cl.tlsClient.SetAuthToken(newToken)
	}
	return newToken, err
}

// Close the client connection if connected
func (cl *AuthClient) Close() {
	if cl.tlsClient != nil {
		cl.tlsClient.Close()
	}
}

// NewAuthClient for authentication and token refresh
//
//	hostPort is the address of the authentication server
//	caCert is the CA of the server
//	timeout of requests. 0 for default.
func NewAuthClient(hostPort string, caCert *x509.Certificate, timeout time.Duration) *AuthClient {

	tlsClient := tlsclient.NewTLSClient(hostPort, nil, caCert, timeout)
	ac := &AuthClient{
		tlsClient: tlsClient,
	}
	return ac
}

// NewAuthClientFromConnection for authentication using an exising connection
// the auth server must be reachable on the same address.
// bearerToken as used by the client connection.
func NewAuthClientFromConnection(cl messaging.IClientConnection, bearerToken string) *AuthClient {

	cinfo := cl.GetConnectionInfo()
	parts, _ := url.Parse(cinfo.ConnectURL)
	tlsClient := tlsclient.NewTLSClient(parts.Host, nil, cinfo.CaCert, cinfo.Timeout)
	tlsClient.SetAuthToken(bearerToken)
	ac := &AuthClient{
		tlsClient: tlsClient,
	}
	return ac
}
