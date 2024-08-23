package embedded

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/embedded"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/teris-io/shortid"
)

// EmbeddedTransport is a singleton for transporting messages for embedded service agents
type EmbeddedTransport struct {
	// map of incoming message handlers for each agent by agentID
	// Used in SendEvent and SendToClient
	handlers map[string]hubclient.MessageHandler

	// ingress handler of messages sent by connected embedded clients
	// this simply sends them straight to the protocol manager which passes it on to
	// the middleware and the digitwin service.
	handleMessageFromClient hubclient.MessageHandler
}

// AddTDForms does not apply to the embedded service
func (svc *EmbeddedTransport) AddTDForms(td *tdd.TD) {
	// nothing to do here
}

// GetProtocolInfo returns information on the protocol provided by the binding.
// This binding is only for embedded services to pub/sub events and actions.
func (svc *EmbeddedTransport) GetProtocolInfo() api.ProtocolInfo {
	inf := api.ProtocolInfo{}
	return inf
}

// receive a message from a client and ensure it has a message ID
// embedded transport use a 'e-' messageID prefix for troubleshooting
func (svc *EmbeddedTransport) handleMessage(msg *hubclient.ThingMessage) hubclient.DeliveryStatus {
	if msg.MessageID == "" {
		msg.MessageID = "e-" + shortid.MustGenerate()
	}
	return svc.handleMessageFromClient(msg)
}

// NewClient create a new messaging client that is already connected to the protocol server.
// It is directly for use by embedded services.
func (svc *EmbeddedTransport) NewClient(agentID string) hubclient.IHubClient {
	// the transport sends messages from client to this binding
	cl := embedded.NewEmbeddedClient(agentID, svc.handleMessage)

	// to send messages from binding to client it must be registered
	svc.handlers[agentID] = cl.HandleMessage
	return cl
}

// SendEvent publishes an event message to all subscribers of this protocol binding
// TODO: currently the embedded services don't send events
func (svc *EmbeddedTransport) SendEvent(event *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {
	stat.Failed(event, errors.New("no handlers for event"))

	for agentID, agent := range svc.handlers {
		// TODO: until subscription is needed by embedded clients, simply don't send them any events.
		// don't send events back to the sender
		if agentID != event.SenderID {

			// authn and authz  dont need to subscribe to events
			// embedded client will still receive directed events.
			_ = agent
			//stat2 := agent(event)
			//if stat2.Error == "" {
			//	stat = stat2
			//}
		}
	}
	return stat
}

// SendToClient sends a request to an embedded client.
// Embedded clients are guaranteed to receive the message.
// This returns an error if the message cannot be delivered.
func (svc *EmbeddedTransport) SendToClient(
	clientID string, msg *hubclient.ThingMessage) (stat hubclient.DeliveryStatus, found bool) {

	handler, found := svc.handlers[clientID]
	if found {
		stat = handler(msg)
	} else {
		err := fmt.Errorf("SendToClient: unknown client: %s", clientID)
		stat.Failed(msg, err)
	}
	return stat, found
}

// Start the protocol binding
//
//	handler to pass incoming messages
func (svc *EmbeddedTransport) Start(handler hubclient.MessageHandler) error {
	svc.handleMessageFromClient = handler
	return nil
}

// Stop the protocol binding
func (svc *EmbeddedTransport) Stop() {
	//
}

// NewEmbeddedBinding creates a new instance of the embedded services binding
// intended for use by services like authn, authz, digitwin directory/inbox/outbox
func NewEmbeddedBinding() *EmbeddedTransport {
	svc := &EmbeddedTransport{
		handlers: make(map[string]hubclient.MessageHandler),
	}
	return svc
}
