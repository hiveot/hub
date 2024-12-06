package clients

import (
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/discovery"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/clients/mqttbinding"
	"github.com/hiveot/hub/wot/transports/clients/ssescclient"
	"github.com/hiveot/hub/wot/transports/clients/wssclient"
	"log/slog"
	"net/url"
	"os"
	"path"
	"time"
)

// TokenFileExt defines the filename extension under which client tokens are stored
// in the keys directory.
const TokenFileExt = ".token"

// ClientFactory is a factory to create client connections
type ClientFactory struct {
	caCert *x509.Certificate
}

// NewClient returns a new client connected to a server
func (fact *ClientFactory) NewClient(fullURL string, clientID string) (transports.IClientConnection, error) {

	cl := NewHubClient(fullURL, clientID, fact.caCert)
	return cl, nil
}

//func (fact *ClientFactory) NewClientFromForm(form td.Form) transports.IClientConnection {
//	// get the connect
//}

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
//	fullURL is the scheme://addr:port/[wspath] the server is listening on
//	clientID to connect as. Also used for the key and token file names
//	certDir is the location of the CA cert and key/token files
//	core optional core selection. Fallback is to auto determine based on URL.
//	 password optional for a user login
func ConnectToHub(fullURL string, clientID string, certDir string, password string) (
	hc transports.IClientConnection, err error) {

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
	hc = NewHubClient(fullURL, clientID, caCert)
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
func ConnectWithTokenFile(hc transports.IClientConnection, keysDir string) error {
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

// NewHubClient returns a new Hub agent client instance
//
// The keyPair string is optional. If not provided a new set of keys will be created.
// Use GetKeyPair to retrieve it for saving to file.
//
// For an embedded connection use the server's NewClient method.
//
//   - fullURL of server to connect to.
//   - clientID is the account/login ID of the client that will be connecting
//   - caCert of server or nil to not verify server cert
func NewHubClient(fullURL string, clientID string, caCert *x509.Certificate) (hc transports.IClientConnection) {

	parts, _ := url.Parse(fullURL)
	clType := parts.Scheme
	if clType == "mqtt" {
		hc = mqttbinding.NewMqttBindingClient(fullURL, clientID, nil, caCert, nil, DefaultTimeout)
	} else if clType == "uds" {
		// FIXME: add support tpc/uds connections for local services
	} else if clType == "wss" {
		hc = wssclient.NewWssTransportClient(fullURL, clientID, nil, caCert, DefaultTimeout)
	} else if clType == "https" || clType == "tls" {
		hc = ssescclient.NewSsescTransportClient(parts.Host, clientID, nil, caCert, nil, DefaultTimeout)
	} else if clType == "" {
		// use NewClient on the embedded server
		//hc = embedded.NewEmbeddedClient(clientID, nil)
	}
	if hc == nil {
		slog.Error("Unknown client type in URL schema", "clientType", clType, "url", fullURL)
	}
	return hc
}
