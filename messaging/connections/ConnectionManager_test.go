package connections_test

import (
	"errors"
	"testing"
	"time"

	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/connections"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddRemoveConnection(t *testing.T) {
	const clientID = "client1"
	const remoteAddr = "remote1"
	const session1ID = "sess1"
	const session2ID = "sess2"

	cm := connections.NewConnectionManager()
	c1 := NewDummyConnection(clientID, remoteAddr, session1ID)
	err := cm.AddConnection(c1)
	require.NoError(t, err)

	//
	c2 := NewDummyConnection(clientID, remoteAddr, session2ID)
	err = cm.AddConnection(c2)
	require.NoError(t, err)

	// connection must exist
	cid1 := c1.cinfo.ConnectionID
	c1b := cm.GetConnectionByConnectionID(clientID, cid1)
	require.NotEmpty(t, c1b)

	// remove the connection
	cm.RemoveConnection(c1b)
	require.NoError(t, err)

	// connection no longer exists
	c1c := cm.GetConnectionByConnectionID(clientID, cid1)
	require.Empty(t, c1c)

	// c2 should remain
	cid2 := c2.cinfo.ConnectionID
	c2a := cm.GetConnectionByConnectionID(clientID, cid2)
	require.NotEmpty(t, c2a)

	// again but this time closing connection 2
	c2b := cm.GetConnectionByClientID(clientID)
	cm.RemoveConnection(c2b)
	//isClosed := c2b.IsClosed()
	//assert.True(t,isClosed)
	c2b = cm.GetConnectionByConnectionID(clientID, cid2)
	require.Empty(t, c2b)

	// close all
	cm.CloseAll()
}

func TestCloseClientConnection(t *testing.T) {
	const client1ID = "client1"
	const client2ID = "client2"
	const remoteAddr = "remote1"
	const session1ID = "sess1"
	const session2ID = "sess2"

	cm := connections.NewConnectionManager()
	c1 := NewDummyConnection(client1ID, remoteAddr, session1ID)
	err := cm.AddConnection(c1)
	require.NoError(t, err)

	c2 := NewDummyConnection(client2ID, remoteAddr, session2ID)
	err = cm.AddConnection(c2)
	require.NoError(t, err)

	// connection must exist
	c1a := cm.GetConnectionByConnectionID(client1ID, c1.cinfo.ConnectionID)
	require.NotNil(t, c1a)

	// close the connection of user1
	cm.CloseAllClientConnections(client1ID)

	// connection no longer exists
	c1b := cm.GetConnectionByConnectionID(client1ID, c1.cinfo.ConnectionID)
	require.Empty(t, c1b)

	// connection user 2 must still exist
	c2a := cm.GetConnectionByConnectionID(client2ID, c2.cinfo.ConnectionID)
	require.NotEmpty(t, c2a)

	// close all
	cm.CloseAll()

	c2b := cm.GetConnectionByConnectionID(client2ID, c2.cinfo.ConnectionID)
	require.Empty(t, c2b)
}

func TestForEachConnection(t *testing.T) {
	const client1ID = "client1"
	const client2ID = "client2"
	const remoteAddr = "remote1"
	const session1ID = "sess1"
	const session2ID = "sess2"

	cm := connections.NewConnectionManager()
	c1 := NewDummyConnection(client1ID, remoteAddr, session1ID)
	err := cm.AddConnection(c1)
	require.NoError(t, err)

	c2 := NewDummyConnection(client2ID, remoteAddr, session2ID)
	err = cm.AddConnection(c2)
	require.NoError(t, err)

	count := 0
	cm.ForEachConnection(func(c messaging.IServerConnection) {
		count++
	})
	assert.Equal(t, 2, count)
}

func TestConnectionTwice(t *testing.T) {
	const client1ID = "client1"
	const client2ID = "client2"
	const remoteAddr = "remote1"
	const session1ID = "sess1"
	const session2ID = "sess2"

	cm := connections.NewConnectionManager()
	c1 := NewDummyConnection(client1ID, remoteAddr, session1ID)
	err := cm.AddConnection(c1)
	require.NoError(t, err)
	// The first connection with the same cid should be disconnected.
	c2 := NewDummyConnection(client1ID, remoteAddr, session1ID)
	err = cm.AddConnection(c2)
	time.Sleep(time.Millisecond * 100)
	require.NoError(t, err)
	assert.False(t, c1.IsConnected())
	assert.True(t, c2.IsConnected())

	// One connection remains
	c1a := cm.GetConnectionByConnectionID(client1ID, c1.cinfo.ConnectionID)
	require.NotNil(t, c1a)
}

func TestPublishEventProp(t *testing.T) {
	const client1ID = "client1"
	const remoteAddr = "remote1"
	const session1ID = "sess1"
	const agent1ID = "agent1"
	const thing1ID = "thing1"
	var dThingID = td.MakeDigiTwinThingID(agent1ID, thing1ID)
	const evName = "event1"
	const propName = "prop1"
	var evCount = 0
	var propCount = 0

	cm := connections.NewConnectionManager()
	c1 := NewDummyConnection(client1ID, remoteAddr, session1ID)
	c1.SubscribeEvent(dThingID, "")
	c1.ObserveProperty(dThingID, "")
	c1.SendNotificationHandler = func(notif *messaging.NotificationMessage) {
		if notif.Operation == wot.OpSubscribeEvent {
			evCount++
		} else if notif.Operation == wot.OpObserveProperty {
			propCount++
		}
	}
	c1.SendRequestHandler = func(req *messaging.RequestMessage, replyTo messaging.IConnection) *messaging.ResponseMessage {
		assert.Fail(t, "unexpected")
		return req.CreateResponse(nil, errors.New("unexpected request"))
	}

	err := cm.AddConnection(c1)
	require.NoError(t, err)

	// publish
	notif1 := messaging.NewNotificationMessage(
		wot.OpSubscribeEvent, dThingID, evName, nil)
	err = cm.SendNotification(notif1)
	assert.NoError(t, err)
	notif2 := messaging.NewNotificationMessage(
		wot.OpObserveProperty, dThingID, propName, nil)
	err = cm.SendNotification(notif2)
	assert.NoError(t, err)

	time.Sleep(time.Millisecond * 10)
	// should receive 1 event and 1 property message
	assert.Equal(t, 1, evCount)
	assert.Equal(t, 1, propCount)
}
