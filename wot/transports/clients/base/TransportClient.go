package base

import (
	"crypto/x509"
	"errors"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/utils"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// TransportClient provides base functionality of all transport clients
type TransportClient struct {
	// ID of this client
	BaseClientID string
	// unique connectionID start with the clientID
	BaseConnectionID string

	// CA certificate to verify the server with
	BaseCaCert *x509.Certificate
	// RPC timeout
	BaseTimeout time.Duration
	// protected operations
	BaseMux sync.RWMutex

	// The full server's URL schema://host:port/path
	BaseFullURL string
	// The server host:port
	BaseHostPort string
	// protocol ProtocolTypeHTTP/SSESC/MQTT/WSS
	BaseProtocolType string

	// client is connected
	BaseIsConnected atomic.Bool

	// Request and Response channel helper
	BaseRnrChan *utils.RnRChan

	// implementation of sendNotification
	BaseSendNotification func(op string, thingID string, name string, data any) error

	// callback for reporting connection status change
	BaseConnectHandler func(connected bool, err error)
	// callback client side handler that receives consumer facing messages from the hub
	BaseNotificationHandler transports.NotificationHandler
	// callback client side handler that receives agent requests from the hub
	BaseRequestHandler transports.RequestHandler
}

// GetClientID returns the client's account ID
func (cl *TransportClient) GetClientID() string {
	return cl.BaseClientID
}

// GetConnectionID returns the client's connection ID
func (cl *TransportClient) GetConnectionID() string {
	return cl.BaseConnectionID
}

// GetProtocolType returns the type of protocol this client supports
func (cl *TransportClient) GetProtocolType() string {
	return cl.BaseProtocolType
}

// GetServerURL returns the schema://address:port/path of the server connection
func (cl *TransportClient) GetServerURL() string {
	return cl.BaseFullURL
}

// IsConnected return whether the return channel is connection, eg can receive data
func (cl *TransportClient) IsConnected() bool {
	return cl.BaseIsConnected.Load()
}

// Marshal encodes the native data into the wire format
func (cl *TransportClient) Marshal(data any) []byte {
	jsonData, _ := jsoniter.Marshal(data)
	return jsonData
}
func (cl *TransportClient) ObserveProperty(thingID string, name string) error {
	if name != "" {
		return cl.BaseSendNotification(wot.OpObserveProperty, thingID, name, nil)
	} else {
		return cl.BaseSendNotification(wot.OpObserveAllProperties, thingID, "", nil)
	}
}

// SetConnectHandler sets the notification handler of connection failure
// Intended to notify the client that a reconnect or relogin is needed.
func (cl *TransportClient) SetConnectHandler(cb func(connected bool, err error)) {
	cl.BaseMux.Lock()
	cl.BaseConnectHandler = cb
	cl.BaseMux.Unlock()
}

// SetNotificationHandler set the handler that receives server notifications
func (cl *TransportClient) SetNotificationHandler(cb transports.NotificationHandler) {
	cl.BaseMux.Lock()
	cl.BaseNotificationHandler = cb
	cl.BaseMux.Unlock()
}

// SetRequestHandler set the handler that receives all agent facing requests
// and returns a reply.
func (cl *TransportClient) SetRequestHandler(cb transports.RequestHandler) {
	cl.BaseMux.Lock()
	cl.BaseRequestHandler = cb
	cl.BaseMux.Unlock()
}

// Subscribe to one or all events of a thing
// name is the event to subscribe to or "" for all events
func (cl *TransportClient) Subscribe(thingID string, name string) error {
	if name != "" {
		return cl.BaseSendNotification(wot.OpSubscribeEvent, thingID, name, nil)
	} else {
		return cl.BaseSendNotification(wot.OpSubscribeAllEvents, thingID, "", nil)
	}
}

// Unmarshal decodes the wire format to native data
func (cl *TransportClient) Unmarshal(raw []byte, reply interface{}) error {
	err := jsoniter.Unmarshal(raw, reply)
	return err
}

// Unobserve a previous observe of a property
func (cl *TransportClient) UnobserveProperty(thingID string, name string) error {
	if name != "" {
		return cl.BaseSendNotification(wot.OpUnobserveProperty, thingID, name, nil)
	} else {
		return cl.BaseSendNotification(wot.OpUnobserveAllProperties, thingID, "", nil)
	}
}

// Unsubscribe from previous subscription
func (cl *TransportClient) Unsubscribe(thingID string, name string) error {
	if name != "" {
		return cl.BaseSendNotification(wot.OpUnsubscribeEvent, thingID, name, nil)
	} else {
		return cl.BaseSendNotification(wot.OpUnsubscribeAllEvents, thingID, "", nil)
	}
}

// WaitForResponse waits for a response message on the given channel,
// or until N seconds passed, or the connection drops.
//
// If a response is received it is written to the given output and nil (no error) is returned
// If anything goes wrong, an error is returned
func (cl *TransportClient) WaitForResponse(
	rChan chan any, requestID string, output any) (err error) {

	// wait for reply
	waitCount := 0
	var completed bool
	var reply any

	for !completed {
		// if the hub connection no longer exists then don't wait any longer
		if !cl.IsConnected() {
			err = errors.New("connection lost")
			break
		}

		// wait at most cl.timeout or until delivery completes or fails
		// if the connection breaks while waiting then tlsClient will be nil.
		if time.Duration(waitCount)*time.Second > cl.BaseTimeout {
			break
		}
		if waitCount > 0 {
			slog.Info("SendRequest (wait)",
				slog.Int("count", waitCount),
				slog.String("clientID", cl.GetClientID()),
				slog.String("requestID", requestID),
			)
		}
		completed, reply = cl.BaseRnrChan.WaitForResponse(rChan, time.Second)
		waitCount++
	}

	// ending the wait
	cl.BaseRnrChan.Close(requestID)
	slog.Info("SendRequest (result)",
		slog.String("clientID", cl.GetClientID()),
		slog.String("requestID", requestID),
	)

	// check for errors
	if err != nil {
		slog.Warn("SendRequest failed", "err", err.Error())
	}
	// only after completion will there be a reply as a result
	if err == nil && output != nil && reply != nil {
		// no choice but to decode
		err = utils.Decode(reply, output)
	}
	return err
}
