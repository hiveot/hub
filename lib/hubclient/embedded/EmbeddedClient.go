package embedded

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
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
	// sendMessage from the client to the protocol binding server
	sendMessage api.MessageHandler
	// client side handler that receives actions from the server
	receiveActionHandler api.MessageHandler
	// client side handler that receives all non-action messages from the server
	receiveEventHandler api.EventHandler
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

func (cl *EmbeddedClient) GetStatus() hubclient.TransportStatus {
	return hubclient.TransportStatus{
		ClientID:         cl.clientID,
		ConnectionStatus: hubclient.Connected,
	}
}

func (cl *EmbeddedClient) Logout() error {
	return nil
}

// ReceiveMessage receives a message from the server for this client
func (cl *EmbeddedClient) ReceiveMessage(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	if msg.MessageType == vocab.MessageTypeAction {
		if cl.receiveActionHandler != nil {
			return cl.receiveActionHandler(msg)
		}
	} else {
		if cl.receiveEventHandler != nil {
			err := cl.receiveEventHandler(msg)
			stat.Completed(msg, err)
			return stat
		}
	}
	// The delivery is complete. Too bad the handler isn't registered. This is almost
	// certainly a bug in the client code, so lets make clear this isn't a transport problem.
	err := fmt.Errorf("no receive handler set for client '%s'", cl.clientID)
	stat.Completed(msg, err)
	return stat
}

// PubAction publishes an action request.
// Since this is a direct call, the response include a reply.
func (cl *EmbeddedClient) PubAction(
	thingID string, key string, payload []byte) (stat api.DeliveryStatus) {

	msg := things.NewThingMessage(vocab.MessageTypeAction, thingID, key, payload, cl.clientID)
	stat = cl.sendMessage(msg)
	return stat
}

// PubConfig publishes a configuration change request
func (cl *EmbeddedClient) PubConfig(thingID string, key string, value string) (stat api.DeliveryStatus) {
	props := map[string]string{key: value}
	propsJson, _ := json.Marshal(props)
	return cl.PubAction(thingID, vocab.ActionTypeProperties, propsJson)
}

// PubEvent publishes an event style message without waiting for a response.
func (cl *EmbeddedClient) PubEvent(
	thingID string, key string, payload []byte) error {

	msg := things.NewThingMessage(vocab.MessageTypeEvent, thingID, key, payload, cl.clientID)
	stat := cl.sendMessage(msg)
	if stat.Error != "" {
		return errors.New(stat.Error)
	}
	return nil
}

// PubProps publishes a properties map
func (cl *EmbeddedClient) PubProps(thingID string, props map[string]string) error {
	payload, _ := json.Marshal(props)
	return cl.PubEvent(thingID, vocab.EventTypeProperties, payload)
}

// PubTD publishes a TD event
func (cl *EmbeddedClient) PubTD(td *things.TD) error {
	payload, _ := json.Marshal(td)
	return cl.PubEvent(td.ID, vocab.EventTypeTD, payload)
}

// RefreshToken does nothing as tokens aren't used
func (cl *EmbeddedClient) RefreshToken(_ string) (newToken string, err error) {
	return "dummytoken", nil
}

// Rpc makes a RPC call using an action and waits for a delivery confirmation.
// The embedded client Rpc calls are synchronous so results are immediately available
func (cl *EmbeddedClient) Rpc(
	thingID string, key string, args interface{}, resp interface{}) error {

	payload, _ := json.Marshal(args)
	msg := things.NewThingMessage(vocab.MessageTypeAction, thingID, key, payload, cl.clientID)
	// this sendMessage is synchronous
	stat := cl.sendMessage(msg)
	if stat.Error == "" && stat.Reply != nil {
		// delivery might be completed but an unmarshal error causes it to fail
		err := json.Unmarshal(stat.Reply, resp)
		if err != nil {
			stat.Error = err.Error()
		}
	}
	if stat.Error != "" {
		return errors.New(stat.Error)
	}
	return nil
}

// SetConnectHandler does nothing as connection is always established
func (cl *EmbeddedClient) SetConnectHandler(cb func(status hubclient.TransportStatus)) {
	return
}

// SetActionHandler set the handler that receives all subscribed action messages.
func (cl *EmbeddedClient) SetActionHandler(cb api.MessageHandler) {
	cl.receiveActionHandler = cb
}

// SetEventHandler set the handler that receives all subscribed messages.
// Use 'Subscribe' to set the type of events and actions to receive
func (cl *EmbeddedClient) SetEventHandler(cb api.EventHandler) {
	cl.receiveEventHandler = cb
}

// Subscribe adds a subscription for one or more events. Events will be passed to
// the handler set with SetventHandler.
//
// Actions directed at this client are automatically passed in. No need to subscribe.
//
// This is pretty coarse grained.
// Subscriptions remain in effect when the connection with the messaging server is interrupted.
//
//	thingID is the ID of the Thing whose events to receive or "" for events from all things
//	key is the event type to receive or "" for any event type
func (cl *EmbeddedClient) Subscribe(thingID string, key string) error {
	return fmt.Errorf("not implemented")
}

// Unsubscribe removes a previous event subscription.
// No more events or requests will be received after Unsubscribe.
func (cl *EmbeddedClient) Unsubscribe(thingID string) error {
	return fmt.Errorf("Unsubscribe not implemented")
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
func NewEmbeddedClient(clientID string, serverHandler api.MessageHandler) *EmbeddedClient {
	cl := EmbeddedClient{
		clientID:    clientID,
		sendMessage: serverHandler,
	}
	return &cl
}

//func(
//	thingID string, method string, args interface{}, reply interface{}) error {
//
//	return func(thingID string, method string, args interface{}, reply interface{}) error {
//		data, _ := json.Marshal(args)
//		tv := things.NewThingMessage(vocab.MessageTypeAction, thingID, method, data, clientID)
//		stat := handler(tv)
//		if stat.Status == api.DeliveryCompleted && stat.Reply != nil {
//			err := json.Unmarshal(stat.Reply, &reply)
//			return err
//		} else if stat.Error != "" {
//			return errors.New(stat.Error)
//		}
//		return nil
//	}
//}

// WriteActionMessage is a convenience function to create an action ThingMessage and pass it to
// a handler for routing to its destination.
// This returns the reply data or an error.
//func WriteActionMessage(
//	thingID string, key string, data []byte, senderID string, handler api.MessageHandler) api.DeliveryStatus {
//	tv := things.NewThingMessage(vocab.MessageTypeAction, thingID, key, data, senderID)
//	return handler(tv)
//}
