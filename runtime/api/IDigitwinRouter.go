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
	// HandleMessage with the message to handle.
	// If a result is immediately available it is returned, otherwise it is sent
	// separately to the connection with the ID of replyTo
	HandleMessage(msg *transports2.ThingMessage, replyTo string) (
		completed bool, output any, err error)
}
