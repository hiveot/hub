package wssserver

import (
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
)

// Handle incoming messages from clients
// These are converted into a standard ThingMessage envelope and passed to
// the handler.

// ForwardAsNotification message is notification for one or multiple clients,
// depending on the operation.
func (c *WssServerConnection) ForwardAsNotification(msg *transports.ThingMessage) {
	// nothing to return
	_, _, _ = c.messageHandler(msg, "")
}

// ForwardAsRequest message is a request style messages to be sent to a destination
// This returns a response message to the sender with the given requestID.
func (c *WssServerConnection) ForwardAsRequest(msg *transports.ThingMessage) {
	// the message handler does the processing
	completed, output, err := c.messageHandler(msg, c.GetConnectionID())

	// if completed, then send the result to the client.
	// otherwise the response will come asynchronously
	if completed || err != nil {
		err2 := c.SendResponse(msg.ThingID, msg.Name, output, err, msg.RequestID)
		if err2 != nil {
			slog.Error("ForwardAsRequest. Failed sending response to client",
				"operation", msg.Operation,
				"thingID", msg.ThingID,
				"name", msg.Name,
				"clientID", c.GetClientID(),
				"connectionID", c.GetConnectionID(),
				"requestID", msg.RequestID,
				"err", err.Error(),
			)
		}
	}
}

// HandleError forwards an error message to the sender
func (c *WssServerConnection) HandleError(wssMsg *ErrorMessage) {
	payload := wssMsg.Title + "\n" + wssMsg.Detail
	msg := transports.NewThingMessage(
		wot.HTOpPublishError, wssMsg.ThingID, wssMsg.Name, payload, c.clientID)
	msg.RequestID = wssMsg.RequestID
	msg.Timestamp = wssMsg.Timestamp
	c.ForwardAsNotification(msg)
}

func (c *WssServerConnection) HandleObserveAllProperties(wssMsg *PropertyMessage) {
	c.observations.SubscribeAll(wssMsg.ThingID)
}

func (c *WssServerConnection) HandleObserveProperty(wssMsg *PropertyMessage) {
	c.observations.Subscribe(wssMsg.ThingID, wssMsg.Name)
}

// HandlePing replies with pong to a ping message
func (c *WssServerConnection) HandlePing(wssMsg *BaseMessage) {
	pongMessage := *wssMsg
	pongMessage.MessageType = MsgTypePong
	c._send(pongMessage)
}

func (c *WssServerConnection) HandleSubscribeAllEvents(wssMsg *EventMessage) {
	c.subscriptions.SubscribeAll(wssMsg.ThingID)
}

func (c *WssServerConnection) HandleSubscribeEvent(wssMsg *EventMessage) {
	c.subscriptions.Subscribe(wssMsg.ThingID, wssMsg.Name)
}
func (c *WssServerConnection) HandleUnobserveAllProperties(wssMsg *PropertyMessage) {
	c.observations.UnsubscribeAll(wssMsg.ThingID)
}

func (c *WssServerConnection) HandleUnobserveProperty(wssMsg *PropertyMessage) {
	c.observations.Unsubscribe(wssMsg.ThingID, wssMsg.Name)
}

func (c *WssServerConnection) HandleUnsubscribeAllEvents(wssMsg *EventMessage) {
	c.subscriptions.UnsubscribeAll(wssMsg.ThingID)
}

