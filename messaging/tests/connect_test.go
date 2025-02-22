package tests

import (
	"context"
	"errors"
	"fmt"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/clients"
	authenticator2 "github.com/hiveot/hub/messaging/clients/authenticator"
	"github.com/hiveot/hub/messaging/servers/hiveotsseserver"
	"github.com/hiveot/hub/messaging/servers/httpserver"
	"github.com/hiveot/hub/messaging/servers/wssserver"
	"github.com/hiveot/hub/messaging/tputils"
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
const testServerHiveotSseURL = "sse://localhost:9445" + hiveotsseserver.DefaultHiveotSsePath
const testServerHiveotWssURL = "wss://localhost:9445" + wssserver.DefaultHiveotWssPath
const testServerWotWssURL = "wss://localhost:9445" + wssserver.DefaultWotWssPath
const testServerMqttWssURL = "mqtts://localhost:9447"

// var defaultProtocol = transports.ProtocolTypeHiveotSSE
var defaultProtocol = messaging.ProtocolTypeHiveotWSS

//var defaultProtocol = transports.ProtocolTypeWotWSS

var transportServer messaging.ITransportServer
var authenticator *tputils.DummyAuthenticator
var certBundle = certs.CreateTestCertBundle()

// NewClient creates a new connected client with the given client ID. The
// transport server must be started first.
//
// This uses the server to generate an auth token.
// This panics if a client cannot be created.
// ClientID is only used for logging
func NewTestClient(clientID string) (messaging.IClientConnection, string) {
	fullURL := testServerHttpURL
	token := authenticator.AddClient(clientID, clientID)

	switch defaultProtocol {
	case messaging.ProtocolTypeHiveotSSE:
		fullURL = testServerHiveotSseURL

	case messaging.ProtocolTypeHiveotWSS:
		fullURL = testServerHiveotWssURL

	case messaging.ProtocolTypeWotHTTPBasic:
		fullURL = testServerHttpURL

	case messaging.ProtocolTypeWotWSS:
		fullURL = testServerWotWssURL

	case messaging.ProtocolTypeWotMQTTWSS:
		fullURL = testServerMqttWssURL
	}
	caCert := certBundle.CaCert
	cc, err := clients.ConnectWithToken(clientID, token, caCert, defaultProtocol, fullURL, testTimeout)
	if err != nil {
		panic("NewClient failed:" + err.Error())
	}
	return cc, token
}

// NewAgent creates a new connected agent client with the given ID. The
// transport server must be started first.
//
// This uses the clientID as password
// This panics if a client cannot be created
func NewAgent(clientID string) (messaging.IClientConnection, *messaging.Agent, string) {
	cc, token := NewTestClient(clientID)

	agent := messaging.NewAgent(cc, nil, nil, nil, testTimeout)
	return cc, agent, token
}

// NewConsumer creates a new connected consumer client with the given ID.
// The transport server must be started first.
//
// This uses the clientID as password
// This panics if a client cannot be created
func NewConsumer(clientID string, getForm messaging.GetFormHandler) (
	messaging.IClientConnection, *messaging.Consumer, string) {

	cc, token := NewTestClient(clientID)
	co := messaging.NewConsumer(cc, testTimeout)
	return cc, co, token
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
	reqHandler messaging.RequestHandler, respHandler messaging.ResponseHandler,
) (srv messaging.ITransportServer, cancelFunc func()) {

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
	case messaging.ProtocolTypeWotHTTPBasic:
		// http only, no subprotocol bindings
		//httpTransportServer, err = httpserver.StartHttpTransportServer(
		//	"localhost", testServerHttpPort, serverCert, caCert, authenticator, cm,
		//	notifHandler, reqHandler, respHandler)
		panic("https is broken")

	case messaging.ProtocolTypeHiveotSSE:
		transportServer = hiveotsseserver.StartHiveotSseServer(
			hiveotsseserver.DefaultHiveotSsePath,
			httpTransportServer, nil, reqHandler, respHandler)

	case messaging.ProtocolTypeHiveotWSS:
		transportServer, err = wssserver.StartWssServer(
			wssserver.DefaultHiveotWssPath,
			&wssserver.HiveotMessageConverter{}, messaging.ProtocolTypeHiveotWSS,
			httpTransportServer, nil, reqHandler, respHandler)

	case messaging.ProtocolTypeWotWSS:
		transportServer, err = wssserver.StartWssServer(
			wssserver.DefaultWotWssPath,
			&wssserver.WotWssMessageConverter{}, messaging.ProtocolTypeWotWSS,
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
func DummyRequestHandler(req *messaging.RequestMessage,
	c messaging.IConnection) (resp *messaging.ResponseMessage) {
	var output any
	var err error

	slog.Info("DummyRequestHandler: Received request", "op", req.Operation)
	//if req.Operation == wot.HTOpRefresh {
	//	oldToken := req.ToString(0)
	//	output, err = authenticator.RefreshToken(req.SenderID, oldToken)
	//} else if req.Operation == wot.HTOpLogout {
	//	authenticator.Logout(c.GetClientID())
	//} else {
	output = req.Input // echo
	//}
	return req.CreateResponse(output, err)
}
func DummyResponseHandler(response *messaging.ResponseMessage) error {

	slog.Info("DummyResponse: Received response", "op", response.Operation)
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
	cc1, co1, _ := NewConsumer(testClientID1, srv.GetForm)
	defer cc1.Disconnect()
	assert.NotNil(t, co1)

	isConnected := cc1.IsConnected()
	assert.True(t, isConnected)
}

func TestLoginRefresh(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	const thingID1 = "thing1"

	srv, cancelFn := StartTransportServer(nil, nil)
	defer cancelFn()
	cc1, co1, token1 := NewConsumer(testClientID1, srv.GetForm)
	_ = co1

	isConnected := cc1.IsConnected()
	// FIXME: separate auth from the client
	//assert.False(t, isConnected)
	//
	//token1, err := cc1.ConnectWithPassword(testClientID1)
	//require.NoError(t, err)
	//require.NotEmpty(t, token1)

	// check if both client and server have the connection ID
	// the server prefixes it with clientID- to ensure no client can steal another's ID
	// removed this test as it is only important for http/sse. Other tests will fail
	// if they are incorrect.
	//cid1 := co1.GetConnectionID()
	//assert.NotEmpty(t, cid1)
	//srvConn := cm.GetConnectionByClientID(testClientID1)
	//require.NotNil(t, srvConn)
	//cid1server := srvConn.GetConnectionID()
	//assert.Equal(t, testClientID1+"-"+cid1, cid1server)

	isConnected = cc1.IsConnected()
	assert.True(t, isConnected)

	parts, _ := url.Parse(srv.GetConnectURL())
	authCl := authenticator2.NewAuthClient(parts.Host, certBundle.CaCert, "cid1", testTimeout)
	token2, err := authCl.RefreshToken(token1)

	// refresh should succeed
	//token2, err := co1.RefreshToken(token1)
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

	//token3, err := co1.RefreshToken(token2)
	token3, err := authCl.RefreshToken(token2)
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
	cc1, co1, token1 := NewConsumer(testClientID1, srv.GetForm)
	_ = cc1
	_ = co1
	defer co1.Disconnect()
	assert.NotEmpty(t, token1)

	// logout
	authCl := authenticator2.NewAuthClientFromConnection(cc1, token1)
	err := authCl.Logout()

	//authenticator2.Logout(cc1, "")
	//err := co1.Logout()
	t.Log("logged out, some warnings are expected next")
	assert.NoError(t, err)

	// This causes Refresh to fail
	token2, err := authCl.RefreshToken(token1)
	//token2, err := co1.RefreshToken(token1)
	assert.Error(t, err)
	assert.Empty(t, token2)
}

//func TestBadLogin(t *testing.T) {
//	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
//
//	srv, cancelFn := StartTransportServer(nil, nil)
//	defer cancelFn()
//
//	cc1, co1, _ := NewConsumer(testClientID1, srv.GetForm)
//
//	// check if this test still works with a valid login
//	token1, err := cc1.ConnectWithPassword(testClientID1)
//	assert.NoError(t, err)
//
//	// failed logins
//	t.Log("Expecting ConnectWithPassword to fail")
//	token2, err := cc1.ConnectWithPassword("badpass")
//	assert.Error(t, err)
//	assert.Empty(t, token2)
//
//	// can't refresh when no longer connected
//	t.Log("Expecting RefreshToken to fail")
//	token4, err := co1.RefreshToken(token1)
//	assert.Error(t, err)
//	assert.Empty(t, token4)
//
//	// disconnect should always succeed
//	cc1.Disconnect()
//
//	// bad client ID
//	t.Log("Expecting ConnectWithPassword('BadID') to fail")
//	cc2, _, _ := NewConsumer("badID", srv.GetForm)
//	token5, err := cc2.ConnectWithPassword(testClientID1)
//	assert.Error(t, err)
//	assert.Empty(t, token5)
//}

func TestBadRefresh(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	srv, cancelFn := StartTransportServer(nil, nil)
	defer cancelFn()
	cc1, co1, token1 := NewConsumer(testClientID1, srv.GetForm)
	_ = co1
	_ = token1
	defer cc1.Disconnect()

	// set the token
	t.Log("Expecting SetBearerToken('badtoken') to fail")
	err := cc1.ConnectWithToken("badtoken")
	require.Error(t, err)

	// reconnect with a valid token and connect with a bad clientid
	err = cc1.ConnectWithToken(token1)
	assert.NoError(t, err)
	parts, _ := url.Parse(srv.GetConnectURL())
	authCl := authenticator2.NewAuthClient(
		parts.Host, certBundle.CaCert, "cid1", testTimeout)
	validToken, err := authCl.RefreshToken(token1)
	//validToken, err := co1.RefreshToken(token1)
	assert.NoError(t, err)
	assert.NotEmpty(t, validToken)
	cc1.Disconnect()
}

// Auto-reconnect using hub client and server
func TestReconnect(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))

	const thingID = "thing1"
	const actionKey = "action1"
	const agentID = "agent1"
	var reconnectedCallback atomic.Bool
	var dThingID = td.MakeDigiTwinThingID(agentID, thingID)
	var srv messaging.ITransportServer
	var cancelFn func()

	// this test handler receives an action and returns a 'delivered status',
	// it is intended to prove reconnect works.
	handleRequest := func(req *messaging.RequestMessage, c messaging.IConnection) *messaging.ResponseMessage {
		slog.Info("Received request", "op", req.Operation)
		var err error
		// prove that the return channel is connected
		if req.Operation == wot.OpInvokeAction {
			go func() {
				// send a asynchronous result after a fraction after returning 'delivered'
				time.Sleep(time.Millisecond * 10)
				require.NotNil(t, c, "client doesnt have a SSE connection")
				output := req.Input
				cinfo := c.GetConnectionInfo()
				c2 := srv.GetConnectionByConnectionID(cinfo.ClientID, cinfo.ConnectionID)
				assert.NotEmpty(t, c2)
				resp := req.CreateResponse(output, nil)
				err = c.SendResponse(resp)
				assert.NoError(t, err)
			}()
			resp := req.CreateResponse(nil, nil)
			resp.Status = messaging.StatusPending
			return resp
		}
		err = errors.New("Unexpected request")
		return req.CreateResponse("", err)
	}
	// start the servers and connect as a client
	srv, cancelFn = StartTransportServer(handleRequest, nil)
	defer cancelFn()

	// connect as client
	cc1, co1, _ := NewConsumer(testClientID1, srv.GetForm)
	//token := authenticator.CreateSessionToken(testClientID1, "", 0)
	//err := cc1.ConnectWithToken(token)
	//require.NoError(t, err)
	defer cc1.Disconnect()

	//  wait until the connection is established

	// 3. close connection server side but keep the session.
	// This should trigger auto-reconnect on the client.
	t.Log("--- force disconnecting all clients ---")
	srv.CloseAll()

	// give client time to reconnect
	ctx1, cancelFn1 := context.WithTimeout(context.Background(), time.Second)
	defer cancelFn1()
	cc1.SetConnectHandler(func(connected bool, err error, c messaging.IConnection) {
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
	time.Sleep(time.Millisecond * 1000)
	err := co1.Rpc(wot.OpInvokeAction, dThingID, actionKey, &rpcArgs, &rpcResp)
	require.NoError(t, err)
	assert.Equal(t, rpcArgs, rpcResp)

	// expect the re-connected callback to be invoked
	assert.True(t, reconnectedCallback.Load())
}

func TestPing(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))

	srv, cancelFn := StartTransportServer(nil, nil)
	defer cancelFn()
	cc1, co1, _ := NewConsumer(testClientID1, srv.GetForm)
	defer cc1.Disconnect()

	//_, err := cc1.ConnectWithPassword(testClientID1)
	//require.NoError(t, err)

	err := co1.Ping()
	assert.NoError(t, err)

	// FIXME: SSE server sends ping event but it isn't received until later???

	var output any
	err = co1.Rpc(wot.HTOpPing, "", "", nil, &output)
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
	serverURL := srv.GetConnectURL()
	_, err := url.Parse(serverURL)
	require.NoError(t, err)
}
