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
	var stat hubclient.RequestStatus

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
		// this client is receiving a status update from an previously sent action.
		// The payload is a deliverystatus object
		err := utils.DecodeAsObject(rxMsg.Data, &stat)
		if err != nil || stat.RequestID == "" || stat.RequestID == "-" {
			slog.Error("WSS message of type delivery update is missing requestID or not a RequestStatus ", "err", err)
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
			slog.Error("handleWSSMessage, no handler registered for client",
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
			slog.Warn("handleWSSMessage, no request handler registered. Request ignored.",
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
			slog.Warn("handleWSSMessage, no message handler registered. Message ignored.",
				slog.String("operation", rxMsg.Operation),
				slog.String("thingID", rxMsg.ThingID),
				slog.String("name", rxMsg.Name),
				slog.String("clientID", cl.clientID))
			return
		}
		msgHandler(rxMsg)
	}
}
