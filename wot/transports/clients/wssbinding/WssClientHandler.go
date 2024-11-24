package wssbinding

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/wot/transports"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
)

// handle receiving an action status update
// this needs a message handler to pass it to.
func (cl *WssBindingClient) handleActionStatus(jsonMsg string) {
	var senderID = ""
	var stat transports.RequestStatus

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
		// TODO: add timeout to recover from deadlock if rChan isn't read
		rChan <- &stat
		return
	} else if cl.messageHandler != nil {
		// pass the message to the client using its registered message handler
		msg := transports.NewThingMessage(
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

// Agent receives an invoke/query action request message.
// Pass it on to the handler and send the result back to the hub.
func (cl *WssBindingClient) handleActionMessage(jsonMsg string) {

	var stat transports.RequestStatus
	wssMsg := ActionMessage{}
	err := jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
	op := MsgTypeToOp[wssMsg.MessageType]
	_ = err
	// agent receives action request
	rxMsg := transports.NewThingMessage(
		op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, wssMsg.SenderID)
	rxMsg.CorrelationID = wssMsg.CorrelationID
	rxMsg.MessageID = wssMsg.MessageID

	if cl.requestHandler == nil {
		// FIXME: return this as an error
		slog.Warn("handleWSSMessage, no request handler registered. Request ignored.",
			slog.String("op", op),
			slog.String("clientID", cl.clientID))
		return
	}
	stat = cl.requestHandler(rxMsg)
	stat.CorrelationID = rxMsg.CorrelationID
	if rxMsg.CorrelationID != "" {
		cl.PubActionStatus(stat) // send the result to the caller
	}
}

// handler receiving an event message from agent.
// This does not send a confirmation reply.
func (cl *WssBindingClient) handleEventMessage(jsonMsg string) {

	wssMsg := EventMessage{}
	err := jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
	op := MsgTypeToOp[wssMsg.MessageType]
	_ = err
	// agent receives action request
	rxMsg := transports.NewThingMessage(
		op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, "")
	rxMsg.CorrelationID = wssMsg.CorrelationID
	rxMsg.MessageID = wssMsg.MessageID
	rxMsg.Timestamp = wssMsg.Timestamp
	cl.messageHandler(rxMsg)
}

// handle receiving an property (pub/sub) message
// property-write messages send a action status result if a correlationID is provided
func (cl *WssBindingClient) handlePropertyMessage(jsonMsg string) {

	var stat transports.RequestStatus
	wssMsg := PropertyMessage{}
	err := jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
	op := MsgTypeToOp[wssMsg.MessageType]
	_ = err
	// agent receives action request
	rxMsg := transports.NewThingMessage(
		op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, "")
	rxMsg.CorrelationID = wssMsg.CorrelationID
	rxMsg.MessageID = wssMsg.MessageID
	rxMsg.Timestamp = wssMsg.Timestamp
	if op == vocab.OpWriteProperty || op == vocab.OpWriteMultipleProperties {
		stat = cl.requestHandler(rxMsg)
		stat.CorrelationID = rxMsg.CorrelationID
		if rxMsg.CorrelationID != "" {
			cl.PubActionStatus(stat) // send the result to the caller
		}
	} else {
		// property reading notification is a response to a read or observe request
		// TODO: match with request
		cl.messageHandler(rxMsg)
	}
}

// handle receiving a TD update message
// this does not send a status update
func (cl *WssBindingClient) handleTDMessage(jsonMsg string) {

	wssMsg := TDMessage{}
	err := jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
	op := MsgTypeToOp[wssMsg.MessageType]
	_ = err
	// agent receives action request
	rxMsg := transports.NewThingMessage(
		op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, "")
	rxMsg.CorrelationID = wssMsg.CorrelationID
	rxMsg.MessageID = wssMsg.MessageID
	rxMsg.Timestamp = wssMsg.Timestamp
	cl.messageHandler(rxMsg)
}

// handleWSSMessage processes the push-message received from the hub.
func (cl *WssBindingClient) handleWSSMessage(jsonMsg string) {
	baseMsg := BaseMessage{}
	err := jsoniter.UnmarshalFromString(jsonMsg, &baseMsg)
	msgType := baseMsg.MessageType
	//senderID := ""            // what goes here?
	_ = err

	//cl.mux.RLock()
	//msgHandler := cl.messageHandler
	//reqHandler := cl.requestHandler
	//cl.mux.RUnlock()

	switch msgType {
	// handle an action status update response
	case MsgTypeActionStatus:
		cl.handleActionStatus(jsonMsg)

	// handle action related messages
	case MsgTypeInvokeAction, MsgTypeQueryAction, MsgTypeQueryAllActions:
		cl.handleActionMessage(jsonMsg)

	// handle event related messages
	case MsgTypeReadEvent, MsgTypeReadAllEvents, MsgTypePublishEvent:
		cl.handleEventMessage(jsonMsg)

	// handle property related messages
	case MsgTypeReadProperty, MsgTypeReadAllProperties, MsgTypeReadMultipleProperties,
		MsgTypeWriteProperty, MsgTypeWriteMultipleProperties,
		MsgTypePropertyReading, MsgTypePropertyReadings:
		cl.handlePropertyMessage(jsonMsg)

	// handle TD related messages
	case MsgTypeReadTD, MsgTypeUpdateTD:
		cl.handleTDMessage(jsonMsg)

	default:
		slog.Warn("Unknown message type", "msgType", msgType)
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
