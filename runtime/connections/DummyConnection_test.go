package connections_test

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/connections"
)

// Dummy connection for testing connection manager
type DummyConnection struct {
	connID        string
	sessID        string
	clientID      string
	remoteAddr    string
	subscriptions connections.Subscriptions

	PublishEventHandler func(dThingID string, name string, value any, messageID string, agentID string)
	PublishPropHandler  func(dThingID string, name string, value any, messageID string, agentID string)
}

func (c *DummyConnection) Close() {}

func (c *DummyConnection) GetConnectionID() string { return c.connID }
func (c *DummyConnection) GetClientID() string     { return c.clientID }
func (c *DummyConnection) GetSessionID() string    { return c.sessID }

func (c *DummyConnection) InvokeAction(thingID string, name string, input any, messageID string, senderID string) (
	status string, output any, err error) {
	return vocab.ProgressStatusCompleted, nil, nil
}

func (c *DummyConnection) PublishActionProgress(stat hubclient.ActionProgress, agentID string) error {
	return nil
}

func (c *DummyConnection) PublishEvent(dThingID string, name string, value any, messageID string, agentID string) {
	if c.PublishEventHandler != nil && c.subscriptions.IsSubscribed(dThingID, name) {
		c.PublishEventHandler(dThingID, name, value, messageID, agentID)
	}
}
func (c *DummyConnection) PublishProperty(dThingID string, name string, value any, messageID string, agentID string) {
	if c.PublishPropHandler != nil && c.subscriptions.IsSubscribed(dThingID, name) {
		c.PublishPropHandler(dThingID, name, value, messageID, agentID)
	}
}
func (c *DummyConnection) SubscribeEvent(dThingID, name string) {
	c.subscriptions.Subscribe(dThingID, name)
}
func (c *DummyConnection) ObserveProperty(dThingID, name string) {
	c.subscriptions.Observe(dThingID, name)
}
func (c *DummyConnection) UnsubscribeEvent(dThingID, name string) {
	c.subscriptions.Unsubscribe(dThingID, name)
}
func (c *DummyConnection) UnobserveProperty(dThingID, name string) {
	c.subscriptions.Unobserve(dThingID, name)
}
func (c *DummyConnection) WriteProperty(thingID, name string, value any, messageID string, senderID string) (status string, err error) {
	return "", nil
}

func NewDummyConnection(clientID, remoteAddr, sessionID string) *DummyConnection {
	connID := clientID + "." + remoteAddr + "." + sessionID
	return &DummyConnection{
		connID:     connID,
		remoteAddr: remoteAddr,
		sessID:     sessionID,
		clientID:   clientID,
	}
}
