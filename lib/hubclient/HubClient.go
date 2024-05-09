package hubclient

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/hubclient/transports/httptransport"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/protocols/direct"
	"log/slog"
	"net/url"
	"os"
	"path"
	"sync"
	"sync/atomic"
)

// TokenFileExt defines the filename extension under which client tokens are stored
// in the keys directory.
const TokenFileExt = ".token"

// HubClient wrapper around the underlying message bus transport.
type HubClient struct {
	//serverURL string
	//caCert    *x509.Certificate

	// login ID
	clientID string
	//
	transport transports.IHubTransport
	// key set after connecting with token
	kp keys.IHiveKey
	//connectionStatus transports.ConnectionStatus

	// keep retrying connection on error (default true)
	retryConnect atomic.Bool

	// mux is used to protect access to handlers and capTable
	mux sync.RWMutex

	// capability map:
	//  map[capID] map[methodName]handler
	capTable map[string]map[string]interface{}

	connectionHandler func(status transports.HubTransportStatus)
	messageHandler    api.MessageHandler
}

// MakeAddress creates a message address optionally with wildcards
// This uses the hiveot address format: {msgType}/{deviceID}/{thingID}/{name}[/{clientID}]
// Where '/' is the address separator for MQTT or '.' for Nats
// Where "+" is the wildcard for MQTT or "*" for Nats
//
//	msgType is the message type: "event", "action", "config" or "rpc".
//	agentID is the thingID of the device or service being addressed. Use "" for wildcard
//	thingID is the ID of the thing including the urn: prefix. Use "" for wildcard
//	name is the event or action name. Use "" for wildcard.
//	clientID is the login ID of the sender. Use "" for subscribe.
//func (hc *HubClient) MakeAddress(msgType, agentID, thingID, name string, clientID string) string {
//	sep, wc, rem := hc.transport.AddressTokens()
//
//	parts := make([]string, 0, 5)
//	if msgType == "" {
//		msgType = vocab.MessageTypeEvent
//	}
//	parts = append(parts, msgType)
//	if agentID == "" {
//		agentID = wc
//	}
//	parts = append(parts, agentID)
//	if thingID == "" {
//		thingID = wc
//	}
//	parts = append(parts, thingID)
//	if name == "" {
//		name = wc
//	}
//	parts = append(parts, name)
//	if clientID == "" {
//		clientID = rem
//	}
//	parts = append(parts, clientID)
//
//	addr := strings.Join(parts, sep)
//	return addr
//}

// SplitAddress separates an address into its components
//
// addr is a hiveot address eg: msgType/things/agentID/thingID/name/clientID
//func (hc *HubClient) SplitAddress(addr string) (msgType, agentID, thingID, name string, senderID string, err error) {
//
//	sep, _, _ := hc.transport.AddressTokens()
//	parts := strings.Split(addr, sep)
//
//	// inbox topics are short
//	if len(parts) >= 1 && strings.HasPrefix(addr, vocab.MessageTypeINBOX) {
//		msgType = parts[0]
//		if len(parts) >= 2 {
//			agentID = parts[1]
//		}
//		return
//	}
//	if len(parts) < 4 {
//		err = errors.New("incomplete address")
//		return
//	}
//	msgType = parts[0]
//	agentID = parts[1]
//	thingID = parts[2]
//	name = parts[3]
//	if len(parts) > 4 {
//		senderID = parts[4]
//	}
//	return
//}

// ClientID the client is authenticated as to the server
func (hc *HubClient) ClientID() string {
	return hc.clientID
}

// ClientKP returns the key-pair used to connect with token or nil if connected with password
func (hc *HubClient) ClientKP() keys.IHiveKey {
	return hc.kp
}

// ConnectWithCert connects to the Hub server using a client certificate.
//
//	kp is the serialized public/private key-pair of this client
//	jwtToken is the token obtained with login or refresh.
func (hc *HubClient) ConnectWithCert(kp keys.IHiveKey, cert *tls.Certificate) error {
	_, err := hc.transport.ConnectWithCert(kp, cert)
	hc.kp = kp
	return err
}

