package transports

import "github.com/hiveot/hub/wot/td"

// ITransportServer is the interface implemented by all transport protocol bindings
type ITransportServer interface {

	// AddTDForms adds the Forms for using this protocol bindings to the provided TD.
	// This adds the operations for reading/writing properties, events and actions
	// Original forms must be removed first as they are no longer applicable.
	AddTDForms(td *td.TD) error

	// CloseAllClientConnections close all connections from the given client.
	// Intended to close connections after a logout.
	CloseAllClientConnections(clientID string)

	// GetConnectionByConnectionID returns the connection with the given ID from this server
	GetConnectionByConnectionID(cid string) IConnection

	// GetConnectionByClientID returns the connection with the given client ID.
	// Intended to find agents to route requests to.
	GetConnectionByClientID(agentID string) IConnection

	// GetForm generates a form for the given operation for this server's transport
	// protocol. Intended to update a TD with forms.
	// Forms can use the following URI variables for top level Things:
	//	{op} for operation
	// 	{thingID} the ID of the thing
	//	{name} the name of the property, event or action affordance
	GetForm(op string, thingID string, name string) *td.Form

	// GetConnectURL returns the URL to connect to this server
	// protocolType is intended for servers that support multiple protocols.
	// Use "" for default.
	GetConnectURL(protocolType string) string

	// GetProtocol returns the server supported protocol
	//GetProtocol() string

	// SendNotification sends an event or property update notification to connected
	// event subscribers or property observers.
	// The subscription is handled by the underlying transport protocol.
	SendNotification(msg *ResponseMessage)

	// CloseAll closes all connections but do not stop the server
	CloseAll()

	// Stop the server after force closing all connections
	Stop()
}
