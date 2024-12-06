package clients

import (
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/wot/td"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/clients/httpbinding"
	"github.com/hiveot/hub/wot/transports/clients/mqttbinding"
	"github.com/hiveot/hub/wot/transports/clients/ssescclient"
	"github.com/hiveot/hub/wot/transports/clients/wssclient"
	"time"
)

var DefaultTimeout = time.Second * 300 // Default is 3, 300 for testing

// CreateTransportClient returns a new protocol binding client instance
// GetForm is the function that provides a form for the operation.
// Only needed for http.
func CreateTransportClient(
	protocolType string, fullURL string, clientID string, caCert *x509.Certificate,
	getForm func(op string) td.Form) (
	bc transports.IClientConnection, err error) {

	switch protocolType {
	case transports.ProtocolTypeHTTP:
		bc = httpbinding.NewHttpTransportClient(
			fullURL, clientID, nil, caCert, getForm, DefaultTimeout)

	case transports.ProtocolTypeMQTT:
		bc = mqttbinding.NewMqttBindingClient(
			fullURL, clientID, nil, caCert, getForm, DefaultTimeout)

	// the default SSE creates a connection for each subscription and observation
	// not bothering with this at the moment. Use sse-sc or websocket.
	//case transports.ProtocolTypeSSE:
	//	bc, err = sse.NewSseBindingClient(fullURL, clientID, nil, caCert, DefaultTimeout)

	case transports.ProtocolTypeSSESC:
		bc = ssescclient.NewSsescTransportClient(
			fullURL, clientID, nil, caCert, getForm, DefaultTimeout)

	case transports.ProtocolTypeWSS:
		bc = wssclient.NewWssTransportClient(fullURL, clientID, nil, caCert, DefaultTimeout)

	default:
		// use NewClient on the embedded server
		//hc = embedded.NewEmbeddedClient(clientID, nil)
	}
	if bc == nil {
		err = fmt.Errorf("unknown protocol type '%s'", protocolType)
	}
	return bc, err
}
