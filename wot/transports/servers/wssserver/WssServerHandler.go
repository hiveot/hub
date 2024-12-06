package wssserver

import (
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/clients/wssclient"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
)

// Handle incoming messages from clients
// These are converted into a standard ThingMessage envelope and passed to
// the handler.

// ForwardAsNotification message is notification for one or multiple clients,
// depending on the operation.
func (c *WssServerConnection) ForwardAsNotification(msg *transports.ThingMessage) {
	c.messageHandler(msg, nil)
}

// ForwardAsRequest message is a request style messages to be sent to a destination
func (c *WssServerConnection) ForwardAsRequest(msg *transports.ThingMessage) {
	c.messageHandler(msg, c)
}

func (c *WssServerConnection) HandleObserveAllProperties(wssMsg *wssclient.PropertyMessage) {
	c.observations.SubscribeAll(wssMsg.ThingID)
}

func (c *WssServerConnection) HandleObserveProperty(wssMsg *wssclient.PropertyMessage) {
	c.observations.Subscribe(wssMsg.ThingID, wssMsg.Name)
}

// HandlePing replies with pong to a ping message
func (c *WssServerConnection) HandlePing(wssMsg *wssclient.BaseMessage) {
	pongMessage := *wssMsg
	pongMessage.MessageType = wssclient.MsgTypePong
	c._send(pongMessage)
}

func (c *WssServerConnection) HandleSubscribeAllEvents(wssMsg *wssclient.EventMessage) {
	c.subscriptions.SubscribeAll(wssMsg.ThingID)
}

func (c *WssServerConnection) HandleSubscribeEvent(wssMsg *wssclient.EventMessage) {
	c.subscriptions.Subscribe(wssMsg.ThingID, wssMsg.Name)
}
func (c *WssServerConnection) HandleUnobserveAllProperties(wssMsg *wssclient.PropertyMessage) {
	c.observations.UnsubscribeAll(wssMsg.ThingID)
}

func (c *WssServerConnection) HandleUnobserveProperty(wssMsg *wssclient.PropertyMessage) {
	c.observations.Unsubscribe(wssMsg.ThingID, wssMsg.Name)
}

func (c *WssServerConnection) HandleUnsubscribeAllEvents(wssMsg *wssclient.EventMessage) {
	c.subscriptions.UnsubscribeAll(wssMsg.ThingID)
}

func (c *WssServerConnection) HandleUnsubscribeEvent(wssMsg *wssclient.EventMessage) {
	c.subscriptions.Unsubscribe(wssMsg.ThingID, wssMsg.Name)
}

// Marshal encodes the native data into the wire format
func (c *WssServerConnection) Marshal(data any) []byte {
	jsonData, _ := jsoniter.Marshal(data)
	return jsonData
}

// Unmarshal decodes the connection wire format to native data
func (c *WssServerConnection) Unmarshal(raw []byte, reply interface{}) error {
	err := jsoniter.Unmarshal(raw, reply)
	return err
}

