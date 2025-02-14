package clients

import (
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/clients/authenticator"
	"github.com/hiveot/hub/transports/clients/discovery"
	"github.com/hiveot/hub/transports/clients/httpsseclient"
	"github.com/hiveot/hub/transports/clients/wssclient"
	"github.com/hiveot/hub/transports/servers/discoserver"
	"github.com/hiveot/hub/transports/servers/hiveotsseserver"
	"github.com/hiveot/hub/transports/servers/wssserver"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
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
	cc transports.IClientConnection, token string, err error) {

	// 1. obtain the CA public cert to verify the server
	//caCert, err := LoadCA(caDir)
	//if err != nil {
	//	return nil, "", err
	//}

	// 2. discover the server
	if connectURL == "" {
		var records []*discovery.DiscoveryResult

		records, err = discovery.DiscoverWithDnsSD(
			"", discoserver.DefaultServiceName, DefaultTimeout, true)
		if err == nil && len(records) > 0 {
			rec0 := records[0]
			connectURL = rec0.ConnectURL
			if connectURL == "" {
				connectURL = rec0.TD
			}
			if authURL == "" {
				authURL = rec0.AuthURL
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
	newToken, err := authenticator.AuthenticateWithPassword(
		authURL, "", loginID, password, caCert, "")
	if err != nil {
		return nil, "", err
	}

	// 4. connect with token
	cc, err = ConnectWithToken(loginID, token, caCert, connectURL, timeout)
	return cc, newToken, err
}

// ConnectWithToken is a convenience function to create a client and connect with a token
// and CA certificate.
//
//	connectURL is the optional URL of the server. Leave empty to auto-discover
//	clientID to identify
//
// This returns the client connection or an error if invalid
func ConnectWithToken(
	clientID string, token string, caCert *x509.Certificate, connectURL string, timeout time.Duration) (
	cc transports.IClientConnection, err error) {

	cc, err = NewClient(clientID, caCert, nil, connectURL, timeout)
	if err == nil {
		err = cc.ConnectWithToken(token)
	}
	return cc, err
}

// ConnectWithTokenFile is a convenience function to create a connection using
// a saved token and optional CA file.
// This is similar to ConnectWithToken but reads a token and CA from file.
//
// keysDir is the directory with the {clientID}.key, {clientID}.token and caCert.pem files.
//
// This returns the connection, token and CaCert used or an error if invalid
func ConnectWithTokenFile(clientID string, keysDir string, connectURL string, timeout time.Duration) (
	cc transports.IClientConnection, token string, caCert *x509.Certificate, err error) {

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
//	https://addr:port/ for http without sse
//	sse://addr:port/wot/sse for http with the sse subprotocol binding
//	sse://addr:port/hiveot/sse for http with the ssesc subprotocol binding
//	wss://addr:port/wot/wss for websocket over TLS
//	wss://addr:port/hiveot/wss for direct messaging websocket over TLS
//	mqtts://addr:port/ for mqtt over websocket over TLS
func GetProtocolFromURL(fullURL string) string {
	// determine the protocol to use from the URL
	protocolType := ""

	parts, err := url.Parse(fullURL)
	if err != nil {
		return ""
	}
	if parts.Scheme == "https" {
		protocolType = transports.ProtocolTypeWotHTTPBasic
	} else if parts.Scheme == wssserver.HiveotWssSchema {
		// websocket protocol can use either WoT or hiveot message envelopes
		protocolType = transports.ProtocolTypeWotWSS
		if strings.HasSuffix(fullURL, wssserver.DefaultHiveotWssPath) {
			protocolType = transports.ProtocolTypeHiveotWSS
		}
	} else if parts.Scheme == hiveotsseserver.HiveotSSESchema {
		protocolType = transports.ProtocolTypeHiveotSSE
		// wot SSE is not supported
	} else {
		protocolType = transports.ProtocolTypeWotWSS
	}
	// there are 2 wss protocols, differentiate using the path
	if strings.HasSuffix(fullURL, wssserver.DefaultHiveotWssPath) {
		protocolType = transports.ProtocolTypeHiveotWSS
	} else if strings.HasPrefix(fullURL, "mqtts") {
		protocolType = transports.ProtocolTypeWotMQTTWSS
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

// NewClient returns a new unconnected client ready for connecting to a thing server.
// Intended for use with the hiveot hub, but should be usable with other things.
//
// This expects the thing level base URL and uses the schema to determine what
// protocol to use. Supported schema's:
// * 'https'  - plain https no async return channel
// * 'wss'    - secure websocket connection.
// * 'sse'    - secure http/hiveot SSE connection. Not an IANA schema.
// * 'mqtts'  - for mqtt over tcp  (future)
//
// Note 1: individual thing affordances can specify a different protocol in its form.
// This is currently not supported.
// Note 2: wss does not support https request other than establishing a connection
// Note 3: ssesc is hiveot only. WoT SSE has too many restrictions. Will likely be phased out in the future.
// Note 4: Use the separate auth method to get a token
//
//	clientID is the ID to authenticate as when using one of the Connect... methods
//	connectURL contains the connection URL for the protocol. Typically the TD baseURL.
//
// caCert is the server's CA certificate to verify the connection.
//
//	If caCert is not provided then the server connection is not verified
//
// getForm handler to return a Form when invoking a Thing request.
//
//	This is intended to be provided by a directory which can retrieve the forms of
//	a Thing for issuing requests. HiveOT presents all Things as a digital twin.
//	The forms of all Things are identical and use URI variables to fill in the
//	operation, thingID and name of the affordance.
//
// The WoT basic sse protocol is not supported as its use-case is too limited.
// Instead the hiveot SSE-Single-Connection protocol is used which wraps the
// sse event payload in a RequestMessage or ResponseMessage envelope. This supports
// messages for multiple Things and multiple affordances.
//
// If no connectURL is specified then this falls back to using the "baseURL" param
// in the discovery record of the hiveot instance to connect to.
//
// timeout is optional maximum wait time for connecting or waiting for responses.
// Use 0 for default.
func NewClient(
	clientID string, caCert *x509.Certificate,
	getForm transports.GetFormHandler, connectURL string, timeout time.Duration) (
	cc transports.IClientConnection, err error) {

	// 1. determine the connection address
	if connectURL == "" {

		// use the first hiveot instance to connect to
		discoList, err := discovery.DiscoverWithDnsSD(
			discoserver.DefaultInstanceName, discoserver.DefaultServiceName,
			timeout, true)
		if err != nil || len(discoList) == 0 {
			return nil, fmt.Errorf("hub not found")
		}
		connectURL = discoList[0].ConnectURL
	}

	// determine the protocol to use from the URL
	protocolType := GetProtocolFromURL(connectURL)
	if protocolType == "" {
		return nil, fmt.Errorf("Unknown protocol type in URL: " + connectURL)
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	// Create the client for the protocol
	switch protocolType {
	case transports.ProtocolTypeHiveotSSE:
		cc = httpsseclient.NewHiveotSseClient(
			connectURL, clientID, nil, caCert, getForm, timeout)

	case transports.ProtocolTypeHiveotWSS:
		msgConverter := &wssserver.HiveotMessageConverter{}
		cc = wssclient.NewHiveotWssClient(connectURL, clientID, caCert,
			msgConverter, transports.ProtocolTypeHiveotWSS, timeout)

	case transports.ProtocolTypeWotWSS:
		msgConverter := &wssserver.WotWssMessageConverter{}
		cc = wssclient.NewHiveotWssClient(connectURL, clientID, caCert,
			msgConverter, transports.ProtocolTypeWotWSS, timeout)

	case transports.ProtocolTypeWotHTTPBasic:
		panic("Don't use HTTPS protocol, use the SSESC or WSS subprotocol instead")
		//cc = httpbasicclient.NewHttpBasicClient(
		//	fullURL, clientID, getForm, caCert, getForm, timeout)

	case transports.ProtocolTypeWotMQTTWSS:
		//	//	bc = mqttclient.NewMqttAgentClient(
		//	//		fullURL, clientID, nil, caCert, timeout)
		panic("mqtt client is not yet supported")

	default:
		// use NewClient on the embedded server
		//hc = embedded.NewEmbeddedClient(clientID, nil)
	}

	return cc, err
}
