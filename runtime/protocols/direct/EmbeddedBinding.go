package direct

import (
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/embedded"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
)

// EmbeddedBinding is a singleton for handling messages for embedded service agents
type EmbeddedBinding struct {
	// map of session handlers by agent IDs
	// Used in SendEvent and SendToClient
	handlers map[string]api.MessageHandler

	// ingress handler of messages sent by connected embedded clients
	// this simply sends them straight to the protocol manager which passes it on to
	// the middleware and the digitwin service.
	handleMessageFromClient api.MessageHandler
}

// AddService adds a handler for an embedded service agents.
// All services must be registered before starting the binding.
//func (svc *EmbeddedBinding) AddService(agentID string, handler api.MessageHandler) {
//	svc.handlers[agentID] = handler
//}

// GetProtocolInfo returns information on the protocol provided by the binding.
// This binding is only for embedded services to pub/sub events and actions.
func (svc *EmbeddedBinding) GetProtocolInfo() api.ProtocolInfo {
	inf := api.ProtocolInfo{}
	return inf
}

// SendToClient sends a request to an embedded client.
// Embedded clients are guaranteed to receive the message.
// This returns an error if the message cannot be delivered.
func (svc *EmbeddedBinding) SendToClient(
	clientID string, action *things.ThingMessage) (stat api.DeliveryStatus, found bool) {

	handler, found := svc.handlers[clientID]
	if found {
		stat = handler(action)
	} else {
		stat.Failed(action, fmt.Errorf("SendToClient: unknown client: %s", clientID))
	}
	return stat, found
}

// SendEvent publishes an event message to all subscribers of this protocol binding
func (svc *EmbeddedBinding) SendEvent(event *things.ThingMessage) (stat api.DeliveryStatus) {
	for _, agent := range svc.handlers {
		stat = agent(event)
	}
	return
}

// NewClient create a new messaging client that is already connected to the protocol server.
// It is directly for use by embedded services.
func (svc *EmbeddedBinding) NewClient(agentID string) hubclient.IHubClient {
	// the transport sends messages from client to this binding
	cl := embedded.NewEmbeddedClient(agentID, svc.handleMessageFromClient)

	// to send messages from binding to client it must be registered
	svc.handlers[agentID] = cl.ReceiveMessage
	return cl
}

// Start the protocol binding
//
//	handler to pass incoming messages
func (svc *EmbeddedBinding) Start(handler api.MessageHandler) error {
	svc.handleMessageFromClient = handler
	return nil
}

// Stop the protocol binding
func (svc *EmbeddedBinding) Stop() {
	//
}

// NewEmbeddedBinding creates a new instance of the embedded services binding
// intended for use by services like authn, authz, digitwin directory/inbox/outbox
func NewEmbeddedBinding() *EmbeddedBinding {
	svc := &EmbeddedBinding{
		handlers: make(map[string]api.MessageHandler),
	}
	return svc
}
