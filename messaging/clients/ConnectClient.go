package clients

import (
	"crypto/x509"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/clients/authenticator"
	"github.com/hiveot/hub/messaging/clients/discovery"
	"github.com/hiveot/hub/messaging/clients/httpsseclient"
	"github.com/hiveot/hub/messaging/clients/wssclient"
	"github.com/hiveot/hub/messaging/servers/discoserver"
	"github.com/hiveot/hub/messaging/servers/hiveotsseserver"
	"github.com/hiveot/hub/messaging/servers/wssserver"
)

// TokenFileExt defines the filename extension under which client tokens are stored
// in the keys directory.
const TokenFileExt = ".token"

var DefaultTimeout = time.Second * 3

// ConnectWithPassword is a convenience function to authenticate with a password
// and create a secured client connection using a CA from the directory.
//
// This discovers the server if no URL is provided or the url is not able to authenticate.
//
//	loginID is the clientID to login as
//	password to authenticate with
//	caCert is the server CA. See also LoadCA()
//	connectURL of the server to connect to, or "" to use discovery
//	authURL of the authentication server, or "" to use connectURL
//
// This returns the client connection with the authentication token used or an error if invalid
func ConnectWithPassword(
	loginID string, password string, caCert *x509.Certificate, connectURL string, authURL string, timeout time.Duration) (
	cc messaging.IClientConnection, token string, err error) {

	// 1. obtain the CA public cert to verify the server
	//caCert, err := LoadCA(caDir)
	//if err != nil {
	//	return nil, "", err
	//}

	// 2. discover the server
	if connectURL == "" {
		var records []*discovery.DiscoveryResult

		records, err = discovery.DiscoverWithDnsSD(
			discoserver.DefaultServiceName, DefaultTimeout, true)
		if err == nil && len(records) > 0 {
			rec0 := records[0]
			// attempt hiveot websocket
			connectURL = rec0.WSSEndpoint
			if connectURL == "" {
				// fall back to SSE
				connectURL = rec0.SSEEndpoint
			}
			if authURL == "" {
				authURL = rec0.AuthEndpoint
			}
		}
	}
	if authURL == "" {
		authURL = connectURL
	}

	if err != nil {
		return nil, "", err
	} else if connectURL == "" {
		return nil, "", fmt.Errorf("WoT Server discovery failed")
	}

	// 3. authenticate. A cid is required to link the authenticated session with
	// the connections using it.
	parts, _ := url.Parse(authURL)
	authCl := authenticator.NewAuthClient(parts.Host, caCert, "auth-cid", timeout)
	//cl := tlsclient.NewTLSClient(parts.Host, nil, caCert, timeout)
	//token, err = authenticator.LoginWithPassword(
	//	cl, "", loginID, password)
	token, err = authCl.LoginWithPassword(loginID, password)
	if err != nil {
		return nil, "", err
	}

	// 4. create protocol client and connect with token
	cc, err = ConnectWithToken(loginID, token, caCert, connectURL, timeout)
	return cc, token, err
}

// ConnectWithToken is a convenience function to create a client and connect with a token
// and CA certificate.
//
//	clientID to identify as
//	token to authenticate with
//	caCert of the server for validating the SSL connect
//	connectURL is the optional URL of the server. Leave empty to auto-discover
//	timeout is optional connection timeout or 0 for default
//
// This returns the client connection or an error if invalid
func ConnectWithToken(
	clientID string, token string, caCert *x509.Certificate, connectURL string, timeout time.Duration) (
	cc messaging.IClientConnection, err error) {

	cc, err = NewHiveotClient(clientID, caCert, connectURL, timeout)
	if err == nil {
		err = cc.ConnectWithToken(token)
	}
	return cc, err
}

// ConnectWithTokenFile is a convenience function to create a connection using
// a saved token and optional CA file.
// This is similar to ConnectWithToken but reads a token and CA from file.
//
//	clientID to identify as
//	keysDir is the directory with the {clientID}.key, {clientID}.token and caCert.pem files.
//	connectURL is the full connection URL of the server to connect to.
//	timeout is optional connection timeout or 0 for default
//
// This returns the connection, token and CaCert used or an error if invalid
func ConnectWithTokenFile(clientID string, keysDir string, connectURL string, timeout time.Duration) (
	cc messaging.IClientConnection, token string, caCert *x509.Certificate, err error) {

	caCert, err = LoadCA(keysDir)
	if err == nil {
		token, err = LoadToken(clientID, keysDir)
	}
	if err == nil {
		cc, err = ConnectWithToken(clientID, token, caCert, connectURL, timeout)
	}
	return cc, token, caCert, err
}

