package httpsse

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/utils"
	jsoniter "github.com/json-iterator/go"
	"github.com/tmaxmax/go-sse"
	"log/slog"

	"strings"
	"time"
)

// handleSSEEvent processes the push-event received from the hub.
func (cl *HttpSSEClient) handleSSEEvent(event sse.Event) {
	var stat hubclient.RequestStatus

	// WORKAROUND since go-sse has no callback for a successful reconnect, simulate one here
	// as soon as data is received. The server could send a 'ping' event on connect.
	if !cl.isConnected.Load() {
		// success!
		slog.Info("handleSSEEvent: connection (re)established")
		// Note: this callback can send notifications to the client,
		// so prevent deadlock by running in the background.
		// (caught by readhistory failing for unknown reason)
		go cl.handleSSEConnect(true, nil)
	}
	// no further processing of a ping needed
	if event.Type == hubclient.PingMessage {
		return
	}
	operation := event.Type        // one of the WotOp... or HTOp... operations
	requestID := event.LastEventID // this is the ID provided by the server
	thingID := ""
	senderID := ""
	name := ""

	// event ID field contains: {thingID}/{name}/{senderID}/{requestID}
	parts := strings.Split(event.LastEventID, "/")
	if len(parts) > 1 {
		thingID = parts[0]
		name = parts[1]
		if len(parts) > 2 {
			senderID = parts[2]
		}
		if len(parts) > 3 {
			requestID = parts[3]
		}
	}

	// ThingMessage is needed to pass requestID, messageType, thingID, name, and sender,
	// as there is no facility in SSE to include metadata.
	// SSE payload is json marshalled by the sse client
	var msgData any
	_ = jsoniter.UnmarshalFromString(event.Data, &msgData)
	rxMsg := &hubclient.ThingMessage{
		ThingID:   thingID,
		Name:      name,
		Operation: operation,
		SenderID:  senderID,
		Created:   time.Now().Format(utils.RFC3339Milli), // TODO: get the real timestamp
		Data:      msgData,
		RequestID: requestID,
	}

	stat.RequestID = rxMsg.RequestID
	slog.Debug("handleSSEEvent",
		//slog.String("Comment", string(event.Comment)),
		slog.String("clientID (me)", cl.clientID),
		slog.String("cid", cl.cid),
		slog.String("operation", rxMsg.Operation),
		slog.String("thingID", rxMsg.ThingID),
		slog.String("name", rxMsg.Name),
		slog.String("requestID", rxMsg.RequestID),
		slog.String("senderID", rxMsg.SenderID),
	)
	cl.mux.RLock()
	msgHandler := cl.messageHandler
	reqHandler := cl.requestHandler
	cl.mux.RUnlock()
	// always handle rpc response
	if rxMsg.Operation == vocab.WotOpPublishActionStatus {
		// this client is receiving a delivery update from a previously sent action.
		// The payload is a deliverystatus object
		err := utils.DecodeAsObject(rxMsg.Data, &stat)
		if err != nil || stat.RequestID == "" || stat.RequestID == "-" {
			slog.Error("SSE message of type delivery update is missing requestID or not a RequestStatus ", "err", err)
			return
		}
		rxMsg.Data = stat
		//err = cl.Decode([]byte(rxMsg.Data), &stat)
		cl.mux.RLock()
		rChan, _ := cl.correlData[stat.RequestID]
		cl.mux.RUnlock()
		if rChan != nil {
			rChan <- &stat
			// if status == DeliveryCompleted || status == Failed {
			cl.mux.Lock()
			delete(cl.correlData, rxMsg.RequestID)
			cl.mux.Unlock()
			return
		} else if msgHandler != nil {
			// pass event to client as this is an unsolicited event
			// it could be a delayed confirmation of delivery
			msgHandler(rxMsg)
		} else {
			// missing rpc or message handler
			slog.Error("handleSSEEvent, no message handler registered for client",
				"clientID", cl.clientID)
			stat.Failed(rxMsg, fmt.Errorf("handleSSEEvent no handler is set, delivery update ignored"))
		}
		return
	}

	// note messages and requests are handled separately
	if rxMsg.Operation == vocab.WotOpInvokeAction ||
		rxMsg.Operation == vocab.WotOpWriteProperty ||
		rxMsg.Operation == vocab.WotOpWriteMultipleProperties {
		// agent receives action request
		if reqHandler == nil {
			slog.Warn("handleSSEEvent, no request handler registered. Request ignored.",
				slog.String("operation", rxMsg.Operation),
				slog.String("thingID", rxMsg.ThingID),
				slog.String("name", rxMsg.Name),
				slog.String("clientID", cl.clientID))
			return
		}
		stat = reqHandler(rxMsg)
		if stat.RequestID != "" {
			cl.PubRequestStatus(stat) // send the result to the caller
		}
	} else {
		// pass everything else to the message handler
		// consumer receive event, property and TD updates
		if msgHandler == nil {
			slog.Warn("handleSSEEvent, no message handler registered. Message ignored.",
				slog.String("operation", rxMsg.Operation),
				slog.String("thingID", rxMsg.ThingID),
				slog.String("name", rxMsg.Name),
				slog.String("clientID", cl.clientID))
			return
		}
		msgHandler(rxMsg)
	}
}
