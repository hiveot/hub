package api

import (
	transports2 "github.com/hiveot/hub/transports"
)

// ActionStatusHandler is the handler for return an action progress to sender.
// Used by router HandleActionResponse to send the action result to the sender.
type ActionStatusHandler func(stat transports2.RequestStatus, agentID string) error

type HandleMessage func(msg *transports2.ThingMessage)

// IDigitwinRouter is the interface for routing the action,event and property messages
// received from consumers and agents. It handles the flow for TD level operations.
type IDigitwinRouter interface {
	// HandleMessage handles updates from agents
	//HandleMessage(msg *transports.ThingMessage)

	// HandleRequest handles action and property write requests from consumers and agents
	// replyTo is the client connection-id to reply to
	//HandleRequest(request *transports.ThingMessage, replyTo string) (stat transports.RequestStatus)

	HandleMessage(msg *transports2.ThingMessage, replyTo transports2.IServerConnection)
}
