package test

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/service"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/clients/sseclient"
	"github.com/hiveot/hub/transports/tputils/tlsclient"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/url"
	"os"
	"path"
	"sync/atomic"
	"testing"
	"time"
)

const serviceID = "hiveoview-test"
const servicePort = 9999

// set to true to test without state service
const noState = true

var testFolder = path.Join(os.TempDir(), "test-hiveoview")

// the following are set by the testmain
var ts *testenv.TestServer

// return the form with href for login operations to the hiveoview server
// these must match the paths in hiveoview CreateRoutes.
func getHiveoviewForm(op string) *td.Form {
	var href string
	var method string
	switch op {
	case wot.HTOpLogin:
		href = "/login"
		method = "POST"
	case wot.HTOpLoginWithForm:
		href = "/loginForm"
		method = "GET"
	case wot.HTOpLogout:
		href = "/logout"
		method = "POST"
	case wot.HTOpRefresh:
		href = "/refresh" // todo
		method = "POST"
	default:
		panic("Unexpected operation: " + op)
	}
	f := td.NewForm(op, href)
	f.SetMethodName(method)
	return &f
}

// Helper function to login as a web client and sse listener
// This will set its cookie to allow for further requests.
// Run the TestLogin test before using this.
// This returns a client. Call Close() when done.
func WebLogin(fullURL string, clientID string,
	onConnection func(bool, error),
	onNotification func(message transports.NotificationMessage),
	onRequest func(message transports.RequestMessage) transports.ResponseMessage) (
	cl transports.IConsumerConnection, err error) {

	//sseCl := clients.NewHubClient(fullURL, clientID, ts.Certs.CaCert)
	// websocket client
	//sseCl := wssclient.NewWssTransportClient(
	//	fullURL, clientID, nil, ts.Certs.CaCert, time.Minute)
	// or sse-sc client

	// use the hub's SSE client to connect to the hiveoview server
	sseCl := sseclient.NewSsescConsumerClient(
		fullURL, clientID, nil, ts.Certs.CaCert,
		getHiveoviewForm, time.Minute)
	sseCl.SetConnectHandler(onConnection)
	sseCl.SetNotificationHandler(onNotification)
	sseCl.SetRequestHandler(onRequest)

	//err = sseCl.ConnectWithLoginForm(clientID)
	// FIXME: password is clientID
	// hiveoview uses a different login path as the hub
	_, err = sseCl.ConnectWithPassword(clientID)

	time.Sleep(time.Second * 10)
	return sseCl, err
}

func TestMain(m *testing.M) {
	var err error
	// raise loglevel where you want it in testing
	logging.SetLogging("warn", "")
	// clean start
	_ = os.RemoveAll(testFolder)
	_ = os.MkdirAll(testFolder, 0700)

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
		ts.Certs.ServerCert, ts.Certs.CaCert, noState)
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
	svc := service.NewHiveovService(servicePort, true,
		nil, "", ts.Certs.ServerCert, ts.Certs.CaCert, noState)
	avcAg, _ := ts.AddConnectService(serviceID)
	require.NotNil(t, avcAg)
	defer avcAg.Disconnect()
	err := svc.Start(avcAg)
	require.NoError(t, err)
	defer svc.Stop()

	// make sure the client to login as exists
	cl1, token1 := ts.AddConnectConsumer(clientID1, authz.ClientRoleOperator)
	//defer cl1.Disconnect()
	cl1.Disconnect()
	_ = token1
	time.Sleep(time.Millisecond * 10)

	// 2: login using plain TLS connection and a form
	hostPort := fmt.Sprintf("localhost:%d", servicePort)
	cl2 := tlsclient.NewTLSClient(
		hostPort, nil, ts.Certs.CaCert, time.Second*60, "cid1")

	// try login. The test user password is the clientID
	// authenticate the connection with the hiveot http/sse service (not the hub server)
	// the service will in turn forward the request to the hub.
	loginMessage := map[string]string{
		"login":    clientID1,
		"password": clientID1,
	}
	// this login will set an auth cookie
	loginJSON, _ := jsoniter.Marshal(loginMessage)
	_ = loginJSON
	resp, _, statusCode, err := cl2.Post("/login", loginJSON, "cid2")
	//resp holds the serialized new token
	cl2.Close()
	require.NoError(t, err)
	assert.Equal(t, 200, statusCode)

	// result contains the new paseto auth token (v4.public.*)
	var newToken string
	err = jsoniter.Unmarshal(resp, &newToken)
	assert.NotEmpty(t, newToken)
	assert.NoError(t, err)
	clientID, sessID, err := ts.Runtime.AuthnSvc.SessionAuth.ValidateToken(newToken)
	_ = clientID
	_ = sessID
	require.NoError(t, err)

	// retrieving about should succeed
	data, statusCode, err := cl2.Get(src.RenderAboutPath)
	_ = statusCode
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
	// todo verify the redirect to /about

	// request using a new client and the given auth token
	//cl3 := tlsclient.NewTLSClient(
	//	hostPort, nil, ts.Certs.CaCert, time.Second*60, "cid3")
	//cl3.SetAuthToken(newToken)
	//data, statusCode, err = cl3.Get(src.RenderAboutPath)
	//_ = statusCode
	//assert.NoError(t, err)
	//assert.NotEmpty(t, data)
	// todo verify the result is the about path, not a redirect to login

	t.Log("TestLogin completed")
}

