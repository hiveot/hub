package hubcl

import (
	"crypto/ecdsa"
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/discovery"
	"github.com/hiveot/hub/lib/hubcl/mqtthubclient"
	"github.com/hiveot/hub/lib/hubcl/natshubclient"
	"github.com/nats-io/nkeys"
	"log/slog"
	"path"
	"strings"
	"time"
)

// NewHubClient returns a new Hub Client instance
//
// The kp keys are either nkeys.KeyPair for nats or ecdsa private key for mqtt.
// With Nats, 'kp' is the client's nkeys.KeyPair. It is used to connect with nkey or sign the server provided nonce
// when using JWT token authentication.
// With MQTT, 'kp' is the client's ecdsa private key.
//
//   - url of server to connect to.
//   - clientID of the client that will be connecting
//   - kp key pair use to identify with (required)
//   - caCert of server or nil to not verify server cert
//   - core server to use, "nats" or "mqtt". Default "" will use nats if url starts with "nats" or mqtt otherwise.
func NewHubClient(url string, clientID string, kp interface{}, caCert *x509.Certificate, core string) hubclient.IHubClient {
	// a kp is not needed when using connect with token file
	//if kp == nil {
	//	panic("kp is required")
	//}
	if core == "nats" || strings.HasPrefix(url, "nats") {
		var key nkeys.KeyPair
		if kp != nil {
			key = kp.(nkeys.KeyPair)
		}
		return natshubclient.NewNatsHubClient(url, clientID, key, caCert)
	}
	var key *ecdsa.PrivateKey
	if kp != nil {
		key = kp.(*ecdsa.PrivateKey)
	}
	return mqtthubclient.NewMqttHubClient(url, clientID, key, caCert)
}

// ConnectToHub helper function to connect to the Hub using existing token and key files.
// This assumes that CA cert, user keys and auth token have already been set up.
//
// The format for the key file is {clientID}.nkey for nats and {clientID}Key.pem for mqtt.
// For format for the token file is {clientID}.token.
//
// 1. If no fullURL is given then use discovery to determine the URL
// 2. Determine the core to use
// 3. Load the CA cert
// 4. Create a hub client
// 5. Connect using token and key files
func ConnectToHub(fullURL string, clientID string, certDir string, core string) (
	hc hubclient.IHubClient, err error) {

	var keyFile string
	var tokenFile string

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
	// 3. Determine which core to use and setup the key and token filenames
	// By convention the key/token filename format is "{name}.key/{name}.token"
	tokenFile = path.Join(certDir, clientID+".token")
	keyFile = path.Join(certDir, clientID+".key")
	if core == "nats" || strings.HasPrefix(fullURL, "nats") {
		// nats with nkeys. The key filename format is "{serviceID}.key"
		hc = natshubclient.NewNatsHubClient(fullURL, clientID, nil, caCert)
	} else {
		// mqtt with ecdsa keys. The key filename format is "{serviceID}.key|token"
		hc = mqtthubclient.NewMqttHubClient(fullURL, clientID, nil, caCert)
	}
	// 4. Connect and auth with token
	slog.Info("connecting to", "serverURL", fullURL)
	err = hc.ConnectWithTokenFile(tokenFile, keyFile)
	if err != nil {
		return nil, err
	}
	return hc, err
}
