package wssserver

import (
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/clients/wssbinding"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"log/slog"
	"time"
)

// Handle incoming requests

// WssServerHandleMessage handles an incoming websocket message
// This separates it in messages and requests and passes it to the message handler
// for further processing.
func (c *WssServerConnection) WssServerHandleMessage(jsonMsg string) {
	var msg *transports.ThingMessage
	msgAsMap := map[string]any{}

	// the operation is needed to determine whether this is a request or send and forget message
	// unfortunately this might lead to double unmarshalling :(
	err := jsoniter.UnmarshalFromString(jsonMsg, &msgAsMap)
	if err != nil {
		slog.Warn("_receive: unmarshalling message failed. Message ignored.",
			"clientID", c.clientID,
			"err", err.Error())
		return
	}
	messageType := utils.DecodeAsString(msgAsMap["messageType"])
	op, _ := wssbinding.MsgTypeToOp[messageType]
	slog.Info("WssServerHandleMessage: Received message",
		"clientID", c.clientID,
		"messageType", messageType,
		"correlationID", msgAsMap["correlationId"])

	switch messageType {

	case wssbinding.MsgTypeActionStatus:
		// hub receives an action result from an agent
		// this will be forwarded to the consumer as a message
		wssMsg := wssbinding.ActionStatusMessage{}
		_ = utils.DecodeAsObject(msgAsMap, &wssMsg)

		// convert the websocket action status message to a hub status
		// these are similar so maybe overkill?
		hubRequestStatus := transports.RequestStatus{
			ThingID:       wssMsg.ThingID,
			Name:          wssMsg.Name,
			CorrelationID: wssMsg.CorrelationID,
			Status:        wssMsg.Status, // FIXME: convert status codes
			Error:         wssMsg.Error,
			Output:        wssMsg.Output,
		}

		msg = transports.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, hubRequestStatus, c.clientID)
		msg.CorrelationID = wssMsg.CorrelationID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		c.ForwardAsEvent(msg)

	case // hub receives action messages from a consumer
		wssbinding.MsgTypeInvokeAction,
		wssbinding.MsgTypeQueryAction,
		wssbinding.MsgTypeQueryAllActions:
		// map the message to a ThingMessage
		wssMsg := wssbinding.ActionMessage{}
		_ = utils.DecodeAsObject(msgAsMap, &wssMsg)
		msg = transports.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, c.clientID)
		msg.CorrelationID = wssMsg.CorrelationID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		c.ForwardAsRequest(msg)

	case // hub receives event action messages
		wssbinding.MsgTypeReadAllEvents,
		//wssbinding.MsgTypeReadMultipleEvents,
		wssbinding.MsgTypePublishEvent,
		wssbinding.MsgTypeReadEvent:
		// map the message to a ThingMessage
		wssMsg := wssbinding.EventMessage{}
		_ = utils.DecodeAsObject(msgAsMap, &wssMsg)
		msg = transports.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, c.clientID)
		msg.CorrelationID = wssMsg.CorrelationID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		if messageType == wssbinding.MsgTypePublishEvent {
			c.ForwardAsEvent(msg)
		} else {
			c.ForwardAsRequest(msg)
		}

	case // digital twin property messages
		wssbinding.MsgTypeReadAllProperties,
		wssbinding.MsgTypeReadMultipleProperties,
		wssbinding.MsgTypeReadProperty,
		wssbinding.MsgTypeWriteMultipleProperties,
		wssbinding.MsgTypeWriteProperty,
		wssbinding.MsgTypePropertyReadings, // agent publishes properties update
		wssbinding.MsgTypePropertyReading:  // agent publishes property update
		// map the message to a ThingMessage
		wssMsg := wssbinding.PropertyMessage{}
		_ = utils.DecodeAsObject(msgAsMap, &wssMsg)
		// FIXME: readmultiple has an array of names
		msg = transports.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, c.clientID)
		msg.CorrelationID = wssMsg.CorrelationID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		if wssMsg.MessageType == wssbinding.MsgTypePropertyReading ||
			wssMsg.MessageType == wssbinding.MsgTypePropertyReadings {
			c.ForwardAsEvent(msg)
		} else {
			c.ForwardAsRequest(msg)
		}
		// td messages
	case wssbinding.MsgTypeReadTD:
		wssMsg := wssbinding.TDMessage{}
		_ = utils.DecodeAsObject(msgAsMap, &wssMsg)
		msg = transports.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, c.clientID)
		msg.CorrelationID = wssMsg.CorrelationID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		c.ForwardAsRequest(msg)

	case wssbinding.MsgTypeUpdateTD:
		wssMsg := wssbinding.TDMessage{}
		_ = utils.DecodeAsObject(msgAsMap, &wssMsg)
		msg = transports.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, c.clientID)
		msg.CorrelationID = wssMsg.CorrelationID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		c.ForwardAsEvent(msg)

	// subscriptions are handled inside this binding
	case wssbinding.MsgTypeObserveAllProperties:
		wssMsg := wssbinding.PropertyMessage{}
		err = jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
		c.HandleObserveAllProperties(&wssMsg)
	//case wssbinding.MsgTypeObserveMultipleProperties:
	//	wssMsg := wssbinding.PropertyMessage{}
	//	err = jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
	//	c.HandleObserveMultipleProperties(&wssMsg)
	case wssbinding.MsgTypeObserveProperty:
		wssMsg := wssbinding.PropertyMessage{}
		err = jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
		c.HandleObserveProperty(&wssMsg)
	case wssbinding.MsgTypeSubscribeAllEvents:
		wssMsg := wssbinding.EventMessage{}
		err = jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
		c.HandleSubscribeAllEvents(&wssMsg)
	case wssbinding.MsgTypeSubscribeEvent:
		wssMsg := wssbinding.EventMessage{}
		err = jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
		c.HandleSubscribeEvent(&wssMsg)
	case wssbinding.MsgTypeUnobserveAllProperties:
		wssMsg := wssbinding.PropertyMessage{}
		err = jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
		c.HandleUnobserveAllProperties(&wssMsg)
	case wssbinding.MsgTypeUnobserveProperty:
		wssMsg := wssbinding.PropertyMessage{}
		err = jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
		c.HandleUnobserveProperty(&wssMsg)
	case wssbinding.MsgTypeUnsubscribeAllEvents:
		wssMsg := wssbinding.EventMessage{}
		err = jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
		c.HandleUnsubscribeAllEvents(&wssMsg)
	case wssbinding.MsgTypeUnsubscribeEvent:
		wssMsg := wssbinding.EventMessage{}
		err = jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
		c.HandleUnsubscribeEvent(&wssMsg)

	// other messages handled inside this binding
	case wssbinding.MsgTypePing:
		wssMsg := wssbinding.BaseMessage{}
		_ = jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
		c.HandlePing(&wssMsg)

	default:
		slog.Warn("_receive: unknown operation", "messageType", messageType)
	}
}

