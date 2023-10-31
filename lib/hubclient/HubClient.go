package hubclient

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/hubclient/transports/mqtttransport"
	"github.com/hiveot/hub/lib/hubclient/transports/natstransport"
	"github.com/hiveot/hub/lib/ser"
	"github.com/hiveot/hub/lib/thing"
	"github.com/hiveot/hub/lib/vocab"
	"log/slog"
	"os"
	"path"
	"strings"
	"time"
)

// TokenFileExt defines the filename extension under which client tokens are stored
// in the keys directory.
const TokenFileExt = ".token"

// KPFileExt defines the filename extension under which public/private keys are stored
// in the keys directory.
const KPFileExt = ".key"

// PubKeyFileExt defines the filename extension under which public key is stored
// in the keys directory.
const PubKeyFileExt = ".pub"

// HubClient wrapper around the underlying message bus transport.
type HubClient struct {
	//serverURL string
	//caCert    *x509.Certificate
	clientID  string
	transport transports.IHubTransport
	//serializedKP string
}

// MakeAddress creates a message address optionally with wildcards
// This uses the hiveot topic format: {msgType}/{deviceID}/{thingID}/{name}[/{clientID}]
// Where '/' is the address separator for MQTT or '.' for Nats
// Where "+" is the wildcard for MQTT or "*" for Nats
//
//	msgType is the message type: "event", "action", "config" or "rpc".
//	agentID is the device or service being addressed. Use "" for wildcard
//	thingID is the ID of the thing managed by the publisher. Use "" for wildcard
//	name is the event or action name. Use "" for wildcard.
//	clientID is the login ID of the sender. Use "" for subscribe.
func (hc *HubClient) MakeAddress(msgType, agentID, thingID, name string, clientID string) string {
	sep, wc, rem := hc.transport.AddressTokens()

	parts := make([]string, 0, 5)
	if msgType == "" {
		msgType = vocab.MessageTypeEvent
	}
	parts = append(parts, msgType)
	if agentID == "" {
		agentID = wc
	}
	parts = append(parts, agentID)
	if thingID == "" {
		thingID = wc
	}
	parts = append(parts, thingID)
	if name == "" {
		name = wc
	}
	parts = append(parts, name)
	if clientID == "" {
		clientID = rem
	}
	parts = append(parts, clientID)

	addr := strings.Join(parts, sep)
	return addr
}

// SplitAddress separates an address into its components
//
// addr is a hiveot address eg: msgType/things/deviceID/thingID/name/clientID
func (hc *HubClient) SplitAddress(addr string) (msgType, agentID, thingID, name string, senderID string, err error) {

	sep, _, _ := hc.transport.AddressTokens()
	parts := strings.Split(addr, sep)

	// inbox topics are short
	if len(parts) >= 1 && parts[0] == "_INBOX" {
		msgType = parts[0]
		if len(parts) >= 2 {
			agentID = parts[1]
		}
		return
	}
	if len(parts) < 4 {
		err = errors.New("incomplete topic")
		return
	}
	msgType = parts[0]
	agentID = parts[1]
	thingID = parts[2]
	name = parts[3]
	if len(parts) > 4 {
		senderID = parts[4]
	}
	return
}

// ClientID the client is authenticated as to the server
func (hc *HubClient) ClientID() string {
	return hc.clientID
}

// ConnectWithToken connects to the Hub server using a user JWT credentials secret
// The token clientID must match that of the client
//
//	 kp is the serialized public/private key-pair of this client
//		jwtToken is the token obtained with login or refresh.
func (hc *HubClient) ConnectWithToken(kp, jwtToken string) error {

	err := hc.transport.ConnectWithToken(kp, jwtToken)
	return err
}

// ConnectWithTokenFile is a convenience function to read token and key
// from file and connect to the server.
//
// keysDir is the directory with the {clientID}.key and {clientID}.token files.
func (hc *HubClient) ConnectWithTokenFile(keysDir string) error {
	var keyStr []byte

	slog.Info("ConnectWithTokenFile",
		slog.String("keysDir", keysDir),
		slog.String("clientID", hc.clientID))
	keyFile := path.Join(keysDir, hc.clientID+KPFileExt)
	tokenFile := path.Join(keysDir, hc.clientID+TokenFileExt)
	token, err := os.ReadFile(tokenFile)
	if err == nil && keyFile != "" {
		keyStr, err = os.ReadFile(keyFile)
	}
	if err != nil {
		return fmt.Errorf("ConnectWithTokenFile failed: %w", err)
	}
	err = hc.transport.ConnectWithToken(string(keyStr), string(token))
	return err
}

// ConnectWithPassword connects to the Hub server using a login ID and password.
func (hc *HubClient) ConnectWithPassword(password string) error {
	err := hc.transport.ConnectWithPassword(password)
	return err
}

