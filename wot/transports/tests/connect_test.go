package tests

import (
	"context"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/clients"
	"github.com/hiveot/hub/wot/transports/connections"
	"github.com/hiveot/hub/wot/transports/servers/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

const testAgentID1 = "agent1"
const testAgentPassword1 = "agent1pass"
const testClientID1 = "client1"
const testClientPassword1 = "client1pass"
const testServerHttpPort = 9445
const testServerHttpURL = "https://localhost:9445"
const testServerSsescURL = "https://localhost:9445/ssesc"
const testServerWssURL = "wss://localhost:9445/wss"
const testServerMqttURL = "mqtts://localhost:9446"

// var defaultProtocol = transports.ProtocolTypeSSESC
var defaultProtocol = transports.ProtocolTypeWSS

var transportServer transports.ITransportServer
var authenticator *DummyAuthenticator
var certBundle = certs.CreateTestCertBundle()

// NewClient creates a new unconnected agent client with the given ID
// This panics if a client cannot be created
func NewAgentClient(clientID string) transports.IClientConnection {
	protocol := defaultProtocol
	fullURL := testServerHttpURL
	caCert := certBundle.CaCert
	bc, err := clients.CreateTransportClient(protocol, fullURL, clientID, caCert)
	if err != nil {
		panic("NewClient failed:" + err.Error())
	}
	// FIXME: align the interfaces for connection, consumer, agent
	return bc
}

// NewConsumerClient creates a new unconnected consumer client with the given ID
// This panics if a client cannot be created
// ClientID is only used for logging
func NewConsumerClient(clientID string) transports.IClientConnection {
	protocol := defaultProtocol
	fullURL := testServerHttpURL

	switch defaultProtocol {
	case transports.ProtocolTypeHTTP:
		fullURL = testServerHttpURL
	case transports.ProtocolTypeSSESC:
		fullURL = testServerSsescURL
	case transports.ProtocolTypeWSS:
		fullURL = testServerWssURL
	case transports.ProtocolTypeMQTT:
		fullURL = testServerMqttURL
	}
	caCert := certBundle.CaCert
	bc, err := clients.CreateTransportClient(protocol, fullURL, clientID, caCert)
	if err != nil {
		panic("NewClient failed:" + err.Error())
	}
	// FIXME: align the interfaces for connection, consumer, agent
	return bc
}

// Create a new form for the given operation
// This uses the default protocol binding server to generate the Form
func NewForm(op string) tdd.Form {
	form := transportServer.GetForm(op)
	return form
}

// start the default transport server
// THis panics if the servers cannot be created
func StartTransportServer(messageHandler transports.ServerMessageHandler) (
	cancelFunc func(), cm *connections.ConnectionManager) {

	caCert := certBundle.CaCert
	serverCert := certBundle.ServerCert
	cm = connections.NewConnectionManager()
	authenticator = NewDummyAuthenticator()
	authenticator.AddClient(testAgentID1, testAgentPassword1)
	authenticator.AddClient(testClientID1, testClientPassword1)

	switch defaultProtocol {
	case transports.ProtocolTypeHTTP, transports.ProtocolTypeSSESC, transports.ProtocolTypeWSS:
		// Start the HTTP binding with SSE-SC and WS sub-protocols
		config := httpserver.NewHttpBindingConfig()
		config.Port = testServerHttpPort
		var err error
		transportServer, err = httpserver.StartHttpTransportServer(
			&config, serverCert, caCert, authenticator, messageHandler, cm)
		if err != nil {
			panic("Unable to create protocol server: " + err.Error())
		}
	}
	return func() {
		transportServer.Stop()
	}, cm
}

func DummyMessageHandler(msg *transports.ThingMessage,
	replyTo transports.IServerConnection) (stat transports.RequestStatus) {
	slog.Info("DummyMessageHandler: Received message", "op", msg.Operation)
	return stat
}

// TestMain sets logging
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")

	result := m.Run()
	os.Exit(result)
}

// test create a server and connect a client
func TestStartStop(t *testing.T) {
	cancelFn, _ := StartTransportServer(DummyMessageHandler)
	defer cancelFn()
	cl1 := NewConsumerClient(testClientID1)
	defer cl1.Disconnect()

	token, err := cl1.ConnectWithPassword(testClientPassword1)
	require.NoError(t, err)
	require.NotEmpty(t, token)
}

func TestLoginRefresh(t *testing.T) {
	t.Log("TestLoginRefresh")
	cancelFn, _ := StartTransportServer(DummyMessageHandler)
	defer cancelFn()
	cl1 := NewConsumerClient(testClientID1)
	defer cl1.Disconnect()

	isConnected := cl1.IsConnected()
	assert.False(t, isConnected)

	token, err := cl1.ConnectWithPassword(testClientPassword1)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	isConnected = cl1.IsConnected()
	assert.True(t, isConnected)

	// refresh should succeed
	newToken, err := cl1.RefreshToken(token)
	require.NoError(t, err)
	require.NotEmpty(t, newToken)

	// end the session
	cl1.Disconnect()

	// should be able to reconnect with the new token
	// NOTE: the runtime session manager doesn't allow this as
	// the session no longer exists, but the authenticator doesn't care.
	token2, err := cl1.ConnectWithToken(newToken)
	require.NoError(t, err)
	assert.NotEmpty(t, token2)
	token3, err := cl1.RefreshToken(token2)
	assert.NoError(t, err)
	assert.NotEmpty(t, token3)

	// end the session
	cl1.Disconnect()
}

