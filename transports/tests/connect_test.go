package tests

import (
	"context"
	"errors"
	"fmt"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/clients"
	"github.com/hiveot/hub/transports/messaging"
	"github.com/hiveot/hub/transports/servers"
	"github.com/hiveot/hub/transports/servers/httpserver"
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
const testClientID1 = "client1"
const testServerHttpPort = 9445
const testServerHttpURL = "https://localhost:9445"

// const testServerHiveotHttpBasicURL = "wss://localhost:9445" + servers.DefaultHiveotHttpBasicPath
const testServerHiveotSseURL = "https://localhost:9445" + servers.DefaultHiveotSsePath
const testServerHiveotWssURL = "wss://localhost:9445" + servers.DefaultHiveotWssPath
const testServerWotWssURL = "wss://localhost:9445" + servers.DefaultWotWssPath
const testServerMqttWssURL = "mqtts://localhost:9447"

// var defaultProtocol = transports.ProtocolTypeSSESC
var defaultProtocol = transports.ProtocolTypeHiveotWSS

var transportServer transports.ITransportServer
var authenticator *tputils.DummyAuthenticator
var certBundle = certs.CreateTestCertBundle()

// NewClient creates a new unconnected client with the given client ID. The
// transport server must be started first.
//
// This uses the clientID as password
// This panics if a client cannot be created
// ClientID is only used for logging
func NewTestClient(clientID string) transports.IClientConnection {
	fullURL := testServerHttpURL
	authenticator.AddClient(clientID, clientID)

	switch defaultProtocol {
	case transports.ProtocolTypeHiveotSSE:
		fullURL = testServerHiveotSseURL

	case transports.ProtocolTypeHiveotWSS:
		fullURL = testServerHiveotWssURL

	case transports.ProtocolTypeWotHTTPBasic:
		fullURL = testServerHttpURL

	case transports.ProtocolTypeWotWSS:
		fullURL = testServerWotWssURL

	case transports.ProtocolTypeWotMQTTWSS:
		fullURL = testServerMqttWssURL
	}
	caCert := certBundle.CaCert
	cc, err := clients.NewClient(fullURL, clientID, caCert, nil, testTimeout)
	if err != nil {
		panic("NewClient failed:" + err.Error())
	}
	return cc
}

// NewAgent creates a new unconnected agent client with the given ID. The
// transport server must be started first.
//
// This uses the clientID as password
// This panics if a client cannot be created
func NewAgent(clientID string) (transports.IClientConnection, *messaging.Agent) {
	cc := NewTestClient(clientID)
	agent := messaging.NewAgent(cc, nil, nil, nil, testTimeout)
	return cc, agent
}

// NewConsumer creates a new unconnected consumer client with the given ID.
// The transport server must be started first.
//
// This uses the clientID as password
// This panics if a client cannot be created
func NewConsumer(clientID string, getForm transports.GetFormHandler) (
	transports.IClientConnection, *messaging.Consumer) {
	cc := NewTestClient(clientID)
	consumer := messaging.NewConsumer(cc, testTimeout)
	return cc, consumer
}

// Create a new form for the given operation
// This uses the default protocol binding server to generate the Form
func NewForm(op, thingID, name string) *td.Form {
	form := transportServer.GetForm(op, thingID, name)
	return form
}

// start the default transport server
// This panics if the server cannot be created
func StartTransportServer(
	reqHandler transports.RequestHandler, respHandler transports.ResponseHandler,
) (srv transports.ITransportServer, cancelFunc func()) {

	caCert := certBundle.CaCert
	serverCert := certBundle.ServerCert
	authenticator = tputils.NewDummyAuthenticator()
	if reqHandler == nil {
		reqHandler = DummyRequestHandler
	}
	if respHandler == nil {
		respHandler = DummyResponseHandler
	}

	httpTransportServer, err := httpserver.StartHttpTransportServer(
		"localhost", testServerHttpPort, serverCert, caCert, authenticator)

	switch defaultProtocol {
	case transports.ProtocolTypeWotHTTPBasic:
		// http only, no subprotocol bindings
		//httpTransportServer, err = httpserver.StartHttpTransportServer(
		//	"localhost", testServerHttpPort, serverCert, caCert, authenticator, cm,
		//	notifHandler, reqHandler, respHandler)
		panic("https is broken")

	case transports.ProtocolTypeHiveotSSE:
		//transportServer = ssescserver.StartSseScTransportServer("", cm, httpTransportServer)
		// add support for hiveot protocol in http/ssesc
		//hiveotwssserver.StartHiveotProtocolServer(
		//	authenticator, cm, httpTransportServer,
		//	notifHandler, reqHandler, respHandler)
		panic("sse-sc is broken")

	case transports.ProtocolTypeWotWSS:
		//httpTransportServer, err = httpserver.StartHttpTransportServer(
		//	"localhost", testServerHttpPort, serverCert, caCert, authenticator, cm,
		//	// FIXME: add handlers
		//	nil, nil, nil)
		//transportServer = wssserver.StartWssTransportServer(
		//	"", cm, httpTransportServer,
		//	notifHandler, reqHandler, respHandler)
		panic("wss (strawman) is broken")

	case transports.ProtocolTypeHiveotWSS:
		transportServer, err = wssserver.StartHiveotWssServer(
			servers.DefaultHiveotWssPath,
			&wssserver.HiveotMessageConverter{}, transports.ProtocolTypeWotWSS,
			httpTransportServer, nil, reqHandler, respHandler)
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
	}
}

// func DummyNotificationHandler(notification transports.NotificationMessage) {
//
//		slog.Info("DummyNotificationHandler: Received notification", "op", notification.Operation)
//		//replyTo.SendResponse(msg.ThingID, msg.Name, "result", msg.CorrelationID)
//	}
func DummyRequestHandler(req *transports.RequestMessage,
	c transports.IConnection) (resp *transports.ResponseMessage) {
	var output any
	var err error

	slog.Info("DummyRequestHandler: Received request", "op", req.Operation)
	if req.Operation == wot.HTOpRefresh {
		oldToken := req.ToString(0)
		output, err = authenticator.RefreshToken(req.SenderID, oldToken)
	} else if req.Operation == wot.HTOpLogout {
		authenticator.Logout(c.GetClientID())
	} else {
		output = req.Input // echo
	}
	return req.CreateResponse(output, err)
}
func DummyResponseHandler(response *transports.ResponseMessage) error {

	slog.Info("DummyResponse: Received request", "op", response.Operation)
	return nil
}

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

	srv, cancelFn := StartTransportServer(nil, nil)
	defer cancelFn()
	cc1, cl1 := NewConsumer(testClientID1, srv.GetForm)
	defer cc1.Disconnect()
	assert.NotNil(t, cl1)

	token, err := cc1.ConnectWithPassword(testClientID1)
	require.NoError(t, err)
	require.NotEmpty(t, token)
}

