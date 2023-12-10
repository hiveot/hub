package hubclient

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/hubclient/transports/mqtttransport"
	"github.com/hiveot/hub/lib/hubclient/transports/natstransport"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/ser"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/vocab"
	"log/slog"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
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
	clientID         string
	transport        transports.IHubTransport
	connectionStatus transports.ConnectionStatus
	connectionInfo   transports.ConnInfo
	// keep retrying connection on error (default true)
	retryConnect atomic.Bool

	// mux is used to protect access to handlers and capTable
	mux sync.RWMutex

	// capability map:
	//  map[capID] map[methodName]handler
	capTable map[string]map[string]interface{}

	actionHandler     func(msg *things.ThingValue) (reply []byte, err error)
	configHandler     func(msg *things.ThingValue) error
	connectionHandler func(status transports.ConnectionStatus, info transports.ConnInfo)
	eventHandler      func(msg *things.ThingValue)
	rpcHandler        func(msg *things.ThingValue) (reply []byte, err error)
}

// MakeAddress creates a message address optionally with wildcards
// This uses the hiveot address format: {msgType}/{deviceID}/{thingID}/{name}[/{clientID}]
// Where '/' is the address separator for MQTT or '.' for Nats
// Where "+" is the wildcard for MQTT or "*" for Nats
//
//	msgType is the message type: "event", "action", "config" or "rpc".
//	agentID is the device or service being addressed. Use "" for wildcard
//	thingID is the ID of the things managed by the publisher. Use "" for wildcard
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
	if len(parts) >= 1 && parts[0] == vocab.MessageTypeINBOX {
		msgType = parts[0]
		if len(parts) >= 2 {
			agentID = parts[1]
		}
		return
	}
	if len(parts) < 4 {
		err = errors.New("incomplete address")
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
//	kp is the serialized public/private key-pair of this client
//	jwtToken is the token obtained with login or refresh.
func (hc *HubClient) ConnectWithToken(kp keys.IHiveKey, jwtToken string) error {
	err := hc.transport.ConnectWithToken(kp, jwtToken)
	return err
}

// ConnectWithTokenFile is a convenience function to read token and key
// from file and connect to the server.
//
// keysDir is the directory with the {clientID}.key and {clientID}.token files.
func (hc *HubClient) ConnectWithTokenFile(keysDir string) error {
	var kp keys.IHiveKey

	slog.Info("ConnectWithTokenFile",
		slog.String("keysDir", keysDir),
		slog.String("clientID", hc.clientID))
	keyFile := path.Join(keysDir, hc.clientID+KPFileExt)
	tokenFile := path.Join(keysDir, hc.clientID+TokenFileExt)
	token, err := os.ReadFile(tokenFile)
	if err == nil && keyFile != "" {
		kp, err = keys.NewKeyFromFile(keyFile)
	}
	if err != nil {
		return fmt.Errorf("ConnectWithTokenFile failed: %w", err)
	}
	err = hc.transport.ConnectWithToken(kp, string(token))
	return err
}

// ConnectWithPassword connects to the Hub server using the clientID and password.
func (hc *HubClient) ConnectWithPassword(password string) error {
	err := hc.transport.ConnectWithPassword(password)
	return err
}

// CreateKeyPair create a new serialized public/private key pair for use by this client
func (hc *HubClient) CreateKeyPair() keys.IHiveKey {
	return hc.transport.CreateKeyPair()
}

// Disconnect the client from the hub and unsubscribe from all topics
func (hc *HubClient) Disconnect() {
	hc.transport.Disconnect()
}

// ConnectionStatus returns the connection status: connected, connecting or disconnected
func (hc *HubClient) ConnectionStatus() transports.ConnectionStatus {
	return hc.connectionStatus
}

// LoadCreateKeyPair loads or creates a public/private key pair using the clientID as filename.
//
//	The key-pair is named {clientID}.key, the public key {clientID}.pub
//
//	clientID is the clientID to use, or "" to use the connecting ID
//	keysDir is the location where the keys are stored.
//
// This returns the serialized private and pub keypair, or an error.
func (hc *HubClient) LoadCreateKeyPair(clientID, keysDir string) (kp keys.IHiveKey, err error) {
	if keysDir == "" {
		return nil, fmt.Errorf("certs directory must be provided")
	}
	if clientID == "" {
		clientID = hc.clientID
	}
	keyFile := path.Join(keysDir, clientID+KPFileExt)
	pubFile := path.Join(keysDir, clientID+PubKeyFileExt)

	// load key from file
	kp, err = keys.NewKeyFromFile(keyFile)

	if err != nil {
		// no keyfile, create the key
		kp = hc.transport.CreateKeyPair()

		// save the key for future use
		err = kp.ExportPrivateToFile(keyFile)
		err2 := kp.ExportPublicToFile(pubFile)
		if err2 != nil {
			err = err2
		}
	}

	return kp, err
}

// onConnect is invoked when the connection status changes.
// This cancels the connection attempt if 'retry' is set to false.
// This passes the info through to the handler, if set.
func (hc *HubClient) onConnect(status transports.ConnectionStatus, info transports.ConnInfo) {
	hc.connectionStatus = status
	hc.connectionInfo = info
	retryConnect := hc.retryConnect.Load()
	if !retryConnect && status != transports.Connected {
		slog.Warn("disconnecting and not retrying", "clientID", hc.clientID)
		hc.Disconnect()
	} else if status == transports.Connected {
		slog.Warn("connection restored", "clientID", hc.clientID)
	} else if status == transports.Disconnected {
		slog.Warn("disconnected", "clientID", hc.clientID)
	} else if status == transports.Connecting {
		slog.Warn("retrying to connect", "clientID", hc.clientID)
	}
	hc.mux.RLock()
	defer hc.mux.RUnlock()
	if hc.connectionHandler != nil {
		hc.connectionHandler(status, info)
	}
}

// Handlers of events and requests. These are dispatched to their appropriate handlers
func (hc *HubClient) onEvent(addr string, payload []byte) {
	messageType, agentID, thingID, name, senderID, err := hc.SplitAddress(addr)

	hc.mux.RLock()
	defer hc.mux.RUnlock()
	if err == nil && hc.eventHandler != nil {
		tv := things.NewThingValue(messageType, agentID, thingID, name, payload, senderID)
		hc.eventHandler(tv)
	}
}

// onRequest determines if this is a configuration, action or RPC request and
// passes it to the handler.
// Requests that are not addressed to this agent are treated as events and do not receive a reply.
func (hc *HubClient) onRequest(addr string, payload []byte) (reply []byte, err error, donotreply bool) {
	messageType, agentID, thingID, name, senderID, err := hc.SplitAddress(addr)
	tv := things.NewThingValue(messageType, agentID, thingID, name, payload, senderID)

	hc.mux.RLock()
	defer hc.mux.RUnlock()
	if agentID != hc.clientID {
		if err == nil && hc.eventHandler != nil {
			tv := things.NewThingValue(messageType, agentID, thingID, name, payload, senderID)
			hc.eventHandler(tv)
		}
		donotreply = true
	} else if messageType == vocab.MessageTypeAction && hc.actionHandler != nil {
		slog.Info("Received action request",
			slog.String("sender", senderID),
			slog.String("thingID", thingID),
			slog.String("action", name),
		)
		reply, err = hc.actionHandler(tv)
	} else if messageType == vocab.MessageTypeRPC && hc.rpcHandler != nil {
		slog.Info("Received RPC request",
			slog.String("sender", senderID),
			slog.String("capability", thingID),
			slog.String("method", name),
		)
		reply, err = hc.rpcHandler(tv)

	} else if messageType == vocab.MessageTypeConfig && hc.configHandler != nil {
		slog.Info("Received config request",
			slog.String("sender", senderID),
			slog.String("thingID", thingID),
			slog.String("property", name),
		)
		err = hc.configHandler(tv)
	} else {
		slog.Warn("received unexpected message type as request. Is this an event with a replyTo field?",
			slog.String("sender", senderID),
			slog.String("addr", addr))
		// TBD pass it on to event handler???
	}
	return reply, err, donotreply
}

// PubAction publishes a request for action from a Thing.
//
//	agentID of the device or service that handles the action.
//	thingID is the destination thingID to whom the action applies.
//	name is the name of the action as described in the Thing's TD
//	payload is the optional serialized payload of the action as described in the Thing's TD
//
// This returns the reply data or an error if an error was returned or no reply was received
func (hc *HubClient) PubAction(
	agentID string, thingID string, name string, payload []byte) ([]byte, error) {

	addr := hc.MakeAddress(vocab.MessageTypeAction, agentID, thingID, name, hc.clientID)
	slog.Info("PubAction", "addr", addr)
	data, err := hc.transport.PubRequest(addr, payload)
	return data, err
}

// PubConfig publishes a Thing configuration change request and wait for confirmation.
// No value is returned.
//
// The client's ID is used as the publisher ID of the action.
//
//	agentID of the device that handles the action for the things or service capability
//	thingID is the destination thingID that handles the action
//	propName is the ID of the property to change as described in the TD properties section
//	payload is the optional payload of the configuration as described in the Thing's TD
//
// This returns an error if an error was returned or no confirmation was received
func (hc *HubClient) PubConfig(
	agentID string, thingID string, propName string, payload []byte) error {

	addr := hc.MakeAddress(vocab.MessageTypeConfig, agentID, thingID, propName, hc.clientID)
	slog.Info("PubConfig", "addr", addr)
	_, err := hc.transport.PubRequest(addr, payload)
	return err
}

// PubEvent publishes a Thing event. The payload is an event value as per TD document.
// Event values are send as text and can be converted to native type based on the information defined in the TD.
// Intended for devices and services to notify of changes to the Things they are the agent for.
//
// 'thingID' is the ID of the 'things' whose event to publish. This is the ID under which the
// TD document is published that describes the things. It can be the ID of the sensor, actuator
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
	err := hc.transport.PubEvent(addr, payload)
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
func (hc *HubClient) PubTD(td *things.TD) error {
	payload, _ := ser.Marshal(td)
	addr := hc.MakeAddress(vocab.MessageTypeEvent, hc.clientID, td.ID, vocab.EventNameTD, hc.clientID)
	slog.Info("PubTD", "addr", addr)
	err := hc.transport.PubEvent(addr, payload)
	return err
}

// SetActionHandler sets the handler of incoming action requests
// This will subscribe to actions directed to this client's agentID.
// The result or error will be sent back to the caller.
func (hc *HubClient) SetActionHandler(handler func(msg *things.ThingValue) (reply []byte, err error)) {
	hc.mux.Lock()
	hc.actionHandler = handler
	hc.mux.Unlock()
	addr := hc.MakeAddress(vocab.MessageTypeAction, hc.clientID, "", "", "")
	_ = hc.transport.Subscribe(addr)
	// the request handler will split actions, config and RPC requests
	hc.transport.SetRequestHandler(hc.onRequest)
}

// SetConfigHandler sets the handler of incoming configuration requests
// This will subscribe to configuration requests directed to this client's agentID.
func (hc *HubClient) SetConfigHandler(handler func(msg *things.ThingValue) error) {
	hc.mux.Lock()
	hc.configHandler = handler
	hc.mux.Unlock()
	addr := hc.MakeAddress(vocab.MessageTypeConfig, hc.clientID, "", "", "")
	_ = hc.transport.Subscribe(addr)
	hc.transport.SetRequestHandler(hc.onRequest)
}

// SetConnectionHandler sets the callback for connection status changes.
func (hc *HubClient) SetConnectionHandler(handler func(status transports.ConnectionStatus, info transports.ConnInfo)) {
	hc.mux.Lock()
	hc.connectionHandler = handler
	hc.mux.Unlock()
}

// SetEventHandler sets the handler of subscribed events.
// Use 'Subscribe' to set the events to listen for.
func (hc *HubClient) SetEventHandler(handler func(msg *things.ThingValue)) {
	hc.mux.Lock()
	hc.eventHandler = handler
	hc.mux.Unlock()
	hc.transport.SetEventHandler(hc.onEvent)
}

// SetRetryConnect enables/disables the retry if a connection with the server is broken.
// The default is enabled, so this can be used to disable it for testing connection issues.
// This can be set before or after connection is established.
func (hc *HubClient) SetRetryConnect(enable bool) {
	hc.retryConnect.Store(enable)
}

// SetRPCHandler sets the handler of all incoming RPC requests
// This will subscribe to RPC requests directed to this client's agentID.
// The result or error will be sent back to the caller.
// See also SetRPCCapability to define capability methods in a table
func (hc *HubClient) SetRPCHandler(handler func(msg *things.ThingValue) (reply []byte, err error)) {
	hc.mux.Lock()
	hc.rpcHandler = handler
	hc.mux.Unlock()
	addr := hc.MakeAddress(vocab.MessageTypeRPC, hc.clientID, "", "", "")
	_ = hc.transport.Subscribe(addr)
	// onRequest will invoke the handler
	hc.transport.SetRequestHandler(hc.onRequest)
}

// SetRPCCapability registers an RPC capability with a table of methods
//
// Intended for implementing RPC handlers by services.
// This uses SetRPCHandler, so they can't be used both.
//
// There is no RemoveRPCHandler. Close the hub connection to clear any subscriptions.
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
func (hc *HubClient) SetRPCCapability(capID string, capMethods map[string]interface{}) {

	// add the capability handler
	if hc.rpcHandler == nil {
		multiCapHandler := func(tv *things.ThingValue) (reply []byte, err error) {
			methods, found := hc.capTable[tv.ThingID]
			if !found {
				err = fmt.Errorf("unknown capability '%s'", tv.ThingID)
				return nil, err
			}
			capMethod, found := methods[tv.Name]
			if !found {
				err = fmt.Errorf("method '%s' not part of capability '%s'", tv.Name, capID)
				slog.Warn("SubRPCCapability; unknown method",
					slog.String("methodName", tv.Name),
					slog.String("senderID", tv.SenderID))
				return nil, err
			}
			ctx := ServiceContext{
				Context:  context.Background(),
				SenderID: tv.SenderID,
			}
			respData, err := HandleRequestMessage(ctx, capMethod, tv.Data)
			return respData, err
		}
		hc.SetRPCHandler(multiCapHandler)
	}
	hc.capTable[capID] = capMethods
	hc.transport.SetRequestHandler(hc.onRequest)
}

// SubEvents adds an event subscription to event handler set the SetEventHandler.
//
//	agentID is the ID of the device or service publishing the event, or "" for any agent.
//	thingID is the ID of the Thing whose events to receive, or "" for any Things.
//	eventName is the name of the event, or "" for any event
//
// The handler receives an event value message with data payload.
func (hc *HubClient) SubEvents(agentID string, thingID string, eventName string) error {

	subAddr := hc.MakeAddress(vocab.MessageTypeEvent, agentID, thingID, eventName, "")
	err := hc.transport.Subscribe(subAddr)
	return err
}

// NewHubClientFromTransport returns a new Hub Client instance for the given transport.
//
//   - message bus transport to use, eg NatsTransport or MqttTransport instance
//   - clientID of the client that will be connecting
func NewHubClientFromTransport(transport transports.IHubTransport, clientID string) *HubClient {
	hc := HubClient{
		clientID:  clientID,
		transport: transport,
		capTable:  make(map[string]map[string]interface{}),
	}
	hc.retryConnect.Store(true)
	transport.SetConnectHandler(hc.onConnect)
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
		//tp = mqtttransport_org.NewMqttTransportOrg(url, clientID, caCert)
	}
	hc := NewHubClientFromTransport(tp, clientID)
	return hc
}
