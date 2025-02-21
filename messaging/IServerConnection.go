package messaging

// IServerConnection is the interface of an incoming client connection on the server.
// Protocol servers must implement this interface to return information to the consumer.
//
// This provides a return channel for sending messages from the digital twin to
// agents or consumers.
//
// Subscription to events or properties can be made externally via this API,
// or handled internally by the protocol handler if the protocol defines the
// messages for subscription.
type IServerConnection interface {
	IConnection

	// SendNotification sends a subscription response notification to a connected client.
	// This is the same as sending a response with the difference that the
	// correlationID will be set to that of the subscription.
	SendNotification(response ResponseMessage)

	//// Subscribe to property changes
	//// Experimental. Intended for integration split protocol http/sse
	//ObserveProperty(thingID, name string)
	//
	//// Subscribe to thing event(s)
	//// Experimental. Intended for integration split protocol http/sse
	//SubscribeEvent(thingID, name string)
	//
	//// UnsubscribeEvent unsubscribes from thing event(s)
	//// Experimental
	//UnsubscribeEvent(thingID, name string)
	//
	//// UnobserveProperty removes observation of property(ies)
	//// Experimental
	//UnobserveProperty(thingID, name string)
}
