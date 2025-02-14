package authenticator

import (
	"crypto/x509"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"github.com/hiveot/hub/transports/tputils/tlsclient"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// AuthenticateWithPassword invokes the hub's password authenticator.
//
// authURL is the http address to invoke. If a path is omitted, the default path
// (HttpPostLoginPath) defined in the http server will be used.
//
// This creates a temporary TLS client identified with the loginID and cid.
// The resulting authentication is linked to an internal session.
//
//	baseURL is the URL of the server to send the request to
//	loginPath the path to post the login request. Default is HttpPostLoginPath
//	loginID to login as
//	password to use with digest auth
//	caCert of the server, if known
//	cid connectionID to differentiate between client instances
//
// This returns an authentication token for connecting with any of the protocols, or an error
func AuthenticateWithPassword(baseURL string, loginPath string,
	loginID string, password string, caCert *x509.Certificate, cid string) (newToken string, err error) {

	// FIXME: use digest auth
	loginMessage := map[string]string{
		"login":    loginID,
		"password": password,
	}
	parts, _ := url.Parse(baseURL)

	if loginPath == "" {
		loginPath = httpserver.HttpPostLoginPath
	}
	// use a sacrificial client
	cl := tlsclient.NewTLSClient(parts.Host, nil, caCert, time.Second*3)
	cl.SetHeader(httpserver.ConnectionIDHeader, cid)
	dataJSON, _ := jsoniter.Marshal(loginMessage)
	outputRaw, status, err := cl.Post(loginPath, dataJSON)

	if err == nil && status == http.StatusOK {
		err = jsoniter.Unmarshal(outputRaw, &newToken)
	}
	cl.Close()

	if err != nil {
		slog.Warn("AuthenticateWithPassword failed: " + err.Error())
	}
	return newToken, err
}
