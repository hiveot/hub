package clients

import (
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/discovery"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/clients/hiveotwssclient"
	"github.com/hiveot/hub/transports/servers"
	"github.com/hiveot/hub/transports/servers/hiveotwssserver"
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

// ConnectClient helper function creates a client transport connection to the
// server using the given protocol. Intended for consumers and Thing agents.
//
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
//	fullURL is the scheme://addr:port/[wssPath] the server is listening on. "" for auto discovery
//	clientID to connect as. Also used for the key and token file names
//	certDir is the credentials directory containing the CA cert (caCert.pem) and key/token files ({clientID}.token)
func ConnectClient(fullURL string, clientID string, certDir string) (
	cc transports.IClientConnection, err error) {

	if clientID == "" {
		return nil, fmt.Errorf("missing clientID")
	}
	// 1. determine the actual address
	if fullURL == "" {
		// return after first result
		disco, err := discovery.LocateHub(time.Second, true)
		if err != nil {
			return nil, fmt.Errorf("Hub not found")
		}
		// FIXME: specified a protocol
		fullURL = disco.HiveotWssURL
		if fullURL == "" {
			fullURL = disco.HiveotSseURL
		}
	}

	// 2. obtain the CA public cert to verify the server
	caCertFile := path.Join(certDir, certs.DefaultCaCertFile)
	caCert, err := certs.LoadX509CertFromPEM(caCertFile)
	if err != nil {
		return nil, err
	}
	// 3. Determine which protocol to use and setup the key and token filenames
	// getForm should be set by the application that has the Thing directory
	cc, _ = NewClient(fullURL, clientID, caCert, nil, 0)
	if cc == nil {
		return nil, fmt.Errorf("unable to create client for URL: %s", fullURL)
	}

	// 4. Connect and auth with token from file
	slog.Info("connecting to", "serverURL", fullURL)
	// agents use token files
	err = ConnectWithTokenFile(cc, certDir)

	if err != nil {
		return nil, err
	}
	return cc, err
}

// ConnectWithTokenFile is a convenience function to read token and key
// from file and connect to the server. Also used by agents.
//
// keysDir is the directory with the {clientID}.key and {clientID}.token files.
func ConnectWithTokenFile(cc transports.IClientConnection, keysDir string) error {
	var kp keys.IHiveKey

	clientID := cc.GetClientID()

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
	//cc.kp = kp
	err = cc.ConnectWithToken(string(token))
	return err
}

// NewClient returns a new client connection for connecting to a wot server.
//
// FullURL contains the full server address as provided by discovery:
//
//	https://addr:port/ for http without sse
//	https://addr:port/wot/sse for http with the sse subprotocol binding
//	https://addr:port/hiveot/sse for http with the ssesc subprotocol binding
//	wss://addr:port/wot/wss for websocket over TLS
//	wss://addr:port/hiveot/wss for direct messaging websocket over TLS
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
// timeout is optional maximum wait time for connecting or waiting for responses.
// Use 0 for default.
func NewClient(
	fullURL string, clientID string, caCert *x509.Certificate,
	getForm transports.GetFormHandler, timeout time.Duration) (
	cc transports.IClientConnection, err error) {

	// determine the protocol to use from the URL
	protocolType := transports.ProtocolTypeHiveotWSS
	if strings.HasPrefix(fullURL, "https") {
		if strings.HasSuffix(fullURL, servers.DefaultHiveotSsePath) {
			protocolType = transports.ProtocolTypeHiveotSSE
		} else if strings.HasSuffix(fullURL, servers.DefaultHiveotWssPath) {
			protocolType = transports.ProtocolTypeHiveotWSS
		} else if strings.HasSuffix(fullURL, servers.DefaultWotWssPath) {
			protocolType = transports.ProtocolTypeHiveotSSE
		} else {
			protocolType = transports.ProtocolTypeWotHTTPBasic
		}
	} else if strings.HasPrefix(fullURL, "wss") {
		protocolType = transports.ProtocolTypeWotWSS
		if strings.HasSuffix(fullURL, servers.DefaultHiveotWssPath) {
			protocolType = transports.ProtocolTypeHiveotWSS
		}
	} else if strings.HasPrefix(fullURL, "mqtts") {
		protocolType = transports.ProtocolTypeWotMQTTWSS
	} else {
		return nil, fmt.Errorf("Unknown protocol type in URL: " + fullURL)
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	// Create the client for the protocol
	switch protocolType {
	case transports.ProtocolTypeHiveotSSE:
		//bc = sseclient.NewSsescAgentClient(
		//	fullURL, clientID, nil, caCert, timeout)
		panic("sse-sc client is broken")

	case transports.ProtocolTypeHiveotWSS:
		msgConverter := &hiveotwssserver.HiveotMessageConverter{}
		cc = hiveotwssclient.NewWssClientConnection(
			fullURL, clientID, caCert, msgConverter, timeout)

	case transports.ProtocolTypeWotWSS:
		//msgConverter := &hiveotwssserver.WotWssMessageConverter{}
		//cc = hiveotwssclient.NewWssClientConnection(
		//	fullURL, clientID, nil, caCert,
		//	msgConverter, nil, timeout)
		panic("wss client is broken")

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