// test login from a client using forms
func TestLoginForm(t *testing.T) {
	const clientID1 = "user1"

	// 1: setup: start a runtime and service; this generates an error that
	//    the state service isnt found. ignore it.
	svc := service.NewHiveovService(servicePort, true,
		nil, "", ts.Certs.ServerCert, ts.Certs.CaCert, noState)
	avcAg, _ := ts.AddConnectService(serviceID)
	require.NotNil(t, avcAg)
	defer avcAg.Disconnect()
	err := svc.Start(avcAg)

	require.NoError(t, err)
	defer svc.Stop()

	// make sure the client to login as exists
	cl1, token1 := ts.AddConnectConsumer(clientID1, authz.ClientRoleOperator)
	//defer cl1.Disconnect()
	cl1.Disconnect()

	_ = token1

	ccount, _ := ts.Runtime.CM.GetNrConnections()
	_ = ccount
	time.Sleep(time.Millisecond * 10)

	// 2: login using plain TLS connection and a form
	hostPort := fmt.Sprintf("localhost:%d", servicePort)
	cl2 := tlsclient.NewTLSClient(
		hostPort, nil, ts.Certs.CaCert, time.Second*60, "cid1")

	// try login. The test user password is the clientID
	// the client should receive a cookie with a token
	formMock := url.Values{}
	formMock.Add("loginID", clientID1)
	formMock.Add("password", clientID1)
	fullURL := fmt.Sprintf("https://%s/loginForm", hostPort)

	// authenticate the connection with the hiveot http/sse service (not the hub server)
	// the service will in turn forward the request to the hub.
	resp, err := cl2.GetHttpClient().PostForm(fullURL, formMock)
	cl2.Close()
	require.NoError(t, err)
	// this should redirect to /dashboard
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "/dashboard", resp.Request.URL.Path)

	// result contains html
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	assert.NoError(t, err)
	assert.NotEmpty(t, body)
	t.Log("TestLogin completed")
}

// test many connections from a single client and confirm they open close and receive messages properly.
func TestMultiConnectDisconnect(t *testing.T) {
	const clientID1 = "user1"
	const agentID = "agent1"
	const testConnections = int32(1)
	const eventName = "event1"
	var webClients = make([]transports.IConsumerConnection, 0)
	var connectCount atomic.Int32
	var disConnectCount atomic.Int32
	var messageCount atomic.Int32
	const waitamoment = time.Millisecond * 10

	logging.SetLogging("info", "")
	// 1: setup: start a runtime and service; this generates an error that
	//    the state service isnt found. ignore it.
	svc := service.NewHiveovService(servicePort, true,
		nil, "", ts.Certs.ServerCert, ts.Certs.CaCert, noState)
	avcAg, _ := ts.AddConnectService(serviceID)
	err := svc.Start(avcAg)

	require.NoError(t, err)
	defer svc.Stop()

	// the agent for publishing events. A TD is needed for them to be accepted.
	ag1, _ := ts.AddConnectAgent(agentID)
	_ = ag1
	td1 := ts.AddTD(agentID, nil)
	_ = td1
	// create the user account this test is going to connect as.
	// hiveoview server only supports HTTP/SSE
	cl1, token1 := ts.AddConnectConsumer(clientID1, authz.ClientRoleOperator)
	defer cl1.Disconnect()
	err = cl1.Subscribe("", "")
	require.NoError(t, err)
	time.Sleep(waitamoment)

	_ = token1
	//handler for web connections
	onConnection := func(connected bool, err error) {
		if connected {
			connectCount.Add(1)
		} else {
			disConnectCount.Add(1)
		}
	}
	// handler for web connection messages
	onNotification := func(msg transports.NotificationMessage) {
		// the UI expects this format for triggering htmx
		expectedType := fmt.Sprintf("dtw:%s:%s/%s", agentID, td1.ID, eventName)
		if expectedType == msg.Operation {
			messageCount.Add(1)
		}
	}

	// 2: connect and subscribe web clients and verify
	// each webclient connection will trigger a separate connection to the hub
	// with its own subscription.
	// The hiveoview server only supports SSE
	hiveoviewURL := svc.GetServerURL()
	for range testConnections {
		sseCl, err := WebLogin(
			hiveoviewURL, clientID1, onConnection, onNotification, nil)
		require.NoError(t, err)
		require.NotNil(t, sseCl)
		webClients = append(webClients, sseCl)
		time.Sleep(waitamoment)
	}
	// connection notification should have been received N times
	time.Sleep(time.Second * 1)
	require.Equal(t, testConnections, connectCount.Load(), "connect count mismatch")
	// the hiveoview session manager should have corresponding connections
	nrSessions := svc.GetSM().GetNrSessions()
	require.Equal(t, testConnections, nrSessions)

	// 3: agent publishes an event, which should be received N times
	notif1 := transports.NewNotificationMessage(wot.HTOpEvent, td1.ID, eventName, "a value")
	err = ag1.SendNotification(notif1)
	require.NoError(t, err)

	// event should have been received N times
	time.Sleep(time.Millisecond * 100)
	assert.Equal(t, testConnections, messageCount.Load(), "missing events")

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
	require.Equal(t, testConnections, disConnectCount.Load(), "disconnect count mismatch")

	// 5: no more messages should be received after disconnecting
	messageCount.Store(0)
	err = ag1.SendNotification(notif1)
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
	nrConnections, _ := ts.Runtime.CM.GetNrConnections()
	nrSessions = svc.GetSM().GetNrSessions()
	if nrConnections > 0 {
		t.Log(fmt.Sprintf(
			"FIXME: expected 0 remaining connections and sessions. "+
				"Got '%d' connections from '%d' sessions", nrConnections, nrSessions))
	}
	//assert.Less(t, 3, count)

	//time.Sleep(time.Millisecond * 100)
}