func TestLogout(t *testing.T) {
	t.Log("TestLogout")
	cancelFn, _ := StartTransportServer(DummyMessageHandler)
	defer cancelFn()

	// check if this test still works with a valid login
	cl1 := NewConsumerClient(testClientID1)
	token, err := cl1.ConnectWithPassword(testClientPassword1)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// logout
	err = cl1.Logout()
	assert.NoError(t, err)

	// refresh should not work
	token1, err := cl1.RefreshToken(token)
	assert.Error(t, err)
	assert.Empty(t, token1)
}

func TestBadLogin(t *testing.T) {
	t.Log("TestBadLogin")
	cancelFn, _ := StartTransportServer(DummyMessageHandler)
	defer cancelFn()

	cl1 := NewConsumerClient(testClientID1)

	// check no login
	form := NewForm(vocab.OpReadAllProperties)
	_, err := cl1.SendOperation(form, "thing1", "", nil, nil, "")
	assert.Error(t, err)

	// check if this test still works with a valid login
	token, err := cl1.ConnectWithPassword(testClientPassword1)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// failed logins
	token, err = cl1.ConnectWithPassword("badpass")
	assert.Error(t, err)
	assert.Empty(t, token)

	// bad token should fail
	token, err = cl1.ConnectWithToken("badtoken")
	assert.Error(t, err)
	assert.Empty(t, token)
	token2, err := cl1.RefreshToken(token)
	assert.Error(t, err)
	assert.Empty(t, token2)

	// close should always succeed
	cl1.Disconnect()

	// bad client ID
	cl2 := NewConsumerClient("badID")
	token, err = cl2.ConnectWithPassword(testClientPassword1)
	assert.Error(t, err)
	assert.Empty(t, token)
}

func TestBadRefresh(t *testing.T) {
	t.Log("TestBadRefresh")
	cancelFn, _ := StartTransportServer(DummyMessageHandler)
	defer cancelFn()
	cl1 := NewConsumerClient(testClientID1)
	defer cl1.Disconnect()

	// set the token
	token, err := cl1.ConnectWithToken("badtoken")
	require.Error(t, err)
	assert.Empty(t, token)
	token, err = cl1.RefreshToken(token)
	assert.Error(t, err)
	assert.Empty(t, token)

	// get a valid token and connect with a bad clientid
	token, err = cl1.ConnectWithPassword(testClientPassword1)
	assert.NoError(t, err)
	validToken, err := cl1.RefreshToken(token)
	assert.NoError(t, err)
	assert.NotEmpty(t, validToken)
	cl1.Disconnect()

	// this next test depends on whether the clientID is sent along or if the server
	// uses only the token to determine/view the clientID.
	// In that case, providing the clientID here is just for logging
	//cl2 := NewConsumerClient("badclientidlogin")
	//defer cl2.Disconnect()
	//token, err = cl2.ConnectWithToken(validToken)
	//assert.Error(t, err)
	//assert.Empty(t, token)
	//token, err = cl2.RefreshToken(token)
	//assert.Error(t, err)
	//assert.Empty(t, token)
}

// Auto-reconnect using hub client and server
func TestReconnect(t *testing.T) {
	t.Log("TestReconnect")

	const thingID = "thing1"
	const actionKey = "action1"
	const agentID = "agent1"
	var reconnectedCallback atomic.Bool
	var dThingID = tdd.MakeDigiTwinThingID(agentID, thingID)

	// this test handler receives an action and returns a 'delivered status',
	// it is intended to prove reconnect works.
	handleMessage := func(msg *transports.ThingMessage, replyTo transports.IServerConnection) (
		stat transports.RequestStatus) {
		slog.Info("Received message", "op", msg.Operation)
		// prove that the return channel is connected
		if msg.Operation == vocab.OpInvokeAction {
			go func() {
				// send a completed update a fraction after returning 'delivered'
				time.Sleep(time.Millisecond)
				require.NotNil(t, replyTo)
				stat2 := transports.RequestStatus{}
				stat2.Completed(msg, msg.Data, nil)
				err := replyTo.PublishActionStatus(stat2, agentID)
				require.NoError(t, err)
			}()
		}
		stat.Delivered(msg)
		return stat
	}
	// start the servers and connect as a client
	cancelFn, cm := StartTransportServer(handleMessage)
	defer cancelFn()

	// connect as client
	cl1 := NewConsumerClient(testClientID1)
	token := authenticator.CreateSessionToken(testClientID1, "", 0)
	_, err := cl1.ConnectWithToken(token)
	require.NoError(t, err)
	defer cl1.Disconnect()

	//  wait until the connection is established

	// 3. close connection server side but keep the session.
	// This should trigger auto-reconnect on the client.
	t.Log("--- force disconnecting all clients ---")
	cm.CloseAll()

	// give client time to reconnect
	ctx1, cancelFn1 := context.WithTimeout(context.Background(), time.Second)
	defer cancelFn1()
	cl1.SetConnectHandler(func(connected bool, err error) {
		if connected {
			cancelFn1()
			reconnectedCallback.Store(true)
		}
	})
	<-ctx1.Done()

	// 4. invoke an action which should return a value
	// An RPC call is the ultimate test
	var rpcArgs string = "rpc test"
	var rpcResp string
	// this client call receives the response from the handler above
	form := NewForm(vocab.OpInvokeAction)
	err = cl1.Rpc(form, dThingID, actionKey, &rpcArgs, &rpcResp)
	require.NoError(t, err)
	assert.Equal(t, rpcArgs, rpcResp)

	// expect the re-connected callback to be invoked
	assert.True(t, reconnectedCallback.Load())
}

// Test getting form for unknown operation
func TestBadForm(t *testing.T) {
	t.Log("TestBadForm")

	form := NewForm("badoperation")
	assert.Nil(t, form)
}
