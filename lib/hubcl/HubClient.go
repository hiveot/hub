package hubcl

import (
	"crypto/ecdsa"
	"crypto/x509"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/lib/hubcl/mqtthubclient"
	"github.com/hiveot/hub/lib/hubcl/natshubclient"
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
	if kp == nil {
		panic("kp is required")
	}
	if core == "nats" || strings.HasPrefix(url, "nats") {
		return natshubclient.NewNatsHubClient(url, clientID, kp.(nkeys.KeyPair), caCert)
	}
	return mqtthubclient.NewMqttHubClient(url, clientID, kp.(*ecdsa.PrivateKey), caCert)
}
