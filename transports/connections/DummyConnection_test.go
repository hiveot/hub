package connections_test

import (
	"fmt"
	transports2 "github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/connections"
	"github.com/hiveot/hub/wot"
)

// Dummy connection for testing connection manager
// this implements the IServerConnection interface.
type DummyConnection struct {
	connectionID  string
	clientID      string
	remoteAddr    string
	observations  connections.Subscriptions
	subscriptions connections.Subscriptions

	SendNotificationHandler transports2.ServerNotificationHandler
	SendRequestHandler      transports2.ServerRequestHandler
}

func (c *DummyConnection) Disconnect() {}

func (c *DummyConnection) GetConnectionID() string { return c.connectionID }
func (c *DummyConnection) GetClientID() string     { return c.clientID }

func (c *DummyConnection) GetProtocolType() string { return "dummy" }

//func (c *DummyConnection) GetSessionID() string    { return c.sessID }

//func (c *DummyConnection) InvokeAction(thingID string, name string, input any, requestID string, senderID string) (
//	status string, output any, err error) {
//	return transports2.RequestCompleted, nil, nil
//}

func (c *DummyConnection) PublishActionStatus(stat transports2.ActionStatus, agentID string) error {
	return nil
}

func (c *DummyConnection) SendError(dThingID, name string, errResponse string, requestID string) {
	if c.SendNotificationHandler != nil {
		c.SendNotificationHandler(wot.HTOpPublishError, dThingID, name, errResponse, "")
	}
}
func (c *DummyConnection) SendNotification(notif transports2.NotificationMessage) {
	if c.SendNotificationHandler != nil && c.subscriptions.IsSubscribed(notif.ThingID, notif.Name) {
		c.SendNotificationHandler(notif)
	}
}
func (c *DummyConnection) SendRequest(msg transports2.RequestMessage) error {
	if c.SendRequestHandler != nil && c.observations.IsSubscribed(msg.ThingID, msg.Name) {
		return c.SendRequestHandler(msg.Operation, msg.ThingID, msg.Name, msg.Data, msg.RequestID)
	}
	return fmt.Errorf("no request sender set")
}

func (c *DummyConnection) SendResponse(dThingID, name string, data any, err error, requestID string) error {
	if err != nil {
		c.SendError(dThingID, name, err.Error(), requestID)
	} else {
		if c.SendNotificationHandler != nil {
			c.SendNotificationHandler(wot.HTOpUpdateActionStatus, dThingID, name, data, requestID)
		}
	}
	return nil
}
func (c *DummyConnection) SubscribeEvent(dThingID, name string) {
	c.subscriptions.Subscribe(dThingID, name)
}
func (c *DummyConnection) ObserveProperty(dThingID, name string) {
	c.observations.Subscribe(dThingID, name)
}
func (c *DummyConnection) UnsubscribeEvent(dThingID, name string) {
	c.subscriptions.Unsubscribe(dThingID, name)
}
func (c *DummyConnection) UnobserveProperty(dThingID, name string) {
	c.observations.Unsubscribe(dThingID, name)
}

//func (c *DummyConnection) WriteProperty(thingID, name string, value any, requestID string, senderID string) (status string, err error) {
//	return "", nil
//}

func NewDummyConnection(clientID, remoteAddr, cid string) *DummyConnection {
	clcid := clientID + "." + remoteAddr + "." + cid
	return &DummyConnection{
		remoteAddr:   remoteAddr,
		connectionID: clcid,
		clientID:     clientID,
	}
}
