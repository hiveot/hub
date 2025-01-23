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
func (cl *WssConsumerClient) wssToNotification(baseMsg wssserver_old.BaseMessage, raw []byte) (
	isNotification bool, notif transports.NotificationMessage) {

	var err error
	isNotification = true

	switch baseMsg.MessageType {
	case wssserver_old.MsgTypePublishEvent:
		wssMsg := wssserver_old.PropertyMessage{}
		err = cl.Unmarshal(raw, &wssMsg)
		notif = transports.NewNotificationResponse(
			wot.HTOpUpdateTD, wssMsg.ThingID, wssMsg.Name, wssMsg.Data)
		notif.Created = wssMsg.Timestamp

	case wssserver_old.MsgTypePropertyReading:
		wssMsg := wssserver_old.PropertyMessage{}
		err = cl.Unmarshal(raw, &wssMsg)
		notif = transports.NewNotificationResponse(
			wot.HTOpUpdateTD, wssMsg.ThingID, wssMsg.Name, wssMsg.Data)
		notif.Created = wssMsg.Timestamp

	case wssserver_old.MsgTypeUpdateTD:
		wssMsg := wssserver_old.TDMessage{}
		err = cl.Unmarshal(raw, &wssMsg)
		notif = transports.NewNotificationResponse(
			wot.HTOpUpdateTD, wssMsg.ThingID, wssMsg.Name, wssMsg.Data)
	default:
		isNotification = false
	}
	_ = err
	return isNotification, notif
}

// Convert a Websocket message to the unified response message
// unified response messages carry the operation from the request.
func (cl *WssConsumerClient) wssToResponse(baseMsg wssserver_old.BaseMessage, raw []byte) (
	isResponse bool, resp *transports.ResponseMessage) {

	isResponse = true

	switch baseMsg.MessageType {
	case wssserver_old.MsgTypeActionStatus:
		wssMsg := wssserver_old.ActionStatusMessage{}
		err := cl.Unmarshal(raw, &wssMsg)
		if wssMsg.Error != "" {
			err = errors.New(wssMsg.Error)
		}
		resp = transports.NewResponseMessage(wot.OpInvokeAction,
			wssMsg.ThingID, wssMsg.Name, wssMsg.Output, err, baseMsg.CorrelationID)

	case wssserver_old.MsgTypePong:
		wssMsg := wssserver_old.ActionStatusMessage{}
		_ = cl.Unmarshal(raw, &wssMsg)
		resp = transports.NewResponseMessage(wot.HTOpPing,
			"", "", wssMsg.Output, nil, baseMsg.CorrelationID)
		resp.Updated = wssMsg.Timestamp

	case wssserver_old.MsgTypeError:
		wssMsg := wssserver_old.ErrorMessage{}
		err := cl.Unmarshal(raw, &wssMsg)
		if err == nil {
			err = errors.New(wssMsg.Title)
		}
		resp = transports.NewResponseMessage(wot.OpInvokeAction,
			wssMsg.ThingID, wssMsg.Name, wssMsg.Detail, err, baseMsg.CorrelationID)

	case wssserver_old.MsgTypePropertyReading:
		wssMsg := wssserver_old.PropertyMessage{}
		err := cl.Unmarshal(raw, &wssMsg)
		resp = transports.NewResponseMessage(wot.OpReadProperty,
			wssMsg.ThingID, wssMsg.Name, wssMsg.Data, err, baseMsg.CorrelationID)
		resp.Updated = wssMsg.Timestamp

	case wssserver_old.MsgTypePropertyReadings:
		wssMsg := wssserver_old.PropertyMessage{}
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
func (cl *WssConsumerClient) HandleWssMessage(baseMsg wssserver_old.BaseMessage, raw []byte) {
	var err error
	msgType := baseMsg.MessageType
	correlationID := baseMsg.CorrelationID
	slog.Info("WssClientHandleMessage",
		slog.String("clientID", cl.GetClientID()),
		slog.String("msgType", msgType),
		slog.String("correlationID", correlationID),
	)
	operation := wssserver_old.MsgTypeToOp[baseMsg.MessageType]

	switch msgType {

	// responses (some of them can also be notifications)
	case wssserver_old.MsgTypeActionStatus,
		wssserver_old.MsgTypePong,
		wssserver_old.MsgTypeError,
		wssserver_old.MsgTypePropertyReading, wssserver_old.MsgTypePropertyReadings:
		isResponse, resp := cl.wssToResponse(baseMsg, raw)
		if isResponse {
			cl.OnResponse(resp)
		}

	// notifications
	case wssserver_old.MsgTypePublishEvent:
	case wssserver_old.MsgTypeUpdateTD:
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
