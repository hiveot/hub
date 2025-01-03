package wssclient

import (
	"errors"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/servers/wssserver"
	"github.com/hiveot/hub/wot"
	"log/slog"
)

// This file contains handlers for consumer messages.

// Convert a Websocket message to the unified notification message
func (cl *WssConsumerClient) wssToNotification(baseMsg wssserver.BaseMessage, raw []byte) (
	isNotification bool, notif transports.NotificationMessage) {

	var err error
	isNotification = true

	switch baseMsg.MessageType {
	case wssserver.MsgTypePublishEvent:
		wssMsg := wssserver.PropertyMessage{}
		err = cl.Unmarshal(raw, &wssMsg)
		notif = transports.NewNotificationMessage(
			wot.HTOpUpdateTD, wssMsg.ThingID, wssMsg.Name, wssMsg.Data)
		notif.Created = wssMsg.Timestamp

	case wssserver.MsgTypePropertyReading:
		wssMsg := wssserver.PropertyMessage{}
		err = cl.Unmarshal(raw, &wssMsg)
		notif = transports.NewNotificationMessage(
			wot.HTOpUpdateTD, wssMsg.ThingID, wssMsg.Name, wssMsg.Data)
		notif.Created = wssMsg.Timestamp

	case wssserver.MsgTypeUpdateTD:
		wssMsg := wssserver.TDMessage{}
		err = cl.Unmarshal(raw, &wssMsg)
		notif = transports.NewNotificationMessage(
			wot.HTOpUpdateTD, wssMsg.ThingID, wssMsg.Name, wssMsg.Data)
	default:
		isNotification = false
	}
	_ = err
	return isNotification, notif
}

// Convert a Websocket message to the unified response message
// unified response messages carry the operation from the request.
func (cl *WssConsumerClient) wssToResponse(baseMsg wssserver.BaseMessage, raw []byte) (
	isResponse bool, resp transports.ResponseMessage) {

	isResponse = true

	switch baseMsg.MessageType {
	case wssserver.MsgTypeActionStatus:
		wssMsg := wssserver.ActionStatusMessage{}
		err := cl.Unmarshal(raw, &wssMsg)
		if wssMsg.Error != "" {
			err = errors.New(wssMsg.Error)
		}
		resp = transports.NewResponseMessage(wot.OpInvokeAction,
			wssMsg.ThingID, wssMsg.Name, wssMsg.Output, err, baseMsg.CorrelationID)

	case wssserver.MsgTypePong:
		resp = transports.NewResponseMessage(wot.HTOpPong,
			"", "", nil, nil, baseMsg.CorrelationID)

	case wssserver.MsgTypeError:
		wssMsg := wssserver.ErrorMessage{}
		err := cl.Unmarshal(raw, &wssMsg)
		if err == nil {
			err = errors.New(wssMsg.Title)
		}
		resp = transports.NewResponseMessage(wot.OpInvokeAction,
			wssMsg.ThingID, wssMsg.Name, wssMsg.Detail, err, baseMsg.CorrelationID)

	case wssserver.MsgTypePropertyReading:
		wssMsg := wssserver.PropertyMessage{}
		err := cl.Unmarshal(raw, &wssMsg)
		resp = transports.NewResponseMessage(wot.OpReadProperty,
			wssMsg.ThingID, wssMsg.Name, wssMsg.Data, err, baseMsg.CorrelationID)
		resp.Updated = wssMsg.Timestamp

	case wssserver.MsgTypePropertyReadings:
		wssMsg := wssserver.PropertyMessage{}
		err := cl.Unmarshal(raw, &wssMsg)
		resp = transports.NewResponseMessage(wot.OpReadAllProperties,
			wssMsg.ThingID, wssMsg.Name, wssMsg.Data, err, baseMsg.CorrelationID)
		resp.Updated = wssMsg.Timestamp
	default:
		isResponse = false
	}
	return isResponse, resp
}

// HandleWssMessage processes the websocket message received from the server.
func (cl *WssConsumerClient) HandleWssMessage(baseMsg wssserver.BaseMessage, raw []byte) {
	var err error
	msgType := baseMsg.MessageType
	correlationID := baseMsg.CorrelationID
	slog.Info("WssClientHandleMessage",
		slog.String("clientID", cl.GetClientID()),
		slog.String("msgType", msgType),
		slog.String("correlationID", correlationID),
	)
	operation := wssserver.MsgTypeToOp[baseMsg.MessageType]

	switch msgType {

	// responses (some of them can also be notifications)
	case wssserver.MsgTypeActionStatus,
		wssserver.MsgTypePong,
		wssserver.MsgTypeError,
		wssserver.MsgTypePropertyReading, wssserver.MsgTypePropertyReadings:
		isResponse, resp := cl.wssToResponse(baseMsg, raw)
		if isResponse {
			cl.OnResponse(resp)
		}

	// notifications
	case wssserver.MsgTypePublishEvent:
	case wssserver.MsgTypeUpdateTD:
		isNotification, notif := cl.wssToNotification(baseMsg, raw)
		if isNotification {
			cl.OnNotification(notif)
		}

	default:
		// everything else is supposed to be a request
		if cl.agentRequestHandler == nil {
			slog.Warn("HandleWssMessage: Unknown message type",
				"msgType", msgType,
				"clientID", cl.GetClientID())
		} else {
			cl.agentRequestHandler(baseMsg, raw)
		}
	}
	if err != nil {
		slog.Warn("invalid message", "op", operation)
	}
}
