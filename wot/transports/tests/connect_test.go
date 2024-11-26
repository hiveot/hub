package tests

import (
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
	"github.com/teris-io/shortid"
	"log/slog"
	"os"
	"testing"
	"time"
)

const testAgentID1 = "agent1"
const testAgentPassword1 = "agent1pass"
const testClientID1 = "client1"
const testClientPassword1 = "client1pass"
const testServerHttpURL = "https://localhost:9445"
const testServerHttpPort = 9445

var defaultProtocol = transports.ProtocolTypeHTTP
var transportServer transports.ITransportServer

var certBundle = certs.CreateTestCertBundle()

// NewClient creates a new unconnected agent client with the given ID
// This panics if a client cannot be created
func NewAgentClient(clientID string) transports.ITransportClient {
	protocol := defaultProtocol
	fullURL := testServerHttpURL
	caCert := certBundle.CaCert
	bc, err := clients.CreateBindingClient(protocol, fullURL, clientID, caCert)
	if err != nil {
		panic("NewClient failed:" + err.Error())
	}
	// FIXME: align the interfaces for connection, consumer, agent
	return bc
}

// NewConsumerClient creates a new unconnected consumer client with the given ID
// This panics if a client cannot be created
func NewConsumerClient(clientID string) transports.ITransportClient {
	protocol := defaultProtocol
	fullURL := testServerHttpURL
	caCert := certBundle.CaCert
	bc, err := clients.CreateBindingClient(protocol, fullURL, clientID, caCert)
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

	switch defaultProtocol {
	case transports.ProtocolTypeHTTP:
	}
	caCert := certBundle.CaCert
	serverCert := certBundle.ServerCert
	cm = connections.NewConnectionManager()
	dummyAuthenticator := NewDummyAuthenticator()
	dummyAuthenticator.AddClient(testAgentID1, testAgentPassword1)
	dummyAuthenticator.AddClient(testClientID1, testClientPassword1)

	// Start the HTTP binding with SSE-SC and WS sub-protocols
	config := httpserver.NewHttpBindingConfig()
	config.Port = testServerHttpPort
	protocolServer, err := httpserver.StartHttpBindingServer(
		&config, serverCert, caCert, dummyAuthenticator, messageHandler, cm)
	if err != nil {
		panic("Unable to create protocol server")
	}
	return func() {
		protocolServer.Stop()
	}, cm
}

func DummyMessageHandler(msg *transports.ThingMessage,
	replyTo transports.IServerConnection) (stat transports.RequestStatus) {
	slog.Info("Received message", "op", msg.Operation)
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

	token, err := cl1.ConnectWithPassword(testClientPassword1)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// refresh should succeed
	newToken, err := cl1.RefreshToken(token)
	time.Sleep(time.Millisecond * 30)
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
	time.Sleep(time.Millisecond)
}

func TestBadLogin(t *testing.T) {
	t.Log("TestBadLogin")
	cancelFn, _ := StartTransportServer(DummyMessageHandler)
	defer cancelFn()

	// check if this test still works with a valid login
	cl1 := NewConsumerClient(testClientID1)
	token, err := cl1.ConnectWithPassword(testClientPassword1)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// failed logins
	token, err = cl1.ConnectWithPassword("badpass")
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
	//
	cl2 := NewConsumerClient("badclientidlogin")

	defer cl2.Disconnect()
	token, err = cl2.ConnectWithToken(validToken)
	assert.Error(t, err)
	assert.Empty(t, token)
	token, err = cl2.RefreshToken(token)
	assert.Error(t, err)
	assert.Empty(t, token)
}

// Auto-reconnect using hub client and server
func TestReconnect(t *testing.T) {
	t.Log("TestReconnect")

	const thingID = "thing1"
	const actionKey = "action1"
	const agentID = "agent1"
	var dThingID = tdd.MakeDigiTwinThingID(agentID, thingID)

	// this test handler receives an action, returns a 'delivered status',
	// and sends a completed status through the sse return channel (SendToClient)

	handleMessage := func(msg *transports.ThingMessage, replyTo transports.IServerConnection) (
		stat transports.RequestStatus) {
		slog.Info("Received message", "op", msg.Operation)

		if msg.Operation == vocab.OpInvokeAction {
			require.NotNil(t, replyTo)
			// send a delayed completion message
			stat2 := transports.RequestStatus{}
			stat2.Completed(msg, msg.Data, nil)
			err := replyTo.PublishActionStatus(stat2, agentID)
			assert.NoError(t, err)

		}
		stat.Delivered(msg)
		return stat
	}
	// start the servers and connect as a client
	cancelFn, cm := StartTransportServer(handleMessage)
	defer cancelFn()
	cl1 := NewConsumerClient(testClientID1)
	defer cl1.Disconnect()

	//// 2. connect a service client. Service auth tokens remain valid between sessions.
	//cl2 := NewClient(testClientID1, "")
	//defer cl1.Disconnect()
	//token1 := dummyAuthenticator.CreateSessionToken(clientLoginID, "mysession", 1000)
	//token2, err := cl2.ConnectWithToken(token1)
	//require.NoError(t, err)
	//assert.NotEmpty(t, token2)

	//  Give some time for the connection to be established
	time.Sleep(time.Millisecond * 10)

	// 3. close connection server side but keep the session.
	// This should trigger auto-reconnect on the client.
	t.Log("--- force disconnecting all clients ---")
	cm.CloseAll()
	time.Sleep(time.Second)

	// give client time to reconnect
	time.Sleep(time.Second * 3)

	// 4. invoke an action which should return a value
	// An RPC call is the ultimate test
	var rpcArgs string = "rpc test"
	var rpcResp string
	// this client call receives the response from the handler above
	corrID := shortid.MustGenerate()
	form := NewForm(vocab.OpInvokeAction)
	stat := cl1.SendOperation(form, dThingID, actionKey, &rpcArgs, &rpcResp, corrID)
	require.Empty(t, stat.Error)
	assert.Equal(t, rpcArgs, rpcResp)
}