func (c *WssServerConnection) HandleUnsubscribeEvent(wssMsg *EventMessage) {
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
	baseMsg := BaseMessage{}

	// the operation is needed to determine whether this is a request or send and forget message
	// unfortunately this needs double unmarshalling :(
	err := c.Unmarshal(raw, &baseMsg)
	if err != nil {
		slog.Warn("_receive: unmarshalling message failed. Message ignored.",
			"clientID", c.clientID,
			"err", err.Error())
		return
	}
	op, _ := MsgTypeToOp[baseMsg.MessageType]
	slog.Info("WssServerHandleMessage: Received message",
		slog.String("clientID", c.clientID),
		slog.String("messageType", baseMsg.MessageType),
		slog.String("requestID", baseMsg.RequestID))

	switch baseMsg.MessageType {

	case MsgTypeActionStatus:
		// hub receives an action result from an agent
		// this will be forwarded to the consumer as a message
		wssMsg := ActionStatusMessage{}
		_ = c.Unmarshal(raw, &wssMsg)

		msg = transports.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Output, c.clientID)
		msg.RequestID = wssMsg.RequestID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		c.ForwardAsNotification(msg)

	case // hub receives action messages from a consumer
		MsgTypeInvokeAction,
		MsgTypeQueryAction,
		MsgTypeQueryAllActions:
		// map the message to a ThingMessage
		wssMsg := ActionMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		msg = transports.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, c.clientID)
		msg.RequestID = wssMsg.RequestID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		c.ForwardAsRequest(msg)

	case // hub receives event action messages
		MsgTypeReadAllEvents,
		//wssbinding.MsgTypeReadMultipleEvents,
		MsgTypePublishEvent,
		MsgTypeReadEvent:
		// map the message to a ThingMessage
		wssMsg := EventMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		msg = transports.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, c.clientID)
		msg.RequestID = wssMsg.RequestID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		if baseMsg.MessageType == MsgTypePublishEvent {
			c.ForwardAsNotification(msg)
		} else {
			c.ForwardAsRequest(msg)
		}

	case // digital twin property messages
		MsgTypeReadAllProperties,
		MsgTypeReadMultipleProperties,
		MsgTypeReadProperty,
		MsgTypeWriteMultipleProperties,
		MsgTypeWriteProperty,
		MsgTypePropertyReadings, // agent publishes properties update
		MsgTypePropertyReading:  // agent publishes property update
		// map the message to a ThingMessage
		wssMsg := PropertyMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		// FIXME: readmultiple has an array of names
		msg = transports.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, c.clientID)
		msg.RequestID = wssMsg.RequestID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		if wssMsg.MessageType == MsgTypePropertyReading ||
			wssMsg.MessageType == MsgTypePropertyReadings {
			c.ForwardAsNotification(msg)
		} else {
			c.ForwardAsRequest(msg)
		}
		// td messages
	case MsgTypeReadTD:
		wssMsg := TDMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		msg = transports.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, c.clientID)
		msg.RequestID = wssMsg.RequestID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		c.ForwardAsRequest(msg)

	case MsgTypeUpdateTD:
		wssMsg := TDMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		msg = transports.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, c.clientID)
		msg.RequestID = wssMsg.RequestID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		c.ForwardAsNotification(msg)

	// subscriptions are handled inside this binding
	case MsgTypeObserveAllProperties:
		wssMsg := PropertyMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		c.HandleObserveAllProperties(&wssMsg)
	//case wssbinding.MsgTypeObserveMultipleProperties:
	//	wssMsg := wssbinding.PropertyMessage{}
	//	err = c.UnmarshalFromString(jsonMsg, &wssMsg)
	//	c.HandleObserveMultipleProperties(&wssMsg)
	case MsgTypeObserveProperty:
		wssMsg := PropertyMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		c.HandleObserveProperty(&wssMsg)
	case MsgTypeSubscribeAllEvents:
		wssMsg := EventMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		c.HandleSubscribeAllEvents(&wssMsg)
	case MsgTypeSubscribeEvent:
		wssMsg := EventMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		c.HandleSubscribeEvent(&wssMsg)
	case MsgTypeUnobserveAllProperties:
		wssMsg := PropertyMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		c.HandleUnobserveAllProperties(&wssMsg)
	case MsgTypeUnobserveProperty:
		wssMsg := PropertyMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		c.HandleUnobserveProperty(&wssMsg)
	case MsgTypeUnsubscribeAllEvents:
		wssMsg := EventMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		c.HandleUnsubscribeAllEvents(&wssMsg)
	case MsgTypeUnsubscribeEvent:
		wssMsg := EventMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		c.HandleUnsubscribeEvent(&wssMsg)

	// other messages handled inside this binding
	case MsgTypeError:
		wssMsg := ErrorMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		c.HandleError(&wssMsg)
	case MsgTypePing:
		wssMsg := BaseMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		c.HandlePing(&wssMsg)

	default:
		slog.Warn("_receive: unknown operation",
			"messageType", baseMsg.MessageType)
	}
}
