package wssclient

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
)

// handleWSSMessage processes the push-message received from the hub.
func (cl *WSSClient) handleWSSMessage(msg map[string]any) {
	var stat hubclient.RequestProgress

	//var msgData any
	//_ = jsoniter.UnmarshalFromString(event.Data, &msgData)
	rxMsg := &hubclient.ThingMessage{}
	//	ThingID: msg.GetThingID(),
	//	Name:    msg.GetName(),
	//	// should these become operations? worth considering
	//	MessageType: msg.messageType, // event, property, action
	//	SenderID:    senderID,
	//	Created:     time.Now().Format(utils.RFC3339Milli), // TODO: get the real timestamp
	//	Data:        msgData,
	//	RequestID:   requestID,
	//}

	stat.RequestID = rxMsg.RequestID
	slog.Debug("handleWSSEMessage",
		//slog.String("Comment", string(event.Comment)),
		slog.String("clientID (me)", cl.clientID),
		slog.String("messageType", rxMsg.MessageType),
		slog.String("thingID", rxMsg.ThingID),
		slog.String("name", rxMsg.Name),
		slog.String("requestID", rxMsg.RequestID),
		slog.String("senderID", rxMsg.SenderID),
	)
	cl.mux.RLock()
	msgHandler := cl.messageHandler
	cl.mux.RUnlock()
	// always handle rpc response
	if rxMsg.MessageType == vocab.MessageTypeProgressUpdate {
		// this client is receiving a delivery update from an previously sent action.
		// The payload is a deliverystatus object
		err := utils.DecodeAsObject(rxMsg.Data, &stat)
		if err != nil || stat.RequestID == "" || stat.RequestID == "-" {
			slog.Error("WSS message of type delivery update is missing requestID or not a RequestProgress ", "err", err)
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
			_ = msgHandler(rxMsg)
		} else {
			// missing rpc or message handler
			slog.Error("handleWSSEMessage, no handler registered for client",
				"clientID", cl.clientID)
			stat.Failed(rxMsg, fmt.Errorf("handleSSEEvent no handler is set, delivery update ignored"))
		}
		return
	}

	if msgHandler == nil {
		slog.Warn("handleWSSEMessage, no handler registered. Message ignored.",
			slog.String("name", rxMsg.Name),
			slog.String("clientID", cl.clientID))
		return
	}

	if rxMsg.MessageType == vocab.MessageTypeEvent {
		// pass event to handler, if set
		_ = msgHandler(rxMsg)
	} else if rxMsg.MessageType == vocab.MessageTypeAction {
		// agent receives action request
		stat = msgHandler(rxMsg)
		if stat.RequestID != "" {
			_ = cl.PubProgressUpdate(stat) // send the result to the caller
		}
	} else if rxMsg.MessageType == vocab.MessageTypeProperty {
		// agent receives write property request
		// or, consumer receives property update request
		// If this client is an agent then this is a property write request
		// If this client is a consumer then this is am observed property update notification
		_ = msgHandler(rxMsg)
		//stat = msgHandler(rxMsg)
		//if stat.RequestID != "" {
		//	cl.PubProgressUpdate(stat)
		//}
	} else {
		// for now, just pass it on to the handler
		// might be useful for testing when substituting a web browser
		slog.Debug("handleWSSEMessage, unknown message type. Continuing anyways.",
			slog.String("message type", rxMsg.MessageType),
			slog.String("clientID", cl.clientID))
		_ = msgHandler(rxMsg)
		if stat.RequestID != "" {
			cl.PubProgressUpdate(stat)
		}
	}
}