// ConnectWithJWT connects to the Hub server using a user JWT credentials secret
// The token clientID must match that of the client
//
//	jwtToken is the token obtained with login or refresh.
func (hc *HubClient) ConnectWithJWT(jwtToken string) error {
	err := hc.transport.ConnectWithJWT(jwtToken)
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
	keyFile := path.Join(keysDir, hc.clientID+keys.KPFileExt)
	tokenFile := path.Join(keysDir, hc.clientID+TokenFileExt)
	token, err := os.ReadFile(tokenFile)
	if err == nil && keyFile != "" {
		kp, err = keys.NewKeyFromFile(keyFile)
	}
	if err != nil {
		return fmt.Errorf("ConnectWithTokenFile failed: %w", err)
	}
	hc.kp = kp
	err = hc.transport.ConnectWithJWT(string(token))
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

// GetStatus returns the transport connection status
func (hc *HubClient) GetStatus() transports.HubTransportStatus {
	return hc.transport.GetStatus()
}

// onConnect is invoked when the connection status changes.
// This cancels the connection attempt if 'retry' is set to false.
// This passes the info through to the handler, if set.
func (hc *HubClient) onConnect(status transports.HubTransportStatus) {
	//hc.connectionStatus = status
	//hc.connectionInfo = info
	retryConnect := hc.retryConnect.Load()
	if !retryConnect && status.ConnectionStatus != transports.Connected {
		slog.Warn("disconnecting and not retrying", "clientID", hc.clientID)
		hc.Disconnect()
	} else if status.ConnectionStatus == transports.Connected {
		slog.Warn("connection restored", "clientID", hc.clientID)
	} else if status.ConnectionStatus == transports.Disconnected {
		slog.Warn("disconnected", "clientID", hc.clientID)
	} else if status.ConnectionStatus == transports.Connecting {
		slog.Warn("retrying to connect", "clientID", hc.clientID)
	}
	hc.mux.RLock()
	handler := hc.connectionHandler
	hc.mux.RUnlock()
	if handler != nil {
		handler(status)
	}
}

// PubAction publishes a request for action from a Thing.
//
//	thingID is the destination thingID to whom the action applies.
//	key is the name of the action as described in the Thing's TD
//	payload is the optional serialized payload of the action as described in the Thing's TD
//
// This returns the delivery status and reply data if delivery is completed
func (hc *HubClient) PubAction(thingID string, key string, payload []byte) api.DeliveryStatus {

	stat := hc.transport.PubAction(thingID, key, payload)
	return stat
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
//func (hc *HubClient) PubConfig(
//	agentID string, thingID string, propName string, payload []byte) error {
//
//	addr := hc.MakeAddress(transports.MessageTypeConfig, agentID, thingID, propName, hc.clientID)
//	slog.Info("PubConfig", "addr", addr)
//	_, err := hc.transport.PubAction(addr, payload)
//	return err
//}

// PubEvent publishes a Thing event. The payload is an event value as per TD document.
// Event values are send as text and can be converted to native type based on its event affordance
// defined in the TD.
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
//	eventID is the key of the event as defined in the event affordance map in the Thing TD
//	payload is the serialized event value, or nil if the event has no value
func (hc *HubClient) PubEvent(thingID string, eventID string, payload []byte) api.DeliveryStatus {

	if eventID == "" {
		errMsg := "PubEvent missing eventID"
		slog.Error(errMsg)
		return api.DeliveryStatus{Status: api.DeliveryFailed, Error: errMsg}
	}
	stat := hc.transport.PubEvent(thingID, eventID, payload)
	return stat
}

// SetMessageHandler set the single handler that receives all subscribed messages
// and messages directed at this client.
//
// The result or error will be sent back to the caller.
func (hc *HubClient) SetMessageHandler(handler api.MessageHandler) {

	hc.mux.Lock()
	hc.messageHandler = handler
	hc.mux.Unlock()

	hc.transport.SetMessageHandler(
		func(msg *things.ThingMessage) api.DeliveryStatus {
			stat := hc.messageHandler(msg)
			return stat
		})
}

// SetConnectionHandler sets the callback for connection status changes.
func (hc *HubClient) SetConnectionHandler(handler func(status transports.HubTransportStatus)) {
	hc.mux.Lock()
	hc.connectionHandler = handler
	hc.mux.Unlock()
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
//func (hc *HubClient) SetRPCHandler(handler func(msg *things.ThingMessage) (reply []byte, err error)) {
//	hc.mux.Lock()
//	hc.rpcHandler = handler
//	hc.mux.Unlock()
//	addr := hc.MakeAddress(
//		vocab.MessageTypeRPC, hc.clientID, "", "", "")
//	_ = hc.transport.Subscribe(addr)
//	// onRequest will invoke the handler
//	hc.transport.SetRequestHandler(hc.onRequest)
//}

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
//func (hc *HubClient) SetRPCCapability(capID string, capMethods map[string]interface{}) {
//
//	// add the capability handler
//	if hc.rpcHandler == nil {
//		multiCapHandler := func(tv *things.ThingMessage) (reply []byte, err error) {
//			methods, found := hc.capTable[tv.ThingID]
//			if !found {
//				err = fmt.Errorf("unknown capability '%s'", tv.ThingID)
//				return nil, err
//			}
//			capMethod, found := methods[tv.Key]
//			if !found {
//				err = fmt.Errorf("method '%s' not part of capability '%s'", tv.Key, capID)
//				slog.Warn("SubRPCCapability; unknown method",
//					slog.String("methodName", tv.Key),
//					slog.String("senderID", tv.SenderID))
//				return nil, err
//			}
//			ctx := ServiceContext{
//				Context:  context.Background(),
//				SenderID: tv.SenderID,
//			}
//			respData, err := HandleRequestMessage(ctx, capMethod, tv.Data)
//			return respData, err
//		}
//		hc.SetRPCHandler(multiCapHandler)
//	}
//	hc.capTable[capID] = capMethods
//	hc.transport.SetRequestHandler(hc.onRequest)
//}

// SubEvents adds an event subscription to event handler set the SetEventHandler.
//
//	agentID is the ID of the device or service publishing the event, or "" for any agent.
//	thingID is the ID of the Thing whose events to receive, or "" for any Things.
//	eventKey is the key of the event from the TD, or "" for any event
//
// The handler receives an event value message with data payload.
//func (hc *HubClient) SubEvents(agentID string, thingID string, eventKey string) error {
//
//	subAddr := hc.MakeAddress(
//		vocab.MessageTypeEvent, agentID, thingID, eventKey, "")
//	err := hc.transport.Subscribe(subAddr)
//	return err
//}

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
//   - fullURL of server to connect to.
//   - clientID is the account/login ID of the client that will be connecting
//   - caCert of server or nil to not verify server cert
func NewHubClient(fullURL string, clientID string, caCert *x509.Certificate) *HubClient {
	var tp transports.IHubTransport
	parts, _ := url.Parse(fullURL)
	tpid := parts.Scheme
	if tpid == "nats" {
		// FIXME: nats
		//tp = natstransport.NewNatsTransport(url, clientID, caCert)
	} else if tpid == "mqtt" {
		// FIXME: support mqtt
		//tp = mqtttransport.NewMqttTransport(url, clientID, caCert)
	} else if tpid == "https" || tpid == "tls" {
		// FIXME: obtain the sse path from elsewhere?
		ssePath := "/sse"
		tp = httptransport.NewHttpSSETransport(parts.Host, ssePath, clientID, caCert)
	} else if tpid == "" {
		tp = direct.NewEmbeddedTransport(clientID, nil)
	}
	hc := NewHubClientFromTransport(tp, clientID)
	return hc
}
