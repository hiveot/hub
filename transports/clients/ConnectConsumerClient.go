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

// ConnectConsumerToHub helper function to connect a consumer to the hiveot Hub,
// using existing token and key files.
//
// This assumes that CA cert and optionally an auth token file have already been
// set up and are available in the certDir.
// The CA cert is named caCert.pem
// The token file is named {certDir}/{clientID}.token
//
// 1. If no fullURL is given then use discovery to determine the URL
// 2. Load the CA cert from the cert dir and token file if it exists.
// 3. Create a hub client
// 4. Connect using token file or given password
//
// getForm is optional and intended to be interoperable with Forms. When connecting
// to the HiveOT hub this can be nil as it will fall back to the build-in messaging
// protocol that uses only request, response and notification message envelopes.
//
//	fullURL is the scheme://addr:port/[wspath] the server is listening on. "" for auto discovery
//	clientID to connect as. Also used as the token file name prefix.
//	certDir is the credentials directory containing the CA cert (caCert.pem) and token files ({clientID}.token)
//	password optional for a user login instead of a token
//	getForm is the consumer's handler to retrieve the Form for an operation on a Thing
func ConnectConsumerToHub(
	fullURL string, clientID string, certDir string, password string,
	getForm func(op, thingID, name string) td.Form) (
	hc transports.IConsumerConnection, err error) {

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
	hc, _ = NewConsumerClient(fullURL, clientID, caCert, getForm, 0)
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
// from file and connect to the server. Also used by agents.
//
// keysDir is the directory with the {clientID}.key and {clientID}.token files.
func ConnectWithTokenFile(hc transports.IConsumerConnection, keysDir string) error {
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

// NewConsumerClient returns a new client instance for consumers
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
func NewConsumerClient(
	fullURL string, clientID string, caCert *x509.Certificate,
	getForm func(op, thingID, name string) td.Form,
	timeout time.Duration) (
	bc transports.IConsumerConnection, err error) {

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
		bc = mqttclient.NewMqttConsumerClient(
			fullURL, clientID, nil, caCert, getForm, timeout)

	// the default SSE creates a connection for each subscription and observation
	case transports.ProtocolTypeSSE:
		//bc, err = sseclient.NewSseBindingClient(fullURL, clientID, nil, caCert, timeout)
		panic("sse client is not yet supported")

	case transports.ProtocolTypeSSESC:
		bc = sseclient.NewSsescConsumerClient(
			fullURL, clientID, nil, caCert, getForm, timeout)

	case transports.ProtocolTypeWSS:
		bc = wssclient.NewWssConsumerClient(
			fullURL, clientID, nil, caCert, getForm, timeout)

	default:
		// use NewClient on the embedded server
		//hc = embedded.NewEmbeddedClient(clientID, nil)
	}

	return bc, err
}
