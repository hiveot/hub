package httpsseclient

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/servers/hiveotsseserver"
	"github.com/hiveot/hub/messaging/servers/httpserver"
	jsoniter "github.com/json-iterator/go"
	"github.com/tmaxmax/go-sse"
	"log/slog"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

// maxSSEMessageSize allow this maximum size of an SSE message
// go-sse allows this increase of allocation size on receiving messages
const maxSSEMessageSize = 1024 * 1024 * 10

// ConnectSSE establishes a new sse connection using the given http client.
//
// If the connection is interrupted, the sse connection retries with backoff period.
// If an authentication error occurs then the onDisconnect handler is invoked with an error.
// If the connection is cancelled then the onDisconnect is invoked without error
//
// This invokes onConnect when the connection is lost. The caller must handle the
// connection established when the first ping is received after successful connection.
func ConnectSSE(cinfo messaging.ConnectionInfo,
	bearerToken string,
	httcl *http.Client,
	onConnect func(bool, error),
	onMessage func(event sse.Event),
) (cancelFn func(), err error) {

	// use context to disconnect the client on Close
	sseCtx, sseCancelFn := context.WithCancel(context.Background())
	bodyReader := bytes.NewReader([]byte{})
	parts, _ := url.Parse(cinfo.ConnectURL) // the sse: schema isn't recognized. Use https
	parts.Scheme = "https"
	connectURL := parts.String()
	req, err := http.NewRequestWithContext(sseCtx, http.MethodGet, connectURL, bodyReader)
	if err != nil {
		sseCancelFn()
		return nil, err
	}
	req.Header.Add(httpserver.ConnectionIDHeader, cinfo.ConnectionID)
	req.Header.Add("Authorization", "bearer "+bearerToken)
	origin := fmt.Sprintf("https://%s", parts.Host)
	req.Header.Add("Origin", origin)

	sseClient := &sse.Client{
		//HTTPClient: httpClient,
		HTTPClient: httcl,
		OnRetry: func(err error, backoff time.Duration) {
			slog.Warn("SSE Connection retry",
				"err", err, "clientID", cinfo.ClientID,
				"backoff", backoff)
			// TODO: how to be notified if the connection is restored?
			//  workaround: in handleSSEEvent, update the connection status
			onConnect(false, err)
		},
	}
	conn := sseClient.NewConnection(req)

	// increase the maximum buffer size to 1M (_maxSSEMessageSize)
	// note this requires go-sse v0.9.0-pre.2 as a minimum.
	//https://github.com/tmaxmax/go-sse/issues/32
	newBuf := make([]byte, 0, 1024*65)
	// TODO: make limit configurable
	conn.Buffer(newBuf, maxSSEMessageSize)
	remover := conn.SubscribeToAll(onMessage)

	// Wait for max 3 seconds to detect a connection
	waitConnectCtx, waitConnectCancelFn := context.WithTimeout(context.Background(), cinfo.Timeout)
	conn.SubscribeEvent(hiveotsseserver.SSEPingEvent, func(event sse.Event) {
		// WORKAROUND since go-sse has no callback for a successful (re)connect, simulate one here.
		// As soon as a connection is established the server could send a 'ping' event.
		// success!
		slog.Info("handleSSEEvent: connection (re)established; setting connected to true")
		onConnect(true, nil)
		waitConnectCancelFn()
	})
	var sseConnErr atomic.Pointer[sse.ConnectionError]
	go func() {
		// connect and wait until the connection ends
		// and report an error if connection ends due to reason other than context cancelled
		// onConnect will be called on receiving the first (ping) message
		//onConnect(true, nil)
		err := conn.Connect()
		// FIXME: pass 401 unauthorized to caller
		if connError, ok := err.(*sse.ConnectionError); ok {
			// since sse retries, this is likely an authentication error
			slog.Error("SSE connection failed (server shutdown or connection interrupted)",
				"clientID", cinfo.ClientID,
				"err", err.Error())
			sseConnErr.Store(connError)
			//err = fmt.Errorf("connect Failed: %w", connError.Err) //connError.Err
			waitConnectCancelFn()
		} else if errors.Is(err, context.Canceled) {
			// context was cancelled. no error
			err = nil
		}
		onConnect(false, err)
		_ = remover
	}()

	// wait for the SSE connection to be established
	<-waitConnectCtx.Done()
	e := waitConnectCtx.Err()
	if errors.Is(e, context.DeadlineExceeded) {
		err = fmt.Errorf("ConnectSSE: Timeout connecting to the server")
		slog.Warn(err.Error())
		sseCancelFn()
	} else if sseConnErr.Load() != nil {
		err = sseConnErr.Load()
		// something else went wrong
		slog.Warn("ConnectSSE: error" + err.Error())
	}
	closeSSEFn := func() {
		// any other cleanup?
		sseCancelFn()
	}
	return closeSSEFn, err
}

