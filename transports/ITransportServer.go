package transports

import "github.com/hiveot/hub/wot/td"

// ActionHandler is the API for service action handling
//type ActionHandler func(msg *ThingMessage) (stat RequestStatus)

// PermissionHandler is the handler that authorizes the sender to perform an operation
//
//	senderID is the account ID of the consumer or agent
//	operation is one of the predefined operations, eg WotOpReadEvent
//	dThingID is the ID of the digital twin Thing the request applies to
//type PermissionHandler func(senderID, operation, dThingID string) bool

// ProtocolInfo contains information provided by the binding
//type ProtocolInfo struct {
//	BaseURL string `json:"baseURL"`
//	// The schema used by the protocol: "https, mqtt, nats"
//	Schema string `json:"schema"`
//	// Transport used by this protocol: "https, mqtt, nats, ..."
//	// Transport IDs uniquely identify the transport: https, mqtts, nats
//	Transport string `json:"transport"`
//}

// ITransportServer is the interface implemented by all transport protocol bindings
type ITransportServer interface {

	// AddTDForms adds the Forms for using this protocol bindings to the provided TD.
	// This adds the operations for reading/writing properties, events and actions
	// Original forms must be removed first as they are no longer applicable.
	AddTDForms(td *td.TD) error

	// GetForm generates a form for the given operation for this server's transport
	// protocol. Intended to update a TD with forms.
	// Forms can use the following URI variables for top level Things:
	//	{op} for operation
	// 	{thingID} the ID of the thing
	//	{name} the name of the property, event or action affordance
	GetForm(op string) td.Form

	// GetProtocolInfo returns information on the protocol provided by the binding.
	//GetProtocolInfo() ProtocolInfo

	// SendNotification broadcast an event or property change to subscribers
	// Use this instead of sending notifications to individual connections
	// as message bus brokers handle their own subscriptions.
	SendNotification(operation string, dThingID, name string, data any)

	// Stop the server
	Stop()
}
