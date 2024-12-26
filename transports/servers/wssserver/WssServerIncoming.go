package wssserver

import (
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"time"
)

// Handle incoming messages from clients
// These are converted into a standard ThingMessage envelope and passed to
// the handler.

func (c *WssServerConnection) HandleObserveAllProperties(wssMsg *PropertyMessage) {
	c.observations.SubscribeAll(wssMsg.ThingID)
}

func (c *WssServerConnection) HandleObserveProperty(wssMsg *PropertyMessage) {
	c.observations.Subscribe(wssMsg.ThingID, wssMsg.Name)
}

// HandlePing replies with pong to a ping message
func (c *WssServerConnection) HandlePing(wssMsg *BaseMessage) {
	// return an action response
	pongMessage := ActionStatusMessage{
		MessageType: MsgTypePong,
		Output:      "pong",
		Timestamp:   time.Now().Format(wot.RFC3339Milli),
		RequestID:   wssMsg.RequestID,
	}
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
// This can be a consumer request, agent notifications or an agent response
func (c *WssServerConnection) WssServerHandleMessage(raw []byte) {
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
		slog.String("messageType", baseMsg.MessageType),
		slog.String("senderID", c.clientID),
		slog.String("requestID", baseMsg.RequestID))

	switch baseMsg.MessageType {

	case MsgTypeActionStatus:
		// hub receives an action status from an agent.
		// this will be forwarded to the consumer as a response
		wssMsg := ActionStatusMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		resp := transports.NewResponseMessage(wot.OpInvokeAction,
			wssMsg.ThingID, wssMsg.Name, wssMsg.Output, nil, wssMsg.RequestID)
		resp.Error = wssMsg.Error
		resp.Status = wssMsg.Status // todo: convert from wss to global names
		resp.Updated = wssMsg.TimeEnded
		_ = c.responseHandler(c.clientID, resp)

	case // hub receives action messages from a consumer. Forward as a request.
		MsgTypeInvokeAction,
		MsgTypeQueryAction,
		MsgTypeQueryAllActions:
		// map the message to a RequestMessage
		wssMsg := ActionMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		req := transports.NewRequestMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, wssMsg.RequestID)
		req.SenderID = c.GetClientID()
		req.Created = wssMsg.Timestamp
		_ = c.requestHandler(req, c.GetConnectionID())

	case // hub receives event request. Forward as request.
		MsgTypeReadAllEvents,
		MsgTypeReadEvent:
		// map the message to a request
		wssMsg := EventMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		req := transports.NewRequestMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, wssMsg.RequestID)
		req.SenderID = c.GetClientID()
		req.Created = wssMsg.Timestamp
		_ = c.requestHandler(req, c.GetConnectionID())

	case // hub receives event notification. Forward as notification.
		MsgTypePublishEvent:
		// map the message to a request
		wssMsg := EventMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		notif := transports.NewNotificationMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data)
		notif.Created = wssMsg.Timestamp
		c.notificationHandler(c.clientID, notif)

	case // property requests. Forward as requests
		MsgTypeReadAllProperties,
		MsgTypeReadMultipleProperties,
		MsgTypeReadProperty,
		MsgTypeWriteMultipleProperties,
		MsgTypeWriteProperty:
		// map the message to a ThingMessage
		wssMsg := PropertyMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		// FIXME: readmultiple has an array of names
		req := transports.NewRequestMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, wssMsg.RequestID)
		req.SenderID = c.GetClientID()
		req.Created = wssMsg.Timestamp
		_ = c.requestHandler(req, c.GetConnectionID())

	case // agent response
		MsgTypePropertyReadings, // agent response
		MsgTypePropertyReading:  // agent response
		// map the message to a ThingMessage
		wssMsg := PropertyMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		// FIXME: readmultiple has an array of names
		resp := transports.NewResponseMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, nil, wssMsg.RequestID)
		resp.Updated = wssMsg.Timestamp
		_ = c.responseHandler(c.clientID, resp)

		// td messages
	case MsgTypeReadTD:
		wssMsg := TDMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		req := transports.NewRequestMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, wssMsg.RequestID)
		req.SenderID = c.GetClientID()
		req.Created = wssMsg.Timestamp
		_ = c.requestHandler(req, c.GetConnectionID())

	case MsgTypeUpdateTD:
		wssMsg := TDMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		notif := transports.NewNotificationMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data)
		notif.Created = wssMsg.Timestamp
		c.notificationHandler(c.clientID, notif)

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
		// agent returned an error
		wssMsg := ErrorMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		slog.Info("WSS Agent returned an error:",
			"senderID", c.clientID,
			"ThingID", wssMsg.ThingID,
			"error", wssMsg.Title)
		resp := transports.NewResponseMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Detail, nil, wssMsg.RequestID)
		resp.Updated = wssMsg.Timestamp
		resp.Error = wssMsg.Title
		_ = c.responseHandler(c.clientID, resp)

	case MsgTypePing:
		wssMsg := BaseMessage{}
		_ = c.Unmarshal(raw, &wssMsg)
		c.HandlePing(&wssMsg)

	default:
		// FIXME: a no-operation with requestID is a response
		slog.Warn("_receive: unknown operation",
			"messageType", baseMsg.MessageType)
	}
}
