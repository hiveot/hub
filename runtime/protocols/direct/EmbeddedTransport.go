package direct

import (
	"crypto/tls"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
)

// EmbeddedTransport is a hub transport that connects directly to the embedded protocol binding
// It can send messages to the hub and subscribe to actions and events from the hub.
//
// This implements the IHubTransport interface for compatibility reasons so it can be
// a drop-in replacement for services that use other transports.
// The following methods are implemented:
//
//	PubAction, PubEvent,
//	Subscribe,
//	SetEventHandler, SetRequestHandler,
type EmbeddedTransport struct {
	// The connected client/agent
	clientID string
	// sendMessage from the client to the protocol binding server
	sendMessage api.MessageHandler
	// handler of receiving messages from the protocol server for the client
	receiveMessage api.MessageHandler
}

// ConnectWithCert always succeeds as a direct connection doesn't need a certificate
func (tp *EmbeddedTransport) ConnectWithCert(kp keys.IHiveKey, cert *tls.Certificate) (token string, err error) {
	return "", nil
}

// ConnectWithPassword always succeeds as a direct connection doesn't need a password
func (tp *EmbeddedTransport) ConnectWithPassword(password string) error {
	return nil
}

// ConnectWithJWT always succeeds as a direct connection doesn't need a token
func (tp *EmbeddedTransport) ConnectWithJWT(token string) error {
	return nil
}
func (tp *EmbeddedTransport) CreateKeyPair() (kp keys.IHiveKey) {
	return nil
}
func (tp *EmbeddedTransport) Disconnect() {
}
func (tp *EmbeddedTransport) GetStatus() transports.HubTransportStatus {
	return transports.HubTransportStatus{
		ClientID:         tp.clientID,
		ConnectionStatus: transports.Connected,
	}
}

// ReceiveMessage receives a message from the server for this client
func (tp *EmbeddedTransport) ReceiveMessage(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	if tp.receiveMessage != nil {
		return tp.receiveMessage(msg)
	}
	stat.Error = fmt.Sprintf("no receive handler set for client '%s'", tp.clientID)
	stat.Status = api.DeliveryFailed
	return stat
}

// PubAction publishes an action request and waits for a response.
func (tp *EmbeddedTransport) PubAction(thingID string, key string, payload []byte) api.DeliveryStatus {
	msg := things.NewThingMessage(vocab.MessageTypeAction, thingID, key, payload, tp.clientID)
	resp := tp.sendMessage(msg)
	return resp
}

// PubEvent publishes an event style message without waiting for a response.
func (tp *EmbeddedTransport) PubEvent(thingID string, key string, payload []byte) api.DeliveryStatus {
	msg := things.NewThingMessage(vocab.MessageTypeEvent, thingID, key, payload, tp.clientID)
	resp := tp.sendMessage(msg)
	return resp
}

// SetConnectHandler does nothing as connection is always established
func (tp *EmbeddedTransport) SetConnectHandler(cb func(status transports.HubTransportStatus)) {
	return
}

// SetMessageHandler set the handler that receives all subscribed messages.
// Use 'Subscribe' to set the type of events and actions to receive
func (tp *EmbeddedTransport) SetMessageHandler(cb api.MessageHandler) {
	tp.receiveMessage = cb
}

// Subscribe adds a subscription for one or more events. Events will be passed to the
// receiveMessage handler.
// Messages directed at this client are automatically passed in. No need to subscribe.
//
// This is pretty coarse grained.
// Subscriptions remain in effect when the connection with the messaging server is interrupted.
//
//	thingID is the ID of the Thing whose events to receive or "" for events from all things
func (tp *EmbeddedTransport) Subscribe(thingID string) error {
	return fmt.Errorf("not implemented")
}

// Unsubscribe removes a previous event subscription.
// No more events or requests will be received after Unsubscribe.
func (tp *EmbeddedTransport) Unsubscribe(thingID string) {
	//
}

// NewEmbeddedTransport creates a new transport for use by clients.
//
//		clientID is the client of the transport
//		handler is the handler to send messages from clients (PubEvent and PubAction) to server
//	 receiver is the handler that receives messages send by the protocol binding.
func NewEmbeddedTransport(clientID string, handler api.MessageHandler) *EmbeddedTransport {
	tp := EmbeddedTransport{
		clientID:    clientID,
		sendMessage: handler,
	}
	return &tp
}
