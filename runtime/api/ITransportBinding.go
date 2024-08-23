package api

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/wot/tdd"
)

// ProtocolInfo contains information provided by the binding
type ProtocolInfo struct {
	BaseURL string `json:"baseURL"`
	// The schema used by the protocol: "https, mqtt, nats"
	Schema string `json:"schema"`
	// Transport used by this protocol: "https, mqtt, nats, ..."
	// Transport IDs uniquely identify the transport: https, mqtts, nats
	Transport string `json:"transport"`
}

// ITransportBinding is the interface implemented by all transport protocol bindings
type ITransportBinding interface {

	// AddTDForms add Forms to the TD for communication with the digital twin
	// using this transport binding.
	// This adds the operations for reading/writing properties, events and actions
	AddTDForms(td *tdd.TD)

	// GetProtocolInfo returns information on the protocol provided by the binding.
	GetProtocolInfo() ProtocolInfo

	// SendToClient sends a message to a connected agent or consumer client.
	//
	// This returns a delivery status, and a reply if delivery is completed.
	// If delivery is not completed then a status update will be returned asynchronously
	// through a 'EventTypeDelivery' event and no error is returned.
	SendToClient(clientID string, msg *hubclient.ThingMessage) (stat hubclient.DeliveryStatus, found bool)

	// SendEvent publishes an event message to all subscribers of this protocol binding
	SendEvent(event *hubclient.ThingMessage) hubclient.DeliveryStatus

	// Start the protocol binding
	//  handler is the handler that processes incoming messages
	Start(handler hubclient.MessageHandler) error

	// Stop the protocol binding
	Stop()
}
