package wssclient

import (
	"fmt"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/transports"
	"log/slog"
)

// Pass the given message to the registered notification handler
func (cl *WssClientConnection) _handleAsNotification(msg *transports.ThingMessage) {
	if cl.BaseNotificationHandler != nil {
		cl.BaseNotificationHandler(msg)
	} else {
		slog.Warn("received notification but no handler is registered",
			"clientID", cl.BaseClientID)
	}
}

// [Agent] receives a request. Pass the message to the registered request
// handler and send a reply if the request has a requestID.
func (cl *WssClientConnection) _handleAsRequest(msg *transports.ThingMessage) {
	var output any
	var err error
	if cl.BaseRequestHandler == nil {
		// FIXME: return this as an error
		err = fmt.Errorf(
			"_handleAsRequest, no request handler registered for agent %s.",
			cl.BaseClientID)
	} else {
		output, err = cl.BaseRequestHandler(msg)
	}
	// if a requestID is provided then send a reply
	if msg.RequestID != "" {
		if err != nil {
			cl.SendError(msg.ThingID, msg.Name, err, msg.RequestID)
		} else {
			cl.SendResponse(msg.ThingID, msg.Name, output, msg.RequestID)
		}
	}
	if err != nil {
		slog.Error(err.Error())
	}
}

// handle receiving an action status update.
// This can be a response to a non-rpc request, or an update to a prior RPC
// request that already received a response.
// This is passed to the client as a notification.
func (cl *WssClientConnection) handleActionStatus(requestID string, raw []byte) {
	var senderID = ""

	wssMsg := ActionStatusMessage{}
	_ = cl.Unmarshal(raw, &wssMsg)
	op := MsgTypeToOp[wssMsg.MessageType]

	// if this is an RPC message then handle it now
	isRPC := cl.BaseRnrChan.HandleResponse(requestID, wssMsg.Output)
	if isRPC {
		// Good, this was a known RPC request. It is handled by the channel listener.
		return
	}

	msg := transports.NewThingMessage(
		op, "", "", wssMsg.Output, senderID)
	cl._handleAsNotification(msg)
}

// Agent receives an action invoke/query request.
// Pass it on to the client handler and send the result back to the
// server asynchronously.
func (cl *WssClientConnection) handleActionMessage(raw []byte) {

	wssMsg := ActionMessage{}
	err := cl.Unmarshal(raw, &wssMsg)
	op := MsgTypeToOp[wssMsg.MessageType]
	_ = err
	// agent receives action request
	rxMsg := transports.NewThingMessage(
		op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, "")
	rxMsg.RequestID = wssMsg.RequestID
	rxMsg.MessageID = wssMsg.MessageID

	cl._handleAsRequest(rxMsg)
}

// handler receiving an event message from agent.
// Pass it on to the client handler as a notification.
func (cl *WssClientConnection) handleEventMessage(raw []byte) {

	wssMsg := EventMessage{}
	err := cl.Unmarshal(raw, &wssMsg)
	op := MsgTypeToOp[wssMsg.MessageType]
	_ = err
	// agent receives action request
	rxMsg := transports.NewThingMessage(
		op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, "")
	rxMsg.RequestID = wssMsg.RequestID
	rxMsg.MessageID = wssMsg.MessageID
	rxMsg.Timestamp = wssMsg.Timestamp
	cl._handleAsNotification(rxMsg)
}

// handle a property write request or update notification message.
// property-write messages send an action status result if a requestID is provided
// notifications tell the client an observed property changed value.
func (cl *WssClientConnection) handlePropertyMessage(raw []byte) {

	wssMsg := PropertyMessage{}
	err := cl.Unmarshal(raw, &wssMsg)
	op := MsgTypeToOp[wssMsg.MessageType]
	_ = err
	// agent receives action request
	rxMsg := transports.NewThingMessage(
		op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, "")
	rxMsg.RequestID = wssMsg.RequestID
	rxMsg.MessageID = wssMsg.MessageID
	rxMsg.Timestamp = wssMsg.Timestamp

	if op == wot.OpWriteProperty || op == wot.OpWriteMultipleProperties {
		// Property write is sent by the server from a consumer. A status update is expected.
		output, err := cl.BaseRequestHandler(rxMsg)
		if rxMsg.RequestID != "" {
			// send the result to the caller
			if err != nil {
				cl.SendResponse(wssMsg.ThingID, wssMsg.Name, output, wssMsg.RequestID)
			} else {
				cl.SendResponse(wssMsg.ThingID, wssMsg.Name, err, wssMsg.RequestID)
			}
		}
	} else {
		// Observed property value change.
		// property reading notification is sent by an agent as a response
		// to a read or observe request
		cl._handleAsNotification(rxMsg)
	}
}

// handle receiving a TD update message
// This is handled as a notification.
func (cl *WssClientConnection) handleTDMessage(raw []byte) {

	wssMsg := TDMessage{}
	err := cl.Unmarshal(raw, &wssMsg)
	op := MsgTypeToOp[wssMsg.MessageType]
	_ = err
	// agent receives action request
	rxMsg := transports.NewThingMessage(
		op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, "")
	rxMsg.RequestID = wssMsg.RequestID
	rxMsg.MessageID = wssMsg.MessageID
	rxMsg.Timestamp = wssMsg.Timestamp
	cl._handleAsNotification(rxMsg)
}

// WssClientHandleMessage processes the push-message received from the hub.
func (cl *WssClientConnection) WssClientHandleMessage(raw []byte) {
	baseMsg := BaseMessage{}
	err := cl.Unmarshal(raw, &baseMsg)
	if err != nil {
		slog.Error("WssClientConnection: message is not a valid websocket message",
			"clientID", cl.BaseClientID,
			"message size", len(raw))
		return
	}
	msgType := baseMsg.MessageType
	requestID := baseMsg.RequestID

	switch msgType {
	// handle an action status update response
	case MsgTypeActionStatus, MsgTypePong:
		cl.handleActionStatus(requestID, raw)

	// handle action related messages
	case MsgTypeInvokeAction, MsgTypeQueryAction, MsgTypeQueryAllActions:
		cl.handleActionMessage(raw)

	// handle event related messages
	case MsgTypeReadEvent, MsgTypeReadAllEvents, MsgTypePublishEvent:
		cl.handleEventMessage(raw)

	// handle property related messages
	case MsgTypeReadProperty, MsgTypeReadAllProperties, MsgTypeReadMultipleProperties,
		MsgTypeWriteProperty, MsgTypeWriteMultipleProperties,
		MsgTypePropertyReading, MsgTypePropertyReadings:
		cl.handlePropertyMessage(raw)

	// handle TD related messages
	case MsgTypeReadTD, MsgTypeUpdateTD:
		cl.handleTDMessage(raw)

	default:
		slog.Warn("Unknown message type", "msgType", msgType)
	}
}
