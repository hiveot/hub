package mqttserver

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/connections"
	"log/slog"
)

const DefaultMqttTcpPort = 8883
const DefaultMqttWssPort = 8884

type MqttTransportServer struct {
	host           string
	tcpPort        int
	wssPort        int
	serverCert     *tls.Certificate
	caCert         *x509.Certificate
	authenticator  transports.IAuthenticator
	messageHandler transports.ServerMessageHandler
	cm             *connections.ConnectionManager

	// convert operation to topics (for building forms)
	op2Topic map[string]string
}

func (svc *MqttTransportServer) AddTDForms(td *tdd.TD) error {
	//svc.AddThingLevelForms(td)
	//svc.AddPropertiesForms(td)
	//svc.AddEventsForms(td)
	//svc.AddActionForms(td)
	return nil
}

// GetForm returns a new HTTP form for the given operation
// Intended for Thing level operations
func (svc *MqttTransportServer) GetForm(op string) tdd.Form {
	controlPacket := ""
	topic, found := svc.op2Topic[op]
	if !found {
		slog.Error("GetForm. Operation doesn't have corresponding message type",
			"op", op)
		return nil
	}
	switch op {
	case wot.OpSubscribeEvent, wot.OpSubscribeAllEvents,
		wot.OpObserveProperty, wot.OpObserveAllProperties:
		controlPacket = "subscribe"
	case wot.OpUnsubscribeEvent, wot.OpUnsubscribeAllEvents,
		wot.OpUnobserveProperty, wot.OpUnobserveAllProperties:
		controlPacket = "unsubscribe"
	case wot.OpReadProperty, wot.OpReadMultipleProperties, wot.OpReadAllProperties,
		wot.OpWriteProperty, wot.OpWriteMultipleProperties,
		wot.OpInvokeAction:
		// NOTE: the spec recommends to use subscribe for reading properties, but that
		// makes no sense (yet).
		// https://w3c.github.io/wot-binding-templates/bindings/protocols/mqtt/index.html#default-mappings
		controlPacket = "publish"
	default:
		controlPacket = "publish"

	}

	hostPort := fmt.Sprintf("mqtts://%s:%d", svc.host, svc.tcpPort)
	form := tdd.Form{}
	form["op"] = op
	form["href"] = hostPort
	form["mqv:retain"] = "false"
	form["mqv:qos"] = "1"
	form["mqv:topic"] = topic
	form["mqv:controlPacket"] = controlPacket

	slog.Warn("GetForm. No form found for operation",
		"op", op,
		"protocol", svc.GetProtocolInfo().Schema)
	return nil
}

// GetProtocolInfo returns info on the protocol supported by this binding
func (svc *MqttTransportServer) GetProtocolInfo() transports.ProtocolInfo {
	//hostName := svc.config.Host
	//if hostName == "" {
	//	hostName = "localhost"
	//}
	baseURL := fmt.Sprintf("https://%s:%d", svc.host, svc.tcpPort)
	inf := transports.ProtocolInfo{
		BaseURL:   baseURL,
		Schema:    "https",
		Transport: "https",
	}
	return inf
}

// Stop the mqtt broker
func (svc *MqttTransportServer) Stop() {
	slog.Warn("Stopping MqttTransportServer not yet implemented")
}

// StartMqttTransportServer creates and starts a new instance of the Mqtt broker
//
// Call stop to end the transport server.
//
//	config
//	privKey
//	caCert
//	sessionAuth for creating and validating authentication tokens
//	dtwService that handles digital thing requests
func StartMqttTransportServer(host string, tcpPort int, wssPort int,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	authenticator transports.IAuthenticator,
	messageHandler transports.ServerMessageHandler,
	cm *connections.ConnectionManager,
) (*MqttTransportServer, error) {
	svc := &MqttTransportServer{
		serverCert:     serverCert,
		caCert:         caCert,
		authenticator:  authenticator,
		messageHandler: messageHandler,
		cm:             cm,
	}
	return svc, fmt.Errorf("Not yet implemented")
}
