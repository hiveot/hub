package wss

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/wssclient"
	"github.com/hiveot/hub/lib/utils"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"log/slog"
	"time"
)

// Handle incoming requests

// HubWssReceive handles an incoming websocket message
// This separates it in messages and requests and passes it to the message handler
// for further processing.
func (c *HubWssConnection) HubWssReceive(jsonMsg string) {
	var msg *hubclient.ThingMessage
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
	slog.Info("HubWssReceive: Received message",
		"clientID", c.clientID,
		"messageType", messageType,
		"correlationID", msgAsMap["correlationId"])

	switch messageType {

	case // action messages
		wssclient.MsgTypeInvokeAction,
		wssclient.MsgTypeQueryAction,
		wssclient.MsgTypeQueryAllActions:
		// map the message to a ThingMessage
		wssMsg := wssclient.ActionMessage{}
		_ = utils.DecodeAsObject(msgAsMap, &wssMsg)
		op, _ := wssclient.MsgTypeToOp[wssMsg.MessageType]
		msg = hubclient.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, c.clientID)
		msg.CorrelationID = wssMsg.CorrelationID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		c.ForwardAsRequest(msg)

	case // event action messages
		wssclient.MsgTypeReadAllEvents,
		//wssclient.MsgTypeReadMultipleEvents,
		wssclient.MsgTypePublishEvent,
		wssclient.MsgTypeReadEvent:
		// map the message to a ThingMessage
		wssMsg := wssclient.EventMessage{}
		_ = utils.DecodeAsObject(msgAsMap, &wssMsg)
		op, _ := wssclient.MsgTypeToOp[wssMsg.MessageType]
		msg = hubclient.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, c.clientID)
		msg.CorrelationID = wssMsg.CorrelationID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		if messageType == wssclient.MsgTypePublishEvent {
			c.ForwardAsEvent(msg)
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
		_ = utils.DecodeAsObject(msgAsMap, &wssMsg)
		op, _ := wssclient.MsgTypeToOp[wssMsg.MessageType]
		// FIXME: readmultiple has an array of names
		msg = hubclient.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, c.clientID)
		msg.CorrelationID = wssMsg.CorrelationID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		if wssMsg.MessageType == wssclient.MsgTypePropertyReading ||
			wssMsg.MessageType == wssclient.MsgTypePropertyReadings {
			c.ForwardAsEvent(msg)
		} else {
			c.ForwardAsRequest(msg)
		}
		// td messages
	case wssclient.MsgTypeReadTD:
		wssMsg := wssclient.TDMessage{}
		_ = utils.DecodeAsObject(msgAsMap, &wssMsg)
		op, _ := wssclient.MsgTypeToOp[wssMsg.MessageType]
		msg = hubclient.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, c.clientID)
		msg.CorrelationID = wssMsg.CorrelationID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		c.ForwardAsRequest(msg)

	case wssclient.MsgTypeUpdateTD:
		wssMsg := wssclient.TDMessage{}
		_ = utils.DecodeAsObject(msgAsMap, &wssMsg)
		op, _ := wssclient.MsgTypeToOp[wssMsg.MessageType]
		msg = hubclient.NewThingMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, c.clientID)
		msg.CorrelationID = wssMsg.CorrelationID
		msg.MessageID = wssMsg.MessageID
		msg.Timestamp = wssMsg.Timestamp
		c.ForwardAsEvent(msg)

	// subscriptions are handled inside this binding
	case wssclient.MsgTypeSubscribeAllEvents:
		wssMsg := wssclient.EventMessage{}
		err = jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
		c.HandleSubscribeAllEvents(&wssMsg)
	case wssclient.MsgTypeSubscribeEvent:
		wssMsg := wssclient.EventMessage{}
		err = jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
		c.HandleSubscribeEvent(&wssMsg)
	case wssclient.MsgTypeUnobserveAllProperties:
		wssMsg := wssclient.PropertyMessage{}
		err = jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
		c.HandleUnobserveAllProperties(&wssMsg)
	case wssclient.MsgTypeUnobserveProperty:
		wssMsg := wssclient.PropertyMessage{}
		err = jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
		c.HandleUnobserveProperty(&wssMsg)
	case wssclient.MsgTypeUnsubscribeAllEvents:
		wssMsg := wssclient.EventMessage{}
		err = jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
		c.HandleUnsubscribeAllEvents(&wssMsg)
	case wssclient.MsgTypeUnsubscribeEvent:
		wssMsg := wssclient.EventMessage{}
		err = jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
		c.HandleUnsubscribeEvent(&wssMsg)

	// other messages handled inside this binding
	case wssclient.MsgTypeRefresh:
		wssMsg := wssclient.ActionMessage{}
		err = utils.DecodeAsObject(msgAsMap, &wssMsg)
		c.HandleRefresh(&wssMsg)
	case wssclient.MsgTypePing:
		wssMsg := wssclient.BaseMessage{}
		_ = jsoniter.UnmarshalFromString(jsonMsg, &wssMsg)
		c.HandlePing(&wssMsg)

	default:
		slog.Warn("_receive: unknown operation", "messageType", messageType)
	}
}

// event style messages are pushed to the digitwin router
func (c *HubWssConnection) ForwardAsEvent(msg *hubclient.ThingMessage) {
	c.dtwRouter.HandleMessage(msg)
}

// request style messages are pushed to the digitwin router
func (c *HubWssConnection) ForwardAsRequest(msg *hubclient.ThingMessage) {
	stat := c.dtwRouter.HandleRequest(msg, c.connectionID)
	// FIXME: map status between protocols
	wssStatus := stat.Status
	reply := wssclient.ActionStatusMessage{
		MessageType:   wssclient.MsgTypeActionStatus,
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
	_, _ = c._send(reply, msg.CorrelationID)
}

// func (c *HubWssConnection) HandleLogin(msg *hubclient.ThingMessage) {
// }
func (c *HubWssConnection) HandleRefresh(wssMsg *wssclient.ActionMessage) {
	// convert to a hub request
	msg := hubclient.NewThingMessage(
		vocab.HTOpRefresh, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, c.clientID)
	msg.CorrelationID = wssMsg.CorrelationID
	msg.MessageID = wssMsg.MessageID
	msg.Timestamp = wssMsg.Timestamp
	c.ForwardAsRequest(msg)
}
func (c *HubWssConnection) HandleObserveAllProperties(wssMsg *wssclient.PropertyMessage) {
}

func (c *HubWssConnection) HandleObserveProperty(wssMsg *wssclient.PropertyMessage) {
}

func (c *HubWssConnection) HandlePing(wssMsg *wssclient.BaseMessage) {
}

func (c *HubWssConnection) HandleSubscribeAllEvents(wssMsg *wssclient.EventMessage) {
}

func (c *HubWssConnection) HandleSubscribeEvent(wssMsg *wssclient.EventMessage) {
}
func (c *HubWssConnection) HandleUnobserveAllProperties(wssMsg *wssclient.PropertyMessage) {
}

func (c *HubWssConnection) HandleUnobserveProperty(wssMsg *wssclient.PropertyMessage) {
}
func (c *HubWssConnection) HandleUnsubscribeAllEvents(wssMsg *wssclient.EventMessage) {
}

func (c *HubWssConnection) HandleUnsubscribeEvent(wssMsg *wssclient.EventMessage) {
}
