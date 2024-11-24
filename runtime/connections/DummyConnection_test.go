package connections_test

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/runtime/connections"
)

// Dummy connection for testing connection manager
type DummyConnection struct {
	connectionID  string
	clientID      string
	remoteAddr    string
	observations  connections.Subscriptions
	subscriptions connections.Subscriptions

	PublishEventHandler func(dThingID string, name string, value any, requestID string, agentID string)
	PublishPropHandler  func(dThingID string, name string, value any, requestID string, agentID string)
}

func (c *DummyConnection) Close() {}

func (c *DummyConnection) GetConnectionID() string { return c.connectionID }
func (c *DummyConnection) GetClientID() string     { return c.clientID }

//func (c *DummyConnection) GetSessionID() string    { return c.sessID }

func (c *DummyConnection) InvokeAction(thingID string, name string, input any, requestID string, senderID string) (
	status string, output any, err error) {
	return vocab.RequestCompleted, nil, nil
}

func (c *DummyConnection) PublishActionStatus(stat transports.RequestStatus, agentID string) error {
	return nil
}

func (c *DummyConnection) PublishEvent(dThingID string, name string, value any, requestID string, agentID string) {
	if c.PublishEventHandler != nil && c.subscriptions.IsSubscribed(dThingID, name) {
		c.PublishEventHandler(dThingID, name, value, requestID, agentID)
	}
}
func (c *DummyConnection) PublishProperty(dThingID string, name string, value any, requestID string, agentID string) {
	if c.PublishPropHandler != nil && c.observations.IsSubscribed(dThingID, name) {
		c.PublishPropHandler(dThingID, name, value, requestID, agentID)
	}
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
func (c *DummyConnection) WriteProperty(thingID, name string, value any, requestID string, senderID string) (status string, err error) {
	return "", nil
}

func NewDummyConnection(clientID, remoteAddr, cid string) *DummyConnection {
	clcid := clientID + "." + remoteAddr + "." + cid
	return &DummyConnection{
		remoteAddr:   remoteAddr,
		connectionID: clcid,
		clientID:     clientID,
	}
}
