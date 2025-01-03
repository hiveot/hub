package clients

import (
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/discovery"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/clients/mqttclient"
	"github.com/hiveot/hub/transports/clients/sseclient"
	"github.com/hiveot/hub/transports/clients/wssclient"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"log/slog"
	"path"
	"strings"
	"time"
)

// ConnectAgentToHub helper function to connect an agent to the hiveot Hub using
// existing token file.
// This assumes that CA cert and auth token have already been set up and are available
// in the certDir.
//
// The token file is named {certDir}/{clientID}.token
//
// 1. If no fullURL is given then use discovery to determine the URL
// 2. Load the CA cert
// 3. Create an agent client
// 4. Connect using token file (agents do not use passwords)
//
//	fullURL is the scheme://addr:port/[wspath] the server is listening on. "" for auto discovery
//	clientID to connect as. Also used for the key and token file names
//	certDir is the credentials directory containing the CA cert (caCert.pem) and key/token files ({clientID}.token)
func ConnectAgentToHub(fullURL string, clientID string, certDir string) (
	hc transports.IAgentConnection, err error) {

	// 1. determine the actual address
	if fullURL == "" {
		// return after first result
		disco, err := discovery.LocateHub(time.Second, true)
		if err != nil {
			return nil, fmt.Errorf("Hub not found")
		}
		// FIXME: pick requested protocol
		fullURL = disco.SsescURL
		if disco.SsescURL == "" {
			fullURL = disco.WssURL
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
	hc, _ = NewAgentClient(fullURL, clientID, caCert, 0)
	if hc == nil {
		return nil, fmt.Errorf("unable to create hub client for URL: %s", fullURL)
	}

	// 4. Connect and auth with token from file
	slog.Info("connecting to", "serverURL", fullURL)
	// agents use token files
	err = ConnectWithTokenFile(hc, certDir)

	if err != nil {
		return nil, err
	}
	return hc, err
}

// NewAgentClient returns a new client protocol instance for agents
//
// FullURL contains the full server address as provided by discovery:
//
//	https://addr:port/ for http without sse
//	https://addr:port/sse for http with the sse subprotocol binding
//	https://addr:port/ssesc for http with the ssesc subprotocol binding
//	wss://addr:port/wss for websocket over TLS
//	mqtts://addr:port/ for mqtt over websocket over TLS
//
// clientID is the ID to authenticate as when using one of the Connect... methods
//
// caCert is the server's CA certificate to verify the connection. Using nil will
// ignore the server certificate check.
//
// Agents do not use forms as WoT does not support agents. This will fall back to
// the hiveot message envelopes.
//
// timeout is optional maximum wait time for connecting or waiting for responses. Use 0 for default.
func NewAgentClient(
	fullURL string, clientID string, caCert *x509.Certificate, timeout time.Duration) (
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
		protocolType = transports.ProtocolTypeMQTTCP
	} else if strings.HasPrefix(fullURL, "mqttwss") {
		// fixme, what is the mqtt wss prefix? how to differentiate from regular websockets
		protocolType = transports.ProtocolTypeMQTTWSS
	} else {
		return nil, fmt.Errorf("Unknown protocol type in URL: " + fullURL)
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	// Create the client for the protocol
	switch protocolType {
	case transports.ProtocolTypeHTTPS:
		panic("Don't use HTTPS protocol, use the SSESC or WSS subprotocol instead")
		//bc = httpclient.NewHttpAgentTransport(
		//	fullURL, clientID, nil, caCert, getForm, timeout)

	//case transports.ProtocolTypeMQTTWSS:
	//	bc = mqttclient.NewMqttAgentClient(
	//		fullURL, clientID, nil, caCert, timeout)

	case transports.ProtocolTypeMQTTCP:
		bc = mqttclient.NewMqttAgentClient(
			fullURL, clientID, nil, caCert, timeout)

	// the default SSE creates a connection for each subscription and observation
	case transports.ProtocolTypeSSE:
		//bc, err = sseclient.NewSseBindingClient(fullURL, clientID, nil, caCert, timeout)
		panic("sse client is not yet supported")

	case transports.ProtocolTypeSSESC:
		bc = sseclient.NewSsescAgentClient(
			fullURL, clientID, nil, caCert, timeout)

	case transports.ProtocolTypeWSS:
		bc = wssclient.NewWssAgentClient(
			fullURL, clientID, nil, caCert, timeout)

	default:
		// use NewClient on the embedded server
		//hc = embedded.NewEmbeddedClient(clientID, nil)
	}

	return bc, err
}
