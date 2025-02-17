package test

import (
	"fmt"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	authz "github.com/hiveot/hub/runtime/authz/api"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/service"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/clients/httpsseclient"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"github.com/hiveot/hub/transports/tputils/tlsclient"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teris-io/shortid"
	"net/http"
	"os"
	"path"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const serviceID = "hiveoview-test"
const servicePort = 9999
const serverAddr = "localhost"

// set to true to test without state service
const noState = true
const timeout = time.Second * 100

var testFolder = path.Join(os.TempDir(), "test-hiveoview")

// the following are set by the testmain
var ts *testenv.TestServer

// return the form with href for login operations to the hiveoview server
// these must match the paths in hiveoview CreateRoutes.
func getHiveoviewForm(op, thingID, name string) *td.Form {
	var href string
	var method string
	switch op {
	//case wot.HTOpLogin:
	//	href = "/login"
	//	method = "POST"
	//case wot.HTOpLoginWithForm:
	//	href = "/loginForm"
	//	method = "GET"
	//case wot.HTOpLogout:
	//	href = "/logout"
	//	method = "POST"
	//case wot.HTOpRefresh:
	//	href = "/refresh" // todo
	//	method = "POST"
	default:
		panic("Unexpected operation: " + op)
	}
	f := td.NewForm(op, href)
	f.SetMethodName(method)
	return &f
}

// Helper function to login as a web client to the hiveoview server using
// the same sse client as the transport provides.
// The TestLogin test must succeed before using this.
// This returns a client. Call Close() when done.
func WebLogin(sseURL string, clientID string,
	onConnection func(bool, error, transports.IConnection),
	onRequest transports.RequestHandler,
	onResponse transports.ResponseHandler,
) (
	cl transports.IConnection, err error) {

	//sseCl := clients.NewHubClient(sseURL, clientID, ts.Certs.CaCert)
	// websocket client
	//sseCl := wssclient.NewWssTransportClient(
	//	sseURL, clientID, nil, ts.Certs.CaCert, time.Minute)
	// or ssesc client

	// use the hub's SSE client to connect to the hiveoview server as a browser.
	// FIXME: hiveoview server uses different SSE event payload than hiveoview
	// the 'event' ID contains thingID etc. We can't change this because
	// htmx sse triggers rely on this format. (for now)
	// FIXME: can htmx sse trigger using additional fields (type=notification, thingID/name=blah?)
	// or is this too painful in htmx.
	sseCl := httpsseclient.NewHiveotSseClient(sseURL,
		clientID, nil, ts.Certs.CaCert,
		getHiveoviewForm, time.Minute)

	// hiveoview uses a different login path as the hub
	//sseCl.SetSSEPath(service.WebSsePath)
	sseCl.SetConnectHandler(onConnection)
	sseCl.SetResponseHandler(onResponse)
	sseCl.SetRequestHandler(onRequest)

	// a login is needed to create a session tied to the token
	//cid := sseCl.GetConnectionID()
	//token, err := authenticator.AuthenticateWithPassword(
	//	sseURL, src.UIPostLoginPath, clientID, clientID, ts.Certs.CaCert, cid)
	// FIXME: password is clientID
	//token, err := ts.LoginWithPassword(clientID, clientID)
	if err != nil {
		return nil, err
	}
	// hiveot http requires a connection-id to link the return channel.
	//cl2.SetHeader(httpserver.ConnectionIDHeader, shortid.MustGenerate())

	// The hiveoview server will set a cookie,needed for further requests.
	// So the login must use the same client as the requests.
	loginMessage := map[string]string{
		"login":    clientID,
		"password": clientID,
	}
	loginJSON, _ := jsoniter.Marshal(loginMessage)
	outputRaw, _, status, err := sseCl.Send(http.MethodPost, src.UIPostLoginPath, loginJSON)
	if err == nil {
		var token string
		err = jsoniter.Unmarshal(outputRaw, &token)
		_ = status
		err = sseCl.ConnectWithToken(token)
	}
	return sseCl, err
}

func TestMain(m *testing.M) {
	var err error
	// raise loglevel where you want it in testing
	logging.SetLogging("warn", "")
	// clean start
	_ = os.RemoveAll(testFolder)
	err = os.MkdirAll(testFolder, 0700)

	ts = testenv.StartTestServer(true)
	if err != nil {
		panic(err)
	}

	res := m.Run()
	ts.Stop()
	os.Exit(res)
}

func TestStartStop(t *testing.T) {
	t.Log("--- TestStartStop ---")

	svc := service.NewHiveovService(servicePort, true, nil, "",
		ts.Certs.ServerCert, ts.Certs.CaCert, noState, timeout)
	hc1, _ := ts.AddConnectService(serviceID)

	err := svc.Start(hc1)
	require.NoError(t, err)
	time.Sleep(time.Second * 3)
	svc.Stop()
}

// test login from a client using password
func TestLogin(t *testing.T) {
	const clientID1 = "user1"

	// 1: setup: start a runtime and service; this generates an error that
	//    the state service isnt found. ignore it.
	svc := service.NewHiveovService(servicePort, true, nil,
		"", ts.Certs.ServerCert, ts.Certs.CaCert, noState, timeout)
	avcAg, _ := ts.AddConnectService(serviceID)
	require.NotNil(t, avcAg)
	defer avcAg.Disconnect()
	err := svc.Start(avcAg)
	require.NoError(t, err)
	defer svc.Stop()

	// make sure the client to login as exists
	cl1, _, token1 := ts.AddConnectConsumer(clientID1, authz.ClientRoleOperator)
	//defer cl1.Disconnect()
	cl1.Disconnect()
	_ = token1
	time.Sleep(time.Millisecond * 10)

	// 2: login using plain TLS connection and a form
	hostPort := fmt.Sprintf("localhost:%d", servicePort)
	cl2 := tlsclient.NewTLSClient(
		hostPort, nil, ts.Certs.CaCert, time.Second*60)

	// hiveot http requires a connection-id to link the return channel.
	cl2.SetHeader(httpserver.ConnectionIDHeader, shortid.MustGenerate())

	// try login. The test user password is the clientID
	// authenticate the connection with the hiveot http/sse service (not the hub server)
	// the service will in turn forward the request to the hub.
	formData := map[string]string{
		"login":    clientID1,
		"password": clientID1,
	}
	// this login will set an auth cookie
	resp, statusCode, err := cl2.PostForm(src.UIPostFormLoginPath, formData)
	//resp holds the serialized new token
	cl2.Close()
	require.NoError(t, err)
	assert.Equal(t, 200, statusCode)

	// login should have redirected to /dashboard. It contained an auth cookie
	//assert.Equal(t, "/dashboard", resp.Request.URL.Path)

	// result contains a redirected web page
	assert.True(t, strings.HasPrefix(string(resp), "<!DOCTYPE html>"))
	t.Log("TestLogin completed")
}

// test many connections from a single client and confirm they open close and receive messages properly.
func TestMultiConnectDisconnect(t *testing.T) {
	const clientID1 = "user1"
	const agentID = "agent1"
	const testConnections = 1
	const eventName = "event1"
	var webClients = make([]transports.IConnection, 0)
	var connectCount atomic.Int32
	var disConnectCount atomic.Int32
	var messageCount atomic.Int32
	const waitamoment = time.Millisecond * 10

	logging.SetLogging("info", "")
	// 1: setup: start a runtime and service; this generates an error that
	//    the state service isnt found. ignore it.
	svc := service.NewHiveovService(servicePort, true, nil,
		"", ts.Certs.ServerCert, ts.Certs.CaCert, noState, timeout)
	avcAg, _ := ts.AddConnectService(serviceID)
	err := svc.Start(avcAg)

	require.NoError(t, err)
	defer svc.Stop()

	// the agent for publishing events. A TD is needed for them to be accepted.
	ag1, _, _ := ts.AddConnectAgent(agentID)
	_ = ag1
	td1 := ts.AddTD(agentID, nil)
	_ = td1

	// create the user account this test is going to connect as.
	// no notifications are expected as it doesnt subscribe
	// hiveoview server only supports HTTP/SSE
	cl1, _, token1 := ts.AddConnectConsumer(clientID1, authz.ClientRoleOperator)
	defer cl1.Disconnect()
	//err = cl1.Subscribe("", "")
	require.NoError(t, err)
	time.Sleep(waitamoment)

	_ = token1
	//handler for web connection notifications
	onConnection := func(connected bool, err error, _ transports.IConnection) {
		if connected {
			connectCount.Add(1)
		} else {
			disConnectCount.Add(1)
		}
	}
	// handler hiveoview SSE notifications
	onResponse := func(msg *transports.ResponseMessage) error {
		// the UI expects this format for triggering htmx
		expectedType := fmt.Sprintf("dtw:%s:%s/%s", agentID, td1.ID, eventName)
		if msg.Operation == expectedType {
			messageCount.Add(1)
		}
		return nil
	}

	// 2: connect and subscribe web clients and verify
	// each webclient connection will trigger a separate connection to the hub
	// with its own subscription.
	// The hiveoview server only supports SSE on a fixed (WebSsePath) path
	hiveoviewURL := fmt.Sprintf("https://localhost:%d%s", servicePort, service.WebSsePath)
	for range testConnections {
		sseCl, err := WebLogin(
			hiveoviewURL, clientID1, onConnection, nil, onResponse)
		// an event streamm error occurs when auth fails
		require.NoError(t, err)
		require.NotNil(t, sseCl)
		webClients = append(webClients, sseCl)
		time.Sleep(waitamoment)
	}
	// connection notification should have been received N times
	time.Sleep(time.Second * 1)
	assert.Equal(t, testConnections, int(connectCount.Load()), "connect count mismatch")
	// the hiveoview session manager should have corresponding connections
	nrSessions := svc.GetSM().GetNrSessions()
	require.Equal(t, testConnections, nrSessions)

	// 3: agent publishes an event, which should be received N times
	err = ag1.PubEvent(td1.ID, eventName, "a value")
	require.NoError(t, err)

	// event should have been received N times
	time.Sleep(time.Millisecond * 100)
	assert.Equal(t, testConnections, int(messageCount.Load()), "missing events")

	// 4: disconnect
	//sm := svc.GetSM()
	for _, c := range webClients {

		// disconnect the client connection
		c.Disconnect()
		time.Sleep(waitamoment)
	}
	//time.Sleep(waitamoment)
	t.Log("All user1 connections should have been closed")
	// disconnection notification should have been received N times
	time.Sleep(waitamoment)
	require.Equal(t, testConnections, int(disConnectCount.Load()), "disconnect count mismatch")

	// 5: no more messages should be received after disconnecting
	messageCount.Store(0)
	err = ag1.PubEvent(td1.ID, eventName, "a value")
	require.NoError(t, err)

	// zero events should have been received
	time.Sleep(waitamoment)
	assert.Equal(t, int32(0), messageCount.Load(), "still receiving events afer disconnect")

	// last, the service should have no connections
	ag1.Disconnect()
	avcAg.Disconnect()
	time.Sleep(waitamoment)

	// FIXME: currently one or two connections remain
	// the root cause of the first is that the first browser load doesn't connect
	// with SSE and this never closes the connection.
	// The second remaining session doesn't happen while debugging .. yeah fun
	//nrConnections, _ := ts.Runtime.TransportsMgr.GetNrConnections()
	//nrSessions = svc.GetSM().GetNrSessions()
	//if nrConnections > 0 {
	//	t.Log(fmt.Sprintf(
	//		"FIXME: expected 0 remaining connections and sessions. "+
	//			"Got '%d' connections from '%d' sessions", nrConnections, nrSessions))
	//}
	//assert.Less(t, 3, count)

	//time.Sleep(time.Millisecond * 100)
}
