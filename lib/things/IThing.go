package things

// IThing is the interface implemented by local Thing instances.
//
// Intended to be implemented by publishers for each of the Things they
// publish. This provides a standard representation of a thing that can be
// interacted with.
//
// Things have 2 roles:
//  1. pass configuration and actions requests to the device
//  2. publish events and config changes to the Hub as events
//
// Each publisher keeps a collection of Thing instances that represent
// the Things it manages. Events from the Thing are published by the
// publisher. Actions received by the publisher are passed on to the Thing.
// The thing Instance provides a method to generate a TD document that
// describes the properties, events and actions of the Thing.
type IThing interface {
	// GetID Return the Thing ID. The Thing ID is the local, to the publisher,
	// unique identifier.
	GetID() string

	// GetTD return the TD document that describes the Thing represented here.
	GetTD() *TD

	// GetEventValues returns the map of latest event values by eventID.
	// The values have been serialized in a format described in the Thing TD
	// event affordance.
	// If 'changed' values are requested then only events whose value
	// have changed since the previous 'changed' request are returned.
	//GetEventValues(changed bool) map[string]string

	// GetPropertyValues returns the map of current property values.
	// The values have been serialized in a format described in the Thing TD
	// property affordance.
	// If 'changed' values are requested then only properties whose value
	// has changed since the previous 'changed' request are returned.
	//GetPropertyValues(changed bool) map[string]string

	// HandleActionRequest applies an incoming action request to the Thing.
	//
	// Intended to be invoked by the hub connection handler.
	// This returns an error if the action request is refused.
	// The action value has been serialized in a format described in the Thing TD
	// action affordance, or is empty if no data is expected.
	//
	// tv contains the action name, value and sender
	HandleActionRequest(tv *ThingMessage) error

	// HandleConfigRequest applies an incoming config request to the Thing.
	// Intended to be invoked by the hub connection handler.
	//
	// This returns an error if the configuration request is refused.
	// The config value has been serialized in a format described in the Thing TD
	// property affordance.
	//
	// tv contains the property name, value and sender
	HandleConfigRequest(tv *ThingMessage) error

	// Rename applies a new friendly name to the Thing.
	// If accepted, this updates the vocab.Name property.
	Rename(newName string) error

	// SetEventCB sets the callback that is notified of Thing events taking place
	// The value has been serialized in a format described in the Thing TD
	// event affordance, or is empty if the event carriers no data.
	//
	// Intended for binding agents/publishers so they can publish an event with the changed value.
	SetEventCB(func(eventID string, value string))

	// SetPropCB sets the callback that is notified when a property change has taken place
	// The value has been serialized in a format described in the Thing TD
	// property affordance.
	//
	// Intended for binding agents/publishers so they can publish a property event with the changed value.
	SetPropCB(func(propID string, value string))
}
