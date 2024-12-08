package clients

import (
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/clients/httpclient"
	"github.com/hiveot/hub/transports/clients/mqttclient"
	"github.com/hiveot/hub/transports/clients/ssescclient"
	"github.com/hiveot/hub/transports/clients/wssclient"
	"github.com/hiveot/hub/wot/td"
	"strings"
	"time"
)

var DefaultTimeout = time.Second * 300 // Default is 3, 300 for testing

// CreateTransportClient returns a new protocol binding client instance
//
// FullURL contains the full server address as provided by discovery:
//
//	https://addr:port/ for http without sse
//	https://addr:port/sse for http with the sse subprotocol binding
//	https://addr:port/ssesc for http with the sse-sc subprotocol binding
//	wss://addr:port/wss for websocket over TLS
//	mqtts://addr:port/ for mqtt over websocket over TLS
//
// GetForm is the function that provides a form for the operation.
// When talking to the hiveot hub, none is needed as the protocolType automatically
// selects the generic form.
func CreateTransportClient(
	fullURL string, clientID string, caCert *x509.Certificate,
	getForm func(op string) td.Form) (
	bc transports.IClientConnection, err error) {

	// determine the protocol to use from the URL
	protocolType := transports.ProtocolTypeWSS
	if strings.HasPrefix(fullURL, "https") {
		if strings.HasSuffix(fullURL, "sse") {
			protocolType = transports.ProtocolTypeSSE
		} else if strings.HasSuffix(fullURL, "ssesc") {
			protocolType = transports.ProtocolTypeSSESC
		} else {
			protocolType = transports.ProtocolTypeHTTPS
		}
	} else if strings.HasPrefix(fullURL, "wss") {
		protocolType = transports.ProtocolTypeWSS
	} else if strings.HasPrefix(fullURL, "mqtts") {
		protocolType = transports.ProtocolTypeMQTTS
	} else {
		return nil, fmt.Errorf("Unknown protocol type in URL: " + fullURL)
	}

	// Create the client for the protocol
	switch protocolType {
	case transports.ProtocolTypeHTTPS:
		bc = httpclient.NewHttpTransportClient(
			fullURL, clientID, nil, caCert, getForm, DefaultTimeout)

	case transports.ProtocolTypeMQTTS:
		bc = mqttclient.NewMqttTransportClient(
			fullURL, clientID, nil, caCert, getForm, DefaultTimeout)

	// the default SSE creates a connection for each subscription and observation
	case transports.ProtocolTypeSSE:
		//bc, err = sseclient.NewSseBindingClient(fullURL, clientID, nil, caCert, DefaultTimeout)
		panic("sse client is not yet supported")

	case transports.ProtocolTypeSSESC:
		bc = ssescclient.NewSsescTransportClient(
			fullURL, clientID, nil, caCert, getForm, DefaultTimeout)

	case transports.ProtocolTypeWSS:
		bc = wssclient.NewWssTransportClient(fullURL, clientID, nil, caCert, DefaultTimeout)

	default:
		// use NewClient on the embedded server
		//hc = embedded.NewEmbeddedClient(clientID, nil)
	}

	return bc, err
}
