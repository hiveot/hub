package tests

import (
	"context"
	"errors"
	"fmt"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/clients"
	"github.com/hiveot/hub/transports/connections"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"github.com/hiveot/hub/transports/servers/ssescserver"
	"github.com/hiveot/hub/transports/servers/wssserver"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"net/url"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

const testTimeout = time.Second * 300
const testAgentID1 = "agent1"
const testAgentPassword1 = "agent1pass"
const testClientID1 = "client1"
const testClientPassword1 = "client1pass"
const testServerHttpPort = 9445
const testServerHttpURL = "https://localhost:9445"
const testServerSseURL = "https://localhost:9445/sse"
const testServerSsescURL = "https://localhost:9445/ssesc"
const testServerWssURL = "wss://localhost:9445/wss"
const testServerMqttURL = "mqtts://localhost:9446"

var defaultProtocol = transports.ProtocolTypeSSESC

// var defaultProtocol = transports.ProtocolTypeWSS
var transportServer transports.ITransportServer
var authenticator *tputils.DummyAuthenticator
var certBundle = certs.CreateTestCertBundle()

// NewAgent creates a new unconnected agent client with the given ID
// This panics if a client cannot be created
// ClientID is only used for logging
func NewAgent(clientID string, getForm func(op string) td.Form) transports.IAgentConnection {
	fullURL := testServerHttpURL

	switch defaultProtocol {
	case transports.ProtocolTypeHTTPS:
		fullURL = testServerHttpURL
	case transports.ProtocolTypeSSE:
		fullURL = testServerSseURL
	case transports.ProtocolTypeSSESC:
		fullURL = testServerSsescURL
	case transports.ProtocolTypeWSS:
		fullURL = testServerWssURL
	case transports.ProtocolTypeMQTTS:
		fullURL = testServerMqttURL
	}
	caCert := certBundle.CaCert
	bc, err := clients.NewTransportClient(fullURL, clientID, caCert, getForm, testTimeout)
	if err != nil {
		panic("NewClient failed:" + err.Error())
	}
	return bc
}

// NewConsumer creates a new unconnected consumer client with the given ID
// This panics if a client cannot be created
// ClientID is only used for logging
func NewConsumer(clientID string, getForm func(op string) td.Form) transports.IConsumerConnection {
	return NewAgent(clientID, getForm)
}

// Create a new form for the given operation
// This uses the default protocol binding server to generate the Form
func NewForm(op string) td.Form {
	form := transportServer.GetForm(op)
	return form
}

// start the default transport server
// This panics if the server cannot be created
func StartTransportServer(
	rqh transports.ServerRequestHandler,
	rph transports.ServerResponseHandler,
	nth transports.ServerNotificationHandler) (

	srv transports.ITransportServer, cancelFunc func(), cm *connections.ConnectionManager) {
	var err error
	caCert := certBundle.CaCert
	serverCert := certBundle.ServerCert
	cm = connections.NewConnectionManager()
	authenticator = tputils.NewDummyAuthenticator()
	authenticator.AddClient(testAgentID1, testAgentPassword1)
	authenticator.AddClient(testClientID1, testClientPassword1)
	var httpTransportServer *httpserver.HttpTransportServer

	switch defaultProtocol {
	case transports.ProtocolTypeHTTPS:
		// http only, no subprotocol bindings
		httpTransportServer, err = httpserver.StartHttpTransportServer(
			"localhost", testServerHttpPort, serverCert, caCert, authenticator, cm,
			rqh, rph, nth)

	case transports.ProtocolTypeSSESC:
		httpTransportServer, err = httpserver.StartHttpTransportServer(
			"localhost", testServerHttpPort, serverCert, caCert, authenticator, cm,
			rqh, rph, nth)
		transportServer = ssescserver.StartSseScTransportServer("", cm, httpTransportServer)

	case transports.ProtocolTypeWSS:
		httpTransportServer, err = httpserver.StartHttpTransportServer(
			"localhost", testServerHttpPort, serverCert, caCert, authenticator, cm,
			// FIXME: add handlers
			nil, nil, nil)
		transportServer = wssserver.StartWssTransportServer(
			"", cm, httpTransportServer,
			rqh, rph, nth)
	default:
		err = errors.New("unknown protocol name: " + defaultProtocol)
	}

	if err != nil {
		panic("Unable to create protocol server: " + err.Error())
	}
	//transportServer.SetRequestHandler(cm.AddConnection)
	//transportServer.SetMessageHandler(cm.AddConnection)

	return transportServer, func() {
		if transportServer != nil {
			transportServer.Stop()
		}
		if httpTransportServer != nil {
			httpTransportServer.Stop()
		}
	}, cm
}

//
//func DummyNotificationHandler(notification transports.NotificationMessage) {
//
//	slog.Info("DummyNotificationHandler: Received notification", "op", notification.Operation)
//	//replyTo.SendResponse(msg.ThingID, msg.Name, "result", msg.RequestID)
//}
//func DummyRequestHandler(request transports.RequestMessage, replyTo string) transports.ResponseMessage {
//
//	slog.Info("DummyRequestHandler: Received request", "op", request.Operation)
//	return request.CreateResponse("result", nil)
//}
//func DummyResponseHandler(response transports.ResponseMessage) error {
//
//	slog.Info("DummyResponse: Received request", "op", response.Operation)
//	return nil
//}

// TestMain sets logging
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	defaultProtocol = defaultProtocol
	result := m.Run()
	os.Exit(result)
}

