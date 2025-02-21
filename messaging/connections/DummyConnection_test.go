package connections_test

import (
	"fmt"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/connections"
)

// Dummy connection for testing connection manager
// this implements the IServerConnection interface.
type DummyConnection struct {
	cinfo         messaging.ConnectionInfo
	remoteAddr    string
	observations  connections.Subscriptions
	subscriptions connections.Subscriptions

	SendRequestHandler  messaging.RequestHandler
	SendResponseHandler messaging.ResponseHandler
}

func (c *DummyConnection) Disconnect() {}

func (c *DummyConnection) GetConnectionInfo() messaging.ConnectionInfo { return c.cinfo }
func (c *DummyConnection) IsConnected() bool                           { return true }

//func (c *DummyConnection) GetSessionID() string    { return c.sessID }

//func (c *DummyConnection) InvokeAction(thingID string, name string, input any, correlationID string, senderID string) (
//	status string, output any, err error) {
//	return transports2.RequestCompleted, nil, nil
//}

func (c *DummyConnection) SendNotification(msg messaging.ResponseMessage) {
	_ = c.SendResponse(&msg)
}

func (c *DummyConnection) SendRequest(msg *messaging.RequestMessage) error {
	if c.SendRequestHandler != nil && c.observations.IsSubscribed(msg.ThingID, msg.Name) {
		c.SendRequestHandler(msg, c)
	}
	return fmt.Errorf("no request sender set")
}

func (c *DummyConnection) SendResponse(resp *messaging.ResponseMessage) error {
	if c.SendResponseHandler != nil {
		c.SendResponseHandler(resp)
	}
	return nil
}

func (c *DummyConnection) SetConnectHandler(h messaging.ConnectionHandler) {
}

// SetRequestHandler is ignored as this is an outgoing 1-way connection
func (c *DummyConnection) SetRequestHandler(h messaging.RequestHandler) {
}

// SetResponseHandler is ignored as this is an outgoing 1-way connection
func (c *DummyConnection) SetResponseHandler(h messaging.ResponseHandler) {
}

func (c *DummyConnection) SubscribeEvent(dThingID, name string) {
	c.subscriptions.Subscribe(dThingID, name, "subscr-1")
}
func (c *DummyConnection) ObserveProperty(dThingID, name string) {
	c.observations.Subscribe(dThingID, name, "observe-1")
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
	return &DummyConnection{
		cinfo: messaging.ConnectionInfo{
			ConnectionID: cid,
			ClientID:     clientID,
		},
		remoteAddr: remoteAddr,
	}
}
