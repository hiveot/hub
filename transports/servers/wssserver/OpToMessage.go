// Package wssbinding with requests made by consumers
package wssserver

import (
	"errors"
	"github.com/hiveot/hub/wot"
	"github.com/teris-io/shortid"
	"time"
)

// const WSSOpPing = "wss-ping"
const WSSOpConnect = "wss-connect"

func opToMessageType(op string) string {
	// yeah not very efficient. FIXME
	for k, v := range MsgTypeToOp {
		if v == op {
			return k
		}
	}
	return ""
}

// OpToMessage creates a websocket message for an operation
func OpToMessage(
	op string, dThingID string, name string, names []string,
	data any, requestID string, senderID string) (msg any, err error) {

	if requestID == "" {
		requestID = shortid.MustGenerate()
	}
	msgType := opToMessageType(op)
	timestamp := time.Now().Format(wot.RFC3339Milli)

	switch op {
	case wot.OpInvokeAction, wot.OpQueryAllActions, wot.OpQueryAction:
		msg = ActionMessage{
			ThingID:     dThingID,
			MessageType: msgType,
			Name:        name,
			RequestID:   requestID,
			SenderID:    senderID,
			Data:        data,
			Timestamp:   timestamp,
		}
	case wot.HTOpActionStatus:
		msg = ActionStatusMessage{
			ThingID:     dThingID,
			MessageType: msgType,
			Status:      "completed",
			Name:        name,
			RequestID:   requestID,
			Output:      data,
			Timestamp:   timestamp,
		}

	case wot.OpObserveAllProperties, wot.OpObserveProperty,
		wot.OpUnobserveAllProperties, wot.OpUnobserveProperty,
		wot.OpReadAllProperties, wot.OpReadProperty, wot.OpReadMultipleProperties,
		wot.OpWriteProperty, wot.HTOpUpdateProperty:
		msg = PropertyMessage{
			ThingID:     dThingID,
			MessageType: msgType,
			Name:        name,
			Names:       names,
			Data:        data,
			RequestID:   requestID,
			Timestamp:   timestamp,
		}
	case wot.HTOpReadAllEvents, wot.HTOpReadEvent,
		//wot.HTOpReadMultipleEvents,
		wot.OpSubscribeEvent, wot.OpSubscribeAllEvents,
		wot.OpUnsubscribeEvent, wot.OpUnsubscribeAllEvents,
		wot.HTOpEvent,
		wot.HTOpPing, wot.HTOpPong:
		msg = EventMessage{
			ThingID:     dThingID,
			MessageType: msgType,
			Name:        name,
			Names:       names,
			Data:        data,
			RequestID:   requestID,
			Timestamp:   timestamp,
		}
	case wot.HTOpReadTD, wot.HTOpReadAllTDs,
		wot.HTOpUpdateTD:
		msg = TDMessage{
			ThingID:     dThingID,
			MessageType: msgType,
			Name:        name,
			Data:        data,
			RequestID:   requestID,
			Timestamp:   timestamp,
		}
	default:
		err = errors.New("Unknown operation")
	}
	return msg, err
}