// WssServerHandleMessage handles an incoming websocket message from a client
func (c *WssServerConnection) WssServerHandleMessage(raw []byte) {
	var msg *transports.ThingMessage
	baseMsg := wssclient.BaseMessage{}

	// the operation is needed to determine whether this is a request or send and forget message
	// unfortunately this needs double unmarshalling :(
	err := c.Unmarshal(raw, &baseMsg)
	if err != nil {
		slog.Warn("_receive: unmarshalling message failed. Message ignored.",
			"clientID", c.clientID,
			"err", err.Error())
		return
	}
	op, _ := wssclient.MsgTypeToOp[baseMsg.MessageType]
	slog.Info("WssServerHandleMessage: Received message",
		"clientID", c.clientID,
		"messageType", baseMsg.MessageType,
		"correlationID", baseMsg.RequestID)

	switch baseMsg.MessageType {

	case wssclient.MsgTypeActionStatus:
		// hub receives an action result from an agent
		// this will be forwarded to the consumer as a message
		wssMsg := wssclient.ActionStatusMessage{}
		_ = c.Unmarshal(raw, &wssMsg)

		msg = transports.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Output, c.clientID)
		msg.RequestID = wssMsg.RequestID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		c.ForwardAsNotification(msg)

	case // hub receives action messages from a consumer
		wssclient.MsgTypeInvokeAction,
		wssclient.MsgTypeQueryAction,
		wssclient.MsgTypeQueryAllActions:
		// map the message to a ThingMessage
		wssMsg := wssclient.ActionMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		msg = transports.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, c.clientID)
		msg.RequestID = wssMsg.RequestID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		c.ForwardAsRequest(msg)

	case // hub receives event action messages
		wssclient.MsgTypeReadAllEvents,
		//wssbinding.MsgTypeReadMultipleEvents,
		wssclient.MsgTypePublishEvent,
		wssclient.MsgTypeReadEvent:
		// map the message to a ThingMessage
		wssMsg := wssclient.EventMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		msg = transports.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, c.clientID)
		msg.RequestID = wssMsg.RequestID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		if baseMsg.MessageType == wssclient.MsgTypePublishEvent {
			c.ForwardAsNotification(msg)
		} else {
			c.ForwardAsRequest(msg)
		}

	case // digital twin property messages
		wssclient.MsgTypeReadAllProperties,
		wssclient.MsgTypeReadMultipleProperties,
		wssclient.MsgTypeReadProperty,
		wssclient.MsgTypeWriteMultipleProperties,
		wssclient.MsgTypeWriteProperty,
		wssclient.MsgTypePropertyReadings, // agent publishes properties update
		wssclient.MsgTypePropertyReading:  // agent publishes property update
		// map the message to a ThingMessage
		wssMsg := wssclient.PropertyMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		// FIXME: readmultiple has an array of names
		msg = transports.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, c.clientID)
		msg.RequestID = wssMsg.RequestID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		if wssMsg.MessageType == wssclient.MsgTypePropertyReading ||
			wssMsg.MessageType == wssclient.MsgTypePropertyReadings {
			c.ForwardAsNotification(msg)
		} else {
			c.ForwardAsRequest(msg)
		}
		// td messages
	case wssclient.MsgTypeReadTD:
		wssMsg := wssclient.TDMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		msg = transports.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, c.clientID)
		msg.RequestID = wssMsg.RequestID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		c.ForwardAsRequest(msg)

	case wssclient.MsgTypeUpdateTD:
		wssMsg := wssclient.TDMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		msg = transports.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, c.clientID)
		msg.RequestID = wssMsg.RequestID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		c.ForwardAsNotification(msg)

	// subscriptions are handled inside this binding
	case wssclient.MsgTypeObserveAllProperties:
		wssMsg := wssclient.PropertyMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		c.HandleObserveAllProperties(&wssMsg)
	//case wssbinding.MsgTypeObserveMultipleProperties:
	//	wssMsg := wssbinding.PropertyMessage{}
	//	err = c.UnmarshalFromString(jsonMsg, &wssMsg)
	//	c.HandleObserveMultipleProperties(&wssMsg)
	case wssclient.MsgTypeObserveProperty:
		wssMsg := wssclient.PropertyMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		c.HandleObserveProperty(&wssMsg)
	case wssclient.MsgTypeSubscribeAllEvents:
		wssMsg := wssclient.EventMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		c.HandleSubscribeAllEvents(&wssMsg)
	case wssclient.MsgTypeSubscribeEvent:
		wssMsg := wssclient.EventMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		c.HandleSubscribeEvent(&wssMsg)
	case wssclient.MsgTypeUnobserveAllProperties:
		wssMsg := wssclient.PropertyMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		c.HandleUnobserveAllProperties(&wssMsg)
	case wssclient.MsgTypeUnobserveProperty:
		wssMsg := wssclient.PropertyMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		c.HandleUnobserveProperty(&wssMsg)
	case wssclient.MsgTypeUnsubscribeAllEvents:
		wssMsg := wssclient.EventMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		c.HandleUnsubscribeAllEvents(&wssMsg)
	case wssclient.MsgTypeUnsubscribeEvent:
		wssMsg := wssclient.EventMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		c.HandleUnsubscribeEvent(&wssMsg)

	// other messages handled inside this binding
	case wssclient.MsgTypePing:
		wssMsg := wssclient.BaseMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		c.HandlePing(&wssMsg)

	default:
		slog.Warn("_receive: unknown operation",
			"messageType", baseMsg.MessageType)
	}
}