// CreateKeyPair create a new serialized public/private key pair for use by this client
func (hc *HubClient) CreateKeyPair() (string, string) {
	return hc.transport.CreateKeyPair()
}

// Disconnect the client from the hub and unsubscribe from all topics
func (hc *HubClient) Disconnect() {
	hc.transport.Disconnect()
}

// LoadCreateKeyPair loads or creates a public/private key pair using the clientID as filename.
//
//	The key-pair is named {clientID}.key, the public key {clientID}.pub
//
//	clientID is the clientID to use, or "" to use the connecting ID
//	keysDir is the location where the keys are stored.
//
// This returns the serialized private and pub keypair, or an error.
func (hc *HubClient) LoadCreateKeyPair(clientID, keysDir string) (serializedKP string, pubKey string, err error) {
	if keysDir == "" {
		return "", "", fmt.Errorf("certs directory must be provided")
	}
	if clientID == "" {
		clientID = hc.clientID
	}
	keyFile := path.Join(keysDir, clientID+KPFileExt)
	pubFile := path.Join(keysDir, clientID+PubKeyFileExt)
	// load key from file
	keyBytes, err := os.ReadFile(keyFile)
	pubBytes, err2 := os.ReadFile(pubFile)

	if err != nil || err2 != nil {
		// no keyfile, create the key
		serializedKP, pubKey = hc.transport.CreateKeyPair()

		// save the key for future use
		err = os.WriteFile(keyFile, []byte(serializedKP), 0400)
		err2 = os.WriteFile(pubFile, []byte(pubKey), 0444)
		if err2 != nil {
			err = err2
		}
	} else {
		serializedKP = string(keyBytes)
		pubKey = string(pubBytes)
	}

	return serializedKP, pubKey, err
}

// PubAction publishes a request for action from a Thing.
//
//	agentID of the device or service that handles the action.
//	thingID is the destination thingID to whom the action applies.
//	name is the name of the action as described in the Thing's TD
//	payload is the optional payload of the action as described in the Thing's TD
//
// This returns the reply data or an error if an error was returned or no reply was received
func (hc *HubClient) PubAction(
	agentID string, thingID string, name string, payload []byte) ([]byte, error) {

	addr := hc.MakeAddress(vocab.MessageTypeAction, agentID, thingID, name, hc.clientID)
	slog.Info("PubAction", "addr", addr)
	data, err := hc.transport.PubRequest(addr, payload)
	return data, err
}

// PubConfig publishes a Thing configuration change request
//
// The client's ID is used as the publisher ID of the action.
//
//		agentID of the device that handles the action for the thing or service capability
//		thingID is the destination thingID that handles the action
//	 propName is the ID of the property to change as described in the TD properties section
//	 payload is the optional payload of the action as described in the Thing's TD
//
// This returns the reply data or an error if an error was returned or no reply was received
func (hc *HubClient) PubConfig(
	agentID string, thingID string, propName string, payload []byte) ([]byte, error) {

	addr := hc.MakeAddress(vocab.MessageTypeConfig, agentID, thingID, propName, hc.clientID)
	slog.Info("PubConfig", "addr", addr)
	data, err := hc.transport.PubRequest(addr, payload)
	return data, err
}

// PubEvent publishes a Thing event. The payload is an event value as per TD document.
// Intended for devices and services to notify of changes to the Things they are the agent for.
//
// 'thingID' is the ID of the 'thing' whose event to publish. This is the ID under which the
// TD document is published that describes the thing. It can be the ID of the sensor, actuator
// or service.
//
// This will use the client's ID as the agentID of the event.
// eventName is the ID of the event described in the TD document 'events' section,
// or one of the predefined events listed above as EventIDXyz
//
//	thingID of the Thing whose event is published
//	eventName is one of the predefined events as described in the Thing TD
//	payload is the serialized event value, or nil if the event has no value
func (hc *HubClient) PubEvent(thingID string, eventName string, payload []byte) error {

	addr := hc.MakeAddress(vocab.MessageTypeEvent, hc.clientID, thingID, eventName, hc.clientID)
	slog.Info("PubEvent", "addr", addr)
	err := hc.transport.Pub(addr, payload)
	return err
}

