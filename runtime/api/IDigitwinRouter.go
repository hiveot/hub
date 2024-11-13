package api

import (
	"github.com/hiveot/hub/lib/hubclient"
)

// RequestProgressHandler is the handler for return an action progress to sender.
// Used by router HandlePublishRequestProgress to send the action result to the sender.
type RequestProgressHandler func(stat hubclient.RequestStatus, agentID string) error

type HandleMessage func(msg *hubclient.ThingMessage)

// IDigitwinRouter is the interface for routing the action,event and property messages
// received from consumers and agents. It handles the flow for TD level operations.
type IDigitwinRouter interface {
	// HandleMessage handles updates from agents
	HandleMessage(msg *hubclient.ThingMessage)

	// HandleRequest handles action and property write requests from consumers and agents
	// replyTo is the client connection-id to reply to
	HandleRequest(request *hubclient.ThingMessage, replyTo string) (stat hubclient.RequestStatus)
}
