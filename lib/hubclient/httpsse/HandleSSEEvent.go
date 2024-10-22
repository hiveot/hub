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
// This is passed on to the client, which must return a delivery
// applied, completed or error status.
// This sends the delivery status to the hub using a delivery event.
func (cl *HttpSSEClient) handleSSEEvent(event sse.Event) {
	var stat hubclient.ActionProgress

	cl.mux.RLock()
	connStatus := cl._status.ConnectionStatus
	cl.mux.RUnlock()
	// WORKAROUND since go-sse has no callback for a successful reconnect, simulate one here
	// as soon as data is received. The server could send a 'ping' event on connect.
	if connStatus != hubclient.Connected {
		// success!
		slog.Info("handleSSEEvent: connection (re)established")
		cl.SetConnectionStatus(hubclient.Connected, nil)
	}
	// no further processing of a ping needed
	if event.Type == hubclient.PingMessage {
		return
	}
	messageType := event.Type      // event, action, property, ping, custom event...
	messageID := event.LastEventID // this is the ID provided by the server
	thingID := ""
	senderID := ""
	name := ""

	// event ID field contains: {thingID}/{name}/{senderID}/{messageID}
	parts := strings.Split(event.LastEventID, "/")
	if len(parts) > 1 {
		thingID = parts[0]
		name = parts[1]
		if len(parts) > 2 {
			senderID = parts[2]
		}
		if len(parts) > 3 {
			messageID = parts[3]
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
		Created:     time.Now().Format(utils.RFC3339Milli), // TODO: get the real timestamp
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
	slog.Debug("handleSSEEvent",
		//slog.String("Comment", string(event.Comment)),
		slog.String("me", cl.clientID),
		slog.String("messageType", rxMsg.MessageType),
		slog.String("thingID", rxMsg.ThingID),
		slog.String("name", rxMsg.Name),
		slog.String("messageID", rxMsg.MessageID),
		slog.String("senderID", rxMsg.SenderID),
	)

	// always handle rpc response
	if rxMsg.MessageType == vocab.MessageTypeProgressUpdate {
		// this client is receiving a delivery update from an previously sent action.
		// The payload is a deliverystatus object
		err := utils.DecodeAsObject(rxMsg.Data, &stat)
		if err != nil || stat.MessageID == "" || stat.MessageID == "-" {
			slog.Error("SSE message of type delivery update is missing messageID or not a ActionProgress ", "err", err)
			return
		}
		rxMsg.Data = stat
		//err = cl.Decode([]byte(rxMsg.Data), &stat)
		cl.mux.RLock()
		rChan, _ := cl._correlData[stat.MessageID]
		cl.mux.RUnlock()
		if rChan != nil {
			rChan <- &stat
			// if status == DeliveryCompleted || status == Failed {
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
				"clientID", cl.clientID)
			stat.Failed(rxMsg, fmt.Errorf("handleSSEEvent no handler is set, delivery update ignored"))
		}
		return
	}

	if cl._messageHandler == nil {
		slog.Warn("handleSSEEvent, no handler registered. Message ignored.",
			slog.String("name", rxMsg.Name),
			slog.String("clientID", cl.clientID))
		return
	}

	if rxMsg.MessageType == vocab.MessageTypeEvent {
		// pass event to handler, if set
		_ = cl._messageHandler(rxMsg)
	} else if rxMsg.MessageType == vocab.MessageTypeAction {
		// agent receives action request
		stat = cl._messageHandler(rxMsg)
		if stat.MessageID != "" {
			cl.PubProgressUpdate(stat) // send the result to the caller
		}
	} else if rxMsg.MessageType == vocab.MessageTypeProperty {
		// agent receives write property request
		// or, consumer receives property update request
		// If this client is an agent then this is a property write request
		// If this client is a consumer then this is am observed property update notification
		_ = cl._messageHandler(rxMsg)
		//stat = cl._messageHandler(rxMsg)
		//if stat.MessageID != "" {
		//	cl.PubProgressUpdate(stat)
		//}
	} else {
		slog.Warn("handleSSEEvent, unknown message type. Message ignored.",
			slog.String("message type", rxMsg.MessageType),
			slog.String("clientID", cl.clientID))
		stat.Failed(rxMsg, fmt.Errorf("handleSSEEvent no handler is set, message ignored"))
		if stat.MessageID != "" {
			cl.PubProgressUpdate(stat)
		}
	}
}
