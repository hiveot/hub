package transports

import "github.com/hiveot/hub/wot/td"

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

	// GetConnectURL returns the URL to connect to this server
	GetConnectURL() string

	// SendNotification broadcast an event or property change to subscribers
	// Use this instead of sending notifications to individual connections
	// as message bus brokers handle their own subscriptions.
	SendNotification(msg NotificationMessage)

	// Stop the server
	Stop()
}
