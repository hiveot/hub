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
)

// handleSSEEvent processes the push-event received from the hub.
// This is passed on to the client, which must return a delivery
// applied, completed or error status.
// This sends the delivery status to the hub using a delivery event.
func (hc *HttpSSEClient) handleSSEEvent(event sse.Event) {
	var stat hubclient.DeliveryStatus

	hc.mux.RLock()
	connStatus := hc._status.ConnectionStatus
	hc.mux.RUnlock()
	// WORKAROUND since go-sse has no callback for a successful reconnect, simulate one here
	// as soon as data is received. The server could send a 'ping' event on connect.
	if connStatus != hubclient.Connected {
		// success!
		slog.Info("handleSSEEvent: connection (re)established")
		hc.SetConnectionStatus(hubclient.Connected, nil)
	}
	// no further processing of a ping needed
	if event.Type == hubclient.PingMessage {
		return
	}
	// we need the actual message ID and ignore '-'
	// this isn't pretty. the sse server uses "-" to avoid go-sse injecting a 'previous id'.
	// so, a messageID of "-" mean no id.
	messageID := event.LastEventID // this is the ID provided by the server
	if messageID == "-" {
		messageID = ""
	}
	messageType := event.Type // event, action, property, ping, custom event...
	thingID := ""
	senderID := ""
	name := ""

	// event Type field contains: {eventType}[/{thingID}[/{name}]]
	parts := strings.Split(event.Type, "/")
	if len(parts) > 1 {
		messageType = parts[0]
		thingID = parts[1]
		if len(parts) > 2 {
			name = parts[2]
		}
		if len(parts) > 3 {
			senderID = parts[3]
		}
	}

	// ThingMessage is needed to pass messageID, messageType, thingID, name, and sender,
	// as there is no facility in SSE to include metadata.
	// SSE payload is json marshalled by the sse client
	var msgData any
	_ = jsoniter.Unmarshal([]byte(event.Data), &msgData)
	rxMsg := &hubclient.ThingMessage{
		ThingID:     thingID,
		Name:        name,
		MessageType: messageType,
		SenderID:    senderID,
		Created:     "",
		Data:        msgData,
		MessageID:   messageID,
	}

	//err := cl.Unmarshal([]byte(event.Data), rxMsg)
	//if err != nil {
	//	slog.Error("handleSSEEvent; Received non-ThingMessage sse event. Ignored",
	//		"eventType", event.Type,
	//		"LastEventID", event.LastEventID,
	//		"err", err.Error())
	//	return
	//}
	stat.MessageID = rxMsg.MessageID
	slog.Info("handleSSEEvent",
		//slog.String("Comment", string(event.Comment)),
		slog.String("me", hc.clientID),
		slog.String("messageType", rxMsg.MessageType),
		slog.String("thingID", rxMsg.ThingID),
		slog.String("name", rxMsg.Name),
		slog.String("messageID", rxMsg.MessageID),
		slog.String("senderID", rxMsg.SenderID),
	)

	// always handle rpc response
	if rxMsg.MessageType == vocab.MessageTypeDeliveryUpdate {
		// this client is receiving a delivery update from an previously sent action.
		// The payload is a deliverystatus object
		err := utils.DecodeAsObject(rxMsg.Data, &stat)
		if err != nil || stat.MessageID == "" || stat.MessageID == "-" {
			slog.Error("SSE message of type delivery update is missing messageID or not a DeliveryStatus ", "err", err)
			return
		}
		rxMsg.Data = stat
		//err = cl.Decode([]byte(rxMsg.Data), &stat)
		hc.mux.RLock()
		rChan, _ := hc._correlData[stat.MessageID]
		hc.mux.RUnlock()
		if rChan != nil {
			rChan <- &stat
			// if status == DeliveryCompleted || status == Failed {
			hc.mux.Lock()
			delete(hc._correlData, rxMsg.MessageID)
			hc.mux.Unlock()
			return
		} else if hc._messageHandler != nil {
			// pass event to client as this is an unsolicited event
			// it could be a delayed confirmation of delivery
			_ = hc._messageHandler(rxMsg)
		} else {
			// missing rpc or message handler
			slog.Error("handleSSEEvent, no handler registered for client",
				"clientID", hc.clientID)
			stat.Failed(rxMsg, fmt.Errorf("handleSSEEvent no handler is set, delivery update ignored"))
		}
		return
	}

	if hc._messageHandler == nil {
		slog.Warn("handleSSEEvent, no handler registered. Message ignored.",
			slog.String("name", rxMsg.Name),
			slog.String("clientID", hc.clientID))
		return
	}

	if rxMsg.MessageType == vocab.MessageTypeEvent {
		// pass event to handler, if set
		_ = hc._messageHandler(rxMsg)
	} else if rxMsg.MessageType == vocab.MessageTypeAction {
		// agent receives action request
		stat = hc._messageHandler(rxMsg)
		if stat.MessageID != "" {
			hc.PubProgressUpdate(stat) // send the result to the caller
		}
	} else if rxMsg.MessageType == vocab.MessageTypeProperty {
		// agent receives write property request
		// or, consumer receives property update request
		// FIXME: Need to differentiate!
		//   property update messages are from agent->consumer and dont confirm delivery
		//   while property write messages are consumer->agent and can confirm.
		stat = hc._messageHandler(rxMsg)
		if stat.MessageID != "" {
			hc.PubProgressUpdate(stat)
		}
	} else {
		slog.Warn("handleSSEEvent, unknown message type. Message ignored.",
			slog.String("message type", rxMsg.MessageType),
			slog.String("clientID", hc.clientID))
		stat.Failed(rxMsg, fmt.Errorf("handleSSEEvent no handler is set, message ignored"))
		if stat.MessageID != "" {
			hc.PubProgressUpdate(stat)
		}
	}
}
