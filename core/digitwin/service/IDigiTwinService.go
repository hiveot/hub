package service

import "github.com/hiveot/hub/lib/things"

//Digital Twin

type RequestStatus struct {
	// The request metadata and value
	things.ThingValue

	// Status holds the request progress status: completed, pending, cancelled, error
	// * completed: the request has been delivered and a result obtained, as per TD action affordance
	// * delivered: the request has been delivered to the agent but device itself is not yet available
	// * pending: the request has been queued and is pending delivery to the agent
	// * cancelled: the request has been cancelled
	// * replaced: the queued request has been replaced by a newer request with the same key
	// * error: the request has not been delivered or is rejected
	Status string `json:"status"`

	// error code if status is error
	Error string `json:"error,omitempty"`
}

// IDigiTwinService defines the interface of a service to manage Thing digital twins
// This service acts as a directory of TD documents and storage of the latest property
// and event values.
//
// As Thing cannot be addressed directly by consumers, the TD documents stored here will be
// modified with forms containing consumer facing protocols.
type IDigiTwinService interface {

	// GetValues returns the latest property and event values of a thing
	GetValues(agentID, thingID string) map[string]things.ThingValue

	// GetTD returns the TD of the digital twin
	GetTD(agentID, thingID string) *things.TD

	// GetTDD returns the JSON serialized TD document of the digital twin
	GetTDD(agentID, thingID string) string

	// AddEvent adds a new event value to a digital twin
	// This will be included in GetProps
	AddEvent(agentID, thingID string, eventName string, event *things.ThingValue) error

	// UpdateProps loads new property values of a digital twin as provided by the thing agent
	UpdateProps(agentID, thingID string, props map[string]string) error

	// UpdateTD updates the TD of the digital twin's as provided by the thing agent
	// This updates the TD stored on the digital twin.
	// The TD is modified with forms for communication via the Hub.
	UpdateTD(agentID string, td *things.TD)

	// UpdateTDD updates the TD document of the digital twin's as provided by the thing agent
	// This updates the TD stored on the digital twin and amends its forms to support
	// the consumer facing protocols.
	UpdateTDD(agentID string, tdd string)

	// RequestAction requests an action on the thing via its digital twin.
	//
	// If the Thing's agent is connected then the digital twin service will pass the
	// action request to the agent and return the result in the ActionResponse message.
	//
	// * If the agent is offline then the action is queued in the agent's inbox, and delivered
	//   when the agent reconnects. An action response message is sent after final delivery.
	// * If the action expires before the agent reconnects then an action response message
	//   is sent to the sender's inbox with the status 'expired'.
	// * If the action is overwritten with a new value then an action response message
	//   is sent to the sender's inbox with the status 'cancelled'.
	//
	// The action expiry is configured through the agent's inbox. The agent can update its account
	// to set the inbox size and expiry per message type.
	RequestAction(action *things.ThingValue) RequestStatus

	// RequestConfig requests a configuration change of the thing via its digital twin
	// This follows the same behavior as described in RequestAction.
	RequestConfig(config *things.ThingValue) RequestStatus
}

// IDigiTwinAgent defines the interface of the digital twin client and service for use by device agents
type IDigiTwinAgent interface {

	// AddEvent adds a new event value to a digital twin
	// This will be included in GetProps
	AddEvent(agentID, thingID string, eventName string, event *things.ThingValue) error

	// UpdateProps loads new property values of a digital twin as provided by the thing agent
	UpdateProps(agentID, thingID string, props map[string]string) error

	// UpdateTD updates the TD of the digital twin's as provided by the thing agent
	// This updates the TD stored on the digital twin.
	// The TD is modified with forms for communication via the Hub.
	UpdateTD(agentID string, td *things.TD)

	// SetActionHandler registers an agent's handler for receiving thing action requests.
	// Intended for agents and invoked by the digital twin when it receives an action request.
	SetActionHandler(func(action *things.ThingValue) RequestStatus)

	// SetConfigHandler registers a agent's handler for receiving thing configuration requests.
	// Intended for agents and invoked by the digital twin when it receives a configuration update request.
	SetConfigHandler(func(config *things.ThingValue) RequestStatus)
}

// IDigiTwinUser defines the interface of the digital twin user
type IDigiTwinUser interface {

	// GetValues returns the latest property and event values of a thing
	// keys is an optional list of property/event keys to get.
	// if no keys are given then all properties/event values are returned
	GetValues(agentID, thingID, keys []string) map[string]things.ThingValue

	// GetTD returns the TD of the digital twin
	GetTD(agentID, thingID string) *things.TD

	// GetTDD returns the JSON serialized TD document of the digital twin
	GetTDD(agentID, thingID string) string

	// RequestAction requests an action on the thing via its digital twin.
	//
	// If the Thing's agent is connected then the digital twin service will pass the
	// action request to the agent and return the result in the ActionResponse message.
	//
	// * If the agent is offline then the action is queued in the agent's inbox, and delivered
	//   when the agent reconnects. An action response message is sent after final delivery.
	// * If the action expires before the agent reconnects then an action response message
	//   is sent to the sender's inbox with the status 'expired'.
	// * If the action is overwritten with a new value then an action response message
	//   is sent to the sender's inbox with the status 'cancelled'.
	//
	// The action expiry is configured through the agent's inbox. The agent can update its account
	// to set the inbox size and expiry per message type.
	RequestAction(agentID, thingID, key string, data string) RequestStatus

	// RequestConfig requests a configuration change of the thing via its digital twin
	// This follows the same behavior as described in RequestAction.
	RequestConfig(agentID, thingID, key string, data string) RequestStatus

	// SetEventHandler registers a user's handler for receiving thing events.
	// Intended for users and invoked by the digital twin when it receives an event.
	SetEventHandler(func(action *things.ThingValue))

	// Subscribe to events from agents and things.
	// Events matching the agentID, thingID and/or eventKey invoke the handler set with SetEventHandler.
	//
	//	agentID is the ID of the device or service publishing the event, or "" for any agent.
	//	thingID is the ID of the Thing whose events to receive, or "" for any Things.
	//	eventName is the name of the event, or "" for any event
	Subscribe(agentID string, thingID string, eventName string) error
}
