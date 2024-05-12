package connect

import (
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/discovery"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/httpclient"
	"github.com/hiveot/hub/lib/keys"
	"log/slog"
	"net/url"
	"os"
	"path"
	"time"
)

// TokenFileExt defines the filename extension under which client tokens are stored
// in the keys directory.
const TokenFileExt = ".token"

// ConnectToHub helper function to connect to the Hub using existing token and key files.
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
//	fullURL is the scheme://addr:port/[wspath] the server is listening on
//	clientID to connect as. Also used for the key and token file names
//	certDir is the location of the CA cert and key/token files
//	core optional core selection. Fallback is to auto determine based on URL.
//	 password optional for a user login
func ConnectToHub(fullURL string, clientID string, certDir string, core string, password string) (
	hc hubclient.IHubClient, err error) {

	// 1. determine the actual address
	if fullURL == "" {
		// return after first result
		fullURL, core = discovery.LocateHub(time.Second, true)
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
	hc = NewHubClient(fullURL, clientID, caCert)

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
func ConnectWithTokenFile(hc hubclient.IHubClient, keysDir string) error {
	var kp keys.IHiveKey

	cid := hc.GetClientID()

	slog.Info("ConnectWithTokenFile",
		slog.String("keysDir", keysDir),
		slog.String("clientID", cid))
	keyFile := path.Join(keysDir, cid+keys.KPFileExt)
	tokenFile := path.Join(keysDir, cid+TokenFileExt)
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
	_, err = hc.ConnectWithJWT(string(token))
	return err
}

// NewHubClient returns a new Hub Client instance depending on the URL scheme.
//
// The keyPair string is optional. If not provided a new set of keys will be created.
// Use GetKeyPair to retrieve it for saving to file.
//
// For an embedded connection use the server's NewClient method.
//
//   - fullURL of server to connect to.
//   - clientID is the account/login ID of the client that will be connecting
//   - caCert of server or nil to not verify server cert
func NewHubClient(fullURL string, clientID string, caCert *x509.Certificate) (hc hubclient.IHubClient) {

	parts, _ := url.Parse(fullURL)
	clType := parts.Scheme
	if clType == "grpc" {
		// FIXME: grpc
		//cl = grpclient.NewGrpcClient(url, clientID, caCert)
	} else if clType == "nats" {
		// FIXME: nats
		//cl = natsclient.NewNatsClient(url, clientID, caCert)
	} else if clType == "mqtt" {
		// FIXME: support mqtt
		//tp = mqtttclient.NewMqttClient(url, clientID, caCert)
	} else if clType == "uds" {
		// FIXME: add support UDS connections for local services
	} else if clType == "https" || clType == "tls" {
		hc = httpclient.NewHttpSSEClient(parts.Host, clientID, caCert)
	} else if clType == "" {
		// use NewClient on the embedded server
		//hc = embedded.NewEmbeddedClient(clientID, nil)
	}
	return hc
}
