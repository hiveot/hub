package clients

import (
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/clients/httpbinding"
	"github.com/hiveot/hub/wot/transports/clients/mqttbinding"
	"github.com/hiveot/hub/wot/transports/clients/ssescclient"
	"github.com/hiveot/hub/wot/transports/clients/wssbinding"
	"time"
)

var DefaultTimeout = time.Second * 300 // Default is 3, 300 for testing

// CreateTransportClient returns a new protocol binding client instance
func CreateTransportClient(
	protocolType string, fullURL string, clientID string, caCert *x509.Certificate) (
	bc transports.IClientConnection, err error) {

	switch protocolType {
	case transports.ProtocolTypeHTTP:
		bc = httpbinding.NewHttpTransportClient(
			fullURL, clientID, nil, caCert, DefaultTimeout)

	case transports.ProtocolTypeMQTT:
		bc = mqttbinding.NewMqttBindingClient(fullURL, clientID, nil, caCert, DefaultTimeout)

	// the default SSE creates a connection for each subscription and observation
	// not bothering with this at the moment. Use sse-sc or websocket.
	//case transports.ProtocolTypeSSE:
	//	bc, err = sse.NewSseBindingClient(fullURL, clientID, nil, caCert, DefaultTimeout)

	case transports.ProtocolTypeSSESC:
		bc = ssescclient.NewSsescBindingClient(fullURL, clientID, nil, caCert, DefaultTimeout)

	case transports.ProtocolTypeWSS:
		bc = wssbinding.NewWssTransportClient(fullURL, clientID, nil, caCert, DefaultTimeout)

	default:
		// use NewClient on the embedded server
		//hc = embedded.NewEmbeddedClient(clientID, nil)
	}
	if bc == nil {
		err = fmt.Errorf("unknown protocol type '%s'", protocolType)
	}
	return bc, err
}