// test create a server and connect a client
func TestStartStop(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))

	srv, cancelFn, _ := StartTransportServer(nil, nil, nil)
	defer cancelFn()
	cl1 := NewConsumer(testClientID1, srv.GetForm)
	defer cl1.Disconnect()

	token, err := cl1.ConnectWithPassword(testClientPassword1)
	require.NoError(t, err)
	require.NotEmpty(t, token)
}

func TestLoginRefresh(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	const thingID1 = "thing1"

	srv, cancelFn, _ := StartTransportServer(nil, nil, nil)
	defer cancelFn()
	cl1 := NewConsumer(testClientID1, srv.GetForm)
	defer cl1.Disconnect()

	isConnected := cl1.IsConnected()
	assert.False(t, isConnected)

	token, err := cl1.ConnectWithPassword(testClientPassword1)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	//time.Sleep(time.Millisecond * 1)

	err = cl1.Ping()
	require.NoError(t, err)

	// FIXME: SSE server sends ping event but it isn't received until later???

	isConnected = cl1.IsConnected()
	assert.True(t, isConnected)

	// refresh should succeed
	newToken, err := cl1.RefreshToken(token)
	require.NoError(t, err)
	require.NotEmpty(t, newToken)

	// end the connection
	cl1.Disconnect()
	time.Sleep(time.Millisecond * 1)

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
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	srv, cancelFn, _ := StartTransportServer(nil, nil, nil)
	defer cancelFn()

	// check if this test still works with a valid login
	cl1 := NewConsumer(testClientID1, srv.GetForm)
	token, err := cl1.ConnectWithPassword(testClientPassword1)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// logout
	err = cl1.Logout()
	t.Log("logged out, some warnings are expected next")
	assert.NoError(t, err)

	// This causes Refresh to fail
	token1, err := cl1.RefreshToken(token)
	assert.Error(t, err)
	assert.Empty(t, token1)
}

func TestBadLogin(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	srv, cancelFn, _ := StartTransportServer(nil, nil, nil)
	defer cancelFn()

	cl1 := NewConsumer(testClientID1, srv.GetForm)

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
	cl2 := NewConsumer("badID", srv.GetForm)
	token, err = cl2.ConnectWithPassword(testClientPassword1)
	assert.Error(t, err)
	assert.Empty(t, token)
}

func TestBadRefresh(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	srv, cancelFn, _ := StartTransportServer(nil, nil, nil)
	defer cancelFn()
	cl1 := NewConsumer(testClientID1, srv.GetForm)
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
	//cl2 := NewClient("badclientidlogin")
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
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))

	const thingID = "thing1"
	const actionKey = "action1"
	const agentID = "agent1"
	var reconnectedCallback atomic.Bool
	var dThingID = td.MakeDigiTwinThingID(agentID, thingID)
	var connMgr *connections.ConnectionManager

	// this test handler receives an action and returns a 'delivered status',
	// it is intended to prove reconnect works.
	handleRequest := func(req transports.RequestMessage, replyTo string) transports.ResponseMessage {
		slog.Info("Received request", "op", req.Operation)
		var err error
		// prove that the return channel is connected
		if req.Operation == wot.OpInvokeAction {
			go func() {
				// send a asynchronous result after a fraction after returning 'delivered'
				time.Sleep(time.Millisecond)
				require.NotNil(t, replyTo)
				output := req.Input
				c := connMgr.GetConnectionByConnectionID(replyTo)
				resp := req.CreateResponse(output, nil)
				err = c.SendResponse(resp)
				assert.NoError(t, err)
			}()
			resp := req.CreateResponse(nil, nil)
			resp.Status = transports.StatusPending
			return resp
		}
		err = errors.New("Unexpected request")
		return req.CreateResponse("", err)
	}
	// start the servers and connect as a client
	srv, cancelFn, cm := StartTransportServer(handleRequest, nil, nil)
	connMgr = cm
	defer cancelFn()

	// connect as client
	cl1 := NewConsumer(testClientID1, srv.GetForm)
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

	err = cl1.Rpc(wot.OpInvokeAction, dThingID, actionKey, &rpcArgs, &rpcResp)
	require.NoError(t, err)
	assert.Equal(t, rpcArgs, rpcResp)

	// expect the re-connected callback to be invoked
	assert.True(t, reconnectedCallback.Load())
}

func TestPing(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))

	srv, cancelFn, _ := StartTransportServer(nil, nil, nil)
	defer cancelFn()
	cl1 := NewConsumer(testClientID1, srv.GetForm)
	defer cl1.Disconnect()

	_, err := cl1.ConnectWithPassword(testClientPassword1)
	require.NoError(t, err)

	var output any
	err = cl1.Rpc(wot.HTOpPing, "", "", nil, &output)
	assert.Equal(t, "pong", output)
	assert.NoError(t, err)
}

// Test getting form for unknown operation
func TestBadForm(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))

	_, cancelFn, _ := StartTransportServer(nil, nil, nil)
	defer cancelFn()

	form := NewForm("badoperation")
	assert.Nil(t, form)
}

// Test getting server URL
func TestServerURL(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))

	srv, cancelFn, _ := StartTransportServer(nil, nil, nil)
	defer cancelFn()
	serverURL := srv.GetConnectURL()
	_, err := url.Parse(serverURL)
	require.NoError(t, err)
}
