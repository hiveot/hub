package hubclient

//// ISubscription interface to underlying subscription mechanism
//type ISubscription interface {
//	Unsubscribe() error
//}

// IHubClient defines the interface of a Thing agent that connects to a messaging server.
//
// TODO split this up in pure transport, consumed thing and exposed thing apis
type IHubClient interface {
	IConsumerClient

	// PubEvent [agent] publishes a Thing event.
	// It returns as soon as delivery to the hub is confirmed.
	// This is intended for agents, not for consumers.
	//
	// Events are published by agents using their native ID, not the digital twin ID.
	// The Hub stores the latest event in the corresponding digital twin and
	// broadcasts it to subscribers of the digital twin Thing.
	//
	// The Thing must be known to the Hub for the event to be accepted.
	// It is not required that the TD defines an event affordance for this event.
	// This will hide it from consumers that go by the TD.
	//
	//	thingID native ID of the thing as used by the agent. The thing must exist.
	//	name of the event to publish as described in the TD.
	//	value with native data to publish, as per TD DataSchema
	//	messageID if the event is in response to a request
	//
	// This returns an error if the event cannot not be delivered to the hub
	PubEvent(thingID string, name string, value any, messageID string) error

	// PubProgressUpdate [agent] sends a delivery progress update to the hub.
	// The hub will update the status of the action in the digital twin and
	// notify the original sender.
	//
	// Intended for agents that have processed an incoming action request asynchronously
	// and need to send an update on further progress.
	PubProgressUpdate(stat ActionProgress)

	// PubProperties [agent] publishes a property value update event to the hub.
	// It returns as soon as delivery to the hub is confirmed.
	// This is intended for agents, not for consumers.
	//
	//	thingID is the native ID of the device (not including the digital twin ID)
	//	props is the property name-value map to publish where value is the native value
	PubProperties(thingID string, props map[string]any) error

	// PubTD publishes a TD document to the Hub.
	// It returns as soon as delivery to the hub is confirmed.
	//
	// This is intended for agents, not for consumers.
	//	id is the Thing ID as seen by the agent (not the digitwin ID)
	//	td is the Thing Description document describing the Thing
	//PubTD(td *tdd.TD) error
	PubTD(thingID string, tdJSON string) error
}