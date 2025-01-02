package connections_test

import (
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/connections"
)

// Dummy connection for testing connection manager
// this implements the IServerConnection interface.
type DummyConnection struct {
	connectionID  string
	clientID      string
	remoteAddr    string
	observations  connections.Subscriptions
	subscriptions connections.Subscriptions

	SendNotificationHandler transports.ServerNotificationHandler
	SendRequestHandler      transports.ServerRequestHandler
	SendResponseHandler     transports.ServerResponseHandler
}

func (c *DummyConnection) Disconnect() {}

func (c *DummyConnection) GetConnectionID() string { return c.connectionID }
func (c *DummyConnection) GetClientID() string     { return c.clientID }

func (c *DummyConnection) GetProtocolType() string { return "dummy" }

//func (c *DummyConnection) GetSessionID() string    { return c.sessID }

//func (c *DummyConnection) InvokeAction(thingID string, name string, input any, correlationID string, senderID string) (
//	status string, output any, err error) {
//	return transports2.RequestCompleted, nil, nil
//}

func (c *DummyConnection) PublishActionStatus(stat digitwin.ActionStatus, agentID string) error {
	return nil
}

func (c *DummyConnection) SendNotification(notif transports.NotificationMessage) {
	if c.SendNotificationHandler != nil && c.subscriptions.IsSubscribed(notif.ThingID, notif.Name) {
		c.SendNotificationHandler(notif)
	}
}
func (c *DummyConnection) SendRequest(msg transports.RequestMessage) error {
	if c.SendRequestHandler != nil && c.observations.IsSubscribed(msg.ThingID, msg.Name) {
		c.SendRequestHandler(msg, c.GetConnectionID())
	}
	return fmt.Errorf("no request sender set")
}

func (c *DummyConnection) SendResponse(resp transports.ResponseMessage) error {
	if c.SendResponseHandler != nil {
		c.SendResponseHandler(resp)
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

//func (c *DummyConnection) WriteProperty(thingID, name string, value any, correlationID string, senderID string) (status string, err error) {
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
