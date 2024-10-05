package embedded

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
)

// EmbeddedClient is a hub client that connects directly to the embedded protocol binding.
// It can send messages to the hub and subscribe to actions and events from the hub.
//
// This implements the IHubClient interface for compatibility reasons so it can be
// a drop-in replacement for services that use other transports.
//
// Since embedded clients are always connected, the Connect and Disconnect methods do nothing,
// While publishing action/rpc request always return 'completed' results.
type EmbeddedClient struct {
	// The connected client/agent
	clientID string
	// sendMessage data from the client to the protocol binding server
	// note that internally, message data is wrapped in a ThingMessage object.
	// the wire format that serializes the data doesn't apply here.
	sendMessage hubclient.MessageHandler
	// client side handler that receives actions from the server
	//messageHandler hubclient.MessageHandler
	// client side handler that receives all non-action messages from the server
	receiveEventHandler hubclient.EventHandler
}

// ConnectWithClientCert always succeeds as a direct connection doesn't need a certificate
func (cl *EmbeddedClient) ConnectWithClientCert(kp keys.IHiveKey, cert *tls.Certificate) error {
	return nil
}

// ConnectWithPassword always succeeds as a direct connection doesn't need a password
func (cl *EmbeddedClient) ConnectWithPassword(password string) (string, error) {
	return "dummytoken", nil
}

// ConnectWithToken always succeeds as a direct connection doesn't need a token
func (cl *EmbeddedClient) ConnectWithToken(token string) (string, error) {
	return "dummytoken", nil
}

// ClientID returns the client's connection ID
func (cl *EmbeddedClient) ClientID() string {
	return cl.clientID
}
func (cl *EmbeddedClient) CreateKeyPair() (kp keys.IHiveKey) {
	return nil
}
func (cl *EmbeddedClient) Disconnect() {
}

// GetProtocolType returns the type of protocol this client supports
func (cl *EmbeddedClient) GetProtocolType() string {
	return "embed"
}
func (cl *EmbeddedClient) GetStatus() hubclient.TransportStatus {
	return hubclient.TransportStatus{
		ClientID:         cl.clientID,
		ConnectionStatus: hubclient.Connected,
	}
}

// HandleMessage receives a message from the embedded transport for this client
//func (cl *EmbeddedClient) HandleMessage(msg *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {
//	if msg.MessageType == vocab.MessageTypeAction || msg.MessageType == vocab.MessageTypeProperty {
//		if cl.messageHandler != nil {
//			return cl.messageHandler(msg)
//		}
//	} else {
//		if cl.receiveEventHandler != nil {
//			err := cl.receiveEventHandler(msg)
//			stat.Completed(msg, nil, err)
//			return stat
//		}
//	}
//	// The delivery is complete. Too bad the handler isn't registered. This is almost
//	// certainly a bug in the client code, so lets make clear this isn't a transport problem.
//	err := fmt.Errorf("no receive handler set for client '%s'", cl.clientID)
//	slog.Error("HandleMessage",
//		"err", err.Error(), "thingID", msg.ThingID, "name", msg.Name)
//	stat.Completed(msg, nil, err)
//	return stat
//}

func (cl *EmbeddedClient) Logout() error {
	return nil
}

// InvokeAction publishes an action request.
// Since this is a direct call, the response include a reply.
func (cl *EmbeddedClient) InvokeAction(
	thingID string, name string, data any) (stat hubclient.DeliveryStatus) {

	msg := hubclient.NewThingMessage(vocab.MessageTypeAction, thingID, name, data, cl.clientID)
	stat = cl.sendMessage(msg)
	return stat
}

// PubProperty publishes a configuration change request
func (cl *EmbeddedClient) PubProperty(thingID string, name string, data any) (stat hubclient.DeliveryStatus) {

	msg := hubclient.NewThingMessage(vocab.MessageTypeProperty, thingID, name, data, cl.clientID)
	stat = cl.sendMessage(msg)
	return stat
}

// PubEvent publishes an event style message without waiting for a response.
func (cl *EmbeddedClient) PubEvent(
	thingID string, name string, data any) error {

	msg := hubclient.NewThingMessage(vocab.MessageTypeEvent, thingID, name, data, cl.clientID)
	stat := cl.sendMessage(msg)
	if stat.Error != "" {
		return errors.New(stat.Error)
	}
	return nil
}

