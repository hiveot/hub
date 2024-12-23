package clients

import (
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/discovery"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/clients/mqttclient"
	"github.com/hiveot/hub/transports/clients/sseclient"
	"github.com/hiveot/hub/transports/clients/wssclient"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
	"os"
	"path"
	"strings"
	"time"
)

// TokenFileExt defines the filename extension under which client tokens are stored
// in the keys directory.
const TokenFileExt = ".token"

var DefaultTimeout = time.Second * 3

// ClientFactory is a factory to create client connections
type ClientFactory struct {
	caCert *x509.Certificate
}

// NewHubClientFactory creates a new client factory for connecting to the hiveot hub
func NewHubClientFactory(certsDir string) (*ClientFactory, error) {
	// obtain the CA public cert to verify the server
	caCertFile := path.Join(certsDir, certs.DefaultCaCertFile)
	caCert, err := certs.LoadX509CertFromPEM(caCertFile)
	if err != nil {
		return nil, err
	}

	cf := &ClientFactory{
		caCert: caCert,
	}
	return cf, nil
}

// ConnectToHub helper function to connect to the hiveot Hub using existing token and key files.
// This assumes that CA cert, user keys and auth token have already been set up and
// are available in the certDir.
// The key-pair file is named {certDir}/{clientID}.key
// The token file is named {certDir}/{clientID}.token
//
// 1. If no fullURL is given then use discovery to determine the URL
// 2. Determine the core to use
// 3. Load the CA cert
// 4. Create a hub client
// 5. Connect using token and key files
//
//	fullURL is the scheme://addr:port/[wspath] the server is listening on. "" for auto discovery
//	clientID to connect as. Also used for the key and token file names
//	certDir is the credentials directory containing the CA cert (caCert.pem) and key/token files ({clientID}.token)
//	core optional core selection. Fallback is to auto determine based on URL.
//	password optional for a user login instead of a token
func ConnectToHub(fullURL string, clientID string, certDir string, password string) (
	hc transports.IAgentConnection, err error) {

	// 1. determine the actual address
	if fullURL == "" {
		// return after first result
		fullURL = discovery.LocateHub(time.Second, true)
		if fullURL == "" {
			return nil, fmt.Errorf("Hub not found")
		}
	}
	if clientID == "" {
		return nil, fmt.Errorf("missing clientID")
	}
	// 2. obtain the CA public cert to verify the server
	caCertFile := path.Join(certDir, certs.DefaultCaCertFile)
	caCert, err := certs.LoadX509CertFromPEM(caCertFile)
	if err != nil {
		return nil, err
	}
	// 3. Determine which protocol to use and setup the key and token filenames
	hc, _ = NewTransportClient(fullURL, clientID, caCert, nil, 0)
	if hc == nil {
		return nil, fmt.Errorf("unable to create hub client for URL: %s", fullURL)
	}

	// 4. Connect and auth with token from file
	slog.Info("connecting to", "serverURL", fullURL)
	if password != "" {
		_, err = hc.ConnectWithPassword(password)
	} else {
		// login with token file
		err = ConnectWithTokenFile(hc, certDir)
	}
	if err != nil {
		return nil, err
	}
	return hc, err
}

// ConnectWithTokenFile is a convenience function to read token and key
// from file and connect to the server.
//
// keysDir is the directory with the {clientID}.key and {clientID}.token files.
func ConnectWithTokenFile(hc transports.IAgentConnection, keysDir string) error {
	var kp keys.IHiveKey

	clientID := hc.GetClientID()

	slog.Info("ConnectWithTokenFile",
		slog.String("keysDir", keysDir),
		slog.String("clientID", clientID))
	keyFile := path.Join(keysDir, clientID+keys.KPFileExt)
	tokenFile := path.Join(keysDir, clientID+TokenFileExt)
	token, err := os.ReadFile(tokenFile)
	if err == nil && keyFile != "" {
		kp, err = keys.NewKeyFromFile(keyFile)
		//TODO: future use for key-pair?
		_ = kp
	}
	if err != nil {
		return fmt.Errorf("ConnectWithTokenFile failed: %w", err)
	}
	//hc.kp = kp
	_, err = hc.ConnectWithToken(string(token))
	return err
}

// NewTransportClient returns a new client protocol instance
//
// FullURL contains the full server address as provided by discovery:
//
//	https://addr:port/ for http without sse
//	https://addr:port/sse for http with the sse subprotocol binding
//	https://addr:port/ssesc for http with the sse-sc subprotocol binding
//	wss://addr:port/wss for websocket over TLS
//	mqtts://addr:port/ for mqtt over websocket over TLS
//
// clientID is the ID to authenticate as when using one of the Connect... methods
//
// caCert is the server's CA certificate to verify the connection. Using nil will
// ignore the server certificate check.
//
// GetForm is the function that provides a form for the operation. When connecting to
// the hiveot hub, this is optional as the protocolType automatically selects the generic form.
//
// timeout is optional maximum wait time for connecting or waiting for responses. Use 0 for default.
func NewTransportClient(
	fullURL string, clientID string, caCert *x509.Certificate,
	getForm func(op string) td.Form, timeout time.Duration) (
	bc transports.IAgentConnection, err error) {

	// determine the protocol to use from the URL
	protocolType := transports.ProtocolTypeWSS
	if strings.HasPrefix(fullURL, "https") {
		if strings.HasSuffix(fullURL, httpserver.DefaultSSEPath) {
			protocolType = transports.ProtocolTypeSSE
		} else if strings.HasSuffix(fullURL, httpserver.DefaultSSESCPath) {
			protocolType = transports.ProtocolTypeSSESC
		} else {
			protocolType = transports.ProtocolTypeHTTPS
		}
	} else if strings.HasPrefix(fullURL, "wss") {
		protocolType = transports.ProtocolTypeWSS
	} else if strings.HasPrefix(fullURL, "mqtts") {
		protocolType = transports.ProtocolTypeMQTTS
	} else {
		return nil, fmt.Errorf("Unknown protocol type in URL: " + fullURL)
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	// Create the client for the protocol
	switch protocolType {
	case transports.ProtocolTypeHTTPS:
		panic("Don't use HTTPS protocol, use the SSE-SC or WSS subprotocol instead")
		//bc = httpclient.NewHttpAgentTransport(
		//	fullURL, clientID, nil, caCert, getForm, timeout)

	case transports.ProtocolTypeMQTTS:
		bc = mqttclient.NewMqttAgentTransport(
			fullURL, clientID, nil, caCert, getForm, timeout)

	// the default SSE creates a connection for each subscription and observation
	case transports.ProtocolTypeSSE:
		//bc, err = sseclient.NewSseBindingClient(fullURL, clientID, nil, caCert, timeout)
		panic("sse client is not yet supported")

	case transports.ProtocolTypeSSESC:
		bc = sseclient.NewSsescAgentTransport(
			fullURL, clientID, nil, caCert, getForm, timeout)

	case transports.ProtocolTypeWSS:
		bc = wssclient.NewWssAgentTransport(
			fullURL, clientID, nil, caCert, getForm, timeout)

	default:
		// use NewClient on the embedded server
		//hc = embedded.NewEmbeddedClient(clientID, nil)
	}

	return bc, err
}