// PubRPCRequest publishes an RPC request to a service and waits for a response.
// Intended for users and services to invoke RPC to services.
//
// Authorization to use the service capability can depend on the user's role. Check the service
// documentation for details. When unauthorized then an error will be returned after a short delay.
//
// The client's ID is used as the senderID of the rpc request.
//
//	agentID of the service that handles the request
//	capability is the capability to invoke
//	methodName is the name of the request method to invoke
//	req is the request message that will be marshalled or nil if no arguments are expected
//	resp is the expected response message that is unmarshalled, or nil if no response is expected
func (hc *HubClient) PubRPCRequest(
	agentID string, capability string, methodName string, req interface{}, resp interface{}) error {

	var payload []byte
	if req != nil {
		payload, _ = ser.Marshal(req)
	}
	addr := hc.MakeAddress(vocab.MessageTypeRPC, agentID, capability, methodName, hc.clientID)
	slog.Info("PubRPCRequest", "addr", addr)

	data, err := hc.transport.PubRequest(addr, payload)
	if err == nil && resp != nil {
		err = ser.Unmarshal(data, resp)
	}
	return err
}

// PubTD publishes an event with a Thing TD document.
// The client's authentication ID will be used as the agentID of the event.
func (hc *HubClient) PubTD(td *thing.TD) error {
	payload, _ := ser.Marshal(td)
	addr := hc.MakeAddress(vocab.MessageTypeEvent, hc.clientID, td.ID, vocab.EventNameTD, hc.clientID)
	slog.Info("PubTD", "addr", addr)
	err := hc.transport.Pub(addr, payload)
	return err
}

// SubActions subscribes to actions requested of this client's Things.
// Intended for use by devices or services to receive requests for its things.
//
// The handler receives an action request message with request payload and returns
// an optional reply or an error when the request wasn't accepted.
//
// The supported actions are defined in the TD document of the things this binding has published.
//
//	thingID is the device thing or service capability to subscribe to, or "" for wildcard
//	cb is the callback to invoke
//
// The handler receives an action request message with request payload and
// must return a reply payload, which can be nil, or return an error.
func (hc *HubClient) SubActions(thingID string,
	handler func(msg *thing.ThingValue) (result []byte, err error)) (transports.ISubscription, error) {

	subAddr := hc.MakeAddress(vocab.MessageTypeAction, hc.clientID, thingID, "", "")

	sub, err := hc.transport.SubRequest(subAddr,
		func(addr string, payload []byte) (result []byte, err error) {

			messageType, agentID, thingID, name, senderID, err := hc.SplitAddress(addr)
			msg := thing.NewThingValue(messageType, agentID, thingID, name, payload, senderID)
			msg.SenderID = senderID
			msg.ValueType = messageType
			if msg.SenderID == "" || err != nil {
				err = fmt.Errorf("SubActions: Received request on invalid address '%s'", addr)
				slog.Warn(err.Error())
				return nil, err
			}
			result, err = handler(msg)
			return result, err
		})
	return sub, err
}

// SubConfig subscribes to configuration change requested of this client's Things.
// Intended for use by devices to receive configuration requests for its things.
// The device's agentID is the ID used to authenticate with the server, eg, this clientID.
//
// The handler receives an action request message with request payload and returns
// an optional reply or an error when the request wasn't accepted.
//
// The supported properties are defined in the TD document of the things this binding has published.
//
//	thingID is the device thing or service capability to subscribe to, or "" for wildcard
//	cb is the callback to invoke
//
// The handler receives an action request message with request payload and
// must reply with msg.Reply or msg.Ack, or return an error
func (hc *HubClient) SubConfig(thingID string, handler func(msg *thing.ThingValue) error) (
	transports.ISubscription, error) {

	subAddr := hc.MakeAddress(vocab.MessageTypeConfig, hc.clientID, thingID, "", "")

	sub, err := hc.transport.SubRequest(subAddr,
		func(addr string, payload []byte) (reply []byte, err error) {

			messageType, agentID, thingID, name, senderID, err := hc.SplitAddress(addr)

			msg := thing.NewThingValue(messageType, agentID, thingID, name, payload, senderID)
			if msg.SenderID == "" || err != nil {
				err = fmt.Errorf("SubConfig: Received request on invalid address '%s'", addr)
				slog.Warn(err.Error())
				return nil, err
			}
			err = handler(msg)
			return nil, err
		})
	return sub, err

}

// SubEvents subscribes to events from a device or service.
//
// Events are passed to the handler in sequential order after the handler returns.
// For parallel processing, create a goroutine to handle the event and return
// immediately.
//
//		agentID is the ID of the device or service publishing the event, or "" for any agent.
//		thingID is the ID of the Thing whose events to receive, or "" for any Things.
//	 eventName is the name of the event, or "" for any event
//
// The handler receives an event value message with data payload.
func (hc *HubClient) SubEvents(agentID string, thingID string, eventName string,
	handler func(msg *thing.ThingValue)) (transports.ISubscription, error) {

	subAddr := hc.MakeAddress(vocab.MessageTypeEvent, agentID, thingID, eventName, "")

	sub, err := hc.transport.Sub(subAddr, func(addr string, payload []byte) {

		_, evAgentID, evThingID, name, _, err := hc.SplitAddress(addr)
		if err != nil {
			slog.Warn("SubEvents: Ignored event on invalid address", "addr", addr)
			return
		}
		eventMsg := &thing.ThingValue{
			AgentID:     evAgentID,
			ThingID:     evThingID,
			Name:        name,
			Data:        payload,
			CreatedMSec: time.Now().UnixMilli(),
		}
		handler(eventMsg)
	})
	return sub, err
}

