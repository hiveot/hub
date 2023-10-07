package hubconnect

import (
	"fmt"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/discovery"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/mqtthubclient"
	"github.com/hiveot/hub/lib/hubclient/natshubclient"
	"log/slog"
	"path"
	"strings"
	"time"
)

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
