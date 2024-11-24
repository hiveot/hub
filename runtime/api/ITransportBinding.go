package api

import (
	"github.com/hiveot/hub/wot/transports"
)

// ActionHandler is the API for service action handling
type ActionHandler func(msg *transports.ThingMessage) (stat transports.RequestStatus)

// PermissionHandler is the handler that authorizes the sender to perform an operation
//
//	senderID is the account ID of the consumer or agent
//	operation is one of the predefined operations, eg WotOpReadEvent
//	dThingID is the ID of the digital twin Thing the request applies to
type PermissionHandler func(senderID, operation, dThingID string) bool

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
// Intended to send messages to the connecting client.
//type ITransportBinding interface {
//
//	// AddTDForms adds the Forms for using this protocol bindings to the provided TD.
//	// This adds the operations for reading/writing properties, events and actions
//	// Original forms must be removed first as they are no longer applicable.
//	AddTDForms(td *tdd.TD) error
//
//	// GetProtocolInfo returns information on the protocol provided by the binding.
//	//GetProtocolInfo() ProtocolInfo
//
//	// GetConnectionByCID returns the client connection for sending messages to a client
//	GetConnectionByCID(cid string) IClientConnection
//
//	// PublishEvent publishes an event message to all connected subscribers
//	//
//	//	dThingID is the Thing ID of the digital twin
//	//	name is the name of the event as per digital twin event affordance
//	//	value is the raw event value as per event affordance data schema
//	//	requestID is the optional ID of a linked action
//	PublishEvent(dThingID string, name string, value any, requestID string, agentID string)
//
//	// PublishProperty publishes a new property value to observers of the property
//	//
//	//	dThingID is the Thing ID of the digital twin
//	//	name is the name of the property as per digital twin property affordance
//	//	value is the raw property value as per property affordance data schema
//	//	requestID is the optional ID of a linked action
//	PublishProperty(dThingID string, name string, value any, requestID string, agentID string)
//
//	// WriteProperty sends a request to write a property to the agent with the given ID
//	//
//	// Only supported on bindings that support subscriptions
//	// This returns found is false if the agent is not connected.
//	//
//	//
//	//WriteProperty(agentID string, thingID string, name string, value any, requestID string,
//	//	senderID string) (found bool, status string, err error)
//}