// SubRPCRequest subscribes a client to receive RPC capability method request.
// Intended for use by services to receive requests for its capabilities.
//
// The capabilityID identifies the interface that is supported. Each
// capability represents one or more methods that are identified by the
// actionName. This is similar to the thingID in events and actions.
//
// The handler must reply with msg.Reply or msg.Ack, or return an error.
func (hc *HubClient) SubRPCRequest(capabilityID string,
	handler func(msg *thing.ThingValue) (reply []byte, err error)) (
	transports.ISubscription, error) {

	subAddr := hc.MakeAddress(vocab.MessageTypeRPC, hc.clientID, capabilityID, "", "")
	sub, err := hc.transport.SubRequest(subAddr,
		func(addr string, payload []byte) (reply []byte, err error) {

			messageType, agentID, thingID, name, senderID, err := hc.SplitAddress(addr)
			msg := thing.NewThingValue(messageType, agentID, thingID, name, payload, senderID)
			if senderID == "" || err != nil {
				err = fmt.Errorf("SubRPCRequest: Received request on invalid address '%s'", addr)
				slog.Warn(err.Error())
				return nil, err
			}
			return handler(msg)
		})
	return sub, err
}

// SubRPCCapability registers RPC capability and handler methods.
// This returns a subscription object.
//
// Intended for implementing RPC handlers by services.
//
// The typical RPC handler is a method that takes a context, an optional single argument
// and returns a result and error status. Supported formats are:
//
//	func()(error)
//	func()(resp interface{}, error)
//	func(ctx *ServiceContext, args interface{})(resp interface{}, error)
//	func(ctx *ServiceContext, args interface{}) error
//	func(ctx *ServiceContext)(resp interface{}, error)
//
// Requests from senders are passed sequentially or concurrently, depending
// on the underlying implementation.
//
// Where:
//
//	ctx is a service context which contains the callerID and other information
//	  about the sender, if known.
//	args and resp is defined can be any serializable type. This uses the serializer
//	  defined in utils/ser. The default is json.
//
// Arguments:
//
//	capID is the capability name (equivalent to thingID) to register.
//	capMethods maps method names to their implementation
func (hc *HubClient) SubRPCCapability(
	capID string, capMethods map[string]interface{}) (transports.ISubscription, error) {

	sub, err := hc.SubRPCRequest(capID, func(msg *thing.ThingValue) (reply []byte, err error) {

		capMethod, found := capMethods[msg.Name]
		if !found {
			err = fmt.Errorf("method '%s' not part of capability '%s'", msg.Name, capID)
			slog.Warn("SubRPCCapability; unknown method", "methodName", msg.Name, "senderID", msg.SenderID)
			return nil, err
		}
		ctx := ServiceContext{
			Context:  context.Background(),
			SenderID: msg.SenderID,
		}
		respData, err := HandleRequestMessage(ctx, capMethod, msg.Data)
		return respData, err
	})
	return sub, err
}

// NewHubClientFromTransport returns a new Hub Client instance for the given transport.
//
//   - message bus transport to use, eg NatsTransport or MqttTransport instance
//   - clientID of the client that will be connecting
func NewHubClientFromTransport(transport transports.IHubTransport, clientID string) *HubClient {
	hc := HubClient{
		clientID:  clientID,
		transport: transport,
	}
	return &hc
}

// NewHubClient returns a new Hub Client instance
//
// The keyPair string is optional. If not provided a new set of keys will be created.
// Use GetKeyPair to retrieve it for saving to file.
//
// Invoke hubClient.ConnectWithXy() to connect
//
//   - url of server to connect to.
//   - clientID of the client that will be connecting
//   - keyPair is this client's serialized private/public key pair, or "" to create them.
//   - caCert of server or nil to not verify server cert
//   - core server to use, "nats" or "mqtt". Default "" will use nats if url starts with "nats" or mqtt otherwise.
func NewHubClient(url string, clientID string, caCert *x509.Certificate, core string) *HubClient {
	// a kp is not needed when using connect with token file
	//if kp == nil {
	//	panic("kp is required")
	//}
	var tp transports.IHubTransport
	if core == "nats" || strings.HasPrefix(url, "nats") {
		tp = natstransport.NewNatsTransport(url, clientID, caCert)
	} else {
		tp = mqtttransport.NewMqttTransport(url, clientID, caCert)
	}
	hc := NewHubClientFromTransport(tp, clientID)
	return hc
}
