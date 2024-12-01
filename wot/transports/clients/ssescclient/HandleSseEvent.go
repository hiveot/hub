package ssescclient

import (
	"fmt"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/utils"
	jsoniter "github.com/json-iterator/go"
	"github.com/tmaxmax/go-sse"
	"log/slog"

	"strings"
	"time"
)

// handleSSEEvent processes the push-event received from the hub.
func (cl *SsescTransportClient) handleSseEvent(event sse.Event) {
	var stat transports.RequestStatus

	// WORKAROUND since go-sse has no callback for a successful reconnect, simulate one here.
	// As soon as a connection is established the server could send a 'ping' event.
	if !cl.isConnected.Load() {
		// success!
		slog.Info("handleSSEEvent: connection (re)established")
		// Note: this callback can send notifications to the client,
		// so prevent deadlock by running in the background.
		// (caught by readhistory failing for unknown reason)
		go cl.handleSSEConnect(true, nil)
	}
	// no further processing of a ping needed
	if event.Type == PingMessage {
		return
	}
	operation := event.Type            // one of the WotOp... or HTOp... operations
	correlationID := event.LastEventID // this is the ID provided by the server
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
			correlationID = parts[3]
		}
	}

	// ThingMessage is needed to pass correlationID, messageType, thingID, name, and sender,
	// as there is no facility in SSE to include metadata.
	// SSE payload is json marshalled by the sse client
	var msgData any
	_ = jsoniter.UnmarshalFromString(event.Data, &msgData)
	rxMsg := &transports.ThingMessage{
		ThingID:       thingID,
		Name:          name,
		Operation:     operation,
		SenderID:      senderID,
		Created:       time.Now().Format(wot.RFC3339Milli), // TODO: get the real timestamp
		Data:          msgData,
		CorrelationID: correlationID,
	}

	stat.CorrelationID = rxMsg.CorrelationID
	slog.Info("handleSseEvent",
		//slog.String("Comment", string(event.Comment)),
		slog.String("clientID (me)", cl.clientID),
		slog.String("cid", cl.GetCID()),
		slog.String("operation", rxMsg.Operation),
		slog.String("thingID", rxMsg.ThingID),
		slog.String("name", rxMsg.Name),
		slog.String("correlationID", rxMsg.CorrelationID),
		slog.String("senderID", rxMsg.SenderID),
	)
	cl.mux.RLock()
	msgHandler := cl.messageHandler
	reqHandler := cl.requestHandler
	cl.mux.RUnlock()
	// always handle rpc response
	if rxMsg.Operation == wot.HTOpUpdateActionStatus {
		// this client is receiving a status update from a previously sent action.
		// The payload is a deliverystatus object
		err := utils.DecodeAsObject(rxMsg.Data, &stat)
		if err != nil || stat.CorrelationID == "" || stat.CorrelationID == "-" {
			slog.Error("handleSseEvent: SSE message of type delivery update is missing requestID or not a RequestStatus ", "err", err)
			return
		}
		rxMsg.Data = stat
		//err = cl.Decode([]byte(rxMsg.Data), &stat)
		cl.mux.RLock()
		rChan, _ := cl.correlData[stat.CorrelationID]
		cl.mux.RUnlock()
		if rChan != nil {
			rChan <- &stat
			// if status == DeliveryCompleted || status == Failed {
			cl.mux.Lock()
			delete(cl.correlData, rxMsg.CorrelationID)
			cl.mux.Unlock()
			return
		} else if msgHandler != nil {
			// pass event to client as this is an unsolicited event
			// it could be a delayed confirmation of delivery
			msgHandler(rxMsg)
		} else {
			// missing rpc or message handler
			slog.Error("handleSseEvent, no message handler registered for client",
				"clientID", cl.clientID,
				"op", rxMsg.Operation)
			stat.Failed(rxMsg, fmt.Errorf("handleSseEvent no handler is set, delivery update ignored"))
		}
		return
	}

	// messages (no reply) and requests (with reply) are handled separately
	if rxMsg.Operation == wot.OpInvokeAction ||
		rxMsg.Operation == wot.OpWriteProperty ||
		rxMsg.Operation == wot.OpWriteMultipleProperties {
		// agent receives action request and sends a reply
		if reqHandler == nil {
			slog.Warn("handleSseEvent, no request handler registered. Request ignored.",
				slog.String("operation", rxMsg.Operation),
				slog.String("thingID", rxMsg.ThingID),
				slog.String("name", rxMsg.Name),
				slog.String("clientID", cl.clientID))
			return
		}
		stat = reqHandler(rxMsg)
		if stat.CorrelationID != "" {
			cl.httpClient.SendOperationStatus(stat) // send the result to the caller
		}
	} else {
		// pass everything else to the message handler. No reply is sent.
		// Eg" consumer receive event, property and TD updates
		if msgHandler == nil {
			slog.Warn("handleSseEvent, no message handler registered. Message ignored.",
				slog.String("operation", rxMsg.Operation),
				slog.String("thingID", rxMsg.ThingID),
				slog.String("name", rxMsg.Name),
				slog.String("clientID", cl.clientID))
			return
		}
		msgHandler(rxMsg)
	}
}
