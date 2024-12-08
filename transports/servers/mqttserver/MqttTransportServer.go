package mqttserver

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	transports2 "github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/connections"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
)

type MqttTransportServer struct {
	host           string
	tcpPort        int
	wssPort        int
	serverCert     *tls.Certificate
	caCert         *x509.Certificate
	authenticator  transports2.IAuthenticator
	messageHandler transports2.ServerMessageHandler
	cm             *connections.ConnectionManager

	// convert operation to topics (for building forms)
	op2Topic map[string]string
}

func (svc *MqttTransportServer) AddTDForms(td *td.TD) error {
	//svc.AddThingLevelForms(td)
	//svc.AddPropertiesForms(td)
	//svc.AddEventsForms(td)
	//svc.AddActionForms(td)
	return nil
}

// GetForm returns a new HTTP form for the given operation
// Intended for Thing level operations
func (svc *MqttTransportServer) GetForm(op string) td.Form {
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
	form := td.Form{}
	form["op"] = op
	form["href"] = hostPort
	form["mqv:retain"] = "false"
	form["mqv:qos"] = "1"
	form["mqv:topic"] = topic
	form["mqv:controlPacket"] = controlPacket

	slog.Warn("GetForm. No form found for operation",
		"op", op)
	return nil
}

// GetProtocolInfo returns info on the protocol supported by this binding
//func (svc *MqttTransportServer) GetProtocolInfo() transports.ProtocolInfo {
//	//hostName := svc.config.Host
//	//if hostName == "" {
//	//	hostName = "localhost"
//	//}
//	baseURL := fmt.Sprintf("https://%s:%d", svc.host, svc.tcpPort)
//	inf := transports.ProtocolInfo{
//		BaseURL:   baseURL,
//		Schema:    "https",
//		Transport: "https",
//	}
//	return inf
//}

// SendNotification broadcast an event or property change to subscribers clients
func (svc *MqttTransportServer) SendNotification(operation string, dThingID, name string, data any) {
	// this is needed so mqtt can broadcast once via the message bus instead all individual connections
	// tbd. An embedded mqtt server can still send per connection?
	slog.Error("todo: implement")
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
	authenticator transports2.IAuthenticator,
	messageHandler transports2.ServerMessageHandler,
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
