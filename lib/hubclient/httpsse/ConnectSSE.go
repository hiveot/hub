package httpsse

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/tmaxmax/go-sse"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// ConnectSSE establishes a sse session over the Hub HTTPS connection.
// All hub messages are send as type ThingMessage, containing thingID, key, payload and sender
//
// If the connection is interrupted, the sse connection retries with backoff period.
// If an authentication error occurs then the onDisconnect handler is invoked with an error.
// If the connection is cancelled then the onDisconnect is invoked without error
func (cl *HttpSSEClient) ConnectSSE(
	sseURL string, bearerToken string, httpClient *http.Client, onDisconnect func(error)) error {

	slog.Info("ConnectSSE", slog.String("sseURL", sseURL))

	// use context to disconnect the client
	sseCtx, sseCancelFn := context.WithCancel(context.Background())
	bodyReader := bytes.NewReader([]byte{})
	req, err := http.NewRequestWithContext(sseCtx, http.MethodGet, sseURL, bodyReader)
	if err != nil {
		sseCancelFn()
		return err
	}
	req.Header.Add("Authorization", "bearer "+bearerToken)
	parts, _ := url.Parse(sseURL)
	origin := fmt.Sprintf("%s://%s", parts.Scheme, parts.Host)
	req.Header.Add("Origin", origin)
	//req.Header.Add("Connection", "keep-alive")

	cl.sseCancelFn = sseCancelFn
	sseClient := &sse.Client{
		HTTPClient: httpClient,
		OnRetry: func(err error, _ time.Duration) {
			slog.Info("SSE Connection retry", "err", err)
			// TODO: how to be notified if the connection is restored?
			//  workaround: in handleSSEEvent, update the connection status
			cl.SetConnectionStatus(hubclient.Connecting, err)
		},
	}
	conn := sseClient.NewConnection(req)

	// increase buffer size to 1M
	// TODO: make limit configurable
	//https://github.com/tmaxmax/go-sse/issues/32
	newBuf := make([]byte, 0, 1024*65)
	conn.Buffer(newBuf, cl._maxSSEMessageSize)

	remover := conn.SubscribeToAll(cl.handleSSEEvent)
	go func() {
		// connect and report an error if connection ends due to reason other than context cancelled
		err := conn.Connect()

		if connError, ok := err.(*sse.ConnectionError); ok {
			// since sse retries, this is likely an authentication error
			slog.Error("SSE connection failed (server shutdown or connection interrupted)",
				"clientID", cl._status.ClientID,
				"err", err.Error())
			_ = connError
			err = fmt.Errorf("Reconnect Failed: %w", connError.Err) //connError.Err
		} else if errors.Is(err, context.Canceled) {
			// context was cancelled. no error
			err = nil
		}
		remover() // cleanup connection
		onDisconnect(err)
		//
	}()
	// FIXME: use a channel to wait for the SSE connection to be established
	time.Sleep(time.Millisecond * 3)
	return nil
}

// handleSSEEvent processes the push-event received from the hub.
// This is passed on to the client, which must return a delivery
// applied, completed or error status.
// This sends the delivery status to the hub using a delivery event.
func (cl *HttpSSEClient) handleSSEEvent(event sse.Event) {
	var stat hubclient.DeliveryStatus

	// WORKAROUND since go-sse has no callback for a successful reconnect, simulate one here
	// as soon as data is received
	if cl._status.ConnectionStatus != hubclient.Connected {
		// success!
		slog.Warn("handleSSEEvent: connection re-established")
		cl.SetConnectionStatus(hubclient.Connected, nil)
	}

	// ThingMessage is needed to pass messageID, messageType, thingID, key, and sender,
	// as there is no facility in SSE to include metadata.
	// SSE payload is json marshalled by the sse client
	rxMsg := &things.ThingMessage{}
	err := cl.Unmarshal([]byte(event.Data), rxMsg)
	if err != nil {
		slog.Error("handleSSEEvent; Received non-ThingMessage sse event. Ignored",
			"eventType", event.Type,
			"LastEventID", event.LastEventID,
			"err", err.Error())
		return
	}
	stat.MessageID = rxMsg.MessageID
	slog.Debug("handleSSEEvent. Received message",
		//slog.String("Comment", string(event.Comment)),
		slog.String("me", cl._status.ClientID),
		slog.String("messageType", rxMsg.MessageType),
		slog.String("thingID", rxMsg.ThingID),
		slog.String("key", rxMsg.Key),
		slog.String("messageID", rxMsg.MessageID),
		slog.String("senderID", rxMsg.SenderID),
	)

	// always handle rpc response
	if rxMsg.MessageType == vocab.MessageTypeEvent && rxMsg.Key == vocab.EventTypeDeliveryUpdate {
		// this client is receiving a delivery update from a previous action.
		// The payload is a deliverystatus object
		err = rxMsg.Decode(&stat)
		if err != nil || stat.MessageID == "" {
			slog.Error("SSE message of type delivery update is missing messageID or not a DeliveryStatus ", "err", err)
			return
		}
		//err = cl.Decode([]byte(rxMsg.Data), &stat)
		cl.mux.RLock()
		rChan, _ := cl._correlData[stat.MessageID]
		cl.mux.RUnlock()
		if rChan != nil {
			rChan <- &stat
			// if status == DeliveryCompleted || status == DeliveryFailed {
			cl.mux.Lock()
			delete(cl._correlData, rxMsg.MessageID)
			cl.mux.Unlock()
			return
		} else if cl._messageHandler != nil {
			// pass event to client as this is an unsolicited event
			// it could be a delayed confirmation of delivery
			_ = cl._messageHandler(rxMsg)
		} else {
			// missing rpc or message handler
			slog.Error("handleSSEEvent, no handler registered for client",
				"clientID", cl.ClientID())
			stat.DeliveryFailed(rxMsg, fmt.Errorf("handleSSEEvent no handler is set, delivery update ignored"))
		}
	} else if rxMsg.MessageType == vocab.MessageTypeEvent {
		if cl._messageHandler != nil {
			// pass event to handler, if set
			_ = cl._messageHandler(rxMsg)
		} else {
			slog.Warn("handleSSEEvent, no event handler registered. Event ignored.",
				slog.String("key", rxMsg.Key),
				slog.String("clientID", cl.ClientID()))
		}
	} else if rxMsg.MessageType == vocab.MessageTypeAction || rxMsg.MessageType == vocab.MessageTypeProperty {
		if cl._messageHandler != nil {
			// pass action to agent for delivery to thing
			stat = cl._messageHandler(rxMsg)
		} else {
			slog.Warn("handleSSEEvent, no action handler registered. Action ignored.",
				slog.String("key", rxMsg.Key),
				slog.String("clientID", cl.ClientID()))
			stat.DeliveryFailed(rxMsg, fmt.Errorf("handleSSEEvent no handler is set, message ignored"))
		}
		cl.SendDeliveryUpdate(stat)
	} else {
		slog.Warn("handleSSEEvent, unknown message type. Message ignored.",
			slog.String("message type", rxMsg.MessageType),
			slog.String("clientID", cl.ClientID()))
		stat.DeliveryFailed(rxMsg, fmt.Errorf("handleSSEEvent no handler is set, message ignored"))
		cl.SendDeliveryUpdate(stat)
	}
}
