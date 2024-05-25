package mqtttransport_test

import (
	"context"
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/lib/things"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"testing"
	"time"
)

const core = "mqtt"

func startTestServer() *testenv.TestServer {
	srv, err := testenv.StartTestServer(core, true)
	if err != nil {
		panic(err)
	}
	return srv
}

func TestReconnect(t *testing.T) {
	logging.SetLogging("info", "")
	const deviceID = "device1"
	rxChan := make(chan string, 1)
	connectedCh := make(chan hubclient.ConnectionStatus)
	msg := "hello world"
	var connectStatus hubclient.ConnectionStatus
	var rxMsg string

	//setup
	ts := startTestServer()

	// connect client and subscribe
	hc1, err := ts.AddConnectUser(deviceID, authapi.ClientTypeDevice, authapi.ClientRoleDevice)
	hc1.SetRetryConnect(true)
	require.NoError(t, err)
	defer hc1.Disconnect()
	err = hc1.SubEvents(deviceID, "", "")
	require.NoError(t, err)
	hc1.SetEventHandler(func(tv *things.ThingMessage) {
		slog.Info("received event")
		rxChan <- string(tv.Data)
	})
	hc1.SetConnectionHandler(func(status hubclient.HubTransportStatus) {
		t.Logf("onconnect callback: status=%s, info=%s", status.ConnectionStatus, status.LastError)
		if status.ConnectionStatus == hubclient.Connected {
			connectedCh <- status.ConnectionStatus
		}
	})

	t.Log("--- Disconnect server and reconnect")
	ts.MsgServer.Stop()
	err = ts.MsgServer.Start()
	require.NoError(t, err)
	err = ts.StartAuth(false)
	require.NoError(t, err)

	t.Log("waiting up to 30 seconds for reconnect")
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Second*30)
	select {
	case <-ctx.Done():
		t.Fail()
	case connectStatus = <-connectedCh:
		cancelFn()
	}
	//time.Sleep(time.Second * 30)
	currentStatus := hc1.GetStatus()
	require.Equal(t, hubclient.Connected, currentStatus.ConnectionStatus, "connection not reestablished")
	require.Equal(t, hubclient.Connected, connectStatus)

	// check if subscription is restored as well
	err = hc1.PubEvent("user1thing2", "user1event", []byte(msg))
	assert.NoError(t, err)
	ctx, cancelFn = context.WithTimeout(context.Background(), time.Second*3)
	//
	select {
	case <-ctx.Done():
		assert.Fail(t, "Timeout. no event received")
	case rxMsg = <-rxChan:
	}
	cancelFn()
	assert.Equal(t, msg, rxMsg)
	hc1.Disconnect()
	t.Log("--- done")
}