func TestLoginRefresh(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	const thingID1 = "thing1"

	srv, cancelFn := StartTransportServer(nil, nil)
	defer cancelFn()
	cc1, cl1 := NewConsumer(testClientID1, srv.GetForm)
	defer cc1.Disconnect()

	isConnected := cc1.IsConnected()
	assert.False(t, isConnected)

	token1, err := cc1.ConnectWithPassword(testClientID1)
	require.NoError(t, err)
	require.NotEmpty(t, token1)

	// check if both client and server have the connection ID
	// the server prefixes it with clientID- to ensure no client can steal another's ID
	// removed this test as it is only important for http/sse. Other tests will fail
	// if they are incorrect.
	//cid1 := cl1.GetConnectionID()
	//assert.NotEmpty(t, cid1)
	//srvConn := cm.GetConnectionByClientID(testClientID1)
	//require.NotNil(t, srvConn)
	//cid1server := srvConn.GetConnectionID()
	//assert.Equal(t, testClientID1+"-"+cid1, cid1server)

	err = cl1.Ping()
	require.NoError(t, err)

	// FIXME: SSE server sends ping event but it isn't received until later???

	isConnected = cc1.IsConnected()
	assert.True(t, isConnected)

	// refresh should succeed
	token2, err := cl1.RefreshToken(token1)
	require.NoError(t, err)
	require.NotEmpty(t, token2)

	// end the connection
	cc1.Disconnect()
	time.Sleep(time.Millisecond * 1)

	// should be able to reconnect with the new token
	// NOTE: the runtime session manager doesn't allow this as
	// the session no longer exists, but the authenticator doesn't care.
	err = cc1.ConnectWithToken(token2)
	require.NoError(t, err)

	token3, err := cl1.RefreshToken(token2)
	assert.NoError(t, err)
	assert.NotEmpty(t, token3)

	// end the session
	cc1.Disconnect()
}

func TestLogout(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))

	srv, cancelFn := StartTransportServer(nil, nil)
	defer cancelFn()

	// check if this test still works with a valid login
	cc1, cl1 := NewConsumer(testClientID1, srv.GetForm)
	token1, err := cc1.ConnectWithPassword(testClientID1)
	assert.NoError(t, err)
	assert.NotEmpty(t, token1)

	// logout
	err = cl1.Logout()
	t.Log("logged out, some warnings are expected next")
	assert.NoError(t, err)

	// This causes Refresh to fail
	token2, err := cl1.RefreshToken(token1)
	assert.Error(t, err)
	assert.Empty(t, token2)
}