// GetProtocolFromURL determines which transport protocol type the URL represents
// This returns an empty string if the protocol cannot be determined
//
// fullURL contains the full server address as provided by discovery:
//
// WARNING: hiveot can use a different protocol for the sse and wss schemas.
// In case of doubt the hiveot protocol is used.
//
//	https://addr:port/ for http-basic
//	sse://addr:port/hiveot/sse for hiveot sse-sc subprotocol binding
//	sse://addr:port/... default WoT SSE (not supported)
//	wss://addr:port/wss  path for websocket direct messaging
//	mqtts://addr:port/ for mqtt over websocket over TLS
func GetProtocolFromURL(fullURL string) string {
	// determine the protocol to use from the URL
	protocolType := ""

	parts, err := url.Parse(fullURL)
	if err != nil {
		return ""
	}

	if parts.Scheme == "" {
		// without a schema pick the basic
		protocolType = messaging.ProtocolTypeWotHTTPBasic
	} else if parts.Scheme == "https" {
		protocolType = messaging.ProtocolTypeWotHTTPBasic
	} else if parts.Scheme == wssserver.WssSchema {
		// websocket protocol can use either WoT or hiveot message envelopes
		protocolType = messaging.ProtocolTypeWSS
	} else if parts.Scheme == hiveotsseserver.HiveotSSESchema {
		// wot SSE is not supported
		protocolType = messaging.ProtocolTypeHiveotSSE
	} else if strings.HasPrefix(fullURL, "mqtts") {
		protocolType = messaging.ProtocolTypeWotMQTTWSS
	} else {
		// fall back to use https basic
		protocolType = messaging.ProtocolTypeWotHTTPBasic
	}
	return protocolType
}

// GetProtocolFromForm determine the protocol type from a WoT form.
// FIXME: forms can contain relative paths instead of full URL. The TD is needed
//
//	base is the TD base URI used for all relative URI references.
//	form is the form whose (sub)protocol to determine.
//func GetProtocolFromForm(base string, form *td.Form) string {
//	subProto, _ := form.GetSubprotocol()
//	if subProto != "" {
//		return subProto
//	}
//	href, _ := form.GetHRef()
//	return GetProtocolFromURL(href)
//}

// LoadCA is a simple helper to load the default CA from file
func LoadCA(caDir string) (*x509.Certificate, error) {
	caCertFile := path.Join(caDir, certs.DefaultCaCertFile)
	caCert, err := certs.LoadX509CertFromPEM(caCertFile)
	return caCert, err
}

// LoadToken is a simple helper to load a saved auth token from file
func LoadToken(clientID string, keysDir string) (string, error) {
	//keyFile := path.Join(keysDir, clientID+keys.KPFileExt)
	tokenFile := path.Join(keysDir, clientID+TokenFileExt)
	token, err := os.ReadFile(tokenFile)
	return string(token), err
}

// NewHiveotClient returns a new unconnected client ready for connecting to a
// hiveot server.
// Intended for use with the hiveot hub, but should be usable with other hiveot servers.
//
// Note 1: This doesnt use forms as request/response envelopes dont need it.
//
//	clientID is the ID to authenticate as when using one of the Connect... methods
//	caCert is the server's CA certificate to verify the connection.
//	connectURL contains the connection URL for the protocol.
//	timeout is optional maximum wait time for connecting or waiting for responses.
//
// If no connectURL is specified then this uses discovery to determine the best protocol.
func NewHiveotClient(
	clientID string, caCert *x509.Certificate, connectURL string, timeout time.Duration) (
	cc messaging.IClientConnection, err error) {

	// 1. determine the connection address
	if connectURL == "" {
		// use the first hiveot instance to connect to
		discoList, err := discovery.DiscoverWithDnsSD(
			discoserver.DefaultServiceName, timeout, true)
		if err != nil || len(discoList) == 0 {
			return nil, fmt.Errorf("hub not found")
		}
		connectURL = discoList[0].WSSEndpoint
		if connectURL == "" {
			connectURL = discoList[0].SSEEndpoint
		}
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	parts, err := url.Parse(connectURL)
	if err != nil {
		return nil, err
	}

	// Create the client for the protocol
	if parts.Scheme == "wss" {
		msgConverter := &wssserver.HiveotMessageConverter{}
		cc = wssclient.NewHiveotWssClient(connectURL, clientID, caCert,
			msgConverter, timeout)
	} else if parts.Scheme == "sse" {
		cc = httpsseclient.NewHiveotSseClient(
			connectURL, clientID, nil, caCert, nil, timeout)
	} else if parts.Scheme == "mqtts" {
		//	bc = mqttclient.NewMqttAgentClient(
		//		fullURL, clientID, nil, caCert, timeout)
		err = fmt.Errorf("NewHiveotClient: mqtt client is not yet supported for '%s'", connectURL)
	} else {
		err = fmt.Errorf("NewHiveotClient: Server URL '%s' does not have a valid schema (wss://.. or sse://...", connectURL)
	}
	return cc, err
}