// ConnectSSE establishes the sse connection using the given bearer token
// cl.handleSseEvent will set 'connected' status when the first ping event is
// received from the server. (go-sse doesn't have a connected callback)
func (cc *HiveotSseClient) ConnectSSE(token string) (err error) {
	if cc.ssePath == "" {
		return fmt.Errorf("connectSSE: Missing SSE path")
	}
	// establish the SSE connection for the return channel
	//sseURL := fmt.Sprintf("https://%s%s", cc.hostPort, cc.ssePath)

	cc.sseCancelFn, err = ConnectSSE(cc.cinfo, token,
		// use the same http client for both http requests and sse connection
		cc.GetTlsClient(),
		cc.handleSSEConnect,
		cc.handleSseEvent)

	return err
}

// handler when the SSE connection is established or fails.
// This invokes the connectHandler callback if provided.
func (cc *HiveotSseClient) handleSSEConnect(connected bool, err error) {
	errMsg := ""

	// if the context is cancelled this is not an error
	if err != nil {
		errMsg = err.Error()
	}
	slog.Info("handleSSEConnect",
		slog.String("clientID", cc.cinfo.ClientID),
		slog.String("connectionID", cc.cinfo.ConnectionID),
		slog.Bool("connected", connected),
		slog.String("err", errMsg))

	var connectionChanged bool = false
	if cc.isConnected.Load() != connected {
		connectionChanged = true
	}
	cc.isConnected.Store(connected)
	if err != nil {
		cc.mux.Lock()
		cc.lastError.Store(&err)
		cc.mux.Unlock()
	}
	cc.mux.RLock()
	handler := cc.appConnectHandler
	cc.mux.RUnlock()

	// Note: this callback can send notifications to the client,
	// so prevent deadlock by running in the background.
	// (caught by readhistory failing for unknown reason)
	if connectionChanged && handler != nil {
		go func() {
			handler(connected, err, cc)
		}()
	}
}

// handleSSEEvent processes the push-event received from the hub.
// This splits the message into notification, response and request
// requests have an operation and correlationID
// responses have no operations and a correlationID
// notifications have an operations and no correlationID
func (cc *HiveotSseClient) handleSseEvent(event sse.Event) {

	// no further processing of a ping needed
	if event.Type == hiveotsseserver.SSEPingEvent {
		return
	}

	// Use the hiveot message envelopes for request, response and notification
	if event.Type == messaging.MessageTypeNotification {
		notif := messaging.NotificationMessage{}
		notif.MessageType = messaging.MessageTypeNotification
		_ = jsoniter.UnmarshalFromString(event.Data, &notif)
		// don't block the receiver flow
		go func() {
			cc.mux.RLock()
			h := cc.appNotificationHandler
			cc.mux.RUnlock()
			if h == nil {
				slog.Error("appNotificationHandler is nil")
			} else {
				h(&notif)
			}
		}()
	} else if event.Type == messaging.MessageTypeRequest {
		req := messaging.RequestMessage{}
		_ = jsoniter.UnmarshalFromString(event.Data, &req)
		go func() {
			cc.mux.RLock()
			h := cc.appRequestHandler
			cc.mux.RUnlock()
			if h == nil {
				slog.Error("appRequestHandler is nil")
			} else {
				resp := h(&req, cc)
				_ = cc.SendResponse(resp)
			}
		}()
	} else if event.Type == messaging.MessageTypeResponse {
		resp := messaging.ResponseMessage{}
		resp.MessageType = messaging.MessageTypeResponse
		_ = jsoniter.UnmarshalFromString(event.Data, &resp)
		// don't block the receiver flow
		go func() {
			cc.mux.RLock()
			h := cc.appResponseHandler
			cc.mux.RUnlock()
			if h == nil {
				slog.Error("appResponseHandler is nil")
			} else {
				_ = h(&resp)
			}
		}()
	} else {
		// all other events are intended for other use-cases such as the UI,
		// and can have a formats of event/{dThingID}/{name}
		// Attempt to deliver this for compatibility with other protocols (such has hiveoview test client)
		notif := messaging.NotificationMessage{}
		notif.MessageType = messaging.MessageTypeNotification
		notif.Data = event.Data
		notif.Operation = event.Type
		// don't block the receiver flow
		go func() {
			cc.mux.RLock()
			h := cc.appNotificationHandler
			cc.mux.RUnlock()
			if h == nil {
				slog.Error("appNotificationHandler is nil")
			} else {
				h(&notif)
			}
		}()
	}
}