func TestBadLogin(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))

	srv, cancelFn := StartTransportServer(nil, nil)
	defer cancelFn()

	cc1, cl1 := NewConsumer(testClientID1, srv.GetForm)

	// check if this test still works with a valid login
	token1, err := cc1.ConnectWithPassword(testClientID1)
	assert.NoError(t, err)

	// failed logins
	token2, err := cc1.ConnectWithPassword("badpass")
	assert.Error(t, err)
	assert.Empty(t, token2)

	// bad token should fail
	err = cc1.ConnectWithToken("badtoken")
	assert.Error(t, err)

	// can't refresh when no longer connected
	token4, err := cl1.RefreshToken(token1)
	assert.Error(t, err)
	assert.Empty(t, token4)

	// disconnect should always succeed
	cc1.Disconnect()

	// bad client ID
	cc2, _ := NewConsumer("badID", srv.GetForm)
	token5, err := cc2.ConnectWithPassword(testClientID1)
	assert.Error(t, err)
	assert.Empty(t, token5)
}

func TestBadRefresh(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	srv, cancelFn := StartTransportServer(nil, nil)
	defer cancelFn()
	cc1, cl1 := NewConsumer(testClientID1, srv.GetForm)
	defer cc1.Disconnect()

	// set the token
	err := cc1.ConnectWithToken("badtoken")
	require.Error(t, err)

	// get a valid token and connect with a bad clientid
	token2, err := cc1.ConnectWithPassword(testClientID1)
	assert.NoError(t, err)
	validToken, err := cl1.RefreshToken(token2)
	assert.NoError(t, err)
	assert.NotEmpty(t, validToken)
	cc1.Disconnect()

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
	var srv transports.ITransportServer
	var cancelFn func()

	// this test handler receives an action and returns a 'delivered status',
	// it is intended to prove reconnect works.
	handleRequest := func(req *transports.RequestMessage, c transports.IConnection) *transports.ResponseMessage {
		slog.Info("Received request", "op", req.Operation)
		var err error
		// prove that the return channel is connected
		if req.Operation == wot.OpInvokeAction {
			go func() {
				// send a asynchronous result after a fraction after returning 'delivered'
				time.Sleep(time.Millisecond)
				require.NotNil(t, c)
				output := req.Input
				c2 := srv.GetConnectionByConnectionID(c.GetConnectionID())
				assert.NotEmpty(t, c2)
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
	srv, cancelFn = StartTransportServer(handleRequest, nil)
	defer cancelFn()

	// connect as client
	cc1, cl1 := NewConsumer(testClientID1, srv.GetForm)
	token := authenticator.CreateSessionToken(testClientID1, "", 0)
	err := cc1.ConnectWithToken(token)
	require.NoError(t, err)
	defer cc1.Disconnect()

	//  wait until the connection is established

	// 3. close connection server side but keep the session.
	// This should trigger auto-reconnect on the client.
	t.Log("--- force disconnecting all clients ---")
	srv.CloseAll()

	// give client time to reconnect
	ctx1, cancelFn1 := context.WithTimeout(context.Background(), time.Second)
	defer cancelFn1()
	cc1.SetConnectHandler(func(connected bool, err error, c transports.IConnection) {
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
	time.Sleep(time.Millisecond * 10)
	err = cl1.Rpc(wot.OpInvokeAction, dThingID, actionKey, &rpcArgs, &rpcResp)
	require.NoError(t, err)
	assert.Equal(t, rpcArgs, rpcResp)

	// expect the re-connected callback to be invoked
	assert.True(t, reconnectedCallback.Load())
}

func TestPing(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))

	srv, cancelFn := StartTransportServer(nil, nil)
	defer cancelFn()
	cc1, cl1 := NewConsumer(testClientID1, srv.GetForm)
	defer cc1.Disconnect()

	_, err := cc1.ConnectWithPassword(testClientID1)
	require.NoError(t, err)

	var output any
	err = cl1.Rpc(wot.HTOpPing, "", "", nil, &output)
	assert.Equal(t, "pong", output)
	assert.NoError(t, err)
}

// Test getting form for unknown operation
func TestBadForm(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))

	_, cancelFn := StartTransportServer(nil, nil)
	defer cancelFn()

	form := NewForm("badoperation", "", "")
	assert.Nil(t, form)
}

// Test getting server URL
func TestServerURL(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))

	srv, cancelFn := StartTransportServer(nil, nil)
	defer cancelFn()
	serverURL := srv.GetConnectURL("")
	_, err := url.Parse(serverURL)
	require.NoError(t, err)
}
