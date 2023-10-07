package hubconnect

import (
	"crypto/ecdsa"
	"crypto/x509"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/mqtthubclient"
	"github.com/hiveot/hub/lib/hubclient/natshubclient"
	"github.com/nats-io/nkeys"
	"strings"
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