// PubProps publishes a properties map
func (cl *EmbeddedClient) PubProps(thingID string, props map[string]any) error {
	payload, _ := json.Marshal(props)
	return cl.PubEvent(thingID, vocab.EventNameProperties, string(payload))
}

// PubTD publishes an agent's TD
func (cl *EmbeddedClient) PubTD(thingID string, tdJSON string) error {
	return cl.PubEvent(thingID, vocab.EventNameTD, tdJSON)
}

// RefreshToken does nothing as tokens aren't used
func (cl *EmbeddedClient) RefreshToken(_ string) (newToken string, err error) {
	return "dummytoken", nil
}

// Rpc makes a RPC call using an action and waits for a delivery confirmation.
// The embedded client Rpc calls are synchronous so results are immediately available
func (cl *EmbeddedClient) Rpc(
	thingID string, name string, args interface{}, resp interface{}) error {

	// the internal wire format is a ThingMessage struct
	msg := hubclient.NewThingMessage(vocab.MessageTypeAction, thingID, name, args, cl.clientID)
	// this sendMessage is synchronous
	stat := cl.sendMessage(msg)
	// the internal response format is a DeliveryStatus struct
	if stat.Error == "" {
		// delivery might be completed but an unmarshal error causes it to fail
		err, _ := stat.Decode(resp)
		if err != nil {
			stat.Error = err.Error()
		}
	}
	if stat.Error != "" {
		return errors.New(stat.Error)
	}
	return nil
}

func (cl *EmbeddedClient) SendDeliveryUpdate(stat hubclient.DeliveryStatus) {
	slog.Info("SendDeliveryUpdate",
		slog.String("Progress", stat.Progress),
		slog.String("MessageID", stat.MessageID),
	)
	statJSON, _ := json.Marshal(&stat)
	// thing
	_ = cl.PubEvent(digitwin.InboxDThingID, vocab.EventNameDeliveryUpdate, string(statJSON))
}

// SendOperation is temporary transition to support using TD forms
func (cl *EmbeddedClient) SendOperation(
	href string, op tdd.Form, data any) (stat hubclient.DeliveryStatus) {

	slog.Info("SendOperation", "href", href, "op", op)
	return stat
}

// SetConnectHandler does nothing as connection is always established
func (cl *EmbeddedClient) SetConnectHandler(cb func(status hubclient.TransportStatus)) {
	return
}

// SetMessageHandler set the handler that receives all messages.
//func (cl *EmbeddedClient) SetMessageHandler(cb hubclient.MessageHandler) {
//	cl.messageHandler = cb
//}

// Subscribe adds a subscription for one or more events. Events will be passed to
// the handler set with SetMessageHandler.
//
// Actions directed at this client are automatically passed in. No need to subscribe.
//
// This is pretty coarse grained.
// Subscriptions remain in effect when the connection with the messaging server is interrupted.
//
//	thingID is the ID of the Thing whose events to receive or "" for events from all things
//	name is the event type to receive or "" for any event type
func (cl *EmbeddedClient) Subscribe(thingID string, name string) error {
	return fmt.Errorf("not implemented")
}

// Unsubscribe removes a previous event subscription.
// No more events or requests will be received after Unsubscribe.
func (cl *EmbeddedClient) Unsubscribe(thingID string, name string) error {
	return fmt.Errorf("UnsubscribeEvent not implemented")
}

// NewEmbeddedClient returns an embedded hub client for connecting to embedded services.
// This implements the IHubClient interface.
//
// The easiest way to connect this client to the server is to use the embedded server
// NewClient() method instead of this method as it links client and server.
//
// This supports a direct call from the client to the service. Basically the equivalent of
// a direct wire.
//
// serverHandler is the server side receiver needed to pass messages from the client to
// the embedded server. Without it, messages cannot reach the server. To receive messages,
// the server has to register this client.
//
// Intended for testing and clients that are also embedded, such as services calling other
// services.
func NewEmbeddedClient(clientID string) *EmbeddedClient {
	cl := EmbeddedClient{
		clientID: clientID,
		//sendMessage: serverHandler,
	}
	return &cl
}
