package ssescclient

import (
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/servers/ssescserver"
	"github.com/hiveot/hub/wot"
	jsoniter "github.com/json-iterator/go"
	"github.com/tmaxmax/go-sse"
	"log/slog"

	"strings"
	"time"
)

// this client is receiving a status update from a previously sent action.
// The payload is a RequestStatus object.
func (cl *SsescTransportClient) handleResponseMessage(
	msg *transports.ThingMessage, raw []byte) {
	//var stat transports.RequestStatus
	//err := utils.DecodeAsObject(raw, &stat)
	//if err != nil || stat.RequestID == "" || stat.RequestID == "-" {
	//	slog.Error("handleSseEvent: SSE message of type delivery update is missing requestID or not a RequestStatus ", "err", err)
	//	return
	//}
	//msg.Data = stat
	// Determine if this was the result of an RPC call
	//handled := cl.BaseRnrChan.HandleResponse(stat.RequestID, stat)
	handled := cl.BaseRnrChan.HandleResponse(msg.RequestID, raw, true)
	if handled {
		return
	} else if cl.BaseNotificationHandler != nil {
		// last resort is to pass it as a notification
		cl.BaseNotificationHandler(msg)
	} else {
		// missing handler
		slog.Error("handleSseEvent update action status, no message handler",
			"clientID", cl.GetClientID(),
			"op", msg.Operation)
	}
}

// Agent received a request that expects a response
func (cl *SsescTransportClient) handleRequestMessage(msg *transports.ThingMessage) {
	// agent receives action request and sends a reply
	if cl.BaseRequestHandler == nil {
		slog.Warn("handleSseEvent, no request handler registered. Request ignored.",
			slog.String("operation", msg.Operation),
			slog.String("thingID", msg.ThingID),
			slog.String("name", msg.Name),
			slog.String("clientID", cl.GetClientID()))
		return
	}
	output, err := cl.BaseRequestHandler(msg)
	if msg.RequestID == "" {
		// no response
	} else if err != nil {
		cl.SendResponse(msg.ThingID, msg.Name, output, err, msg.RequestID)
	}
}

// Anything that isn't a request or response is passed up as a notification
func (cl *SsescTransportClient) handleNotificationMessage(msg *transports.ThingMessage) {

	// pass everything else to the message handler. No reply is sent.
	// Eg" consumer receive event, property and TD updates
	if cl.BaseNotificationHandler == nil {
		slog.Warn("handleSseEvent, no message handler registered. Message ignored.",
			slog.String("operation", msg.Operation),
			slog.String("thingID", msg.ThingID),
			slog.String("name", msg.Name),
			slog.String("clientID", cl.GetClientID()))
		return
	}
	cl.BaseNotificationHandler(msg)
}

// handleSSEEvent processes the push-event received from the hub.
func (cl *SsescTransportClient) handleSseEvent(event sse.Event) {

	// WORKAROUND since go-sse has no callback for a successful reconnect, simulate one here.
	// As soon as a connection is established the server could send a 'ping' event.
	if !cl.IsConnected() {
		// success!
		slog.Info("handleSSEEvent: connection (re)established")
		// Note: this callback can send notifications to the client,
		// so prevent deadlock by running in the background.
		// (caught by readhistory failing for unknown reason)
		go cl.handleSSEConnect(true, nil)
	}
	// no further processing of a ping needed
	if event.Type == ssescserver.SSEPingEvent {
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
	rxMsg := &transports.ThingMessage{
		ThingID:   thingID,
		Name:      name,
		Operation: operation,
		SenderID:  senderID,
		Timestamp: time.Now().Format(wot.RFC3339Milli), // TODO: get the real timestamp
		Data:      msgData,
		RequestID: requestID,
	}
	slog.Info("handleSseEvent",
		//slog.String("Comment", string(event.Comment)),
		slog.String("clientID (me)", cl.GetClientID()),
		slog.String("connectionID", cl.GetConnectionID()),
		slog.String("operation", rxMsg.Operation),
		slog.String("thingID", rxMsg.ThingID),
		slog.String("name", rxMsg.Name),
		slog.String("requestID", rxMsg.RequestID),
		slog.String("senderID", rxMsg.SenderID),
	)
	// always handle rpc response
	switch rxMsg.Operation {
	case wot.HTOpUpdateActionStatus:
		// this client is receiving a status update from a previously sent action.
		cl.handleResponseMessage(rxMsg, []byte(event.Data))
	case wot.OpInvokeAction, wot.OpWriteProperty, wot.OpWriteMultipleProperties:
		cl.handleRequestMessage(rxMsg)
	default: // some kind of notification
		cl.handleNotificationMessage(rxMsg)
		// pass everything else to the message handler. No reply is sent.
	}
}