// event style messages are pushed to the digitwin router
func (c *WssServerConnection) ForwardAsEvent(msg *transports.ThingMessage) {
	c.requestHandler(msg, nil)
}

// request style messages are pushed to the digitwin router
func (c *WssServerConnection) ForwardAsRequest(msg *transports.ThingMessage) {
	stat := c.requestHandler(msg, c)

	// FIXME: map status between protocols
	wssStatus := stat.Status
	reply := wssbinding.ActionStatusMessage{
		MessageType:   wssbinding.MsgTypeActionStatus,
		Status:        wssStatus,
		Output:        stat.Output,
		TimeRequested: msg.Created,
		TimeEnded:     time.Now().Format(utils.RFC3339Milli),
		Timestamp:     time.Now().Format(utils.RFC3339Milli),
		MessageID:     shortid.MustGenerate(),
		CorrelationID: msg.CorrelationID,
	}
	if stat.Error != "" {
		reply.Error = stat.Error
	}
	_, _ = c._send(reply)
}

func (c *WssServerConnection) HandleObserveAllProperties(wssMsg *wssbinding.PropertyMessage) {
	c.observations.SubscribeAll(wssMsg.ThingID)
}

func (c *WssServerConnection) HandleObserveProperty(wssMsg *wssbinding.PropertyMessage) {
	c.observations.Subscribe(wssMsg.ThingID, wssMsg.Name)
}

func (c *WssServerConnection) HandlePing(wssMsg *wssbinding.BaseMessage) {
	pongMessage := *wssMsg
	pongMessage.MessageType = wssbinding.MsgTypePong
	_, _ = c._send(pongMessage)
}

func (c *WssServerConnection) HandleSubscribeAllEvents(wssMsg *wssbinding.EventMessage) {
	c.subscriptions.SubscribeAll(wssMsg.ThingID)
}

func (c *WssServerConnection) HandleSubscribeEvent(wssMsg *wssbinding.EventMessage) {
	c.subscriptions.Subscribe(wssMsg.ThingID, wssMsg.Name)
}
func (c *WssServerConnection) HandleUnobserveAllProperties(wssMsg *wssbinding.PropertyMessage) {
	c.observations.UnsubscribeAll(wssMsg.ThingID)
}

func (c *WssServerConnection) HandleUnobserveProperty(wssMsg *wssbinding.PropertyMessage) {
	c.observations.Unsubscribe(wssMsg.ThingID, wssMsg.Name)
}
func (c *WssServerConnection) HandleUnsubscribeAllEvents(wssMsg *wssbinding.EventMessage) {
	c.subscriptions.UnsubscribeAll(wssMsg.ThingID)
}

func (c *WssServerConnection) HandleUnsubscribeEvent(wssMsg *wssbinding.EventMessage) {
	c.subscriptions.Unsubscribe(wssMsg.ThingID, wssMsg.Name)
}
