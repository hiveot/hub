package wssclient

import (
	"github.com/hiveot/hub/lib/hubclient"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
)

// handle receiving an action status update
// this needs a message handler to pass it to.
func (cl *WSSClient) handleActionStatus(jsonMsg string) {
	var senderID = ""
	var stat hubclient.RequestStatus

	wssMsg := ActionStatusMessage{}
	_ = jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
	op := MsgTypeToOp[wssMsg.MessageType]

	stat.Status = wssMsg.Status
	stat.CorrelationID = wssMsg.CorrelationID
	stat.Output = wssMsg.Output
	stat.Error = wssMsg.Error

	cl.mux.RLock()
	rChan, isRPC := cl.correlData[stat.CorrelationID]
	cl.mux.RUnlock()

	// if rChan exists someone is waiting for this status update
	if isRPC {
		// it is up to the receiver to remove the correlation data after no more
		// messages are expected.
		// Warning: This channel is only 1 deep so if it isn't read this will block
		// on the next action reply message.
		// TODO: add timeout to recover from deadlock
		rChan <- &stat
		return
	} else if cl.messageHandler != nil {
		// pass the message to the client using its registered message handler
		msg := hubclient.NewThingMessage(
			op, "", "", wssMsg.Output, senderID)

		// pass event to client as this is an unsolicited push event
		// it could be a delayed confirmation of delivery
		cl.messageHandler(msg)
	} else {
		// missing rpc or message handler
		slog.Error("handleWSSMessage, no handler registered for client",
			"op", op,
			"clientID", cl.clientID)
		//stat.Status = StatusFailed
		//stat.Error = fmt.Errorf("handleSSEEvent no handler is set, delivery update ignored")
	}
}

// Agent receives an invoke/query action message.
// Pass it on to the handler and send back the result.
func (cl *WSSClient) handleActionMessage(jsonMsg string) {

	var stat hubclient.RequestStatus
	wssMsg := ActionMessage{}
	err := jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
	op := MsgTypeToOp[wssMsg.MessageType]
	_ = err
	// agent receives action request
	rxMsg := hubclient.NewThingMessage(
		op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, "")
	rxMsg.CorrelationID = wssMsg.CorrelationID
	rxMsg.MessageID = wssMsg.MessageID

	if cl.requestHandler == nil {
		// FIXME: return this as an error
		slog.Warn("handleWSSMessage, no request handler registered. Request ignored.",
			slog.String("op", op),
			slog.String("clientID", cl.clientID))
		return
	}
	// FIXME: convert messageType to operation
	// FIXME: get senderID
	stat = cl.requestHandler(rxMsg)
	stat.CorrelationID = rxMsg.CorrelationID
	if rxMsg.CorrelationID != "" {
		cl.PubActionStatus(stat) // send the result to the caller
	}
}

// handle receiving an event (pub/sub) message
func (cl *WSSClient) handleEventMessage(jsonMsg string) {

	var stat hubclient.RequestStatus
	wssMsg := EventMessage{}
	err := jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
	op := MsgTypeToOp[wssMsg.MessageType]
	_ = err
	// agent receives action request
	rxMsg := hubclient.NewThingMessage(
		op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, "")
	rxMsg.CorrelationID = wssMsg.CorrelationID
	rxMsg.MessageID = wssMsg.MessageID
	rxMsg.Timestamp = wssMsg.Timestamp
	stat = cl.requestHandler(rxMsg)
	stat.CorrelationID = rxMsg.CorrelationID
	if rxMsg.CorrelationID != "" {
		cl.PubActionStatus(stat) // send the result to the caller
	}
}

// handle receiving an property (pub/sub) message
func (cl *WSSClient) handlePropertyMessage(jsonMsg string) {

	var stat hubclient.RequestStatus
	wssMsg := PropertyMessage{}
	err := jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
	op := MsgTypeToOp[wssMsg.MessageType]
	_ = err
	// agent receives action request
	rxMsg := hubclient.NewThingMessage(
		op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, "")
	rxMsg.CorrelationID = wssMsg.CorrelationID
	rxMsg.MessageID = wssMsg.MessageID
	rxMsg.Timestamp = wssMsg.Timestamp
	stat = cl.requestHandler(rxMsg)
	stat.CorrelationID = rxMsg.CorrelationID
	if rxMsg.CorrelationID != "" {
		cl.PubActionStatus(stat) // send the result to the caller
	}
}

// handle receiving a TD update message
func (cl *WSSClient) handleTDMessage(jsonMsg string) {

	var stat hubclient.RequestStatus
	wssMsg := TDMessage{}
	err := jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
	op := MsgTypeToOp[wssMsg.MessageType]
	_ = err
	// agent receives action request
	rxMsg := hubclient.NewThingMessage(
		op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, "")
	rxMsg.CorrelationID = wssMsg.CorrelationID
	rxMsg.MessageID = wssMsg.MessageID
	rxMsg.Timestamp = wssMsg.Timestamp
	stat = cl.requestHandler(rxMsg)
	stat.CorrelationID = rxMsg.CorrelationID
	if rxMsg.CorrelationID != "" {
		cl.PubActionStatus(stat) // send the result to the caller
	}
}

// handleWSSMessage processes the push-message received from the hub.
func (cl *WSSClient) handleWSSMessage(jsonMsg string) {
	baseMsg := BaseMessage{}
	err := jsoniter.UnmarshalFromString(jsonMsg, &baseMsg)
	msgType := baseMsg.MessageType
	//senderID := ""            // what goes here?
	_ = err

	//cl.mux.RLock()
	//msgHandler := cl.messageHandler
	//reqHandler := cl.requestHandler
	//cl.mux.RUnlock()

	// always handle rpc response
	switch msgType {
	case MsgTypeActionStatus:
		cl.handleActionStatus(jsonMsg)

	// messages and requests are handled separately
	case MsgTypeInvokeAction, MsgTypeQueryAction, MsgTypeQueryAllActions:
		cl.handleActionMessage(jsonMsg)

	// messages and requests are handled separately
	case MsgTypeReadEvent, MsgTypeReadAllEvents:
		cl.handleEventMessage(jsonMsg)

	// messages and requests are handled separately
	case MsgTypeReadProperty, MsgTypeReadAllProperties, MsgTypeReadMultipleProperties,
		MsgTypeWriteProperty, MsgTypeWriteMultipleProperties:
		cl.handlePropertyMessage(jsonMsg)

	// messages and requests are handled separately
	case MsgTypeReadTD, MsgTypeUpdateTD:
		cl.handleTDMessage(jsonMsg)

	default:

		//// pass everything else to the message handler
		//// consumer receive event, property and TD updates
		//if msgHandler == nil {
		//	slog.Warn("handleWSSMessage, no message handler registered. Message ignored.",
		//		slog.String("op", op),
		//		slog.String("thingID", baseMsg.ThingID),
		//		slog.String("clientID", cl.clientID))
		//	return
		//}
		//msgHandler(&rxMsg)
	}
}
